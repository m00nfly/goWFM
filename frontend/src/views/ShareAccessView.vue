<template>
  <div class="workspace-page share-access-content" :class="{ dark: themeStore.isDark }">
    <!-- 主内容 -->
    <main class="share-main">
      <!-- 错误状态 -->
      <div v-if="error" class="error-container">
        <n-result status="error" :title="error" description="分享链接无效或已过期" />
      </div>

      <!-- 加载中 -->
      <n-spin v-else :show="loading" class="share-spin">
        <template v-if="!loading && shareInfo">
          <!-- 分享信息头部 -->
          <div class="share-header">
            <div class="share-header-left">
              <n-icon size="52" color="var(--theme-color, #3b82f6)" class="share-avatar-icon"><PersonCircle /></n-icon>
              <div class="share-meta">
                <h1 class="share-title">{{ shareInfo.name || '文件分享' }}</h1>
                <div class="share-meta-items">
                  <span class="meta-item">
                    <n-icon size="16"><PersonOutline /></n-icon>
                    {{ shareInfo.owner_name || '匿名' }}
                  </span>
                  <span class="meta-item">
                    <n-icon size="16"><CalendarOutline /></n-icon>
                    {{ expireDisplay }}
                  </span>
                  <span class="meta-item">
                    <n-icon size="16"><DocumentsOutline /></n-icon>
                    {{ shareInfo.file_count }}个文件 ({{ formatSize(shareInfo.total_size) }})
                  </span>
                </div>
              </div>
            </div>
          </div>

          <!-- 文件卡片网格 -->
          <div class="file-surface">
            <div class="file-grid" :class="{ 'single-file': shareInfo.files.length === 1 }">
              <div v-for="file in shareInfo.files" :key="file.id" class="file-card">
                <div class="file-icon-area" :style="{ color: getFileColor(file.file_name) }">
                  <n-icon :size="28"><component :is="getFileIconComp(file.file_name)" /></n-icon>
                </div>
                <div class="file-info">
                  <h3 class="file-name" :title="file.file_name">{{ file.file_name }}</h3>
                  <p class="file-size">{{ formatSize(file.file_size) }}</p>
                </div>
                <div class="file-actions">
                  <button
                    class="action-btn action-download"
                    :disabled="isFilePending(file.id)"
                    :aria-label="`下载 ${file.file_name}`"
                    @click="downloadFile(file)"
                  >
                    <n-icon size="16"><DownloadOutline /></n-icon>
                    <span>{{ pendingAction[file.id] === 'download' ? '获取中' : '下载' }}</span>
                  </button>
                  <n-tooltip trigger="hover" placement="top" :delay="300">
                    <template #trigger>
                      <button
                        class="action-btn action-link"
                        :disabled="isFilePending(file.id)"
                        :aria-label="`获取 ${file.file_name} 的一次性下载链接`"
                        @click="copyDownloadLink(file)"
                      >
                        <n-icon size="16"><LinkOutline /></n-icon>
                        <span>{{ pendingAction[file.id] === 'link' ? '获取中' : '链接' }}</span>
                      </button>
                    </template>
                    链接仅在有效期内使用一次；再次下载请重新点击“链接”获取新链接
                  </n-tooltip>
                </div>
              </div>
            </div>
          </div>
        </template>
      </n-spin>
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { NIcon, NResult, NSpin, NTooltip, useMessage } from 'naive-ui'
import api from '@/api'
import { useThemeStore } from '@/stores/theme'
import { formatSize } from '@/utils/format'
import { copyToClipboard } from '@/utils/clipboard'
import {
  PersonCircle,
  PersonOutline,
  CalendarOutline,
  DocumentsOutline,
  DownloadOutline,
  LinkOutline,
  Image,
  DocumentText,
  Document,
  CodeSlash,
  Archive,
  Videocam,
  MusicalNotes,
} from '@vicons/ionicons5'

const route = useRoute()
const message = useMessage()
const themeStore = useThemeStore()

// 分享数据
const loading = ref(true)
const error = ref('')

interface ShareInfo {
  name: string
  owner_name: string
  expire_at: string | null
  created_at: string
  file_count: number
  total_size: number
  files: Array<{
    id: number
    file_name: string
    file_size: number
    download_count: number
  }>
}
const shareInfo = ref<ShareInfo | null>(null)
type ShareFile = ShareInfo['files'][number]
type PendingAction = 'download' | 'link'
const pendingAction = ref<Record<number, PendingAction | undefined>>({})

