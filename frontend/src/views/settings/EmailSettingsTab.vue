<template>
  <div class="workspace-form-scroll settings-tab-scroll">
    <n-spin :show="loading">
	  <n-form class="settings-tab-form" label-placement="left" label-width="160px" :show-feedback="false" :model="form">
        <section class="settings-section">
          <header class="settings-section-header smtp-heading">
            <div>
              <h2>SMTP 服务</h2>
              <p>用于发送密码找回及后续系统通知邮件</p>
            </div>
            <div class="test-action">
              <transition name="test-feedback">
                <div
                  v-if="testResult"
                  class="test-result"
                  :class="`is-${testResult.type}`"
                  role="status"
                  :aria-live="testResult.type === 'error' ? 'assertive' : 'polite'"
                >
                  <component :is="testResult.type === 'success' ? CheckmarkCircleOutline : AlertCircleOutline" class="test-result-icon" />
                  <div class="test-result-copy">
                    <strong>{{ testResult.title }}</strong>
                    <span>{{ testResult.detail }}</span>
                  </div>
                </div>
              </transition>
              <n-button type="primary" secondary :loading="testing" :disabled="saving" @click="handleTest">邮件测试</n-button>
            </div>
          </header>
          <div class="settings-section-body">
            <n-form-item label="SMTP 服务器" required>
              <n-input v-model:value="form.smtp_host" placeholder="如 smtp.example.com" />
            </n-form-item>
            <n-form-item label="SMTP 端口" required>
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
                :placeholder="hasPassword ? '已设置，留空不修改' : '请输入 SMTP 密码'"
              />
            </n-form-item>
			<n-form-item label="发件人名称">
			  <n-input v-model:value="form.sender_name" placeholder="留空时自动使用站点名称" clearable />
            </n-form-item>
            <n-form-item label="发件人 Email" required>
              <n-input v-model:value="form.sender_email" placeholder="如 noreply@example.com" />
            </n-form-item>
            <n-form-item label="启用 TLS">
              <n-switch v-model:value="form.enable_tls" />
              <span class="workspace-inline-note">465 端口使用隐式 TLS，其他端口使用 STARTTLS</span>
            </n-form-item>
            <n-form-item label="跳过 TLS 验证">
              <n-switch v-model:value="form.skip_tls_verify" />
              <span class="workspace-inline-note">仅用于可信的自签名 SMTP 服务器</span>
            </n-form-item>
          </div>
        </section>

        <section class="settings-section template-section">
          <header class="settings-section-header template-heading">
            <div>
              <h2>邮件模板</h2>
              <p>当前支持重置密码通知，模板结构已预留更多通知类型</p>
            </div>
            <n-select v-model:value="selectedTemplate" :options="templateOptions" class="template-select" />
          </header>
          <div class="template-variables" aria-label="可用模板变量">
			<span class="template-variables-label">可用变量</span>
			<n-tooltip v-for="item in templateVariables" :key="item.key" placement="top">
			  <template #trigger>
                <button
                  type="button"
                  class="template-variable"
                  :aria-label="`复制变量 ${item.key}：${item.hint}`"
                  @click="copyTemplateVariable(item)"
                >
                  <code>{{ item.key }}</code>
                </button>
              </template>
              {{ item.label }}：{{ item.hint }}，点击复制
            </n-tooltip>
          </div>
          <div class="template-workspace">
            <div class="template-editor">
              <n-form-item label="邮件主题" label-placement="top" required>
                <n-input v-model:value="activeTemplate.subject" placeholder="重置您的 {{.SiteName}} 密码" />
              </n-form-item>
              <n-form-item label="HTML 内容" label-placement="top" required>
                <n-input
                  v-model:value="activeTemplate.html"
                  type="textarea"
                  :autosize="{ minRows: 18, maxRows: 30 }"
                  class="html-editor"
                  placeholder="请输入完整 HTML 邮件模板"
                  spellcheck="false"
                />
              </n-form-item>
            </div>
            <div class="template-preview-wrap">
              <div class="preview-toolbar">
                <strong>实时预览</strong>
                <span>{{ previewSubject }}</span>
              </div>
              <iframe
                class="template-preview"
                title="邮件模板实时预览"
                sandbox=""
                referrerpolicy="no-referrer"
                :srcdoc="previewHTML"
              />
            </div>
          </div>
        </section>

        <footer class="settings-tab-actions">
          <n-button type="primary" :loading="saving" :disabled="testing" @click="handleSave">保存设置</n-button>
        </footer>
      </n-form>
    </n-spin>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { NButton, NForm, NFormItem, NInput, NInputNumber, NSelect, NSpin, NSwitch, NTooltip, useMessage } from 'naive-ui'
