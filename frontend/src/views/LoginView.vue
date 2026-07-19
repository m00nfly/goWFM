<template>
  <div class="login-page" :class="{ dark: themeStore.isDark, 'has-custom-bg': !!loginBgUrl }" :style="loginBgStyle">
    <main class="login-shell" aria-label="登录">
      <section class="brand-panel" :aria-label="orgName || 'goWFM'">
        <BrandIdentity
          :logo="customLogo"
          :name="orgName || 'goWFM'"
          kicker="私有文件工作台"
          variant="login"
        />

        <div class="brand-copy">
          <h1>回到你的安全文件空间</h1>
          <p>集中管理文件、分享链接和团队权限，继续处理今天的工作。</p>
        </div>

        <div class="visual-stage" aria-hidden="true">
          <div class="visual-card">
            <img :src="heroImage" class="hero-art" alt="" />
          </div>
          <div class="signal-card signal-primary">
            <ShieldCheckmarkOutline />
            <span>权限检查</span>
          </div>
          <div class="signal-card signal-secondary">
            <KeyOutline />
            <span>安全会话</span>
          </div>
        </div>

        <div class="brand-points" aria-label="平台能力">
          <div class="brand-point">
            <ShieldCheckmarkOutline />
            <span>自托管部署</span>
          </div>
          <div class="brand-point">
            <KeyOutline />
            <span>双重验证</span>
          </div>
          <div class="brand-point">
            <FolderOutline />
            <span>文件与分享</span>
          </div>
        </div>
      </section>

      <section class="auth-panel" aria-label="账号登录">
        <button
          class="theme-toggle"
          type="button"
          @click="themeStore.toggleTheme()"
          :title="themeStore.isDark ? '切换亮色' : '切换暗色'"
          :aria-label="themeStore.isDark ? '切换亮色' : '切换暗色'"
        >
          <MoonOutline v-if="!themeStore.isDark" class="toggle-icon" />
          <SunnyOutline v-else class="toggle-icon" />
        </button>

        <div class="auth-card">
          <div class="auth-header">
			<p class="auth-label">{{ authHeading.label }}</p>
			<h2>{{ authHeading.title }}</h2>
			<p>{{ authHeading.description }}</p>
          </div>

		  <form @submit.prevent="handleFormSubmit" class="login-form">
            <Transition name="auth-swap" mode="out-in">
			  <div v-if="passwordFlow === 'forgot'" key="forgot-password" class="form-panel">
				<template v-if="!forgotSent">
				  <div class="security-notice">
					<MailOutline />
					<span>输入账户绑定邮箱。无论账户是否存在，系统都会返回相同结果。</span>
				  </div>
				  <div class="input-group">
					<div class="label-row">
					  <label class="input-label" for="forgot-email">邮箱</label>
					  <span class="field-hint">账户绑定邮箱</span>
					</div>
					<div class="input-wrapper">
					  <MailOutline class="input-icon" />
					  <input id="forgot-email" ref="forgotEmailRef" v-model="forgotEmail" type="email" required
						placeholder="name@example.com" class="input-field" autocomplete="email" @keydown.enter.prevent="handleForgotPassword" />
					</div>
				  </div>
				  <div v-if="captchaEnabled" class="input-group">
					<div class="label-row">
					  <label class="input-label" for="forgot-captcha">验证码</label>
					  <span class="field-hint">点击图片刷新</span>
					</div>
					<div class="captcha-row">
					  <div class="input-wrapper captcha-input">
						<LockClosedOutline class="input-icon" />
						<input id="forgot-captcha" v-model="form.captcha_code" type="text" required placeholder="输入验证码"
						  class="input-field" autocomplete="off" @keydown.enter.prevent="handleForgotPassword" />
					  </div>
					  <button type="button" class="captcha-image" @click="refreshCaptcha" title="刷新验证码" aria-label="刷新验证码">
						<img v-if="captchaImage" :src="captchaImage" alt="验证码" />
						<span v-else>加载中...</span>
					  </button>
					</div>
				  </div>
				  <button type="button" class="login-btn" :disabled="forgotLoading" @click="handleForgotPassword">
					<span v-if="forgotLoading" class="spinner"></span>
					<span>{{ forgotLoading ? '提交中...' : '发送重置邮件' }}</span>
				  </button>
				</template>
				<div v-else class="request-complete">
				  <div class="request-complete-icon"><MailOutline /></div>
				  <strong>请检查邮箱</strong>
				  <p>如果该邮箱对应有效账户，重置邮件将在稍后送达。链接 15 分钟内有效。</p>
				</div>
				<button type="button" class="back-btn" @click="returnToLogin"><span aria-hidden="true">←</span>返回登录</button>
			  </div>

			  <div v-else-if="passwordFlow === 'reset'" key="reset-password" class="form-panel">
				<div v-if="resetChecking" class="reset-checking"><span class="spinner spinner-dark"></span><span>正在验证重置链接...</span></div>
				<template v-else-if="!resetError">
				  <div class="security-notice">
					<ShieldCheckmarkOutline />
					<span>{{ resetTOTPRequired ? '此账户已启用 TOTP，提交新密码时必须完成二次验证。' : '链接验证通过。重置后其他登录会话将自动失效。' }}</span>
				  </div>
				  <div class="input-group">
					<div class="label-row"><label class="input-label" for="reset-password">新密码</label><span class="field-hint">至少 6 位</span></div>
					<div class="input-wrapper">
					  <LockClosedOutline class="input-icon" />
					  <input id="reset-password" v-model="resetForm.new_password" :type="showResetPassword ? 'text' : 'password'" required
						placeholder="请输入新密码" class="input-field input-field-with-action" autocomplete="new-password" />
					  <button type="button" class="eye-btn" @click="showResetPassword = !showResetPassword" :aria-label="showResetPassword ? '隐藏密码' : '显示密码'">
						<EyeOffOutline v-if="showResetPassword" class="eye-icon" /><EyeOutline v-else class="eye-icon" />
					  </button>
					</div>
				  </div>
				  <div class="input-group">
					<div class="label-row"><label class="input-label" for="reset-confirm">确认新密码</label><span class="field-hint">再次输入</span></div>
					<div class="input-wrapper"><LockClosedOutline class="input-icon" />
					  <input id="reset-confirm" v-model="resetForm.confirm_password" :type="showResetPassword ? 'text' : 'password'" required
						placeholder="请再次输入新密码" class="input-field" autocomplete="new-password" @keydown.enter.prevent="handleResetPassword" />
					</div>
				  </div>
				  <div v-if="resetTOTPRequired" class="input-group">
					<div class="label-row"><label class="input-label" for="reset-totp">TOTP 验证码</label><span class="field-hint">当前 6 位数字</span></div>
					<div class="input-wrapper"><KeyOutline class="input-icon" />
					  <input id="reset-totp" v-model="resetForm.totp_code" type="text" required placeholder="例如 123456"
						class="input-field" autocomplete="one-time-code" inputmode="numeric" maxlength="6" @keydown.enter.prevent="handleResetPassword" />
					</div>
				  </div>
				  <button type="button" class="login-btn" :disabled="resetLoading" @click="handleResetPassword">
					<span v-if="resetLoading" class="spinner"></span><span>{{ resetLoading ? '重置中...' : '确认重置密码' }}</span>
				  </button>
				</template>
				<div v-else class="request-complete error-state">
				  <div class="request-complete-icon"><KeyOutline /></div><strong>链接不可用</strong><p>{{ resetError }}</p>
				</div>
				<button type="button" class="back-btn" @click="openForgotPassword"><span aria-hidden="true">←</span>重新申请重置链接</button>
			  </div>

			  <div v-else-if="passwordFlow === 'reset-success'" key="reset-success" class="form-panel">
				<div class="request-complete">
				  <div class="request-complete-icon"><ShieldCheckmarkOutline /></div><strong>密码重置成功</strong><p>所有旧登录会话已失效，请使用新密码重新登录。</p>
				</div>
				<button type="button" class="login-btn" @click="returnToLogin">返回登录</button>
			  </div>

			  <div v-else-if="!totpRequired && !totpSetupRequired" key="credentials" class="form-panel">
                <div class="input-group">
                  <div class="label-row">
                    <label class="input-label" for="login-username">账号</label>
                    <span class="field-hint">用户名或 Email</span>
                  </div>
                  <div class="input-wrapper">
                    <MailOutline class="input-icon" />
                    <input
                      id="login-username"
                      v-model="form.username"
                      type="text"
                      required
                      placeholder="请输入账号"
                      class="input-field"
                      autocomplete="username"
                    />
                  </div>
                </div>

                <div class="input-group">
                  <div class="label-row">
                    <label class="input-label" for="login-password">密码</label>
                    <span class="field-hint">区分大小写</span>
                  </div>
                  <div class="input-wrapper">
                    <LockClosedOutline class="input-icon" />
                    <input
                      id="login-password"
                      ref="passwordRef"
                      v-model="form.password"
                      :type="showPassword ? 'text' : 'password'"
                      required
                      placeholder="请输入密码"
                      class="input-field input-field-with-action"
                      autocomplete="current-password"
                    />
                    <button
                      type="button"
                      class="eye-btn"
                      @click="showPassword = !showPassword"
                      :aria-label="showPassword ? '隐藏密码' : '显示密码'"
                      :title="showPassword ? '隐藏密码' : '显示密码'"
                    >
                      <EyeOffOutline v-if="showPassword" class="eye-icon" />
                      <EyeOutline v-else class="eye-icon" />
                    </button>
                  </div>
                </div>

                <div v-if="captchaEnabled" class="input-group">
                  <div class="label-row">
                    <label class="input-label" for="login-captcha">验证码</label>
                    <span class="field-hint">点击图片刷新</span>
                  </div>
                  <div class="captcha-row">
                    <div class="input-wrapper captcha-input">
                      <LockClosedOutline class="input-icon" />
                      <input
                        id="login-captcha"
                        v-model="form.captcha_code"
                        type="text"
                        required
                        placeholder="输入验证码"
                        class="input-field"
                        autocomplete="off"
                      />
                    </div>
                    <button type="button" class="captcha-image" @click="refreshCaptcha" title="刷新验证码" aria-label="刷新验证码">
                      <img v-if="captchaImage" :src="captchaImage" alt="验证码" />
                      <span v-else>加载中...</span>
                    </button>
                  </div>
                </div>

				<div class="login-options-row">
				  <label class="remember-row">
					<input type="checkbox" v-model="rememberMe" class="checkbox" />
					<span>保持登录状态</span>
				  </label>
				  <button v-if="passwordResetEnabled" type="button" class="forgot-link" @click="openForgotPassword">忘记密码？</button>
				</div>

                <button type="submit" class="login-btn" :disabled="loading">
                  <span v-if="loading" class="spinner"></span>
                  <span>{{ loading ? '登录中...' : '登录' }}</span>
                </button>
              </div>

              <div v-else-if="totpRequired" key="totp" class="form-panel">
                <div class="totp-notice">
                  <ShieldCheckmarkOutline />
                  <span>你的账号已通过密码校验，请完成第二步。</span>
                </div>

                <div class="input-group">
                  <div class="label-row">
                    <label class="input-label" for="login-totp">验证码</label>
                    <span class="field-hint">6 位验证码或恢复码</span>
                  </div>
                  <div class="input-wrapper">
                    <LockClosedOutline class="input-icon" />
                    <input
                      id="login-totp"
                      ref="totpCodeRef"
                      v-model="totpCode"
                      type="text"
                      required
                      placeholder="例如 123456"
                      class="input-field"
                      autocomplete="one-time-code"
                      inputmode="numeric"
                      maxlength="10"
                      @keydown.enter.prevent="handleTOTPLogin"
                    />
                  </div>
                </div>

                <label class="remember-row">
                  <input type="checkbox" v-model="trustDevice" class="checkbox" />
                  <span>信任此设备，<span class="tabular-num">{{ trustDays }}</span> 天内无需再次验证</span>
                </label>

                <button type="button" class="login-btn" :disabled="totpLoading" @click="handleTOTPLogin">
                  <span v-if="totpLoading" class="spinner"></span>
                  <span>{{ totpLoading ? '验证中...' : '验证登录' }}</span>
                </button>

                <button type="button" class="back-btn" @click="totpRequired = false; totpCode = ''">
                  <span aria-hidden="true">←</span>
                  返回修改账号密码
                </button>
              </div>

              <div v-else key="totp-setup" class="form-panel">
                <div class="totp-notice">
                  <ShieldCheckmarkOutline />
                  <span>{{ totpSetupStep === 3 ? '新验证器绑定成功，请妥善保存恢复码。' : '账号密码验证成功，请完成新验证器绑定。' }}</span>
                </div>

                <template v-if="totpSetupStep === 1">
                  <div class="login-qr-wrap">
                    <img :src="totpSetupQr" alt="TOTP 绑定二维码" class="login-qr" />
                  </div>
                  <div class="input-group">
                    <div class="label-row">
                      <label class="input-label" for="login-totp-setup">Authenticator 验证码</label>
                      <span class="field-hint">6 位数字</span>
                    </div>
                    <div class="input-wrapper">
                      <LockClosedOutline class="input-icon" />
                      <input id="login-totp-setup" ref="totpCodeRef" v-model="totpCode" type="text"
                        required placeholder="例如 123456" class="input-field" autocomplete="one-time-code"
                        inputmode="numeric" maxlength="6" @keydown.enter.prevent="handleTOTPSetupLogin" />
                    </div>
                  </div>
                  <button type="button" class="login-btn" :disabled="totpLoading" @click="handleTOTPSetupLogin">
                    <span v-if="totpLoading" class="spinner"></span>
                    <span>{{ totpLoading ? '验证中...' : '验证并绑定' }}</span>
                  </button>
                  <button type="button" class="back-btn" @click="resetTOTPFlow"><span aria-hidden="true">←</span>返回修改账号密码</button>
                </template>

                <template v-else>
                  <div class="login-recovery-codes">
                    <code v-for="code in totpRecoveryCodes" :key="code">{{ code }}</code>
                  </div>
                  <p class="setup-secret-hint">每个恢复码只能使用一次，关闭后将无法再次查看。</p>
                  <button type="button" class="login-btn" @click="copyTOTPRecoveryCodes">复制恢复码</button>
                  <button type="button" class="back-btn" @click="finishTOTPSetupLogin">我已保存，进入系统</button>
                </template>
              </div>
            </Transition>
          </form>

          <div class="bottom-links">
            <template v-if="orgName">
              <a v-if="orgLink" :href="orgLink" target="_blank" rel="noopener noreferrer">{{ orgName }}</a>
              <span v-else>{{ orgName }}</span>
              <span class="sep" aria-hidden="true"></span>
            </template>
            <a href="https://gowfm.dev" target="_blank" rel="noopener noreferrer">goWFM 官网</a>
            <span class="sep" aria-hidden="true"></span>
            <a href="https://github.com/m00nfly/gowfm" target="_blank" rel="noopener noreferrer">GitHub</a>
            <template v-if="version">
              <span class="sep" aria-hidden="true"></span>
              <span class="version-text">版本 {{ version }}</span>
            </template>
          </div>
        </div>
      </section>
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import {
  FolderOutline,
  MailOutline,
  LockClosedOutline,
  EyeOutline,
  EyeOffOutline,
  SunnyOutline,
  MoonOutline,
  ShieldCheckmarkOutline,
  KeyOutline,
} from '@vicons/ionicons5'
import api from '@/api'
import { useUserStore } from '@/stores/user'
import { useThemeStore } from '@/stores/theme'
import { useConfig } from '@/composables/useConfig'
import BrandIdentity from '@/components/BrandIdentity.vue'
import heroImage from '@/assets/hero.png'

