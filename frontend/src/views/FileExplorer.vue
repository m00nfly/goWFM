<template>
  <div class="file-explorer" :class="{ dark: themeStore.isDark }">
    <!-- 面包屑导航 -->
    <div class="breadcrumb">
      <n-icon size="18" class="breadcrumb-icon">
        <FolderOpenOutline />
      </n-icon>
      <span class="breadcrumb-link" @click="navigateTo('/')">根目录</span>
      <template v-for="(seg, idx) in pathSegments" :key="seg.path">
        <span class="breadcrumb-sep">/</span>
        <span
          v-if="idx === pathSegments.length - 1"
          class="breadcrumb-current"
        >{{ seg.name }}</span>
        <span
          v-else
          class="breadcrumb-link"
          @click="navigateTo(seg.path)"
        >{{ seg.name }}</span>
      </template>
    </div>

    <!-- 操作工具栏 -->
    <div class="toolbar">
      <div class="toolbar-left">
        <n-button v-if="hasPermUpload" @click="showMkdirModal = true">
          <template #icon><n-icon><AddCircleOutline /></n-icon></template>
          新建目录
        </n-button>
        <n-button v-if="hasPermUpload" type="primary" @click="showUploadModal = true">
          <template #icon><n-icon><CloudUploadOutline /></n-icon></template>
          上传文件
        </n-button>
        <n-button v-if="currentPath !== '/'" @click="goToParent">
          <template #icon><n-icon><ArrowBackOutline /></n-icon></template>
          返回
        </n-button>
      </div>
      <!-- 批量操作栏 -->
      <div v-if="selectedItems.length > 0" class="batch-bar">
        <n-dropdown trigger="hover" :options="selectedDropdownOptions" :render-label="renderDropdownLabel" placement="bottom-start" :max-height="300" scrollable>
          <span class="batch-info batch-info-clickable">已选择 {{ selectedItems.length }} 项</span>
        </n-dropdown>
        <n-button size="small" @click="batchDownload" v-if="userStore.hasPermission(2)">批量下载</n-button>
        <n-button size="small" @click="batchShare" v-if="hasPermShare">批量分享</n-button>
        <n-button size="small" type="error" @click="batchDelete" v-if="userStore.user?.is_admin || userStore.hasPermission(4)">批量删除</n-button>
        <n-button size="small" quaternary @click="selectedItems = []">取消选择</n-button>
      </div>
      <div class="toolbar-right">
        <n-input
          v-model:value="searchKeyword"
          placeholder="搜索文件..."
          clearable
          class="search-input"
        >
          <template #prefix>
            <n-icon><SearchOutline /></n-icon>
          </template>
        </n-input>
        <n-button quaternary circle @click="refresh">
          <template #icon><n-icon><RefreshOutline /></n-icon></template>
        </n-button>
        <n-button quaternary circle @click="toggleViewMode">
          <template #icon>
            <n-icon><ListOutline v-if="viewMode === 'grid'" /><GridOutline v-else /></n-icon>
          </template>
        </n-button>
      </div>
    </div>

    <!-- 无权限提示 -->
    <n-result
      v-if="permissionDenied"
      status="403"
      title="无访问权限"
      :description="permissionDeniedMsg"
      class="permission-denied-result"
    />

    <!-- 文件列表 (列表模式) -->
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

    <!-- 网格视图 -->
    <div v-else class="file-grid-container">
      <div v-if="filteredFiles.length === 0" class="grid-empty">
        <n-empty description="暂无文件" />
      </div>
      <div v-else class="file-grid">
        <div
          v-for="file in filteredFiles"
          :key="file.path || file.name"
          class="grid-card"
          :class="{ 'grid-card-selected': checkedKeySet.has(file.path || file.name) }"
          @click="onGridCardClick(file)"
        >
          <n-checkbox
            class="grid-card-checkbox"
            :checked="checkedKeySet.has(file.path || file.name)"
            @update:checked="toggleGridSelection(file.path || file.name, $event)"
            @click.stop
          />
          <!-- 图标 - 带彩色背景 -->
          <div class="grid-card-icon" :class="{ 'icon-folder': file.is_directory, 'icon-file': !file.is_directory }">
            <n-icon :size="32" :color="getFileIcon(file.name, file.is_directory).color">
              <component :is="getFileIcon(file.name, file.is_directory).icon" />
            </n-icon>
          </div>
          <div class="grid-card-name" :title="file.name">{{ file.name }}</div>
          <div class="grid-card-info">
            <span v-if="!file.is_directory">{{ formatSize(file.size) }}</span>
            <span v-else>文件夹</span>
          </div>
          <!-- 操作按钮 - 底部居中 -->
          <div class="grid-card-actions">
            <n-tooltip v-if="file.is_directory" trigger="hover" placement="top">
              <template #trigger>
                <button class="card-action-btn" @click.stop="navigateTo(file.path)">
                  <n-icon size="16" color="#3b82f6"><EnterOutline /></n-icon>
                </button>
              </template>
              进入目录
            </n-tooltip>
            <template v-else>
              <n-tooltip v-if="file.can_download" trigger="hover" placement="top">
                <template #trigger>
                  <button class="card-action-btn" @click.stop="downloadFile(file)">
                    <n-icon size="16" color="#3b82f6"><CloudDownloadOutline /></n-icon>
                  </button>
                </template>
                下载
              </n-tooltip>
              <n-tooltip v-if="hasPermShare" trigger="hover" placement="top">
                <template #trigger>
                  <button class="card-action-btn" @click.stop="shareFile(file)">
                    <n-icon size="16" color="#3b82f6"><ShareSocialOutline /></n-icon>
                  </button>
                </template>
                分享
              </n-tooltip>
            </template>
            <n-tooltip v-if="file.can_delete" trigger="hover" placement="top">
              <template #trigger>
                <button class="card-action-btn" @click.stop="deleteEntry(file)">
                  <n-icon size="16" color="#ef4444"><TrashOutline /></n-icon>
                </button>
              </template>
              删除
            </n-tooltip>
            <n-tooltip v-if="file.can_change" trigger="hover" placement="top">
              <template #trigger>
                <button class="card-action-btn" @click.stop="openMoveModal(file)">
                  <n-icon size="16" color="#64748b"><CreateOutline /></n-icon>
                </button>
              </template>
              移动/重命名
            </n-tooltip>
          </div>
        </div>
      </div>
    </div>

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
    <n-modal v-model:show="showShareModal" preset="dialog" title="创建文件分享" positive-text="创建" negative-text="取消" :positive-button-props="{ loading: shareLoading }" @positive-click="handleCreateShare" @negative-click="showShareModal = false" :mask-closable="false">
      <n-form label-placement="left" label-width="80">
        <n-form-item label="文件">
          <div v-if="shareFilePaths.length === 1">
            <n-input :value="shareFilePaths[0]" readonly />
          </div>
          <div v-else class="share-file-list">
            <n-tag v-for="p in shareFilePaths" :key="p" size="small" style="margin: 2px 4px;">{{ p.split('/').pop() }}</n-tag>
            <p style="color: #999; font-size: 12px; margin-top: 4px;">共 {{ shareFilePaths.length }} 个文件</p>
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
        return row.is_directory ? '—' : formatSize(row.size)
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

  shareFilePaths.value = filePaths
  shareExpireDays.value = 3
  const displayName = userStore.user?.display_name || userStore.user?.username || ''
  shareName.value = `由${displayName}分享的${filePaths.length}个文件`
  showShareModal.value = true
}

