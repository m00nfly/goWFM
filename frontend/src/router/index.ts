import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { useConfig } from '@/composables/useConfig'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue'),
      meta: { public: true },
    },
    {
      path: '/',
      component: () => import('@/layouts/MainLayout.vue'),
      children: [
        {
          path: '',
          name: 'files',
          component: () => import('@/views/FileExplorer.vue'),
          meta: { requiresAuth: true },
        },
        {
          path: 'setup',
          name: 'setup',
          component: () => import('@/views/SetupView.vue'),
          meta: { public: true },
        },
        {
          path: 'share/:token',
          name: 'share-access',
          component: () => import('@/views/ShareAccessView.vue'),
          meta: { public: true },
        },
        {
          path: 'shares',
          name: 'shares',
          component: () => import('@/views/ShareManagementView.vue'),
          meta: { permission: 8 },
        },
        {
          path: 'logs',
          name: 'logs',
          component: () => import('@/views/LogsView.vue'),
          meta: { permission: 16 },
        },
        {
          path: 'admin/users',
          name: 'users',
          component: () => import('@/views/UserManagementView.vue'),
          meta: { admin: true },
        },
        {
          path: 'admin/settings',
          name: 'admin-settings',
          component: () => import('@/views/SystemSettingsView.vue'),
          meta: { admin: true },
        },
        {
          path: 'settings',
          name: 'settings',
          component: () => import('@/views/UserSettingsView.vue'),
        },
      ],
    },
  ],
})

router.beforeEach(async (to, _from, next) => {
  const userStore = useUserStore()
  const { fetchConfig, config } = useConfig()

  // Resolve meta from all matched route records (supports nested routes)
  const requiresAuth = to.matched.some(r => r.meta.requiresAuth)
  const isPublic = to.matched.some(r => r.meta.public)
  const requiresAdmin = to.matched.some(r => r.meta.admin)
  const permissionMeta = to.matched.find(r => r.meta.permission !== undefined)
  const requiredPermission = permissionMeta?.meta.permission as number | undefined

  if (!userStore.initialized) {
    await userStore.fetchMe()

    // 未登录时，检查是否需要跳转到初始化页面
    if (!userStore.user && to.name !== 'setup') {
      try {
        await fetchConfig()
        if (config.value?.needs_setup) {
          return next('/setup')
        }
      } catch {
        // ignore
      }
    }
  }

  // public 路由直接放行（含 Guest 模式下的 MainLayout 子页面）
  if (isPublic) {
	if (to.name === 'login' && userStore.user && !to.query.reset_token) {
      return next('/')
    }
    return next()
  }

  // 需要登录但未登录
  if (!userStore.user) {
    return next('/login')
  }

  // 管理员强制启用但尚未绑定时，只允许进入个人设置完成绑定。
  if ((userStore.user.totp_reset_required || (userStore.user.totp_forced && !userStore.user.totp_enabled)) && to.name !== 'settings') {
    return next('/settings')
  }

  if (requiresAuth && !userStore.user) {
    return next('/login')
  }

  if (requiresAdmin && !userStore.user?.is_admin) {
    return next('/')
  }

  if (requiredPermission && !userStore.hasPermission(requiredPermission)) {
    return next('/')
  }

  next()
})

export default router
