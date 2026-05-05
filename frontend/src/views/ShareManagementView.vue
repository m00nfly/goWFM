<template>
  <n-card
    class="shares-card"
    :bordered="false"
    content-style="padding: 12px 16px; display: flex; flex-direction: column; height: 100%;"
  >
      <div class="shares-summary">
        当前共有 {{ totalCount }} 个文件分享，{{ validCount }} 个有效，{{ expiredCount }} 个已过期
      </div>
      <n-space align="center" style="margin-bottom: 12px;">
        <n-input v-model:value="filterFileName" placeholder="按文件名筛选" clearable size="small" style="width: 200px;" />
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
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, h } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { NCard, NSpace, NButton, NDataTable, NTooltip, NTag, NInput, NSelect, NPopconfirm, useMessage } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import api from '@/api'

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
]

// 分享者数据
const shareUsers = ref<{id: number, username: string}[]>([])
const ownerMap = computed(() => new Map(shareUsers.value.map(u => [u.id, u.username])))
const ownerOptions = computed(() => shareUsers.value.map(u => ({ label: u.username, value: u.id })))

// 统计（基于全量数据）
const totalCount = computed(() => shares.value.length)
const validCount = computed(() => shares.value.filter((s: any) => s.status === 'valid').length)
const expiredCount = computed(() => shares.value.filter((s: any) => s.status === 'expired').length)

// 筛选过滤
const filteredShares = computed(() => {
  return shares.value.filter(s => {
    if (filterFileName.value && !s.file_name.toLowerCase().includes(filterFileName.value.toLowerCase())) return false
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
  if (!row.expire_at) return '永久有效'
  const remaining = Math.ceil((new Date(row.expire_at).getTime() - Date.now()) / 86400000)
  if (remaining > 0) {
    return `到期时间: ${row.expire_at}\n剩余: ${remaining}天`
  }
  return `到期时间: ${row.expire_at}\n已过期`
}

const columns: DataTableColumns = [
  {
    title: '文件名',
    key: 'file_name',
    className: 'col-name',
    render: (row: any) =>
      h(NTooltip, { trigger: 'hover' }, {
        trigger: () => h('span', {
          class: 'file-link',
          onClick: () => navigateToFile(row.file_path),
        }, row.file_name),
        default: () => row.file_path,
      }),
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
    render: (row: any) =>
      h(NTooltip, { trigger: 'hover', style: 'white-space: pre-line' }, {
        trigger: () => h(NTag, {
          type: row.status === 'valid' ? 'success' : 'error',
          size: 'small',
        }, () => row.status === 'valid' ? '有效' : '已过期'),
        default: () => getStatusTooltip(row),
      }),
  },
  {
    title: '创建时间',
    key: 'created_at',
    width: 170,
    className: 'col-time',
  },
  {
    title: '下载次数',
    key: 'access_count',
    width: 100,
    className: 'col-count',
  },
  {
    title: '操作',
    key: 'actions',
    width: 150,
    className: 'col-actions',
    render: (row: any) =>
      h(NSpace, { size: 'small' }, () => [
        h(NButton, { size: 'small', onClick: () => copyLink(row) }, () => '复制链接'),
        h(NPopconfirm, {
          onPositiveClick: () => handleDelete(row),
        }, {
          trigger: () => h(NButton, { size: 'small', type: 'error' }, () => '删除'),
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

function copyLink(row: any) {
  const link = `${window.location.origin}/share/${row.token}`
  navigator.clipboard.writeText(link)
  message.success('链接已复制')
}

async function handleDelete(row: any) {
  try {
    await api.delete(`/api/shares/${row.id}`)
    message.success('分享链接已删除')
    fetchShares()
  } catch (err: any) {
    message.error(err.response?.data?.error || '删除失败')
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
  height: calc(100vh - 112px);
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

/* 文件名列自适应换行 */
.shares-data-table :deep(.col-name .n-data-table-td__ellipsis),
.shares-data-table :deep(.col-name) {
  white-space: normal !important;
  max-width: 50%;
}

.shares-data-table :deep(.col-name .file-link) {
  color: var(--primary-color);
  cursor: pointer;
  word-break: break-all;
  overflow-wrap: anywhere;
  line-height: 1.5;
}

.shares-data-table :deep(.col-name .file-link:hover) {
  text-decoration: underline;
}

/* 其他列保持单行 */
.shares-data-table :deep(.col-status),
.shares-data-table :deep(.col-owner),
.shares-data-table :deep(.col-time),
.shares-data-table :deep(.col-count),
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
</style>
