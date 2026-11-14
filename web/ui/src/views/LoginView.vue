<script setup>
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { api } from '../api'

const emit = defineEmits(['success'])
const { t } = useI18n()

const form = ref({ Username: '', Password: '' })
const loading = ref(false)

async function login() {
  if (!form.value.Username || !form.value.Password) {
    ElMessage.warning(t('请输入用户名和密码'))
    return
  }
  loading.value = true
  try {
    const data = await api('/loginFunc', { method: 'POST', body: form.value })
    if (data && data.Code !== 200) {
      throw new Error(data.Msg || t('登录失败'))
    }
    ElMessage.success(t('登录成功'))
    emit('success')
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-page">
    <div class="login-card">
      <div class="login-brand">
        <div class="logo">D</div>
        <div>
          <h1>DDNS-GO-ACME</h1>
          <p>{{ $t('动态域名解析与 SSL 证书管理') }}</p>
        </div>
      </div>

      <el-form label-position="top" @submit.prevent="login">
        <el-form-item :label="$t('用户名')">
          <el-input v-model="form.Username" size="large" :placeholder="$t('请输入用户名')" autocomplete="username" />
        </el-form-item>
        <el-form-item :label="$t('密码')">
          <el-input
            v-model="form.Password"
            size="large"
            type="password"
            show-password
            :placeholder="$t('请输入密码')"
            autocomplete="current-password"
            @keyup.enter="login"
          />
        </el-form-item>
        <el-button type="primary" size="large" style="width: 100%" :loading="loading" @click="login">{{ $t('登 录') }}</el-button>
      </el-form>

      <p class="login-hint">{{ $t('首次使用请直接输入用户名和密码完成初始化，需在启动后 30 分钟内完成。') }}</p>
    </div>
  </div>
</template>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: radial-gradient(circle at 20% 20%, #dbeafe 0, transparent 45%),
    radial-gradient(circle at 80% 30%, #e0e7ff 0, transparent 40%), #f5f7fb;
  padding: 24px;
}

.login-card {
  width: 380px;
  background: #fff;
  border-radius: 18px;
  padding: 32px;
  box-shadow: 0 20px 60px rgba(15, 23, 42, 0.12);
  border: 1px solid #eef2f8;
}

.login-brand {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-bottom: 26px;
}

.logo {
  width: 48px;
  height: 48px;
  border-radius: 14px;
  background: linear-gradient(135deg, #409eff, #6366f1);
  color: #fff;
  font-size: 24px;
  font-weight: 800;
  display: flex;
  align-items: center;
  justify-content: center;
}

h1 {
  font-size: 22px;
  margin: 0;
}

.login-brand p {
  margin: 4px 0 0;
  color: #6b7280;
  font-size: 13px;
}

.login-hint {
  color: #9ca3af;
  font-size: 12px;
  line-height: 1.7;
  margin: 18px 0 0;
}
</style>
