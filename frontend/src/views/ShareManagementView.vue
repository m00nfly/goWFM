<template>
  <div class="workspace-page shares-page">
    <section class="workspace-surface">
      <header class="workspace-header">
        <div class="workspace-title-block">
          <h1 class="workspace-title">{{ isAdmin ? '管理分享' : '我的分享' }}</h1>
          <p class="workspace-subtitle">
            {{ isAdmin ? '查看和维护所有账户创建的分享链接' : '查看、复制和维护由您创建的文件分享' }}
          </p>
        </div>
        <div class="workspace-stat-grid shares-stats">
          <div class="workspace-stat">
            <div class="workspace-stat-label">全部分享</div>
            <div class="workspace-stat-value">{{ totalCount }}</div>
          </div>
          <div class="workspace-stat">
            <div class="workspace-stat-label">有效</div>
            <div class="workspace-stat-value">{{ validCount }}</div>
          </div>
          <div class="workspace-stat">
            <div class="workspace-stat-label">已过期</div>
            <div class="workspace-stat-value">{{ expiredCount }}</div>
          </div>
        </div>
      </header>

      <div v-if="isAdmin" class="workspace-toolbar shares-toolbar">
        <div class="workspace-toolbar-group">
          <n-input
            v-model:value="filterKeyword"
            class="shares-filter-name"
            placeholder="按文件名或分享 ID 筛选"
            clearable
            size="small"
          />
          <n-select
            v-model:value="filterOwnerId"
            class="shares-filter-owner"
            :options="ownerOptions"
            placeholder="按分享者筛选"
            clearable
            size="small"
          />
          <n-select
            v-model:value="filterStatus"
            class="shares-filter-status"
            :options="statusOptions"
            placeholder="按状态筛选"
            clearable
            size="small"
          />
        </div>
        <div class="workspace-count-pill">
          当前显示 <strong>{{ filteredShares.length }}</strong> 条
        </div>
      </div>

      <div class="shares-list" aria-live="polite">
        <template v-if="loading">
          <div
            v-for="index in 4"
            :key="index"
            class="share-card share-card-skeleton"
            :class="{ 'is-admin': isAdmin }"
          >
            <div class="share-main">
              <n-skeleton circle :width="40" :height="40" />
              <div class="share-main-content">
                <n-skeleton text style="width: 54%" />
                <n-skeleton text style="width: 78%" />
              </div>
            </div>
            <n-skeleton class="share-status" round style="width: 52px" />
            <div class="share-meta-grid">
              <n-skeleton v-for="item in 3" :key="item" text />
            </div>
            <n-skeleton v-if="isAdmin" class="share-owner-column" circle :width="26" :height="26" />
            <n-skeleton class="share-actions" text style="width: 112px" />
          </div>
        </template>

        <n-empty
          v-else-if="filteredShares.length === 0"
          class="shares-empty"
          :description="hasFilters ? '没有符合筛选条件的分享' : '暂无分享记录'"
        />

        <article
          v-for="share in filteredShares"
          v-else
          :id="`share-card-${share.id}`"
          :key="share.id"
          class="share-card"
          :class="{ 'is-admin': isAdmin, 'is-highlighted': share.id === highlightId }"
        >
          <div class="share-main">
            <span class="share-record-icon" role="img" aria-label="分享记录">
              <n-icon :size="22"><ShareSocialOutline /></n-icon>
            </span>
            <div class="share-main-content">
              <button class="share-file-button" type="button" @click="openFilesModal(share, $event)">
                <span class="share-file-title">{{ share.file_name || '未命名分享' }}</span>
                <span class="share-file-summary">
                  <n-icon :size="15"><DocumentsOutline /></n-icon>
                  {{ share.file_count }} 个文件 · 查看文件列表
                </span>
              </button>

              <div class="share-token-row">
                <span class="share-token-label">分享 ID</span>
                <code class="share-token">{{ share.token }}</code>
              </div>
            </div>
          </div>

          <div class="share-status">
            <n-tooltip trigger="hover" :style="{ whiteSpace: 'pre-line' }">
              <template #trigger>
                <n-tag :type="statusTagType(share.status)" size="small" round>
                  {{ statusLabel(share.status) }}
                </n-tag>
              </template>
              {{ getStatusTooltip(share) }}
            </n-tooltip>
          </div>

          <div class="share-meta-grid">
            <div class="share-meta-item">
              <span class="share-meta-label">访问次数</span>
              <strong class="share-meta-value tabular-nums">{{ share.access_count }}</strong>
            </div>
            <div class="share-meta-item">
              <span class="share-meta-label">创建时间</span>
              <strong class="share-meta-value tabular-nums">{{ share.created_at }}</strong>
            </div>
            <div class="share-meta-item">
              <span class="share-meta-label">到期时间</span>
              <strong class="share-meta-value tabular-nums">{{ share.expire_at || '永久有效' }}</strong>
            </div>
          </div>

          <div v-if="isAdmin" class="share-owner-column">
            <span class="share-meta-label">分享者</span>
            <n-tooltip trigger="hover" placement="top">
              <template #trigger>
                <span class="share-owner-avatar" tabindex="0">
                  <UserAvatar
                    :size="26"
                    :avatar="share.owner.avatar"
                    :name="share.owner.display_name || share.owner.username"
                  />
                </span>
              </template>
              {{ share.owner.username }}
            </n-tooltip>
          </div>

          <div class="share-actions" aria-label="分享操作">
            <n-tooltip trigger="hover" placement="top">
              <template #trigger>
                <n-button class="share-action-button copy" quaternary circle @click="copyLink(share)">
                  <template #icon><n-icon :size="19"><LinkOutline /></n-icon></template>
                </n-button>
              </template>
              复制链接
            </n-tooltip>
            <n-tooltip trigger="hover" placement="top">
              <template #trigger>
                <n-button class="share-action-button edit" quaternary circle @click="openEditModal(share)">
                  <template #icon><n-icon :size="19"><CreateOutline /></n-icon></template>
                </n-button>
              </template>
              编辑
            </n-tooltip>
            <n-popconfirm @positive-click="handleDelete(share)">
              <template #trigger>
                <n-tooltip trigger="hover" placement="top">
                  <template #trigger>
                    <n-button class="share-action-button delete" quaternary circle>
                      <template #icon><n-icon :size="19"><TrashOutline /></n-icon></template>
                    </n-button>
                  </template>
                  删除
                </n-tooltip>
              </template>
              确认删除此分享链接？
            </n-popconfirm>
          </div>
        </article>
      </div>
    </section>
  </div>

  <n-modal
    v-model:show="showFilesModal"
    preset="card"
    :title="filesModalTitle"
    :auto-focus="false"
    class="files-modal"
    style="width: 640px; max-width: 92vw"
    @after-enter="focusFilesModalContent"
    @after-leave="restoreFilesModalTrigger"
  >
    <div ref="filesModalContentRef" class="files-modal-content" tabindex="-1" aria-label="分享文件列表">
      <div v-if="modalFiles.length > 0" class="files-modal-list">
        <button
          v-for="file in modalFiles"
          :key="file.id"
          class="file-item"
          type="button"
          @click="navigateToFile(file.file_path)"
        >
          <span class="file-name-group">
            <n-icon :size="19"><DocumentOutline /></n-icon>
            <span class="file-name-link">{{ file.file_name }}</span>
          </span>
          <span class="file-download-count">下载 {{ file.download_count }} 次</span>
        </button>
      </div>
      <n-empty v-else description="暂无文件" />
    </div>
  </n-modal>

  <n-modal
    v-model:show="showEditModal"
    preset="dialog"
    title="编辑分享"
    positive-text="保存"
    negative-text="取消"
    :positive-button-props="{ loading: editLoading }"
    :mask-closable="false"
    @positive-click="handleEditSave"
    @negative-click="showEditModal = false"
  >
    <n-form label-placement="top">
      <n-form-item label="分享名称">
        <n-input v-model:value="editName" placeholder="分享名称" clearable :maxlength="100" />
      </n-form-item>
      <n-form-item label="有效期（天）">
        <n-input-number
          v-model:value="editExpireDays"
          :min="0"
          :max="365"
          placeholder="0 表示永久有效"
          style="width: 100%"
        />
      </n-form-item>
    </n-form>
  </n-modal>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  NButton,
  NEmpty,
  NForm,
  NFormItem,
  NIcon,
  NInput,
  NInputNumber,
  NModal,
  NPopconfirm,
  NSelect,
  NSkeleton,
  NTag,
  NTooltip,
  useMessage,
} from 'naive-ui'
import {
  CreateOutline,
  DocumentOutline,
  DocumentsOutline,
  LinkOutline,
  ShareSocialOutline,
  TrashOutline,
} from '@vicons/ionicons5'
import api from '@/api'
import UserAvatar from '@/components/UserAvatar.vue'
import { useUserStore } from '@/stores/user'
import { copyToClipboard } from '@/utils/clipboard'

