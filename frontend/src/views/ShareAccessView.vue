<template>
  <div class="share-access-content" :class="{ dark: themeStore.isDark }">
    <!-- 主内容 -->
    <main class="share-main">
      <!-- 错误状态 -->
      <div v-if="error" class="error-container">
        <n-result status="error" :title="error" description="分享链接无效或已过期" />
      </div>

      <!-- 加载中 -->
      <n-spin v-else :show="loading">
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
          <div class="file-grid" :class="{ 'single-file': shareInfo.files.length === 1 }">
            <div v-for="file in shareInfo.files" :key="file.file_name" class="file-card">
              <div class="file-icon-area" :style="{ color: getFileColor(file.file_name) }">
                <n-icon :size="32"><component :is="getFileIconComp(file.file_name)" /></n-icon>
              </div>
              <div class="file-info">
                <h3 class="file-name" :title="file.file_name">{{ file.file_name }}</h3>
                <p class="file-size">{{ formatSize(file.file_size) }}</p>
              </div>
              <div class="file-actions">
                <button class="action-btn action-download" @click="downloadFile(file)">
                  <n-icon size="16"><DownloadOutline /></n-icon>
                  <span>下载</span>
                </button>
                <button class="action-btn action-link" @click="copyDownloadLink(file)">
                  <n-icon size="16"><LinkOutline /></n-icon>
                  <span>链接</span>
                </button>
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
import { NIcon, NResult, NSpin, useMessage } from 'naive-ui'
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
    file_name: string
    file_size: number
    file_path: string
    download_count: number
  }>
}
const shareInfo = ref<ShareInfo | null>(null)

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

// 操作
function downloadFile(file: { file_name: string }) {
  const token = route.params.token as string
  const url = `/share/${token}/${encodeURIComponent(file.file_name)}`
  window.location.href = url
}

async function copyDownloadLink(file: { file_name: string }) {
  const token = route.params.token as string
  const url = `${window.location.origin}/share/${token}/${encodeURIComponent(file.file_name)}`
  const ok = await copyToClipboard(url)
  if (ok) {
    message.success('下载链接已复制到剪贴板')
  } else {
    message.error('复制失败！')
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
  height: 100%;
  overflow-y: auto;
}

.dark.share-access-content {
  /* dark mode handled by parent MainLayout, local class for scoped selector matching */
}

/* ---- 主内容区 ---- */
.share-main {
  padding: 32px 0 40px;
  max-width: 1280px;
  width: 100%;
  margin: 0 auto;
  box-sizing: border-box;
}

.error-container {
  display: flex;
  justify-content: center;
  padding-top: 60px;
}

/* ---- 分享信息头部 ---- */
.share-header {
  margin-bottom: 40px;
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 24px;
}

.share-header-left {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}

.share-avatar-icon {
  flex-shrink: 0;
  filter: drop-shadow(0 2px 8px rgba(var(--theme-color-rgb, 59, 130, 246), 0.3));
}

.share-meta {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.share-title {
  font-size: 24px;
  font-weight: 700;
  color: #0f172a;
  margin: 0;
  line-height: 1.3;
  transition: color 0.3s ease;
}

.dark .share-title {
  color: #f1f5f9;
}

.share-meta-items {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 16px;
}

.meta-item {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: #64748b;
  transition: color 0.3s ease;
}

.dark .meta-item {
  color: #94a3b8;
}

/* ---- 文件卡片网格 ---- */
.file-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(260px, 100%), 1fr));
  gap: 20px;
}

.file-grid.single-file {
  max-width: 320px;
  margin: 0 auto;
}

.file-card {
  background: #fff;
  border: 1px solid rgba(var(--theme-color-rgb, 59, 130, 246), 0.08);
  border-radius: 20px;
  padding: 24px;
  display: flex;
  flex-direction: column;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.file-card:hover {
  transform: translateY(-4px);
  border-color: rgba(var(--theme-color-rgb, 59, 130, 246), 0.3);
  box-shadow: 0 12px 28px rgba(var(--theme-color-rgb, 59, 130, 246), 0.1);
}

.dark .file-card {
  background: #1e293b;
  border-color: rgba(var(--theme-color-rgb, 59, 130, 246), 0.12);
}

.dark .file-card:hover {
  border-color: rgba(var(--theme-color-rgb, 59, 130, 246), 0.4);
  box-shadow: 0 12px 28px rgba(var(--theme-color-rgb, 59, 130, 246), 0.18);
}

.file-icon-area {
  width: 64px;
  height: 64px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 12px;
  flex-shrink: 0;
  background: rgba(var(--theme-color-rgb, 59, 130, 246), 0.1);
}

.dark .file-icon-area {
  background: rgba(var(--theme-color-rgb, 59, 130, 246), 0.12);
}

.file-info {
  flex: 1;
  margin-bottom: 16px;
}

.file-name {
  font-size: 13px;
  font-weight: 700;
  color: #1e293b;
  margin: 0 0 4px;
  word-break: break-word;
  overflow-wrap: anywhere;
  line-height: 1.4;
  transition: color 0.3s ease;
}

.dark .file-name {
  color: #f1f5f9;
}

.file-size {
  font-size: 12px;
  color: #94a3b8;
  margin: 0;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  font-weight: 500;
}

/* ---- 操作按钮 ---- */
.file-actions {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}

.action-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 10px 0;
  border: none;
  border-radius: 10px;
  cursor: pointer;
  font-size: 13px;
  font-weight: 600;
  transition: all 0.2s ease;
}

.action-download {
  background: rgba(var(--theme-color-rgb, 59, 130, 246), 0.08);
  color: var(--theme-color, #3b82f6);
}

.action-download:hover {
  background: var(--theme-color, #3b82f6);
  color: #fff;
}

.dark .action-download {
  background: rgba(var(--theme-color-rgb, 59, 130, 246), 0.15);
  color: #60a5fa;
}

.dark .action-download:hover {
  background: var(--theme-color, #3b82f6);
  color: #fff;
}

.action-link {
  background: transparent;
  border: 1px solid #e2e8f0;
  color: #64748b;
}

.action-link:hover {
  border-color: var(--theme-color, #3b82f6);
  color: var(--theme-color, #3b82f6);
}

.dark .action-link {
  border-color: #475569;
  color: #94a3b8;
}

.dark .action-link:hover {
  border-color: #60a5fa;
  color: #60a5fa;
}

/* ---- 响应式 ---- */
@media (max-width: 640px) {
  .share-title {
    font-size: 20px;
  }

  .share-header-left {
    flex-direction: column;
    align-items: flex-start;
  }

  .share-meta-items {
    gap: 8px;
  }
}
</style>
