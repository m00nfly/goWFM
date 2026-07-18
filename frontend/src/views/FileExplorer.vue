<template>
  <div class="file-explorer" :class="{ dark: themeStore.isDark }">
    <section class="explorer-overview" aria-label="文件库概览">
      <div class="overview-main">
        <div class="overview-icon">
          <n-icon size="24"><FolderOpenOutline /></n-icon>
        </div>
        <div class="overview-copy">
          <div class="breadcrumb" aria-label="当前位置">
            <button class="breadcrumb-link" type="button" @click="navigateTo('/')">根目录</button>
            <template v-for="(seg, idx) in pathSegments" :key="seg.path">
              <span class="breadcrumb-sep">/</span>
              <span
                v-if="idx === pathSegments.length - 1"
                class="breadcrumb-current"
              >{{ seg.name }}</span>
              <button
                v-else
                class="breadcrumb-link"
                type="button"
                @click="navigateTo(seg.path)"
              >{{ seg.name }}</button>
            </template>
          </div>
          <h1>{{ currentDirectoryName }}</h1>
        </div>
      </div>

      <div class="overview-side">
        <div class="stat-strip" aria-label="当前目录统计">
          <div class="stat-item">
            <span class="stat-value">{{ folderCount }}</span>
            <span class="stat-label">文件夹</span>
          </div>
          <div class="stat-item">
            <span class="stat-value">{{ fileCount }}</span>
            <span class="stat-label">文件</span>
          </div>
          <div class="stat-item stat-item-wide">
            <span class="stat-value">{{ visibleSizeLabel }}</span>
            <span class="stat-label">当前文件大小</span>
          </div>
        </div>
      </div>
    </section>

    <section
      class="content-surface"
      :class="{ 'has-floating-batch': selectedItems.length > 0 }"
      aria-label="文件列表"
    >
      <div class="content-header">
        <div class="header-tools-actions">
          <n-button secondary @click="refresh">
            <template #icon><n-icon><RefreshOutline /></n-icon></template>
            刷新
          </n-button>
          <n-button v-if="canUploadToCurrentDirectory" secondary @click="showMkdirModal = true">
            <template #icon><n-icon><AddCircleOutline /></n-icon></template>
            新建目录
          </n-button>
          <n-button v-if="canUploadToCurrentDirectory" type="primary" @click="showUploadModal = true">
            <template #icon><n-icon><CloudUploadOutline /></n-icon></template>
            上传文件
          </n-button>
          <n-button v-if="currentPath !== '/'" secondary @click="goToParent">
            <template #icon><n-icon><ArrowBackOutline /></n-icon></template>
            返回上级
          </n-button>
        </div>

        <n-input
          v-model:value="searchKeyword"
          placeholder="搜索文件名"
          clearable
          class="search-input"
        >
          <template #prefix>
            <n-icon><SearchOutline /></n-icon>
          </template>
        </n-input>

        <div class="view-switch" role="group" aria-label="视图切换">
          <button
            class="view-switch-btn"
            :class="{ active: viewMode === 'list' }"
            type="button"
            aria-label="列表视图"
            title="列表视图"
            @click="viewMode = 'list'"
          >
            <n-icon size="18"><ListOutline /></n-icon>
          </button>
          <button
            class="view-switch-btn"
            :class="{ active: viewMode === 'grid' }"
            type="button"
            aria-label="网格视图"
            title="网格视图"
            @click="viewMode = 'grid'"
          >
            <n-icon size="18"><GridOutline /></n-icon>
          </button>
        </div>
      </div>

      <n-result
        v-if="permissionDenied"
        status="403"
        title="无访问权限"
        :description="permissionDeniedMsg"
        class="permission-denied-result"
      />

      <div v-else-if="viewMode === 'list'" class="file-list">
        <n-data-table
          class="file-data-table"
          size="small"
          flex-height
          virtual-scroll
          :columns="columns"
          :data="filteredFiles"
          :bordered="false"
          :loading="loading"
          :row-key="rowKey"
          :row-class-name="rowClassName"
          :checked-row-keys="checkedKeys"
          @update:checked-row-keys="onCheckedKeysChange"
        />
      </div>

      <div v-else class="file-grid-container">
        <div v-if="filteredFiles.length === 0" class="grid-empty">
          <n-empty :description="searchKeyword ? '没有匹配的文件' : '暂无文件'" />
        </div>
        <div v-else class="file-grid">
          <article
            v-for="file in filteredFiles"
            :key="file.path || file.name"
            class="grid-card"
            :class="{ 'grid-card-selected': checkedKeySet.has(file.path || file.name) }"
            @click="onGridCardClick(file)"
          >
            <div class="grid-card-top">
              <n-checkbox
                class="grid-card-checkbox"
                :checked="checkedKeySet.has(file.path || file.name)"
                @update:checked="toggleGridSelection(file.path || file.name, $event)"
                @click.stop
              />
              <span class="grid-card-kind">{{ file.is_directory ? '文件夹' : formatSize(file.size) }}</span>
            </div>

            <div class="grid-card-icon" :class="{ 'icon-folder': file.is_directory, 'icon-file': !file.is_directory }">
              <n-icon :size="34" :color="getFileIcon(file.name, file.is_directory).color">
                <component :is="getFileIcon(file.name, file.is_directory).icon" />
              </n-icon>
            </div>

            <div class="grid-card-body">
              <h3 class="grid-card-name" :title="file.name">{{ file.name }}</h3>
              <p class="grid-card-info">
                {{ file.is_directory ? '点击进入目录' : formatTime(file.mod_time) }}
              </p>
            </div>

            <div class="grid-card-actions">
              <n-tooltip v-if="file.is_directory" trigger="hover" placement="top">
                <template #trigger>
                  <button class="card-action-btn primary" type="button" @click.stop="navigateTo(file.path)">
                    <n-icon size="16"><EnterOutline /></n-icon>
                  </button>
                </template>
                进入目录
              </n-tooltip>
              <template v-else>
                <n-tooltip v-if="file.can_download" trigger="hover" placement="top">
                  <template #trigger>
                    <button class="card-action-btn primary" type="button" @click.stop="downloadFile(file)">
                      <n-icon size="16"><CloudDownloadOutline /></n-icon>
                    </button>
                  </template>
                  下载
                </n-tooltip>
                <n-tooltip v-if="hasPermShare" trigger="hover" placement="top">
                  <template #trigger>
                    <button class="card-action-btn primary" type="button" @click.stop="shareFile(file)">
                      <n-icon size="16"><ShareSocialOutline /></n-icon>
                    </button>
                  </template>
                  分享
                </n-tooltip>
              </template>
              <n-tooltip v-if="file.can_change" trigger="hover" placement="top">
                <template #trigger>
                  <button class="card-action-btn" type="button" @click.stop="openMoveModal(file)">
                    <n-icon size="16"><CreateOutline /></n-icon>
                  </button>
                </template>
                移动/重命名
              </n-tooltip>
              <n-tooltip v-if="file.can_delete" trigger="hover" placement="top">
                <template #trigger>
                  <button class="card-action-btn danger" type="button" @click.stop="deleteEntry(file)">
                    <n-icon size="16"><TrashOutline /></n-icon>
                  </button>
                </template>
                删除
              </n-tooltip>
            </div>
          </article>
        </div>
      </div>

      <section v-if="selectedItems.length > 0" class="batch-bar" aria-live="polite">
        <div class="batch-summary">
          <n-dropdown trigger="hover" :options="selectedDropdownOptions" :render-label="renderDropdownLabel" placement="top-start" :max-height="300" scrollable>
            <button class="batch-info batch-info-clickable" type="button">
              已选择 {{ selectedItems.length }} 项
            </button>
          </n-dropdown>
          <span>{{ selectedFileCount }} 个文件</span>
          <span>{{ selectedFolderCount }} 个文件夹</span>
        </div>
        <div class="batch-actions">
          <n-button size="small" secondary @click="batchDownload" v-if="userStore.hasPermission(2)">批量下载</n-button>
          <n-button size="small" secondary @click="batchShare" v-if="hasPermShare">批量分享</n-button>
          <n-button size="small" type="error" secondary @click="batchDelete" v-if="userStore.user?.is_admin || userStore.hasPermission(4)">批量删除</n-button>
          <n-button size="small" quaternary @click="selectedItems = []">取消选择</n-button>
        </div>
      </section>
    </section>

    <!-- 上传文件模态框 -->
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

    <!-- 新建文件夹模态框 -->
    <n-modal v-model:show="showMkdirModal" title="新建文件夹" preset="dialog">
      <n-input v-model:value="mkdirName" placeholder="文件夹名称" />
      <template #action>
        <n-button @click="showMkdirModal = false">取消</n-button>
        <n-button type="primary" :loading="mkdirLoading" @click="handleMkdir">创建</n-button>
      </template>
    </n-modal>

    <!-- 移动/重命名模态框 -->
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

    <!-- 变更所有者模态框 -->
    <n-modal v-model:show="showOwnerModal" title="变更所有者" preset="dialog">
      <n-select v-model:value="newOwnerId" :options="allUsers" placeholder="选择新所有者" />
      <template #action>
        <n-button @click="showOwnerModal = false">取消</n-button>
        <n-button type="primary" @click="handleOwnerChange">确认</n-button>
      </template>
    </n-modal>

    <!-- 创建分享模态框 -->
    <n-modal v-model:show="showShareModal" preset="dialog" title="创建文件分享" positive-text="创建" negative-text="取消" :positive-button-props="{ loading: shareLoading }" @positive-click="handleCreateShare" @negative-click="cancelShareCreation" :mask-closable="false">
      <n-form label-placement="left" label-width="80">
        <n-form-item label="文件">
          <div class="share-file-list">
            <n-tag
              v-for="p in shareFilePaths"
              :key="p"
              class="share-file-tag"
              size="small"
              :closable="!shareLoading"
              :title="p.split('/').pop()"
              @close="removeShareFilePath(p)"
            >
              {{ p.split('/').pop() }}
            </n-tag>
            <p class="share-file-count">共 {{ shareFilePaths.length }} 个文件</p>
          </div>
        </n-form-item>
        <n-form-item label="分享名称">
          <n-input v-model:value="shareName" placeholder="自动生成（可自定义）" clearable :maxlength="100" />
        </n-form-item>
        <n-form-item label="有效期(天)">
          <n-input-number v-model:value="shareExpireDays" :min="0" :max="365" placeholder="0 表示永久有效" style="width: 100%" />
        </n-form-item>
      </n-form>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, h } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  NButton, NDataTable, NModal, NForm, NFormItem, NInput, NInputNumber,
  NSelect, NIcon, NTooltip, NResult, NEmpty, NSpace, NCheckbox, NTag, NDropdown, useMessage, useDialog
} from 'naive-ui'
import type { DataTableColumns, DataTableRowKey } from 'naive-ui'
import api from '@/api'
import { useUserStore } from '@/stores/user'
import { useThemeStore } from '@/stores/theme'
import { useViewport } from '@/composables/useViewport'
import { formatSize } from '@/utils/format'
import {
  FolderOpen,
  FolderOpenOutline,
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
  SearchOutline,
  ListOutline,
  GridOutline,
} from '@vicons/ionicons5'

