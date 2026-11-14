<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { store, saveConfig } from '../store'
import { api } from '../api'
import providers from '../data/providers'
import CertDeployDialog from '../components/CertDeployDialog.vue'

const list = ref([])
const loading = ref(false)
const dialogVisible = ref(false)
const editingName = ref('')
const form = ref({})
let timer = null

const certSupported = computed(() => store.config?.CertSupportedProviders || ['cloudflare', 'dnspod', 'tencentcloud'])

const dnsOptions = computed(() =>
  (store.config?.DnsConf || [])
    .map((d, index) => ({ index, ...d }))
    .filter((d) => certSupported.value.includes(d.DnsName))
)

function providerLabel(key) {
  return providers[key]?.name?.[store.lang === 'en' ? 'en' : 'zh-cn'] || key
}

function dnsLabel(index) {
  const d = store.config?.DnsConf?.[index]
  if (!d) return `配置 ${index + 1}`
  return `${providerLabel(d.DnsName)} - ${d.Name || '配置 ' + (index + 1)}`
}

function caLabel(ca) {
  return ca === 'letsencrypt-staging' ? "Let's Encrypt 测试环境" : "Let's Encrypt"
}

function statusTag(row) {
  if (row.Issuing) return { type: 'warning', text: '申请中' }
  if (!row.Issued) return { type: 'info', text: '未签发' }
  if (row.DaysLeft <= 0) return { type: 'danger', text: '已过期' }
  if (row.DaysLeft <= 30) return { type: 'warning', text: '即将到期' }
  return { type: 'success', text: '正常' }
}

async function refresh() {
  loading.value = true
  try {
    const data = await api('/api/cert/list')
    list.value = data?.data || []
  } catch (e) {
    // 轮询失败时不打断页面
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  refresh()
  timer = setInterval(refresh, 5000)
})

onUnmounted(() => clearInterval(timer))

function openCreate() {
  if (!dnsOptions.value.length) {
    ElMessage.warning('请先在「DNS 解析」中添加 Cloudflare、DNSPod 或腾讯云配置')
    return
  }
  editingName.value = ''
  form.value = {
    Enable: true,
    Name: '',
    DomainsText: '',
    Email: '',
    CA: 'letsencrypt',
    DNSProvider: dnsOptions.value[0].index,
    KeyType: 'ec256',
    CertDir: '',
    RenewBefore: 30
  }
  dialogVisible.value = true
}

function openEdit(row) {
  const conf = (store.config.CertConf || []).find((c) => c.Name === row.Name)
  if (!conf) {
    ElMessage.error('未找到该证书配置，请刷新页面')
    return
  }
  editingName.value = conf.Name
  form.value = {
    ...conf,
    DomainsText: (conf.Domains || []).join('\n'),
    RenewBefore: conf.RenewBefore || 30
  }
  dialogVisible.value = true
}

async function saveCert(issueAfter) {
  const domains = form.value.DomainsText.split(/\r?\n/).map((s) => s.trim()).filter(Boolean)
  if (!form.value.Name.trim()) {
    ElMessage.warning('请填写配置名称')
    return
  }
  if (!domains.length) {
    ElMessage.warning('请填写证书域名')
    return
  }
  if (!form.value.Email.trim()) {
    ElMessage.warning('请填写 ACME 账号邮箱')
    return
  }

  const item = {
    Enable: form.value.Enable,
    Name: form.value.Name.trim(),
    Domains: domains,
    Email: form.value.Email.trim(),
    CA: form.value.CA,
    DNSProvider: Number(form.value.DNSProvider),
    KeyType: form.value.KeyType,
    CertDir: (form.value.CertDir || '').trim(),
    RenewBefore: Number(form.value.RenewBefore) || 30
  }

  const certConf = store.config.CertConf || (store.config.CertConf = [])
  const idx = certConf.findIndex((c) => c.Name === editingName.value)
  if (idx >= 0) {
    certConf[idx] = item
  } else {
    certConf.push(item)
  }

  try {
    await saveConfig()
    dialogVisible.value = false
    await refresh()
    if (issueAfter) {
      await issue(item.Name)
    } else {
      ElMessage.success('证书配置已保存')
    }
  } catch (e) {
    ElMessage.error(e.message)
  }
}

