<template>
  <div class="dashboard-page">
    <div class="dashboard-scroll">
      <section class="dashboard-intro">
        <div>
          <p class="dashboard-kicker">管理员工作台</p>
          <h1>{{ greeting }}，{{ displayName }}</h1>
          <p>查看文件服务的存储、分享、用户与安全状态。</p>
        </div>
        <div class="dashboard-actions">
          <n-button secondary @click="router.push('/logs')">
            <template #icon><n-icon><DocumentTextOutline /></n-icon></template>
            查看日志
          </n-button>
          <n-button type="primary" @click="router.push('/files')">
            <template #icon><n-icon><FolderOpenOutline /></n-icon></template>
            文件管理
          </n-button>
        </div>
      </section>

      <template v-if="loading">
        <div class="metric-grid" aria-label="正在加载概览数据">
          <n-skeleton v-for="index in 5" :key="index" height="140px" :sharp="false" />
        </div>
        <div class="dashboard-main-grid">
          <n-skeleton height="310px" :sharp="false" />
          <n-skeleton height="310px" :sharp="false" />
        </div>
      </template>

      <n-result
        v-else-if="errorMessage"
        status="error"
        title="仪表盘加载失败"
        :description="errorMessage"
        class="dashboard-error"
      >
        <template #footer>
          <n-button type="primary" @click="fetchDashboard">重新加载</n-button>
        </template>
      </n-result>

      <template v-else-if="dashboard">
        <section class="metric-grid" aria-label="系统概览">
          <article class="metric-card metric-card-storage">
            <div class="metric-heading">
              <span class="metric-icon"><n-icon><ServerOutline /></n-icon></span>
              <span>存储空间</span>
              <small v-if="dashboard.scan_status.scanning" class="storage-scanning">正在扫描</small>
            </div>
            <div class="storage-overview">
              <div class="storage-ring" :style="storageRingStyle" role="img" :aria-label="`共享目录占用 ${storagePercentLabel}`">
                <div><strong>{{ storagePercentLabel }}</strong><span>已占用</span></div>
              </div>
              <div class="storage-values">
                <div><strong>{{ formatBytes(dashboard.storage.shared_used_bytes) }}</strong><span>实际占用</span></div>
                <div><strong>{{ formatBytes(dashboard.storage.disk_available_bytes) }}</strong><span>剩余空间</span></div>
                <div><strong>{{ formatBytes(storageTotal) }}</strong><span>存储总量</span></div>
              </div>
            </div>
            <p class="storage-rule">存储总量按共享目录实际占用与所在分区剩余空间动态计算。</p>
          </article>

          <article class="metric-card">
            <div class="metric-heading">
              <span class="metric-icon"><n-icon><DocumentsOutline /></n-icon></span>
              <span>文件资源</span>
            </div>
            <strong class="metric-value">{{ numberFormat(dashboard.storage.file_count) }}</strong>
            <span class="metric-detail">{{ numberFormat(dashboard.storage.directory_count) }} 个目录</span>
          </article>

          <article class="metric-card">
            <div class="metric-heading">
              <span class="metric-icon"><n-icon><PeopleOutline /></n-icon></span>
              <span>用户</span>
            </div>
            <strong class="metric-value">{{ numberFormat(dashboard.summary.users.total) }}</strong>
            <span class="metric-detail">{{ dashboard.summary.users.totp_enabled }} 位已启用 TOTP</span>
          </article>

          <article class="metric-card">
            <div class="metric-heading">
              <span class="metric-icon"><n-icon><ShareSocialOutline /></n-icon></span>
              <span>有效分享</span>
            </div>
            <strong class="metric-value">{{ numberFormat(dashboard.summary.shares.valid) }}</strong>
            <span class="metric-detail" :class="{ warning: dashboard.summary.shares.expiring_soon > 0 }">
              {{ dashboard.summary.shares.expiring_soon }} 个将在 3 天内到期
            </span>
          </article>

          <article class="metric-card">
            <div class="metric-heading">
              <span class="metric-icon" :class="{ 'metric-icon-warning': securityWarnings > 0 }">
                <n-icon><ShieldCheckmarkOutline /></n-icon>
              </span>
              <span>安全状态</span>
            </div>
            <strong class="metric-value">{{ securityWarnings === 0 ? '良好' : `${securityWarnings} 项` }}</strong>
            <span class="metric-detail">{{ securityWarnings === 0 ? '关键防护均已启用' : '建议检查未启用项' }}</span>
          </article>
        </section>

        <section class="today-strip" aria-label="今日活动">
          <div class="today-title">
            <span>今日活动</span>
            <small>数据实时汇总自操作日志</small>
          </div>
          <div v-for="item in todayItems" :key="item.key" class="today-item">
            <strong>{{ numberFormat(item.value) }}</strong>
            <span>{{ item.label }}</span>
          </div>
          <button class="text-action" type="button" @click="fetchDashboard">
            <n-icon><RefreshOutline /></n-icon>
            刷新
          </button>
        </section>

        <div class="dashboard-main-grid">
          <section class="dashboard-panel activity-panel">
            <div class="panel-header activity-panel-header">
              <div>
                <h2>最近 {{ activityRange }} 天操作趋势</h2>
                <p>上传、下载、创建分享与失败登录</p>
              </div>
              <div class="activity-controls">
                <div class="range-control">
                  <span>最近</span>
                  <n-input-number v-model:value="activityRangeDraft" :min="1" :max="activityMaxDays" :precision="0" size="small" class="activity-range-input" @keyup.enter="applyActivityRange" />
                  <span>天</span>
                  <n-button size="small" secondary :loading="activityLoading" @click="applyActivityRange">应用</n-button>
                </div>
                <div class="chart-type-switch" role="group" aria-label="图表类型">
                  <button type="button" :class="{ active: chartType === 'bar' }" @click="chartType = 'bar'">柱状图</button>
                  <button type="button" :class="{ active: chartType === 'line' }" @click="chartType = 'line'">折线图</button>
                </div>
              </div>
            </div>
            <div class="chart-legend" aria-label="图例">
              <span v-for="series in activitySeries" :key="series.key"><i :class="`legend-${series.key}`"></i>{{ series.label }}</span>
              <small>范围上限 {{ activityMaxDays }} 天，由日志保留天数决定</small>
            </div>
            <div class="activity-chart-viewport">
              <div v-if="chartType === 'bar'" class="activity-chart" :style="chartTrackStyle" role="img" :aria-label="activityChartLabel">
                <div v-for="day in dashboard.activity" :key="day.date" class="chart-day">
                  <div class="chart-bars">
                    <span
                      v-for="series in activitySeries"
                      :key="series.key"
                      class="bar-hit"
                      tabindex="0"
                    >
                      <i class="bar" :class="`bar-${series.key}`" :style="barStyle(day[series.key])"></i>
                      <span class="chart-tooltip">{{ shortDate(day.date) }} · {{ series.label }} {{ day[series.key] }}</span>
                    </span>
                  </div>
                  <span class="chart-date">{{ shortDate(day.date) }}</span>
                </div>
              </div>
              <div v-else class="line-chart" :style="chartTrackStyle" role="img" :aria-label="activityChartLabel">
                <div class="line-plot">
                  <svg viewBox="0 0 1000 180" preserveAspectRatio="none" aria-hidden="true">
                    <polyline v-for="series in activitySeries" :key="series.key" :class="`line-${series.key}`" :points="linePoints(series.key)" />
                  </svg>
                  <template v-for="(day, index) in dashboard.activity" :key="day.date">
                    <span
                      v-for="(series, seriesIndex) in activitySeries"
                      :key="series.key"
                      class="line-hit"
                      :class="[`line-hit-${series.key}`, { 'edge-start': index === 0, 'edge-end': index === dashboard.activity.length - 1 }]"
                      :style="linePointStyle(index, day[series.key], seriesIndex)"
                      tabindex="0"
                    >
                      <i></i>
                      <span class="chart-tooltip">{{ shortDate(day.date) }} · {{ series.label }} {{ day[series.key] }}</span>
                    </span>
                  </template>
                </div>
                <div class="line-date-grid" :style="{ gridTemplateColumns: `repeat(${dashboard.activity.length}, minmax(34px, 1fr))` }">
                  <span v-for="day in dashboard.activity" :key="day.date">{{ shortDate(day.date) }}</span>
                </div>
              </div>
            </div>
          </section>

          <section class="dashboard-panel security-panel">
            <div class="panel-header">
              <div>
                <h2>系统与安全</h2>
                <p>{{ securityWarnings === 0 ? '关键防护配置正常' : `${securityWarnings} 项配置需要关注` }}</p>
              </div>
              <button class="text-action" type="button" @click="router.push('/admin/settings')">系统设置</button>
            </div>
            <div class="security-list">
              <div v-for="check in dashboard.security" :key="check.key" class="security-row">
                <span class="security-status" :class="check.ok ? 'is-ok' : check.optional ? 'is-neutral' : 'is-warning'">
                  <n-icon><CheckmarkCircleOutline v-if="check.ok" /><InformationCircleOutline v-else-if="check.optional" /><AlertCircleOutline v-else /></n-icon>
                </span>
                <div>
                  <strong>{{ check.label }}</strong>
                  <span>{{ check.description }}</span>
                </div>
                <em>{{ check.status || (check.ok ? '已启用' : '待配置') }}</em>
              </div>
            </div>
          </section>

          <section class="dashboard-panel logs-panel">
            <div class="panel-header">
              <div>
                <h2>最近活动</h2>
                <p>最新的关键操作记录</p>
              </div>
              <button class="text-action" type="button" @click="router.push('/logs')">全部日志</button>
            </div>
            <div v-if="dashboard.recent_logs.length" class="activity-list">
              <button
                v-for="log in dashboard.recent_logs"
                :key="log.id"
                class="activity-row"
                type="button"
                @click="router.push('/logs')"
              >
                <span class="activity-icon"><n-icon><PulseOutline /></n-icon></span>
                <div class="activity-copy">
                  <span><strong>{{ log.username }}</strong> {{ actionLabel(log.action) }}</span>
                  <small :title="log.target_path || log.ip_address">{{ log.target_path || log.ip_address || '系统' }}</small>
                </div>
                <time>{{ relativeTime(log.created_at) }}</time>
              </button>
            </div>
            <div v-else class="panel-empty">暂无操作记录</div>
          </section>

          <section class="dashboard-panel shares-panel">
            <div class="panel-header">
              <div>
                <h2>分享概况</h2>
                <p>最近创建的文件分享</p>
              </div>
              <button class="text-action" type="button" @click="router.push('/shares')">分享管理</button>
            </div>
            <div v-if="dashboard.recent_shares.length" class="share-list">
              <button
                v-for="share in dashboard.recent_shares"
                :key="share.id"
                class="share-row"
                type="button"
                @click="router.push({ path: '/shares', query: { highlightId: share.id } })"
              >
                <span class="share-icon"><n-icon><LinkOutline /></n-icon></span>
                <div class="share-copy">
                  <strong :title="share.name">{{ share.name }}</strong>
                  <span>{{ share.owner }} · {{ share.access_count }} 次访问</span>
                </div>
                <span class="share-status" :class="`status-${share.status}`">{{ shareStatusLabel(share.status) }}</span>
              </button>
            </div>
            <div v-else class="panel-empty">暂无分享记录</div>
          </section>

          <section class="dashboard-panel storage-panel">
            <div class="panel-header">
              <div>
                <h2>共享目录构成</h2>
                <p class="path-text" :title="dashboard.storage.root_path">{{ dashboard.storage.root_path }}</p>
              </div>
              <span v-if="!dashboard.storage.scan_complete" class="scan-warning">部分文件未能读取</span>
            </div>
            <div v-if="dashboard.storage.categories.length" class="category-list">
              <div v-for="category in dashboard.storage.categories" :key="category.key" class="category-row">
                <div class="category-meta">
                  <span>{{ category.label }}</span>
                  <small>{{ numberFormat(category.count) }} 个文件</small>
                </div>
                <div class="category-meter" aria-hidden="true">
                  <span :style="{ width: categoryWidth(category.bytes) }"></span>
                </div>
                <strong>{{ formatBytes(category.bytes) }}</strong>
              </div>
            </div>
            <div v-else class="panel-empty">共享目录中暂无文件</div>
          </section>

          <section class="dashboard-panel quick-panel">
            <div class="panel-header">
              <div>
                <h2>快捷管理</h2>
                <p>常用管理员入口</p>
              </div>
            </div>
            <div class="quick-grid">
              <button type="button" @click="router.push('/files')"><n-icon><FolderOpenOutline /></n-icon><span>文件管理<small>浏览与上传文件</small></span></button>
              <button type="button" @click="router.push('/admin/users')"><n-icon><PeopleOutline /></n-icon><span>用户管理<small>账户与权限</small></span></button>
              <button type="button" @click="router.push('/shares')"><n-icon><ShareSocialOutline /></n-icon><span>分享管理<small>有效期与访问量</small></span></button>
              <button type="button" @click="router.push('/admin/settings')"><n-icon><SettingsOutline /></n-icon><span>系统设置<small>安全与服务配置</small></span></button>
            </div>
          </section>
        </div>

        <p class="dashboard-updated">数据更新时间：{{ formatDateTime(dashboard.generated_at) }}</p>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { NButton, NIcon, NInputNumber, NResult, NSkeleton, useMessage } from 'naive-ui'
