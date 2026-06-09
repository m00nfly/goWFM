<template>
  <div class="main-layout" :class="{ dark: themeStore.isDark }">
    <!-- 顶部导航栏 - 固定毛玻璃效果 -->
    <header class="top-header">
      <div class="header-inner">
        <!-- Logo -->
        <div class="header-brand" @click="router.push('/')">
          <div class="brand-icon">
            <n-icon size="20" color="#fff"><FolderOpenOutline /></n-icon>
          </div>
          <span class="brand-text">{{ orgName || 'goWFM' }}</span>
        </div>

        <!-- 右侧操作区 -->
        <div class="header-actions">
          <!-- 导航图标 - 宽屏 -->
          <div v-show="!isNarrow" class="nav-icons">
            <n-tooltip trigger="hover" placement="bottom">
              <template #trigger>
                <button class="nav-icon-btn" :class="{ active: activeMenuKey === '/' }" @click="router.push('/')">
                  <n-icon size="22"><FolderOpenOutline /></n-icon>
                </button>
              </template>
              文件管理
            </n-tooltip>

            <n-tooltip v-if="userStore.user?.is_admin || userStore.hasPermission(8)" trigger="hover" placement="bottom">
              <template #trigger>
                <button class="nav-icon-btn" :class="{ active: activeMenuKey === shareMenuKey }" @click="router.push(shareMenuKey)">
                  <n-badge :value="shareBadgeCount" :type="shareBadgeType" :show="shareBadgeCount > 0" :offset="[-4, 4]">
                    <n-icon size="22"><ShareSocialOutline /></n-icon>
                  </n-badge>
                </button>
              </template>
              {{ userStore.user?.is_admin ? '分享管理' : '我的分享' }}
            </n-tooltip>

            <n-tooltip v-if="userStore.hasPermission(16)" trigger="hover" placement="bottom">
              <template #trigger>
                <button class="nav-icon-btn" :class="{ active: activeMenuKey === '/logs' }" @click="router.push('/logs')">
                  <n-icon size="22"><DocumentTextOutline /></n-icon>
                </button>
              </template>
              操作日志
            </n-tooltip>

            <n-tooltip v-if="userStore.user?.is_admin" trigger="hover" placement="bottom">
              <template #trigger>
                <button class="nav-icon-btn" :class="{ active: activeMenuKey === '/admin/users' }" @click="router.push('/admin/users')">
                  <n-icon size="22"><PeopleOutline /></n-icon>
                </button>
              </template>
              用户管理
            </n-tooltip>

            <n-tooltip v-if="userStore.user?.is_admin" trigger="hover" placement="bottom">
              <template #trigger>
                <button class="nav-icon-btn" :class="{ active: activeMenuKey === '/admin/settings' }" @click="router.push('/admin/settings')">
                  <n-icon size="22"><SettingsOutline /></n-icon>
                </button>
              </template>
              系统设置
            </n-tooltip>
          </div>

          <!-- 导航图标 - 窄屏折叠菜单 -->
          <n-popselect
            v-if="isNarrow"
            v-model:value="popNavValue"
            :options="popNavOptions"
            trigger="click"
            @update:value="onPopNavSelect"
          >
            <button class="nav-icon-btn">
              <n-badge :value="shareBadgeCount" :type="shareBadgeType" :show="shareBadgeCount > 0" dot :offset="[-2, 2]">
                <n-icon size="22"><MenuOutline /></n-icon>
              </n-badge>
            </button>
          </n-popselect>

          <!-- 主题切换 -->
          <n-tooltip trigger="hover" placement="bottom">
            <template #trigger>
              <button class="nav-icon-btn" @click="themeStore.toggleTheme()">
                <n-icon size="22"><SunnyOutline v-if="themeStore.isDark" /><MoonOutline v-else /></n-icon>
              </button>
            </template>
            {{ themeStore.isDark ? '切换亮色' : '切换暗色' }}
          </n-tooltip>

          <!-- 分隔线 -->
          <div class="header-divider"></div>

          <!-- 用户下拉 -->
          <n-dropdown trigger="click" :options="userDropdownOptions" @select="onUserAction">
            <div class="user-trigger">
              <n-avatar
                round
                :size="32"
                :style="{ backgroundColor: '#3b82f6', cursor: 'pointer', fontSize: '14px' }"
              >
                {{ avatarLetter }}
              </n-avatar>
              <span class="user-display-name">{{ displayName }}</span>
            </div>
          </n-dropdown>
        </div>
      </div>
    </header>

    <!-- 主内容区 -->
    <main class="main-content">
      <div class="content-wrapper">
        <router-view />
      </div>
      <div class="content-footer">
        <div class="footer-content">
          <template v-if="orgLink">
            <a :href="orgLink" target="_blank" class="footer-org-link">{{ orgName || orgLink }}</a>
            <span class="footer-separator">|</span>
          </template>
          <template v-else-if="orgName">
            <span class="footer-org-text">{{ orgName }}</span>
            <span class="footer-separator">|</span>
          </template>
          <a :href="appLink" target="_blank" class="footer-app-link">goWFM</a>
          <a v-if="appGithub" :href="appGithub" target="_blank" class="footer-github-link">
            <n-icon :size="14"><LogoGithub /></n-icon>
          </a>
          <span v-if="version" class="footer-version">ver: {{ version }}</span>
        </div>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, h, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import {
  NDropdown, NIcon, NAvatar, NTooltip, NBadge, NPopselect,
} from 'naive-ui'
import {
  FolderOpenOutline,
  ShareSocialOutline,
  DocumentTextOutline,
  PeopleOutline,
  SettingsOutline,
  LogOutOutline,
  LogoGithub,
  SunnyOutline,
  MoonOutline,
  MenuOutline,
} from '@vicons/ionicons5'
import { useUserStore } from '@/stores/user'
import { useThemeStore } from '@/stores/theme'
import api from '@/api'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()
const themeStore = useThemeStore()

