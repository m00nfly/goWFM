<template>
  <div class="main-layout" :class="{ dark: themeStore.isDark }">
    <!-- 顶部导航栏 - 固定毛玻璃效果 -->
    <header class="top-header">
      <div class="header-inner">
        <!-- Logo -->
        <div class="header-brand" @click="router.push('/')">
          <BrandIdentity :logo="customLogo" :name="orgName || 'goWFM'" variant="header" />
        </div>

        <!-- 右侧操作区 -->
        <div class="header-actions">
          <!-- 导航图标 - 宽屏（仅登录后显示） -->
          <div v-show="!isNarrow && userStore.user" class="nav-icons">
            <n-tooltip trigger="hover" placement="bottom">
              <template #trigger>
                <button class="nav-icon-btn" :class="{ active: activeMenuKey === '/' }" @click="router.push('/')">
                  <n-icon size="22"><FolderOpenOutline /></n-icon>
                </button>
              </template>
              文件管理
            </n-tooltip>

            <n-tooltip v-if="userStore.user?.is_admin || userStore.hasPermission(8)" trigger="hover" placement="bottom">
              <template #trigger>
                <button class="nav-icon-btn" :class="{ active: activeMenuKey === shareMenuKey }" @click="router.push(shareMenuKey)">
                  <n-badge :value="shareBadgeCount" :type="shareBadgeType" :show="shareBadgeCount > 0" :offset="[-4, 4]">
                    <n-icon size="22"><ShareSocialOutline /></n-icon>
                  </n-badge>
                </button>
              </template>
              {{ userStore.user?.is_admin ? '分享管理' : '我的分享' }}
            </n-tooltip>

            <n-tooltip v-if="userStore.hasPermission(16)" trigger="hover" placement="bottom">
              <template #trigger>
                <button class="nav-icon-btn" :class="{ active: activeMenuKey === '/logs' }" @click="router.push('/logs')">
                  <n-icon size="22"><DocumentTextOutline /></n-icon>
                </button>
              </template>
              操作日志
            </n-tooltip>

            <n-tooltip v-if="userStore.user?.is_admin" trigger="hover" placement="bottom">
              <template #trigger>
                <button class="nav-icon-btn" :class="{ active: activeMenuKey === '/admin/users' }" @click="router.push('/admin/users')">
                  <n-icon size="22"><PeopleOutline /></n-icon>
                </button>
              </template>
              用户管理
            </n-tooltip>

            <n-tooltip v-if="userStore.user?.is_admin" trigger="hover" placement="bottom">
              <template #trigger>
                <button class="nav-icon-btn" :class="{ active: activeMenuKey === '/admin/settings' }" @click="router.push('/admin/settings')">
                  <n-icon size="22"><SettingsOutline /></n-icon>
                </button>
              </template>
              系统设置
            </n-tooltip>
          </div>

          <!-- 导航图标 - 窄屏折叠菜单（仅登录后显示） -->
          <n-popselect
            v-if="isNarrow && userStore.user"
            v-model:value="popNavValue"
            :options="popNavOptions"
            trigger="click"
            @update:value="onPopNavSelect"
          >
            <button class="nav-icon-btn">
              <n-badge :value="shareBadgeCount" :type="shareBadgeType" :show="shareBadgeCount > 0" dot :offset="[-2, 2]">
                <n-icon size="22"><MenuOutline /></n-icon>
              </n-badge>
            </button>
          </n-popselect>

          <!-- 主题切换 -->
          <n-tooltip trigger="hover" placement="bottom">
            <template #trigger>
              <button class="nav-icon-btn" @click="themeStore.toggleTheme()">
                <n-icon size="22"><SunnyOutline v-if="themeStore.isDark" /><MoonOutline v-else /></n-icon>
              </button>
            </template>
            {{ themeStore.isDark ? '切换亮色' : '切换暗色' }}
          </n-tooltip>

          <template v-if="userStore.user">
            <!-- 分隔线 -->
            <div class="header-divider"></div>

            <!-- 用户下拉 -->
            <n-dropdown trigger="click" :options="userDropdownOptions" @select="onUserAction">
              <div class="user-trigger">
                <UserAvatar
                  :size="34"
                  :avatar="userStore.user.avatar"
                  :name="displayName"
                  class="user-avatar"
                />
                <span class="user-display-name">{{ displayName }}</span>
              </div>
            </n-dropdown>
          </template>
        </div>
      </div>
    </header>

    <!-- 主内容区 -->
    <main class="main-content">
      <div class="content-wrapper">
        <router-view />
      </div>
      <div class="content-footer">
        <div class="footer-content">
          <template v-if="orgLink">
            <a :href="orgLink" target="_blank" class="footer-org-link">{{ orgName || orgLink }}</a>
            <span class="footer-separator">|</span>
          </template>
          <template v-else-if="orgName">
            <span class="footer-org-text">{{ orgName }}</span>
            <span class="footer-separator">|</span>
          </template>
          <a :href="appLink" target="_blank" class="footer-app-link">goWFM</a>
          <a v-if="appGithub" :href="appGithub" target="_blank" class="footer-github-link">
            <n-icon :size="14"><LogoGithub /></n-icon>
          </a>
          <span v-if="version" class="footer-version">ver: {{ version }}</span>
        </div>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, h, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import {
  NDropdown, NIcon, NTooltip, NBadge, NPopselect,
} from 'naive-ui'
import {
  FolderOpenOutline,
  ShareSocialOutline,
  DocumentTextOutline,
  PeopleOutline,
  SettingsOutline,
  LogOutOutline,
  LogoGithub,
  SunnyOutline,
  MoonOutline,
  MenuOutline,
} from '@vicons/ionicons5'
import { useUserStore } from '@/stores/user'
import { useThemeStore } from '@/stores/theme'
import { useConfig } from '@/composables/useConfig'
import { useViewport } from '@/composables/useViewport'
import api from '@/api'
import UserAvatar from '@/components/UserAvatar.vue'
import BrandIdentity from '@/components/BrandIdentity.vue'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()
const themeStore = useThemeStore()
const { config, fetchConfig } = useConfig()