const router = useRouter()
const message = useMessage()
const userStore = useUserStore()
const themeStore = useThemeStore()
const { config, fetchConfig } = useConfig()
const loading = ref(false)
const showPassword = ref(false)
const showResetPassword = ref(false)
const passwordRef = ref<HTMLInputElement | null>(null)

type PasswordFlow = 'login' | 'forgot' | 'reset' | 'reset-success'
const passwordFlow = ref<PasswordFlow>('login')
const forgotEmail = ref('')
const forgotEmailRef = ref<HTMLInputElement | null>(null)
const forgotLoading = ref(false)
const forgotSent = ref(false)
const resetToken = ref('')
const resetChecking = ref(false)
const resetLoading = ref(false)
const resetTOTPRequired = ref(false)
const resetError = ref('')
const resetForm = reactive({ new_password: '', confirm_password: '', totp_code: '' })
const passwordResetEnabled = computed(() => config.value?.allow_email_password_reset === true && config.value?.email_active === true)

const orgName = ref('')
const orgLink = ref('')
const version = ref('')
const loginBgUrl = ref('')
const customLogo = ref('')
const captchaEnabled = ref(false)
const captchaImage = ref('')

const form = reactive({
  username: '',
  password: '',
  captcha_id: '',
  captcha_code: '',
})

