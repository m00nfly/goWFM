<template>
  <MainLayout>
    <n-card title="我的分享">
      <n-space justify="end" :size="12" style="margin-bottom: 16px">
        <n-button type="primary" @click="showCreateModal = true">创建分享</n-button>
      </n-space>
      <n-data-table :columns="columns" :data="shares" :bordered="false" striped :loading="loading" />

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
  </MainLayout>
</template>

<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import { useRoute } from 'vue-router'
import MainLayout from '@/layouts/MainLayout.vue'
import { NCard, NSpace, NButton, NDataTable, NModal, NForm, NFormItem, NInput, NInputNumber, useMessage } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import api from '@/api'

const route = useRoute()
const message = useMessage()
const shares = ref<any[]>([])
const loading = ref(false)
const showCreateModal = ref(false)
const createLoading = ref(false)
const shareFilePath = ref('')
const expireDays = ref(3)

const columns: DataTableColumns = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '文件路径', key: 'file_path' },
  { title: '链接', key: 'token', render: (row: any) => `/share/${row.token}` },
  { title: '过期时间', key: 'expire_at', render: (row: any) => row.expire_at || '永久' },
  { title: '创建时间', key: 'created_at' },
  { title: '访问次数', key: 'access_count' },
  {
    title: '操作',
    key: 'actions',
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