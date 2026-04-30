<template>
  <n-card class="file-explorer-card" :bordered="false" content-style="display: flex; flex-direction: column; height: 100%;">
      <template #header>
        <n-space justify="space-between" align="center">
          <n-breadcrumb>
            <n-breadcrumb-item @click="navigateTo('/')">
              <span class="root-breadcrumb">根目录</span>
            </n-breadcrumb-item>
            <n-breadcrumb-item v-for="seg in pathSegments" :key="seg.path" @click="navigateTo(seg.path)">
              {{ seg.name }}
            </n-breadcrumb-item>
          </n-breadcrumb>
          <n-tooltip trigger="hover">
            <template #trigger>
              <n-button size="small" @click="refresh">
                <template #icon><n-icon><refresh-outline /></n-icon></template>
              </n-button>
            </template>
            刷新
          </n-tooltip>
        </n-space>
      </template>

      <n-space :size="8" class="toolbar-row">
        <n-tooltip v-if="hasPermUpload" trigger="hover">
          <template #trigger>
            <n-button type="primary" @click="showUploadModal = true">
              <template #icon><n-icon><cloud-upload-outline /></n-icon></template>
            </n-button>
          </template>
          上传文件
        </n-tooltip>
        <n-tooltip v-if="hasPermUpload" trigger="hover">
          <template #trigger>
            <n-button @click="showMkdirModal = true">
              <template #icon><n-icon><AddCircleOutline /></n-icon></template>
            </n-button>
          </template>
          新建文件夹
        </n-tooltip>
        <n-tooltip v-if="currentPath !== '/'" trigger="hover">
          <template #trigger>
            <n-button @click="goToParent">
              <template #icon><n-icon><arrow-back-outline /></n-icon></template>
            </n-button>
          </template>
          返回上级
        </n-tooltip>
      </n-space>

      <div class="file-table-wrapper">
        <n-data-table
          class="file-data-table"
          size="small"
          flex-height
          :columns="columns"
          :data="entries"
          :bordered="false"
          striped
          :loading="loading"
          :row-key="(row: any) => row.path || row.name"
          style="height: 100%;"
        />
      </div>

      <n-modal v-model:show="showUploadModal" title="上传文件" preset="dialog">
        <n-form label-placement="left" label-width="80">
          <n-form-item label="目标目录">
            <n-input :value="currentPath" :disabled="true" />
          </n-form-item>
          <n-form-item label="选择文件">
            <input type="file" @change="onFileSelect" />
          </n-form-item>
        </n-form>
        <template #action>
          <n-button @click="showUploadModal = false">取消</n-button>
          <n-button type="primary" :loading="uploading" @click="handleUpload">上传</n-button>
        </template>
      </n-modal>

      <n-modal v-model:show="showMkdirModal" title="新建文件夹" preset="dialog">
        <n-input v-model:value="mkdirName" placeholder="文件夹名称" />
        <template #action>
          <n-button @click="showMkdirModal = false">取消</n-button>
          <n-button type="primary" :loading="mkdirLoading" @click="handleMkdir">创建</n-button>
        </template>
      </n-modal>

      <n-modal v-model:show="showMoveModal" title="移动/重命名" preset="dialog">
        <n-form label-placement="left" label-width="80">
          <n-form-item label="原路径">
            <n-input :value="moveSource" :disabled="true" />
          </n-form-item>
          <n-form-item label="新路径">
            <n-input v-model:value="moveDest" placeholder="输入新路径（相对路径）" />
          </n-form-item>
        </n-form>
        <template #action>
          <n-button @click="showMoveModal = false">取消</n-button>
          <n-button type="primary" :loading="moveLoading" @click="handleMove">确认</n-button>
        </template>
      </n-modal>

      <n-modal v-model:show="showOwnerModal" title="变更所有者" preset="dialog">
        <n-select v-model:value="newOwnerId" :options="allUsers" placeholder="选择新所有者" />
        <template #action>
          <n-button @click="showOwnerModal = false">取消</n-button>
          <n-button type="primary" @click="handleOwnerChange">确认</n-button>
        </template>
      </n-modal>
    </n-card>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, h, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  NCard, NSpace, NButton, NDataTable, NModal, NForm, NFormItem, NInput,
  NBreadcrumb, NBreadcrumbItem, NSelect, NIcon, NTooltip, useMessage
} from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import api from '@/api'
import { useUserStore } from '@/stores/user'
import { formatSize } from '@/utils/format'
import {
  FolderOpen,
  Image,
  DocumentText,
  Document,
  CodeSlash,
  Archive,
  Videocam,
  MusicalNotes,
  EnterOutline,
  CloudDownloadOutline,
  ShareSocialOutline,
  TrashOutline,
  CloudUploadOutline,
  CreateOutline,
  AddCircleOutline,
  RefreshOutline,
  ArrowBackOutline,
} from '@vicons/ionicons5'

