<template>
  <n-card title="操作日志" style="display: flex; flex-direction: column; height: 100%">
      <n-space :size="12" style="margin-bottom: 12px">
        <n-date-picker v-model:value="dateRange" type="daterange" clearable />
        <n-input v-model:value="filterUser" placeholder="用户ID" clearable style="width: 100px" />
        <n-select v-model:value="filterAction" :options="actionOptions" clearable placeholder="操作类型" style="width: 160px" />
        <n-input v-model:value="filterPath" placeholder="目标路径" clearable style="width: 200px" />
        <n-button @click="fetchLogs">筛选</n-button>
      </n-space>

      <n-data-table
        :columns="columns"
        :data="logs"
        :bordered="false"
        striped
        :loading="loading"
        size="small"
        :scroll-x="900"
        :max-height="tableHeight"
        :flex-height="false"
      />
      <n-space justify="center" style="margin-top: 12px">
        <n-pagination v-model:page="page" :page-count="totalPages" @update:page="fetchLogs" />
      </n-space>
    </n-card>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, h } from 'vue'
import {
  NCard, NSpace, NButton, NDataTable, NInput, NSelect, NDatePicker, NPagination, useMessage
} from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import api from '@/api'

const message = useMessage()
const logs = ref<any[]>([])
const loading = ref(false)
const page = ref(1)
const pageSize = 50
const total = ref(0)
const totalPages = computed(() => Math.ceil(total.value / pageSize) || 1)

const dateRange = ref<[number, number] | null>(null)
const filterUser = ref('')
const filterAction = ref<string | null>(null)
const filterPath = ref('')

// ---------- 表格自适应高度 ----------
// 布局偏移：header(56) + footer(44) + content padding(48) + card header+padding(78) + 筛选栏+分页(94)
const LAYOUT_OFFSET = 320
const windowHeight = ref(window.innerHeight)
const tableHeight = computed(() => Math.max(200, windowHeight.value - LAYOUT_OFFSET))

function onResize() {
  windowHeight.value = window.innerHeight
}

// ---------- 操作类型映射 ----------
const actionOptions = [
  { label: '登录', value: 'LOGIN' },
  { label: '登录失败', value: 'LOGIN_FAIL' },
  { label: '创建目录', value: 'CREATE_DIR' },
  { label: '上传文件', value: 'UPLOAD' },
  { label: '下载文件', value: 'DOWNLOAD' },
  { label: '删除文件', value: 'DELETE_FILE' },
  { label: '删除目录', value: 'DELETE_DIR' },
  { label: '创建分享', value: 'SHARE_CREATE' },
  { label: '访问分享', value: 'SHARE_ACCESS' },
  { label: '删除分享', value: 'SHARE_DELETE' },
  { label: '变更所有者', value: 'CHANGE_OWNER' },
  { label: '创建用户', value: 'USER_CREATE' },
  { label: '更新用户', value: 'USER_UPDATE' },
  { label: '删除用户', value: 'USER_DELETE' },
  { label: '移动/重命名', value: 'MOVE' },
]

const actionLabelMap = new Map(actionOptions.map(o => [o.value, o.label]))

// ---------- 时间格式化 ----------
function formatTime(iso: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (isNaN(d.getTime())) return iso
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

// ---------- 表格列定义 ----------
const columns: DataTableColumns = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '用户', key: 'username', width: 100 },
  {
    title: '操作类型',
    key: 'action',
    width: 110,
    render: (row: any) => actionLabelMap.get(row.action) ?? row.action,
  },
  { title: '目标路径', key: 'target_path', ellipsis: { tooltip: true } },
  { title: 'IP地址', key: 'ip_address', width: 130 },
  { title: '详情', key: 'details', ellipsis: { tooltip: true } },
  {
    title: '时间',
    key: 'created_at',
    width: 165,
    render: (row: any) => formatTime(row.created_at),
  },
]

onMounted(() => {
  window.addEventListener('resize', onResize)
  fetchLogs()
})

onUnmounted(() => {
  window.removeEventListener('resize', onResize)
})

async function fetchLogs() {
  loading.value = true
  try {
    const params: any = { page: page.value, page_size: pageSize }
    if (dateRange.value) {
      params.start_time = new Date(dateRange.value[0]).toISOString()
      params.end_time = new Date(dateRange.value[1]).toISOString()
    }
    if (filterUser.value) params.user_id = filterUser.value
    if (filterAction.value) params.action = filterAction.value
    if (filterPath.value) params.target_path = filterPath.value

    const res = await api.get('/api/logs', { params })
    logs.value = res.data.logs
    total.value = res.data.total
  } catch (err: any) {
    message.error(err.response?.data?.error || '获取日志失败')
  } finally {
    loading.value = false
  }
}
</script>