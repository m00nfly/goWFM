<template>
  <div class="setup-page">
    <n-card title="系统初始化设置" class="setup-card">
      <n-form ref="formRef" :model="form" :rules="rules" label-placement="left" label-width="auto">
        <n-divider>管理员账户</n-divider>
        <n-form-item label="admin密码" path="admin_password">
          <n-input v-model:value="form.admin_password" type="password" placeholder="至少6位密码" />
        </n-form-item>

        <n-divider>系统配置</n-divider>
        <n-form-item label="站点名称" path="org_name">
          <n-input v-model:value="form.org_name" placeholder="可选，显示在页脚" />
        </n-form-item>
        <n-form-item label="站点链接" path="org_link">
          <n-input v-model:value="form.org_link" placeholder="可选，站点名称的超链接" />
        </n-form-item>
        <n-form-item label="数据存储路径" path="data_root_path">
          <n-input v-model:value="form.data_root_path" placeholder="/absolute/path/to/data" />
        </n-form-item>
        <n-form-item label="web端口" path="server_port">
          <n-input-number v-model:value="form.server_port" :min="1" :max="65535" />
        </n-form-item>
        <n-form-item label="Session密钥" path="session_secret">
          <n-input v-model:value="form.session_secret" placeholder="留空则自动生成" />
        </n-form-item>
        <n-form-item label="日志级别" path="log_level">
          <n-select v-model:value="form.log_level" :options="logLevelOptions" />
        </n-form-item>
        <n-form-item label="最大上传大小" path="max_upload_size_mb">
          <n-input-number
            v-model:value="form.max_upload_size_mb"
            :min="1"
            :max="102400"
            style="width: 180px"
          />
          <span style="margin-left: 8px; color: #999; font-size: 13px">MB（默认 1024 MB）</span>
        </n-form-item>

        <n-space vertical :size="12">
          <n-button type="primary" block :loading="loading" @click="handleSubmit">保存配置</n-button>
        </n-space>
      </n-form>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { NCard, NForm, NFormItem, NInput, NInputNumber, NSelect, NButton, NSpace, NDivider, useMessage } from 'naive-ui'
import type { FormInst, FormRules } from 'naive-ui'
import api from '@/api'

const router = useRouter()
const message = useMessage()
const formRef = ref<FormInst | null>(null)
const loading = ref(false)

const logLevelOptions = [
  { label: 'Debug', value: 'debug' },
  { label: 'Info', value: 'info' },
  { label: 'Warn', value: 'warn' },
  { label: 'Error', value: 'error' },
]

const form = reactive({
  admin_password: '',
  org_name: '',
  org_link: '',
  data_root_path: '',
  server_port: 8080,
  session_secret: '',
  log_level: 'info',
  db_path: 'wfm.db',
  max_upload_size_mb: 1024,
})

const rules: FormRules = {
  admin_password: [
    { required: true, message: '请输入管理员密码', trigger: 'blur' },
    { min: 6, message: '密码至少6位', trigger: 'blur' },
  ],
  data_root_path: [
    { required: true, message: '请输入数据存储路径', trigger: 'blur' },
  ],
}

onMounted(async () => {
  try {
    const res = await api.get('/api/setup/status')
    if (!res.data.needs_setup) {
      router.replace('/login')
    }
  } catch {
    // server might not be ready
  }
})

async function handleSubmit() {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }
  loading.value = true
  try {
    await api.post('/api/setup', {
      ...form,
      max_upload_size: form.max_upload_size_mb * 1024 * 1024,
    })
    message.success('初始化完成！请登录管理员账户')
    router.replace('/login')
  } catch (err: any) {
    message.error(err.response?.data?.error || '初始化失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.setup-page {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background: #f5f5f5;
}
.setup-card {
  width: 520px;
}
</style>