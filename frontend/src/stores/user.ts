import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '@/api'

interface User {
  id: number
  username: string
  display_name: string
  email: string
  is_admin: boolean
  permissions: number
  totp_enabled: boolean
  totp_forced: boolean
  totp_reset_required: boolean
  share_stats?: {
    expired: number
    valid: number
  }
}

export const useUserStore = defineStore('user', () => {
  const user = ref<User | null>(null)
  const initialized = ref(false)

  // ---------- 分享统计 ----------

  const shareExpiredCount = computed(() => user.value?.share_stats?.expired ?? 0)
  const shareValidCount = computed(() => user.value?.share_stats?.valid ?? 0)

  /** 分享创建成功后，本地 +1 valid */
  function onShareCreated() {
    if (user.value?.share_stats) {
      user.value.share_stats.valid++
    }
  }

  /** 分享删除成功后，根据状态本地 -1 */
  function onShareDeleted(status: 'valid' | 'expired') {
    if (!user.value?.share_stats) return
    if (status === 'expired') {
      user.value.share_stats.expired = Math.max(0, user.value.share_stats.expired - 1)
    } else {
      user.value.share_stats.valid = Math.max(0, user.value.share_stats.valid - 1)
    }
  }

  // ---------- 基础方法 ----------

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

  return {
    user, initialized,
    shareExpiredCount, shareValidCount,
    onShareCreated, onShareDeleted,
    fetchMe, logout, hasPermission,
  }
})
