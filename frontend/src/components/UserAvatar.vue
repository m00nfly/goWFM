<template>
  <n-avatar
    v-if="avatar"
    round
    :size="size"
    :src="avatar"
    :img-props="{ alt: `${displayName}的头像`, draggable: false }"
    class="app-user-avatar is-custom-avatar"
    :aria-label="`${displayName}的头像`"
  />
  <n-avatar
    v-else
    round
    :size="size"
    class="app-user-avatar is-default-avatar"
    :aria-label="`${displayName}的头像`"
  >
    <span class="avatar-initials" :style="{ fontSize: `${initialFontSize}px` }" aria-hidden="true">{{ initials }}</span>
  </n-avatar>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { NAvatar } from 'naive-ui'

const props = withDefaults(defineProps<{
  avatar?: string
  name?: string
  size?: number
}>(), {
  avatar: '',
  name: '',
  size: 40,
})

const displayName = computed(() => props.name.trim() || '用户')
const initialFontSize = computed(() => Math.max(11, Math.round(props.size * 0.28)))
const initials = computed(() => {
  const value = displayName.value
  const words = value.split(/\s+/).filter(Boolean)
  if (words.length > 1) {
    return words.slice(0, 2).map(word => Array.from(word)[0]).join('').toUpperCase()
  }
  return Array.from(value).slice(0, 2).join('').toUpperCase()
})
</script>

<style scoped>
.app-user-avatar {
  flex: 0 0 auto;
  color: var(--workspace-accent);
  background:
    linear-gradient(145deg, rgba(var(--workspace-accent-rgb), 0.18), rgba(var(--workspace-accent-rgb), 0.07)),
    var(--workspace-surface) !important;
  outline: 1px solid rgba(0, 0, 0, 0.1);
  outline-offset: -1px;
  box-shadow:
    0 1px 2px rgba(39, 55, 82, 0.08),
    0 6px 16px rgba(var(--workspace-accent-rgb), 0.12);
}

.app-user-avatar :deep(img) {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.avatar-initials {
  font-weight: 750;
  letter-spacing: 0.02em;
  line-height: 1;
  user-select: none;
}

:global(.dark .app-user-avatar) {
  outline-color: rgba(255, 255, 255, 0.1);
  box-shadow: 0 6px 16px rgba(2, 6, 23, 0.24);
}
</style>
