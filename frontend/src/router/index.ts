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
      path: '/',
      name: 'files',
      component: () => import('@/views/FileExplorer.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/shares',
      name: 'shares',
      component: () => import('@/views/MySharesView.vue'),
      meta: { requiresAuth: true, permission: 8 },
    },
    {
      path: '/logs',
      name: 'logs',
      component: () => import('@/views/LogsView.vue'),
      meta: { requiresAuth: true, permission: 16 },
    },
    {
      path: '/admin/users',
      name: 'users',
      component: () => import('@/views/UserManagementView.vue'),
      meta: { requiresAuth: true, admin: true },
    },
    {
      path: '/settings',
      name: 'settings',
      component: () => import('@/views/UserSettingsView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/share/:token',
      name: 'share-access',
      component: () => import('@/views/ShareAccessView.vue'),
      meta: { public: true },
    },
  ],
})

router.beforeEach(async (to, _from, next) => {
  const userStore = useUserStore()

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

  if (to.meta.public) {
    if (to.name === 'login' && userStore.user) {
      return next('/')
    }
    return next()
  }

  if (to.meta.requiresAuth && !userStore.user) {
    return next('/login')
  }

  if (to.meta.admin && !userStore.user?.is_admin) {
    return next('/')
  }

  if (to.meta.permission && !userStore.hasPermission(to.meta.permission as number)) {
    return next('/')
  }

  next()
})

export default router