const route = useRoute()
const router = useRouter()
const message = useMessage()
const dialog = useDialog()
const userStore = useUserStore()
const themeStore = useThemeStore()

// === 跨目录选择数据模型 ===
interface SelectedItem {
  key: string
  name: string
  is_directory: boolean
}

// === 状态 ===
const entries = ref<any[]>([])
const loading = ref(false)
const highlightName = ref<string | null>(null)
let highlightTimer: ReturnType<typeof setTimeout> | null = null
const permissionDenied = ref(false)
const permissionDeniedMsg = ref('')
const currentPath = ref('/')
const showUploadModal = ref(false)
const showMkdirModal = ref(false)
const showMoveModal = ref(false)
const showOwnerModal = ref(false)
const showShareModal = ref(false)
const shareFilePaths = ref<string[]>([])
const shareName = ref('')
const shareExpireDays = ref(7)
const shareLoading = ref(false)
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
const currentDirectoryAllowsUpload = ref(false)

// === 新增状态 ===
const searchKeyword = ref('')
// 跨目录选择：selectedItems 为单一数据源，checkedKeys/checkedKeySet 由此衍生
const selectedItems = ref<SelectedItem[]>([])
const checkedKeys = computed<DataTableRowKey[]>(() => selectedItems.value.map(i => i.key))
const checkedKeySet = computed(() => new Set(checkedKeys.value))
const viewMode = ref<'list' | 'grid'>('list')