const version = ref('')
const orgName = ref('')
const orgLink = ref('')
const appLink = ref('https://gowfm.dev')
const appGithub = ref('https://github.com/m00nfly/gowfm')

// ---------- 响应式导航 ----------

const NAV_BREAKPOINT = 768
const windowWidth = ref(window.innerWidth)
const isNarrow = computed(() => windowWidth.value < NAV_BREAKPOINT)

function onResize() {
  windowWidth.value = window.innerWidth
}

// ---------- 分享 Badge ----------

const shareBadgeCount = computed(() => {
  if (userStore.shareExpiredCount > 0) return userStore.shareExpiredCount
  if (userStore.shareValidCount > 0) return userStore.shareValidCount
  return 0
})

const shareBadgeType = computed<'error' | 'info'>(() => {
  if (userStore.shareExpiredCount > 0) return 'error'
  return 'info'
})

// ---------- 用户显示 ----------

const displayName = computed(() =>
  userStore.user?.display_name || userStore.user?.username || '用户'
)

const avatarLetter = computed(() => {
  const name = displayName.value
  return name ? name.charAt(0).toUpperCase() : 'U'
})

// ---------- 导航配置 ----------

const shareMenuKey = computed(() =>
  userStore.user?.is_admin ? '/admin/shares' : '/shares'
)

// 高亮的导航 key
const activeMenuKey = computed(() => {
  const p = route.path
  if (p.startsWith('/admin/shares')) return '/admin/shares'
  if (p.startsWith('/admin/users')) return '/admin/users'
  if (p.startsWith('/admin/settings')) return '/admin/settings'
  if (p === '/shares') return '/shares'
  if (p === '/logs') return '/logs'
  if (p === '/settings') return '/settings'
  return '/'
})

// ---------- 窄屏折叠导航 ----------

const popNavValue = ref<string | null>(null)

const popNavOptions = computed(() => {
  const opts: Array<{ label: string; value: string }> = [
    { label: '文件管理', value: '/' },
  ]
  if (userStore.user?.is_admin || userStore.hasPermission(8)) {
    const label = userStore.user?.is_admin ? '分享管理' : '我的分享'
    const badgeText = shareBadgeCount.value > 0
      ? ` (${shareBadgeCount.value})`
      : ''
    opts.push({ label: label + badgeText, value: shareMenuKey.value })
  }
  if (userStore.hasPermission(16)) {
    opts.push({ label: '操作日志', value: '/logs' })
  }
  if (userStore.user?.is_admin) {
    opts.push({ label: '用户管理', value: '/admin/users' })
    opts.push({ label: '系统设置', value: '/admin/settings' })
  }
  return opts
})

function onPopNavSelect(value: string) {
  router.push(value)
  popNavValue.value = null
}

// 用户下拉菜单
const userDropdownOptions = computed(() => [
  {
    label: '个人设置',
    key: 'settings',
    icon: () => h(NIcon, null, () => h(SettingsOutline)),
  },
  { type: 'divider', key: 'd2' },
  {
    label: '退出登录',
    key: 'logout',
    icon: () => h(NIcon, null, () => h(LogOutOutline)),
  },
])

// ---------- 事件处理 ----------

async function onUserAction(key: string) {
  if (key === 'settings') {
    router.push('/settings')
  } else if (key === 'logout') {
    try { await api.post('/api/auth/logout') } catch { /* ignore */ }
    userStore.logout()
    router.push('/login')
  }
}

// ---------- 生命周期 ----------

