<template>
  <n-card
    title="分享管理"
    class="shares-card"
    :bordered="false"
    content-style="padding: 12px 16px; display: flex; flex-direction: column; min-height: 0;"
  >
      <div class="shares-summary">
        当前共有 {{ totalCount }} 个文件分享，{{ validCount }} 个有效，{{ expiredCount }} 个已过期
      </div>
      <n-space align="center" style="margin-bottom: 12px;">
        <n-input v-model:value="filterFileName" placeholder="按文件名/Token筛选" clearable size="small" style="width: 200px;" />
        <n-select v-model:value="filterOwnerId" :options="ownerOptions" placeholder="按分享者筛选" clearable size="small" style="width: 160px;" />
        <n-select v-model:value="filterStatus" :options="statusOptions" placeholder="按状态筛选" clearable size="small" style="width: 130px;" />
      </n-space>
      <div class="shares-table-wrapper">
        <n-data-table
          class="shares-data-table"
          size="small"
          flex-height
          :columns="columns"
          :data="filteredShares"
          :bordered="false"
          striped
          :loading="loading"
          :row-key="(row: any) => row.id"
          :row-class-name="rowClassName"
          style="height: 100%;"
        />
      </div>

    </n-card>

    <n-modal v-model:show="showFilesModal" preset="card" title="文件列表" style="width: 600px; max-width: 90vw;">
      <n-spin :show="filesModalLoading">
        <div v-if="modalFiles.length > 0" class="files-modal-list">
          <div v-for="file in modalFiles" :key="file.file_name" class="file-item">
            <span class="file-name-link" @click="navigateToFile(file.file_path)">{{ file.file_name }}</span>
            <span class="file-download-count">下载 {{ file.download_count }} 次</span>
          </div>
        </div>
        <n-empty v-else description="暂无文件" />
      </n-spin>
    </n-modal>

    <!-- 编辑分享弹窗 -->
    <n-modal v-model:show="showEditModal" preset="dialog" title="编辑分享" positive-text="保存" negative-text="取消" :positive-button-props="{ loading: editLoading }" @positive-click="handleEditSave" @negative-click="showEditModal = false" :mask-closable="false">
      <n-form label-placement="left" label-width="80">
        <n-form-item label="分享名称">
          <n-input v-model:value="editName" placeholder="分享名称" clearable :maxlength="100" />
        </n-form-item>
        <n-form-item label="有效期(天)">
          <n-input-number v-model:value="editExpireDays" :min="0" :max="365" placeholder="0 表示永久有效" style="width: 100%" />
        </n-form-item>
      </n-form>
    </n-modal>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, h } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { NCard, NSpace, NButton, NDataTable, NTooltip, NTag, NInput, NSelect, NPopconfirm, NModal, NSpin, NEmpty, NIcon, NForm, NFormItem, NInputNumber, useMessage } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { CopyOutline, TrashOutline, CreateOutline } from '@vicons/ionicons5'
import api from '@/api'
import { copyToClipboard } from '@/utils/clipboard'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const route = useRoute()
const message = useMessage()
const shares = ref<any[]>([])
const loading = ref(false)
const highlightId = ref<number | null>(null)
let highlightTimer: ReturnType<typeof setTimeout> | null = null

// 筛选状态
const filterFileName = ref('')
const filterOwnerId = ref<number | null>(null)
const filterStatus = ref<string | null>(null)

const statusOptions = [
  { label: '有效', value: 'valid' },
  { label: '已过期', value: 'expired' },
  { label: '无效', value: 'deleted' },
]

// 分享者数据
const shareUsers = ref<{id: number, username: string}[]>([])
const ownerMap = computed(() => new Map(shareUsers.value.map(u => [u.id, u.username])))
const ownerOptions = computed(() => shareUsers.value.map(u => ({ label: u.username, value: u.id })))

// 统计（基于全量数据）
const totalCount = computed(() => shares.value.length)
const validCount = computed(() => shares.value.filter((s: any) => s.status === 'valid').length)
const expiredCount = computed(() => shares.value.filter((s: any) => s.status === 'expired').length)

// 文件详情 modal 状态
const showFilesModal = ref(false)
const filesModalLoading = ref(false)
const modalFiles = ref<Array<{file_name: string, file_path: string, file_size: number, download_count: number}>>([])

// 编辑 modal 状态
const showEditModal = ref(false)
const editLoading = ref(false)
const editId = ref<number>(0)
const editName = ref('')
const editExpireDays = ref<number>(7)