import { AlertCircleOutline, CheckmarkCircleOutline } from '@vicons/ionicons5'
import api from '@/api'

type Template = { subject: string; html: string }
type TestResult = { type: 'success' | 'error'; title: string; detail: string }

const message = useMessage()
const loading = ref(false)
const saving = ref(false)
const testing = ref(false)
const hasPassword = ref(false)
const testResult = ref<TestResult | null>(null)
const lastSaveError = ref('')
const selectedTemplate = ref('reset_password')
const templateOptions = [{ label: '重置密码', value: 'reset_password' }]
const templateVariables = [
	{ key: '{{.SiteName}}', label: '站点名称', hint: '插入系统设置中的站点名称' },
	{ key: '{{.Username}}', label: '用户名', hint: '插入接收邮件用户的登录名称' },
	{ key: '{{.ResetURL}}', label: '重置链接', hint: '插入一次性密码重置链接' },
	{ key: '{{.ExpiresMinutes}}', label: '有效分钟数', hint: '插入重置链接的有效时间' },
]

const form = ref({
  smtp_host: '',
  smtp_port: 587,
  smtp_username: '',
  smtp_password: '',
  enable_tls: true,
  skip_tls_verify: false,
	sender_name: '',
  sender_email: '',
  templates: {
    reset_password: { subject: '', html: '' } as Template,
  } as Record<string, Template>,
})

const activeTemplate = computed(() => form.value.templates[selectedTemplate.value] || form.value.templates.reset_password)
const previewValues: Record<string, string> = {
  SiteName: 'goWFM',
  Username: 'moon',
  ResetURL: 'https://example.com/login?reset_token=preview',
  ExpiresMinutes: '15',
}

function renderPreview(value: string) {
  return value.replace(/\{\{\s*\.([A-Za-z]+)\s*\}\}/g, (match, key) => previewValues[key] ?? match)
}

async function copyTemplateVariable(item: typeof templateVariables[number]) {
	try {
	  await navigator.clipboard.writeText(item.key)
	  message.success(`${item.key} 已复制`)
	} catch {
	  message.error('复制失败，请手动复制变量')
	}
}

const previewSubject = computed(() => renderPreview(activeTemplate.value.subject) || '未设置邮件主题')
const previewHTML = computed(() => {
  const content = renderPreview(activeTemplate.value.html)
  const csp = `<meta http-equiv="Content-Security-Policy" content="default-src 'none'; img-src data: https: http:; style-src 'unsafe-inline'; font-src data:">`
	if (!content) return `<!doctype html><html><head>${csp}</head><body style="font-family:sans-serif;padding:24px;color:#667085">输入 HTML 后将在此处实时预览</body></html>`
	if (/<head(\s[^>]*)?>/i.test(content)) return content.replace(/<head(\s[^>]*)?>/i, match => `${match}${csp}`)
	if (/<html(\s[^>]*)?>/i.test(content)) return content.replace(/<html(\s[^>]*)?>/i, match => `${match}<head>${csp}</head>`)
	return `<!doctype html><html><head>${csp}</head><body>${content}</body></html>`
})

onMounted(async () => {
  loading.value = true
  try {
    const res = await api.get('/api/admin/config/email')
    Object.assign(form.value, res.data)
    if (!form.value.templates?.reset_password) {
      form.value.templates = { reset_password: { subject: '', html: '' } }
    }
    hasPassword.value = res.data.has_password || false
    form.value.smtp_password = ''
  } catch (err: any) {
    message.error(err.response?.data?.error || '读取邮件设置失败')
  } finally {
    loading.value = false
  }
})