import {
  AlertCircleOutline,
  CheckmarkCircleOutline,
  DocumentTextOutline,
  DocumentsOutline,
  FolderOpenOutline,
  InformationCircleOutline,
  LinkOutline,
  PeopleOutline,
  PulseOutline,
  RefreshOutline,
  ServerOutline,
  SettingsOutline,
  ShareSocialOutline,
  ShieldCheckmarkOutline,
} from '@vicons/ionicons5'
import api from '@/api'
import { useUserStore } from '@/stores/user'

interface StorageCategory { key: string; label: string; bytes: number; count: number }
type ActivityKey = 'upload' | 'download' | 'share_create' | 'login_fail'
interface ActivityDay extends Record<ActivityKey, number> { date: string }
interface RecentLog { id: number; username: string; action: string; target_path: string; ip_address: string; created_at: string }
interface RecentShare { id: number; name: string; owner: string; status: 'valid' | 'expired' | 'deleted'; access_count: number; expire_at: string }
interface SecurityCheck { key: string; label: string; description: string; ok: boolean; optional?: boolean; status?: string }
interface DashboardData {
  generated_at: string
  storage: {
    root_path: string
    shared_used_bytes: number
    disk_available_bytes: number
    disk_total_bytes: number
    file_count: number
    directory_count: number
    scan_complete: boolean
    scanned_at: string
    categories: StorageCategory[]
  }
  summary: {
    users: { total: number; admins: number; totp_enabled: number }
    shares: { valid: number; expiring_soon: number; expired: number; access_count: number }
    today: Record<string, number>
  }
  activity: ActivityDay[]
  activity_days: number
  activity_max_days: number
  recent_logs: RecentLog[]
  recent_shares: RecentShare[]
  security: SecurityCheck[]
  scan_status: {
    ready: boolean
    scanning: boolean
    last_error: string
    last_completed_at: string | null
    next_scan_at: string | null
  }
}

