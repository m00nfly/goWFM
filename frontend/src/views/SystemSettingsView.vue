<template>
  <div class="workspace-page sys-settings-wrap">
    <section class="workspace-surface sys-settings-surface">
      <header class="workspace-header">
        <div class="workspace-title-block">
          <h1 class="workspace-title">系统设置</h1>
          <p class="workspace-subtitle">集中维护基础、外观、安全、日志、邮件、分享与磁盘扫描策略</p>
        </div>
      </header>

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
        <n-tab-pane name="scan" tab="磁盘扫描">
          <ScanSettingsTab />
        </n-tab-pane>
      </n-tabs>
    </section>
  </div>
</template>

<script setup lang="ts">
import { NTabs, NTabPane } from 'naive-ui'
import BasicSettingsTab from './settings/BasicSettingsTab.vue'
import SecuritySettingsTab from './settings/SecuritySettingsTab.vue'
import LogSettingsTab from './settings/LogSettingsTab.vue'
import EmailSettingsTab from './settings/EmailSettingsTab.vue'
import AppearanceSettingsTab from './settings/AppearanceSettingsTab.vue'
import ShareSettingsTab from './settings/ShareSettingsTab.vue'
import ScanSettingsTab from './settings/ScanSettingsTab.vue'
</script>

<style scoped>
.sys-settings-tabs {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.sys-settings-tabs :deep(.n-tabs-nav) {
  padding: 0 12px;
  border-bottom: 1px solid var(--workspace-border-soft);
  background: rgba(var(--workspace-accent-rgb), 0.025);
}

.sys-settings-tabs :deep(.n-tabs-tab) {
  padding-top: 10px;
  padding-bottom: 10px;
  font-weight: 650;
}

.sys-settings-tabs :deep(.n-tabs-pane-wrapper) {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

.sys-settings-tabs :deep(.n-tab-pane) {
  min-height: 100%;
  padding: 0;
}
</style>
