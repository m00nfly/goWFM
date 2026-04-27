<template>
  <div class="share-access-page">
    <n-card class="share-card">
      <n-result v-if="error" status="error" :title="error" description="分享链接无效或已过期" />
      <n-spin v-else :show="loading">
        <div v-if="fileInfo" class="share-info">
          <h2>{{ fileInfo.name }}</h2>
          <p>大小: {{ formatSize(fileInfo.size) }}</p>
          <n-button type="primary" @click="downloadFile">下载文件</n-button>
        </div>
      </n-spin>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { NCard, NResult, NSpin, NButton, useMessage } from 'naive-ui'
import api from '@/api'
import { formatSize } from '@/utils/format'

const route = useRoute()
const message = useMessage()
const loading = ref(true)
const error = ref('')
const fileInfo = ref<{ name: string; size: number } | null>(null)

onMounted(async () => {
  loading.value = true
  try {
    const token = route.params.token as string
    const res = await api.get(`/share/${token}/info`)
    fileInfo.value = {
      name: res.data.file_name,
      size: res.data.file_size,
    }
  } catch (err: any) {
    error.value = err.response?.data?.error || '获取分享信息失败'
  } finally {
    loading.value = false
  }
})

async function downloadFile() {
  try {
    const token = route.params.token as string
    const res = await api.get(`/share/${token}`, { responseType: 'blob' })
    const url = window.URL.createObjectURL(res.data)
    const a = document.createElement('a')
    a.href = url
    a.download = fileInfo.value?.name || 'file'
    a.click()
    window.URL.revokeObjectURL(url)
  } catch {
    message.error('下载失败')
  }
}
</script>

<style scoped>
.share-access-page {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background: #f5f5f5;
}
.share-card {
  width: 480px;
}
.share-info {
  text-align: center;
}
.share-info h2 {
  margin-bottom: 8px;
}
.share-info p {
  color: #666;
  margin-bottom: 16px;
}
</style>