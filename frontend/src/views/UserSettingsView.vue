<template>
  <div class="workspace-page user-settings-page" :class="{ dark: themeStore.isDark }">
    <section class="workspace-surface user-settings-surface">
      <header class="workspace-header">
        <div class="workspace-title-block">
          <h1 class="workspace-title">个人设置</h1>
          <p class="workspace-subtitle">维护个人资料、登录密码与二次认证状态</p>
        </div>
      </header>

      <div class="workspace-form-scroll user-settings-scroll">
        <div class="settings-grid">
          <n-card :bordered="false" class="workspace-mini-card settings-card profile-card">
          <template #header><div class="settings-card-heading"><strong>个人资料</strong><span>用于页面展示和账号联系</span></div></template>
          <n-form :model="form" label-placement="top" class="settings-form">
            <div class="settings-field-grid">
            <n-form-item label="显示名称">
              <n-input v-model:value="form.display_name" placeholder="请输入显示名称" />
            </n-form-item>
            <n-form-item label="邮箱">
              <n-input v-model:value="form.email" placeholder="请输入邮箱地址" />
            </n-form-item>
            </div>
            <div class="card-actions">
              <n-button type="primary" :loading="saving" @click="handleSave">保存</n-button>
            </div>
          </n-form>
          </n-card>

          <n-card :bordered="false" class="workspace-mini-card settings-card password-card">
          <template #header><div class="settings-card-heading"><strong>修改密码</strong><span>定期更新密码可降低账号风险</span></div></template>
          <n-form :model="pwForm" label-placement="top" class="settings-form">
            <n-form-item label="当前密码">
              <n-input v-model:value="pwForm.current_password" type="password" show-password-on="click" placeholder="请输入当前密码" />
            </n-form-item>
            <n-form-item label="新密码">
              <n-input v-model:value="pwForm.new_password" type="password" show-password-on="click" placeholder="至少 6 位" />
            </n-form-item>
            <n-form-item v-if="totpEnabled" label="TOTP 验证码">
              <n-input
                v-model:value="pwForm.totp_code"
                placeholder="如已启用 TOTP，修改密码需要验证"
                maxlength="6"
                class="totp-code-input"
              />
            </n-form-item>
            <div class="card-actions">
              <n-button type="primary" :loading="pwSaving" @click="handlePasswordChange">修改密码</n-button>
            </div>
          </n-form>
          </n-card>

          <n-card :bordered="false" class="workspace-mini-card settings-card security-card">
          <template #header><div class="settings-card-heading"><strong>二次认证 (TOTP)</strong><span>使用验证器 APP 保护账号登录</span></div></template>
          <n-spin :show="totpLoading">
            <div class="totp-section">
              <div class="security-summary">
                <div class="security-copy">
                  <div class="totp-status-row">
                    <n-tag :type="totpEnabled ? 'success' : 'default'">{{ totpEnabled ? '已启用' : '未启用' }}</n-tag>
                    <span v-if="totpEnabled" class="muted-text">剩余恢复码：{{ recoveryRemaining }} 个</span>
                  </div>
                  <span v-if="totpForced" class="muted-text">管理员已强制启用，无法自行关闭</span>
                  <span v-else-if="totpResetRequired" class="muted-text">恢复码已使用，请重新绑定或关闭 OTP</span>
                  <span v-else class="muted-text">启用后，登录时需要输入验证器生成的动态验证码</span>
                </div>
                <div class="totp-actions">
                  <n-button v-if="!totpEnabled" type="primary" @click="openTotpModal">{{ totpResetRequired ? '重新绑定 OTP' : '启用 TOTP' }}</n-button>
                  <n-popconfirm v-if="totpResetRequired && !totpForced" @positive-click="handleDisable" positive-text="关闭 OTP" negative-text="取消">
                    <template #trigger><n-button type="error" secondary :loading="disableLoading">关闭 OTP</n-button></template>
                    关闭后将不再需要二次验证，确认继续？
                  </n-popconfirm>
                  <n-popconfirm v-if="totpEnabled && !totpForced" @positive-click="handleDisable" positive-text="确认" negative-text="取消">
                    <template #trigger><n-button type="error" secondary :loading="disableLoading">禁用 TOTP</n-button></template>
                    禁用后将清除所有恢复码和信任设备，确认？
                  </n-popconfirm>
                </div>
              </div>
            </div>
          </n-spin>
          </n-card>
        </div>
      </div>
    </section>

    <!-- TOTP 设置向导弹窗 -->
    <n-modal
      v-model:show="showTotpModal"
      title="启用 TOTP 二次认证"
      preset="dialog"
      :closable="!(totpForced && !totpEnabled)"
      :mask-closable="!(totpForced && !totpEnabled)"
      :close-on-esc="!(totpForced && !totpEnabled)"
      style="width: min(500px, calc(100vw - 32px))"
      @close="handleModalClose"
    >
      <!-- 扫码并验证 -->
      <div v-if="setupStep === 1" class="totp-setup-modal">
        <p class="setup-desc">请使用 Microsoft Authenticator 或 Google Authenticator 扫描以下二维码：</p>
        <div class="qr-wrap">
          <img v-if="qrImage" :src="qrImage" alt="TOTP QR Code" class="qr-img" />
          <n-spin v-else size="medium" />
        </div>
        <n-form-item label="APP 中显示的 6 位验证码" style="margin-top: 12px">
          <n-input v-model:value="verifyCode" placeholder="例如 123456" maxlength="6" inputmode="numeric" />
        </n-form-item>
      </div>

      <!-- 步骤 3：显示恢复码 -->
      <div v-if="setupStep === 3" class="totp-setup-modal">
        <n-alert type="warning" style="margin-bottom: 12px">
          请立即保存以下恢复码！关闭此页面后将无法再次查看。每个恢复码只能使用一次。
        </n-alert>
        <div class="recovery-codes">
          <div v-for="(code, idx) in recoveryCodes" :key="idx" class="recovery-code-item">
            {{ code }}
          </div>
        </div>
      </div>

      <template #action>
        <template v-if="setupStep === 1">
          <n-button v-if="!(totpForced && !totpEnabled)" @click="handleModalClose">取消</n-button>
          <n-button type="primary" :loading="verifyLoading" @click="handleVerify">验证</n-button>
        </template>
        <template v-if="setupStep === 3">
          <n-button type="primary" @click="copyRecoveryCodes">复制恢复码</n-button>
          <n-button @click="finishSetup">完成</n-button>
        </template>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, onMounted } from 'vue'
