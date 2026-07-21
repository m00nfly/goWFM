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
          <n-form-item label="品牌 Logo" class="logo-form-item">
            <div class="logo-editor">
              <div class="logo-preview-panel">
                <div class="logo-preview-stage">
                  <img
                    v-if="form.custom_logo"
                    :src="form.custom_logo"
                    class="logo-preview"
                    alt="当前品牌 Logo 预览"
                  />
                  <div v-else class="default-logo-preview" aria-label="默认 Logo 预览">
                    <n-icon :size="24"><FolderOpenOutline /></n-icon>
                    <span>goWFM</span>
                  </div>
                </div>
                <div class="logo-preview-copy">
                  <strong>{{ form.custom_logo ? '当前自定义 Logo' : '默认 Logo' }}</strong>
                  <span>横版与方形图片都会保持原始比例展示</span>
                </div>
              </div>
              <div class="logo-controls">
                <n-upload
                  v-model:file-list="logoFileList"
                  :max="1"
                  accept=".png,.jpg,.jpeg,.svg"
                  :default-upload="false"
                  :show-file-list="false"
                  @change="handleLogoChange"
                >
                  <n-button secondary>
                    <template #icon><n-icon><CloudUploadOutline /></n-icon></template>
                    选择图片
                  </n-button>
                </n-upload>
                <n-button secondary :disabled="!form.custom_logo" @click="resetLogo">
                  <template #icon><n-icon><RefreshOutline /></n-icon></template>
                  恢复默认
                </n-button>
              </div>
              <p class="logo-help">支持 PNG、JPG、SVG，文件大小不超过 200 KB。建议使用透明背景图片。</p>
            </div>
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
            <h2>登录页品牌面板</h2>
            <p>使用 Markdown 或安全 HTML 自定义 Logo 下方的品牌内容</p>
          </header>
          <div class="settings-section-body brand-panel-settings">
            <n-form-item label="启用自定义内容">
              <n-switch v-model:value="form.custom_brand_panel_enabled" />
              <span class="workspace-inline-note">关闭时显示系统默认品牌介绍，已填写内容会保留</span>
            </n-form-item>
            <n-form-item label="面板内容" class="brand-content-form-item">
              <div class="brand-content-workspace">
                <div class="brand-content-editor">
                  <n-input
                    v-model:value="form.custom_brand_panel_content"
                    type="textarea"
                    :disabled="!form.custom_brand_panel_enabled"
                    :autosize="{ minRows: 10, maxRows: 20 }"
                    :maxlength="102400"
                    show-count
                    placeholder="# 团队文件中心&#10;&#10;在这里写入 Markdown，或直接粘贴安全 HTML。"
                    spellcheck="false"
                  />
                  <p>支持标题、段落、列表、链接、图片、引用和代码。脚本、表单、内联样式及危险属性会被自动移除。</p>
                </div>
                <div class="brand-content-preview" :class="{ 'is-disabled': !form.custom_brand_panel_enabled }">
                  <span class="brand-content-preview-label">登录页预览</span>
                  <div v-if="brandPanelPreview" class="brand-content-prose" v-html="brandPanelPreview"></div>
                  <div v-else class="brand-content-empty">输入内容后在此处预览</div>
                </div>
              </div>
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
import { computed, ref, onMounted } from 'vue'
import { NForm, NFormItem, NInput, NInputNumber, NButton, NIcon, NSwitch, NUpload, NRadioGroup, NRadioButton, NColorPicker, NAlert, NSpin, useMessage } from 'naive-ui'
import type { UploadFileInfo } from 'naive-ui'
import { CloudUploadOutline, FolderOpenOutline, RefreshOutline } from '@vicons/ionicons5'
import api from '@/api'
import { renderBrandPanelContent } from '@/utils/brandPanel'

const message = useMessage()
const loading = ref(false)
const saving = ref(false)
const logoFileList = ref<UploadFileInfo[]>([])
const form = ref({
  login_bg_url: '',
  default_theme: 'light',
  theme_color: '#3B82F6',
  custom_logo: '',
  custom_brand_panel_enabled: false,
  custom_brand_panel_content: '',
  server_port: 8080,
  enable_https: false,
  ssl_cert: '',
  ssl_key: '',
})

const presetColors = [
  '#3B82F6', '#10B981', '#F59E0B', '#EF4444', '#8B5CF6',
  '#EC4899', '#06B6D4', '#6366F1', '#14B8A6', '#F97316',
]

const brandPanelPreview = computed(() => renderBrandPanelContent(form.value.custom_brand_panel_content))

function handleLogoChange({ file }: { file: UploadFileInfo }) {
  if (!file.file) return
  const extension = file.name.toLowerCase().split('.').pop()
  if (!extension || !['png', 'jpg', 'jpeg', 'svg'].includes(extension)) {
    logoFileList.value = []
    message.error('请选择 PNG、JPG 或 SVG 图片')
    return
  }
  if (file.file.size > 200 * 1024) {
    logoFileList.value = []
    message.error('Logo文件不能超过200KB')
    return
  }
  const reader = new FileReader()
  reader.onload = (e) => {
    form.value.custom_logo = e.target?.result as string
  }
  reader.onerror = () => {
    logoFileList.value = []
    message.error('Logo 图片读取失败')
  }
  reader.readAsDataURL(file.file)
}

