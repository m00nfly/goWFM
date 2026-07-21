<template>
  <div class="workspace-form-scroll settings-tab-scroll">
    <n-spin :show="loading">
      <n-form class="settings-tab-form" label-placement="left" label-width="170px" :show-feedback="false" :model="form">
        <section class="settings-section">
          <header class="settings-section-header">
            <h2>磁盘文件扫描</h2>
            <p>管理共享目录统计的手动校准与后台定时扫描策略</p>
          </header>
          <div class="settings-section-body">
            <n-alert type="info" :show-icon="true" class="settings-alert">
              goWFM 每次启动时都会完整扫描共享目录，日常通过 goWFM 进行的文件操作会即时更新统计。
            </n-alert>

            <n-form-item label="当前扫描状态">
              <div class="scan-status-line">
                <n-tag :type="status.scanning ? 'warning' : status.last_error ? 'error' : 'success'" size="small">
                  {{ status.scanning ? '扫描中' : status.last_error ? '扫描异常' : status.ready ? '统计就绪' : '等待扫描' }}
                </n-tag>
                <span>{{ statusText }}</span>
              </div>
            </n-form-item>

            <n-form-item label="立即扫描">
              <div class="scan-action-line">
                <n-button :loading="status.scanning" secondary type="primary" @click="runScan">立即扫描共享目录</n-button>
                <span class="workspace-inline-note">文件数量较多时可能持续较长时间，但不会阻塞 Dashboard 读取已有统计。</span>
              </div>
            </n-form-item>

            <n-form-item label="后台定时自动扫描">
              <n-switch :value="form.auto_scan_enabled" @update:value="handleAutoScanChange" />
              <span class="workspace-inline-note">默认关闭</span>
            </n-form-item>

            <n-form-item v-if="form.auto_scan_enabled" label="自动扫描频率">
              <n-input-number v-model:value="form.interval_hours" :min="1" :max="720" :precision="0" style="width: 180px" />
              <span class="workspace-inline-note">小时 / 次，默认每 1 小时</span>
            </n-form-item>
          </div>
        </section>

        <footer class="settings-tab-actions">
          <n-button type="primary" :loading="saving" @click="saveSettings">保存设置</n-button>
        </footer>
      </n-form>
    </n-spin>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { NAlert, NButton, NForm, NFormItem, NInputNumber, NSpin, NSwitch, NTag, useDialog, useMessage } from 'naive-ui'
import api from '@/api'

interface ScanStatus {
  ready: boolean
  scanning: boolean
  last_reason: string
  last_error: string
  last_started_at: string | null
  last_completed_at: string | null
  next_scan_at: string | null
}

const message = useMessage()
const dialog = useDialog()
const loading = ref(false)
const saving = ref(false)
const form = ref({ auto_scan_enabled: false, interval_hours: 1 })
const status = ref<ScanStatus>({
  ready: false, scanning: false, last_reason: '', last_error: '',
  last_started_at: null, last_completed_at: null, next_scan_at: null,
})
let pollTimer: ReturnType<typeof setTimeout> | null = null

const statusText = computed(() => {
  if (status.value.scanning) return '正在重新统计共享目录，请稍候。'
  if (status.value.last_error) return status.value.last_error
  if (status.value.last_completed_at) return `上次完成：${formatTime(status.value.last_completed_at)}`
  return '尚无可用的扫描结果。'
})

function formatTime(value: string) {
  return new Date(value).toLocaleString('zh-CN', { hour12: false })
}

async function loadData() {
  loading.value = true
  try {
    const [configResponse, statusResponse] = await Promise.all([
      api.get('/api/admin/config/scan'),
      api.get('/api/admin/storage-scan/status'),
    ])
    Object.assign(form.value, configResponse.data)
    status.value = statusResponse.data
    if (status.value.scanning) scheduleStatusPoll()
  } catch (error: any) {
    message.error(error.response?.data?.error || '获取磁盘扫描设置失败')
  } finally {
    loading.value = false
  }
}

function handleAutoScanChange(enabled: boolean) {
  if (!enabled) {
    form.value.auto_scan_enabled = false
    return
  }
  dialog.warning({
    title: '确认启用后台定时扫描？',
    content: '该功能适用于管理员或其它程序会绕过 goWFM，直接在系统底层修改共享目录的情况。完整扫描会遍历全部文件与目录，并产生磁盘 I/O；文件数量越大，性能影响越明显。若所有文件操作都通过 goWFM 完成，则无需启用。',
    positiveText: '确认启用',
    negativeText: '保持关闭',
    maskClosable: false,
    onPositiveClick: () => { form.value.auto_scan_enabled = true },
  })
}

async function saveSettings() {
  saving.value = true
  try {
    await api.put('/api/admin/config/scan', form.value)
    message.success('磁盘扫描设置已保存')
  } catch (error: any) {
    message.error(error.response?.data?.error || '保存失败')
  } finally {
    saving.value = false
  }
}

async function runScan() {
  try {
    await api.post('/api/admin/storage-scan/run')
    status.value.scanning = true
    message.success('磁盘扫描已开始')
    scheduleStatusPoll()
  } catch (error: any) {
    if (error.response?.status === 409) {
      status.value.scanning = true
      scheduleStatusPoll()
      return
    }
    message.error(error.response?.data?.error || '启动扫描失败')
  }
}

function scheduleStatusPoll() {
  if (pollTimer) clearTimeout(pollTimer)
  pollTimer = setTimeout(async () => {
    try {
      const response = await api.get('/api/admin/storage-scan/status')
      status.value = response.data
    } finally {
      if (status.value.scanning) scheduleStatusPoll()
    }
  }, 1000)
}

onMounted(loadData)
onUnmounted(() => { if (pollTimer) clearTimeout(pollTimer) })
</script>

<style scoped>
.scan-status-line,
.scan-action-line {
  min-width: 0;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  color: var(--workspace-text-muted);
  font-size: 12px;
}

@media (max-width: 768px) {
  .scan-action-line { align-items: flex-start; flex-direction: column; }
}
</style>