function validateForm(showErrorToast = true) {
	if (!form.value.smtp_host.trim() || !form.value.sender_email.trim()) {
		lastSaveError.value = '请填写 SMTP 服务器和发件人 Email'
		if (showErrorToast) message.warning(lastSaveError.value)
    return false
  }
  if (!/^\S+@\S+\.\S+$/.test(form.value.sender_email)) {
		lastSaveError.value = '请输入有效的发件人 Email'
		if (showErrorToast) message.warning(lastSaveError.value)
    return false
  }
  if (!activeTemplate.value.subject.trim() || !activeTemplate.value.html.trim()) {
		lastSaveError.value = '邮件主题与 HTML 内容不能为空'
		if (showErrorToast) message.warning(lastSaveError.value)
    return false
  }
	lastSaveError.value = ''
  return true
}

async function saveSettings(showSuccess = true, showErrorToast = true) {
  if (!validateForm(showErrorToast)) return false
  saving.value = true
  try {
    const payload = { ...form.value, templates: { ...form.value.templates } }
    if (!payload.smtp_password && hasPassword.value) delete (payload as any).smtp_password
    await api.put('/api/admin/config/email', payload)
    if (form.value.smtp_password) {
      hasPassword.value = true
      form.value.smtp_password = ''
    }
    if (showSuccess) message.success('邮件设置已保存')
    return true
  } catch (err: any) {
		lastSaveError.value = err.response?.data?.error || '保存失败'
		if (showErrorToast) message.error(lastSaveError.value)
    return false
  } finally {
    saving.value = false
  }
}

async function handleSave() {
  await saveSettings()
}

async function handleTest() {
	testResult.value = null
  testing.value = true
  try {
		if (!await saveSettings(false, false)) {
		  testResult.value = { type: 'error', title: '设置保存失败', detail: lastSaveError.value || '请检查当前邮件设置' }
		  return
		}
		const res = await api.post('/api/admin/email/test', {})
		const recipient = res.data?.recipient || form.value.sender_email
		testResult.value = {
		  type: 'success',
		  title: '发送成功',
		  detail: recipient ? `测试邮件已发送至 ${recipient}` : (res.data?.message || '测试邮件已发送'),
		}
  } catch (err: any) {
		const data = err.response?.data || {}
		const serverReply = data.smtp_code
		  ? `SMTP ${data.smtp_code}${data.smtp_message ? ` ${data.smtp_message}` : ''}`
		  : ''
		const errorDetail = data.error || err.message || '测试邮件发送失败'
		testResult.value = {
		  type: 'error',
		  title: '发送失败',
		  detail: serverReply ? `${errorDetail}；服务器响应：${serverReply}` : errorDetail,
		}
  } finally {
    testing.value = false
  }
}
</script>

<style scoped>
.template-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
}

.template-select {
  width: 180px;
  flex: 0 0 auto;
}

.template-variables {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
	padding: 14px 24px 16px;
  color: var(--workspace-text-muted);
	background: color-mix(in srgb, var(--workspace-accent) 5%, var(--workspace-surface));
  font-size: 13px;
}

.template-variables-label {
	margin-right: 2px;
	font-weight: 650;
	white-space: nowrap;
}

.template-variable {
	position: relative;
	min-height: 32px;
	display: inline-flex;
	align-items: center;
	padding: 0 9px;
	border: 1px solid color-mix(in srgb, var(--workspace-accent) 32%, transparent);
	border-radius: 8px;
	color: var(--workspace-accent);
	background: color-mix(in srgb, var(--workspace-accent) 10%, var(--workspace-surface));
	font: inherit;
	cursor: pointer;
	transition-property: color, background-color, border-color, box-shadow, transform;
	transition-duration: 160ms;
	transition-timing-function: ease;
}

.template-variable::after {
	position: absolute;
	inset: -4px 0;
	content: '';
}