async function issue(name) {
  try {
    await api('/api/cert/issue', { method: 'POST', body: { Name: name } })
    ElMessage.success('已开始申请，请稍候查看状态')
    await refresh()
  } catch (e) {
    ElMessage.error(e.message)
  }
}

async function removeCert(row) {
  try {
    await ElMessageBox.confirm(`确定删除证书配置「${row.Name}」吗？已签发的证书文件不会被删除。`, '删除配置', { type: 'warning' })
  } catch {
    return
  }
  store.config.CertConf = (store.config.CertConf || []).filter((c) => c.Name !== row.Name)
  try {
    await saveConfig()
    await refresh()
    ElMessage.success('已删除')
  } catch (e) {
    ElMessage.error(e.message)
  }
}

function download(row, format, password = '') {
  const params = new URLSearchParams({ name: row.Name, format, password })
  window.open('/api/cert/download?' + params.toString(), '_blank')
}

async function downloadWithPassword(row, format) {
  try {
    const { value } = await ElMessageBox.prompt('请输入 PFX 导出密码（可留空）', '导出证书', {
      confirmButtonText: '下载',
      cancelButtonText: '取消',
      inputValue: ''
    })
    download(row, format, value || '')
  } catch {
    // 取消
  }
}

async function downloadPem(row) {
  if (!row.Issued) {
    ElMessage.warning('证书尚未签发')
    return
  }
  download(row, 'pem')
}

// ---------- 终端部署 / 查看内容 ----------
const deployVisible = ref(false)
const deployRow = ref(null)
const deployTab = ref('cmd')