interface ShareFile {
  id: number
  file_name: string
  file_path: string
  download_count: number
}

interface ShareOwner {
  id: number
  username: string
  display_name: string
  avatar: string
}

interface ShareItem {
  id: number
  token: string
  name: string
  file_name: string
  file_count: number
  files: ShareFile[]
  owner: ShareOwner
  status: 'valid' | 'expired' | 'deleted'
  created_at: string
  expire_at: string | null
  access_count: number
}

const router = useRouter()
const route = useRoute()
const message = useMessage()
const userStore = useUserStore()
const isAdmin = computed(() => userStore.user?.is_admin === true)

const shares = ref<ShareItem[]>([])
const loading = ref(false)
const highlightId = ref<number | null>(null)
let highlightTimer: ReturnType<typeof setTimeout> | null = null

const filterKeyword = ref('')
const filterOwnerId = ref<number | null>(null)
const filterStatus = ref<string | null>(null)
const statusOptions = [
  { label: '有效', value: 'valid' },
  { label: '已过期', value: 'expired' },
  { label: '无效', value: 'deleted' },
]

const ownerOptions = computed(() => {
  const owners = new Map<number, ShareOwner>()
  for (const share of shares.value) owners.set(share.owner.id, share.owner)
  return Array.from(owners.values()).map(owner => ({
    label: owner.username,
    value: owner.id,
  }))
})

