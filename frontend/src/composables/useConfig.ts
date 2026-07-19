import { ref } from 'vue'
import api from '@/api'

export interface PublicConfig {
  needs_setup: boolean
  site_name: string
  site_link: string
  version: string
  login_bg_url: string
  default_theme: string
  theme_color: string
  custom_logo: string
  enable_captcha: boolean
  totp_trust_days: number
	email_active: boolean
	allow_email_password_reset: boolean
	allow_email_share: boolean
}

// 模块级缓存：整个应用生命周期内只请求一次
const config = ref<PublicConfig | null>(null)
let configPromise: Promise<void> | null = null

export function useConfig() {
  /** 获取站点配置（仅首次调用发起请求） */
	async function fetchConfig(force = false) {
		if (force) configPromise = null
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

  return { config, fetchConfig }
}