const expireDisplay = computed(() => {
  if (!shareInfo.value?.expire_at) return '永久有效'
  const d = new Date(shareInfo.value.expire_at)
  const year = d.getFullYear()
  const month = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${year}-${month}-${day} 过期`
})

// 文件图标逻辑
function getFileColor(name: string): string {
  const ext = name.toLowerCase().split('.').pop() || ''
  if (['jpg', 'jpeg', 'png', 'gif', 'svg', 'webp', 'bmp', 'ico', 'tiff', 'tif'].includes(ext))
    return '#52c41a'
  if (['mp4', 'avi', 'mov', 'mkv', 'webm', 'flv', 'wmv', 'm4v'].includes(ext))
    return '#fa8c16'
  if (['mp3', 'wav', 'flac', 'aac', 'ogg', 'wma', 'm4a', 'ape'].includes(ext))
    return '#eb2f96'
  if (['zip', 'tar', 'gz', 'rar', '7z', 'bz2', 'xz', 'tgz', 'zst'].includes(ext))
    return '#8c8c8c'
  if (['js', 'ts', 'jsx', 'tsx', 'vue', 'py', 'go', 'java', 'c', 'cpp', 'h', 'rs', 'rb',
       'php', 'swift', 'kt', 'html', 'css', 'scss', 'less', 'json', 'xml', 'yaml',
       'yml', 'toml', 'sql', 'sh', 'bash', 'cmd', 'ps1', 'bat'].includes(ext))
    return '#1890ff'
  if (['pdf', 'doc', 'docx', 'xls', 'xlsx', 'ppt', 'pptx', 'txt', 'md', 'log', 'csv', 'rtf'].includes(ext))
    return '#722ed1'
  return '#8c8c8c'
}

function getFileIconComp(name: string) {
  const ext = name.toLowerCase().split('.').pop() || ''
  if (['jpg', 'jpeg', 'png', 'gif', 'svg', 'webp', 'bmp', 'ico', 'tiff', 'tif'].includes(ext))
    return Image
  if (['mp4', 'avi', 'mov', 'mkv', 'webm', 'flv', 'wmv', 'm4v'].includes(ext))
    return Videocam
  if (['mp3', 'wav', 'flac', 'aac', 'ogg', 'wma', 'm4a', 'ape'].includes(ext))
    return MusicalNotes
  if (['zip', 'tar', 'gz', 'rar', '7z', 'bz2', 'xz', 'tgz', 'zst'].includes(ext))
    return Archive
  if (['js', 'ts', 'jsx', 'tsx', 'vue', 'py', 'go', 'java', 'c', 'cpp', 'h', 'rs', 'rb',
       'php', 'swift', 'kt', 'html', 'css', 'scss', 'less', 'json', 'xml', 'yaml',
       'yml', 'toml', 'sql', 'sh', 'bash', 'cmd', 'ps1', 'bat'].includes(ext))
    return CodeSlash
  if (['pdf', 'doc', 'docx', 'xls', 'xlsx', 'ppt', 'pptx', 'txt', 'md', 'log', 'csv', 'rtf'].includes(ext))
    return DocumentText
  return Document
}

function isFilePending(fileID: number) {
  return pendingAction.value[fileID] !== undefined
}

async function getTemporaryDownloadURL(file: ShareFile): Promise<string> {
  const token = route.params.token as string
  const response = await api.post(`/share/${token}/files/${file.id}/download-link`)
  return new URL(response.data.url, window.location.origin).toString()
}

// 操作
async function downloadFile(file: ShareFile) {
  if (isFilePending(file.id)) return
  pendingAction.value[file.id] = 'download'
  try {
    const url = await getTemporaryDownloadURL(file)
    window.location.assign(url)
  } catch (err: any) {
    message.error(err.response?.data?.error || '获取临时下载链接失败')
  } finally {
    delete pendingAction.value[file.id]
  }
}

async function copyDownloadLink(file: ShareFile) {
  if (isFilePending(file.id)) return
  pendingAction.value[file.id] = 'link'
  try {
    const url = await getTemporaryDownloadURL(file)
    const ok = await copyToClipboard(url)
    if (ok) {
      message.success('一次性下载链接已复制到剪贴板')
    } else {
      message.error('复制失败，请重新获取链接')
    }
  } catch (err: any) {
    message.error(err.response?.data?.error || '获取临时下载链接失败')
  } finally {
    delete pendingAction.value[file.id]
  }
}

// 初始化
onMounted(async () => {
  const token = route.params.token as string

  try {
    const shareRes = await api.get(`/share/${token}/info`)
    shareInfo.value = shareRes.data
  } catch (err: any) {
    error.value = err.response?.data?.error || '获取分享信息失败'
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.share-access-content {
  overflow: hidden;
  background: var(--workspace-bg);
}

/* ---- 主内容区 ---- */
.share-main {
  width: 100%;
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.error-container {
  display: flex;
  justify-content: center;
  padding-top: 44px;
}

.share-spin {
  flex: 1;
  min-height: 0;
}

.share-spin :deep(.n-spin-content) {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

/* ---- 分享信息头部 ---- */
.share-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 10px;
  padding: 12px;
  border: 1px solid var(--workspace-border);
  border-radius: var(--workspace-radius-xl);
  background:
    linear-gradient(180deg, rgba(var(--workspace-accent-rgb), 0.06), rgba(var(--workspace-accent-rgb), 0)),
    var(--workspace-surface);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.66);
}

.share-header-left {
  display: flex;
  align-items: center;
  min-width: 0;
  gap: 12px;
}

.share-avatar-icon {
  flex-shrink: 0;
  filter: drop-shadow(0 8px 18px rgba(var(--workspace-accent-rgb), 0.18));
}

.share-meta {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.share-title {
  margin: 0;
  color: var(--workspace-text);
  font-size: 18px;
  font-weight: 760;
  line-height: 1.25;
  overflow-wrap: anywhere;
}

.share-meta-items {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.meta-item {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: var(--workspace-text-muted);
  font-variant-numeric: tabular-nums;
}

/* ---- 文件卡片网格 ---- */
.file-surface {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  padding: 10px;
  border: 1px solid var(--workspace-border);
  border-radius: var(--workspace-radius-xl);
  background: var(--workspace-surface);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.66);
}

.file-grid {
  height: 100%;
  overflow-y: auto;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(220px, 100%), 1fr));
  align-content: start;
  gap: 10px;
  padding: 6px 2px 8px;
  scrollbar-gutter: stable;
}

.file-grid.single-file {
  width: min(340px, 100%);
  margin: 0 auto;
}

.file-card {
  min-width: 0;
  display: flex;
  flex-direction: column;
  padding: 14px;
  border: 1px solid var(--workspace-border-soft);
  border-radius: var(--workspace-radius-lg);
  background: color-mix(in srgb, var(--workspace-surface-soft) 84%, transparent);
  transition:
    transform 0.18s ease,
    border-color 0.18s ease,
    box-shadow 0.18s ease,
    background-color 0.18s ease;
}

.file-card:hover {
  transform: translateY(-2px);
  border-color: rgba(var(--workspace-accent-rgb), 0.32);
  box-shadow:
    inset 0 1px 0 rgba(var(--workspace-accent-rgb), 0.12),
    0 10px 24px rgba(var(--workspace-accent-rgb), 0.1);
}

.dark .share-header,
.dark .file-surface {
  box-shadow: inset 0 1px 0 rgba(248, 250, 252, 0.08);
}

.file-icon-area {
  width: 52px;
  height: 52px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 12px;
  flex-shrink: 0;
  background: rgba(var(--workspace-accent-rgb), 0.1);
}

.file-info {
  flex: 1;
  min-width: 0;
  margin-bottom: 12px;
}

.file-name {
  font-size: 13px;
  font-weight: 700;
  color: var(--workspace-text);
  margin: 0 0 4px;
  word-break: break-word;
  overflow-wrap: anywhere;
  line-height: 1.4;
}

.file-size {
  font-size: 12px;
  color: var(--workspace-text-muted);
  margin: 0;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  font-weight: 500;
  font-variant-numeric: tabular-nums;
}

/* ---- 操作按钮 ---- */
.file-actions {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}

.action-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 40px;
  padding: 0 10px;
  border: 1px solid transparent;
  border-radius: var(--workspace-radius-md);
  cursor: pointer;
  font-size: 13px;
  font-weight: 600;
  transition:
    background-color 0.16s ease,
    border-color 0.16s ease,
    color 0.16s ease,
    transform 0.16s ease;
}

.action-btn:hover {
  transform: translateY(-1px);
}

.action-btn:active:not(:disabled) {
  transform: scale(0.96);
}

.action-btn:disabled {
  cursor: wait;
  opacity: 0.62;
}

.action-btn:disabled:hover {
  transform: none;
}

.action-download {
  background: rgba(var(--workspace-accent-rgb), 0.1);
  color: var(--workspace-accent);
}

.action-download:hover {
  background: var(--workspace-accent);
  color: var(--workspace-on-accent);
}

.action-link {
  background: var(--workspace-field);
  border-color: var(--workspace-border);
  color: var(--workspace-text-muted);
}

.action-link:hover {
  border-color: rgba(var(--workspace-accent-rgb), 0.34);
  color: var(--workspace-accent);
}

/* ---- 响应式 ---- */
@media (max-width: 640px) {
  .share-title {
    font-size: 17px;
  }

  .share-header-left {
    flex-direction: column;
    align-items: flex-start;
  }

  .share-meta-items {
    gap: 8px;
  }

  .file-surface,
  .share-header {
    border-radius: var(--workspace-radius-lg);
  }
}
</style>