const filteredShares = computed(() => shares.value.filter(share => {
  const keyword = filterKeyword.value.trim().toLocaleLowerCase()
  if (keyword) {
    const matchesFile = share.files.some(file => file.file_name.toLocaleLowerCase().includes(keyword))
    const matchesShare = share.file_name.toLocaleLowerCase().includes(keyword) || share.token.toLocaleLowerCase().includes(keyword)
    if (!matchesFile && !matchesShare) return false
  }
  if (filterOwnerId.value !== null && share.owner.id !== filterOwnerId.value) return false
  if (filterStatus.value && share.status !== filterStatus.value) return false
  return true
}))

const hasFilters = computed(() => Boolean(filterKeyword.value || filterOwnerId.value !== null || filterStatus.value))
const totalCount = computed(() => shares.value.length)
const validCount = computed(() => shares.value.filter(share => share.status === 'valid').length)
const expiredCount = computed(() => shares.value.filter(share => share.status === 'expired').length)

const showFilesModal = ref(false)
const modalFiles = ref<ShareFile[]>([])
const activeShareName = ref('')
const filesModalContentRef = ref<HTMLElement | null>(null)
let filesModalTrigger: HTMLElement | null = null
const filesModalTitle = computed(() => `${activeShareName.value || '分享文件'} · ${modalFiles.value.length} 项`)

const showEditModal = ref(false)
const editLoading = ref(false)
const editId = ref(0)
const editName = ref('')
const editExpireDays = ref(7)

function statusLabel(status: ShareItem['status']) {
  return status === 'expired' ? '已过期' : status === 'deleted' ? '无效' : '有效'
}

function statusTagType(status: ShareItem['status']): 'success' | 'error' | 'warning' {
  return status === 'expired' ? 'error' : status === 'deleted' ? 'warning' : 'success'
}