// === 视图切换 ===
function toggleViewMode() {
  viewMode.value = viewMode.value === 'list' ? 'grid' : 'list'
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

function downloadIfAllowed(file: any) {
  if (file.can_download) {
    downloadFile(file)
  }
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
  try {
    const res = await api.get('/api/files', { params: { path: currentPath.value } })
    entries.value = res.data.entries || []
    permissionDenied.value = false
    permissionDeniedMsg.value = ''
    applyHighlight()
  } catch (err: any) {
    console.error('fetchFiles failed:', err)
    entries.value = []
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
  shareFilePaths.value = [row.path]
  shareExpireDays.value = 7
  shareName.value = row.name
  showShareModal.value = true
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
/* ===== 容器 & 设计令牌 ===== */
.file-explorer {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;

  /* ── 浅色模式 ── */
  --fe-bg-card: #fff;
  --fe-bg-action: #f1f5f9;
  --fe-fg-primary: #1e293b;
  --fe-fg-muted: #94a3b8;
  --fe-border-card: rgba(var(--theme-color-rgb, 59, 130, 246), 0.1);

  /* 主题色透明度档位 */
  --fe-a-row-hover: 0.04;
  --fe-a-row-sel: 0.08;
  --fe-a-card-hover-border: 0.35;
  --fe-a-card-shadow: 0.1;
  --fe-a-batch-bg: 0.06;
  --fe-a-batch-border: 0.2;
  --fe-a-action-hover: 0.1;
  --fe-a-file-icon: 0.1;
}

/* ===== 暗色模式 ===== */
.dark {
  --fe-bg-card: #1e293b;
  --fe-bg-action: #334155;
  --fe-fg-primary: #f1f5f9;
  --fe-fg-muted: #64748b;
  --fe-border-card: rgba(var(--theme-color-rgb, 59, 130, 246), 0.15);

  --fe-a-row-hover: 0.1;
  --fe-a-row-sel: 0.18;
  --fe-a-card-hover-border: 0.5;
  --fe-a-card-shadow: 0.2;
  --fe-a-batch-bg: 0.12;
  --fe-a-batch-border: 0.3;
  --fe-a-action-hover: 0.18;
  --fe-a-file-icon: 0.12;
}

/* ===== 面包屑 ===== */
.breadcrumb {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px;
  font-size: 14px;
  padding-bottom: 4px;
}

.breadcrumb-icon {
  margin-right: 6px;
  vertical-align: middle;
  color: var(--fe-fg-muted);
}

.breadcrumb-link {
  color: var(--theme-color, #3b82f6);
  cursor: pointer;
  transition: opacity 0.2s;
}
.breadcrumb-link:hover { opacity: 0.8; }

.breadcrumb-sep  { color: var(--fe-fg-muted); }
.breadcrumb-current { font-weight: 700; color: var(--fe-fg-primary); }

/* ===== 工具栏 ===== */
.toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  padding: 8px 0;
}

.search-input {
  width: 220px;
}

.toolbar-left,
.toolbar-right { display: flex; align-items: center; gap: 8px; }
.toolbar-right { margin-left: auto; }

/* ===== 批量操作栏 ===== */
.batch-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 6px 12px;
  background: rgba(var(--theme-color-rgb, 59, 130, 246), var(--fe-a-batch-bg));
  border: 1px solid rgba(var(--theme-color-rgb, 59, 130, 246), var(--fe-a-batch-border));
  border-radius: 8px;
  animation: fe-slide-in 0.2s ease;
}

.batch-info {
  font-size: 13px;
  color: var(--theme-color, #3b82f6);
  font-weight: 500;
}

.batch-info-clickable {
  cursor: pointer;
  padding: 2px 4px;
  border-radius: 4px;
  transition: background 0.15s;
}
.batch-info-clickable:hover {
  background: rgba(var(--theme-color-rgb, 59, 130, 246), 0.15);
}

@keyframes fe-slide-in {
  from { opacity: 0; transform: translateY(-8px); }
  to   { opacity: 1; transform: translateY(0); }
}

/* ===== 无权限 ===== */
.permission-denied-result {
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
}

/* ===== 列表容器 ===== */
.file-list {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--fe-bg-card);
  border-radius: 8px;
}

/* ===== 网格视图 ===== */
.file-grid-container {
  flex: 1;
  overflow-y: auto;
}

.file-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 16px;
  padding: 16px 10px;
}

.grid-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 200px;
}

