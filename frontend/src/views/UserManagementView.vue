<template>
  <div class="workspace-page users-page">
    <section class="workspace-surface">
      <header class="workspace-header">
        <div class="workspace-title-block">
          <h1 class="workspace-title">用户管理</h1>
          <p class="workspace-subtitle">维护账号、权限、管理员身份与 TOTP 状态</p>
        </div>
        <div class="workspace-actions">
          <div class="workspace-count-pill">
            <strong>{{ users.length }}</strong>
            个用户
          </div>
          <n-button type="primary" @click="showCreateModal = true">
            <template #icon>
              <n-icon><AddOutline /></n-icon>
            </template>
            创建用户
          </n-button>
        </div>
      </header>

      <div class="workspace-table-shell users-table-wrapper">
        <n-data-table
          class="workspace-data-table users-data-table"
          size="small"
          flex-height
          :columns="columns"
          :data="users"
          :bordered="false"
          striped
          style="height: 100%;"
        />
      </div>
    </section>
  </div>

  <n-modal v-model:show="showCreateModal" title="创建用户" preset="dialog">
    <n-form ref="createFormRef" :model="createForm" :rules="createRules" label-placement="left" label-width="80">
      <n-form-item label="用户名" path="username">
        <n-input v-model:value="createForm.username" />
      </n-form-item>
      <n-form-item label="密码" path="password">
        <n-input v-model:value="createForm.password" type="password" />
      </n-form-item>
      <n-form-item label="显示名称" path="display_name">
        <n-input v-model:value="createForm.display_name" />
      </n-form-item>
      <n-form-item label="邮箱" path="email">
        <n-input v-model:value="createForm.email" />
      </n-form-item>
      <n-form-item label="权限" path="permissions">
        <n-checkbox-group v-model:value="permChecks">
          <n-space>
            <n-checkbox :value="1">浏览</n-checkbox>
            <n-checkbox :value="2">下载</n-checkbox>
            <n-checkbox :value="4">上传</n-checkbox>
            <n-checkbox :value="8">分享</n-checkbox>
            <n-checkbox :value="16">日志</n-checkbox>
          </n-space>
        </n-checkbox-group>
      </n-form-item>
    </n-form>
    <template #action>
      <n-button @click="showCreateModal = false">取消</n-button>
      <n-button type="primary" :loading="createLoading" @click="handleCreate">创建</n-button>
    </template>
  </n-modal>

  <n-modal v-model:show="showEditModal" title="编辑用户" preset="dialog">
    <n-form :model="editForm" label-placement="left" label-width="80">
      <n-form-item label="显示名称">
        <n-input v-model:value="editForm.display_name" />
      </n-form-item>
      <n-form-item label="邮箱">
        <n-input v-model:value="editForm.email" />
      </n-form-item>
      <n-form-item label="管理员">
        <n-switch v-model:value="editForm.is_admin" />
      </n-form-item>
      <n-form-item label="TOTP">
        <n-switch v-model:value="editForm.totp_enabled" />
        <span class="workspace-inline-note">管理员强制启用后，用户需在个人设置中扫码绑定</span>
      </n-form-item>
      <n-form-item label="权限">
        <n-checkbox-group v-model:value="editPermChecks">
          <n-space>
            <n-checkbox :value="1">浏览</n-checkbox>
            <n-checkbox :value="2">下载</n-checkbox>
            <n-checkbox :value="4">上传</n-checkbox>
            <n-checkbox :value="8">分享</n-checkbox>
            <n-checkbox :value="16">日志</n-checkbox>
          </n-space>
        </n-checkbox-group>
      </n-form-item>
    </n-form>
    <template #action>
      <n-button @click="showEditModal = false">取消</n-button>
      <n-button type="primary" :loading="editLoading" @click="handleEdit">保存</n-button>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed, h } from 'vue'
import {
  NSpace, NButton, NDataTable, NModal, NForm, NFormItem, NInput,
  NCheckboxGroup, NCheckbox, NSwitch, NTag, NIcon, NTooltip, useMessage,
} from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { AddOutline, CreateOutline, TrashOutline } from '@vicons/ionicons5'
import api from '@/api'
import { useViewport } from '@/composables/useViewport'

const message = useMessage()
const { isMobile } = useViewport()
const users = ref<any[]>([])
const showCreateModal = ref(false)
const showEditModal = ref(false)
const createLoading = ref(false)
const editLoading = ref(false)

const createFormRef = ref<any>(null)
const createForm = reactive({ username: '', password: '', display_name: '', email: '' })
const permChecks = ref<number[]>([1, 2])
const createRules = {
  username: [{ required: true, message: '必填' }],
  password: [{ required: true, message: '必填' }, { min: 6, message: '至少6位' }],
}

const editForm = reactive({ id: 0, display_name: '', email: '', is_admin: false, totp_enabled: false })
const editPermChecks = ref<number[]>([])
const originalTotpEnabled = ref(false)

