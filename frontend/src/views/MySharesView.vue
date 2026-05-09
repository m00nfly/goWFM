<template>
  <n-card
    class="shares-card"
    :bordered="false"
    content-style="padding: 12px 16px; display: flex; flex-direction: column; height: 100%;"
  >
      <div class="shares-summary">
        当前有 {{ totalCount }} 个文件分享，{{ validCount }} 个有效，{{ expiredCount }} 个已过期
      </div>
      <div class="shares-table-wrapper">
        <n-data-table
          class="shares-data-table"
          size="small"
          flex-height
          :columns="columns"
          :data="shares"
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
import { NCard, NSpace, NButton, NDataTable, NTooltip, NTag, useMessage } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import api from '@/api'
import { copyToClipboard } from '@/utils/clipboard'

const router = useRouter()
const route = useRoute()
const message = useMessage()
const shares = ref<any[]>([])
const loading = ref(false)
const highlightId = ref<number | null>(null)
let highlightTimer: ReturnType<typeof setTimeout> | null = null
const totalCount = computed(() => shares.value.length)
const validCount = computed(() => shares.value.filter((s: any) => s.status === 'valid').length)
const expiredCount = computed(() => shares.value.filter((s: any) => s.status === 'expired').length)

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
        h(NButton, { size: 'small', type: 'error', onClick: () => handleDelete(row) }, () => '删除'),
      ]),
  },
]

onMounted(() => {
  fetchShares()
})

async function fetchShares() {
  loading.value = true
  try {
    const res = await api.get('/api/shares/my')
    shares.value = res.data
    applyHighlight()
  } catch (err: any) {
    message.error(err.response?.data?.error || '获取分享列表失败')
  } finally {
    loading.value = false
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
  if (!confirm('确认删除此分享链接？')) return
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