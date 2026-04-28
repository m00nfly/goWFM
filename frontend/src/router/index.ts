import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '@/stores/user'
import api from '@/api'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/setup',
      name: 'setup',
      component: () => import('@/views/SetupView.vue'),
      meta: { public: true },
    },
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue'),
      meta: { public: true },
    },
    {
      path: '/share/:token',
      name: 'share-access',
      component: () => import('@/views/ShareAccessView.vue'),
      meta: { public: true },
    },
    {
      path: '/',
      component: () => import('@/layouts/MainLayout.vue'),
      meta: { requiresAuth: true },
      children: [
        {
          path: '',
          name: 'files',
          component: () => import('@/views/FileExplorer.vue'),
        },
        {
          path: 'shares',
          name: 'shares',
          component: () => import('@/views/MySharesView.vue'),
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

  // Resolve meta from all matched route records (supports nested routes)
  const requiresAuth = to.matched.some(r => r.meta.requiresAuth)
  const isPublic = to.matched.some(r => r.meta.public)
  const requiresAdmin = to.matched.some(r => r.meta.admin)
  const permissionMeta = to.matched.find(r => r.meta.permission !== undefined)
  const requiredPermission = permissionMeta?.meta.permission as number | undefined

  if (!userStore.initialized) {
    await userStore.fetchMe()

    if (!userStore.user) {
      try {
        const res = await api.get('/api/setup/status')
        if (res.data.needs_setup) {
          return next('/setup')
        }
      } catch {
        // ignore
      }
    }
  }

  if (isPublic) {
    if (to.name === 'login' && userStore.user) {
      return next('/')
    }
    return next()
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