/* ── 卡片 ── */
.grid-card {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding: 20px 16px 16px;
  background: var(--fe-bg-card);
  border-radius: 12px;
  border: 1px solid var(--fe-border-card);
  cursor: pointer;
  transition: all 0.2s ease;
  user-select: none;

  /* 虚拟化渲染：浏览器自动跳过视口外卡片的布局/绘制 */
  content-visibility: auto;
  contain-intrinsic-size: auto 180px;
}

.grid-card:hover {
  transform: translateY(-2px);
  border-color: rgba(var(--theme-color-rgb, 59, 130, 246), var(--fe-a-card-hover-border));
  box-shadow: 0 4px 18px rgba(var(--theme-color-rgb, 59, 130, 246), var(--fe-a-card-shadow));
}

.grid-card-selected {
  background: rgba(var(--theme-color-rgb, 59, 130, 246), var(--fe-a-row-sel));
  border-color: var(--theme-color, #3b82f6);
}
.grid-card-selected:hover {
  box-shadow: 0 4px 18px rgba(var(--theme-color-rgb, 59, 130, 246), calc(var(--fe-a-card-shadow) * 1.6));
}

/* 复选框 */
.grid-card-checkbox {
  position: absolute;
  top: 8px;
  left: 8px;
  opacity: 0;
  transition: opacity 0.2s;
}
.grid-card:hover .grid-card-checkbox,
.grid-card-selected .grid-card-checkbox { opacity: 1; }

/* 操作按钮组 */
.grid-card-actions {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  width: 100%;
  margin-top: auto;
  padding-top: 8px;
  opacity: 0;
  transition: opacity 0.2s;
}
.grid-card:hover .grid-card-actions { opacity: 1; }

.card-action-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  background: var(--fe-bg-action);
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
  color: var(--fe-fg-muted);
}
.card-action-btn:hover {
  background: rgba(var(--theme-color-rgb, 59, 130, 246), var(--fe-a-action-hover));
  color: var(--theme-color, #3b82f6);
}

/* 图标背景 */
.grid-card-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 64px;
  height: 64px;
  border-radius: 12px;
  margin-bottom: 12px;
}
.icon-folder,
.icon-file   { background: rgba(var(--theme-color-rgb, 59, 130, 246), var(--fe-a-file-icon)); }

