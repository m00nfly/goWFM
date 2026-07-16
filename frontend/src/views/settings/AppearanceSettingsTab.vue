<template>
  <div class="workspace-form-scroll settings-tab-scroll">
    <n-spin :show="loading">
	  <n-form class="settings-tab-form" label-placement="left" label-width="160px" :show-feedback="false" :model="form">
        <section class="settings-section">
          <header class="settings-section-header">
            <h2>品牌与主题</h2>
            <p>维护登录页品牌资源与系统默认外观</p>
          </header>
          <div class="settings-section-body">
          <n-form-item label="自定义Logo">
            <n-upload
              :max="1"
              accept=".png,.jpg,.jpeg,.svg"
              :default-upload="false"
              @change="handleLogoChange"
            >
              <n-button size="small">上传Logo</n-button>
            </n-upload>
            <span class="workspace-inline-note">PNG/JPG/SVG, 最大200KB</span>
            <img v-if="form.custom_logo" :src="form.custom_logo" class="logo-preview" />
          </n-form-item>
          <n-form-item label="登录背景图URL">
            <n-input v-model:value="form.login_bg_url" placeholder="留空使用默认背景，支持外部URL" clearable />
          </n-form-item>
          <n-form-item label="默认主题">
            <n-radio-group v-model:value="form.default_theme">
              <n-radio-button value="light">浅色</n-radio-button>
              <n-radio-button value="dark">深色</n-radio-button>
            </n-radio-group>
          </n-form-item>
          <n-form-item label="主题色">
            <n-color-picker
              v-model:value="form.theme_color"
              :swatches="presetColors"
              style="width: 200px"
            />
          </n-form-item>
          </div>
        </section>

        <section class="settings-section">
          <header class="settings-section-header">
            <h2>Web 服务</h2>
            <p>配置服务端口、HTTPS 与 TLS 证书</p>
          </header>
          <div class="settings-section-body">
          <n-form-item label="Web端口">
            <n-input-number v-model:value="form.server_port" :min="1" :max="65535" style="width: 150px" />
          </n-form-item>
          <n-form-item label="启用HTTPS">
            <n-switch v-model:value="form.enable_https" />
          </n-form-item>
          <template v-if="form.enable_https">
            <n-form-item label="SSL证书">
              <n-input v-model:value="form.ssl_cert" type="textarea" :rows="3" placeholder="粘贴PEM格式证书内容，或留空使用自签名证书" />
            </n-form-item>
            <n-form-item label="SSL私钥">
              <n-input v-model:value="form.ssl_key" type="textarea" :rows="3" placeholder="粘贴PEM格式私钥内容" />
            </n-form-item>
            <n-alert type="info" class="settings-alert">
              若不上传证书，系统将自动生成自签名证书（仅适用于内网/测试环境）。
            </n-alert>
          </template>

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
import { NForm, NFormItem, NInput, NInputNumber, NButton, NSwitch, NUpload, NRadioGroup, NRadioButton, NColorPicker, NAlert, NSpin, useMessage } from 'naive-ui'
import type { UploadFileInfo } from 'naive-ui'
import api from '@/api'

const message = useMessage()
const loading = ref(false)
const saving = ref(false)
const form = ref({
  login_bg_url: '',
  default_theme: 'light',
  theme_color: '#3B82F6',
  custom_logo: '',
  server_port: 8080,
  enable_https: false,
  ssl_cert: '',
  ssl_key: '',
})

const presetColors = [
  '#3B82F6', '#10B981', '#F59E0B', '#EF4444', '#8B5CF6',
  '#EC4899', '#06B6D4', '#6366F1', '#14B8A6', '#F97316',
]

function handleLogoChange({ file }: { file: UploadFileInfo }) {
  if (!file.file) return
  if (file.file.size > 200 * 1024) {
    message.error('Logo文件不能超过200KB')
    return
  }
  const reader = new FileReader()
  reader.onload = (e) => {
    form.value.custom_logo = e.target?.result as string
  }
  reader.readAsDataURL(file.file)
}

onMounted(async () => {
  loading.value = true
  try {
    const res = await api.get('/api/admin/config/appearance')
    form.value.login_bg_url = res.data.login_bg_url || ''
    form.value.default_theme = res.data.default_theme || 'light'
    form.value.theme_color = res.data.theme_color || '#3B82F6'
    form.value.custom_logo = res.data.custom_logo || ''
    form.value.server_port = res.data.server_port || 8080
    form.value.enable_https = res.data.enable_https || false
    // ssl_cert/key 不从服务端回显
  } catch { /* ignore */ } finally {
    loading.value = false
  }
})

async function handleSave() {
  saving.value = true
  try {
    const res = await api.put('/api/admin/config/appearance', form.value)
    message.success('保存成功')
    if (res.data.restart_required) {
      message.warning('端口或HTTPS配置变更需要重启服务后生效', { duration: 5000 })
    }
  } catch (err: any) {
    message.error(err.response?.data?.error || '保存失败')
  } finally {
    saving.value = false
    //立即刷新页面
    window.location.reload()
  }
}
</script>

<style scoped>
.logo-preview {
  height: 40px;
  margin-left: 12px;
  border-radius: var(--workspace-radius-sm);
}

.settings-alert {
  margin-bottom: 12px;
}
</style>
