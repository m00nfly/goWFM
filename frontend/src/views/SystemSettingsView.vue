<template>
  <div class="sys-settings-wrap">
    <n-card :bordered="false" title="系统设置" class="sys-settings-card">
      <n-tabs type="line" animated class="sys-settings-tabs">
        <n-tab-pane name="basic" tab="基础设置">
          <BasicSettingsTab />
        </n-tab-pane>
        <n-tab-pane name="appearance" tab="外观设置">
          <AppearanceSettingsTab />
        </n-tab-pane>
        <n-tab-pane name="security" tab="安全设置">
          <SecuritySettingsTab />
        </n-tab-pane>
        <n-tab-pane name="log" tab="日志设置">
          <LogSettingsTab />
        </n-tab-pane>
        <n-tab-pane name="email" tab="邮件设置">
          <EmailSettingsTab />
        </n-tab-pane>
        <n-tab-pane name="share" tab="分享设置">
          <ShareSettingsTab />
        </n-tab-pane>
      </n-tabs>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { NCard, NTabs, NTabPane } from 'naive-ui'
import BasicSettingsTab from './settings/BasicSettingsTab.vue'
import SecuritySettingsTab from './settings/SecuritySettingsTab.vue'
import LogSettingsTab from './settings/LogSettingsTab.vue'
import EmailSettingsTab from './settings/EmailSettingsTab.vue'
import AppearanceSettingsTab from './settings/AppearanceSettingsTab.vue'
import ShareSettingsTab from './settings/ShareSettingsTab.vue'
</script>

<style scoped>
/* 约束整体高度，防止页面撑出视口 */
.sys-settings-wrap {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.sys-settings-card {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.sys-settings-card :deep(.n-card-content) {
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.sys-settings-tabs {
  flex: 1;
  min-height: 0;
}

/* 核心：用 vh 给 pane-wrapper 一个独立于父链的明确高度上限，触发 overflow-y 滚动 */
.sys-settings-tabs :deep(.n-tabs-pane-wrapper) {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}
</style>