onMounted(async () => {
  window.addEventListener('resize', onResize)

  try {
    const res = await api.get('/api/config/info')
    orgName.value = res.data.org_name || ''
    orgLink.value = res.data.org_link || ''
    version.value = res.data.version || ''
  } catch { /* ignore */ }
})

onUnmounted(() => {
  window.removeEventListener('resize', onResize)
})
</script>

<style scoped>
.main-layout {
  min-height: 100vh;
  background: #f8fafc;
  transition: background 0.3s ease;
}

.dark.main-layout {
  background: #0f172a;
}

/* ---- 顶部导航栏 - 毛玻璃效果 ---- */
.top-header {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: 100;
  backdrop-filter: blur(12px);
  background-color: rgba(255, 255, 255, 0.85);
  border-bottom: 1px solid #e2e8f0;
  transition: background-color 0.3s ease, border-color 0.3s ease;
}

.dark .top-header {
  background-color: rgba(15, 23, 42, 0.85);
  border-bottom: 1px solid #1e293b;
}

.header-inner {
  max-width: 1280px;
  margin: 0 auto;
  padding: 0 24px;
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

/* ---- Logo ---- */
.header-brand {
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  user-select: none;
}

.brand-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  background: #3b82f6;
  padding: 8px;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.3);
}

.brand-text {
  font-size: 18px;
  font-weight: 700;
  color: #0f172a;
  letter-spacing: -0.025em;
  transition: color 0.3s ease;
}

.dark .brand-text {
  color: #f1f5f9;
}

@media (max-width: 640px) {
  .brand-text {
    display: none;
  }
}

/* ---- 右侧操作区 ---- */
.header-actions {
  display: flex;
  align-items: center;
  gap: 4px;
}

/* ---- 导航图标 ---- */
.nav-icons {
  display: flex;
  align-items: center;
  gap: 2px;
}

.nav-icon-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 38px;
  height: 38px;
  border: none;
  background: transparent;
  border-radius: 8px;
  cursor: pointer;
  color: #0f172a;
  transition: all 0.2s ease;
  font-size: 22px;
}

.nav-icon-btn:hover {
  background: #f1f5f9;
  color: #3b82f6;
}

.nav-icon-btn.active {
  background: #eff6ff;
  color: #3b82f6;
}

.dark .nav-icon-btn {
  color: #f1f5f9
}

.dark .nav-icon-btn:hover {
  background: #1e293b;
  color: #60a5fa;
}

.dark .nav-icon-btn.active {
  background: #1e3a5f;
  color: #60a5fa;
}

/* ---- 分隔线 ---- */
.header-divider {
  width: 1px;
  height: 24px;
  background: #e2e8f0;
  margin: 0 8px;
  transition: background 0.3s ease;
}

.dark .header-divider {
  background: #334155;
}

/* ---- 用户触发 ---- */
.user-trigger {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  padding: 4px 8px 4px 4px;
  border-radius: 8px;
  transition: background-color 0.2s;
}

.user-trigger:hover {
  background-color: #f1f5f9;
}

.dark .user-trigger:hover {
  background-color: #1e293b;
}

.user-display-name {
  font-size: 13px;
  font-weight: 500;
  color: #334155;
  max-width: 100px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  transition: color 0.3s ease;
}

.dark .user-display-name {
  color: #cbd5e1;
}

@media (max-width: 768px) {
  .user-display-name {
    display: none;
  }
}

/* ---- 主内容区 ---- */
.main-content {
  padding-top: 56px;
  height: 100vh;
  display: flex;
  flex-direction: column;
}

.content-wrapper {
  flex: 1;
  width: 95%;
  max-width: 1280px;
  margin: 0 auto;
  padding: 24px 24px 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

/* ---- 底部版权 ---- */
.content-footer {
  text-align: center;
  padding: 18px;
  font-size: 12px;
  color: #94a3b8;
  transition: color 0.3s ease;
}

.dark .content-footer {
  color: #475569;
}

.footer-content {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.footer-separator {
  color: #cbd5e1;
}

.dark .footer-separator {
  color: #334155;
}

.footer-app-link {
  color: #3b82f6;
  text-decoration: none;
  font-weight: 500;
}

.footer-app-link:hover {
  text-decoration: underline;
}

.footer-version {
  color: #cbd5e1;
  font-size: 11px;
}

.dark .footer-version {
  color: #334155;
}

.footer-org-link {
  color: #3b82f6;
  text-decoration: none;
}

.footer-org-link:hover {
  text-decoration: underline;
}

.footer-org-text {
  color: #64748b;
}

.dark .footer-org-text {
  color: #64748b;
}

.footer-github-link {
  color: #64748b;
  display: inline-flex;
  align-items: center;
  transition: color 0.2s;
}

.footer-github-link:hover {
  color: #334155;
}

.dark .footer-github-link:hover {
  color: #cbd5e1;
}
</style>