const version = ref('')
const orgName = ref('')
const orgLink = ref('')
const customLogo = ref('')
const appLink = ref('https://gowfm.dev')
const appGithub = ref('https://github.com/m00nfly/gowfm')

// ---------- 响应式导航 ----------

const { isMobile: isNarrow, sync: syncViewport } = useViewport()

// ---------- 分享 Badge ----------

const shareBadgeCount = computed(() => {
  if (userStore.shareExpiredCount > 0) return userStore.shareExpiredCount
  if (userStore.shareValidCount > 0) return userStore.shareValidCount
  return 0
})

const shareBadgeType = computed<'error' | 'info'>(() => {
  if (userStore.shareExpiredCount > 0) return 'error'
  return 'info'
})

// ---------- 用户显示 ----------

const displayName = computed(() =>
  userStore.user?.display_name || userStore.user?.username || '用户'
)

// ---------- 导航配置 ----------

const shareMenuKey = computed(() => '/shares')

// 高亮的导航 key
const activeMenuKey = computed(() => {
  const p = route.path
  if (p.startsWith('/admin/users')) return '/admin/users'
  if (p.startsWith('/admin/settings')) return '/admin/settings'
  if (p === '/shares') return '/shares'
  if (p === '/logs') return '/logs'
  if (p === '/settings') return '/settings'
  return '/'
})

// ---------- 窄屏折叠导航 ----------

const popNavValue = ref<string | null>(null)

const popNavOptions = computed(() => {
  const opts: Array<{ label: string; value: string }> = [
    { label: '文件管理', value: '/' },
  ]
  if (userStore.user?.is_admin || userStore.hasPermission(8)) {
    const label = userStore.user?.is_admin ? '分享管理' : '我的分享'
    const badgeText = shareBadgeCount.value > 0
      ? ` (${shareBadgeCount.value})`
      : ''
    opts.push({ label: label + badgeText, value: shareMenuKey.value })
  }
  if (userStore.hasPermission(16)) {
    opts.push({ label: '操作日志', value: '/logs' })
  }
  if (userStore.user?.is_admin) {
    opts.push({ label: '用户管理', value: '/admin/users' })
    opts.push({ label: '系统设置', value: '/admin/settings' })
  }
  return opts
})

function onPopNavSelect(value: string) {
  router.push(value)
  popNavValue.value = null
}

// 用户下拉菜单
const userDropdownOptions = computed(() => [
  {
    label: '个人设置',
    key: 'settings',
    icon: () => h(NIcon, null, () => h(SettingsOutline)),
  },
  { type: 'divider', key: 'd2' },
  {
    label: '退出登录',
    key: 'logout',
    icon: () => h(NIcon, null, () => h(LogOutOutline)),
  },
])

// ---------- 事件处理 ----------

async function onUserAction(key: string) {
  if (key === 'settings') {
    router.push('/settings')
  } else if (key === 'logout') {
    try { await api.post('/api/auth/logout') } catch { /* ignore */ }
    userStore.logout()
    router.push('/login')
  }
}

// ---------- 生命周期 ----------

onMounted(async () => {
  window.addEventListener('resize', syncViewport)

  await fetchConfig()
  if (config.value) {
    orgName.value = config.value.site_name || ''
    orgLink.value = config.value.site_link || ''
    version.value = config.value.version || ''
    customLogo.value = config.value.custom_logo || ''
  }
})

onUnmounted(() => {
  window.removeEventListener('resize', syncViewport)
})
</script>

<style scoped>
.main-layout {
  min-height: 100dvh;
  background: var(--workspace-bg);
  color: var(--workspace-text);
  transition:
    background-color 0.2s ease,
    color 0.2s ease;
}

