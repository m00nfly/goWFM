import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

type ThemeMode = 'light' | 'dark' | 'system'

const STORAGE_KEY = 'gowfm-theme'

export const useThemeStore = defineStore('theme', () => {
  // 用户选择的模式: light / dark / system
  const mode = ref<ThemeMode>(loadStoredMode())

  // 系统是否偏好暗色
  const systemPrefersDark = ref(getSystemPreference())

  // 实际是否暗色
  const isDark = computed(() => {
    if (mode.value === 'system') return systemPrefersDark.value
    return mode.value === 'dark'
  })

  // 监听系统主题变化
  let mediaQuery: MediaQueryList | null = null

  function init() {
    // 监听系统主题变化
    if (typeof window !== 'undefined' && window.matchMedia) {
      mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
      mediaQuery.addEventListener('change', onSystemThemeChange)
    }
    applyTheme()
  }

  function onSystemThemeChange(e: MediaQueryListEvent) {
    systemPrefersDark.value = e.matches
    applyTheme()
  }

  function setMode(newMode: ThemeMode) {
    mode.value = newMode
    localStorage.setItem(STORAGE_KEY, newMode)
    applyTheme()
  }

  /** 切换亮/暗：当前暗→亮，当前亮→暗，直接设为 light/dark（不再走 system） */
  function toggleTheme() {
    if (mode.value === 'system') {
      // 首次切换：从 system 跳到明确选择
      setMode(isDark.value ? 'light' : 'dark')
    } else {
      setMode(isDark.value ? 'light' : 'dark')
    }
  }

  /** 将 dark class 应用到 <html> */
  function applyTheme() {
    if (typeof document === 'undefined') return
    const html = document.documentElement
    if (isDark.value) {
      html.classList.add('dark')
    } else {
      html.classList.remove('dark')
    }
  }

  function cleanup() {
    if (mediaQuery) {
      mediaQuery.removeEventListener('change', onSystemThemeChange)
    }
  }

  return { mode, isDark, init, setMode, toggleTheme, cleanup }
})

function loadStoredMode(): ThemeMode {
  try {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored === 'light' || stored === 'dark' || stored === 'system') return stored
  } catch { /* ignore */ }
  return 'system'
}

function getSystemPreference(): boolean {
  try {
    return window.matchMedia('(prefers-color-scheme: dark)').matches
  } catch {
    return false
  }
}
