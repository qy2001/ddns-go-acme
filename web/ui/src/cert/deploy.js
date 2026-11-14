// 服务器类型与对应的证书文件组合、终端命令生成

export const SERVER_TYPES = [
  { value: 'nginx', label: 'Nginx（证书链 + 私钥）' },
  { value: 'apache', label: 'Apache（证书 + 中间链 + 私钥）' },
  { value: 'haproxy', label: 'HAProxy（证书链与私钥合并单文件）' },
  { value: 'caddy', label: 'Caddy（证书链 + 私钥）' },
  { value: 'iis', label: 'IIS（PFX 单文件）' },
  { value: 'tomcat', label: 'Tomcat（PFX 单文件）' },
  { value: 'pem', label: '通用 PEM（证书链 + 私钥）' }
]

export function isPfxServer(server) {
  return server === 'iis' || server === 'tomcat'
}

// 各服务器需要的文件：key 内容类型、label 展示名、name 默认文件名、sensitive 是否敏感
export function filesFor(server, domain) {
  switch (server) {
    case 'apache':
      return [
        { key: 'cert', label: '服务器证书', name: `${domain}.crt` },
        { key: 'chain', label: '中间证书链', name: `${domain}.chain.crt` },
        { key: 'privkey', label: '私钥', name: `${domain}.key`, sensitive: true }
      ]
    case 'haproxy':
      return [{ key: 'merged', label: '合并证书（证书链+私钥）', name: `${domain}.pem`, sensitive: true }]
    case 'iis':
    case 'tomcat':
      return [{ key: 'pfx', label: 'PFX 证书（含私钥）', name: `${domain}.pfx`, sensitive: true }]
    case 'nginx':
    case 'caddy':
      return [
        { key: 'fullchain', label: '证书链', name: `${domain}.crt` },
        { key: 'privkey', label: '私钥', name: `${domain}.key`, sensitive: true }
      ]
    default:
      return [
        { key: 'fullchain', label: '证书链', name: `${domain}.pem` },
        { key: 'privkey', label: '私钥', name: `${domain}.key`, sensitive: true }
      ]
  }
}

// 单个文件的文本内容（PFX 返回 base64 字符串）
export function fileText(file, contents) {
  if (!contents) return ''
  if (file.key === 'pfx') return contents.pfx || ''
  if (file.key === 'merged') return (contents.fullchain || '') + (contents.privkey || '')
  return contents[file.key] || ''
}

export function toBase64(str) {
  const bytes = new TextEncoder().encode(str)
  let bin = ''
  for (const b of bytes) bin += String.fromCharCode(b)
  return btoa(bin)
}

function configExample(server, files, os) {
  const c = os === 'cmd' ? 'REM' : '#'
  const n = (key) => files.find((f) => f.key === key)?.name || ''
  switch (server) {
    case 'nginx':
      return [
        `${c} Nginx 配置参考：`,
        `${c}   ssl_certificate     /etc/ssl/${n('fullchain')};`,
        `${c}   ssl_certificate_key /etc/ssl/${n('privkey')};`
      ]
    case 'apache':
      return [
        `${c} Apache 配置参考：`,
        `${c}   SSLCertificateFile /etc/ssl/${n('cert')}`,
        `${c}   SSLCertificateChainFile /etc/ssl/${n('chain')}`,
        `${c}   SSLCertificateKeyFile /etc/ssl/${n('privkey')}`
      ]
    case 'haproxy':
      return [`${c} HAProxy 配置参考：`, `${c}   bind :443 ssl crt /etc/ssl/${n('merged')}`]
    case 'caddy':
      return [`${c} Caddyfile 配置参考：`, `${c}   tls /etc/ssl/${n('fullchain')} /etc/ssl/${n('privkey')}`]
    case 'iis':
      return [`${c} IIS 导入：certlm.msc → 个人 → 所有任务 → 导入，选择 ${n('pfx')}`]
    case 'tomcat':
      return [
        `${c} Tomcat 配置参考：`,
        `${c}   <Connector port="443" SSLEnabled="true" keystoreFile="${n('pfx')}" keystoreType="PKCS12" keystorePass="你的密码" />`
      ]
    default:
      return []
  }
}

// 生成终端写文件命令（当前目录），mode=text 文本直写 / base64 解码写入
export function buildCommands({ os, mode, files, contents, server }) {
  if (!contents) return ''
  const items = files.map((f) => {
    if (f.key === 'pfx') return { ...f, b64: contents.pfx || '' }
    const text = fileText(f, contents)
    if (!text) return { ...f, text: '', b64: '' }
    const normalized = text.endsWith('\n') ? text : text + '\n'
    return { ...f, text: normalized, b64: toBase64(normalized) }
  })
  if (items.some((f) => !f.b64)) return ''

  const lines = []
  for (const f of items) {
    const useText = mode === 'text' && f.key !== 'pfx'
    if (os === 'linux') {
      if (useText) {
        lines.push(`cat > '${f.name}' <<'DDNS_GO_EOF'`)
        lines.push(f.text.replace(/\n$/, ''))
        lines.push('DDNS_GO_EOF')
      } else {
        lines.push(`echo '${f.b64}' | base64 -d > '${f.name}'`)
      }
      if (f.sensitive) lines.push(`chmod 600 '${f.name}'`)
    } else if (os === 'powershell') {
      if (useText) {
        lines.push("@'")
        lines.push(f.text.replace(/\n$/, ''))
        lines.push(`'@ | Set-Content -Path '${f.name}' -Encoding ascii`)
      } else {
        lines.push(`[IO.File]::WriteAllBytes((Join-Path $PWD '${f.name}'), [Convert]::FromBase64String('${f.b64}'))`)
      }
    } else {
      // Windows CMD
      if (useText) {
        f.text.replace(/\n$/, '').split('\n').forEach((row, i) => {
          lines.push(`${i === 0 ? '>' : '>>'} "${f.name}" echo ${row}`)
        })
      } else {
        const tmp = `%TEMP%\\${f.name}.b64`
        const chunks = f.b64.match(/.{1,64}/g) || []
        chunks.forEach((chunk, i) => {
          lines.push(`${i === 0 ? '>' : '>>'} "${tmp}" echo ${chunk}`)
        })
        lines.push(`certutil -decode "${tmp}" "${f.name}"`)
        lines.push(`del "${tmp}"`)
      }
    }
    lines.push('')
  }
  lines.push(...configExample(server, files, os))
  return lines.join('\n')
}
