<template>
  <div class="login-page" :class="{ dark: themeStore.isDark }" :style="loginBgStyle">
    <!-- 背景装饰 -->
    <template v-if="!loginBgUrl">
      <div class="blob blob-blue"></div>
      <div class="blob blob-purple"></div>
    </template>

    <!-- 主题切换按钮 -->
    <button class="theme-toggle" @click="themeStore.toggleTheme()" :title="themeStore.isDark ? '切换亮色' : '切换暗色'">
      <MoonOutline v-if="!themeStore.isDark" class="toggle-icon" />
      <SunnyOutline v-else class="toggle-icon" />
    </button>

    <!-- 登录卡片 -->
    <div class="login-wrapper">
      <div class="glass-card">
        <!-- Logo & 标题 -->
        <div class="card-header">
          <div v-if="customLogo" class="logo-custom">
            <img :src="customLogo" class="logo-img" alt="Logo" />
          </div>
          <div v-else class="logo-icon">
            <FolderOutline />
          </div>
          <h1 class="title">{{ orgName || 'goWFM' }}</h1>
          <p class="subtitle">{{ totpRequired ? '请输入二次验证码' : '登录您的账号' }}</p>
        </div>

        <!-- 登录表单 -->
        <form @submit.prevent="handleLogin" class="login-form">
          <!-- 正常登录表单（TOTP 未触发时显示） -->
          <template v-if="!totpRequired">
            <div class="input-group">
              <label class="input-label">账号</label>
              <div class="input-wrapper">
                <MailOutline class="input-icon" />
                <input
                  v-model="form.username"
                  type="text"
                  required
                  placeholder="账号/Email"
                  class="input-field"
                  autocomplete="username"
                />
              </div>
            </div>

            <div class="input-group">
              <label class="input-label">密码</label>
              <div class="input-wrapper">
                <LockClosedOutline class="input-icon" />
                <input
                  ref="passwordRef"
                  v-model="form.password"
                  :type="showPassword ? 'text' : 'password'"
                  required
                  placeholder="••••••••"
                  class="input-field"
                  autocomplete="current-password"
                />
                <button type="button" class="eye-btn" @click="showPassword = !showPassword">
                  <EyeOffOutline v-if="showPassword" class="eye-icon" />
                  <EyeOutline v-else class="eye-icon" />
                </button>
              </div>
            </div>

            <div v-if="captchaEnabled" class="input-group">
              <label class="input-label">验证码</label>
              <div class="captcha-row">
                <div class="input-wrapper" style="flex: 1">
                  <LockClosedOutline class="input-icon" />
                  <input
                    v-model="form.captcha_code"
                    type="text"
                    required
                    placeholder="请输入验证码"
                    class="input-field"
                    autocomplete="off"
                  />
                </div>
                <div class="captcha-image" @click="refreshCaptcha" title="点击刷新验证码">
                  <img v-if="captchaImage" :src="captchaImage" alt="验证码" />
                  <span v-else>加载中...</span>
                </div>
              </div>
            </div>

            <label class="remember-row">
              <input type="checkbox" v-model="rememberMe" class="checkbox" />
              <span>保持登录状态</span>
            </label>

            <button type="submit" class="login-btn" :disabled="loading">
              <span v-if="loading" class="spinner"></span>
              <span v-else>立即登录</span>
            </button>
          </template>

          <!-- TOTP 二次验证（凭据验证通过后显示） -->
          <template v-else>
            <div class="totp-notice">
              请输入 Authenticator APP 中的 6 位验证码，或使用恢复码
            </div>
            <div class="input-group">
              <label class="input-label">验证码</label>
              <div class="input-wrapper">
                <LockClosedOutline class="input-icon" />
                <input
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
              <span>信任此设备（{{ trustDays }} 天内无需再次验证）</span>
            </label>
            <button type="button" class="login-btn" :disabled="totpLoading" @click="handleTOTPLogin">
              <span v-if="totpLoading" class="spinner"></span>
              <span v-else>验证并登录</span>
            </button>
            <button type="button" class="back-btn" @click="totpRequired = false; totpCode = ''">← 返回修改账号密码</button>
          </template>
        </form>

        <!-- 底部链接 -->
        <div class="bottom-links">
          <template v-if="orgName">
            <a v-if="orgLink" :href="orgLink" target="_blank" rel="noopener noreferrer">{{ orgName }}</a>
            <span v-else>{{ orgName }}</span>
            <span class="sep">·</span>
          </template>
          <a href="https://gowfm.dev" target="_blank" rel="noopener noreferrer">goWFM</a>
          <span class="sep">·</span>
          <a href="https://github.com/m00nfly/gowfm" target="_blank" rel="noopener noreferrer">GitHub</a>
          <span v-if="version" class="sep">·</span>
          <span v-if="version" class="version-text">ver: {{ version }}</span>
        </div>
      </div>
    </div>
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
} from '@vicons/ionicons5'
import api from '@/api'
import { useUserStore } from '@/stores/user'
import { useThemeStore } from '@/stores/theme'

