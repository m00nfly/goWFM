<template>
  <div class="workspace-page shares-page">
    <section class="workspace-surface">
      <header class="workspace-header">
        <div class="workspace-title-block">
          <h1 class="workspace-title">我的分享</h1>
          <p class="workspace-subtitle">查看、复制和维护由您创建的文件分享</p>
        </div>
        <div class="workspace-stat-grid shares-stats">
          <div class="workspace-stat">
            <div class="workspace-stat-label">全部分享</div>
            <div class="workspace-stat-value">{{ totalCount }}</div>
          </div>
          <div class="workspace-stat">
            <div class="workspace-stat-label">有效</div>
            <div class="workspace-stat-value">{{ validCount }}</div>
          </div>
          <div class="workspace-stat">
            <div class="workspace-stat-label">已过期</div>
            <div class="workspace-stat-value">{{ expiredCount }}</div>
          </div>
        </div>
      </header>

      <div class="workspace-table-shell shares-table-wrapper">
        <n-data-table
          class="workspace-data-table shares-data-table"
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
    </section>
  </div>

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
import { NSpace, NButton, NDataTable, NTooltip, NTag, NPopconfirm, NModal, NSpin, NEmpty, NIcon, NForm, NFormItem, NInput, NInputNumber, useMessage } from 'naive-ui'
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
.shares-stats {
  width: min(420px, 100%);
}

/* 分享ID列 */
.shares-data-table :deep(.col-token) {
  color: var(--workspace-text-muted);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}

/* 分享文件列自适应换行 */
.shares-data-table :deep(.col-files .n-data-table-td__ellipsis),
.shares-data-table :deep(.col-files) {
  white-space: normal !important;
  max-width: 50%;
}

.shares-data-table :deep(.col-files .file-link) {
  color: var(--workspace-accent);
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
.shares-data-table :deep(.col-time),
.shares-data-table :deep(.col-actions) {
  white-space: nowrap;
}

.shares-data-table :deep(.highlighted-row td) {
  background-color: var(--workspace-row-selected) !important;
  transition: background-color 0.3s ease;
}

/* 文件列表 modal */
.files-modal-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.files-modal-list .file-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding: 9px 11px;
  border: 1px solid var(--workspace-border-soft);
  border-radius: var(--workspace-radius-md);
  background: var(--workspace-surface-soft);
}

.files-modal-list .file-name-link {
  color: var(--workspace-accent);
  cursor: pointer;
  word-break: break-all;
}

.files-modal-list .file-name-link:hover {
  text-decoration: underline;
}

.files-modal-list .file-download-count {
  white-space: nowrap;
  color: var(--workspace-text-muted);
  font-size: 12px;
  margin-left: 16px;
  font-variant-numeric: tabular-nums;
}

@media (max-width: 768px) {
  .shares-stats {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}
</style>