function getStatusTooltip(share: ShareItem): string {
  if (share.status === 'deleted') {
    return share.expire_at ? `到期时间：${share.expire_at}\n源文件已删除` : '到期时间：永久\n源文件已删除'
  }
  if (!share.expire_at) return '永久有效'
  const remaining = Math.ceil((new Date(share.expire_at).getTime() - Date.now()) / 86400000)
  return remaining > 0
    ? `到期时间：${share.expire_at}\n剩余：${remaining} 天`
    : `到期时间：${share.expire_at}\n已过期`
}

function openFilesModal(share: ShareItem, event: MouseEvent) {
  filesModalTrigger = event.currentTarget instanceof HTMLElement ? event.currentTarget : null
  filesModalTrigger?.blur()
  activeShareName.value = share.file_name
  modalFiles.value = share.files
  showFilesModal.value = true
}

function focusFilesModalContent() {
  filesModalContentRef.value?.focus({ preventScroll: true })
}

function restoreFilesModalTrigger() {
  if (filesModalTrigger?.isConnected) {
    filesModalTrigger.focus({ preventScroll: true })
  }
  filesModalTrigger = null
}

function navigateToFile(filePath: string) {
  if (!filePath) {
    message.error('文件路径无效，无法跳转')
    return
  }
  const separatorIndex = filePath.lastIndexOf('/')
  const dirPath = separatorIndex > 0 ? filePath.slice(0, separatorIndex) : '/'
  const fileName = filePath.slice(separatorIndex + 1)
  showFilesModal.value = false
  router.push({ path: '/', query: { path: dirPath, highlight: fileName } })
}

async function applyHighlight() {
  const value = Number(route.query.highlightId)
  highlightId.value = Number.isFinite(value) && value > 0 ? value : null
  if (highlightTimer) clearTimeout(highlightTimer)
  if (highlightId.value === null) return

  await nextTick()
  document.getElementById(`share-card-${highlightId.value}`)?.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
  highlightTimer = setTimeout(() => {
    highlightId.value = null
  }, 4000)
}

async function fetchShares() {
  loading.value = true
  try {
    const response = await api.get<ShareItem[]>('/api/shares')
    shares.value = response.data
    await applyHighlight()
  } catch (error: any) {
    message.error(error.response?.data?.error || '获取分享列表失败')
  } finally {
    loading.value = false
  }
}

async function copyLink(share: ShareItem) {
  const ok = await copyToClipboard(`${window.location.origin}/share/${share.token}`)
  ok ? message.success('链接已复制') : message.error('复制失败，请手动复制')
}

async function handleDelete(share: ShareItem) {
  try {
    await api.delete(`/api/shares/${share.id}`)
    message.success('分享链接已删除')
    if (share.owner.id === userStore.user?.id) {
      userStore.onShareDeleted(share.status === 'expired' ? 'expired' : 'valid')
    }
    await fetchShares()
  } catch (error: any) {
    message.error(error.response?.data?.error || '删除失败')
  }
}

function openEditModal(share: ShareItem) {
  editId.value = share.id
  editName.value = share.name || share.file_name
  editExpireDays.value = 7
  showEditModal.value = true
}

async function handleEditSave() {
  editLoading.value = true
  try {
    await api.put(`/api/shares/${editId.value}`, {
      name: editName.value,
      expire_days: editExpireDays.value,
    })
    message.success('分享已更新')
    showEditModal.value = false
    await fetchShares()
  } catch (error: any) {
    message.error(error.response?.data?.error || '更新失败')
  } finally {
    editLoading.value = false
  }
}

onMounted(fetchShares)
watch(() => route.query.highlightId, applyHighlight)
onUnmounted(() => {
  if (highlightTimer) clearTimeout(highlightTimer)
})
</script>

<style scoped>
.shares-stats {
  width: min(420px, 100%);
}

.shares-filter-name {
  width: 230px;
}

.shares-filter-owner {
  width: 160px;
}

.shares-filter-status {
  width: 130px;
}

.shares-list {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 12px;
  scrollbar-gutter: stable;
}

