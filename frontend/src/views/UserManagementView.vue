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

      <n-spin :show="usersLoading" class="users-card-shell">
        <div v-if="users.length" class="users-card-list" aria-label="用户列表">
          <article v-for="user in users" :key="user.id" class="user-card">
            <div class="user-card-profile">
              <UserAvatar
                :size="60"
                :avatar="user.avatar"
                :name="user.display_name || user.username"
              />
              <div class="user-identity">
                <div class="user-name-row">
                  <h2>{{ user.display_name || user.username }}</h2>
                  <n-tag size="small" :type="user.is_admin ? 'error' : user.id === 0 ? 'default' : 'info'">
                    {{ user.is_admin ? '管理员' : user.id === 0 ? '系统账户' : '普通用户' }}
                  </n-tag>
                </div>
                <span class="username">@{{ user.username }} · ID {{ user.id }}</span>
                <span class="user-email">{{ user.email || '未设置邮箱' }}</span>
              </div>
            </div>

            <div class="user-card-details">
              <div class="user-detail-group permission-summary">
                <span class="detail-label">功能权限</span>
                <div v-if="permissionLabels(user.permissions).length" class="permission-tags">
                  <n-tag v-for="permission in permissionLabels(user.permissions)" :key="permission" size="small" :bordered="false">
                    {{ permission }}
                  </n-tag>
                </div>
                <span v-else class="detail-empty">未授予功能权限</span>
              </div>
              <div class="user-detail-group totp-summary">
                <span class="detail-label">二次认证</span>
                <n-tag :type="totpStatus(user).type" size="small">{{ totpStatus(user).label }}</n-tag>
              </div>
            </div>

            <div v-if="user.id !== 0" class="user-card-actions">
              <n-button secondary @click="openEdit(user)">
                <template #icon><n-icon><CreateOutline /></n-icon></template>
                编辑
              </n-button>
              <n-button secondary type="error" @click="handleDelete(user)">
                <template #icon><n-icon><TrashOutline /></n-icon></template>
                删除
              </n-button>
            </div>
            <div v-else class="user-card-actions system-account-note">由系统维护</div>
          </article>
        </div>
        <n-empty v-else-if="!usersLoading" description="暂无用户" class="users-empty" />
      </n-spin>
    </section>
  </div>

  <n-modal v-model:show="showCreateModal" title="创建用户" preset="dialog" class="create-user-modal">
    <n-form ref="createFormRef" :model="createForm" :rules="createRules" label-placement="top" class="edit-user-form">
      <section class="edit-section">
        <div class="edit-section-heading"><h3>账户信息</h3><span>用于登录系统</span></div>
        <div class="edit-field-grid">
          <n-form-item label="用户名" path="username"><n-input v-model:value="createForm.username" placeholder="请输入用户名" /></n-form-item>
          <n-form-item label="初始密码" path="password"><n-input v-model:value="createForm.password" type="password" show-password-on="click" placeholder="至少 6 位" /></n-form-item>
          <n-form-item label="显示名称"><n-input v-model:value="createForm.display_name" placeholder="选填" /></n-form-item>
		  <n-form-item label="邮箱" path="email"><n-input v-model:value="createForm.email" placeholder="用于通知与密码找回" /></n-form-item>
        </div>
      </section>
      <section class="edit-section">
        <div class="edit-section-heading"><h3>角色与权限</h3></div>
        <div class="setting-row">
          <div class="setting-copy"><strong>管理员</strong><span>拥有全部系统管理权限</span></div>
          <n-switch v-model:value="createForm.is_admin" />
        </div>
        <div class="permission-block">
          <span class="permission-label">功能权限</span>
          <n-checkbox-group :value="permChecks" @update:value="handleCreatePermissionChange"><div class="permission-grid">
            <n-checkbox :value="1">浏览</n-checkbox><n-checkbox :value="2">下载</n-checkbox><n-checkbox :value="4">全局上传</n-checkbox><n-checkbox :value="8">分享</n-checkbox><n-checkbox :value="16">日志</n-checkbox>
          </div></n-checkbox-group>
        </div>
      </section>
      <section class="edit-section">
        <div class="edit-section-heading"><h3>账户安全</h3></div>
        <div class="setting-row">
          <div class="setting-copy"><strong>强制 OTP</strong><span>首次登录时绑定，启用后用户不可自行关闭 OTP</span></div>
          <n-switch v-model:value="createForm.totp_forced" />
        </div>
      </section>
    </n-form>
    <template #action>
      <n-button @click="showCreateModal = false">取消</n-button>
      <n-button type="primary" :loading="createLoading" @click="handleCreate">创建</n-button>
    </template>
  </n-modal>

  <n-modal v-model:show="showEditModal" title="编辑用户" preset="dialog" class="edit-user-modal">
	<n-form ref="editFormRef" :model="editForm" :rules="editRules" label-placement="top" class="edit-user-form">
      <section class="edit-section">
        <div class="edit-section-heading">
          <h3>基础信息</h3>
          <span>{{ editForm.username }}</span>
        </div>
        <div class="edit-field-grid">
          <n-form-item label="显示名称">
            <n-input v-model:value="editForm.display_name" placeholder="请输入显示名称" />
          </n-form-item>
		  <n-form-item label="邮箱" path="email">
            <n-input v-model:value="editForm.email" placeholder="请输入邮箱地址" />
          </n-form-item>
        </div>
      </section>

      <section class="edit-section">
        <div class="edit-section-heading"><h3>角色与权限</h3></div>
        <div class="setting-row">
          <div class="setting-copy"><strong>管理员</strong><span>拥有全部系统管理权限</span></div>
          <n-switch v-model:value="editForm.is_admin" />
        </div>
        <div class="permission-block">
          <span class="permission-label">功能权限</span>
          <n-checkbox-group :value="editPermChecks" @update:value="handleEditPermissionChange">
            <div class="permission-grid">
              <n-checkbox :value="1">浏览</n-checkbox>
              <n-checkbox :value="2">下载</n-checkbox>
              <n-checkbox :value="4">全局上传</n-checkbox>
              <n-checkbox :value="8">分享</n-checkbox>
              <n-checkbox :value="16">日志</n-checkbox>
            </div>
          </n-checkbox-group>
        </div>
      </section>

      <section class="edit-section">
        <div class="edit-section-heading"><h3>账户安全</h3></div>
        <div class="setting-row security-row">
          <div class="setting-copy"><strong>强制 OTP</strong><span>启用后用户不可自行关闭 OTP</span></div>
          <div class="security-actions">
            <n-switch v-model:value="editForm.totp_forced" />
            <n-popconfirm
              v-if="editForm.totp_enabled"
              positive-text="确认重置"
              negative-text="取消"
              @positive-click="handleResetTOTP"
            >
              <template #trigger>
                <n-button secondary type="warning" :loading="resetLoading">
                  <template #icon><n-icon><RefreshOutline /></n-icon></template>
                  重置密钥
                </n-button>
              </template>
              <div class="reset-confirm-copy">
                <strong>确认重置 {{ editForm.username }} 的 OTP 密钥？</strong>
                <span>旧密钥、恢复码和信任设备将立即失效，现有会话也会被注销。</span>
              </div>
            </n-popconfirm>
          </div>
        </div>
      </section>
    </n-form>
    <template #action>
      <n-button @click="showEditModal = false">取消</n-button>
      <n-button type="primary" :loading="editLoading" @click="handleEdit">保存</n-button>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import {
  NButton, NModal, NForm, NFormItem, NInput, NEmpty, NSpin,
  NCheckboxGroup, NCheckbox, NSwitch, NTag, NIcon, NPopconfirm, useMessage, useDialog,
} from 'naive-ui'
import { AddOutline, CreateOutline, RefreshOutline, TrashOutline } from '@vicons/ionicons5'
import api from '@/api'
import UserAvatar from '@/components/UserAvatar.vue'

