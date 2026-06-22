<template>
  <n-card :bordered="false" title="系统设置">
    <n-tabs type="line" animated>
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