const route = useRoute()
const router = useRouter()
const message = useMessage()
const userStore = useUserStore()

const entries = ref<any[]>([])
const loading = ref(false)
const currentPath = ref('/')
const showUploadModal = ref(false)
const showMkdirModal = ref(false)
const showMoveModal = ref(false)
const showOwnerModal = ref(false)
const uploading = ref(false)
const mkdirLoading = ref(false)
const moveLoading = ref(false)
const selectedFile = ref<File | null>(null)
const mkdirName = ref('')
const moveSource = ref('')
const moveDest = ref('')
const ownerChangePath = ref('')
const newOwnerId = ref<number | null>(null)
const allUsers = ref<{ label: string; value: number }[]>([])

const hasPermUpload = computed(() => userStore.hasPermission(4))
const hasPermShare = computed(() => userStore.hasPermission(8))
const isAdmin = computed(() => userStore.user?.is_admin)

const pathSegments = computed(() => {
  if (currentPath.value === '/') return []
  const parts = currentPath.value.split('/').filter(Boolean)
  return parts.map((name, i) => ({
    name,
    path: '/' + parts.slice(0, i + 1).join('/'),
  }))
})

// 上级目录路径
const parentPath = computed(() => {
  if (currentPath.value === '/') return '/'
  const parts = currentPath.value.split('/').filter(Boolean)
  parts.pop()
  return '/' + parts.join('/') || '/'
})

// 根据文件名和类型返回对应的图标和颜色
function getFileIcon(name: string, isDir: boolean): { icon: any; color: string } {
  if (isDir) return { icon: FolderOpen, color: '#e6a23c' }

  const ext = name.toLowerCase().split('.').pop() || ''

  // 图片
  if (['jpg', 'jpeg', 'png', 'gif', 'svg', 'webp', 'bmp', 'ico', 'tiff', 'tif'].includes(ext))
    return { icon: Image, color: '#67c23a' }
  // 视频
  if (['mp4', 'avi', 'mov', 'mkv', 'webm', 'flv', 'wmv', 'm4v'].includes(ext))
    return { icon: Videocam, color: '#e6a23c' }
  // 音频
  if (['mp3', 'wav', 'flac', 'aac', 'ogg', 'wma', 'm4a', 'ape'].includes(ext))
    return { icon: MusicalNotes, color: '#f56c6c' }
  // 压缩包
  if (['zip', 'tar', 'gz', 'rar', '7z', 'bz2', 'xz', 'tgz', 'zst'].includes(ext))
    return { icon: Archive, color: '#909399' }
  // 代码
  if (['js', 'ts', 'jsx', 'tsx', 'vue', 'py', 'go', 'java', 'c', 'cpp', 'h', 'rs', 'rb',
       'php', 'swift', 'kt', 'html', 'css', 'scss', 'less', 'json', 'xml', 'yaml',
       'yml', 'toml', 'sql', 'sh', 'bash', 'cmd', 'ps1', 'bat'].includes(ext))
    return { icon: CodeSlash, color: '#409eff' }
  // 文档
  if (['pdf', 'doc', 'docx', 'xls', 'xlsx', 'ppt', 'pptx', 'txt', 'md', 'log', 'csv', 'rtf'].includes(ext))
    return { icon: DocumentText, color: '#409eff' }
  // 默认
  return { icon: Document, color: '#909399' }
}

// 图标按钮 + Tooltip（render 函数中用）
function iconBtn(iconComp: any, tooltip: string, onClick: () => void, color?: string) {
  return h(NTooltip, { trigger: 'hover', placement: 'top' }, {
    default: () => tooltip,
    trigger: () =>
      h(NButton, { size: 'small', quaternary: true, onClick }, {
        icon: () => h(NIcon, { size: 18, color: color || undefined }, () => h(iconComp)),
      }),
  })
}