const message = useMessage()
const dialog = useDialog()
const users = ref<any[]>([])
const usersLoading = ref(false)
const showCreateModal = ref(false)
const showEditModal = ref(false)
const createLoading = ref(false)
const editLoading = ref(false)

const createFormRef = ref<any>(null)
const createForm = reactive({ username: '', password: '', display_name: '', email: '', is_admin: false, totp_forced: false })
const permChecks = ref<number[]>([1, 2])
const createRules = {
  username: [{ required: true, message: '必填' }],
  password: [{ required: true, message: '必填' }, { min: 6, message: '至少6位' }],
	email: [
	  { required: true, message: '邮箱为必填项', trigger: ['input', 'blur'] },
	  { type: 'email' as const, message: '请输入有效的邮箱地址', trigger: ['input', 'blur'] },
	],
}

const editFormRef = ref<any>(null)
const editForm = reactive({ id: 0, username: '', display_name: '', email: '', is_admin: false, totp_enabled: false, totp_forced: false })
const editRules = {
	email: [
	  { required: true, message: '邮箱为必填项', trigger: ['input', 'blur'] },
	  { type: 'email' as const, message: '请输入有效的邮箱地址', trigger: ['input', 'blur'] },
	],
}
const editPermChecks = ref<number[]>([])
const originalTotpForced = ref(false)
const resetLoading = ref(false)