// === 视口状态（共享自 MainLayout 的 resize 监听） ===
const { isMobile } = useViewport()

// === 计算属性 ===
const canUploadToCurrentDirectory = computed(() =>
  userStore.hasPermission(4) || currentDirectoryAllowsUpload.value,
)
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

const currentDirectoryName = computed(() => {
  if (currentPath.value === '/') return '根目录'
  return currentPath.value.split('/').filter(Boolean).pop() || '根目录'
})

const parentPath = computed(() => {
  if (currentPath.value === '/') return '/'
  const parts = currentPath.value.split('/').filter(Boolean)
  parts.pop()
  return '/' + parts.join('/') || '/'
})

// 前端搜索过滤
const filteredFiles = computed(() => {
  if (!searchKeyword.value) return entries.value
  const keyword = searchKeyword.value.toLowerCase()
  return entries.value.filter((f: any) => f.name.toLowerCase().includes(keyword))
})

const folderCount = computed(() =>
  entries.value.filter((item: any) => item.is_directory).length,
)

const fileCount = computed(() =>
  entries.value.filter((item: any) => !item.is_directory).length,
)

const visibleSize = computed(() =>
  filteredFiles.value.reduce((sum: number, item: any) => {
    if (item.is_directory) return sum
    return sum + (Number(item.size) || 0)
  }, 0),
)

const visibleSizeLabel = computed(() => formatSize(visibleSize.value))

const selectedFileCount = computed(() =>
  selectedItems.value.filter(item => !item.is_directory).length,
)

const selectedFolderCount = computed(() =>
  selectedItems.value.filter(item => item.is_directory).length,
)

// === 文件图标 ===
function getFileIcon(name: string, isDir: boolean): { icon: any; color: string } {
  if (isDir) return { icon: FolderOpen, color: '#faad14' }

  const ext = name.toLowerCase().split('.').pop() || ''

  if (['jpg', 'jpeg', 'png', 'gif', 'svg', 'webp', 'bmp', 'ico', 'tiff', 'tif'].includes(ext))
    return { icon: Image, color: '#52c41a' }
  if (['mp4', 'avi', 'mov', 'mkv', 'webm', 'flv', 'wmv', 'm4v'].includes(ext))
    return { icon: Videocam, color: '#fa8c16' }
  if (['mp3', 'wav', 'flac', 'aac', 'ogg', 'wma', 'm4a', 'ape'].includes(ext))
    return { icon: MusicalNotes, color: '#eb2f96' }
  if (['zip', 'tar', 'gz', 'rar', '7z', 'bz2', 'xz', 'tgz', 'zst'].includes(ext))
    return { icon: Archive, color: '#8c8c8c' }
  if (['js', 'ts', 'jsx', 'tsx', 'vue', 'py', 'go', 'java', 'c', 'cpp', 'h', 'rs', 'rb',
       'php', 'swift', 'kt', 'html', 'css', 'scss', 'less', 'json', 'xml', 'yaml',
       'yml', 'toml', 'sql', 'sh', 'bash', 'cmd', 'ps1', 'bat'].includes(ext))
    return { icon: CodeSlash, color: '#1890ff' }
  if (['pdf', 'doc', 'docx', 'xls', 'xlsx', 'ppt', 'pptx', 'txt', 'md', 'log', 'csv', 'rtf'].includes(ext))
    return { icon: DocumentText, color: '#722ed1' }

  return { icon: Document, color: '#8c8c8c' }
}

// === 时间格式化 ===
function formatTime(dateStr: string): string {
  const d = new Date(dateStr)
  const year = d.getFullYear()
  const month = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  const hours = String(d.getHours()).padStart(2, '0')
  const minutes = String(d.getMinutes()).padStart(2, '0')
  return `${year}-${month}-${day} ${hours}:${minutes}`
}

// === 图标按钮渲染 ===
function iconBtn(iconComp: any, tooltip: string, onClick: () => void, color?: string) {
  return h(NTooltip, { trigger: 'hover', placement: 'top' }, {
    default: () => tooltip,
    trigger: () =>
      h(NButton, { size: 'small', quaternary: true, onClick, class: 'action-btn' }, {
        icon: () => h(NIcon, { size: 18, color: color || undefined }, () => h(iconComp)),
      }),
  })
}

// === 行标识 ===
function rowKey(row: any) {
  return row.path || row.name
}