import {
  NCard, NForm, NFormItem, NInput, NButton, NSpin, NTag, NPopconfirm,
  NAlert, NModal, useMessage,
} from 'naive-ui'
import api from '@/api'
import { useUserStore } from '@/stores/user'
import { useThemeStore } from '@/stores/theme'

const message = useMessage()
const userStore = useUserStore()
const themeStore = useThemeStore()
const saving = ref(false)
const pwSaving = ref(false)

const form = reactive({ display_name: '', email: '' })
const pwForm = reactive({ current_password: '', new_password: '', totp_code: '' })

// TOTP
const totpLoading = ref(true)
const totpEnabled = ref(false)
const totpForced = ref(false)
const totpResetRequired = ref(false)
const recoveryRemaining = ref(0)
const setupStep = ref(0) // 0=none, 1=QR + verify, 3=recovery codes
const qrImage = ref('')
const verifyCode = ref('')
const verifyLoading = ref(false)
const disableLoading = ref(false)
const recoveryCodes = ref<string[]>([])
const showTotpModal = ref(false)

onMounted(async () => {
  if (userStore.user) {
    form.display_name = userStore.user.display_name
    form.email = userStore.user.email
  }
  await loadTOTPStatus()
})

async function loadTOTPStatus() {
  totpLoading.value = true
  try {
    const res = await api.get('/api/users/me/totp/status')
    totpEnabled.value = res.data.totp_enabled
    totpForced.value = res.data.totp_forced
    totpResetRequired.value = res.data.reset_required
    recoveryRemaining.value = res.data.recovery_codes_remaining || 0
    if (res.data.setup_required && !showTotpModal.value) {
      message.warning(totpResetRequired.value ? '恢复码登录后旧 OTP 已失效，请重新绑定或选择关闭 OTP' : '管理员要求启用 TOTP，请先完成扫码绑定')
      await openTotpModal()
    }
  } catch { /* ignore */ } finally {
    totpLoading.value = false
  }
}

async function handleSave() {
  saving.value = true
  try {
    await api.put('/api/users/me', form)
    await userStore.fetchMe()
    message.success('保存成功')
  } catch (err: any) {
    message.error(err.response?.data?.error || '保存失败')
  } finally {
    saving.value = false
  }
}

async function handlePasswordChange() {
  if (!pwForm.current_password || !pwForm.new_password) {
    message.warning('请填写所有密码字段')
    return
  }
  pwSaving.value = true
  try {
    await api.put('/api/users/me/password', {
      current_password: pwForm.current_password,
      new_password: pwForm.new_password,
      totp_code: pwForm.totp_code,
    })
    message.success('密码修改成功')
    pwForm.current_password = ''
    pwForm.new_password = ''
    pwForm.totp_code = ''
  } catch (err: any) {
    message.error(err.response?.data?.error || '密码修改失败')
  } finally {
    pwSaving.value = false
  }
}