const columns: DataTableColumns = [
  {
    title: '名称',
    key: 'name',
    className: 'col-name',
    sorter: (a: any, b: any) => a.name.localeCompare(b.name),
    render(row: any) {
      const { icon, color } = getFileIcon(row.name, row.is_directory)
      const iconEl = h(NIcon, { size: 18, color, style: { marginRight: '6px', verticalAlign: 'middle', flexShrink: 0 } }, () => h(icon))
      if (row.is_directory) {
        return h('div', { class: 'name-cell' },
          h(NButton, { text: true, type: 'info', onClick: () => navigateTo(row.path) }, () => [iconEl, h('span', { class: 'name-text' }, row.name)]),
        )
      }
      return h('div', { class: 'name-cell' }, [
        iconEl,
        h('span', { class: 'name-text' }, row.name),
      ])
    },
  },
  {
    title: '大小',
    key: 'size',
    className: 'col-size',
    width: 110,
    sorter: (a: any, b: any) => a.size - b.size,
    render(row: any) {
      return row.is_directory ? '—' : formatSize(row.size)
    },
  },
  {
    title: '修改时间',
    key: 'mod_time',
    className: 'col-time',
    width: 170,
    sorter: (a: any, b: any) => new Date(a.mod_time).getTime() - new Date(b.mod_time).getTime(),
    render(row: any) {
      return new Date(row.mod_time).toLocaleString()
    },
  },
  {
    title: '所有者',
    key: 'owner_name',
    className: 'col-owner',
    width: 140,
    render(row: any) {
      return isAdmin.value
        ? h(NButton, { text: true, type: 'primary', onClick: () => openOwnerModal(row) }, () => row.owner_name)
        : row.owner_name as string
    },
  },
  {
    title: '操作',
    key: 'actions',
    className: 'col-actions',
    width: 175,
    render(row: any) {
      const btns: any[] = []

      if (row.is_directory) {
        btns.push(iconBtn(EnterOutline, '进入目录', () => navigateTo(row.path), '#3B82F6'))
      } else {
        if (row.can_download) btns.push(iconBtn(CloudDownloadOutline, '下载文件', () => downloadFile(row), '#3B82F6'))
        if (hasPermShare.value) btns.push(iconBtn(ShareSocialOutline, '分享文件', () => shareFile(row), '#3B82F6'))
      }
      if (row.can_delete) btns.push(iconBtn(TrashOutline, '删除', () => deleteEntry(row), '#d03050'))
      if (row.can_change) btns.push(iconBtn(CreateOutline, '移动/重命名', () => openMoveModal(row)))

      return h(NSpace, { size: 2 }, () => btns)
    },
  },
]

onMounted(() => {
  const p = (route.query.path as string) || '/'
  currentPath.value = p
  fetchFiles()
  if (isAdmin.value) fetchAllUsers()
})

watch(() => route.query.path, (newPath) => {
  const p = (newPath as string) || '/'
  if (p !== currentPath.value) {
    currentPath.value = p
    fetchFiles()
  }
})

async function fetchFiles() {
  loading.value = true
  try {
    const res = await api.get('/api/files', { params: { path: currentPath.value } })
    entries.value = res.data.entries || []
  } catch (err: any) {
    console.error('fetchFiles failed:', err)
    message.error(err.response?.data?.error || '获取文件列表失败')
    entries.value = []
  } finally {
    loading.value = false
  }
}

function navigateTo(path: string) {
  router.push({ query: { path } })
}

function goToParent() {
  navigateTo(parentPath.value)
}

function refresh() {
  fetchFiles()
}

function onFileSelect(e: Event) {
  const target = e.target as HTMLInputElement
  selectedFile.value = target.files?.[0] || null
}

async function handleUpload() {
  if (!selectedFile.value) { message.warning('请选择文件'); return }
  uploading.value = true
  try {
    const formData = new FormData()
    formData.append('file', selectedFile.value)
    formData.append('path', currentPath.value)
    await api.post('/api/files/upload', formData, { headers: { 'Content-Type': 'multipart/form-data' } })
    message.success('上传成功')
    showUploadModal.value = false
    selectedFile.value = null
    await fetchFiles()
  } catch (err: any) {
    message.error(err.response?.data?.error || '上传失败')
  } finally {
    uploading.value = false
  }
}

