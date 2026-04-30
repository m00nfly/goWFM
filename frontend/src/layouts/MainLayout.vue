<template>
  <n-layout class="main-layout" has-sider>
    <!-- 可折叠侧边栏 -->
    <n-layout-sider
      bordered
      collapse-mode="width"
      :collapsed-width="64"
      :width="220"
      :collapsed="collapsed"
      show-trigger="bar"
      @collapse="collapsed = true"
      @expand="collapsed = false"
    >
      <div class="sidebar-inner">
        <div class="sidebar-top">
          <div class="sidebar-header" @click="router.push('/')">
            <n-icon size="24" color="#3B82F6">
              <folder-open-outline />
            </n-icon>
            <span v-show="!collapsed" class="sidebar-title">goWFM</span>
          </div>
          <n-menu
            :value="activeMenuKey"
            :collapsed="collapsed"
            :collapsed-icon-size="22"
            :options="menuOptions"
            @update:value="onMenuSelect"
          />
        </div>
        <div v-show="!collapsed" class="sidebar-footer">
          <template v-if="orgLink">
            <a :href="orgLink" target="_blank" class="org-link">{{ orgName || orgLink }}</a>
          </template>
          <template v-else-if="orgName">
            <span class="org-text">{{ orgName }}</span>
          </template>
          <span class="copyright">&copy; 2026 goWFM</span>
          <span v-if="version" class="version">{{ version }}</span>
        </div>
      </div>
    </n-layout-sider>

    <!-- 右侧主体 -->
    <n-layout>
      <!-- 顶部栏 -->
      <n-layout-header bordered class="main-header">
        <div class="header-left">
          <span class="site-name">{{ orgName || 'goWFM' }}</span>
          <span v-if="pageTitle" class="breadcrumb-sep">&gt;</span>
          <span v-if="pageTitle" class="page-title">{{ pageTitle }}</span>
        </div>
        <div class="header-right">
          <n-dropdown trigger="click" :options="userDropdownOptions" @select="onUserAction">
            <n-button text class="user-btn">
              <template #icon>
                <n-icon size="20"><person-circle-outline /></n-icon>
              </template>
              <span class="user-name">{{ userStore.user?.display_name || userStore.user?.username || '用户' }}</span>
            </n-button>
          </n-dropdown>
        </div>
      </n-layout-header>

      <!-- 内容区 -->
      <n-layout-content class="main-content">
        <router-view />
      </n-layout-content>

    </n-layout>
  </n-layout>
</template>

<script setup lang="ts">
import { ref, computed, h, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import {
  NLayout, NLayoutSider, NLayoutHeader, NLayoutContent,
  NMenu, NButton, NDropdown, NIcon,
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
} from '@vicons/ionicons5'
import { useUserStore } from '@/stores/user'
import api from '@/api'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()

const collapsed = ref(false)
const orgName = ref('')
const orgLink = ref('')
const version = ref('')

// ---------- 菜单配置 ----------

const menuOptions = computed<MenuOption[]>(() => {
  const items: MenuOption[] = [
    {
      label: '文件管理',
      key: '/',
      icon: () => h(NIcon, null, () => h(FolderOpenOutline)),
    },
  ]

  if (userStore.hasPermission(8)) {
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
  if (p.startsWith('/admin/users')) return '/admin/users'
  if (p.startsWith('/admin/settings')) return '/admin/settings'
  if (p === '/shares') return '/shares'
  if (p === '/logs') return '/logs'
  if (p === '/settings') return '/settings'
  return '/'
})

// 页面标题
const pageTitle = computed(() => {
  const map: Record<string, string> = {
    '/': '文件管理',
    '/shares': '我的分享',
    '/logs': '操作日志',
    '/admin/users': '用户管理',
    '/admin/settings': '系统设置',
    '/settings': '个人设置',
  }
  return map[activeMenuKey.value] || ''
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

/* ---- 侧边栏 ---- */
.sidebar-inner {
  display: flex;
  flex-direction: column;
  height: 100%;
}
.sidebar-top {
  flex: 1;
  overflow-y: auto;
}
.sidebar-header {
  display: flex;
  align-items: center;
  gap: 10px;
  height: 56px;
  padding: 0 20px;
  cursor: pointer;
  user-select: none;
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
}
.sidebar-title {
  font-size: 18px;
  font-weight: 700;
  color: #3B82F6;
  white-space: nowrap;
}

/* ---- 顶部栏 ---- */
.main-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  height: 56px;
}
.header-left {
  display: flex;
  align-items: center;
  gap: 8px;
}
.site-name {
  font-size: 16px;
  font-weight: 600;
  color: #3B82F6;
}
.breadcrumb-sep {
  font-size: 14px;
  color: #bbb;
  font-weight: 400;
}
.page-title {
  font-size: 16px;
  font-weight: 600;
  color: #333;
}
.header-right {
  display: flex;
  align-items: center;
}
.user-btn {
  font-size: 14px;
}
.user-name {
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* ---- 内容区 ---- */
.main-content {
  padding: 24px;
  background: #f5f7fa;
  min-height: calc(100vh - 56px);
}

/* ---- 侧边栏底部信息 ---- */
.sidebar-footer {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  padding: 12px 16px;
  border-top: 1px solid rgba(0, 0, 0, 0.06);
  font-size: 12px;
  color: #999;
  line-height: 1.6;
}
.org-link {
  color: #3B82F6;
  text-decoration: none;
  font-size: 12px;
}
.org-link:hover {
  text-decoration: underline;
}
.org-text {
  color: #999;
}
.copyright {
  color: #bbb;
}
.version {
  color: #ccc;
  font-size: 11px;
}
</style>