// TOTP functions
async function openTotpModal() {
  showTotpModal.value = true
  await startSetup()
}

function handleModalClose() {
  if (totpForced.value && !totpEnabled.value) {
    message.warning('完成 TOTP 绑定后才能继续使用系统')
    return
  }
  showTotpModal.value = false
  setupStep.value = 0
  qrImage.value = ''
  verifyCode.value = ''
  recoveryCodes.value = []
}

async function startSetup() {
  try {
    const res = await api.post('/api/users/me/totp/setup')
    qrImage.value = res.data.qr_code
    setupStep.value = 1
  } catch (err: any) {
    message.error(err.response?.data?.error || '生成绑定信息失败')
  }
}

async function handleVerify() {
  if (!verifyCode.value || verifyCode.value.length < 6) {
    message.warning('请输入 6 位验证码')
    return
  }
  verifyLoading.value = true
  try {
    const res = await api.post('/api/users/me/totp/verify', { code: verifyCode.value })
    recoveryCodes.value = res.data.recovery_codes
    setupStep.value = 3
    verifyCode.value = ''
  } catch (err: any) {
    message.error(err.response?.data?.error || '验证失败')
  } finally {
    verifyLoading.value = false
  }
}

async function handleDisable() {
  disableLoading.value = true
  try {
    await api.post('/api/users/me/totp/disable')
    totpEnabled.value = false
    totpResetRequired.value = false
    recoveryRemaining.value = 0
    message.success('TOTP 已禁用')
  } catch (err: any) {
    message.error(err.response?.data?.error || '禁用失败')
  } finally {
    disableLoading.value = false
  }
}

function copyRecoveryCodes() {
  const text = recoveryCodes.value.join('\n')
  navigator.clipboard.writeText(text).then(() => {
    message.success('已复制到剪贴板')
  }).catch(() => {
    message.warning('复制失败，请手动复制')
  })
}

function finishSetup() {
  setupStep.value = 0
  qrImage.value = ''
  recoveryCodes.value = []
  showTotpModal.value = false
  loadTOTPStatus().then(() => userStore.fetchMe())
}
</script>

<style scoped>
.user-settings-page {
  width: 100%;
}

.user-settings-scroll {
  padding: 10px;
}

.settings-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  align-items: start;
  gap: 16px;
}

.security-card { grid-column: 1 / -1; }

.settings-card {
  height: 100%;
}

.settings-card :deep(.n-card__content) {
  display: flex;
  flex-direction: column;
}

.settings-card-heading { display: grid; gap: 4px; }
.settings-card-heading strong { font-size: 16px; text-wrap: balance; }
.settings-card-heading span { color: var(--workspace-text-muted); font-size: 12px; font-weight: 400; text-wrap: pretty; }
.settings-form { height: 100%; display: flex; flex-direction: column; }
.settings-field-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; }

.card-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 4px;
}

.settings-form .card-actions { margin-top: auto; }

.totp-code-input {
  max-width: 240px;
}

.totp-section {
  min-height: 72px;
}

.security-summary { display: flex; align-items: center; justify-content: space-between; gap: 24px; }
.security-copy { display: grid; gap: 8px; }

.totp-status-row,
.totp-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
}

.muted-text {
  color: var(--workspace-text-muted);
  font-size: 13px;
}

.totp-setup-modal {
  padding: 4px 0;
}

.setup-desc {
  color: var(--workspace-text-muted);
  font-size: 14px;
  margin-bottom: 16px;
}

.qr-wrap {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 200px;
  height: 200px;
  border: 1px solid var(--workspace-border);
  border-radius: var(--workspace-radius-md);
  background: var(--workspace-surface);
}

.qr-img {
  width: 180px;
  height: 180px;
}

.recovery-codes {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 8px;
}

.recovery-code-item {
  font-family: monospace;
  font-size: 14px;
  padding: 8px 12px;
  color: var(--workspace-text);
  background: rgba(var(--workspace-accent-rgb), 0.09);
  border: 1px solid var(--workspace-border-soft);
  border-radius: var(--workspace-radius-sm);
  text-align: center;
  letter-spacing: 2px;
}

@media (max-width: 640px) {
  .settings-grid {
    grid-template-columns: 1fr;
    gap: 12px;
  }

  .security-card { grid-column: auto; }
  .settings-field-grid { grid-template-columns: 1fr; }
  .security-summary { align-items: stretch; flex-direction: column; }

  .card-actions {
    justify-content: stretch;
  }

  .card-actions .n-button {
    width: 100%;
  }

  .totp-actions .n-button {
    width: 100%;
  }
}
</style>