// === 表格列定义 ===
const columns = computed<DataTableColumns>(() => {
  const cols: DataTableColumns = [
    { type: 'selection' },
    {
      title: '名称',
      key: 'name',
      className: 'col-name',
      sorter: (a: any, b: any) => a.name.localeCompare(b.name),
      render(row: any) {
        const { icon, color } = getFileIcon(row.name, row.is_directory)
        const iconEl = h(NIcon, { size: 18, color, style: { marginRight: '8px', verticalAlign: 'middle', flexShrink: '0' } }, () => h(icon))
        if (row.is_directory) {
          return h('div', { class: 'name-cell' },
            h(NButton, { text: true, onClick: () => navigateTo(row.path) }, () => [iconEl, h('span', { class: 'name-text' }, row.name)]),
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
      width: 80,
      sorter: (a: any, b: any) => a.size - b.size,
      render(row: any) {
        return row.is_directory ? '-' : formatSize(row.size)
      },
    },
  ]

  // 宽视口 (≥768px) 显示次要列
  if (!isMobile.value) {
    cols.push(
      {
        title: '修改时间',
        key: 'mod_time',
        width: 140,
        sorter: (a: any, b: any) => new Date(a.mod_time).getTime() - new Date(b.mod_time).getTime(),
        render(row: any) {
          return formatTime(row.mod_time)
        },
      },
      {
        title: '所有者',
        key: 'owner_name',
        width: 120,
        render(row: any) {
          return isAdmin.value
            ? h(NButton, { text: true, type: 'primary', onClick: () => openOwnerModal(row) }, () => row.owner_name)
            : (row.owner_name as string)
        },
      },
    )
  }

  // 操作列宽度随视口自适应
  cols.push({
    title: '操作',
    key: 'actions',
    width: isMobile.value ? 120 : 180,
    className: 'col-actions',
    render(row: any) {
      const btns: any[] = []

      if (row.is_directory) {
        btns.push(iconBtn(EnterOutline, '进入目录', () => navigateTo(row.path), '#1890ff'))
      } else {
        if (row.can_download) btns.push(iconBtn(CloudDownloadOutline, '下载文件', () => downloadFile(row), '#1890ff'))
        if (hasPermShare.value) btns.push(iconBtn(ShareSocialOutline, '分享文件', () => shareFile(row), '#1890ff'))
      }
      if (row.can_delete) btns.push(iconBtn(TrashOutline, '删除', () => deleteEntry(row), '#d03050'))
      if (row.can_change) btns.push(iconBtn(CreateOutline, '移动/重命名', () => openMoveModal(row)))

      return h(NSpace, { size: 2, wrap: false }, () => btns)
    },
  })

  return cols
})

// === 行样式 ===
function rowClassName(row: any) {
  const classes: string[] = []
  if (highlightName.value && row.name === highlightName.value) {
    classes.push('highlighted-row')
  }
  const key = row.path || row.name
  if (checkedKeySet.value.has(key)) {
    classes.push('checked-row')
  }
  return classes.join(' ')
}

// === 多选 ===
function onCheckedKeysChange(keys: DataTableRowKey[]) {
  const newKeySet = new Set(keys.map(String))
  const current = [...selectedItems.value]

  // 保留仍在选中集合中的项
  const kept = current.filter(i => newKeySet.has(i.key))

  // 新选中的项：从当前 entries 获取元数据
  for (const k of newKeySet) {
    const key = String(k)
    if (!kept.some(i => i.key === key)) {
      const entry = entries.value.find((e: any) => (e.path || e.name) === key)
      if (entry) {
        kept.push({ key, name: entry.name, is_directory: entry.is_directory })
      }
    }
  }

  selectedItems.value = kept
}

// === 批量操作 ===
function batchDelete() {
  const keys = [...checkedKeys.value]
  const count = keys.length
  const d = dialog.warning({
    title: '确认批量删除',
    content: `确认删除选中的 ${count} 个项目？此操作不可恢复。`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      d.loading = true
      let successCount = 0
      for (const key of keys) {
        try {
          await api.delete('/api/files', { data: { path: key } })
          successCount++
        } catch (err: any) {
          message.error(`删除 "${key}" 失败: ${err.response?.data?.error || '未知错误'}`)
        }
      }
      if (successCount > 0) {
        message.success(`成功删除 ${successCount} 项`)
      }
      selectedItems.value = []
      d.loading = false
      await fetchFiles()
    },
  })
}

function batchDownload() {
  const files = selectedItems.value.filter(i => !i.is_directory)
  if (files.length === 0) {
    message.warning('选中项中没有可下载的文件')
    return
  }
  for (const f of files) {
    window.open(`/api/download?path=${encodeURIComponent(f.key)}`, '_blank')
  }
}

function batchShare() {
  const filePaths = selectedItems.value
    .filter(i => !i.is_directory)
    .map(i => i.key)

  if (filePaths.length === 0) {
    message.warning('请至少选择一个文件（目录不可分享）')
    return
  }

  openShareModal(filePaths)
}

// === 网格视图选中（仅 checkbox 触发）===
function toggleGridSelection(key: string, checked: boolean) {
  if (checked) {
    const item = entries.value.find((e: any) => (e.path || e.name) === key)
    if (item) {
      selectedItems.value = [...selectedItems.value, {
        key,
        name: item.name,
        is_directory: item.is_directory,
      }]
    }
  } else {
    selectedItems.value = selectedItems.value.filter(i => i.key !== key)
  }
}

// === 从已选列表中移除单项 ===
function removeSelectedItem(key: string) {
  selectedItems.value = selectedItems.value.filter(i => i.key !== key)
}

// === 已选文件下拉列表 ===
const selectedDropdownOptions = computed(() =>
  selectedItems.value.map(item => ({
    key: item.key,
    label: item.name,
  }))
)

// === 下拉标签自定义渲染（renderLabel prop，保留选项容器的padding与垂直排列） ===
function renderDropdownLabel(option: any) {
  const item = selectedItems.value.find(i => i.key === option.key)
  return h(NTag, {
    type: item?.is_directory ? 'error' : 'info',
    closable: true,
    size: 'small',
    onClose: () => removeSelectedItem(option.key),
  }, { default: () => option.label })
}

// === 网格卡片点击（非 checkbox 区域）===
function onGridCardClick(file: any) {
  if (file.is_directory) {
    navigateTo(file.path)
  }
  // 文件暂不做任何操作，预留后期文件预览功能入口
}

// === 高亮逻辑 ===
function applyHighlight() {
  const name = route.query.highlight as string | undefined
  if (highlightTimer) {
    clearTimeout(highlightTimer)
    highlightTimer = null
  }
  if (name) {
    highlightName.value = name
    highlightTimer = setTimeout(() => {
      highlightName.value = null
    }, 4000)
  } else {
    highlightName.value = null
  }
}

// === 生命周期 ===
onMounted(() => {
  const p = (route.query.path as string) || '/'
  currentPath.value = p
  fetchFiles()
  if (isAdmin.value) fetchAllUsers()

  // 读取外部跳转带入的 highlight 参数后，清除 URL 中的 query 参数
  if (route.query.path || route.query.highlight) {
    applyHighlight()
    router.replace({ query: {} })
  }
})

// === 数据获取 ===
async function fetchFiles() {
  loading.value = true
  currentDirectoryAllowsUpload.value = false
  try {
    const res = await api.get('/api/files', { params: { path: currentPath.value } })
    entries.value = res.data.entries || []
    currentDirectoryAllowsUpload.value = res.data.can_upload === true
    permissionDenied.value = false
    permissionDeniedMsg.value = ''
    applyHighlight()
  } catch (err: any) {
    console.error('fetchFiles failed:', err)
    entries.value = []
    currentDirectoryAllowsUpload.value = false
    if (err.response?.status === 403) {
      permissionDenied.value = true
      permissionDeniedMsg.value = err.response?.data?.error || '您没有访问此目录的权限'
    } else {
      permissionDenied.value = false
      permissionDeniedMsg.value = ''
      message.error(err.response?.data?.error || '获取文件列表失败')
    }
  } finally {
    loading.value = false
  }
}

// === 导航 ===
function navigateTo(path: string) {
  if (path === currentPath.value) return
  currentPath.value = path
  searchKeyword.value = ''
  // 跨目录保留选择，不在此处清除 selectedItems
  fetchFiles()
}

function goToParent() {
  navigateTo(parentPath.value)
}

function refresh() {
  fetchFiles()
}

// === 文件操作 ===
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

function deleteEntry(row: any) {
  const d = dialog.warning({
    title: '确认删除',
    content: `确认删除 "${row.name}"？此操作不可恢复。`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      d.loading = true
      try {
        await api.delete('/api/files', { data: { path: row.path } })
        message.success('删除成功')
      } catch (err: any) {
        message.error(err.response?.data?.error || '删除失败')
      } finally {
        d.loading = false
      }
      await fetchFiles()
    },
  })
}

function downloadFile(row: any) {
  window.open(`/api/download?path=${encodeURIComponent(row.path)}`, '_blank')
}

function shareFile(row: any) {
  openShareModal([row.path])
}

function openShareModal(filePaths: string[]) {
  shareFilePaths.value = [...filePaths]
  shareExpireDays.value = 3
  shareName.value = buildDefaultShareName(filePaths.length)
  showShareModal.value = true
}

function buildDefaultShareName(fileCount: number) {
  const sharerName = userStore.user?.display_name || userStore.user?.username || '用户'
  return `由 ${sharerName} 分享的 ${fileCount} 个文件`
}

function removeShareFilePath(filePath: string) {
  const previousDefaultName = buildDefaultShareName(shareFilePaths.value.length)
  shareFilePaths.value = shareFilePaths.value.filter(path => path !== filePath)

  if (shareFilePaths.value.length === 0) {
    cancelShareCreation()
    return
  }

  if (shareName.value === previousDefaultName) {
    shareName.value = buildDefaultShareName(shareFilePaths.value.length)
  }
}

function cancelShareCreation() {
  showShareModal.value = false
  shareFilePaths.value = []
  shareName.value = ''
}

async function handleCreateShare() {
  shareLoading.value = true
  try {
    const res = await api.post('/api/shares', {
      file_paths: shareFilePaths.value,
      expire_days: shareExpireDays.value,
      name: shareName.value,
    })
    message.success('分享创建成功')
    showShareModal.value = false
    userStore.onShareCreated()
    router.push({ path: '/shares', query: { highlightId: String(res.data.id) } })
  } catch (e: any) {
    message.error(e.response?.data?.error || '创建分享失败')
  } finally {
    shareLoading.value = false
  }
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
.file-explorer {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
  overflow: hidden;
  color: var(--fe-text);
  background: transparent;

  --fe-page: #f4f7fb;
  --fe-surface: #f8fafc;
  --fe-surface-strong: #eef3f8;
  --fe-surface-soft: #f1f5f9;
  --fe-field: #f8fafc;
  --fe-text: #152033;
  --fe-text-muted: #637083;
  --fe-text-subtle: #8a97aa;
  --fe-border: #dbe3ee;
  --fe-border-soft: #e7edf5;
  --fe-accent: var(--theme-color, #3b82f6);
  --fe-accent-rgb: var(--theme-color-rgb, 59, 130, 246);
  --fe-danger: #c2415b;
  --fe-success: #168a5b;
  --fe-folder: #c78215;
  --fe-radius-lg: 18px;
  --fe-radius-md: 14px;
  --fe-radius-sm: 10px;
  --fe-shadow-soft: 0 18px 50px rgba(42, 59, 87, 0.10);
  --fe-shadow-card: 0 12px 34px rgba(42, 59, 87, 0.12);
  --fe-row-hover: rgba(var(--fe-accent-rgb), 0.055);
  --fe-row-selected: rgba(var(--fe-accent-rgb), 0.105);
  --fe-control-shadow: 0 8px 22px rgba(var(--fe-accent-rgb), 0.13);
}

.dark {
  --fe-page: #0f172a;
  --fe-surface: #172033;
  --fe-surface-strong: #1e2a42;
  --fe-surface-soft: #111a2d;
  --fe-field: #111827;
  --fe-text: #edf3fb;
  --fe-text-muted: #a7b2c3;
  --fe-text-subtle: #78869a;
  --fe-border: #2b3a51;
  --fe-border-soft: #223047;
  --fe-danger: #fb7185;
  --fe-success: #34d399;
  --fe-folder: #f7b955;
  --fe-shadow-soft: 0 20px 56px rgba(2, 6, 23, 0.38);
  --fe-shadow-card: 0 16px 42px rgba(2, 6, 23, 0.35);
  --fe-row-hover: rgba(var(--fe-accent-rgb), 0.13);
  --fe-row-selected: rgba(var(--fe-accent-rgb), 0.20);
  --fe-control-shadow: 0 10px 26px rgba(2, 6, 23, 0.28);
}

.explorer-overview {
  position: relative;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 12px;
  padding: 12px 14px;
  overflow: hidden;
  border: 1px solid var(--fe-border);
  border-radius: var(--fe-radius-lg);
  background:
    linear-gradient(135deg, rgba(var(--fe-accent-rgb), 0.12), transparent 34%),
    linear-gradient(180deg, var(--fe-surface), var(--fe-surface-soft));
  box-shadow: inset 0 1px 0 rgba(248, 250, 252, 0.70);
}

.explorer-overview::before {
  content: "";
  position: absolute;
  inset: 0;
  pointer-events: none;
  background:
    linear-gradient(90deg, rgba(248, 250, 252, 0.55), transparent 40%),
    radial-gradient(circle at 92% 20%, rgba(var(--fe-accent-rgb), 0.12), transparent 28%);
}

.dark .explorer-overview::before {
  background:
    linear-gradient(90deg, rgba(15, 23, 42, 0.18), transparent 40%),
    radial-gradient(circle at 92% 20%, rgba(var(--fe-accent-rgb), 0.18), transparent 30%);
}

.dark .explorer-overview {
  box-shadow: inset 0 1px 0 rgba(248, 250, 252, 0.08);
}

.overview-main,
.overview-side {
  position: relative;
  z-index: 1;
}

.overview-main {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.overview-icon {
  display: grid;
  place-items: center;
  width: 42px;
  height: 42px;
  flex: 0 0 auto;
  border: 1px solid rgba(var(--fe-accent-rgb), 0.18);
  border-radius: 13px;
  color: var(--fe-accent);
  background: rgba(var(--fe-accent-rgb), 0.10);
  box-shadow: inset 0 1px 0 rgba(248, 250, 252, 0.68);
}

.overview-icon :deep(svg) {
  width: 21px;
  height: 21px;
}

.dark .overview-icon {
  box-shadow: inset 0 1px 0 rgba(248, 250, 252, 0.10);
}

.overview-copy {
  min-width: 0;
}

.breadcrumb {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 5px;
  min-width: 0;
  margin-bottom: 3px;
  font-size: 12px;
  color: var(--fe-text-muted);
}

.breadcrumb-link {
  max-width: 180px;
  overflow: hidden;
  border: 0;
  padding: 0;
  background: transparent;
  color: var(--fe-accent);
  cursor: pointer;
  font: inherit;
  text-overflow: ellipsis;
  white-space: nowrap;
  transition: color 0.18s ease, opacity 0.18s ease;
}

.breadcrumb-link:hover {
  color: var(--theme-color-hover, var(--fe-accent));
}

.breadcrumb-link:focus-visible,
.view-switch-btn:focus-visible,
.batch-info-clickable:focus-visible,
.card-action-btn:focus-visible {
  outline: 2px solid rgba(var(--fe-accent-rgb), 0.50);
  outline-offset: 2px;
}

.breadcrumb-sep {
  color: var(--fe-text-subtle);
}

.breadcrumb-current {
  max-width: 220px;
  overflow: hidden;
  font-weight: 700;
  color: var(--fe-text);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.overview-copy h1 {
  margin: 0;
  overflow: hidden;
  color: var(--fe-text);
  font-size: clamp(20px, 2.2vw, 26px);
  line-height: 1.12;
  font-weight: 800;
  letter-spacing: -0.02em;
  text-overflow: ellipsis;
  white-space: nowrap;
  text-wrap: balance;
}

.overview-side {
  display: flex;
  align-items: stretch;
}

.stat-strip {
  display: grid;
  grid-template-columns: repeat(3, minmax(80px, auto));
  min-width: 286px;
  overflow: hidden;
  border: 1px solid var(--fe-border-soft);
  border-radius: 12px;
  background: rgba(248, 250, 252, 0.66);
  backdrop-filter: blur(14px);
}

.dark .stat-strip {
  background: rgba(15, 23, 42, 0.28);
}

.stat-item {
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 2px;
  min-height: 50px;
  padding: 8px 10px;
}

.stat-item + .stat-item {
  border-left: 1px solid var(--fe-border-soft);
}

.stat-value {
  color: var(--fe-text);
  font-size: 16px;
  line-height: 1;
  font-weight: 800;
  letter-spacing: -0.02em;
  font-variant-numeric: tabular-nums;
}

.stat-label {
  color: var(--fe-text-muted);
  font-size: 11px;
  line-height: 1.3;
}

.header-tools-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  flex-wrap: wrap;
}

.search-input {
  flex: 1 1 220px;
  width: auto;
  min-width: 180px;
  max-width: 420px;
}

.search-input :deep(.n-input) {
  min-height: 40px;
  border-radius: 10px;
  background-color: var(--fe-field);
}

.view-switch {
  display: inline-flex;
  gap: 4px;
  padding: 3px;
  border: 1px solid var(--fe-border-soft);
  border-radius: 11px;
  background: var(--fe-surface-strong);
}

.view-switch-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  min-width: 40px;
  height: 40px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--fe-text-muted);
  cursor: pointer;
  font-size: 13px;
  font-weight: 700;
  transition:
    background-color 0.18s ease,
    color 0.18s ease,
    box-shadow 0.18s ease,
    transform 0.18s ease;
}

.view-switch-btn:hover {
  color: var(--fe-text);
}

.view-switch-btn.active {
  color: var(--fe-accent);
  background: var(--fe-surface);
  box-shadow: 0 6px 16px rgba(42, 59, 87, 0.10);
}

.dark .view-switch-btn.active {
  box-shadow: 0 8px 18px rgba(2, 6, 23, 0.28);
}

.view-switch-btn:active,
.card-action-btn:active,
.batch-info-clickable:active {
  transform: translateY(1px);
}

.header-tools-actions {
  flex: 0 1 auto;
  justify-content: flex-start;
}

.header-tools-actions :deep(.n-button) {
  min-height: 40px;
  border-radius: 10px;
  font-weight: 700;
}

.header-tools-actions :deep(.n-button--primary-type) {
  box-shadow: var(--fe-control-shadow);
}

.batch-bar {
  position: absolute;
  right: 14px;
  bottom: 14px;
  left: 14px;
  z-index: 2;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  max-width: 820px;
  margin: 0 auto;
  padding: 8px 10px;
  border: 1px solid rgba(var(--fe-accent-rgb), 0.24);
  border-radius: 14px;
  background:
    linear-gradient(90deg, rgba(var(--fe-accent-rgb), 0.12), rgba(var(--fe-accent-rgb), 0.06)),
    var(--fe-surface);
  box-shadow:
    0 0 0 1px rgba(255, 255, 255, 0.42) inset,
    0 16px 34px rgba(42, 59, 87, 0.18);
  backdrop-filter: blur(14px);
}

.dark .batch-bar {
  box-shadow:
    0 0 0 1px rgba(255, 255, 255, 0.08) inset,
    0 16px 34px rgba(2, 6, 23, 0.38);
}

.batch-summary,
.batch-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.batch-summary {
  color: var(--fe-text-muted);
  font-size: 12px;
}

.batch-info {
  border: 0;
  color: var(--fe-accent);
  background: transparent;
  font-size: 12px;
  font-weight: 800;
}

.batch-info-clickable {
  cursor: pointer;
  padding: 4px 7px;
  border-radius: 9px;
  transition: background-color 0.18s ease, transform 0.18s ease;
}

.batch-info-clickable:hover {
  background: rgba(var(--fe-accent-rgb), 0.12);
}

.batch-actions :deep(.n-button) {
  min-height: 30px;
  border-radius: 9px;
  font-weight: 700;
}

.content-surface {
  position: relative;
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--fe-border);
  border-radius: var(--fe-radius-lg);
  background: var(--fe-surface);
  box-shadow: 0 1px 0 rgba(248, 250, 252, 0.70) inset;
}

.content-surface.has-floating-batch .file-grid-container {
  padding-bottom: 62px;
}

.dark .content-surface {
  box-shadow: 0 1px 0 rgba(248, 250, 252, 0.08) inset;
}

.content-header {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  padding: 8px 10px;
  border-bottom: 1px solid var(--fe-border-soft);
}

.content-header .view-switch {
  flex: 0 0 auto;
  margin-left: auto;
}

.permission-denied-result {
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
}

.file-list {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--fe-surface);
}

