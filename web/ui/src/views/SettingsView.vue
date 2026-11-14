<script setup>
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { store, saveConfig } from '../store'
import { setLocale } from '../i18n'
import { api } from '../api'

const { t } = useI18n()
const password = ref('')
const saving = ref(false)
const testing = ref(false)

function onLangChange(lang) {
  store.lang = lang
  setLocale(lang)
}

async function save() {
  saving.value = true
  try {
    await saveConfig(password.value)
    password.value = ''
    ElMessage.success(t('保存成功'))
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    saving.value = false
  }
}

async function testWebhook() {
  if (!store.config.WebhookURL) {
    ElMessage.warning(t('请先填写 Webhook URL'))
    return
  }
  testing.value = true
  try {
    await api('/webhookTest', {
      method: 'POST',
      body: {
        URL: store.config.WebhookURL,
        RequestBody: store.config.WebhookRequestBody,
        Headers: store.config.WebhookHeaders
      }
    })
    ElMessage.success(t('已发送测试请求，请查看日志确认结果'))
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    testing.value = false
  }
}
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h2 class="page-title">{{ $t('系统设置') }}</h2>
        <p class="page-desc">{{ $t('账号安全、访问控制与 Webhook 通知。') }}</p>
      </div>
      <el-button type="primary" :loading="saving" @click="save">{{ $t('保存设置') }}</el-button>
    </div>

    <el-card class="card" shadow="never">
      <template #header><b>{{ $t('账号') }}</b></template>
      <el-form label-position="top" class="form-grid">
        <el-form-item :label="$t('用户名')">
          <el-input v-model="store.config.Username" :placeholder="$t('登录用户名')" />
        </el-form-item>
        <el-form-item :label="$t('新密码')">
          <el-input v-model="password" type="password" show-password :placeholder="$t('留空则不修改密码')" />
        </el-form-item>
      </el-form>
      <div class="hint">{{ $t('密码会以 bcrypt 形式保存；修改密码后当前登录状态仍然有效。') }}</div>
    </el-card>

    <el-card class="card" shadow="never">
      <template #header><b>{{ $t('访问控制') }}</b></template>
      <el-form label-position="top" class="form-grid">
        <el-form-item :label="$t('禁止公网访问')">
          <el-switch v-model="store.config.NotAllowWanAccess" />
          <div class="hint">{{ $t('开启后仅允许内网 IP 访问管理页面，强烈建议保持开启。') }}</div>
        </el-form-item>
        <el-form-item :label="$t('界面语言')">
          <el-select v-model="store.config.Lang" style="width: 100%" @change="onLangChange">
            <el-option label="简体中文" value="zh" />
            <el-option label="English" value="en" />
          </el-select>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card class="card" shadow="never">
      <template #header>
        <div class="section-head">
          <b>{{ $t('Webhook 通知') }}</b>
          <el-button size="small" :loading="testing" @click="testWebhook">{{ $t('测试') }}</el-button>
        </div>
      </template>
      <el-form label-position="top">
        <el-form-item label="Webhook URL">
          <el-input v-model="store.config.WebhookURL" placeholder="https://example.com/webhook" />
        </el-form-item>
        <el-form-item :label="$t('请求体（为空则发送 GET）')">
          <el-input
            v-model="store.config.WebhookRequestBody"
            type="textarea"
            :rows="4"
            :placeholder="$t('支持变量：#{ipv4Addr} #{ipv4Result} #{ipv4Domains} #{ipv6Addr} #{ipv6Result} #{ipv6Domains} #{timestamp}')"
          />
        </el-form-item>
        <el-form-item :label="$t('请求头')">
          <el-input
            v-model="store.config.WebhookHeaders"
            type="textarea"
            :rows="2"
            :placeholder="$t('每行一个，例如 Authorization: Bearer xxx')"
          />
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<style scoped>
.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
</style>