const router = useRouter()
const message = useMessage()
const userStore = useUserStore()
const themeStore = useThemeStore()
const loading = ref(false)
const showPassword = ref(false)
const passwordRef = ref<HTMLInputElement | null>(null)

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
const totpCode = ref('')
const totpLoading = ref(false)
const trustDevice = ref(false)
const trustDays = ref(30)
const loginToken = ref('')
const totpCodeRef = ref<HTMLInputElement | null>(null)

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
  // 已登录则跳转首页
  if (userStore.user) {
    router.replace('/')
    return
  }
  // 获取配置信息
  try {
    const res = await api.get('/api/config/info')
    orgName.value = res.data.site_name || ''
    orgLink.value = res.data.site_link || ''
    version.value = res.data.version || ''
    loginBgUrl.value = res.data.login_bg_url || ''
    customLogo.value = res.data.custom_logo || ''
    captchaEnabled.value = res.data.enable_captcha || false
    // 如果启用验证码则自动获取
    if (captchaEnabled.value) {
      await refreshCaptcha()
    }
  } catch {
    // 忽略错误，使用默认值
  }
})

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
    if (res.data.totp_required) {
      // 需要 TOTP 二次验证
      totpRequired.value = true
      loginToken.value = res.data.login_token
      // 获取信任天数配置
      try {
        const configRes = await api.get('/api/config/info')
        trustDays.value = configRes.data.totp_trust_days || 30
      } catch { /* use default */ }
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
</script>

<style scoped>
.login-page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  padding: 16px;
  position: relative;
  overflow: hidden;
  background: #f8fafc;
  transition: background 0.3s ease;
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Noto Sans SC', sans-serif;
}

.login-page.dark {
  background: #030712;
}

/* ============ 背景装饰 ============ */
.blob {
  position: absolute;
  width: 500px;
  height: 500px;
  border-radius: 50%;
  filter: blur(80px);
  z-index: 0;
  animation: blob-move 25s infinite alternate;
}

.blob-blue {
  top: -10%;
  left: -10%;
  background: linear-gradient(180deg, rgba(59, 130, 246, 0.2) 0%, rgba(37, 99, 235, 0.2) 100%);
}

.blob-purple {
  bottom: -10%;
  right: -10%;
  animation-delay: -10s;
  background: linear-gradient(180deg, rgba(147, 51, 234, 0.1) 0%, rgba(79, 70, 229, 0.1) 100%);
}

@keyframes blob-move {
  from { transform: translate(-10%, -10%); }
  to { transform: translate(20%, 20%); }
}

/* ============ 主题切换按钮 ============ */
.theme-toggle {
  position: fixed;
  top: 24px;
  right: 24px;
  z-index: 10;
  padding: 12px;
  background: #fff;
  border: none;
  border-radius: 16px;
  box-shadow: 0 4px 24px rgba(0, 0, 0, 0.1);
  cursor: pointer;
  transition: transform 0.2s ease;
  color: #475569;
}

