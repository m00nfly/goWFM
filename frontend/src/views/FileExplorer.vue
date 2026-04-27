<template>
  <MainLayout>
    <n-card>
      <template #header>
        <n-space justify="space-between" align="center">
          <n-breadcrumb>
            <n-breadcrumb-item @click="navigateTo('/')">根目录</n-breadcrumb-item>
            <n-breadcrumb-item v-for="seg in pathSegments" :key="seg.path" @click="navigateTo(seg.path)">
              {{ seg.name }}
            </n-breadcrumb-item>
          </n-breadcrumb>
          <n-button size="small" @click="refresh">刷新</n-button>
        </n-space>
      </template>

      <n-space :size="12" style="margin-bottom: 16px">
        <n-button v-if="hasPermUpload" type="primary" @click="showUploadModal = true">上传文件</n-button>
        <n-button v-if="hasPermUpload" @click="showMkdirModal = true">新建文件夹</n-button>
      </n-space>

      <n-data-table :columns="columns" :data="entries" :bordered="false" striped :loading="loading" :row-key="(row: any) => row.path" />

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
  </MainLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, h, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import MainLayout from '@/layouts/MainLayout.vue'
import {
  NCard, NSpace, NButton, NDataTable, NModal, NForm, NFormItem, NInput,
  NBreadcrumb, NBreadcrumbItem, NSelect, useMessage
} from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import api from '@/api'
import { useUserStore } from '@/stores/user'
import { formatSize } from '@/utils/format'

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

const columns: DataTableColumns = [
  {
    title: '名称',
    key: 'name',
    sorter: (a: any, b: any) => a.name.localeCompare(b.name),
    render(row: any) {
      return row.is_directory
        ? h(NButton, { text: true, type: 'info', onClick: () => navigateTo(row.path) }, () => row.name)
        : h('span', null, row.name)
    },
  },
  {
    title: '大小',
    key: 'size',
    sorter: (a: any, b: any) => a.size - b.size,
    render(row: any) {
      return row.is_directory ? '-' : formatSize(row.size)
    },
  },
  {
    title: '修改时间',
    key: 'mod_time',
    sorter: (a: any, b: any) => new Date(a.mod_time).getTime() - new Date(b.mod_time).getTime(),
    render(row: any) {
      return new Date(row.mod_time).toLocaleString()
    },
  },
  {
    title: '所有者',
    key: 'owner_name',
    render(row: any) {
      return isAdmin.value
        ? h(NButton, { text: true, type: 'primary', onClick: () => openOwnerModal(row) }, () => row.owner_name)
        : row.owner_name as string
    },
  },
  {
    title: '操作',
    key: 'actions',
    width: 240,
    render(row: any) {
      const btns: any[] = []
      if (row.is_directory) {
        btns.push(h(NButton, { size: 'small', onClick: () => navigateTo(row.path) }, () => '进入'))
      } else {
        if (row.can_download) btns.push(h(NButton, { size: 'small', onClick: () => downloadFile(row) }, () => '下载'))
        if (hasPermShare.value) btns.push(h(NButton, { size: 'small', onClick: () => shareFile(row) }, () => '分享'))
      }
      if (row.can_delete) btns.push(h(NButton, { size: 'small', type: 'error', onClick: () => deleteEntry(row) }, () => '删除'))
      btns.push(h(NButton, { size: 'small', onClick: () => openMoveModal(row) }, () => '移动'))
      return h(NSpace, { size: 4 }, () => btns)
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

function refresh() { fetchFiles() }

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
    message.success('移动成功')
    showMoveModal.value = false
    await fetchFiles()
  } catch (err: any) {
    message.error(err.response?.data?.error || '移动失败')
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