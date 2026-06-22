<template>
  <n-spin :show="loading">
    <n-form label-placement="left" label-width="160px" :model="form">
      <n-form-item label="默认过期天数">
        <n-input-number v-model:value="form.default_expire_days" :min="1" :max="3650" style="width: 150px" />
        <span style="margin-left: 8px; color: #999">天</span>
      </n-form-item>
      <n-form-item label="每用户最大分享数">
        <n-input-number v-model:value="form.max_shares_per_user" :min="0" :max="100000" style="width: 150px" />
        <span style="margin-left: 8px; color: #999">0 表示不限制</span>
      </n-form-item>
      <n-form-item label="允许匿名下载">
        <n-switch v-model:value="form.allow_anonymous_download" />
        <span style="margin-left: 8px; color: #999; font-size: 12px">关闭后分享链接需要登录才能下载</span>
      </n-form-item>

      <n-form-item>
        <n-button type="primary" :loading="saving" @click="handleSave">保存</n-button>
      </n-form-item>
    </n-form>
  </n-spin>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { NForm, NFormItem, NInputNumber, NButton, NSwitch, NSpin, useMessage } from 'naive-ui'
import api from '@/api'

const message = useMessage()
const loading = ref(false)
const saving = ref(false)
const form = ref({
  default_expire_days: 7,
  max_shares_per_user: 0,
  allow_anonymous_download: true,
})

onMounted(async () => {
  loading.value = true
  try {
    const res = await api.get('/api/admin/config/share')
    Object.assign(form.value, res.data)
  } catch { /* ignore */ } finally {
    loading.value = false
  }
})

async function handleSave() {
  saving.value = true
  try {
    await api.put('/api/admin/config/share', form.value)
    message.success('保存成功')
  } catch (err: any) {
    message.error(err.response?.data?.error || '保存失败')
  } finally {
    saving.value = false
  }
}
</script>