/* ---- 顶部导航栏 - 毛玻璃效果 ---- */
.top-header {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: 100;
  backdrop-filter: blur(12px);
  background-color: color-mix(in srgb, var(--workspace-surface) 88%, transparent);
  border-bottom: 1px solid var(--workspace-border-soft);
  box-shadow: 0 8px 28px rgba(39, 55, 82, 0.08);
  transition:
    background-color 0.2s ease,
    border-color 0.2s ease,
    box-shadow 0.2s ease;
}

.header-inner {
  max-width: 1280px;
  margin: 0 auto;
  padding: 0 24px;
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

/* ---- Logo ---- */
.header-brand {
  min-width: 0;
  max-width: min(360px, 40vw);
  flex: 1 1 auto;
  display: flex;
  align-items: center;
  cursor: pointer;
  user-select: none;
  height: 100%;
  overflow: hidden;
}

.header-brand:active :deep(.brand-identity__fallback) {
  transform: scale(0.96);
}

/* ---- 右侧操作区 ---- */
.header-actions {
  min-width: 0;
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  gap: 4px;
}

/* ---- 导航图标 ---- */
.nav-icons {
  display: flex;
  align-items: center;
  gap: 2px;
}

.nav-icon-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border: 1px solid transparent;
  background: transparent;
  border-radius: var(--workspace-radius-md);
  cursor: pointer;
  color: var(--workspace-text);
  transition:
    background-color 0.16s ease,
    border-color 0.16s ease,
    color 0.16s ease,
    transform 0.16s ease,
    box-shadow 0.16s ease;
  font-size: 22px;
}

.nav-icon-btn:hover {
  background: rgba(var(--workspace-accent-rgb), 0.07);
  border-color: rgba(var(--workspace-accent-rgb), 0.14);
  color: var(--workspace-accent);
}

.nav-icon-btn.active {
  background: rgba(var(--workspace-accent-rgb), 0.11);
  border-color: rgba(var(--workspace-accent-rgb), 0.22);
  color: var(--workspace-accent);
  box-shadow: inset 0 0 0 1px rgba(var(--workspace-accent-rgb), 0.06);
}

.nav-icon-btn:active {
  transform: scale(0.96);
}

/* ---- 分隔线 ---- */
.header-divider {
  width: 1px;
  height: 24px;
  background: var(--workspace-border);
  margin: 0 8px;
  transition: background-color 0.2s ease;
}

/* ---- 用户触发 ---- */
.user-trigger {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 40px;
  cursor: pointer;
  padding: 4px 8px 4px 4px;
  border: 1px solid transparent;
  border-radius: var(--workspace-radius-md);
  transition:
    background-color 0.16s ease,
    border-color 0.16s ease,
    transform 0.16s ease;
}

.user-trigger:hover {
  background-color: rgba(var(--workspace-accent-rgb), 0.06);
  border-color: rgba(var(--workspace-accent-rgb), 0.12);
}

.user-trigger:active {
  transform: scale(0.96);
}

.user-avatar {
  cursor: pointer;
}

.user-display-name {
  font-size: 13px;
  font-weight: 500;
  color: var(--workspace-text);
  max-width: 100px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  transition: color 0.2s ease;
}

@media (max-width: 768px) {
  .user-display-name {
    display: none;
  }
}

/* ---- 主内容区 ---- */
.main-content {
  padding-top: 56px;
  height: 100dvh;
  display: flex;
  flex-direction: column;
}

.content-wrapper {
  flex: 1;
  width: min(100%, 1280px);
  max-width: 1280px;
  margin: 0 auto;
  padding: 18px 20px 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

/* ---- 底部版权 ---- */
.content-footer {
  text-align: center;
  padding: 12px 18px;
  font-size: 12px;
  color: var(--workspace-text-soft);
  transition: color 0.2s ease;
}

.footer-content {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-wrap: wrap;
  gap: 6px;
}

.footer-separator {
  color: var(--workspace-border);
}

.footer-app-link {
  color: var(--workspace-accent);
  text-decoration: none;
  font-weight: 500;
  transition:
    color 0.16s ease,
    text-decoration-color 0.16s ease;
}

.footer-app-link:hover {
  text-decoration: underline;
}

.footer-version {
  color: var(--workspace-text-soft);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
}

.footer-org-link {
  color: var(--workspace-accent);
  text-decoration: none;
  transition:
    color 0.16s ease,
    text-decoration-color 0.16s ease;
}

.footer-org-link:hover {
  text-decoration: underline;
}

.footer-org-text {
  color: var(--workspace-text-muted);
}

.footer-github-link {
  color: var(--workspace-text-muted);
  display: inline-flex;
  align-items: center;
  transition: color 0.16s ease;
}

.footer-github-link:hover {
  color: var(--workspace-accent);
}

@media (max-width: 768px) {
  .header-inner {
    padding: 0 12px;
  }

  .content-wrapper {
    width: 100%;
    padding: 12px 12px 0;
  }

  .content-footer {
    padding: 10px 12px;
  }
}
</style>