const rememberMe = ref(false)

// TOTP 相关
const totpRequired = ref(false)
const totpSetupRequired = ref(false)
const totpSetupQr = ref('')
const totpSetupStep = ref(1)
const totpRecoveryCodes = ref<string[]>([])
const totpCode = ref('')
const totpLoading = ref(false)
const trustDevice = ref(false)
const trustDays = ref(30)
const loginToken = ref('')
const totpCodeRef = ref<HTMLInputElement | null>(null)

const authHeading = computed(() => {
  if (passwordFlow.value === 'forgot') {
    return forgotSent.value
      ? { label: '邮件已申请', title: '下一步请检查邮箱', description: '为保护账户隐私，系统不会显示该邮箱是否已注册。' }
      : { label: '找回密码', title: '恢复账户访问', description: '我们会向账户绑定邮箱发送一次性重置链接。' }
  }
  if (passwordFlow.value === 'reset') {
    return { label: '安全重置', title: '设置新密码', description: '重置链接只能使用一次，并将在 15 分钟后失效。' }
  }
  if (passwordFlow.value === 'reset-success') {
    return { label: '重置完成', title: '账户已恢复', description: '新密码现已生效。' }
  }
  if (totpSetupRequired.value) {
    return {
      label: '绑定验证器',
      title: totpSetupStep.value === 3 ? '保存恢复码' : '重新绑定 TOTP',
      description: totpSetupStep.value === 3 ? '这是恢复账户访问权限的唯一备用凭据。' : '原密钥已失效，请完成新验证器绑定。',
    }
  }
  if (totpRequired.value) {
    return { label: '二次验证', title: '完成安全确认', description: '输入 Authenticator 应用中的验证码，恢复码也可使用。' }
  }
  return { label: '账号登录', title: '欢迎回来', description: '使用你的账号进入工作台。' }
})

