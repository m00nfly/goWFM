<template>
  <div class="share-access-page">
    <div class="share-container">
      <n-result v-if="error" status="error" :title="error" description="分享链接无效或已过期" />
      <n-spin v-else :show="loading">
        <div v-if="files.length > 0" class="file-grid">
          <n-card v-for="file in files" :key="file.name" class="file-card" hoverable>
            <div class="file-info">
              <h3 class="file-name">{{ file.name }}</h3>
              <p class="file-size">{{ formatSize(file.size) }}</p>
            </div>
            <div class="file-actions">
              <n-button type="primary" size="small" @click="downloadFile(file)">下载文件</n-button>
              <n-tooltip trigger="hover">
                <template #trigger>
                  <n-button size="small" @click="copyDownloadLink(file)">复制链接</n-button>
                </template>
                复制链接使用wget/curl直接下载
              </n-tooltip>
            </div>
          </n-card>
        </div>
      </n-spin>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { NCard, NResult, NSpin, NButton, NTooltip, useMessage } from 'naive-ui'
import api from '@/api'
import { formatSize } from '@/utils/format'
import { copyToClipboard } from '@/utils/clipboard'

const route = useRoute()
const message = useMessage()
const loading = ref(true)
const error = ref('')
const files = ref<Array<{ name: string; size: number }>>([])

onMounted(async () => {
  loading.value = true
  try {
    const token = route.params.token as string
    const res = await api.get(`/share/${token}/info`)
    files.value = (res.data.files || []).map((f: any) => ({
      name: f.file_name,
      size: f.file_size,
    }))
  } catch (err: any) {
    error.value = err.response?.data?.error || '获取分享信息失败'
  } finally {
    loading.value = false
  }
})

function downloadFile(file: { name: string; size: number }) {
  const token = route.params.token as string
  const url = `/share/${token}/${encodeURIComponent(file.name)}`
  window.location.href = url
}

async function copyDownloadLink(file: { name: string; size: number }) {
  const token = route.params.token as string
  const url = `${window.location.origin}/share/${token}/${encodeURIComponent(file.name)}`
  const ok = await copyToClipboard(url)
  if (ok) {
    message.success('下载链接已复制到剪贴板')
  } else {
    message.error('复制失败！')
  }
}
</script>

<style scoped>
.share-access-page {
  display: flex;
  justify-content: center;
  align-items: flex-start;
  min-height: 100vh;
  padding: 60px 20px;
  background: #f5f5f5;
}

.share-container {
  width: 100%;
  max-width: 900px;
}

.file-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 16px;
}

/* 单文件时居中 */
.file-grid:has(:only-child) {
  max-width: 320px;
  margin: 0 auto;
}

.file-card {
  text-align: center;
}

.file-name {
  font-size: 15px;
  font-weight: 500;
  margin-bottom: 6px;
  word-break: break-all;
}

.file-size {
  color: #666;
  font-size: 13px;
  margin-bottom: 12px;
}

.file-actions {
  display: flex;
  justify-content: center;
  gap: 8px;
}
</style>