function permissionLabels(p: number): string[] {
  const labels: string[] = []
  if (p & 1) labels.push('浏览')
  if (p & 2) labels.push('下载')
  if (p & 4) labels.push('全局上传')
  if (p & 8) labels.push('分享')
  if (p & 16) labels.push('日志')
  return labels
}

function totpStatus(user: any): { label: string; type: 'default' | 'info' | 'success' | 'warning' } {
  if (user.totp_reset_required) return { label: '待重新绑定', type: 'warning' }
  if (user.totp_forced) {
    return user.totp_enabled
      ? { label: '强制启用', type: 'success' }
      : { label: '待绑定', type: 'warning' }
  }
  return user.totp_enabled
    ? { label: '自主启用', type: 'info' }
    : { label: '未启用', type: 'default' }
}

function calcPerms(checks: number[]): number {
  return checks.reduce((sum, v) => sum | v, 0)
}

function confirmGlobalUploadPermission(): Promise<boolean> {
  return new Promise((resolve) => {
    let settled = false
    const finish = (confirmed: boolean) => {
      if (settled) return
      settled = true
      resolve(confirmed)
    }
    dialog.warning({
      title: '确认授予全局上传权限',
      content: '用户默认可向自有目录上传文件，确定授予任意目录的上传权限？',
      positiveText: '确定授予',
      negativeText: '取消',
      maskClosable: false,
      onPositiveClick: () => finish(true),
      onNegativeClick: () => finish(false),
      onClose: () => finish(false),
    })
  })
}

async function updatePermissionChecks(current: number[], next: number[], apply: (value: number[]) => void) {
  const grantsGlobalUpload = !current.includes(4) && next.includes(4)
  if (grantsGlobalUpload && !(await confirmGlobalUploadPermission())) return
  apply(next)
}

function handleCreatePermissionChange(value: Array<string | number>) {
  const next = value.map(Number)
  void updatePermissionChecks(permChecks.value, next, value => { permChecks.value = value })
}

function handleEditPermissionChange(value: Array<string | number>) {
  const next = value.map(Number)
  void updatePermissionChecks(editPermChecks.value, next, value => { editPermChecks.value = value })
}

onMounted(() => fetchUsers())