const loginBgStyle = computed(() => {
  if (loginBgUrl.value) {
    return {
      backgroundImage: `url(${loginBgUrl.value})`,
      backgroundSize: 'cover',
      backgroundPosition: 'center',
      backgroundRepeat: 'no-repeat'
    }
  }
  return {}
})

onMounted(async () => {
	const tokenFromURL = new URLSearchParams(window.location.search).get('reset_token') || ''
	if (tokenFromURL) {
	  resetToken.value = tokenFromURL
	  passwordFlow.value = 'reset'
	  window.history.replaceState({}, document.title, window.location.pathname)
	}
  // 已登录则跳转首页
	if (userStore.user && !tokenFromURL) {
    router.replace('/')
    return
  }
  // 获取配置信息
  try {
	await fetchConfig(true)
    orgName.value = config.value?.site_name || ''
    orgLink.value = config.value?.site_link || ''
    version.value = config.value?.version || ''
    loginBgUrl.value = config.value?.login_bg_url || ''
    customLogo.value = config.value?.custom_logo || ''
    captchaEnabled.value = config.value?.enable_captcha || false
    trustDays.value = config.value?.totp_trust_days || 30
    // 如果启用验证码则自动获取
    if (captchaEnabled.value) {
      await refreshCaptcha()
    }
  } catch {
    // 忽略错误，使用默认值
  }
	if (tokenFromURL) await inspectResetToken()
})

function openForgotPassword() {
	if (!passwordResetEnabled.value) {
		message.warning('系统未开放自主密码找回功能，请联系管理员处理！')
		return
	}
	passwordFlow.value = 'forgot'
	forgotSent.value = false
	resetError.value = ''
	resetToken.value = ''
	totpRequired.value = false
	totpSetupRequired.value = false
	if (captchaEnabled.value) refreshCaptcha()
	setTimeout(() => forgotEmailRef.value?.focus(), 100)
}

function returnToLogin() {
	passwordFlow.value = 'login'
	forgotSent.value = false
	forgotEmail.value = ''
	resetToken.value = ''
	resetError.value = ''
	resetTOTPRequired.value = false
	Object.assign(resetForm, { new_password: '', confirm_password: '', totp_code: '' })
	if (captchaEnabled.value) refreshCaptcha()
	setTimeout(() => passwordRef.value?.focus(), 100)
}

async function handleForgotPassword() {
	if (!/^\S+@\S+\.\S+$/.test(forgotEmail.value.trim())) {
	  message.warning('请输入有效的账户绑定邮箱')
	  return
	}
	if (captchaEnabled.value && !form.captcha_code) {
	  message.warning('请输入验证码')
	  return
	}
	forgotLoading.value = true
	try {
	  await api.post('/api/auth/password-reset/request', {
		email: forgotEmail.value.trim(),
		captcha_id: form.captcha_id,
		captcha_code: form.captcha_code,
	  })
	  forgotSent.value = true
	} catch (err: any) {
	  message.error(err.response?.data?.error || '提交失败，请稍后重试')
	  if (captchaEnabled.value) await refreshCaptcha()
	} finally {
	  forgotLoading.value = false
	}
}

async function inspectResetToken() {
	resetChecking.value = true
	resetError.value = ''
	try {
	  const res = await api.post('/api/auth/password-reset/status', { token: resetToken.value })
	  resetTOTPRequired.value = !!res.data.totp_required
	} catch (err: any) {
	  resetError.value = err.response?.data?.error || '重置链接无效或已过期'
	} finally {
	  resetChecking.value = false
	}
}

async function handleResetPassword() {
	if (resetForm.new_password.length < 6) {
	  message.warning('新密码至少需要 6 位')
	  return
	}
	if (resetForm.new_password !== resetForm.confirm_password) {
	  message.warning('两次输入的新密码不一致')
	  return
	}
	if (resetTOTPRequired.value && !/^\d{6}$/.test(resetForm.totp_code)) {
	  message.warning('请输入当前 6 位 TOTP 验证码')
	  return
	}
	resetLoading.value = true
	try {
	  await api.post('/api/auth/password-reset/complete', {
		token: resetToken.value,
		new_password: resetForm.new_password,
		totp_code: resetForm.totp_code,
	  })
	  passwordFlow.value = 'reset-success'
	  resetToken.value = ''
	  Object.assign(resetForm, { new_password: '', confirm_password: '', totp_code: '' })
	} catch (err: any) {
	  const error = err.response?.data?.error || '密码重置失败'
	  message.error(error)
	  if (!err.response?.data?.code) resetError.value = error
	  resetForm.totp_code = ''
	} finally {
	  resetLoading.value = false
	}
}

