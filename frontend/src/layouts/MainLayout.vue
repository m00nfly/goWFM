<template>
  <n-layout class="main-layout">
    <!-- 顶部导航栏 - 全宽固定 -->
    <n-layout-header class="top-header">
      <div class="header-brand" @click="router.push('/')">
        <n-icon size="35" color="#3B82F6">
              <folder-open-outline />
        </n-icon>
        <span class="brand-text">{{ orgName || 'goWFM' }}</span>
      </div>
      <div class="header-actions">
        <n-dropdown trigger="click" :options="userDropdownOptions" @select="onUserAction">
          <div class="user-trigger">
            <n-avatar
              round
              :size="32"
              :style="{ backgroundColor: '#1890ff', cursor: 'pointer', fontSize: '14px' }"
            >
              {{ avatarLetter }}
            </n-avatar>
            <span class="user-display-name">{{ displayName }}</span>
          </div>
        </n-dropdown>
      </div>
    </n-layout-header>

    <!-- 侧边栏 + 内容区 -->
    <n-layout has-sider class="body-layout">
      <n-layout-sider
        class="side-nav"
        collapse-mode="width"
        :collapsed-width="64"
        :width="175"
        :collapsed="collapsed"
        show-trigger="bar"
        @collapse="collapsed = true"
        @expand="collapsed = false"
      >
        <div class="sider-inner">
          <div class="sider-menu-area">
            <n-menu
              :value="activeMenuKey"
              :collapsed="collapsed"
              :collapsed-icon-size="22"
              :options="menuOptions"
              @update:value="onMenuSelect"
            />
          </div>
        </div>
      </n-layout-sider>

      <n-layout-content class="main-content">
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
              <n-icon :size="14"><logo-github /></n-icon>
            </a>
            <span v-if="version" class="footer-version">ver: {{ version }}</span>
          </div>
        </div>
      </n-layout-content>
    </n-layout>
  </n-layout>
</template>

<script setup lang="ts">
import { ref, computed, h, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import {
  NLayout, NLayoutSider, NLayoutHeader, NLayoutContent,
  NMenu, NDropdown, NIcon, NAvatar,
} from 'naive-ui'
import type { MenuOption } from 'naive-ui'
import {
  FolderOpenOutline,
  ShareSocialOutline,
  DocumentTextOutline,
  PeopleOutline,
  SettingsOutline,
  CogOutline,
  PersonCircleOutline,
  LogOutOutline,
  LogoGithub,
} from '@vicons/ionicons5'
import { useUserStore } from '@/stores/user'
import api from '@/api'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()

const collapsed = ref(false)
const version = ref('')
const orgName = ref('')
const orgLink = ref('')
const appLink = ref('https://gowfm.dev')
const appGithub = ref('https://github.com/m00nfly/gowfm')

// ---------- 用户显示 ----------

const displayName = computed(() =>
  userStore.user?.display_name || userStore.user?.username || '用户'
)

const avatarLetter = computed(() => {
  const name = displayName.value
  return name ? name.charAt(0).toUpperCase() : 'U'
})

// ---------- 菜单配置 ----------

const menuOptions = computed<MenuOption[]>(() => {
  const items: MenuOption[] = [
    {
      label: '文件管理',
      key: '/',
      icon: () => h(NIcon, null, () => h(FolderOpenOutline)),
    },
  ]

  if (userStore.user?.is_admin) {
    items.push({
      label: '分享管理',
      key: '/admin/shares',
      icon: () => h(NIcon, null, () => h(ShareSocialOutline)),
    })
  } else if (userStore.hasPermission(8)) {
    items.push({
      label: '我的分享',
      key: '/shares',
      icon: () => h(NIcon, null, () => h(ShareSocialOutline)),
    })
  }

  if (userStore.hasPermission(16)) {
    items.push({
      label: '操作日志',
      key: '/logs',
      icon: () => h(NIcon, null, () => h(DocumentTextOutline)),
    })
  }

  if (userStore.user?.is_admin) {
    items.push({
      label: '用户管理',
      key: '/admin/users',
      icon: () => h(NIcon, null, () => h(PeopleOutline)),
    })
    items.push({
      label: '系统设置',
      key: '/admin/settings',
      icon: () => h(NIcon, null, () => h(CogOutline)),
    })
  }

  return items
})

// 高亮的菜单 key
const activeMenuKey = computed(() => {
  const p = route.path
  if (p.startsWith('/admin/shares')) return '/admin/shares'
  if (p.startsWith('/admin/users')) return '/admin/users'
  if (p.startsWith('/admin/settings')) return '/admin/settings'
  if (p === '/shares') return '/shares'
  if (p === '/logs') return '/logs'
  if (p === '/settings') return '/settings'
  return '/'
})

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

function onMenuSelect(key: string) {
  router.push(key)
}

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
  try {
    const res = await api.get('/api/config/info')
    orgName.value = res.data.org_name || ''
    orgLink.value = res.data.org_link || ''
    version.value = res.data.version || ''
  } catch { /* ignore */ }
})
</script>

