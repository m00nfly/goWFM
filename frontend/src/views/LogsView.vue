<template>
  <div class="workspace-page logs-page">
    <section class="workspace-surface">
      <header class="workspace-header">
        <div class="workspace-title-block">
          <h1 class="workspace-title">操作日志</h1>
          <p class="workspace-subtitle">检索系统访问、文件操作与配置变更记录</p>
        </div>
        <div class="workspace-count-pill">
          <strong>{{ total }}</strong>
          条记录
        </div>
      </header>

      <div class="workspace-toolbar logs-toolbar">
        <div class="workspace-toolbar-group">
          <n-date-picker
            v-model:value="dateRange"
            class="logs-date-picker"
            type="datetimerange"
            clearable
            :default-time="['00:00:01', '23:59:59']"
          />
          <n-select
            v-model:value="filterUsername"
            class="logs-user-select"
            :options="userOptions"
            clearable
            filterable
            placeholder="用户名"
          />
          <n-select
            v-model:value="filterAction"
            class="logs-action-select"
            :options="actionOptions"
            clearable
            placeholder="操作类型"
          />
          <n-input v-model:value="filterPath" class="logs-path-input" placeholder="目标对象" clearable />
        </div>
        <div class="workspace-actions">
          <n-button type="primary" secondary @click="fetchLogs">搜索</n-button>
        </div>
      </div>

      <div class="workspace-table-shell logs-table-wrapper">
        <n-data-table
          class="workspace-data-table logs-data-table"
          :columns="columns"
          :data="logs"
          :bordered="false"
          striped
          :loading="loading"
          size="small"
          :scroll-x="900"
          flex-height
          style="height: 100%;"
        />
      </div>
      <div class="workspace-pagination">
        <n-pagination v-model:page="page" :page-count="totalPages" @update:page="fetchLogs" />
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  NButton, NDataTable, NInput, NSelect, NDatePicker, NPagination, useMessage
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
const filterUsername = ref<string | null>(null)
const userOptions = ref<{ label: string; value: string }[]>([])
const filterAction = ref<string | null>(null)
const filterPath = ref('')

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
  { label: '变更设定', value: 'CONFIG_CHANGE' },
	{ label: '申请重置密码', value: 'PASSWORD_RESET_REQUEST' },
	{ label: '完成重置密码', value: 'PASSWORD_RESET_COMPLETE' },
]

// ---------- 配置类型标签映射 ----------
const configTypeLabel = [
  { label: '基础设置', value: 'basic' },
  { label: '外观设置', value: 'appearance' },
  { label: '安全设置', value: 'security' },
  { label: '日志设置', value: 'log' },
  { label: '邮件设置', value: 'email' },
  { label: '分享设置', value: 'share' },
]

const actionLabelMap = new Map(actionOptions.map(o => [o.value, o.label]))
const configTypeLabelMap = new Map(configTypeLabel.map(o => [o.value, o.label]))

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
  {
    title: '时间',
    key: 'created_at',
    width: 165,
    render: (row: any) => formatTime(row.created_at),
  },
  { title: '用户', key: 'username', width: 100 },
  {
    title: '操作类型',
    key: 'action',
    width: 110,
    render: (row: any) => actionLabelMap.get(row.action) ?? row.action,
  },
  {
    title: '目标对象',
    key: 'target_path',
    ellipsis: { tooltip: true },
    render: (row: any) => configTypeLabelMap.get(row.target_path) ?? row.target_path,
  },
  { title: 'IP地址', key: 'ip_address', width: 130 },
  { title: '详情', key: 'details', ellipsis: { tooltip: true } },
]

onMounted(async () => {
  try {
    const res = await api.get('/api/logs/users')
    userOptions.value = (res.data as any[]).map((u: any) => ({
      label: u.username,
      value: u.username,
    }))
  } catch { /* 无权限或失败则不展示下拉 */ }
  fetchLogs()
})

async function fetchLogs() {
  loading.value = true
  try {
    const params: any = { page: page.value, page_size: pageSize }
    if (dateRange.value) {
      params.start_time = new Date(dateRange.value[0]).toISOString()
      params.end_time = new Date(dateRange.value[1]).toISOString()
    }
    if (filterUsername.value) params.username = filterUsername.value
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

<style scoped>
.logs-date-picker {
  width: 310px;
}

.logs-user-select {
  width: 140px;
}

.logs-action-select {
  width: 160px;
}

.logs-path-input {
  width: 200px;
}

.logs-data-table :deep(.n-data-table-td) {
  font-variant-numeric: tabular-nums;
}

@media (max-width: 768px) {
  .logs-date-picker,
  .logs-user-select,
  .logs-action-select,
  .logs-path-input {
    width: 100%;
  }
}
</style>