function resetLogo() {
  form.value.custom_logo = ''
  logoFileList.value = []
  message.info('已恢复默认 Logo，保存设置后生效')
}

onMounted(async () => {
  loading.value = true
  try {
    const res = await api.get('/api/admin/config/appearance')
    form.value.login_bg_url = res.data.login_bg_url || ''
    form.value.default_theme = res.data.default_theme || 'light'
    form.value.theme_color = res.data.theme_color || '#3B82F6'
    form.value.custom_logo = res.data.custom_logo || ''
    form.value.custom_brand_panel_enabled = res.data.custom_brand_panel_enabled === true
    form.value.custom_brand_panel_content = res.data.custom_brand_panel_content || ''
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
    window.location.reload()
  } catch (err: any) {
    message.error(err.response?.data?.error || '保存失败')
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.logo-form-item {
  grid-column: 1 / -1;
}

.logo-form-item :deep(.n-form-item-blank) {
  justify-content: stretch;
}

.logo-editor {
  width: 100%;
  min-width: 0;
}

.logo-preview-panel {
  display: grid;
  grid-template-columns: minmax(180px, 280px) minmax(0, 1fr);
  align-items: center;
  gap: 14px;
}

.logo-preview-stage {
  width: 100%;
  height: 92px;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  padding: 14px;
  border: 1px solid var(--workspace-border-soft);
  border-radius: var(--workspace-radius-md);
  background:
    linear-gradient(135deg, rgba(var(--workspace-accent-rgb), 0.055), transparent),
    var(--workspace-surface-soft);
}

.logo-preview {
  display: block;
  width: auto;
  height: auto;
  max-width: 100%;
  max-height: 64px;
  object-fit: contain;
}

.default-logo-preview {
  display: inline-flex;
  align-items: center;
  gap: 9px;
  color: var(--workspace-text);
  font-size: 18px;
  font-weight: 760;
}

.default-logo-preview .n-icon {
  color: var(--workspace-accent);
}

.logo-preview-copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.logo-preview-copy strong {
  color: var(--workspace-text);
  font-size: 13px;
}

.logo-preview-copy span,
.logo-help {
  color: var(--workspace-text-muted);
  font-size: 12px;
  line-height: 1.5;
  text-wrap: pretty;
}

.logo-controls {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 12px;
}

.logo-help {
  margin: 8px 0 0;
}

.settings-alert {
  margin-bottom: 12px;
}

.brand-content-form-item {
  grid-column: 1 / -1;
}

.brand-content-form-item :deep(.n-form-item-blank) {
  justify-content: stretch;
}

.brand-content-workspace {
  width: 100%;
  min-width: 0;
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(280px, 0.8fr);
  gap: 14px;
}

.brand-content-editor {
  min-width: 0;
}

.brand-content-editor > p {
  margin: 8px 0 0;
  color: var(--workspace-text-muted);
  font-size: 12px;
  line-height: 1.5;
  text-wrap: pretty;
}

.brand-content-preview {
  min-height: 248px;
  overflow: auto;
  padding: 16px;
  border: 1px solid var(--workspace-border-soft);
  border-radius: var(--workspace-radius-md);
  background: var(--workspace-surface-soft);
  transition-property: opacity, border-color;
  transition-duration: 180ms;
}

.brand-content-preview.is-disabled {
  opacity: 0.58;
}

.brand-content-preview-label {
  display: block;
  margin-bottom: 14px;
  color: var(--workspace-text-soft);
  font-size: 11px;
  font-weight: 700;
}

.brand-content-empty {
  min-height: 170px;
  display: grid;
  place-items: center;
  color: var(--workspace-text-soft);
  font-size: 12px;
}

.brand-content-prose {
  color: var(--workspace-text);
  font-size: 13px;
  line-height: 1.65;
  overflow-wrap: anywhere;
}

.brand-content-prose :deep(h1),
.brand-content-prose :deep(h2),
.brand-content-prose :deep(h3) {
  margin: 0 0 10px;
  color: var(--workspace-text);
  line-height: 1.25;
  text-wrap: balance;
}

.brand-content-prose :deep(h1) { font-size: 22px; }
.brand-content-prose :deep(h2) { font-size: 18px; }
.brand-content-prose :deep(h3) { font-size: 15px; }

.brand-content-prose :deep(p),
.brand-content-prose :deep(ul),
.brand-content-prose :deep(ol),
.brand-content-prose :deep(blockquote) {
  margin: 0 0 10px;
}

.brand-content-prose :deep(ul),
.brand-content-prose :deep(ol) {
  padding-left: 20px;
}

.brand-content-prose :deep(a) {
  color: var(--workspace-accent);
}

.brand-content-prose :deep(img) {
  display: block;
  max-width: 100%;
  height: auto;
  border-radius: var(--workspace-radius-sm);
  outline: 1px solid rgba(0, 0, 0, 0.1);
  outline-offset: -1px;
}

:global(html.dark) .brand-content-prose :deep(img) {
  outline-color: rgba(255, 255, 255, 0.1);
}

.brand-content-prose :deep(code) {
  padding: 2px 5px;
  border-radius: 4px;
  background: var(--workspace-surface-strong);
  font-size: 0.92em;
}

@media (max-width: 640px) {
  .logo-preview-panel {
    grid-template-columns: 1fr;
  }

  .logo-preview-stage {
    max-width: none;
  }

  .brand-content-workspace {
    grid-template-columns: 1fr;
  }
}
</style>