<style scoped>
.main-layout {
  min-height: 100vh;
}

/* ---- 顶部导航栏 ---- */
.top-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 56px;
  padding: 0 24px;
  background: #fff;
  border-bottom: 1px solid #e8e8e8;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.05);
  position: sticky;
  top: 0;
  z-index: 100;
}

.header-brand {
  display: flex;
  align-items: center;
  cursor: pointer;
  user-select: none;
}

.brand-text {
  font-size: 20px;
  font-weight: 600;
  color: #1890ff;
  margin-left: 10px;
}

.header-actions {
  display: flex;
  align-items: center;
}

.user-trigger {
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 6px;
  transition: background-color 0.2s;
}

.user-trigger:hover {
  background-color: #f5f5f5;
}

.user-display-name {
  font-size: 14px;
  color: #333;
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* ---- 下方布局 ---- */
.body-layout {
  height: calc(100vh - 56px);
}

/* ---- 侧边栏 ---- */
.side-nav {
  background: #fff !important;
  border-right: 1px solid #e8e8e8 !important;
}

.sider-inner {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.sider-menu-area {
  flex: 1;
  overflow-y: auto;
  padding-top: 8px;
}

/* 菜单选中态: 圆角 + 文字加粗 + 浅色背景 */
.side-nav :deep(.n-menu-item-content--selected) {
  font-weight: 600;
}

.side-nav :deep(.n-menu-item-content) {
  position: relative;
  font-size: 14px;
}

.side-nav :deep(.n-menu-item-content__icon) {
  margin-right: 12px !important;
  margin-left: 8px !important;
}

/* ---- 主内容区 ---- */
.main-content {
  background: #f0f2f5;
  display: flex;
  flex-direction: column;
  height: calc(100vh - 56px);
  overflow: hidden;
}

.content-wrapper {
  flex: 1;
  height: calc(100vh - 135px);
  width: 95%;
  margin: 0 auto;
  padding: 24px 24px 12px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

/* ---- 底部版权 ---- */
.content-footer {
  text-align: center;
  padding: 12px;
  font-size: 12px;
  color: #999;
}

.footer-content {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.footer-separator {
  color: #d9d9d9;
}

.footer-app-link {
  color: #1890ff;
  text-decoration: none;
  font-weight: 500;
}

.footer-app-link:hover {
  text-decoration: underline;
}

.footer-version {
  color: #bbb;
  font-size: 11px;
}

.footer-org-link {
  color: #1890ff;
  text-decoration: none;
}

.footer-org-link:hover {
  text-decoration: underline;
}

.footer-org-text {
  color: #666;
}

.footer-github-link {
  color: #666;
  display: inline-flex;
  align-items: center;
  transition: color 0.2s;
}

.footer-github-link:hover {
  color: #333;
}
</style>
