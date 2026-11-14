<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import en from 'element-plus/es/locale/lang/en'
import { Document, SwitchButton } from '@element-plus/icons-vue'
import { store, loadConfig } from './store'
import { api } from './api'
import LoginView from './views/LoginView.vue'

const route = useRoute()
const router = useRouter()

const logsVisible = ref(false)
const logs = ref([])
const clearing = ref(false)
let logTimer = null

const locale = computed(() => (store.lang === 'en' ? en : zhCn))
const version = computed(() => store.config?.Version || 'DEV')

onMounted(async () => {
  try {
    await loadConfig()
  } catch (e) {
    ElMessage.error(e.message)
  }
})

onUnmounted(() => {
  clearInterval(logTimer)
})

async function onLoginSuccess() {
  await loadConfig()
  router.replace('/dns')
}

async function openLogs() {
  logsVisible.value = true
  await refreshLogs()
  clearInterval(logTimer)
  logTimer = setInterval(refreshLogs, 3000)
}

function closeLogs() {
  logsVisible.value = false
  clearInterval(logTimer)
}

async function refreshLogs() {
  try {
    const data = await api('/logs')
    logs.value = Array.isArray(data) ? data : []
  } catch (e) {
    // 忽略轮询错误
  }
}

async function clearLogs() {
  clearing.value = true
  try {
    await api('/clearLog', { method: 'POST' })
    logs.value = []
  } finally {
    clearing.value = false
  }
}

async function changeLang(lang) {
  try {
    await api('/setLang', { method: 'POST', body: { Lang: lang } })
    store.lang = lang
    store.config.Lang = lang
    ElMessage.success(lang === 'en' ? 'Language switched' : '语言已切换')
  } catch (e) {
    ElMessage.error(e.message)
  }
}

async function logout() {
  try {
    await fetch('/logout', { credentials: 'same-origin' })
  } finally {
    store.authenticated = false
    store.config = null
    closeLogs()
  }
}
</script>

<template>
  <el-config-provider :locale="locale">
    <LoginView v-if="!store.authenticated" @success="onLoginSuccess" />

    <el-container v-else class="layout">
      <el-aside width="220px" class="aside">
        <div class="brand">
          <div class="brand-logo">D</div>
          <div>
            <div class="brand-name">DDNS-GO-ACME</div>
            <div class="brand-ver">{{ version }}</div>
          </div>
        </div>

        <el-menu :default-active="route.path" router class="nav-menu">
          <el-menu-item index="/dns">
            <el-icon><Connection /></el-icon>
            <span>DNS 解析</span>
          </el-menu-item>
          <el-menu-item index="/certs">
            <el-icon><Lock /></el-icon>
            <span>SSL 证书</span>
          </el-menu-item>
          <el-menu-item index="/settings">
            <el-icon><Setting /></el-icon>
            <span>系统设置</span>
          </el-menu-item>
        </el-menu>

        <div class="aside-footer">
          <el-select :model-value="store.lang" size="small" style="width: 100%" @change="changeLang">
            <el-option label="简体中文" value="zh" />
            <el-option label="English" value="en" />
          </el-select>
        </div>
      </el-aside>

      <el-container>
        <el-header class="header">
          <div class="header-title">{{ route.path === '/certs' ? 'SSL 证书' : route.path === '/settings' ? '系统设置' : 'DNS 解析' }}</div>
          <div class="header-actions">
            <el-button text :icon="Document" @click="openLogs">日志</el-button>
            <el-button text :icon="SwitchButton" @click="logout">退出</el-button>
          </div>
        </el-header>

        <el-main class="main">
          <router-view />
        </el-main>
      </el-container>
    </el-container>

    <el-drawer v-model="logsVisible" title="运行日志" size="560px" @close="closeLogs">
      <div style="margin-bottom: 12px">
        <el-button size="small" :loading="clearing" @click="clearLogs">清空日志</el-button>
      </div>
      <div class="log-view">{{ logs.join('') || '暂无日志' }}</div>
    </el-drawer>
  </el-config-provider>
</template>

<style scoped>
.layout {
  height: 100vh;
}

.aside {
  background: #0b1220;
  color: #e5e7eb;
  display: flex;
  flex-direction: column;
  padding: 18px 12px;
}

.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 4px 10px 18px;
}

.brand-logo {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  background: linear-gradient(135deg, #409eff, #6366f1);
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 800;
  color: #fff;
  font-size: 18px;
}

.brand-name {
  font-weight: 700;
  letter-spacing: 0.5px;
}

.brand-ver {
  font-size: 12px;
  color: #94a3b8;
}

.nav-menu {
  background: transparent;
  border-right: none;
  flex: 1;
}

.nav-menu :deep(.el-menu-item) {
  color: #cbd5e1;
  border-radius: 10px;
  margin-bottom: 6px;
}

.nav-menu :deep(.el-menu-item.is-active) {
  background: rgba(64, 158, 255, 0.18);
  color: #93c5fd;
}

.nav-menu :deep(.el-menu-item:hover) {
  background: rgba(148, 163, 184, 0.12);
}

.aside-footer {
  padding: 12px 6px 0;
}

.header {
  background: #fff;
  border-bottom: 1px solid #eaeef5;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.header-title {
  font-weight: 650;
  font-size: 16px;
}

.main {
  background: var(--ddns-bg);
  padding: 0;
}
</style>