async function refreshCaptcha() {
  try {
    const res = await api.get('/api/auth/captcha')
    if (res.data.enabled) {
      captchaEnabled.value = true
      captchaImage.value = res.data.captcha_image
      form.captcha_id = res.data.captcha_id
      form.captcha_code = ''
    } else {
      captchaEnabled.value = false
    }
  } catch {
    captchaEnabled.value = false
  }
}

async function handleLogin() {
  if (!form.username || !form.password) {
    message.warning('请输入用户名和密码')
    return
  }
  if (captchaEnabled.value && !form.captcha_code) {
    message.warning('请输入验证码')
    return
  }

  loading.value = true
  try {
    const res = await api.post('/api/auth/login', form)
    if (res.data.totp_setup_required) {
      totpSetupRequired.value = true
      loginToken.value = res.data.login_token
      totpSetupQr.value = res.data.qr_code
      totpSetupStep.value = 1
      return
    }
    if (res.data.totp_required) {
      // 需要 TOTP 二次验证
      totpRequired.value = true
      loginToken.value = res.data.login_token
      // 自动聚焦 TOTP 输入框
      setTimeout(() => totpCodeRef.value?.focus(), 100)
      return
    }
    // 登录成功
    await userStore.fetchMe()
    message.success('登录成功')
    router.replace('/')
  } catch (err: any) {
    if (captchaEnabled.value) {
      refreshCaptcha()
    }
    message.error(err.response?.data?.error || '登录失败')
  } finally {
    loading.value = false
  }
}

function handleFormSubmit() {
	if (passwordFlow.value === 'forgot') return handleForgotPassword()
	if (passwordFlow.value === 'reset') return handleResetPassword()
	if (passwordFlow.value !== 'login') return
	if (totpRequired.value) return handleTOTPLogin()
	if (totpSetupRequired.value && totpSetupStep.value === 1) return handleTOTPSetupLogin()
	return handleLogin()
}

async function handleTOTPLogin() {
  if (!totpCode.value) {
    message.warning('请输入验证码')
    return
  }

  totpLoading.value = true
  try {
    await api.post('/api/auth/login/totp', {
      login_token: loginToken.value,
      code: totpCode.value,
      trust_device: trustDevice.value,
    })
    await userStore.fetchMe()
    message.success('登录成功')
    router.replace('/')
  } catch (err: any) {
    message.error(err.response?.data?.error || '验证失败')
    totpCode.value = ''
    totpCodeRef.value?.focus()
  } finally {
    totpLoading.value = false
  }
}

async function handleTOTPSetupLogin() {
  if (!/^\d{6}$/.test(totpCode.value)) {
    message.warning('请输入 6 位验证码')
    return
  }
  totpLoading.value = true
  try {
    const res = await api.post('/api/auth/login/totp/setup', { login_token: loginToken.value, code: totpCode.value })
    totpRecoveryCodes.value = res.data.recovery_codes || []
    totpSetupStep.value = 3
    totpCode.value = ''
  } catch (err: any) {
    message.error(err.response?.data?.error || '绑定失败')
    totpCode.value = ''
    totpCodeRef.value?.focus()
  } finally {
    totpLoading.value = false
  }
}

function copyTOTPRecoveryCodes() {
  navigator.clipboard.writeText(totpRecoveryCodes.value.join('\n'))
    .then(() => message.success('恢复码已复制'))
    .catch(() => message.warning('复制失败，请手动记录'))
}

async function finishTOTPSetupLogin() {
  await userStore.fetchMe()
  message.success('TOTP 绑定成功，已登录')
  router.replace('/')
}

function resetTOTPFlow() {
  totpRequired.value = false
  totpSetupRequired.value = false
  totpCode.value = ''
  loginToken.value = ''
  totpSetupQr.value = ''
  totpSetupStep.value = 1
  totpRecoveryCodes.value = []
}
</script>