.theme-toggle:hover {
  transform: scale(1.1);
}

.dark .theme-toggle {
  background: #1f2937;
  color: #fbbf24;
}

.toggle-icon {
  width: 20px;
  height: 20px;
}

/* ============ 登录卡片容器 ============ */
.login-wrapper {
  width: 100%;
  max-width: 440px;
  z-index: 1;
}

.glass-card {
  backdrop-filter: blur(20px);
  background-color: rgba(255, 255, 255, 0.7);
  border: 1px solid rgba(255, 255, 255, 0.3);
  border-radius: 40px;
  padding: 48px;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.15);
}

.dark .glass-card {
  background-color: rgba(17, 24, 39, 0.7);
  border: 1px solid rgba(255, 255, 255, 0.1);
}

/* ============ 卡片头部 ============ */
.card-header {
  text-align: center;
  margin-bottom: 40px;
}

/* 自定义 Logo（矩形图片自适应） */
.logo-custom {
  display: flex;
  justify-content: center;
  margin-bottom: 16px;
}

.logo-img {
  max-height: 64px;
  max-width: 200px;
  width: auto;
  height: auto;
  object-fit: contain;
}

/* 默认图标（无 Logo 时） */
.logo-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: var(--theme-color, #3b82f6);
  padding: 12px;
  border-radius: 16px;
  box-shadow: 0 8px 24px rgba(var(--theme-color-rgb, 59, 130, 246), 0.3);
  margin-bottom: 16px;
  color: #fff;
  font-size: 32px;
}

.title {
  font-size: 30px;
  font-weight: 700;
  letter-spacing: -0.025em;
  color: #0f172a;
  margin: 0;
}

.dark .title {
  color: #fff;
}

.subtitle {
  color: #64748b;
  margin-top: 8px;
  font-size: 15px;
}

.dark .subtitle {
  color: #94a3b8;
}

