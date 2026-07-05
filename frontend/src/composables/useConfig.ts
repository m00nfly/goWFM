import { ref } from 'vue'
import api from '@/api'

// 模块级缓存：整个应用生命周期内只请求一次
const config = ref<Record<string, any> | null>(null)
const setupStatus = ref<{ needs_setup: boolean } | null>(null)
let configPromise: Promise<void> | null = null
let setupPromise: Promise<void> | null = null

export function useConfig() {
  /** 获取站点配置（仅首次调用发起请求） */
  async function fetchConfig() {
    if (configPromise) return configPromise
    configPromise = (async () => {
      try {
        const res = await api.get('/api/config/info')
        config.value = res.data
      } catch {
        // ignore
      }
    })()
    return configPromise
  }

  /** 获取系统初始化状态（仅首次调用发起请求） */
  async function fetchSetupStatus() {
    if (setupPromise) return setupPromise
    setupPromise = (async () => {
      try {
        const res = await api.get('/api/setup/status')
        setupStatus.value = res.data
      } catch {
        // ignore
      }
    })()
    return setupPromise
  }

  return { config, setupStatus, fetchConfig, fetchSetupStatus }
}