<style scoped>
.login-page {
  --page-bg: #eef3f7;
  --page-ink: #102033;
  --muted-ink: #5b6a7b;
  --soft-ink: #7d8b9a;
  --panel-bg: rgba(255, 255, 255, 0.9);
  --panel-strong: #fbfdff;
  --line: rgba(16, 32, 51, 0.12);
  --line-strong: rgba(16, 32, 51, 0.2);
  --field-bg: #f8fafc;
  --field-hover: #fbfdff;
  --accent: var(--theme-color, #2563eb);
  --accent-rgb: var(--theme-color-rgb, 37, 99, 235);
  --accent-pressed: var(--theme-color-pressed, #1d4ed8);
  --shadow-soft:
    0 1px 2px rgba(16, 32, 51, 0.05),
    0 24px 70px rgba(16, 32, 51, 0.12);
  min-height: 100dvh;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
  padding: clamp(16px, 2.2vw, 24px);
  color: var(--page-ink);
  background:
    linear-gradient(135deg, rgba(255, 255, 255, 0.72), rgba(238, 243, 247, 0.42)),
    radial-gradient(circle at 12% 18%, rgba(var(--accent-rgb), 0.18), transparent 32%),
    radial-gradient(circle at 88% 74%, rgba(22, 163, 74, 0.14), transparent 30%),
    var(--page-bg);
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Noto Sans SC', sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

.login-page::before {
  content: "";
  position: absolute;
  inset: 0;
  pointer-events: none;
  background-image:
    linear-gradient(rgba(16, 32, 51, 0.035) 1px, transparent 1px),
    linear-gradient(90deg, rgba(16, 32, 51, 0.035) 1px, transparent 1px);
  background-size: 56px 56px;
  mask-image: linear-gradient(180deg, rgba(0, 0, 0, 0.72), transparent 84%);
}

.login-page.has-custom-bg::after {
  content: "";
  position: absolute;
  inset: 0;
  pointer-events: none;
  background:
    linear-gradient(90deg, rgba(238, 243, 247, 0.92), rgba(238, 243, 247, 0.58)),
    radial-gradient(circle at 70% 50%, transparent, rgba(16, 32, 51, 0.18));
}

.login-page.dark {
  --page-bg: #07111f;
  --page-ink: #f3f7fb;
  --muted-ink: #a8b5c4;
  --soft-ink: #7f8ea2;
  --panel-bg: rgba(12, 22, 36, 0.9);
  --panel-strong: #101c2e;
  --line: rgba(255, 255, 255, 0.11);
  --line-strong: rgba(255, 255, 255, 0.18);
  --field-bg: rgba(255, 255, 255, 0.055);
  --field-hover: rgba(255, 255, 255, 0.08);
  --shadow-soft:
    0 1px 2px rgba(0, 0, 0, 0.24),
    0 28px 90px rgba(0, 0, 0, 0.42);
  background:
    linear-gradient(135deg, rgba(7, 17, 31, 0.94), rgba(11, 26, 45, 0.86)),
    radial-gradient(circle at 12% 18%, rgba(var(--accent-rgb), 0.26), transparent 34%),
    radial-gradient(circle at 88% 74%, rgba(22, 163, 74, 0.14), transparent 32%),
    var(--page-bg);
}

.login-page.dark::before {
  background-image:
    linear-gradient(rgba(255, 255, 255, 0.04) 1px, transparent 1px),
    linear-gradient(90deg, rgba(255, 255, 255, 0.04) 1px, transparent 1px);
}

.login-page.dark.has-custom-bg::after {
  background:
    linear-gradient(90deg, rgba(7, 17, 31, 0.93), rgba(7, 17, 31, 0.64)),
    radial-gradient(circle at 70% 50%, transparent, rgba(0, 0, 0, 0.28));
}

.login-shell {
  width: min(1016px, 100%);
  min-height: min(640px, calc(100dvh - 32px));
  position: relative;
  z-index: 1;
  display: grid;
  grid-template-columns: minmax(0, 1.08fr) minmax(390px, 0.72fr);
  overflow: hidden;
  border: 1px solid var(--line);
  border-radius: 28px;
  background: var(--panel-bg);
  box-shadow: var(--shadow-soft);
  backdrop-filter: blur(22px);
  -webkit-backdrop-filter: blur(22px);
}

.brand-panel {
  position: relative;
  min-height: 608px;
  display: grid;
  grid-template-rows: auto auto 1fr auto;
  gap: 24px;
  padding: 36px;
  overflow: hidden;
  background:
    linear-gradient(145deg, rgba(var(--accent-rgb), 0.12), transparent 36%),
    linear-gradient(180deg, rgba(255, 255, 255, 0.26), rgba(255, 255, 255, 0));
}

.brand-panel::before {
  content: "";
  position: absolute;
  inset: 14px;
  border: 1px solid var(--line);
  border-radius: 24px;
  pointer-events: none;
}

.brand-copy {
  position: relative;
  z-index: 1;
  max-width: 560px;
}

.brand-copy h1 {
  margin: 0;
  font-size: clamp(32px, 4.2vw, 54px);
  line-height: 1.04;
  letter-spacing: 0;
  text-wrap: balance;
}

.brand-copy p {
  max-width: 46ch;
  margin: 14px 0 0;
  font-size: 15px;
  line-height: 1.62;
  color: var(--muted-ink);
  text-wrap: pretty;
}

.visual-stage {
  position: relative;
  align-self: center;
  min-height: 242px;
  z-index: 1;
}

.visual-card {
  width: min(286px, 68%);
  aspect-ratio: 1 / 1;
  margin: 2px auto 0;
  display: grid;
  place-items: center;
  border-radius: 24px;
  background:
    linear-gradient(145deg, rgba(255, 255, 255, 0.72), rgba(255, 255, 255, 0.16)),
    rgba(var(--accent-rgb), 0.08);
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.48),
    0 28px 70px rgba(16, 32, 51, 0.14);
}

.dark .visual-card {
  background:
    linear-gradient(145deg, rgba(255, 255, 255, 0.12), rgba(255, 255, 255, 0.035)),
    rgba(var(--accent-rgb), 0.1);
}

.hero-art {
  width: min(248px, 84%);
  height: auto;
  filter: drop-shadow(0 24px 34px rgba(var(--accent-rgb), 0.2));
}

.signal-card {
  position: absolute;
  min-height: 40px;
  display: inline-flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  border: 1px solid var(--line);
  border-radius: 999px;
  color: var(--page-ink);
  background: color-mix(in srgb, var(--panel-strong) 86%, transparent);
  box-shadow: 0 16px 34px rgba(16, 32, 51, 0.12);
}

.signal-card svg,
.brand-point svg,
.totp-notice svg {
  width: 18px;
  height: 18px;
  color: var(--accent);
}

.login-qr-wrap {
  display: flex;
  justify-content: center;
  padding: 8px;
}

.login-qr {
  width: 180px;
  height: 180px;
  border-radius: 12px;
  background: #fff;
  padding: 8px;
}

.setup-secret-hint {
  margin: 0;
  color: var(--muted-ink);
  font-size: 12px;
  overflow-wrap: anywhere;
}

.login-recovery-codes {
  display: grid;
  gap: 8px;
  padding: 14px;
  border-radius: 12px;
  background: var(--field-bg);
  box-shadow: inset 0 0 0 1px var(--line);
  text-align: center;
}

.login-recovery-codes code {
  font-size: 16px;
  font-variant-numeric: tabular-nums;
  letter-spacing: 0.08em;
}

.signal-primary {
  top: 26px;
  right: 24px;
}

.signal-secondary {
  left: 22px;
  bottom: 26px;
}

.brand-points {
  position: relative;
  z-index: 1;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.brand-point {
  min-height: 62px;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px;
  border: 1px solid var(--line);
  border-radius: 16px;
  color: var(--muted-ink);
  background: rgba(255, 255, 255, 0.42);
}

.dark .brand-point {
  background: rgba(255, 255, 255, 0.045);
}

.auth-panel {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px;
  border-left: 1px solid var(--line);
  background: color-mix(in srgb, var(--panel-strong) 72%, transparent);
}

.theme-toggle {
  position: absolute;
  top: 18px;
  right: 18px;
  width: 44px;
  height: 44px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--line);
  border-radius: 14px;
  color: var(--page-ink);
  background: var(--panel-strong);
  box-shadow: 0 10px 26px rgba(16, 32, 51, 0.08);
  cursor: pointer;
  transition-property: transform, border-color, background-color, color;
  transition-duration: 180ms;
  transition-timing-function: ease;
}

.theme-toggle:hover {
  border-color: var(--line-strong);
  transform: translateY(-1px);
}

.theme-toggle:active {
  transform: scale(0.98);
}

.toggle-icon {
  width: 20px;
  height: 20px;
}

.auth-card {
  width: 100%;
  max-width: 360px;
}

.auth-header {
  margin-bottom: 22px;
}

.auth-label {
  margin: 0 0 8px;
  font-size: 13px;
  font-weight: 700;
  color: var(--accent);
}

.auth-header h2 {
  margin: 0;
  font-size: 30px;
  line-height: 1.12;
  letter-spacing: 0;
  color: var(--page-ink);
  text-wrap: balance;
}

.auth-header p {
  margin: 8px 0 0;
  color: var(--muted-ink);
  line-height: 1.65;
  text-wrap: pretty;
}

.login-form,
.form-panel {
  display: flex;
  flex-direction: column;
}

.login-form {
  position: relative;
}

.form-panel {
  gap: 16px;
}

.auth-swap-enter-active {
  transition-property: opacity, transform, filter;
  transition-duration: 260ms;
  transition-timing-function: cubic-bezier(0.2, 0, 0, 1);
}

.auth-swap-leave-active {
  transition-property: opacity, transform, filter;
  transition-duration: 150ms;
  transition-timing-function: ease-in;
}

.auth-swap-enter-from {
  opacity: 0;
  filter: blur(4px);
  transform: translateY(10px);
}

.auth-swap-leave-to {
  opacity: 0;
  filter: blur(4px);
  transform: translateY(-8px);
}

.input-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.label-row {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
}

.input-label {
  display: block;
  font-size: 14px;
  font-weight: 700;
  color: var(--page-ink);
}

.field-hint {
  font-size: 12px;
  color: var(--soft-ink);
  white-space: nowrap;
}

.input-wrapper {
  position: relative;
  display: flex;
  align-items: center;
}

.input-icon {
  position: absolute;
  left: 14px;
  width: 20px;
  height: 20px;
  color: var(--soft-ink);
  pointer-events: none;
}

.input-field {
  width: 100%;
  min-height: 48px;
  padding: 13px 14px 13px 44px;
  border: 1px solid var(--line);
  border-radius: 14px;
  outline: none;
  color: var(--page-ink);
  background: var(--field-bg);
  font: inherit;
  font-size: 15px;
  transition-property: background-color, border-color, box-shadow, color;
  transition-duration: 180ms;
  transition-timing-function: ease;
}

.input-field-with-action {
  padding-right: 52px;
}

.input-field::placeholder {
  color: var(--soft-ink);
}

.input-field:hover {
  background: var(--field-hover);
}

.input-field:focus {
  border-color: rgba(var(--accent-rgb), 0.58);
  box-shadow: 0 0 0 4px rgba(var(--accent-rgb), 0.14);
}

.eye-btn {
  position: absolute;
  right: 7px;
  width: 40px;
  height: 40px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 12px;
  color: var(--soft-ink);
  background: transparent;
  cursor: pointer;
  transition-property: background-color, color, transform;
  transition-duration: 180ms;
  transition-timing-function: ease;
}

.eye-btn:hover {
  color: var(--page-ink);
  background: rgba(var(--accent-rgb), 0.08);
}

.eye-btn:active {
  transform: scale(0.98);
}

.eye-btn:focus-visible,
.theme-toggle:focus-visible,
.captcha-image:focus-visible,
.forgot-link:focus-visible,
.back-btn:focus-visible,
.login-btn:focus-visible {
  outline: none;
  box-shadow: 0 0 0 4px rgba(var(--accent-rgb), 0.18);
}

.eye-icon {
  width: 20px;
  height: 20px;
}

.captcha-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 116px;
  gap: 10px;
}

.captcha-input {
  min-width: 0;
}

.captcha-image {
  min-height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border: 1px solid var(--line);
  border-radius: 14px;
  color: var(--muted-ink);
  background: var(--field-bg);
  cursor: pointer;
  font: inherit;
  transition-property: border-color, background-color, transform;
  transition-duration: 180ms;
  transition-timing-function: ease;
}

.captcha-image:hover {
  border-color: rgba(var(--accent-rgb), 0.42);
  background: var(--field-hover);
}

.captcha-image:active {
  transform: scale(0.98);
}

.captcha-image img {
  height: 48px;
  width: 100%;
  object-fit: cover;
  display: block;
  outline: 1px solid rgba(0, 0, 0, 0.1);
  outline-offset: -1px;
}

.dark .captcha-image img {
  outline-color: rgba(255, 255, 255, 0.1);
}

.captcha-image span {
  padding: 0 10px;
  font-size: 12px;
}

.remember-row {
  min-height: 40px;
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--muted-ink);
  font-size: 14px;
  line-height: 1.45;
  cursor: pointer;
  text-wrap: pretty;
}

