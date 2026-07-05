import { ref, computed } from 'vue'

// 模块级单例：整个应用生命周期内共享同一份视口状态
const windowWidth = ref(window.innerWidth)

export const VIEWPORT_BREAKPOINT = 768

export function useViewport() {
  const isMobile = computed(() => windowWidth.value < VIEWPORT_BREAKPOINT)

  /** 由 MainLayout 的 resize 事件驱动，更新共享宽度 */
  function sync() {
    windowWidth.value = window.innerWidth
  }

  return { isMobile, windowWidth, sync }
}