.grid-card-name {
  font-size: 13px;
  font-weight: 700;
  color: var(--fe-fg-primary);
  width: 100%;
  word-break: break-word;
  overflow-wrap: anywhere;
  line-height: 1.4;
  margin-bottom: 4px;
}

.grid-card-info {
  font-size: 11px;
  color: var(--fe-fg-muted);
}

/* ===== 数据表格 ===== */
.file-data-table { flex: 1; }

/* 行高 */
.file-data-table :deep(.n-data-table-td),
.file-data-table :deep(.n-data-table-th) {
  padding-top: 8px !important;
  padding-bottom: 8px !important;
  font-size: 13px;
}
.file-data-table :deep(.n-data-table-th) { font-weight: 600; }

/* 行悬停 */
.file-data-table :deep(.n-data-table-tr:hover > .n-data-table-td) {
  background-color: rgba(var(--theme-color-rgb, 59, 130, 246), var(--fe-a-row-hover)) !important;
}

/* 选中行 */
.file-data-table :deep(.checked-row > .n-data-table-td) {
  background-color: rgba(var(--theme-color-rgb, 59, 130, 246), var(--fe-a-row-sel)) !important;
}

/* 高亮行（绿色闪动 - 语义色，不跟随主题） */
.file-data-table :deep(.highlighted-row > .n-data-table-td) {
  background-color: rgba(24, 160, 88, 0.08) !important;
  transition: background-color 0.3s ease;
}

/* 名称列换行 */
.file-data-table :deep(.col-name),
.file-data-table :deep(.col-name .n-data-table-td__ellipsis) {
  white-space: normal !important;
}

.file-data-table :deep(.name-cell) {
  display: flex;
  align-items: center;
  min-width: 0;
}

.file-data-table :deep(.name-text) {
  font-weight: 700;
  white-space: normal;
  word-break: break-all;
  overflow-wrap: anywhere;
  line-height: 1.5;
}

.file-data-table :deep(.col-name .n-button__content) {
  font-weight: 700;
  white-space: normal;
  word-break: break-all;
  overflow-wrap: anywhere;
  text-align: left;
  line-height: 1.5;
}

/* 操作列 */
.file-data-table :deep(.col-actions) { white-space: nowrap; }

.file-data-table :deep(.col-actions .action-btn) {
  opacity: 0;
  transition: opacity 0.2s ease;
}
.file-data-table :deep(.n-data-table-tr:hover .col-actions .action-btn),
.file-data-table :deep(.checked-row .col-actions .action-btn) { opacity: 1; }

/* 暗色表格背景 */
.dark .file-data-table :deep(.n-data-table),
.dark .file-data-table :deep(.n-data-table-wrapper) {
  background-color: var(--fe-bg-card);
}
.dark .file-data-table :deep(.n-data-table-th) {
  background-color: var(--fe-bg-card) !important;
}
.dark .file-data-table :deep(.n-data-table-td) {
  background-color: transparent;
}
.dark .file-data-table :deep(.n-data-table-base-table) {
  border-color: var(--fe-border-card);
}

/* ===== 分享弹窗 ===== */
.share-file-list {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
}

/* ===== 移动端响应式 ===== */
@media (max-width: 767px) {
  .file-data-table :deep(.n-data-table-td),
  .file-data-table :deep(.n-data-table-th) {
    padding-top: 6px !important;
    padding-bottom: 6px !important;
  }

  /* 触屏无 hover，操作按钮始终可见 */
  .file-data-table :deep(.col-actions .action-btn) {
    opacity: 1;
  }
}
</style>