// 筛选过滤（同时匹配 file_name 和 token）
const filteredShares = computed(() => {
  return shares.value.filter(s => {
    if (filterFileName.value) {
      const keyword = filterFileName.value.toLowerCase()
      if (!s.file_name.toLowerCase().includes(keyword) && !(s.token && s.token.toLowerCase().includes(keyword))) return false
    }
    if (filterOwnerId.value && s.owner_id !== filterOwnerId.value) return false
    if (filterStatus.value && s.status !== filterStatus.value) return false
    return true
  })
})

function rowClassName(row: any) {
  return row.id === highlightId.value ? 'highlighted-row' : ''
}

function applyHighlight() {
  const hid = route.query.highlightId
  if (hid) {
    highlightId.value = Number(hid)
    if (highlightTimer) clearTimeout(highlightTimer)
    highlightTimer = setTimeout(() => {
      highlightId.value = null
    }, 4000)
  }
}

function navigateToFile(filePath: string) {
  const dirPath = filePath.substring(0, filePath.lastIndexOf('/')) || '/'
  const fileName = filePath.split('/').pop()
  router.push({ path: '/', query: { path: dirPath, highlight: fileName } })
}

function getStatusTooltip(row: any): string {
  if (row.status === 'deleted') {
    return row.expire_at ? `到期时间: ${row.expire_at}\n源文件已删除` : '到期时间: 永久\n源文件已删除'
  }
  if (!row.expire_at) return '永久有效'
  const remaining = Math.ceil((new Date(row.expire_at).getTime() - Date.now()) / 86400000)
  if (remaining > 0) {
    return `到期时间: ${row.expire_at}\n剩余: ${remaining}天`
  }
  return `到期时间: ${row.expire_at}\n已过期`
}

async function openFilesModal(row: any) {
  showFilesModal.value = true
  filesModalLoading.value = true
  try {
    const res = await api.get(`/share/${row.token}/info`)
    modalFiles.value = res.data.files || []
  } catch (err: any) {
    message.error('获取文件列表失败')
    modalFiles.value = []
  } finally {
    filesModalLoading.value = false
  }
}

const columns: DataTableColumns = [
  {
    title: '分享ID',
    key: 'token',
    className: 'col-token',
    width: 300,
    render: (row: any) => {
      // const shortToken = row.token ? row.token.substring(0, 8) + '...' : ''
      // return h(NTooltip, { trigger: 'hover' }, {
      //   trigger: () => h('span', {}, shortToken),
      //   default: () => row.token,
      // })
      return h('span', {}, row.token)
    },
  },
  {
    title: '分享文件',
    key: 'file_name',
    className: 'col-files',
    render: (row: any) => {
      const isMulti = row.file_count && row.file_count > 1
      if (isMulti) {
        return h('span', {
          class: 'file-link',
          onClick: () => openFilesModal(row),
        }, `分享${row.file_count}个文件`)
      }
      return h('span', {
        class: 'file-link',
        onClick: () => navigateToFile(row.file_path),
      }, row.file_name)
    },
  },
  {
    title: '分享者',
    key: 'owner_id',
    width: 120,
    className: 'col-owner',
    render(row: any) {
      return ownerMap.value.get(row.owner_id) || '未知用户'
    },
  },
  {
    title: '分享状态',
    key: 'status',
    width: 100,
    className: 'col-status',
    render(row: any) {
      let tagType: 'success' | 'error' | 'warning' = 'success'
      let label = '有效'
      if (row.status === 'expired') {
        tagType = 'error'
        label = '已过期'
      } else if (row.status === 'deleted') {
        tagType = 'warning'
        label = '无效'
      }
      return h(NTooltip, { trigger: 'hover', style: 'white-space: pre-line' }, {
        trigger: () => h(NTag, { type: tagType, size: 'small' }, () => label),
        default: () => getStatusTooltip(row),
      })
    },
  },
  {
    title: '访问次数',
    key: 'access_count',
    width: 90,
    className: 'col-count',
  },
  {
    title: '创建时间',
    key: 'created_at',
    width: 170,
    className: 'col-time',
  },
  {
    title: '操作',
    key: 'actions',
    width: 120,
    className: 'col-actions',
    render: (row: any) =>
      h(NSpace, { size: 2, wrap: false }, () => [
        h(NTooltip, { trigger: 'hover', placement: 'top' }, {
          default: () => '复制链接',
          trigger: () => h(NButton, { size: 'small', quaternary: true, class: 'action-btn', onClick: () => copyLink(row) }, {
            icon: () => h(NIcon, { size: 18, color: '#1890ff' }, () => h(CopyOutline)),
          }),
        }),
        h(NTooltip, { trigger: 'hover', placement: 'top' }, {
          default: () => '编辑',
          trigger: () => h(NButton, { size: 'small', quaternary: true, class: 'action-btn', onClick: () => openEditModal(row) }, {
            icon: () => h(NIcon, { size: 18, color: '#faad14' }, () => h(CreateOutline)),
          }),
        }),
        h(NPopconfirm, {
          onPositiveClick: () => handleDelete(row),
        }, {
          trigger: () => h(NTooltip, { trigger: 'hover', placement: 'top' }, {
            default: () => '删除',
            trigger: () => h(NButton, { size: 'small', quaternary: true, class: 'action-btn' }, {
              icon: () => h(NIcon, { size: 18, color: '#d03050' }, () => h(TrashOutline)),
            }),
          }),
          default: () => '确认删除此分享链接？',
        }),
      ]),
  },
]

