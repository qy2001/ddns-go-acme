<script setup>
import { computed, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { store, saveConfig } from '../store'
import providers from '../data/providers'

const selected = ref(0)
const saving = ref(false)

const dnsConf = computed(() => store.config?.DnsConf || [])
const current = computed(() => dnsConf.value[selected.value])
const providerMeta = computed(() => providers[current.value?.DnsName] || {})
const providerHelp = computed(() => providerMeta.value.helpHtml?.[store.lang === 'en' ? 'en' : 'zh-cn'] || '')
const idLabel = computed(() => providerMeta.value.idLabel || 'ID')
const secretLabel = computed(() => providerMeta.value.secretLabel || 'Secret')
const ifaceNames = computed(() => (store.config?.AllInterfaces || []).map((i) => i.Name))

const providerOptions = computed(() =>
  Object.entries(providers).map(([key, meta]) => ({
    value: key,
    label: meta.name?.[store.lang === 'en' ? 'en' : 'zh-cn'] || key
  }))
)

function providerLabel(key) {
  return providers[key]?.name?.[store.lang === 'en' ? 'en' : 'zh-cn'] || key
}

function addConf() {
  store.config.DnsConf.push({
    Name: '新配置',
    DnsName: 'alidns',
    DnsID: '',
    DnsSecret: '',
    DnsExtParam: '',
    TTL: '600',
    Ipv4Enable: true,
    Ipv4GetType: 'url',
    Ipv4Url: 'https://myip4.ipip.net,https://ddns.oray.com/checkip,https://www.3322.org/dyndns/getip',
    Ipv4NetInterface: '',
    Ipv4Cmd: '',
    Ipv4Domains: '',
    Ipv6Enable: false,
    Ipv6GetType: 'netInterface',
    Ipv6Url: '',
    Ipv6NetInterface: '',
    Ipv6Cmd: '',
    Ipv6Reg: '',
    Ipv6Domains: '',
    HttpInterface: ''
  })
  selected.value = store.config.DnsConf.length - 1
}

async function removeConf(index) {
  try {
    await ElMessageBox.confirm('确定删除该 DNS 配置吗？保存后生效。', '删除配置', { type: 'warning' })
  } catch {
    return
  }
  store.config.DnsConf.splice(index, 1)
  if (selected.value >= store.config.DnsConf.length) {
    selected.value = Math.max(0, store.config.DnsConf.length - 1)
  }
}

async function save() {
  saving.value = true
  try {
    await saveConfig()
    ElMessage.success('保存成功，已触发一次解析更新')
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h2 class="page-title">DNS 解析</h2>
        <p class="page-desc">配置公网 IP 获取方式与域名解析服务商，保存后立即执行一次更新。</p>
      </div>
      <div>
        <el-button @click="addConf">新增配置</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存配置</el-button>
      </div>
    </div>

    <el-empty v-if="!dnsConf.length" description="暂无 DNS 配置">
      <el-button type="primary" @click="addConf">新增配置</el-button>
    </el-empty>

    <div v-else class="dns-layout">
      <el-card class="card conf-list" shadow="never">
        <div
          v-for="(c, i) in dnsConf"
          :key="i"
          class="conf-item"
          :class="{ active: i === selected }"
          @click="selected = i"
        >
          <div class="conf-main">
            <div class="conf-name">{{ c.Name || '未命名配置' }}</div>
            <div class="conf-sub">{{ providerLabel(c.DnsName) }}</div>
          </div>
          <el-button text type="danger" size="small" @click.stop="removeConf(i)">删除</el-button>
        </div>
      </el-card>

      <div v-if="current" class="conf-form">
        <el-card class="card" shadow="never">
          <template #header><b>服务商</b></template>
          <el-form label-position="top" class="form-grid">
            <el-form-item label="配置名称">
              <el-input v-model="current.Name" placeholder="例如：家庭宽带" />
            </el-form-item>

            <el-form-item label="DNS 服务商">
              <el-select v-model="current.DnsName" filterable style="width: 100%">
                <el-option v-for="opt in providerOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
              </el-select>
            </el-form-item>

            <el-form-item>
              <template #label>
                <span>{{ idLabel }}</span>
                <span v-if="providerHelp" class="provider-help" v-html="providerHelp"></span>
              </template>
              <el-input v-model="current.DnsID" type="password" show-password :placeholder="idLabel" />
            </el-form-item>

            <el-form-item :label="secretLabel">
              <el-input v-model="current.DnsSecret" type="password" show-password :placeholder="secretLabel" />
            </el-form-item>

            <el-form-item label="扩展参数（可选）">
              <el-input v-model="current.DnsExtParam" placeholder="部分服务商需要，例如团队 ID" />
            </el-form-item>

            <el-form-item label="TTL（秒）">
              <el-input v-model="current.TTL" placeholder="600" />
            </el-form-item>

            <el-form-item label="请求网卡（可选）">
              <el-select v-model="current.HttpInterface" clearable filterable allow-create style="width: 100%" placeholder="默认网卡">
                <el-option v-for="name in ifaceNames" :key="name" :label="name" :value="name" />
              </el-select>
            </el-form-item>
          </el-form>
        </el-card>

        <el-card class="card" shadow="never">
          <template #header>
            <div class="section-head">
              <b>IPv4</b>
              <el-switch v-model="current.Ipv4Enable" active-text="启用" />
            </div>
          </template>
          <el-form label-position="top" class="form-grid">
            <el-form-item label="获取方式">
              <el-radio-group v-model="current.Ipv4GetType">
                <el-radio-button value="url">接口获取</el-radio-button>
                <el-radio-button value="netInterface">网卡获取</el-radio-button>
                <el-radio-button value="cmd">命令获取</el-radio-button>
              </el-radio-group>
            </el-form-item>

            <el-form-item v-if="current.Ipv4GetType === 'url'" label="IP 接口（多个用英文逗号分隔）">
              <el-input v-model="current.Ipv4Url" placeholder="https://myip4.ipip.net" />
            </el-form-item>

            <el-form-item v-else-if="current.Ipv4GetType === 'netInterface'" label="网卡">
              <el-select v-model="current.Ipv4NetInterface" clearable filterable style="width: 100%" placeholder="选择网卡">
                <el-option v-for="name in ifaceNames" :key="name" :label="name" :value="name" />
              </el-select>
            </el-form-item>

            <el-form-item v-else label="命令">
              <el-input v-model="current.Ipv4Cmd" placeholder="执行命令获取 IP" />
            </el-form-item>

            <el-form-item class="full" label="域名（每行一个）">
              <el-input v-model="current.Ipv4Domains" type="textarea" :rows="4" placeholder="www.example.com&#10;sub:example.com" />
              <div class="hint">支持自动拆分根域名，也可用 子域名:根域名 指定；Cloudflare 等可追加 ?参数。</div>
            </el-form-item>
          </el-form>
        </el-card>

        <el-card class="card" shadow="never">
          <template #header>
            <div class="section-head">
              <b>IPv6</b>
              <el-switch v-model="current.Ipv6Enable" active-text="启用" />
            </div>
          </template>
          <el-form label-position="top" class="form-grid">
            <el-form-item label="获取方式">
              <el-radio-group v-model="current.Ipv6GetType">
                <el-radio-button value="url">接口获取</el-radio-button>
                <el-radio-button value="netInterface">网卡获取</el-radio-button>
                <el-radio-button value="cmd">命令获取</el-radio-button>
              </el-radio-group>
            </el-form-item>

            <el-form-item v-if="current.Ipv6GetType === 'url'" label="IP 接口（多个用英文逗号分隔）">
              <el-input v-model="current.Ipv6Url" placeholder="https://myip6.ipip.net" />
            </el-form-item>

            <el-form-item v-else-if="current.Ipv6GetType === 'netInterface'" label="网卡">
              <el-select v-model="current.Ipv6NetInterface" clearable filterable style="width: 100%" placeholder="选择网卡">
                <el-option v-for="name in ifaceNames" :key="name" :label="name" :value="name" />
              </el-select>
            </el-form-item>

            <el-form-item v-else label="命令">
              <el-input v-model="current.Ipv6Cmd" placeholder="执行命令获取 IP" />
            </el-form-item>

            <el-form-item label="地址正则（可选）">
              <el-input v-model="current.Ipv6Reg" placeholder="用于从网卡多地址中筛选" />
            </el-form-item>

            <el-form-item class="full" label="域名（每行一个）">
              <el-input v-model="current.Ipv6Domains" type="textarea" :rows="4" placeholder="www.example.com" />
            </el-form-item>
          </el-form>
        </el-card>
      </div>
    </div>
  </div>
</template>

<style scoped>
.dns-layout {
  display: grid;
  grid-template-columns: 240px minmax(0, 1fr);
  gap: 16px;
  align-items: start;
}

.conf-list :deep(.el-card__body) {
  padding: 8px;
}

.conf-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 10px 12px;
  border-radius: 10px;
  cursor: pointer;
  transition: background 0.2s;
}

.conf-item:hover {
  background: #f3f6fb;
}

.conf-item.active {
  background: #e8f1ff;
}

.conf-name {
  font-weight: 600;
  font-size: 14px;
}

.conf-sub {
  font-size: 12px;
  color: #6b7280;
  margin-top: 2px;
}

.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

@media (max-width: 860px) {
  .dns-layout {
    grid-template-columns: 1fr;
  }
}
</style>