const columns = computed<DataTableColumns>(() => {
  const cols: DataTableColumns = [
    { title: 'ID', key: 'id', width: 60 },
    { title: '用户名', key: 'username' },
  ]

  if (!isMobile.value) {
    cols.push(
      { title: '显示名称', key: 'display_name' },
      { title: '邮箱', key: 'email' },
    )
  }

  cols.push(
    { title: '管理员', key: 'is_admin', width: 80, render: (row: any) => row.is_admin ? '是' : '否' },
    {
      title: 'TOTP',
      key: 'totp_enabled',
      width: 90,
      render: (row: any) =>
        h(NTag, { type: row.totp_enabled ? 'success' : 'default', size: 'small' }, () =>
          row.totp_enabled ? '已启用' : '未启用',
        ),
    },
  )

  if (!isMobile.value) {
    cols.push({ title: '权限', key: 'permissions', render: (row: any) => permLabel(row.permissions) })
  }

  cols.push({
    title: '操作',
    key: 'actions',
    width: 96,
    render: (row: any) =>
      row.id === 0
        ? null
        : h(NSpace, { size: 2, wrap: false }, () => [
            iconAction(CreateOutline, '编辑', () => openEdit(row), '#1890ff'),
            iconAction(TrashOutline, '删除', () => handleDelete(row), '#d03050'),
          ]),
  })

  return cols
})

function iconAction(iconComp: any, tooltip: string, onClick: () => void, color: string) {
  return h(NTooltip, { trigger: 'hover', placement: 'top' }, {
    default: () => tooltip,
    trigger: () =>
      h(NButton, { size: 'small', quaternary: true, class: 'action-btn', onClick }, {
        icon: () => h(NIcon, { size: 18, color }, () => h(iconComp)),
      }),
  })
}

function permLabel(p: number): string {
  const labels: string[] = []
  if (p & 1) labels.push('浏览')
  if (p & 2) labels.push('下载')
  if (p & 4) labels.push('上传')
  if (p & 8) labels.push('分享')
  if (p & 16) labels.push('日志')
  return labels.join(', ')
}

function calcPerms(checks: number[]): number {
  return checks.reduce((sum, v) => sum | v, 0)
}

onMounted(() => fetchUsers())

async function fetchUsers() {
  try {
    const res = await api.get('/api/users')
    users.value = res.data
  } catch (err: any) {
    message.error(err.response?.data?.error || '获取用户列表失败')
  }
}

async function handleCreate() {
  try {
    await createFormRef.value?.validate()
  } catch {
    return
  }
  createLoading.value = true
  try {
    await api.post('/api/users', { ...createForm, permissions: calcPerms(permChecks.value) })
    message.success('用户创建成功')
    showCreateModal.value = false
    fetchUsers()
  } catch (err: any) {
    message.error(err.response?.data?.error || '创建失败')
  } finally {
    createLoading.value = false
  }
}

function openEdit(row: any) {
  editForm.id = row.id
  editForm.display_name = row.display_name
  editForm.email = row.email
  editForm.is_admin = row.is_admin
  editForm.totp_enabled = row.totp_enabled
  originalTotpEnabled.value = row.totp_enabled
  editPermChecks.value = []
  if (row.permissions & 1) editPermChecks.value.push(1)
  if (row.permissions & 2) editPermChecks.value.push(2)
  if (row.permissions & 4) editPermChecks.value.push(4)
  if (row.permissions & 8) editPermChecks.value.push(8)
  if (row.permissions & 16) editPermChecks.value.push(16)
  showEditModal.value = true
}

async function handleEdit() {
  editLoading.value = true
  try {
    await api.put(`/api/users/${editForm.id}`, {
      display_name: editForm.display_name,
      email: editForm.email,
      is_admin: editForm.is_admin,
      permissions: calcPerms(editPermChecks.value),
    })

    // 如果 TOTP 状态变更，调用专门接口
    if (editForm.totp_enabled !== originalTotpEnabled.value) {
      await api.put(`/api/users/${editForm.id}/totp`, {
        totp_enabled: editForm.totp_enabled,
      })
    }

    message.success('用户更新成功')
    showEditModal.value = false
    fetchUsers()
  } catch (err: any) {
    message.error(err.response?.data?.error || '更新失败')
  } finally {
    editLoading.value = false
  }
}

async function handleDelete(row: any) {
  if (!confirm(`确认删除用户 "${row.username}"？`)) return
  try {
    await api.delete(`/api/users/${row.id}`)
    message.success('用户删除成功')
    fetchUsers()
  } catch (err: any) {
    message.error(err.response?.data?.error || '删除失败')
  }
}

</script>

<style scoped>
.users-data-table :deep(.n-data-table-td) {
  font-variant-numeric: tabular-nums;
}
</style>
