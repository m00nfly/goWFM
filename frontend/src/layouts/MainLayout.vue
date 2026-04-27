<template>
  <n-layout class="main-layout">
    <n-layout-header bordered class="main-header">
      <div class="header-left">
        <span class="logo" @click="router.push('/')">WFM</span>
        <span v-if="orgName" class="org-name">{{ orgName }}</span>
      </div>
      <div class="header-right">
        <n-dropdown :options="userMenuOptions" @select="onUserMenuSelect">
          <n-button text>{{ userStore.user?.display_name || userStore.user?.username || '用户' }}</n-button>
        </n-dropdown>
      </div>
    </n-layout-header>
    <n-layout-content class="main-content">
      <slot />
    </n-layout-content>
    <n-layout-footer bordered class="main-footer">
      <span>
        <template v-if="orgLink">
          <a :href="orgLink" target="_blank" class="org-link">{{ orgName || orgLink }}</a>
        </template>
        <template v-else>{{ orgName }}</template>
        &copy; {{ new Date().getFullYear() }} WFM - 文件管理系统
      </span>
    </n-layout-footer>
  </n-layout>
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { NLayout, NLayoutHeader, NLayoutContent, NLayoutFooter, NButton, NDropdown } from 'naive-ui'
import { useUserStore } from '@/stores/user'
import api from '@/api'

const router = useRouter()
const userStore = useUserStore()
const orgName = ref('')
const orgLink = ref('')

const userMenuOptions = computed(() => {
  const options = [
    { label: '个人设置', key: 'settings' },
    { label: '退出登录', key: 'logout' },
  ]
  if (userStore.user?.is_admin) {
    options.splice(1, 0, { label: '用户管理', key: 'users' })
  }
  if (userStore.hasPermission(8)) {
    options.splice(1, 0, { label: '我的分享', key: 'shares' })
  }
  if (userStore.hasPermission(16)) {
    options.splice(1, 0, { label: '操作日志', key: 'logs' })
  }
  return options
})

onMounted(async () => {
  try {
    const res = await api.get('/api/config/info')
    orgName.value = res.data.org_name || ''
    orgLink.value = res.data.org_link || ''
  } catch { /* ignore */ }
})

async function onUserMenuSelect(key: string) {
  if (key === 'settings') {
    router.push('/settings')
  } else if (key === 'users') {
    router.push('/admin/users')
  } else if (key === 'shares') {
    router.push('/shares')
  } else if (key === 'logs') {
    router.push('/logs')
  } else if (key === 'logout') {
    try { await api.post('/api/auth/logout') } catch {}
    userStore.logout()
    router.push('/login')
  }
}
</script>

<style scoped>
.main-layout {
  min-height: 100vh;
}
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
  gap: 12px;
}
.logo {
  font-size: 20px;
  font-weight: 700;
  color: #3B82F6;
  cursor: pointer;
}
.org-name {
  font-size: 14px;
  color: #666;
}
.main-content {
  padding: 24px;
}
.main-footer {
  text-align: center;
  padding: 12px;
  color: #999;
  font-size: 13px;
}
.org-link {
  color: #3B82F6;
  text-decoration: none;
}
.org-link:hover {
  text-decoration: underline;
}
</style>