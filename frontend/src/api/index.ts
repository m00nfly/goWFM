import axios from 'axios'

const api = axios.create({
  baseURL: '',
  withCredentials: true,
})

api.interceptors.response.use(
  (res) => res,
  (err) => {
    const status = err.response?.status
    const configUrl = err.config?.url || ''

    if (status === 401) {
      const isAuthEndpoint = configUrl.includes('/api/auth/login') || configUrl.includes('/api/setup') || configUrl.includes('/api/auth/me')
      const isPublicPage = window.location.pathname === '/login' || window.location.pathname.startsWith('/share/')
      if (!isAuthEndpoint && !isPublicPage) {
        window.location.href = '/login'
        return new Promise(() => {})
      }
    }

    if (status === 403) {
      if (err.response?.data?.code === 'TOTP_SETUP_REQUIRED') {
        if (window.location.pathname !== '/settings') window.location.href = '/settings'
        return Promise.reject(err)
      }
      const msg = (window as any).$message
      if (msg?.error) {
        msg.error('权限不足')
      } else {
        console.warn('权限不足')
      }
    }

    return Promise.reject(err)
  },
)

export default api