/* ============ 表单 ============ */
.login-form {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.input-group {
  display: flex;
  flex-direction: column;
}

.input-label {
  display: block;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: #64748b;
  margin-bottom: 8px;
  padding-left: 4px;
  transition: all 0.2s ease;
}

.dark .input-label {
  color: #94a3b8;
}

.input-group:focus-within .input-label {
  color: var(--theme-color, #3b82f6);
  transform: translateY(-2px);
}

.input-wrapper {
  position: relative;
  display: flex;
  align-items: center;
}

.input-icon {
  position: absolute;
  left: 16px;
  width: 20px;
  height: 20px;
  color: #94a3b8;
  pointer-events: none;
}

.input-field {
  width: 100%;
  padding: 16px 16px 16px 48px;
  background: rgba(255, 255, 255, 0.5);
  border: 1px solid #e2e8f0;
  border-radius: 16px;
  outline: none;
  font-size: 15px;
  color: #0f172a;
  transition: all 0.2s ease;
  font-family: inherit;
}

.input-field::placeholder {
  color: #94a3b8;
}

.input-field:focus {
  border-color: var(--theme-color, #3b82f6);
  box-shadow: 0 0 0 3px rgba(var(--theme-color-rgb, 59, 130, 246), 0.15);
}

.dark .input-field {
  background: rgba(17, 24, 39, 0.5);
  border-color: #374151;
  color: #fff;
}

.dark .input-field::placeholder {
  color: #6b7280;
}

.dark .input-field:focus {
  border-color: var(--theme-color, #3b82f6);
  box-shadow: 0 0 0 3px rgba(var(--theme-color-rgb, 59, 130, 246), 0.2);
}

.eye-btn {
  position: absolute;
  right: 16px;
  background: none;
  border: none;
  cursor: pointer;
  color: #94a3b8;
  padding: 0;
  display: flex;
  align-items: center;
  transition: color 0.2s;
}

.eye-btn:hover {
  color: #475569;
}

.dark .eye-btn:hover {
  color: #cbd5e1;
}

.eye-icon {
  width: 20px;
  height: 20px;
}

/* ============ 记住我 ============ */
.remember-row {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  font-size: 14px;
  color: #475569;
}

.dark .remember-row {
  color: #94a3b8;
}

.checkbox {
  width: 16px;
  height: 16px;
  border-radius: 4px;
  border-color: #cbd5e1;
  accent-color: var(--theme-color, #3b82f6);
  cursor: pointer;
}

/* ============ 登录按钮 ============ */
.login-btn {
  width: 100%;
  background: var(--theme-color, #3b82f6);
  color: #fff;
  font-weight: 700;
  font-size: 16px;
  padding: 16px;
  border: none;
  border-radius: 16px;
  box-shadow: 0 8px 24px rgba(var(--theme-color-rgb, 59, 130, 246), 0.3);
  cursor: pointer;
  transition: all 0.2s ease;
  font-family: inherit;
  min-height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.login-btn:hover:not(:disabled) {
  background: var(--theme-color-pressed, #2563eb);
}

.login-btn:active:not(:disabled) {
  transform: scale(0.98);
}

.login-btn:disabled {
  opacity: 0.7;
  cursor: not-allowed;
}

/* ============ 加载动画 ============ */
.spinner {
  width: 20px;
  height: 20px;
  border: 2.5px solid rgba(255, 255, 255, 0.3);
  border-top-color: #fff;
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* ============ 底部链接 ============ */
.bottom-links {
  margin-top: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  font-size: 12px;
  color: #94a3b8;
  flex-wrap: wrap;
}

.bottom-links a {
  color: #94a3b8;
  text-decoration: none;
  transition: color 0.2s;
}

.bottom-links a:hover {
  color: #475569;
}

.dark .bottom-links a:hover {
  color: #cbd5e1;
}

.bottom-links .sep {
  margin: 0 2px;
  color: #cbd5e1;
}

.dark .bottom-links .sep {
  color: #4b5563;
}

.bottom-links .version-text {
  color: #94a3b8;
  font-size: 11px;
}

/* ============ 验证码 ============ */
.captcha-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.captcha-image {
  flex-shrink: 0;
  height: 48px;
  border-radius: 12px;
  overflow: hidden;
  cursor: pointer;
  border: 1px solid #e2e8f0;
  transition: border-color 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f0f4f8;
}

.captcha-image:hover {
  border-color: var(--theme-color, #3b82f6);
}

.dark .captcha-image {
  border-color: #374151;
  background: #1e293b;
}

.dark .captcha-image:hover {
  border-color: var(--theme-color, #3b82f6);
}

.captcha-image img {
  height: 48px;
  display: block;
}

.captcha-image span {
  font-size: 12px;
  color: #94a3b8;
  padding: 0 8px;
}

/* ============ TOTP 样式 ============ */
.totp-notice {
  text-align: center;
  color: #64748b;
  font-size: 13px;
  margin-bottom: 4px;
  line-height: 1.6;
}

.dark .totp-notice {
  color: #94a3b8;
}

.back-btn {
  width: 100%;
  background: transparent;
  color: #64748b;
  font-size: 13px;
  padding: 8px;
  border: none;
  cursor: pointer;
  font-family: inherit;
  transition: color 0.2s;
}

.back-btn:hover {
  color: #475569;
}

.dark .back-btn {
  color: #94a3b8;
}

.dark .back-btn:hover {
  color: #cbd5e1;
}

/* ============ 响应式 ============ */
@media (max-width: 480px) {
  .glass-card {
    padding: 32px 24px;
    border-radius: 28px;
  }

  .title {
    font-size: 24px;
  }

  .logo-img {
    max-height: 48px;
    max-width: 160px;
  }
}
</style>