const router = useRouter()
const userStore = useUserStore()
const message = useMessage()
const loading = ref(true)
const errorMessage = ref('')
const dashboard = ref<DashboardData | null>(null)
const activityRange = ref(7)
const activityRangeDraft = ref(7)
const activityMaxDays = ref(30)
const activityLoading = ref(false)
const chartType = ref<'bar' | 'line'>('bar')
const activitySeries: Array<{ key: ActivityKey; label: string }> = [
  { key: 'upload', label: '上传' },
  { key: 'download', label: '下载' },
  { key: 'share_create', label: '分享' },
  { key: 'login_fail', label: '失败登录' },
]

const displayName = computed(() => userStore.user?.display_name || userStore.user?.username || '管理员')
const greeting = computed(() => {
  const hour = new Date().getHours()
  if (hour < 6) return '夜深了'
  if (hour < 12) return '早上好'
  if (hour < 18) return '下午好'
  return '晚上好'
})
const securityWarnings = computed(() => dashboard.value?.security.filter(item => !item.ok && !item.optional).length || 0)
const storageTotal = computed(() => (dashboard.value?.storage.shared_used_bytes || 0) + (dashboard.value?.storage.disk_available_bytes || 0))
const storagePercent = computed(() => storageTotal.value > 0 ? (dashboard.value?.storage.shared_used_bytes || 0) / storageTotal.value * 100 : 0)
const storagePercentLabel = computed(() => `${storagePercent.value >= 10 ? storagePercent.value.toFixed(0) : storagePercent.value.toFixed(1)}%`)
const storageRingStyle = computed(() => ({
  background: `conic-gradient(var(--workspace-accent) ${storagePercent.value}%, var(--workspace-surface-strong) 0)`,
}))
const chartMax = computed(() => {
  if (!dashboard.value) return 1
  return Math.max(1, ...dashboard.value.activity.flatMap(day => [day.upload, day.download, day.share_create, day.login_fail]))
})
const todayItems = computed(() => {
  const today = dashboard.value?.summary.today || {}
  return [
    { key: 'upload', label: '上传', value: today.upload || 0 },
    { key: 'download', label: '下载', value: today.download || 0 },
    { key: 'share_access', label: '分享访问', value: today.share_access || 0 },
    { key: 'login_fail', label: '失败登录', value: today.login_fail || 0 },
  ]
})
const activityChartLabel = computed(() => dashboard.value?.activity.map(day =>
  `${day.date} 上传 ${day.upload}，下载 ${day.download}，分享 ${day.share_create}，失败登录 ${day.login_fail}`,
).join('；') || '')
const chartTrackStyle = computed(() => ({
  minWidth: `${Math.max(640, (dashboard.value?.activity.length || 7) * 42)}px`,
  gridTemplateColumns: `repeat(${dashboard.value?.activity.length || 7}, minmax(34px, 1fr))`,
}))

