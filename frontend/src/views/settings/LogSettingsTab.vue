<template>
  <n-spin :show="loading">
    <n-form label-placement="left" label-width="140px" :model="form">
      <n-form-item label="日志保留天数">
        <n-input-number v-model:value="form.retention_days" :min="1" :max="3650" style="width: 150px" />
        <span style="margin-left: 8px; color: #999">天（超过此天数的日志将自动清理）</span>
      </n-form-item>

      <n-form-item label="启用的日志类型">
        <n-checkbox-group v-model:value="form.enabled_log_types">
          <n-grid :cols="3" :x-gap="12" :y-gap="8">
            <n-gi v-for="item in allLogTypes" :key="item.value">
              <n-checkbox :value="item.value" :label="item.label" />
            </n-gi>
          </n-grid>
        </n-checkbox-group>
      </n-form-item>

      <n-form-item>
        <n-space>
          <n-button type="primary" :loading="saving" @click="handleSave">保存</n-button>
          <n-button @click="selectAll">全选</n-button>
          <n-button @click="deselectAll">全不选</n-button>
        </n-space>
      </n-form-item>
    </n-form>
  </n-spin>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { NForm, NFormItem, NInputNumber, NCheckboxGroup, NCheckbox, NGrid, NGi, NButton, NSpace, NSpin, useMessage } from 'naive-ui'
import api from '@/api'

const message = useMessage()
const loading = ref(false)
const saving = ref(false)
const form = ref({
  retention_days: 30,
  enabled_log_types: [] as string[],
})

const allLogTypes = [
  { value: 'LOGIN', label: '登录成功' },
  { value: 'LOGIN_FAIL', label: '登录失败' },
  { value: 'BLOCK_IP', label: 'IP封锁' },
  { value: 'BLOCK_ACCOUNT', label: '账号封锁' },
  { value: 'CONFIG_CHANGE', label: '配置变更' },
  { value: 'CREATE_DIR', label: '创建目录' },
  { value: 'UPLOAD', label: '文件上传' },
  { value: 'DOWNLOAD', label: '文件下载' },
  { value: 'DELETE_FILE', label: '删除文件' },
  { value: 'DELETE_DIR', label: '删除目录' },
  { value: 'SHARE_CREATE', label: '创建分享' },
  { value: 'SHARE_ACCESS', label: '访问分享' },
  { value: 'SHARE_DELETE', label: '删除分享' },
  { value: 'CHANGE_OWNER', label: '更改所有者' },
  { value: 'USER_CREATE', label: '创建用户' },
  { value: 'USER_UPDATE', label: '更新用户' },
  { value: 'USER_DELETE', label: '删除用户' },
  { value: 'MOVE', label: '移动文件' },
]

function selectAll() {
  form.value.enabled_log_types = allLogTypes.map(t => t.value)
}

function deselectAll() {
  form.value.enabled_log_types = []
}

onMounted(async () => {
  loading.value = true
  try {
    const res = await api.get('/api/admin/config/log')
    Object.assign(form.value, res.data)
    if (!form.value.enabled_log_types) form.value.enabled_log_types = []
  } catch { /* ignore */ } finally {
    loading.value = false
  }
})

async function handleSave() {
  saving.value = true
  try {
    await api.put('/api/admin/config/log', form.value)
    message.success('保存成功')
  } catch (err: any) {
    message.error(err.response?.data?.error || '保存失败')
  } finally {
    saving.value = false
  }
}
</script>
