<template>
  <div class="workspace-form-scroll settings-tab-scroll">
    <n-spin :show="loading">
      <n-form class="settings-tab-form" label-placement="left" label-width="160px" :model="form">
        <section class="settings-section">
          <header class="settings-section-header">
            <h2>基础参数</h2>
            <p>配置站点身份、访问地址与文件存储限制</p>
          </header>
          <div class="settings-section-body">
            <n-form-item label="站点名称">
              <n-input v-model:value="form.site_name" placeholder="请输入站点名称" />
            </n-form-item>
            <n-form-item label="站点链接">
              <n-input v-model:value="form.site_link" placeholder="如 https://example.com" />
            </n-form-item>
            <n-form-item label="分享目录路径">
              <n-input v-model:value="form.data_root_path" placeholder="文件存储的绝对路径" />
            </n-form-item>
            <n-form-item label="最大上传大小">
              <n-input-number v-model:value="uploadSizeMB" :min="1" :max="102400" style="width: 180px" />
              <span class="workspace-inline-note">MB</span>
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
import { ref, computed, onMounted } from 'vue'
import { NForm, NFormItem, NInput, NInputNumber, NButton, NSpin, useMessage } from 'naive-ui'
import api from '@/api'

const message = useMessage()
const loading = ref(false)
const saving = ref(false)
const form = ref({
  site_name: '',
  site_link: '',
  data_root_path: '',
  max_upload_size: 1073741824,
})

const uploadSizeMB = computed({
  get: () => Math.round(form.value.max_upload_size / 1024 / 1024),
  set: (v: number) => { form.value.max_upload_size = v * 1024 * 1024 },
})

onMounted(async () => {
  loading.value = true
  try {
    const res = await api.get('/api/admin/config/basic')
    Object.assign(form.value, res.data)
  } catch { /* ignore */ } finally {
    loading.value = false
  }
})

async function handleSave() {
  saving.value = true
  try {
    const res = await api.put('/api/admin/config/basic', form.value)
    message.success('保存成功')
    if (res.data.restart_required) {
      message.warning('部分配置需要重启服务后生效', { duration: 5000 })
    }
  } catch (err: any) {
    message.error(err.response?.data?.error || '保存失败')
  } finally {
    saving.value = false
  }
}
</script>
