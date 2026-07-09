<template>
  <n-spin :show="loading">
    <n-form label-placement="left" label-width="180px" :model="form">
      <n-form-item label="会话超时">
        <n-input-number v-model:value="form.session_timeout" :min="10" :max="525600" style="width: 150px" />
        <span style="margin-left: 8px; color: #999">分钟（当前: {{ timeoutDisplay }}）</span>
      </n-form-item>
      <n-form-item label="登录页启用验证码">
        <n-switch v-model:value="form.enable_captcha" />
      </n-form-item>
      <template v-if="form.enable_captcha">
        <n-form-item label="验证码长度">
          <n-input-number v-model:value="form.captcha_code_length" :min="4" :max="10" style="width: 120px" />
          <span style="margin-left: 8px; color: #999">个</span>
        </n-form-item>
      </template>

      <n-divider>IP 封锁设置</n-divider>
      <n-form-item label="启用IP自动封锁">
        <n-switch v-model:value="form.ip_block_enabled" />
      </n-form-item>
      <template v-if="form.ip_block_enabled">
        <n-form-item label="失败次数阈值">
          <n-input-number v-model:value="form.ip_block_max_failures" :min="1" :max="100" style="width: 120px" />
          <span style="margin-left: 8px; color: #999">次</span>
        </n-form-item>
        <n-form-item label="检测时间窗口">
          <n-input-number v-model:value="form.ip_block_window" :min="60" :max="86400" style="width: 120px" />
          <span style="margin-left: 8px; color: #999">秒</span>
        </n-form-item>
        <n-form-item label="封锁时长">
          <n-input-number v-model:value="form.ip_block_duration" :min="60" :max="86400" style="width: 120px" />
          <span style="margin-left: 8px; color: #999">秒</span>
        </n-form-item>
      </template>

      <n-divider>账号封锁设置</n-divider>
      <n-form-item label="启用账号自动封锁">
        <n-switch v-model:value="form.account_block_enabled" />
      </n-form-item>
      <template v-if="form.account_block_enabled">
        <n-form-item label="失败次数阈值">
          <n-input-number v-model:value="form.account_block_max_failures" :min="1" :max="100" style="width: 120px" />
          <span style="margin-left: 8px; color: #999">次</span>
        </n-form-item>
        <n-form-item label="检测时间窗口">
          <n-input-number v-model:value="form.account_block_window" :min="60" :max="86400" style="width: 120px" />
          <span style="margin-left: 8px; color: #999">秒</span>
        </n-form-item>
        <n-form-item label="封锁时长">
          <n-input-number v-model:value="form.account_block_duration" :min="60" :max="86400" style="width: 120px" />
          <span style="margin-left: 8px; color: #999">秒</span>
        </n-form-item>
      </template>

      <n-divider>封锁例外白名单</n-divider>
      <n-form-item label="白名单IP/网段">
        <n-dynamic-tags v-model:value="form.whitelist_ips" />
      </n-form-item>
      <n-alert type="info" style="margin-bottom: 16px">
        支持 IP 地址（如 192.168.1.100）或 CIDR 网段（如 10.0.0.0/8）。白名单内的 IP 永远不会被封锁，且可登录已被封锁的账号。
      </n-alert>

      <n-divider>TOTP 二次认证</n-divider>
      <n-form-item label="信任设备有效期">
        <n-input-number v-model:value="form.totp_trust_days" :min="1" :max="365" style="width: 120px" />
        <span style="margin-left: 8px; color: #999">天（设为 0 则不允许信任设备）</span>
      </n-form-item>

      <n-form-item>
        <n-button type="primary" :loading="saving" @click="handleSave">保存</n-button>
      </n-form-item>
    </n-form>
  </n-spin>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { NForm, NFormItem, NInputNumber, NButton, NSwitch, NDivider, NDynamicTags, NAlert, NSpin, useMessage } from 'naive-ui'
import api from '@/api'

const message = useMessage()
const loading = ref(false)
const saving = ref(false)
const form = ref({
  session_timeout: 10080,
  enable_captcha: false,
  captcha_code_length: 6,
  ip_block_enabled: false,
  ip_block_max_failures: 5,
  ip_block_window: 300,
  ip_block_duration: 1800,
  account_block_enabled: false,
  account_block_max_failures: 5,
  account_block_window: 300,
  account_block_duration: 1800,
  whitelist_ips: [] as string[],
  totp_trust_days: 30,
})

const timeoutDisplay = computed(() => {
  const m = form.value.session_timeout
  if (m >= 1440) return `${Math.round(m / 1440)} 天`
  if (m >= 60) return `${Math.round(m / 60)} 小时`
  return `${m} 分钟`
})

onMounted(async () => {
  loading.value = true
  try {
    const res = await api.get('/api/admin/config/security')
    Object.assign(form.value, res.data)
    if (!form.value.whitelist_ips) form.value.whitelist_ips = []
  } catch { /* ignore */ } finally {
    loading.value = false
  }
})

async function handleSave() {
  saving.value = true
  try {
    await api.put('/api/admin/config/security', form.value)
    message.success('保存成功')
  } catch (err: any) {
    message.error(err.response?.data?.error || '保存失败')
  } finally {
    saving.value = false
  }
}
</script>
