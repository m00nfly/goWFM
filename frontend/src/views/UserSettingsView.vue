<template>
  <div class="user-settings-page" :class="{ dark: themeStore.isDark }">
    <n-card title="个人设置" :bordered="false" class="user-settings-card">
      <div class="settings-grid">
        <n-card title="个人资料" :bordered="false" class="settings-card">
          <n-form :model="form" label-placement="top">
            <n-form-item label="显示名称">
              <n-input v-model:value="form.display_name" />
            </n-form-item>
            <n-form-item label="邮箱">
              <n-input v-model:value="form.email" />
            </n-form-item>
            <div class="card-actions">
              <n-button type="primary" :loading="saving" @click="handleSave">保存</n-button>
            </div>
          </n-form>
        </n-card>

        <n-card title="修改密码" :bordered="false" class="settings-card">
          <n-form :model="pwForm" label-placement="top">
            <n-form-item label="当前密码">
              <n-input v-model:value="pwForm.current_password" type="password" />
            </n-form-item>
            <n-form-item label="新密码">
              <n-input v-model:value="pwForm.new_password" type="password" />
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

        <n-card title="二次认证 (TOTP)" :bordered="false" class="settings-card">
          <n-spin :show="totpLoading">
            <div class="totp-section">
              <div class="totp-status-row">
                <n-tag :type="totpEnabled ? 'success' : 'default'">
                  {{ totpEnabled ? '已启用' : '未启用' }}
                </n-tag>
                <span v-if="totpEnabled" class="muted-text">
                  剩余恢复码：{{ recoveryRemaining }} 个
                </span>
              </div>
              <div class="totp-actions">
                <n-button v-if="!totpEnabled" type="primary" size="small" @click="openTotpModal">
                  启用 TOTP
                </n-button>
                <n-popconfirm v-if="totpEnabled" @positive-click="handleDisable" positive-text="确认" negative-text="取消">
                  <template #trigger>
                    <n-button type="error" size="small" :loading="disableLoading">禁用 TOTP</n-button>
                  </template>
                  禁用后将清除所有恢复码和信任设备，确认?
                </n-popconfirm>
              </div>
            </div>
          </n-spin>
        </n-card>
      </div>
    </n-card>

    <!-- TOTP 设置向导弹窗 -->
    <n-modal
      v-model:show="showTotpModal"
      title="启用 TOTP 二次认证"
      preset="dialog"
      style="width: min(500px, calc(100vw - 32px))"
      @close="handleModalClose"
    >
      <!-- 步骤 1：扫码绑定 -->
      <div v-if="setupStep === 1" class="totp-setup-modal">
        <p class="setup-desc">请使用 Microsoft Authenticator 或 Google Authenticator 扫描以下二维码：</p>
        <div class="qr-wrap">
          <img v-if="qrImage" :src="qrImage" alt="TOTP QR Code" class="qr-img" />
          <n-spin v-else size="medium" />
        </div>
        <n-form-item label="手动输入密钥" style="margin-top: 12px">
          <n-input :value="totpSecret" readonly style="font-family: monospace" />
        </n-form-item>
      </div>

      <!-- 步骤 2：输入验证码确认 -->
      <div v-if="setupStep === 2" class="totp-setup-modal">
        <p class="setup-desc">请输入 APP 中显示的 6 位验证码以完成绑定：</p>
        <n-space align="center">
          <n-input v-model:value="verifyCode" placeholder="例如 123456" maxlength="6" style="width: 160px" />
          <n-button type="primary" size="small" :loading="verifyLoading" @click="handleVerify">验证</n-button>
        </n-space>
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
          <n-button @click="handleModalClose">取消</n-button>
          <n-button type="primary" @click="setupStep = 2">下一步</n-button>
        </template>
        <template v-if="setupStep === 2">
          <n-button @click="handleModalClose">取消</n-button>
          <n-button type="primary" :loading="verifyLoading" @click="handleVerify">验证</n-button>
        </template>
        <template v-if="setupStep === 3">
          <n-button type="primary" @click="copyRecoveryCodes">复制全部恢复码</n-button>
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
const recoveryRemaining = ref(0)
const setupStep = ref(0) // 0=none, 1=QR, 2=verify, 3=recovery codes
const qrImage = ref('')
const totpSecret = ref('')
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
    recoveryRemaining.value = res.data.recovery_codes_remaining || 0
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
  showTotpModal.value = false
  setupStep.value = 0
  qrImage.value = ''
  totpSecret.value = ''
  verifyCode.value = ''
  recoveryCodes.value = []
}

