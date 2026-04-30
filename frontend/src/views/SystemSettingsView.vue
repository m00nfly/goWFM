<template>
  <n-card :bordered="false">
    <n-form label-placement="left" label-width="auto">
      <n-divider>站点信息</n-divider>
      <n-form-item label="站点名称">
        <n-input v-model:value="form.org_name" placeholder="未设置" disabled />
      </n-form-item>
      <n-form-item label="站点链接">
        <n-input v-model:value="form.org_link" placeholder="未设置" disabled />
      </n-form-item>

      <n-divider>服务配置</n-divider>
      <n-form-item label="数据存储路径">
        <n-input :value="configInfo.data_root_path || '（请在 config.json 中查看）'" disabled />
      </n-form-item>
      <n-form-item label="Web 端口">
        <n-input-number :value="configInfo.server_port || 8080" disabled />
      </n-form-item>
      <n-form-item label="日志级别">
        <n-select :value="configInfo.log_level || 'info'" :options="logLevelOptions" disabled />
      </n-form-item>
      <n-form-item label="最大上传大小">
        <n-input-number
          :value="configInfo.max_upload_size_mb || 1024"
          disabled
          style="width: 180px"
        />
        <span style="margin-left: 8px; color: #999; font-size: 13px">MB</span>
      </n-form-item>

      <n-divider>数据库</n-divider>
      <n-form-item label="数据库路径">
        <n-input :value="configInfo.db_path || 'gowfm.db'" disabled />
      </n-form-item>

      <n-alert type="info" style="margin-top: 16px">
        系统配置保存在 <code>config.json</code> 文件中，修改后需重启服务生效。
      </n-alert>
    </n-form>
  </n-card>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import {
  NCard, NForm, NFormItem, NInput, NInputNumber, NSelect, NDivider, NAlert, useMessage,
} from 'naive-ui'
import api from '@/api'

const message = useMessage()

const logLevelOptions = [
  { label: 'Debug', value: 'debug' },
  { label: 'Info', value: 'info' },
  { label: 'Warn', value: 'warn' },
  { label: 'Error', value: 'error' },
]

const form = reactive({
  org_name: '',
  org_link: '',
})

const configInfo = ref<Record<string, any>>({})

onMounted(async () => {
  try {
    const res = await api.get('/api/config/info')
    form.org_name = res.data.org_name || ''
    form.org_link = res.data.org_link || ''
    configInfo.value = res.data
  } catch (err: any) {
    message.error('获取系统配置失败')
  }
})
</script>