.file-grid-container {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  background:
    linear-gradient(180deg, rgba(var(--fe-accent-rgb), 0.035), transparent 180px),
    var(--fe-surface);
}

.file-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 14px;
  padding: 16px;
}

.grid-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 260px;
}

.grid-card {
  position: relative;
  display: flex;
  flex-direction: column;
  min-height: 192px;
  padding: 12px;
  overflow: hidden;
  border: 1px solid var(--fe-border-soft);
  border-radius: var(--fe-radius-md);
  background: var(--fe-surface);
  cursor: pointer;
  user-select: none;
  transition:
    border-color 0.18s ease,
    box-shadow 0.18s ease,
    transform 0.18s ease,
    background-color 0.18s ease;
  content-visibility: auto;
  contain-intrinsic-size: auto 192px;
}

.grid-card::before {
  content: "";
  position: absolute;
  inset: 0;
  pointer-events: none;
  background: linear-gradient(135deg, rgba(var(--fe-accent-rgb), 0.09), transparent 45%);
  opacity: 0;
  transition: opacity 0.18s ease;
}

.grid-card:hover {
  transform: translateY(-2px);
  border-color: rgba(var(--fe-accent-rgb), 0.34);
  box-shadow: var(--fe-shadow-card);
}

.grid-card:hover::before {
  opacity: 1;
}