const actionLabels: Record<string, string> = {
  LOGIN: '登录了系统', LOGIN_FAIL: '登录失败', LOGIN_TOTP: '完成了双重认证', LOGIN_TOTP_FAIL: '双重认证失败',
  UPLOAD: '上传了文件', DOWNLOAD: '下载了文件', CREATE_DIR: '创建了目录', MOVE: '移动了文件',
  DELETE_FILE: '删除了文件', DELETE_DIR: '删除了目录', SHARE_CREATE: '创建了分享', SHARE_ACCESS: '访问了分享',
  SHARE_DELETE: '删除了分享', CHANGE_OWNER: '变更了所有者', USER_CREATE: '创建了用户', USER_UPDATE: '更新了用户',
  USER_DELETE: '删除了用户', CONFIG_CHANGE: '更新了系统配置', BLOCK_IP: '触发了 IP 封锁', BLOCK_ACCOUNT: '触发了账户封锁',
}

async function fetchDashboard() {
  loading.value = true
  errorMessage.value = ''
  try {
    const response = await api.get<DashboardData>('/api/dashboard')
    dashboard.value = response.data
    activityRange.value = response.data.activity_days
    activityRangeDraft.value = response.data.activity_days
    activityMaxDays.value = response.data.activity_max_days
  } catch (error: any) {
    errorMessage.value = error.response?.data?.error || '无法连接服务器，请稍后重试。'
  } finally {
    loading.value = false
  }
}

async function applyActivityRange() {
  if (!dashboard.value || activityLoading.value) return
  const requested = Math.min(activityMaxDays.value, Math.max(1, Math.round(activityRangeDraft.value || 7)))
  activityRangeDraft.value = requested
  activityLoading.value = true
  try {
    const response = await api.get<{ activity: ActivityDay[]; days: number; max_days: number }>('/api/dashboard/activity', { params: { days: requested } })
    dashboard.value.activity = response.data.activity
    activityRange.value = response.data.days
    activityRangeDraft.value = response.data.days
    activityMaxDays.value = response.data.max_days
  } catch (error: any) {
    message.error(error.response?.data?.error || '获取操作趋势失败')
  } finally {
    activityLoading.value = false
  }
}

function formatBytes(value: number) {
  if (!Number.isFinite(value) || value <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
  const index = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1)
  const scaled = value / 1024 ** index
  return `${scaled >= 100 || index === 0 ? scaled.toFixed(0) : scaled.toFixed(1)} ${units[index]}`
}