.share-card {
  display: grid;
  grid-template-areas: "main status meta actions";
  grid-template-columns: minmax(280px, 1.2fr) 72px minmax(360px, 1fr) auto;
  align-items: center;
  gap: 18px;
  padding: 14px 14px 14px 16px;
  border-radius: var(--workspace-radius-lg);
  background: color-mix(in srgb, var(--workspace-surface-soft) 52%, var(--workspace-surface));
  box-shadow:
    0 0 0 1px rgba(0, 0, 0, 0.055),
    0 1px 2px -1px rgba(39, 55, 82, 0.08),
    0 8px 20px rgba(39, 55, 82, 0.055);
  transition-property: box-shadow, background-color;
  transition-duration: 180ms;
  transition-timing-function: ease-out;
}

.share-card.is-admin {
  grid-template-areas: "main status meta owner actions";
  grid-template-columns: minmax(280px, 1.15fr) 72px minmax(350px, 1fr) 62px auto;
}

.share-card:hover {
  box-shadow:
    0 0 0 1px rgba(var(--workspace-accent-rgb), 0.2),
    0 2px 4px -1px rgba(39, 55, 82, 0.1),
    0 12px 26px rgba(39, 55, 82, 0.09);
}

.share-card.is-highlighted {
  background: color-mix(in srgb, var(--workspace-row-selected) 68%, var(--workspace-surface));
  box-shadow:
    0 0 0 2px rgba(var(--workspace-accent-rgb), 0.48),
    0 12px 28px rgba(var(--workspace-accent-rgb), 0.12);
}

:global(.dark) .share-card {
  box-shadow: 0 0 0 1px rgba(255, 255, 255, 0.08);
}

:global(.dark) .share-card:hover {
  box-shadow: 0 0 0 1px rgba(255, 255, 255, 0.13);
}

.share-card-skeleton {
  min-height: 102px;
}

.share-main {
  grid-area: main;
  min-width: 0;
  display: grid;
  grid-template-columns: 40px minmax(0, 1fr);
  align-items: center;
  gap: 12px;
}

.share-main-content {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 9px;
}

.share-record-icon {
  width: 40px;
  height: 40px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--workspace-radius-md);
  background: rgba(var(--workspace-accent-rgb), 0.09);
  color: var(--workspace-accent);
  box-shadow:
    0 0 0 1px rgba(var(--workspace-accent-rgb), 0.13),
    0 4px 12px rgba(var(--workspace-accent-rgb), 0.08);
}

.share-owner-avatar {
  width: 40px;
  height: 40px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  outline: none;
}

.share-owner-avatar:focus-visible {
  box-shadow: 0 0 0 3px rgba(var(--workspace-accent-rgb), 0.22);
}

.share-file-button {
  min-width: 0;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
  padding: 2px 0;
  border: 0;
  background: transparent;
  color: inherit;
  cursor: pointer;
  text-align: left;
  transition-property: scale;
  transition-duration: 150ms;
  transition-timing-function: ease-out;
}

.share-file-button:active,
.share-action-button:active,
.file-item:active {
  scale: 0.96;
}

