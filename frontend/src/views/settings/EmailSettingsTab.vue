<template>
  <div class="workspace-form-scroll settings-tab-scroll">
    <n-spin :show="loading">
      <n-form class="settings-tab-form" label-placement="left" label-width="160px" :model="form">
        <section class="settings-section">
          <header class="settings-section-header">
            <h2>SMTP 服务</h2>
            <p>配置系统通知邮件的发送服务器与安全连接策略</p>
          </header>
          <div class="settings-section-body">
            <n-form-item label="SMTP 服务器">
              <n-input v-model:value="form.smtp_host" placeholder="如 smtp.example.com" />
            </n-form-item>
            <n-form-item label="SMTP 端口">
              <n-input-number v-model:value="form.smtp_port" :min="1" :max="65535" style="width: 150px" />
            </n-form-item>
            <n-form-item label="用户名">
              <n-input v-model:value="form.smtp_username" placeholder="SMTP 认证用户名" />
            </n-form-item>
            <n-form-item label="密码">
              <n-input
                v-model:value="form.smtp_password"
                type="password"
                show-password-on="click"
                :placeholder="hasPassword ? '已设置（留空不修改）' : '请输入 SMTP 密码'"
              />
            </n-form-item>
            <n-form-item label="发件人地址">
              <n-input v-model:value="form.sender_address" placeholder="如 noreply@example.com" />
            </n-form-item>
            <n-form-item label="启用 TLS">
              <n-switch v-model:value="form.enable_tls" />
            </n-form-item>
            <n-form-item label="跳过 TLS 验证">
              <n-switch v-model:value="form.skip_tls_verify" />
              <span class="workspace-inline-note">仅用于自签名证书的SMTP服务器</span>
            </n-form-item>

          </div>
        </section>
        <footer class="settings-tab-actions">
          <n-button type="primary" :loading="saving" @click="handleSave">保存设置</n-button>
        </footer>
      </n-form>
    </n-spin>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { NForm, NFormItem, NInput, NInputNumber, NButton, NSwitch, NSpin, useMessage } from 'naive-ui'
import api from '@/api'

const message = useMessage()
const loading = ref(false)
const saving = ref(false)
const hasPassword = ref(false)
const form = ref({
  smtp_host: '',
  smtp_port: 587,
  smtp_username: '',
  smtp_password: '',
  enable_tls: true,
  skip_tls_verify: false,
  sender_address: '',
})

onMounted(async () => {
  loading.value = true
  try {
    const res = await api.get('/api/admin/config/email')
    Object.assign(form.value, res.data)
    hasPassword.value = res.data.has_password || false
    form.value.smtp_password = '' // 不回显密码
  } catch { /* ignore */ } finally {
    loading.value = false
  }
})

async function handleSave() {
  saving.value = true
  try {
    const payload = { ...form.value }
    // 密码为空且之前已设置，则不发送密码字段（后端不更新）
    if (!payload.smtp_password && hasPassword.value) {
      delete (payload as any).smtp_password
    }
    await api.put('/api/admin/config/email', payload)
    message.success('保存成功')
    if (form.value.smtp_password) hasPassword.value = true
  } catch (err: any) {
    message.error(err.response?.data?.error || '保存失败')
  } finally {
    saving.value = false
  }
}
</script>
