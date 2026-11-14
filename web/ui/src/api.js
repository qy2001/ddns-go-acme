import { store } from './store'

// api 统一请求封装，401 时把状态切回未登录
export async function api(url, options = {}) {
  const opts = { credentials: 'same-origin', ...options }
  if (opts.body && typeof opts.body !== 'string') {
    opts.headers = { 'Content-Type': 'application/json', ...(opts.headers || {}) }
    opts.body = JSON.stringify(opts.body)
  }
  const res = await fetch(url, opts)
  if (res.status === 401) {
    store.authenticated = false
    throw new Error('未登录')
  }
  const contentType = res.headers.get('content-type') || ''
  const data = contentType.includes('application/json')
    ? await res.json().catch(() => null)
    : await res.text()
  if (!res.ok) {
    const msg = (data && (data.result || data.Msg)) || (typeof data === 'string' ? data : '') || `请求失败 (${res.status})`
    throw new Error(msg)
  }
  return data
}