async function fetchUsers() {
  usersLoading.value = true
  try {
    const res = await api.get('/api/users')
    users.value = res.data
  } catch (err: any) {
    message.error(err.response?.data?.error || '获取用户列表失败')
  } finally {
    usersLoading.value = false
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
    Object.assign(createForm, { username: '', password: '', display_name: '', email: '', is_admin: false, totp_forced: false })
    permChecks.value = [1, 2]
    fetchUsers()
  } catch (err: any) {
    message.error(err.response?.data?.error || '创建失败')
  } finally {
    createLoading.value = false
  }
}

function openEdit(row: any) {
  editForm.id = row.id
  editForm.username = row.username
  editForm.display_name = row.display_name
  editForm.email = row.email
  editForm.is_admin = row.is_admin
  editForm.totp_enabled = row.totp_enabled
  editForm.totp_forced = row.totp_forced
  originalTotpForced.value = row.totp_forced
  editPermChecks.value = []
  if (row.permissions & 1) editPermChecks.value.push(1)
  if (row.permissions & 2) editPermChecks.value.push(2)
  if (row.permissions & 4) editPermChecks.value.push(4)
  if (row.permissions & 8) editPermChecks.value.push(8)
  if (row.permissions & 16) editPermChecks.value.push(16)
  showEditModal.value = true
}