function openDeploy(row, tab) {
  deployRow.value = row
  deployTab.value = tab
  deployVisible.value = true
}
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h2 class="page-title">SSL 证书</h2>
        <p class="page-desc">通过 Let's Encrypt 自动申请与续期证书，使用 Cloudflare / DNSPod 的 DNS-01 验证，无需开放 80 端口。</p>
      </div>
      <div>
        <el-button :loading="loading" @click="refresh">刷新状态</el-button>
        <el-button type="primary" @click="openCreate">新建证书</el-button>
      </div>
    </div>

    <el-alert
      v-if="!dnsOptions.length"
      type="info"
      show-icon
      :closable="false"
      title="请先在「DNS 解析」中添加 Cloudflare、DNSPod 或腾讯云配置，证书申请会复用其中的 API 凭证。"
      style="margin-bottom: 16px"
    />

    <el-card class="card" shadow="never">
      <el-table :data="list" v-loading="loading" empty-text="暂无证书配置">
        <el-table-column prop="Name" label="名称" min-width="130" show-overflow-tooltip />
        <el-table-column label="域名" min-width="220">
          <template #default="{ row }">
            <el-tag v-for="d in row.Domains" :key="d" size="small" effect="plain" style="margin: 2px 4px 2px 0">{{ d }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="颁发机构" min-width="150">
          <template #default="{ row }">{{ caLabel(row.CA) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="110">
          <template #default="{ row }">
            <el-tag :type="statusTag(row).type">{{ statusTag(row).text }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="到期时间" min-width="170">
          <template #default="{ row }">
            <span v-if="row.Issued">{{ new Date(row.NotAfter).toLocaleString() }}</span>
            <span v-else class="hint">-</span>
          </template>
        </el-table-column>
        <el-table-column label="剩余天数" width="100">
          <template #default="{ row }">
            <span v-if="row.Issued">{{ row.DaysLeft }}</span>
            <span v-else class="hint">-</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="270" fixed="right">
          <template #default="{ row }">
            <el-button size="small" :disabled="row.Issuing" @click="issue(row.Name)">
              {{ row.Issued ? '续期' : '申请' }}
            </el-button>
            <el-dropdown size="small" trigger="click">
              <el-button size="small" :disabled="!row.Issued">
                下载<el-icon class="el-icon--right"><ArrowDown /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item @click="downloadPem(row)">PEM（Nginx/Apache）</el-dropdown-item>
                  <el-dropdown-item @click="downloadWithPassword(row, 'pfx')">PFX（IIS/Windows）</el-dropdown-item>
                  <el-dropdown-item @click="downloadWithPassword(row, 'zip')">全部格式 ZIP</el-dropdown-item>
                  <el-dropdown-item divided @click="openDeploy(row, 'cmd')">终端部署命令</el-dropdown-item>
                  <el-dropdown-item @click="openDeploy(row, 'content')">查看证书内容</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
            <el-button size="small" text type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button size="small" text type="danger" @click="removeCert(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card v-if="list.some((x) => x.LastError)" class="card" shadow="never">
      <template #header><b>最近错误</b></template>
      <div v-for="row in list.filter((x) => x.LastError)" :key="row.Name" class="error-row">
        <b>{{ row.Name }}</b>
        <span class="hint">{{ row.LastError }}</span>
      </div>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="editingName ? '编辑证书' : '新建证书'" width="640px">
      <el-form label-position="top">
        <div class="form-grid">
          <el-form-item label="配置名称" required>
            <el-input v-model="form.Name" placeholder="例如：NAS 证书" />
          </el-form-item>
          <el-form-item label="ACME 账号邮箱" required>
            <el-input v-model="form.Email" placeholder="用于接收 CA 紧急通知，注册 ACME 账号" />
          </el-form-item>

          <el-form-item label="CA">
            <el-select v-model="form.CA" style="width: 100%">
              <el-option label="Let's Encrypt（正式）" value="letsencrypt" />
              <el-option label="Let's Encrypt（测试环境）" value="letsencrypt-staging" />
            </el-select>
          </el-form-item>

          <el-form-item label="DNS-01 验证配置" required>
            <el-select v-model="form.DNSProvider" style="width: 100%" placeholder="选择 DNS 配置">
              <el-option v-for="opt in dnsOptions" :key="opt.index" :label="dnsLabel(opt.index)" :value="opt.index" />
            </el-select>
          </el-form-item>

          <el-form-item label="私钥类型">
            <el-select v-model="form.KeyType" style="width: 100%">
              <el-option label="EC256（推荐）" value="ec256" />
              <el-option label="EC384" value="ec384" />
              <el-option label="RSA2048" value="rsa2048" />
              <el-option label="RSA4096" value="rsa4096" />
            </el-select>
          </el-form-item>

          <el-form-item label="提前续期天数">
            <el-input-number v-model="form.RenewBefore" :min="1" :max="90" style="width: 100%" />
          </el-form-item>

          <el-form-item class="full" label="证书域名（每行一个，泛域名用 *.example.com）" required>
            <el-input v-model="form.DomainsText" type="textarea" :rows="4" placeholder="example.com&#10;*.example.com" />
          </el-form-item>

          <el-form-item class="full" label="保存目录（可选）">
            <el-input v-model="form.CertDir" placeholder="留空则保存到配置目录下的 .ddns_go_certs" />
          </el-form-item>

          <el-form-item class="full">
            <el-switch v-model="form.Enable" active-text="启用自动申请/续期" />
          </el-form-item>
        </div>
      </el-form>

      <div class="hint">
        泛域名必须使用 DNS-01 验证；首次建议先使用测试环境验证配置，成功后再切换正式环境。
      </div>

      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button @click="saveCert(false)">仅保存</el-button>
        <el-button type="primary" @click="saveCert(true)">保存并申请</el-button>
      </template>
    </el-dialog>

    <CertDeployDialog
      v-model:visible="deployVisible"
      :name="deployRow?.Name || ''"
      :domains="deployRow?.Domains || []"
      :initial-tab="deployTab"
    />
  </div>
</template>

<style scoped>
.error-row {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 6px 0;
  border-bottom: 1px dashed #eef2f8;
}

.error-row:last-child {
  border-bottom: none;
}

.terminal-output :deep(.el-textarea__inner) {
  font-family: 'JetBrains Mono', Consolas, 'Courier New', monospace;
  font-size: 12px;
  line-height: 1.6;
  background: #0f172a;
  color: #d7e3f4;
  border-radius: 10px;
}
</style>