.template-variable code {
	color: inherit;
	font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', monospace;
	font-size: 12px;
	font-weight: 700;
}

.template-variable:hover {
	border-color: color-mix(in srgb, var(--workspace-accent) 52%, transparent);
	background: color-mix(in srgb, var(--workspace-accent) 16%, var(--workspace-surface));
	transform: translateY(-1px);
}

.template-variable:active {
	transform: scale(0.96);
}

.template-variable:focus-visible {
	outline: none;
	box-shadow: 0 0 0 3px color-mix(in srgb, var(--workspace-accent) 22%, transparent);
}

.template-workspace {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(320px, 0.9fr);
  gap: 18px;
  padding: 0 24px 24px;
}

.template-editor :deep(.n-form-item) {
  display: block;
}

.html-editor :deep(textarea) {
  font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', monospace;
  font-size: 12px;
  line-height: 1.65;
  tab-size: 2;
}

.template-preview-wrap {
  min-height: 460px;
  overflow: hidden;
  border: 1px solid var(--workspace-border);
  border-radius: var(--workspace-radius-lg);
  background: #fff;
  box-shadow: 0 10px 28px rgba(25, 49, 83, 0.08);
}

.preview-toolbar {
  min-height: 56px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 3px;
  padding: 9px 14px;
  color: #172033;
  background: #f7f9fc;
  border-bottom: 1px solid rgba(23, 32, 51, 0.1);
}

.preview-toolbar span {
  max-width: 100%;
  overflow: hidden;
  color: #667085;
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.template-preview {
  display: block;
  width: 100%;
  height: 500px;
  border: 0;
  background: #fff;
}

.smtp-heading {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 18px;
}

.test-action {
	min-width: 0;
	display: flex;
	align-items: center;
	justify-content: flex-end;
	gap: 12px;
}

.test-action > .n-button {
	min-height: 40px;
	flex: 0 0 auto;
}

.test-result {
	min-width: 0;
	max-width: 540px;
	display: flex;
	align-items: flex-start;
	gap: 8px;
	padding: 8px 10px;
	border-radius: var(--workspace-radius-md);
	font-size: 12px;
	line-height: 1.45;
}

.test-result.is-success {
	color: #166534;
	background: rgba(22, 163, 74, 0.1);
}

.test-result.is-error {
	color: #b42318;
	background: rgba(208, 48, 80, 0.1);
}

html.dark .test-result.is-success {
	color: #86efac;
}

html.dark .test-result.is-error {
	color: #fda4af;
}

.test-result-icon {
	width: 18px;
	height: 18px;
	margin-top: 1px;
	flex: 0 0 auto;
}

.test-result-copy {
	min-width: 0;
	display: flex;
	flex-direction: column;
	gap: 2px;
}

.test-result-copy strong {
	font-weight: 760;
}

.test-result-copy span {
	overflow-wrap: anywhere;
	text-wrap: pretty;
}

.test-feedback-enter-active,
.test-feedback-leave-active {
	transition-property: opacity, transform, filter;
	transition-duration: 180ms;
	transition-timing-function: cubic-bezier(0.2, 0, 0, 1);
}

.test-feedback-enter-from,
.test-feedback-leave-to {
	opacity: 0;
	filter: blur(4px);
	transform: translateY(4px) scale(0.98);
}

@media (max-width: 980px) {
  .template-workspace {
    grid-template-columns: 1fr;
  }

	.smtp-heading {
		align-items: stretch;
		flex-direction: column;
	}

	.test-action {
		justify-content: flex-start;
	}
}

@media (max-width: 640px) {
	.template-heading,
	.smtp-heading {
    display: block;
  }

  .template-select {
    width: 100%;
    margin-top: 12px;
  }

  .template-variables,
	.template-workspace {
    padding-right: 16px;
    padding-left: 16px;
  }

	.test-action {
		align-items: stretch;
		flex-direction: column;
		margin-top: 12px;
	}

	.test-action > .n-button {
    min-height: 40px;
  }
}
</style>
