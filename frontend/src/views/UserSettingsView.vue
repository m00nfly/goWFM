<template>
  <n-card title="个人设置">
      <n-form :model="form" label-placement="left" label-width="80">
        <n-form-item label="显示名称">
          <n-input v-model:value="form.display_name" />
        </n-form-item>
        <n-form-item label="邮箱">
          <n-input v-model:value="form.email" />
        </n-form-item>
        <n-button type="primary" :loading="saving" @click="handleSave">保存</n-button>
      </n-form>

      <n-divider>修改密码</n-divider>
      <n-form :model="pwForm" label-placement="left" label-width="80">
        <n-form-item label="当前密码">
          <n-input v-model:value="pwForm.current_password" type="password" />
        </n-form-item>
        <n-form-item label="新密码">
          <n-input v-model:value="pwForm.new_password" type="password" />
        </n-form-item>
        <n-button type="primary" :loading="pwSaving" @click="handlePasswordChange">修改密码</n-button>
      </n-form>
    </n-card>
</template>

<script setup lang="ts">
import { reactive, ref, onMounted } from 'vue'
import { NCard, NForm, NFormItem, NInput, NButton, NDivider, useMessage } from 'naive-ui'
import api from '@/api'
import { useUserStore } from '@/stores/user'

const message = useMessage()
const userStore = useUserStore()
const saving = ref(false)
const pwSaving = ref(false)

const form = reactive({ display_name: '', email: '' })
const pwForm = reactive({ current_password: '', new_password: '' })

onMounted(() => {
  if (userStore.user) {
    form.display_name = userStore.user.display_name
    form.email = userStore.user.email
  }
})

async function handleSave() {
  saving.value = true
  try {
    await api.put('/api/users/me', form)
    await userStore.fetchMe()
    message.success('保存成功')
  } catch (err: any) {
    message.error(err.response?.data?.error || '保存失败')
  } finally {
    saving.value = false
  }
}

async function handlePasswordChange() {
  pwSaving.value = true
  try {
    await api.put('/api/users/me/password', pwForm)
    message.success('密码修改成功')
    pwForm.current_password = ''
    pwForm.new_password = ''
  } catch (err: any) {
    message.error(err.response?.data?.error || '密码修改失败')
  } finally {
    pwSaving.value = false
  }
}
</script>