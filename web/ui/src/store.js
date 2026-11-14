import { reactive } from 'vue'

export const store = reactive({
  ready: false,
  authenticated: false,
  config: null,
  lang: 'zh'
})

// loadConfig 拉取完整配置，401 表示未登录
export async function loadConfig() {
  const res = await fetch('/api/config', { credentials: 'same-origin' })
  if (res.status === 401) {
    store.authenticated = false
    store.ready = true
    return false
  }
  if (!res.ok) {
    throw new Error('加载配置失败')
  }
  const data = await res.json()
  data.DnsConf = data.DnsConf || []
  data.CertConf = data.CertConf || []
  store.config = data
  store.lang = data.Lang || 'zh'
  store.authenticated = true
  store.ready = true
  return true
}

// saveConfig 保存全部配置，password 为空表示不修改密码
export async function saveConfig(password = '') {
  const c = store.config
  if (!c) throw new Error('配置尚未加载')
  const payload = {
    Username: c.Username,
    Password: password,
    Lang: c.Lang,
    NotAllowWanAccess: c.NotAllowWanAccess,
    WebhookURL: c.WebhookURL || '',
    WebhookRequestBody: c.WebhookRequestBody || '',
    WebhookHeaders: c.WebhookHeaders || '',
    DnsConf: c.DnsConf || [],
    CertConf: c.CertConf || []
  }
  const res = await fetch('/save', {
    method: 'POST',
    credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  })
  const data = await res.json().catch(() => ({ result: '保存失败' }))
  if (data.result !== 'ok') {
    throw new Error(data.result || '保存失败')
  }
  await loadConfig()
}