.login-options-row {
  min-height: 40px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.forgot-link {
  min-height: 40px;
  padding: 0 2px;
  border: 0;
  color: var(--accent);
  background: transparent;
  font: inherit;
  font-size: 14px;
  font-weight: 650;
  white-space: nowrap;
  cursor: pointer;
  transition-property: color, transform;
  transition-duration: 180ms;
}

.forgot-link:hover {
  color: var(--accent-pressed);
}

.forgot-link:active {
  transform: scale(0.96);
}

.checkbox {
  width: 18px;
  height: 18px;
  flex: 0 0 auto;
  border-radius: 6px;
  accent-color: var(--accent);
  cursor: pointer;
}

.tabular-num {
  font-variant-numeric: tabular-nums;
}

.login-btn {
  width: 100%;
  min-height: 50px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  border: 0;
  border-radius: 14px;
  color: #f8fbff;
  background: var(--accent);
  box-shadow: 0 18px 34px rgba(var(--accent-rgb), 0.26);
  font: inherit;
  font-size: 16px;
  font-weight: 760;
  cursor: pointer;
  transition-property: transform, background-color, box-shadow, opacity;
  transition-duration: 180ms;
  transition-timing-function: ease;
}

.login-btn:hover:not(:disabled) {
  background: var(--accent-pressed);
  box-shadow: 0 22px 40px rgba(var(--accent-rgb), 0.3);
  transform: translateY(-1px);
}

.login-btn:active:not(:disabled) {
  transform: scale(0.98);
}

.login-btn:disabled {
  opacity: 0.68;
  cursor: not-allowed;
}

.spinner {
  width: 18px;
  height: 18px;
  border: 2.5px solid rgba(255, 255, 255, 0.35);
  border-top-color: #f8fbff;
  border-radius: 50%;
  animation: spin 0.7s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.totp-notice,
.security-notice {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 12px;
  border: 1px solid rgba(var(--accent-rgb), 0.18);
  border-radius: 14px;
  color: var(--muted-ink);
  background: rgba(var(--accent-rgb), 0.07);
  font-size: 14px;
  line-height: 1.55;
}

.totp-notice svg,
.security-notice svg {
	width: 20px;
	height: 20px;
  margin-top: 1px;
  flex: 0 0 auto;
}

.totp-notice > span,
.security-notice > span {
	min-width: 0;
	flex: 1 1 auto;
	overflow-wrap: anywhere;
	text-wrap: pretty;
}

.request-complete {
  min-height: 190px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 24px;
  border-radius: 14px;
  color: var(--muted-ink);
  background: rgba(var(--accent-rgb), 0.07);
  text-align: center;
}

.request-complete strong {
  color: var(--page-ink);
  font-size: 18px;
}

.request-complete p {
  margin: 0;
  line-height: 1.65;
  text-wrap: pretty;
}

.request-complete-icon {
  width: 48px;
  height: 48px;
  display: grid;
  place-items: center;
  border-radius: 14px;
  color: var(--accent);
  background: rgba(var(--accent-rgb), 0.12);
}

.request-complete-icon svg {
  width: 24px;
  height: 24px;
}

.error-state .request-complete-icon {
  color: #d03050;
  background: rgba(208, 48, 80, 0.1);
}

.reset-checking {
  min-height: 220px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: var(--muted-ink);
}

.spinner-dark {
  border-color: rgba(var(--accent-rgb), 0.2);
  border-top-color: var(--accent);
}

.back-btn {
  min-height: 40px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: 0;
  border-radius: 14px;
  color: var(--muted-ink);
  background: transparent;
  font: inherit;
  font-size: 14px;
  cursor: pointer;
  transition-property: color, background-color, transform;
  transition-duration: 180ms;
  transition-timing-function: ease;
}

.back-btn:hover {
  color: var(--page-ink);
  background: rgba(var(--accent-rgb), 0.08);
}

.back-btn:active {
  transform: scale(0.98);
}

.bottom-links {
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid var(--line);
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--soft-ink);
  font-size: 12px;
}

.bottom-links a {
  color: var(--muted-ink);
  text-decoration: none;
  transition-property: color;
  transition-duration: 180ms;
  transition-timing-function: ease;
}

.bottom-links a:hover {
  color: var(--page-ink);
}

.sep {
  width: 4px;
  height: 4px;
  border-radius: 999px;
  background: var(--line-strong);
}

.version-text {
  color: var(--soft-ink);
  font-variant-numeric: tabular-nums;
}

@media (prefers-reduced-motion: reduce) {
  *,
  *::before,
  *::after {
    animation-duration: 0.001ms !important;
    animation-iteration-count: 1 !important;
    scroll-behavior: auto !important;
    transition-duration: 0.001ms !important;
  }
}

@media (max-width: 920px) {
  .login-page {
    padding: 20px;
    align-items: stretch;
  }

  .login-shell {
    min-height: auto;
    grid-template-columns: 1fr;
  }

  .brand-panel {
    min-height: auto;
    gap: 24px;
    padding: 34px;
  }

  .brand-panel::before,
  .visual-stage,
  .brand-points {
    display: none;
  }

  .brand-copy h1 {
    font-size: clamp(32px, 8vw, 44px);
  }

  .auth-panel {
    border-left: 0;
    border-top: 1px solid var(--line);
    padding: 34px;
  }
}

@media (max-width: 560px) {
  .login-page {
    padding: 12px;
  }

  .login-shell {
    border-radius: 24px;
  }

  .brand-panel,
  .auth-panel {
    padding: 24px;
  }

  .theme-toggle {
    top: 18px;
    right: 18px;
  }

  .auth-header h2 {
    font-size: 28px;
  }

  .captcha-row {
    grid-template-columns: 1fr;
  }

  .captcha-image {
    width: 100%;
  }

  .label-row {
    align-items: flex-start;
    flex-direction: column;
    gap: 3px;
  }

  .field-hint {
    white-space: normal;
  }
}
</style>
