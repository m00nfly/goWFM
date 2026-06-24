<template>
  <n-config-provider :theme="themeStore.isDark ? darkTheme : undefined" :theme-overrides="themeOverrides">
    <n-dialog-provider>
      <n-message-provider>
        <router-view />
      </n-message-provider>
    </n-dialog-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { NConfigProvider, NMessageProvider, NDialogProvider, darkTheme } from 'naive-ui'
import type { GlobalThemeOverrides } from 'naive-ui'
import { useThemeStore } from '@/stores/theme'
import api from '@/api'

const themeStore = useThemeStore()

// 主题色（从外观设置中读取）
const themeColor = ref('#3b82f6')

// 颜色工具：HEX → HSL → 调整亮度 → HEX
function hexToHsl(hex: string): [number, number, number] {
  const r = parseInt(hex.slice(1, 3), 16) / 255
  const g = parseInt(hex.slice(3, 5), 16) / 255
  const b = parseInt(hex.slice(5, 7), 16) / 255
  const max = Math.max(r, g, b), min = Math.min(r, g, b)
  let h = 0, s = 0
  const l = (max + min) / 2
  if (max !== min) {
    const d = max - min
    s = l > 0.5 ? d / (2 - max - min) : d / (max + min)
    if (max === r) h = ((g - b) / d + (g < b ? 6 : 0)) / 6
    else if (max === g) h = ((b - r) / d + 2) / 6
    else h = ((r - g) / d + 4) / 6
  }
  return [Math.round(h * 360), Math.round(s * 100), Math.round(l * 100)]
}

function hslToHex(h: number, s: number, l: number): string {
  s /= 100; l /= 100
  const a = s * Math.min(l, 1 - l)
  const f = (n: number) => {
    const k = (n + h / 30) % 12
    const color = l - a * Math.max(Math.min(k - 3, 9 - k, 1), -1)
    return Math.round(255 * color).toString(16).padStart(2, '0')
  }
  return `#${f(0)}${f(8)}${f(4)}`
}

function deriveColors(hex: string) {
  const [h, s, l] = hexToHsl(hex)
  return {
    base: hex,
    hover: hslToHex(h, Math.min(s + 10, 100), Math.min(l + 12, 90)),
    pressed: hslToHex(h, s, Math.max(l - 10, 15)),
  }
}

// 响应式主题覆盖
const themeOverrides = computed<GlobalThemeOverrides>(() => {
  const colors = deriveColors(themeColor.value)
  return {
    common: {
      primaryColor: colors.base,
      primaryColorHover: colors.hover,
      primaryColorPressed: colors.pressed,
      primaryColorSuppl: colors.base,
      borderRadius: '8px',
    },
  }
})

// 应用 CSS 变量到 :root
function applyCSSVar() {
  const colors = deriveColors(themeColor.value)
  const root = document.documentElement
  root.style.setProperty('--theme-color', colors.base)
  root.style.setProperty('--theme-color-hover', colors.hover)
  root.style.setProperty('--theme-color-pressed', colors.pressed)
  root.style.setProperty('--theme-color-rgb', hexToRgbString(colors.base))
}

function hexToRgbString(hex: string): string {
  const r = parseInt(hex.slice(1, 3), 16)
  const g = parseInt(hex.slice(3, 5), 16)
  const b = parseInt(hex.slice(5, 7), 16)
  return `${r}, ${g}, ${b}`
}

watch(themeColor, applyCSSVar)

onMounted(async () => {
  themeStore.init()
  try {
    const res = await api.get('/api/config/info')
    if (res.data.theme_color) {
      themeColor.value = res.data.theme_color
    }
  } catch { /* ignore */ }
})

onUnmounted(() => {
  themeStore.cleanup()
})
</script>