onMounted(() => {
  fetchShares()
  fetchShareUsers()
})

async function fetchShares() {
  loading.value = true
  try {
    const res = await api.get('/api/admin/shares')
    shares.value = res.data
    applyHighlight()
  } catch (err: any) {
    message.error(err.response?.data?.error || '获取分享列表失败')
  } finally {
    loading.value = false
  }
}

async function fetchShareUsers() {
  try {
    const res = await api.get('/api/admin/share-users')
    shareUsers.value = res.data
  } catch (err: any) {
    message.error(err.response?.data?.error || '获取分享者列表失败')
  }
}

async function copyLink(row: any) {
  const link = `${window.location.origin}/share/${row.token}`
  const ok = await copyToClipboard(link)
  if (ok) {
    message.success('链接已复制')
  } else {
    message.error('复制失败，请手动复制')
  }
}

async function handleDelete(row: any) {
  try {
    await api.delete(`/api/shares/${row.id}`)
    message.success('分享链接已删除')
    const userStore = useUserStore()
    userStore.onShareDeleted(row.status === 'expired' ? 'expired' : 'valid')
    fetchShares()
  } catch (err: any) {
    message.error(err.response?.data?.error || '删除失败')
  }
}

function openEditModal(row: any) {
  editId.value = row.id
  editName.value = row.name || row.file_name || ''
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
    fetchShares()
  } catch (err: any) {
    message.error(err.response?.data?.error || '更新失败')
  } finally {
    editLoading.value = false
  }
}

watch(() => route.query.highlightId, () => {
  applyHighlight()
})

onUnmounted(() => {
  if (highlightTimer) clearTimeout(highlightTimer)
})
</script>

<style scoped>
.shares-card {
  height: calc(100vh - 135px);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.shares-card :deep(.n-card__content) {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.shares-table-wrapper {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

/* 分享ID列 */
.shares-data-table :deep(.col-token) {
  font-family: monospace;
  font-size: 12px;
}

/* 分享文件列自适应换行 */
.shares-data-table :deep(.col-files .n-data-table-td__ellipsis),
.shares-data-table :deep(.col-files) {
  white-space: normal !important;
  max-width: 50%;
}

.shares-data-table :deep(.col-files .file-link) {
  color: var(--primary-color);
  cursor: pointer;
  word-break: break-all;
  overflow-wrap: anywhere;
  line-height: 1.5;
}

.shares-data-table :deep(.col-files .file-link:hover) {
  text-decoration: underline;
}

/* 其他列保持单行 */
.shares-data-table :deep(.col-status),
.shares-data-table :deep(.col-owner),
.shares-data-table :deep(.col-time),
.shares-data-table :deep(.col-actions) {
  white-space: nowrap;
}

/* 行高紧凑化 */
.shares-data-table :deep(.n-data-table-td),
.shares-data-table :deep(.n-data-table-th) {
  padding-top: 6px !important;
  padding-bottom: 6px !important;
  font-size: 13px;
}
.shares-data-table :deep(.n-data-table-th) {
  font-weight: 600;
}

.shares-data-table :deep(.highlighted-row td) {
  background-color: rgba(24, 160, 88, 0.08) !important;
  transition: background-color 0.3s ease;
}

.shares-summary {
  text-align: left;
  font-size: 14px;
  color: #666;
  padding: 8px 0;
}

/* 文件列表 modal */
.files-modal-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.files-modal-list .file-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  border-radius: 6px;
  background: #f9f9f9;
}

.dark .files-modal-list .file-item {
  background: rgb(70, 70, 80);
}

.files-modal-list .file-name-link {
  color: var(--primary-color);
  cursor: pointer;
  word-break: break-all;
}

.files-modal-list .file-name-link:hover {
  text-decoration: underline;
}

.files-modal-list .file-download-count {
  white-space: nowrap;
  color: #999;
  font-size: 12px;
  margin-left: 16px;
}
</style>