async function startSetup() {
  try {
    const res = await api.post('/api/users/me/totp/setup')
    qrImage.value = res.data.qr_code
    totpSecret.value = res.data.secret
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
  totpSecret.value = ''
  recoveryCodes.value = []
  showTotpModal.value = false
  loadTOTPStatus()
}
</script>

<style scoped>
.user-settings-page {
  width: 100%;
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;

  --us-bg-card: #fff;
  --us-fg-primary: #1e293b;
  --us-fg-muted: #64748b;
  --us-border-card: rgba(var(--theme-color-rgb, 59, 130, 246), 0.12);
  --us-a-card-hover-border: 0.35;
  --us-a-card-shadow: 0.1;
  --us-a-card-shadow-base: 0.06;
  --us-a-card-accent: 0.08;
}

.user-settings-page.dark {
  --us-bg-card: #1e293b;
  --us-fg-primary: #f1f5f9;
  --us-fg-muted: #94a3b8;
  --us-border-card: rgba(var(--theme-color-rgb, 59, 130, 246), 0.18);
  --us-a-card-hover-border: 0.5;
  --us-a-card-shadow: 0.22;
  --us-a-card-shadow-base: 0.16;
  --us-a-card-accent: 0.12;
}

.user-settings-card {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.user-settings-card :deep(.n-card__content) {
  overflow-y: auto;
}

.settings-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 320px), 1fr));
  gap: 16px;
  align-items: start;
}

.settings-card {
  height: 100%;
  border-radius: 8px;
  background: var(--us-bg-card);
  border: 1px solid var(--us-border-card);
  box-shadow: 0 2px 10px rgba(var(--theme-color-rgb, 59, 130, 246), var(--us-a-card-shadow-base));
  transition: transform 0.2s ease, border-color 0.2s ease, box-shadow 0.2s ease;
}

.settings-card:hover {
  transform: translateY(-2px);
  border-color: rgba(var(--theme-color-rgb, 59, 130, 246), var(--us-a-card-hover-border));
  box-shadow: 0 4px 18px rgba(var(--theme-color-rgb, 59, 130, 246), var(--us-a-card-shadow));
}

.settings-card :deep(.n-card-header) {
  color: var(--us-fg-primary);
}

.settings-card :deep(.n-card-header__main) {
  font-weight: 700;
}

.settings-card :deep(.n-card__content) {
  color: var(--us-fg-primary);
}

.settings-card :deep(.n-card__content) {
  display: flex;
  flex-direction: column;
}

.card-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 4px;
}

.totp-code-input {
  max-width: 240px;
}

.totp-section {
  display: flex;
  flex-direction: column;
  min-height: 132px;
  justify-content: space-between;
  gap: 16px;
}

.totp-status-row,
.totp-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
}

.muted-text {
  color: var(--us-fg-muted);
  font-size: 13px;
}

.totp-setup-modal {
  padding: 4px 0;
}

.setup-desc {
  color: #666;
  font-size: 14px;
  margin-bottom: 16px;
}

.qr-wrap {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 200px;
  height: 200px;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  background: #fff;
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
  color: var(--us-fg-primary);
  background: rgba(var(--theme-color-rgb, 59, 130, 246), var(--us-a-card-accent));
  border-radius: 6px;
  text-align: center;
  letter-spacing: 2px;
}

@media (max-width: 640px) {
  .settings-grid {
    gap: 12px;
  }

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