.grid-card-selected {
  border-color: rgba(var(--fe-accent-rgb), 0.56);
  background: rgba(var(--fe-accent-rgb), 0.095);
}

.grid-card-selected:hover {
  box-shadow: 0 14px 36px rgba(var(--fe-accent-rgb), 0.18);
}

.grid-card-top {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-height: 28px;
}

.grid-card-checkbox {
  opacity: 0;
  transition: opacity 0.18s ease;
}

.grid-card:hover .grid-card-checkbox,
.grid-card-selected .grid-card-checkbox {
  opacity: 1;
}

.grid-card-kind {
  max-width: 116px;
  overflow: hidden;
  padding: 4px 8px;
  border-radius: 999px;
  color: var(--fe-text-muted);
  background: var(--fe-surface-strong);
  font-size: 11px;
  font-weight: 700;
  line-height: 1.2;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.grid-card-icon {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 62px;
  height: 62px;
  margin: 10px auto 12px;
  border: 1px solid transparent;
  border-radius: 18px;
}

.icon-folder {
  border-color: rgba(199, 130, 21, 0.22);
  background:
    linear-gradient(135deg, rgba(199, 130, 21, 0.16), rgba(199, 130, 21, 0.07)),
    var(--fe-surface-soft);
}

.icon-file {
  border-color: rgba(var(--fe-accent-rgb), 0.18);
  background:
    linear-gradient(135deg, rgba(var(--fe-accent-rgb), 0.14), rgba(var(--fe-accent-rgb), 0.05)),
    var(--fe-surface-soft);
}

.grid-card-body {
  position: relative;
  z-index: 1;
  display: flex;
  flex: 1;
  flex-direction: column;
  align-items: center;
  min-width: 0;
  text-align: center;
}

.grid-card-name {
  display: -webkit-box;
  width: 100%;
  min-height: 40px;
  margin: 0;
  overflow: hidden;
  color: var(--fe-text);
  font-size: 13px;
  font-weight: 800;
  line-height: 1.45;
  overflow-wrap: anywhere;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.grid-card-info {
  width: 100%;
  margin: 5px 0 0;
  overflow: hidden;
  color: var(--fe-text-muted);
  font-size: 11px;
  line-height: 1.35;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.grid-card-actions {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  width: 100%;
  min-height: 34px;
  margin-top: 10px;
  opacity: 0;
  transition: opacity 0.18s ease;
}

.grid-card:hover .grid-card-actions,
.grid-card-selected .grid-card-actions {
  opacity: 1;
}

.card-action-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border: 1px solid var(--fe-border-soft);
  border-radius: 10px;
  background: var(--fe-surface-strong);
  color: var(--fe-text-muted);
  cursor: pointer;
  transition:
    background-color 0.18s ease,
    border-color 0.18s ease,
    color 0.18s ease,
    transform 0.18s ease;
}

.card-action-btn:hover {
  border-color: rgba(var(--fe-accent-rgb), 0.22);
  background: rgba(var(--fe-accent-rgb), 0.11);
  color: var(--fe-accent);
}

.card-action-btn.primary {
  color: var(--fe-accent);
}

.card-action-btn.danger {
  color: var(--fe-danger);
}

.card-action-btn.danger:hover {
  border-color: rgba(194, 65, 91, 0.24);
  background: rgba(194, 65, 91, 0.10);
  color: var(--fe-danger);
}

.file-data-table {
  flex: 1;
}

.file-data-table :deep(.n-data-table) {
  color: var(--fe-text);
  background-color: var(--fe-surface);
}

.file-data-table :deep(.n-data-table-wrapper) {
  background-color: var(--fe-surface);
}

.file-data-table :deep(.n-data-table-td),
.file-data-table :deep(.n-data-table-th) {
  padding-top: 10px !important;
  padding-bottom: 10px !important;
  font-size: 13px;
  border-color: var(--fe-border-soft) !important;
}

.file-data-table :deep(.n-data-table-th) {
  color: var(--fe-text-muted);
  background-color: var(--fe-surface-soft) !important;
  font-size: 12px;
  font-weight: 800;
}

.file-data-table :deep(.n-data-table-td) {
  background-color: var(--fe-surface);
}

.file-data-table :deep(.n-data-table-tr:hover > .n-data-table-td) {
  background-color: var(--fe-row-hover) !important;
}

.file-data-table :deep(.checked-row > .n-data-table-td) {
  background-color: var(--fe-row-selected) !important;
}

.file-data-table :deep(.highlighted-row > .n-data-table-td) {
  background-color: rgba(22, 138, 91, 0.12) !important;
  transition: background-color 0.3s ease;
}

.file-data-table :deep(.col-name),
.file-data-table :deep(.col-name .n-data-table-td__ellipsis) {
  white-space: normal !important;
}

.file-data-table :deep(.name-cell) {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.file-data-table :deep(.name-text) {
  color: var(--fe-text);
  font-weight: 800;
  white-space: normal;
  word-break: break-all;
  overflow-wrap: anywhere;
  line-height: 1.45;
}

.file-data-table :deep(.col-name .n-button__content) {
  color: var(--fe-accent);
  font-weight: 800;
  white-space: normal;
  word-break: break-all;
  overflow-wrap: anywhere;
  text-align: left;
  line-height: 1.45;
}

.file-data-table :deep(.col-actions) {
  white-space: nowrap;
}

.file-data-table :deep(.col-actions .action-btn) {
  opacity: 0;
  border-radius: 10px;
  transition:
    opacity 0.18s ease,
    background-color 0.18s ease,
    transform 0.18s ease;
}

.file-data-table :deep(.n-data-table-tr:hover .col-actions .action-btn),
.file-data-table :deep(.checked-row .col-actions .action-btn) {
  opacity: 1;
}

.file-data-table :deep(.col-actions .action-btn:active) {
  transform: translateY(1px);
}

.dark .file-data-table :deep(.n-data-table),
.dark .file-data-table :deep(.n-data-table-wrapper) {
  background-color: var(--fe-surface);
}

.dark .file-data-table :deep(.n-data-table-th) {
  background-color: var(--fe-surface-soft) !important;
}

.dark .file-data-table :deep(.n-data-table-td) {
  background-color: var(--fe-surface);
}

.dark .file-data-table :deep(.n-data-table-base-table) {
  border-color: var(--fe-border);
}

.share-file-list {
  width: 100%;
  min-width: 0;
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  gap: 4px 8px;
}

.share-file-tag {
  min-width: 0;
  max-width: 100%;
}

.share-file-tag :deep(.n-tag__content) {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.share-file-count {
  flex: 0 0 100%;
  margin: 0;
  color: var(--fe-text-muted);
  font-size: 12px;
  line-height: 1.4;
}

@media (hover: none) {
  .grid-card-checkbox,
  .grid-card-actions,
  .file-data-table :deep(.col-actions .action-btn) {
    opacity: 1;
  }
}

@media (max-width: 1024px) {
  .explorer-overview {
    grid-template-columns: 1fr;
  }

  .overview-side {
    justify-content: stretch;
  }

  .stat-strip {
    width: 100%;
    min-width: 0;
  }

}

@media (max-width: 767px) {
  .file-explorer {
    gap: 9px;
  }

  .explorer-overview {
    padding: 10px 12px;
  }

  .overview-main {
    align-items: flex-start;
  }

  .overview-icon {
    width: 38px;
    height: 38px;
    border-radius: 12px;
  }

  .overview-copy h1 {
    font-size: 21px;
    white-space: normal;
  }

  .stat-strip {
    grid-template-columns: 1fr 1fr;
  }

  .stat-item {
    min-height: 44px;
    padding: 7px 9px;
  }

  .stat-value {
    font-size: 15px;
  }

  .stat-item-wide {
    grid-column: 1 / -1;
    border-top: 1px solid var(--fe-border-soft);
    border-left: 0 !important;
  }

  .content-header {
    padding: 8px;
  }

  .header-tools-actions {
    width: 100%;
  }

  .search-input {
    min-width: 160px;
    max-width: none;
  }

  .view-switch {
    width: auto;
  }

  .header-tools-actions :deep(.n-button) {
    flex: 1 1 calc(50% - 6px);
  }

  .batch-bar {
    right: 10px;
    bottom: 10px;
    left: 10px;
    align-items: stretch;
    flex-direction: column;
    gap: 8px;
  }

  .content-surface.has-floating-batch .file-grid-container {
    padding-bottom: 108px;
  }

  .batch-actions :deep(.n-button) {
    flex: 1 1 calc(50% - 6px);
  }

  .file-grid {
    grid-template-columns: repeat(auto-fill, minmax(148px, 1fr));
    gap: 12px;
    padding: 12px;
  }

  .grid-card {
    min-height: 182px;
  }

  .file-data-table :deep(.n-data-table-td),
  .file-data-table :deep(.n-data-table-th) {
    padding-top: 8px !important;
    padding-bottom: 8px !important;
  }
}

@media (prefers-reduced-motion: reduce) {
  .file-explorer *,
  .file-explorer *::before,
  .file-explorer *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    scroll-behavior: auto !important;
    transition-duration: 0.01ms !important;
  }
}
</style>