async function handleEdit() {
	try {
	  await editFormRef.value?.validate()
	} catch {
	  return
	}
  editLoading.value = true
  try {
    await api.put(`/api/users/${editForm.id}`, {
      display_name: editForm.display_name,
      email: editForm.email,
      is_admin: editForm.is_admin,
      permissions: calcPerms(editPermChecks.value),
    })

    // 如果 TOTP 状态变更，调用专门接口
    if (editForm.totp_forced !== originalTotpForced.value) {
      await api.put(`/api/users/${editForm.id}/totp`, {
        totp_forced: editForm.totp_forced,
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

async function handleResetTOTP() {
  resetLoading.value = true
  try {
    await api.post(`/api/users/${editForm.id}/totp/reset`)
    editForm.totp_enabled = false
    message.success('TOTP 密钥已重置，用户下次登录时必须重新绑定')
    fetchUsers()
  } catch (err: any) {
    message.error(err.response?.data?.error || '重置失败')
  } finally {
    resetLoading.value = false
  }
}

</script>

<style scoped>
.users-card-shell {
  flex: 1;
  min-height: 0;
}

.users-card-shell :deep(.n-spin-content) {
  height: 100%;
  min-height: 0;
}

.users-card-list {
  height: 100%;
  overflow-y: auto;
  display: grid;
  align-content: start;
  gap: 10px;
  padding: 10px;
  scrollbar-gutter: stable;
}

.user-card {
  min-width: 0;
  display: grid;
  grid-template-columns: minmax(250px, 1.15fr) minmax(320px, 1.5fr) auto;
  align-items: center;
  gap: 20px;
  padding: 16px;
  border-radius: var(--workspace-radius-lg);
  background: var(--workspace-surface);
  box-shadow:
    0 0 0 1px var(--workspace-border-soft),
    0 2px 8px rgba(39, 55, 82, 0.05);
  transition-property: box-shadow, transform;
  transition-duration: 160ms;
  transition-timing-function: ease-out;
}

.user-card:hover {
  transform: translateY(-1px);
  box-shadow:
    0 0 0 1px rgba(var(--workspace-accent-rgb), 0.2),
    0 8px 20px rgba(var(--workspace-accent-rgb), 0.08);
}

.user-card-profile {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 14px;
}

.user-identity {
  min-width: 0;
  display: grid;
  gap: 3px;
}

.user-name-row {
  min-width: 0;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 7px;
}

.user-name-row h2 {
  min-width: 0;
  margin: 0;
  color: var(--workspace-text);
  font-size: 15px;
  font-weight: 750;
  line-height: 1.35;
  overflow-wrap: anywhere;
  text-wrap: balance;
}

.username,
.user-email {
  overflow: hidden;
  color: var(--workspace-text-muted);
  font-size: 12px;
  line-height: 1.4;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.username {
  font-variant-numeric: tabular-nums;
}

.user-card-details {
  min-width: 0;
  display: grid;
  grid-template-columns: minmax(0, 1fr) 110px;
  gap: 20px;
}

.user-detail-group {
  min-width: 0;
  display: grid;
  align-content: center;
  gap: 7px;
}

.detail-label {
  color: var(--workspace-text-soft);
  font-size: 11px;
  font-weight: 600;
}

.permission-tags {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
}

.detail-empty,
.system-account-note {
  color: var(--workspace-text-soft);
  font-size: 12px;
}

.user-card-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  white-space: nowrap;
}

.user-card-actions :deep(.n-button) {
  min-width: 72px;
}

.users-empty {
  padding-top: 80px;
}

:deep(.edit-user-modal),
:deep(.create-user-modal) {
  width: min(620px, calc(100vw - 32px));
}

.edit-user-form {
  display: grid;
  gap: 14px;
}

.edit-section {
  padding: 16px;
  border-radius: 14px;
  background: var(--workspace-soft-bg, rgba(127, 127, 127, 0.06));
  box-shadow: inset 0 0 0 1px rgba(127, 127, 127, 0.1);
}

.edit-section-heading {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
}

.edit-section-heading h3 {
  margin: 0;
  font-size: 14px;
  line-height: 1.4;
  text-wrap: balance;
}

.edit-section-heading span,
.setting-copy span {
  color: var(--n-text-color-3);
  font-size: 12px;
}

.edit-field-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.edit-field-grid :deep(.n-form-item) {
  margin-bottom: 0;
}

.setting-row {
  min-height: 44px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.setting-copy {
  display: grid;
  gap: 3px;
}

.permission-block {
  margin-top: 14px;
}

.permission-label {
  display: block;
  margin-bottom: 10px;
  color: var(--n-label-text-color);
  font-size: 13px;
  font-weight: 500;
}

.permission-grid {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px 24px;
}

.permission-grid :deep(.n-checkbox) {
  flex: 0 0 auto;
  white-space: nowrap;
}

.security-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.security-actions :deep(.n-button) {
  min-height: 40px;
}

.reset-confirm-copy {
  max-width: 280px;
  display: grid;
  gap: 5px;
}

.reset-confirm-copy span {
  color: var(--n-text-color-3);
  font-size: 12px;
  line-height: 1.5;
  text-wrap: pretty;
}

@media (max-width: 560px) {
  .edit-field-grid {
    grid-template-columns: 1fr;
  }

  .security-row {
    align-items: flex-start;
    flex-direction: column;
  }

  .security-actions {
    width: 100%;
    justify-content: space-between;
  }
}

@media (max-width: 960px) {
  .user-card {
    grid-template-columns: minmax(0, 1fr) auto;
    gap: 14px;
  }

  .user-card-details {
    grid-column: 1 / -1;
    grid-row: 2;
  }

  .user-card-actions {
    grid-column: 2;
    grid-row: 1;
  }
}

@media (max-width: 640px) {
  .user-card {
    grid-template-columns: 1fr;
    align-items: stretch;
    padding: 14px;
  }

  .user-card-details {
    grid-column: 1;
    grid-row: auto;
    grid-template-columns: 1fr;
    gap: 12px;
    padding-top: 12px;
    box-shadow: inset 0 1px 0 var(--workspace-border-soft);
  }

  .user-card-actions {
    grid-column: 1;
    grid-row: auto;
    justify-content: stretch;
  }

  .user-card-actions :deep(.n-button) {
    flex: 1;
  }
}

@media (prefers-reduced-motion: reduce) {
  .user-card {
    transition: none;
  }

  .user-card:hover {
    transform: none;
  }
}
</style>