.share-file-title {
  max-width: 100%;
  overflow: hidden;
  color: var(--workspace-text);
  font-size: 14px;
  font-weight: 730;
  line-height: 1.4;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.share-file-button:hover .share-file-title {
  color: var(--workspace-accent);
}

.share-file-button:focus-visible {
  border-radius: var(--workspace-radius-sm);
  outline: 2px solid rgba(var(--workspace-accent-rgb), 0.42);
  outline-offset: 3px;
}

.share-file-summary {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  color: var(--workspace-text-muted);
  font-size: 12px;
  line-height: 1.35;
}

.share-token-row {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 8px;
}

.share-token-label {
  color: var(--workspace-text-soft);
  font-size: 11px;
  white-space: nowrap;
}

.share-token {
  overflow: hidden;
  color: var(--workspace-text-muted);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.share-meta-grid {
  grid-area: meta;
  min-width: 0;
  display: grid;
  grid-template-columns: minmax(72px, 0.55fr) minmax(144px, 1fr) minmax(144px, 1fr);
  gap: 16px;
}

.share-status {
  grid-area: status;
  min-width: 0;
  align-self: stretch;
  display: flex;
  align-items: center;
  justify-content: center;
}

.share-owner-column {
  grid-area: owner;
  min-width: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 3px;
}

.share-meta-item {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.share-meta-label {
  color: var(--workspace-text-soft);
  font-size: 11px;
  line-height: 1.3;
}

.share-meta-value {
  overflow: hidden;
  color: var(--workspace-text);
  font-size: 12px;
  font-weight: 620;
  line-height: 1.4;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tabular-nums {
  font-variant-numeric: tabular-nums;
}

.share-actions {
  grid-area: actions;
  display: flex;
  align-items: center;
  gap: 2px;
}

.share-action-button {
  width: 40px;
  height: 40px;
  transition-property: scale, background-color;
  transition-duration: 150ms;
  transition-timing-function: ease-out;
}

.share-action-button.copy {
  color: var(--workspace-accent);
}

.share-action-button.edit {
  color: #d68b12;
}

.share-action-button.delete {
  color: #d03050;
}

.shares-empty {
  margin: auto;
  padding: 42px 16px;
}

.files-modal-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.files-modal-content {
  border-radius: var(--workspace-radius-md);
  outline: none;
}

.files-modal-content:focus-visible {
  box-shadow: 0 0 0 3px rgba(var(--workspace-accent-rgb), 0.2);
}

.file-item {
  width: 100%;
  min-height: 48px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 9px 12px;
  border: 0;
  border-radius: var(--workspace-radius-md);
  background: var(--workspace-surface-soft);
  color: inherit;
  cursor: pointer;
  text-align: left;
  box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.055);
  transition-property: scale, box-shadow, background-color;
  transition-duration: 150ms;
  transition-timing-function: ease-out;
}

.file-item:hover {
  background: var(--workspace-row-hover);
  box-shadow: 0 0 0 1px rgba(var(--workspace-accent-rgb), 0.22);
}

.file-item:focus-visible {
  outline: 2px solid rgba(var(--workspace-accent-rgb), 0.45);
  outline-offset: 2px;
}

.file-name-group {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 9px;
  color: var(--workspace-accent);
}

.file-name-link {
  overflow: hidden;
  font-size: 13px;
  font-weight: 620;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-download-count {
  flex: 0 0 auto;
  color: var(--workspace-text-muted);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

@media (max-width: 1180px) {
  .share-card {
    grid-template-areas:
      "main status meta"
      "actions actions actions";
    grid-template-columns: minmax(240px, 1fr) 68px minmax(320px, 1fr);
  }

  .share-card.is-admin {
    grid-template-areas:
      "main status meta owner"
      "actions actions actions actions";
    grid-template-columns: minmax(240px, 1fr) 68px minmax(300px, 1fr) 58px;
  }

  .share-actions {
    justify-content: flex-end;
    padding-top: 2px;
  }
}

@media (max-width: 768px) {
  .shares-stats {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .shares-filter-name,
  .shares-filter-owner,
  .shares-filter-status {
    width: 100%;
  }

  .share-card {
    grid-template-areas:
      "main status"
      "meta meta"
      "actions actions";
    grid-template-columns: minmax(0, 1fr) auto;
    gap: 14px;
    padding: 14px;
  }

  .share-card.is-admin {
    grid-template-areas:
      "main status"
      "meta meta"
      "owner actions";
    grid-template-columns: minmax(0, 1fr) auto;
  }

  .share-status {
    min-width: 58px;
  }

  .share-meta-grid {
    grid-template-columns: 0.6fr 1fr;
  }

  .share-meta-item:last-child {
    grid-column: 1 / -1;
  }

  .share-actions {
    justify-content: flex-start;
    border-top: 1px solid var(--workspace-border-soft);
    padding-top: 10px;
  }

  .share-owner-column {
    align-items: flex-start;
    border-top: 1px solid var(--workspace-border-soft);
    padding-top: 10px;
  }
}

@media (max-width: 480px) {
  .shares-list {
    padding: 9px;
  }

  .share-main {
    grid-template-columns: 36px minmax(0, 1fr);
    gap: 10px;
  }

  .share-record-icon {
    width: 36px;
    height: 36px;
  }

  .file-item {
    align-items: flex-start;
    flex-direction: column;
    gap: 5px;
  }
}
</style>
