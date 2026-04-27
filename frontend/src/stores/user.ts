import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '@/api'

interface User {
  id: number
  username: string
  display_name: string
  email: string
  is_admin: boolean
  permissions: number
}

export const useUserStore = defineStore('user', () => {
  const user = ref<User | null>(null)
  const initialized = ref(false)

  async function fetchMe() {
    try {
      const res = await api.get('/api/auth/me')
      user.value = res.data
    } catch {
      user.value = null
    } finally {
      initialized.value = true
    }
  }

  function logout() {
    user.value = null
    initialized.value = false
  }

  function hasPermission(bit: number): boolean {
    if (!user.value) return false
    if (user.value.is_admin) return true
    return (user.value.permissions & bit) !== 0
  }

  return { user, initialized, fetchMe, logout, hasPermission }
})