async function handleMkdir() {
  if (!mkdirName.value) { message.warning('请输入文件夹名称'); return }
  mkdirLoading.value = true
  try {
    await api.post('/api/files/mkdir', { path: currentPath.value, name: mkdirName.value })
    message.success('文件夹创建成功')
    showMkdirModal.value = false
    mkdirName.value = ''
    await fetchFiles()
  } catch (err: any) {
    message.error(err.response?.data?.error || '创建失败')
  } finally {
    mkdirLoading.value = false
  }
}

async function deleteEntry(row: any) {
  if (!confirm(`确认删除 "${row.name}"？`)) return
  try {
    await api.delete('/api/files', { data: { path: row.path } })
    message.success('删除成功')
    await fetchFiles()
  } catch (err: any) {
    message.error(err.response?.data?.error || '删除失败')
  }
}

function downloadFile(row: any) {
  window.open(`/api/download?path=${encodeURIComponent(row.path)}`, '_blank')
}

function shareFile(row: any) {
  router.push({ path: '/shares', query: { shareFile: row.path } })
}

function openMoveModal(row: any) {
  moveSource.value = row.path
  moveDest.value = row.path
  showMoveModal.value = true
}

async function handleMove() {
  moveLoading.value = true
  try {
    await api.put('/api/files/move', { source: moveSource.value, destination: moveDest.value })
    message.success('移动/重命名成功')
    showMoveModal.value = false
    await fetchFiles()
  } catch (err: any) {
    message.error(err.response?.data?.error || '移动/重命名失败')
  } finally {
    moveLoading.value = false
  }
}

function openOwnerModal(row: any) {
  ownerChangePath.value = row.path
  newOwnerId.value = row.owner_id
  showOwnerModal.value = true
}

async function handleOwnerChange() {
  try {
    await api.put('/api/files/owner', { path: ownerChangePath.value, owner_id: newOwnerId.value })
    message.success('所有者变更成功')
    showOwnerModal.value = false
    await fetchFiles()
  } catch (err: any) {
    message.error(err.response?.data?.error || '变更失败')
  }
}

async function fetchAllUsers() {
  try {
    const res = await api.get('/api/users')
    allUsers.value = res.data.map((u: any) => ({
      label: u.display_name || u.username,
      value: u.id,
    }))
  } catch { /* ignore */ }
}
</script>

<style scoped>
.file-explorer-card {
  height: calc(100vh - 100px);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.file-explorer-card :deep(.n-card__content) {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

/* 顶部路径面包屑区域：灰色圆角边框 */
.file-explorer-card :deep(.n-card-header) {
  padding-top: 24px;
  padding-bottom: 0;
}
.file-explorer-card :deep(.n-card-header__main) {
  padding: 10px 14px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fafafa;
}

/* 根目录默认粗体 */
.root-breadcrumb {
  font-weight: 600;
}

/* 工具栏与面包屑目视觉间距 ≈ .n-card-header__main padding-top (10px) */
.toolbar-row {
  margin-top: 10px;
  margin-bottom: 10px;
}

.file-table-wrapper {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

/* 紧凑化表格行高 */
.file-data-table :deep(.n-data-table-td),
.file-data-table :deep(.n-data-table-th) {
  padding-top: 6px !important;
  padding-bottom: 6px !important;
  font-size: 13px;
}

.file-data-table :deep(.n-data-table-th) {
  font-weight: 600;
}

/* 名称列：自动吸纳剩余宽度，长文件名按字符换行，不被截断 */
.file-data-table :deep(.col-name .n-data-table-td__ellipsis),
.file-data-table :deep(.col-name) {
  white-space: normal !important;
}
.file-data-table :deep(.col-name .name-cell) {
  display: flex;
  align-items: flex-start;
  min-width: 0;
  word-break: break-all;
  overflow-wrap: anywhere;
}
.file-data-table :deep(.col-name .name-text) {
  white-space: normal;
  word-break: break-all;
  overflow-wrap: anywhere;
  line-height: 1.5;
}
.file-data-table :deep(.col-name .n-button__content) {
  white-space: normal;
  word-break: break-all;
  overflow-wrap: anywhere;
  text-align: left;
  line-height: 1.5;
}

/* 非名称列：紧凑不换行 */
.file-data-table :deep(.col-time),
.file-data-table :deep(.col-actions) {
  white-space: nowrap;
}
</style>
