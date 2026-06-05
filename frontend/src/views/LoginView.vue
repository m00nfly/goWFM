<template>
  <div class="login-page" :class="{ dark: themeStore.isDark }">
    <!-- 背景装饰 -->
    <div class="blob blob-blue"></div>
    <div class="blob blob-purple"></div>

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
          <div class="logo-icon">
            <FolderOutline />
          </div>
          <h1 class="title">欢迎回来</h1>
          <p class="subtitle">请登录您的 goWFM 账号</p>
        </div>

        <!-- 登录表单 -->
        <form @submit.prevent="handleLogin" class="login-form">
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
                @keyup.enter="handleLogin"
              />
              <button type="button" class="eye-btn" @click="showPassword = !showPassword">
                <EyeOffOutline v-if="showPassword" class="eye-icon" />
                <EyeOutline v-else class="eye-icon" />
              </button>
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
        </form>

        <!-- 底部链接 -->
        <div class="bottom-links">
          <a href="#">隐私政策</a>
          <a href="#">服务条款</a>
          <a href="#">联系支持</a>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
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

const form = reactive({
  username: '',
  password: '',
})

const rememberMe = ref(false)

onMounted(() => {
  // 已登录则跳转首页
  if (userStore.user) {
    router.replace('/')
  }
})

async function handleLogin() {
  if (!form.username || !form.password) {
    message.warning('请输入用户名和密码')
    return
  }

  loading.value = true
  try {
    await api.post('/api/auth/login', form)
    await userStore.fetchMe()
    message.success('登录成功')
    router.replace('/')
  } catch (err: any) {
    message.error(err.response?.data?.error || '登录失败')
  } finally {
    loading.value = false
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

.logo-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: #3b82f6;
  padding: 12px;
  border-radius: 16px;
  box-shadow: 0 8px 24px rgba(59, 130, 246, 0.3);
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
  color: #3b82f6;
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
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.15);
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
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.2);
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
  accent-color: #3b82f6;
  cursor: pointer;
}

/* ============ 登录按钮 ============ */
.login-btn {
  width: 100%;
  background: #3b82f6;
  color: #fff;
  font-weight: 700;
  font-size: 16px;
  padding: 16px;
  border: none;
  border-radius: 16px;
  box-shadow: 0 8px 24px rgba(59, 130, 246, 0.3);
  cursor: pointer;
  transition: all 0.2s ease;
  font-family: inherit;
  min-height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.login-btn:hover:not(:disabled) {
  background: #2563eb;
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
  justify-content: center;
  gap: 24px;
  font-size: 12px;
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

/* ============ 响应式 ============ */
@media (max-width: 480px) {
  .glass-card {
    padding: 32px 24px;
    border-radius: 28px;
  }

  .title {
    font-size: 24px;
  }
}
</style>
