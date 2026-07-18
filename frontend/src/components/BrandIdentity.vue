<template>
  <div
    class="brand-identity"
    :class="[
      `brand-identity--${variant}`,
      {
        'brand-identity--wordmark': isWordmark,
        'brand-identity--loading': logoStatus === 'loading',
      },
    ]"
  >
    <div class="brand-identity__logo-frame">
      <img
        v-if="hasCustomLogo && logoStatus !== 'error'"
        :key="logo"
        :src="logo"
        class="brand-identity__logo"
        :alt="`${displayName} Logo`"
        @load="handleLogoLoad"
        @error="handleLogoError"
      />
      <span v-else class="brand-identity__fallback" aria-hidden="true">
        <n-icon :size="variant === 'header' ? 20 : 25"><FolderOpenOutline /></n-icon>
      </span>
    </div>

    <div v-if="showCopy" class="brand-identity__copy">
      <span v-if="kicker" class="brand-identity__kicker">{{ kicker }}</span>
      <span class="brand-identity__name" :title="displayName">{{ displayName }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { NIcon } from 'naive-ui'
import { FolderOpenOutline } from '@vicons/ionicons5'

type BrandVariant = 'header' | 'login'
type LogoStatus = 'idle' | 'loading' | 'loaded' | 'error'

const WORDMARK_ASPECT_RATIO = 2.2

const props = withDefaults(defineProps<{
  logo?: string
  name?: string
  kicker?: string
  variant?: BrandVariant
}>(), {
  logo: '',
  name: 'goWFM',
  kicker: '',
  variant: 'header',
})

const logoStatus = ref<LogoStatus>('idle')
const logoAspectRatio = ref(0)

const hasCustomLogo = computed(() => props.logo.trim().length > 0)
const displayName = computed(() => props.name.trim() || 'goWFM')
const isWordmark = computed(() =>
  logoStatus.value === 'loaded' && logoAspectRatio.value >= WORDMARK_ASPECT_RATIO,
)
const showCopy = computed(() => !isWordmark.value)

watch(
  () => props.logo,
  (logo) => {
    logoAspectRatio.value = 0
    logoStatus.value = logo.trim() ? 'loading' : 'idle'
  },
  { immediate: true },
)

function handleLogoLoad(event: Event) {
  const image = event.currentTarget as HTMLImageElement
  if (!image.naturalWidth || !image.naturalHeight) {
    handleLogoError()
    return
  }
  logoAspectRatio.value = image.naturalWidth / image.naturalHeight
  logoStatus.value = 'loaded'
}

function handleLogoError() {
  logoAspectRatio.value = 0
  logoStatus.value = 'error'
}
</script>

<style scoped>
.brand-identity {
  min-width: 0;
  max-width: 100%;
  display: flex;
  align-items: center;
}

.brand-identity__logo-frame {
  min-width: 0;
  flex: 0 1 auto;
  display: inline-flex;
  align-items: center;
  justify-content: flex-start;
  overflow: hidden;
}

.brand-identity__logo {
  display: block;
  width: auto;
  height: auto;
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
  transition-property: opacity;
  transition-duration: 150ms;
  transition-timing-function: ease-out;
}

.brand-identity--loading .brand-identity__logo {
  opacity: 0;
}

.brand-identity__copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.brand-identity__name {
  display: block;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.brand-identity--header {
  height: 100%;
  gap: 10px;
}

.brand-identity--header .brand-identity__logo-frame {
  height: 36px;
  max-width: min(160px, 28vw);
}

.brand-identity--header:not(.brand-identity--wordmark) .brand-identity__logo-frame {
  width: 36px;
  max-width: 36px;
  flex: 0 0 36px;
}

.brand-identity--header .brand-identity__fallback {
  width: 36px;
  height: 36px;
  flex: 0 0 36px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--workspace-radius-md);
  color: var(--workspace-on-accent);
  background: var(--workspace-accent);
  box-shadow: 0 8px 18px rgba(var(--workspace-accent-rgb), 0.2);
  transition-property: transform;
  transition-duration: 150ms;
  transition-timing-function: ease-out;
}

.brand-identity--header .brand-identity__name {
  color: var(--workspace-text);
  font-size: 18px;
  font-weight: 700;
  line-height: 1.15;
}

.brand-identity--login {
  position: relative;
  z-index: 1;
  gap: 14px;
}

.brand-identity--login .brand-identity__logo-frame {
  width: 50px;
  height: 50px;
  max-width: 50px;
  flex: 0 0 50px;
}

.brand-identity--login.brand-identity--wordmark {
  width: 100%;
}

.brand-identity--login.brand-identity--wordmark .brand-identity__logo-frame {
  width: auto;
  height: 64px;
  max-width: min(280px, 100%);
  flex: 0 1 auto;
}

.brand-identity--login .brand-identity__fallback {
  width: 50px;
  height: 50px;
  flex: 0 0 50px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 16px;
  color: var(--accent);
  background: var(--panel-strong);
  box-shadow: 0 0 0 1px var(--line), 0 14px 34px rgba(16, 32, 51, 0.12);
}

.brand-identity--login .brand-identity__copy {
  gap: 3px;
}

.brand-identity--login .brand-identity__kicker {
  color: var(--soft-ink);
  font-size: 12px;
  line-height: 1.35;
}

.brand-identity--login .brand-identity__name {
  color: var(--page-ink);
  font-size: 16px;
  font-weight: 760;
  line-height: 1.3;
}

@media (max-width: 640px) {
  .brand-identity--header .brand-identity__copy {
    display: none;
  }

  .brand-identity--header .brand-identity__logo-frame {
    max-width: min(140px, 42vw);
  }
}

@media (max-width: 560px) {
  .brand-identity--login.brand-identity--wordmark .brand-identity__logo-frame {
    height: 56px;
    max-width: min(240px, 100%);
  }
}

@media (prefers-reduced-motion: reduce) {
  .brand-identity__logo,
  .brand-identity__fallback {
    transition-duration: 0.001ms;
  }
}
</style>