function numberFormat(value: number) { return new Intl.NumberFormat('zh-CN').format(value || 0) }
function shortDate(value: string) { return value.slice(5).replace('-', '/') }
function formatDateTime(value: string) { return new Date(value).toLocaleString('zh-CN', { hour12: false }) }
function actionLabel(action: string) { return actionLabels[action] || '执行了系统操作' }
function shareStatusLabel(status: RecentShare['status']) { return status === 'valid' ? '有效' : status === 'expired' ? '已过期' : '已失效' }
function barStyle(value: number) { return { height: value > 0 ? `${Math.max(5, value / chartMax.value * 100)}%` : '2px', opacity: value > 0 ? '1' : '.16' } }
function linePoints(key: ActivityKey) {
  const activity = dashboard.value?.activity || []
  if (!activity.length) return ''
  return activity.map((day, index) => {
    const x = activity.length === 1 ? 500 : index / (activity.length - 1) * 1000
    const y = 170 - day[key] / chartMax.value * 150
    return `${x},${y}`
  }).join(' ')
}
function linePointStyle(index: number, value: number, seriesIndex: number) {
  const length = dashboard.value?.activity.length || 1
  const left = length === 1 ? 50 : index / (length - 1) * 100
  const top = (170 - value / chartMax.value * 150) / 180 * 100
  return { left: `${left}%`, top: `${top}%`, marginLeft: `${(seriesIndex - 1.5) * 4}px` }
}
function categoryWidth(bytes: number) {
  const total = dashboard.value?.storage.shared_used_bytes || 0
  return total > 0 ? `${Math.max(2, bytes / total * 100)}%` : '0%'
}
function relativeTime(value: string) {
  const time = new Date(value.replace(' ', 'T') + (value.includes('Z') || value.includes('+') ? '' : 'Z')).getTime()
  if (!Number.isFinite(time)) return value
  const diff = Math.max(0, Date.now() - time)
  if (diff < 60_000) return '刚刚'
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)} 分钟前`
  if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)} 小时前`
  return `${Math.floor(diff / 86_400_000)} 天前`
}

onMounted(fetchDashboard)
</script>

