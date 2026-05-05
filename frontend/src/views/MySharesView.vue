<template>
  <n-card
    class="shares-card"
    :bordered="false"
    content-style="padding: 12px 16px; display: flex; flex-direction: column; height: 100%;"
  >
      <n-space justify="end" :size="12" style="margin-bottom: 12px">
        <n-button type="primary" @click="showCreateModal = true">创建分享</n-button>
      </n-space>
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
          style="height: 100%;"
        />
      </div>

      <n-modal v-model:show="showCreateModal" title="创建分享链接" preset="dialog">
        <n-form label-placement="left" label-width="80">
          <n-form-item label="文件路径">
            <n-input v-model:value="shareFilePath" placeholder="相对路径" />
          </n-form-item>
          <n-form-item label="有效期">
            <n-input-number v-model:value="expireDays" :min="0" placeholder="0为永久" />
            <span style="margin-left: 8px; color: #999">天（0表示永久）</span>
          </n-form-item>
        </n-form>
        <template #action>
          <n-button @click="showCreateModal = false">取消</n-button>
          <n-button type="primary" :loading="createLoading" @click="handleCreateShare">创建</n-button>
        </template>
      </n-modal>
    </n-card>
</template>

<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NCard, NSpace, NButton, NDataTable, NModal, NForm, NFormItem, NInput, NInputNumber, NTooltip, NTag, useMessage } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import api from '@/api'

const route = useRoute()
const router = useRouter()
const message = useMessage()
const shares = ref<any[]>([])
const loading = ref(false)
const showCreateModal = ref(false)
const createLoading = ref(false)
const shareFilePath = ref('')
const expireDays = ref(3)

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
        h(NButton, { size: 'small', type: 'error', onClick: () => handleDelete(row) }, () => '删除'),
      ]),
  },
]

onMounted(() => {
  if (route.query.shareFile) {
    shareFilePath.value = route.query.shareFile as string
    showCreateModal.value = true
  }
  fetchShares()
})

async function fetchShares() {
  loading.value = true
  try {
    const res = await api.get('/api/shares/my')
    shares.value = res.data
  } catch (err: any) {
    message.error(err.response?.data?.error || '获取分享列表失败')
  } finally {
    loading.value = false
  }
}

async function handleCreateShare() {
  if (!shareFilePath.value) { message.warning('请输入文件路径'); return }
  createLoading.value = true
  try {
    const res = await api.post('/api/shares', {
      file_path: shareFilePath.value,
      expire_days: expireDays.value,
    })
    message.success('分享链接创建成功')
    showCreateModal.value = false
    shareFilePath.value = ''
    fetchShares()
  } catch (err: any) {
    message.error(err.response?.data?.error || '创建失败')
  } finally {
    createLoading.value = false
  }
}

function copyLink(row: any) {
  const link = `${window.location.origin}/share/${row.token}`
  navigator.clipboard.writeText(link)
  message.success('链接已复制')
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
</style>