<style scoped>
.dashboard-page { flex: 1; min-height: 0; overflow: hidden; }
.dashboard-scroll { height: 100%; overflow-y: auto; padding: 2px 2px 18px; scrollbar-width: none; }
.dashboard-scroll::-webkit-scrollbar { display: none; width: 0; height: 0; }
.dashboard-intro { display: flex; align-items: flex-end; justify-content: space-between; gap: 24px; padding: 10px 4px 18px; }
.dashboard-kicker { margin-bottom: 5px; color: var(--workspace-accent); font-size: 12px; font-weight: 700; }
.dashboard-intro h1 { color: var(--workspace-text); font-size: clamp(24px, 3vw, 32px); line-height: 1.2; letter-spacing: -0.025em; }
.dashboard-intro p:not(.dashboard-kicker) { margin-top: 7px; color: var(--workspace-text-muted); font-size: 13px; }
.dashboard-actions { display: flex; gap: 8px; flex: 0 0 auto; }
.metric-grid { display: grid; grid-template-columns: repeat(6, minmax(0, 1fr)); gap: 10px; }
.metric-card { min-width: 0; min-height: 140px; padding: 15px; border: 1px solid var(--workspace-border-soft); border-radius: var(--workspace-radius-lg); background: var(--workspace-surface); box-shadow: inset 0 1px 0 rgba(255, 255, 255, .55); display: flex; flex-direction: column; }
.metric-card-storage { grid-column: span 2; }
.metric-heading { display: flex; align-items: center; gap: 8px; color: var(--workspace-text-muted); font-size: 12px; font-weight: 650; }
.metric-icon { width: 30px; height: 30px; display: inline-flex; align-items: center; justify-content: center; border-radius: var(--workspace-radius-sm); color: var(--workspace-accent); background: rgba(var(--workspace-accent-rgb), .09); font-size: 17px; }
.metric-icon-warning { color: #b45309; background: rgba(245, 158, 11, .12); }
.metric-value { margin-top: auto; color: var(--workspace-text); font-size: 29px; line-height: 1; letter-spacing: -.035em; font-variant-numeric: tabular-nums; }
.metric-detail { margin-top: 7px; color: var(--workspace-text-muted); font-size: 11px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.metric-detail.warning { color: #b45309; }
.storage-scanning { margin-left: auto; color: var(--workspace-accent); font-size: 10px; font-weight: 650; }
.storage-overview { display: grid; grid-template-columns: 66px minmax(0, 1fr); align-items: center; gap: 12px; margin-top: 9px; }
.storage-ring { width: 66px; height: 66px; padding: 6px; border-radius: 50%; }
.storage-ring > div { width: 100%; height: 100%; border-radius: 50%; display: flex; align-items: center; justify-content: center; flex-direction: column; background: var(--workspace-surface); }
.storage-ring strong { color: var(--workspace-text); font-size: 14px; line-height: 1; font-variant-numeric: tabular-nums; }
.storage-ring span { margin-top: 3px; color: var(--workspace-text-soft); font-size: 8px; }
.storage-values { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 8px; }
.storage-values div { min-width: 0; }
.storage-values strong { display: block; color: var(--workspace-text); font-size: 16px; line-height: 1; letter-spacing: -.025em; font-variant-numeric: tabular-nums; white-space: nowrap; }
.storage-values span { display: block; margin-top: 5px; color: var(--workspace-text-muted); font-size: 11px; }
.storage-rule { margin-top: 10px; color: var(--workspace-text-soft); font-size: 10px; line-height: 1.4; }
.today-strip { display: grid; grid-template-columns: minmax(170px, 1.5fr) repeat(4, minmax(80px, .6fr)) auto; align-items: center; gap: 8px; margin-top: 10px; padding: 10px 14px; border: 1px solid var(--workspace-border-soft); border-radius: var(--workspace-radius-lg); background: color-mix(in srgb, var(--workspace-surface) 88%, transparent); }
.today-title { display: flex; flex-direction: column; gap: 2px; }
.today-title span { color: var(--workspace-text); font-size: 13px; font-weight: 700; }
.today-title small { color: var(--workspace-text-soft); font-size: 10px; }
.today-item { display: flex; align-items: baseline; gap: 5px; }
.today-item strong { color: var(--workspace-text); font-size: 18px; font-variant-numeric: tabular-nums; }
.today-item span { color: var(--workspace-text-muted); font-size: 11px; white-space: nowrap; }
.text-action { border: 0; background: transparent; color: var(--workspace-accent); font: inherit; font-size: 12px; font-weight: 650; cursor: pointer; display: inline-flex; align-items: center; gap: 4px; white-space: nowrap; }
.text-action:hover { text-decoration: underline; text-underline-offset: 3px; }
.text-action:active { transform: translateY(1px); }
.dashboard-main-grid { display: grid; grid-template-columns: minmax(0, 1.55fr) minmax(320px, 1fr); gap: 10px; margin-top: 10px; }
.dashboard-panel { min-width: 0; padding: 15px; border: 1px solid var(--workspace-border-soft); border-radius: var(--workspace-radius-lg); background: var(--workspace-surface); box-shadow: inset 0 1px 0 rgba(255, 255, 255, .55); }
:global(html.dark .metric-card) {
  border-color: rgba(var(--workspace-accent-rgb), .17);
  background:
    linear-gradient(
      145deg,
      rgba(var(--workspace-accent-rgb), .11) 0%,
      rgba(var(--workspace-accent-rgb), .025) 44%,
      transparent 74%
    ),
    color-mix(in srgb, var(--workspace-surface-strong) 72%, var(--workspace-surface));
  box-shadow:
    inset 0 1px 0 rgba(248, 250, 252, .08),
    inset 0 0 0 1px rgba(var(--workspace-accent-rgb), .025),
    0 10px 24px rgba(2, 6, 23, .18);
}
:global(html.dark .dashboard-panel) {
  border-color: rgba(var(--workspace-accent-rgb), .14);
  background:
    linear-gradient(
      155deg,
      rgba(var(--workspace-accent-rgb), .075) 0%,
      rgba(var(--workspace-accent-rgb), .018) 38%,
      transparent 68%
    ),
    color-mix(in srgb, var(--workspace-surface-strong) 64%, var(--workspace-surface));
  box-shadow:
    inset 0 1px 0 rgba(248, 250, 252, .07),
    inset 0 0 0 1px rgba(var(--workspace-accent-rgb), .02),
    0 10px 26px rgba(2, 6, 23, .16);
}
:global(html.dark .today-strip) {
  border-color: rgba(var(--workspace-accent-rgb), .15);
  background:
    linear-gradient(
      90deg,
      rgba(var(--workspace-accent-rgb), .09),
      rgba(var(--workspace-accent-rgb), .018) 42%,
      transparent 72%
    ),
    color-mix(in srgb, var(--workspace-surface-strong) 60%, var(--workspace-surface));
  box-shadow: inset 0 1px 0 rgba(248, 250, 252, .065);
}
.panel-header { display: flex; align-items: flex-start; justify-content: space-between; gap: 14px; margin-bottom: 13px; }
.panel-header h2 { color: var(--workspace-text); font-size: 14px; line-height: 1.3; }
.panel-header p { margin-top: 3px; color: var(--workspace-text-muted); font-size: 11px; }
.activity-panel { --chart-upload: #3b82f6; --chart-download: #f59e0b; --chart-share: #10b981; --chart-failure: #e05260; }
.activity-panel-header { align-items: center; }
.activity-controls { display: flex; align-items: center; justify-content: flex-end; flex-wrap: wrap; gap: 8px; }
.range-control { display: flex; align-items: center; gap: 5px; color: var(--workspace-text-muted); font-size: 10px; }
.activity-range-input { width: 76px; }
.chart-type-switch { display: inline-flex; padding: 2px; border: 1px solid var(--workspace-border-soft); border-radius: var(--workspace-radius-sm); background: var(--workspace-surface-soft); }
.chart-type-switch button { min-height: 26px; padding: 0 9px; border: 0; border-radius: 6px; color: var(--workspace-text-muted); background: transparent; font-size: 10px; cursor: pointer; }
.chart-type-switch button.active { color: var(--workspace-accent); background: var(--workspace-surface); box-shadow: 0 1px 5px rgba(39,55,82,.1); }
.chart-legend { display: flex; flex-wrap: wrap; align-items: center; gap: 8px 12px; min-height: 22px; margin-bottom: 4px; color: var(--workspace-text-muted); font-size: 10px; }
.chart-legend span { display: inline-flex; align-items: center; gap: 4px; }
.chart-legend i { width: 7px; height: 7px; border-radius: 2px; }
.chart-legend small { margin-left: auto; color: var(--workspace-text-soft); font-size: 9px; }
.legend-upload, .bar-upload { background: var(--chart-upload); }
.legend-download, .bar-download { background: var(--chart-download); }
.legend-share_create, .bar-share_create { background: var(--chart-share); }
.legend-login_fail, .bar-login_fail { background: var(--chart-failure); }
.activity-chart-viewport { overflow-x: auto; overflow-y: hidden; scrollbar-width: thin; scrollbar-color: var(--workspace-border) transparent; }
.activity-chart { height: 228px; display: grid; gap: 9px; padding: 26px 2px 0; border-bottom: 1px solid var(--workspace-border-soft); background: repeating-linear-gradient(to bottom, transparent 0, transparent 49px, var(--workspace-border-soft) 50px); }
.chart-day { min-width: 0; display: flex; flex-direction: column; justify-content: flex-end; align-items: stretch; }
.chart-bars { flex: 1; display: flex; align-items: flex-end; justify-content: center; gap: 3px; min-height: 0; }
.bar-hit { position: relative; width: min(9px, 21%); height: 100%; display: flex; align-items: flex-end; outline: none; }
.bar { display: block; width: 100%; min-height: 2px; border-radius: 3px 3px 1px 1px; transition: height .3s cubic-bezier(.16,1,.3,1), filter .16s ease; }
.bar-hit:hover .bar, .bar-hit:focus-visible .bar { filter: brightness(1.08) saturate(1.15); }
.chart-date { height: 25px; padding-top: 7px; text-align: center; color: var(--workspace-text-soft); font-size: 10px; font-variant-numeric: tabular-nums; }
.chart-tooltip { position: absolute; z-index: 3; left: 50%; bottom: calc(100% + 7px); transform: translateX(-50%) translateY(3px); padding: 5px 7px; border: 1px solid var(--workspace-border-soft); border-radius: 7px; color: var(--workspace-text); background: var(--workspace-surface); box-shadow: var(--workspace-shadow-card); font-size: 9px; font-style: normal; font-weight: 600; line-height: 1; white-space: nowrap; pointer-events: none; opacity: 0; transition: opacity .14s ease, transform .14s ease; }
.bar-hit .chart-tooltip { top: -22px; bottom: auto; }
.bar-hit:hover .chart-tooltip, .bar-hit:focus .chart-tooltip, .line-hit:hover .chart-tooltip, .line-hit:focus .chart-tooltip { opacity: 1; transform: translateX(-50%) translateY(0); }
.chart-day:first-child .chart-tooltip, .line-hit.edge-start .chart-tooltip { left: 0; transform: translateX(-4px) translateY(3px); }
.chart-day:last-child .chart-tooltip, .line-hit.edge-end .chart-tooltip { right: 0; left: auto; transform: translateX(4px) translateY(3px); }
.chart-day:first-child .bar-hit:hover .chart-tooltip, .chart-day:first-child .bar-hit:focus .chart-tooltip, .line-hit.edge-start:hover .chart-tooltip, .line-hit.edge-start:focus .chart-tooltip { transform: translateX(-4px) translateY(0); }
.chart-day:last-child .bar-hit:hover .chart-tooltip, .chart-day:last-child .bar-hit:focus .chart-tooltip, .line-hit.edge-end:hover .chart-tooltip, .line-hit.edge-end:focus .chart-tooltip { transform: translateX(4px) translateY(0); }
.line-chart { height: 228px; }
.line-plot { position: relative; height: 202px; margin-inline: 8px; background: repeating-linear-gradient(to bottom, transparent 0, transparent 49px, var(--workspace-border-soft) 50px); }
.line-plot svg { position: absolute; inset: 0; width: 100%; height: 100%; overflow: visible; }
.line-plot polyline { fill: none; vector-effect: non-scaling-stroke; stroke-width: 2; stroke-linecap: round; stroke-linejoin: round; }
.line-upload { stroke: var(--chart-upload); }
.line-download { stroke: var(--chart-download); }
.line-share_create { stroke: var(--chart-share); }
.line-login_fail { stroke: var(--chart-failure); }
.line-hit { position: absolute; width: 16px; height: 16px; transform: translate(-50%, -50%); border-radius: 50%; outline: none; cursor: default; }
.line-hit > i { position: absolute; inset: 4px; border: 2px solid var(--workspace-surface); border-radius: 50%; box-shadow: 0 0 0 1px currentColor; background: currentColor; }
.line-hit-upload { color: var(--chart-upload); }
.line-hit-download { color: var(--chart-download); }
.line-hit-share_create { color: var(--chart-share); }
.line-hit-login_fail { color: var(--chart-failure); }
.line-hit .chart-tooltip { bottom: calc(100% + 3px); }
.line-date-grid { height: 26px; display: grid; align-items: end; border-bottom: 1px solid var(--workspace-border-soft); }
.line-date-grid span { padding-bottom: 7px; color: var(--workspace-text-soft); text-align: center; font-size: 10px; font-variant-numeric: tabular-nums; }
.security-list, .activity-list, .share-list, .category-list { display: flex; flex-direction: column; }
.security-row { display: grid; grid-template-columns: 28px minmax(0, 1fr) auto; align-items: center; gap: 8px; min-height: 42px; border-top: 1px solid var(--workspace-border-soft); }
.security-row:first-child { border-top: 0; }
.security-status { display: inline-flex; font-size: 18px; }
.security-status.is-ok { color: #228b60; }
.security-status.is-warning { color: #c17a16; }
.security-status.is-neutral { color: var(--workspace-text-soft); }
.security-row div { min-width: 0; display: flex; flex-direction: column; gap: 2px; }
.security-row strong { color: var(--workspace-text); font-size: 11px; }
.security-row span:not(.security-status) { color: var(--workspace-text-muted); font-size: 9px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.security-row em { color: var(--workspace-text-soft); font-size: 10px; font-style: normal; }
.activity-row, .share-row { width: 100%; display: grid; grid-template-columns: 30px minmax(0, 1fr) auto; align-items: center; gap: 9px; min-height: 45px; padding: 0; border: 0; border-top: 1px solid var(--workspace-border-soft); color: inherit; background: transparent; text-align: left; cursor: pointer; }
.activity-row:first-child, .share-row:first-child { border-top: 0; }
.activity-row:hover, .share-row:hover { background: var(--workspace-row-hover); }
.activity-icon, .share-icon { width: 27px; height: 27px; border-radius: var(--workspace-radius-sm); display: inline-flex; align-items: center; justify-content: center; color: var(--workspace-accent); background: rgba(var(--workspace-accent-rgb), .08); }
.activity-copy, .share-copy { min-width: 0; display: flex; flex-direction: column; gap: 3px; }
.activity-copy span, .share-copy strong { color: var(--workspace-text); font-size: 11px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.activity-copy small, .share-copy span { color: var(--workspace-text-muted); font-size: 9px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.activity-row time { color: var(--workspace-text-soft); font-size: 9px; white-space: nowrap; }
.share-status { padding: 3px 7px; border-radius: 999px; font-size: 9px; white-space: nowrap; }
.status-valid { color: #16704c; background: rgba(34,160,107,.11); }
.status-expired, .status-deleted { color: #a2464d; background: rgba(216,90,97,.1); }
.path-text { max-width: 420px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.scan-warning { color: #b45309; font-size: 10px; }
.category-row { min-height: 42px; display: grid; grid-template-columns: 110px minmax(80px, 1fr) 76px; align-items: center; gap: 12px; border-top: 1px solid var(--workspace-border-soft); }
.category-row:first-child { border-top: 0; }
.category-meta { display: flex; flex-direction: column; gap: 2px; }
.category-meta span { color: var(--workspace-text); font-size: 11px; font-weight: 650; }
.category-meta small { color: var(--workspace-text-soft); font-size: 9px; }
.category-meter { height: 5px; border-radius: 999px; overflow: hidden; background: var(--workspace-surface-strong); }
.category-meter span { display: block; height: 100%; border-radius: inherit; background: var(--workspace-accent); }
.category-row > strong { color: var(--workspace-text); font-size: 11px; text-align: right; font-variant-numeric: tabular-nums; }
.quick-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; }
.quick-grid button { min-width: 0; display: flex; align-items: center; gap: 9px; padding: 11px; border: 1px solid var(--workspace-border-soft); border-radius: var(--workspace-radius-md); color: var(--workspace-text); background: var(--workspace-surface-soft); text-align: left; cursor: pointer; }
.quick-grid button:hover { border-color: rgba(var(--workspace-accent-rgb), .25); background: var(--workspace-row-hover); }
.quick-grid button:active { transform: translateY(1px); }
.quick-grid .n-icon { flex: 0 0 auto; color: var(--workspace-accent); font-size: 19px; }
.quick-grid span { min-width: 0; display: flex; flex-direction: column; gap: 2px; font-size: 11px; font-weight: 650; }
.quick-grid small { color: var(--workspace-text-muted); font-size: 9px; font-weight: 400; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.panel-empty { min-height: 150px; display: flex; align-items: center; justify-content: center; color: var(--workspace-text-soft); font-size: 11px; }
.dashboard-error { margin-top: 40px; padding: 30px; border-radius: var(--workspace-radius-xl); background: var(--workspace-surface); }
.dashboard-updated { margin: 12px 2px 0; text-align: right; color: var(--workspace-text-soft); font-size: 10px; }

@media (max-width: 980px) {
  .metric-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .metric-card-storage { grid-column: span 2; }
  .dashboard-main-grid { grid-template-columns: 1fr; }
  .today-strip { grid-template-columns: minmax(130px, 1.2fr) repeat(4, minmax(64px, .7fr)) auto; }
}

@media (max-width: 680px) {
  .dashboard-scroll { padding-bottom: 10px; }
  .dashboard-intro { align-items: flex-start; flex-direction: column; gap: 14px; padding-top: 6px; }
  .dashboard-actions { width: 100%; }
  .dashboard-actions :deep(.n-button) { flex: 1; }
  .metric-grid { grid-template-columns: 1fr 1fr; }
  .metric-card-storage { grid-column: span 2; }
  .metric-card { padding: 13px; }
  .metric-value { font-size: 25px; }
  .today-strip { grid-template-columns: repeat(4, 1fr); }
  .today-title { grid-column: 1 / -1; }
  .today-strip > .text-action { position: absolute; right: 28px; margin-top: -54px; }
  .today-item { flex-direction: column; align-items: flex-start; gap: 0; }
  .today-item strong { font-size: 17px; }
  .activity-panel { padding-inline: 10px; }
  .panel-header { flex-wrap: wrap; }
  .activity-panel-header { align-items: flex-start; }
  .activity-controls { width: 100%; justify-content: space-between; }
  .chart-legend small { width: 100%; margin-left: 0; }
  .activity-chart, .line-chart { height: 210px; }
  .line-plot { height: 184px; }
  .activity-chart { gap: 4px; }
  .chart-bars { gap: 2px; }
  .category-row { grid-template-columns: 80px minmax(60px, 1fr) 64px; gap: 8px; }
}

@media (max-width: 430px) {
  .storage-overview { grid-template-columns: 58px minmax(0, 1fr); gap: 9px; }
  .storage-ring { width: 58px; height: 58px; }
  .storage-values { gap: 5px; }
  .storage-values strong { font-size: 13px; }
  .metric-card:not(.metric-card-storage) { min-height: 116px; }
  .metric-heading { align-items: flex-start; flex-direction: column; gap: 6px; }
  .quick-grid { grid-template-columns: 1fr; }
  .activity-row time { display: none; }
}

@media (prefers-reduced-motion: reduce) {
  .bar { transition-duration: .001ms; }
}
</style>
