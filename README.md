# ddns-go-acme

自动获得你的公网 IPv4 / IPv6 地址并解析到对应域名，同时内置 **ACME SSL 证书申请、自动续期与多格式部署**，一个二进制搞定 DDNS 与证书管理。

> 本项目基于 [jeessy2/ddns-go](https://github.com/jeessy2/ddns-go)（v6）二次开发，感谢原作者及所有贡献者。在原项目 DDNS 能力之上，新增 ACME 证书功能，并将 Web 端重写为 Vue3 + Element Plus 单页应用。

## 与原项目的区别

| 能力 | 说明 |
| --- | --- |
| ACME 证书申请 | 基于 [go-acme/lego](https://github.com/go-acme/lego)，DNS-01 验证，支持泛域名 |
| 自动续期 | 后台每 6 小时检查一次，默认到期前 30 天自动续期（可调） |
| DNS-01 服务商 | Cloudflare、DNSPod、腾讯云（复用 DDNS 配置中的 API 密钥） |
| 证书导出 | PEM（Nginx/Apache）、PFX/PKCS12（IIS/Windows，可设密码）、全格式 ZIP |
| 按服务器部署 | Nginx / Apache / HAProxy / Caddy / IIS / Tomcat 等类型，一键生成终端写文件命令（文本直写或 Base64），或在线查看/复制证书内容 |
| 全新 Web 界面 | Vue3 + Element Plus + Vite，构建产物通过 `go:embed` 打入单文件二进制 |

原项目的 DDNS 能力完整保留：

- 支持 `阿里云` `腾讯云` `Dnspod` `Cloudflare` `华为云` `Callback` `百度云` `Porkbun` `GoDaddy` `Namecheap` `NameSilo` `Dynadot` 等 20+ 服务商
- 支持接口 / 网卡 / 命令获取 IP，支持多服务商、多域名、多级域名、TTL、自定义参数
- 支持 Webhook 通知、系统服务方式运行、网页查看最近 50 条日志
- 默认勾选「禁止从公网访问」，首次登录 30 分钟内自设账号密码

## 快速开始

从 Releases 下载对应平台压缩包（当前提供 **Windows / Linux amd64**）：

```bash
# Windows（PowerShell）
.\ddns-go-acme-windows-amd64.exe

# Linux
chmod +x ddns-go-acme-linux-amd64
./ddns-go-acme-linux-amd64
```

浏览器访问 `http://localhost:9876`，首次启动 **30 分钟内**输入的账号密码即为初始账号。

安装为系统服务：

```bash
# Mac/Linux
sudo ./ddns-go-acme -s install
# Windows（管理员）
.\ddns-go-acme.exe -s install
```

常用参数：

| 参数 | 说明 |
| --- | --- |
| `-l` | 监听地址，默认 `:9876` |
| `-f` | DDNS 同步间隔秒数，默认 300 |
| `-c` | 自定义配置文件路径，默认 `~/.ddns_go_config.yaml` |
| `-cacheTimes` | 间隔 N 次与服务商比对，默认 5 |
| `-noweb` | 不启动 Web 服务 |
| `-skipVerify` | 跳过 TLS 证书校验 |
| `-dns` | 自定义 DNS 服务器，如 `8.8.8.8` |
| `-resetPassword` | 重置登录密码 |

## SSL 证书使用

1. 在「DNS 解析」页配置好 Cloudflare / DNSPod / 腾讯云的 API 密钥。
2. 打开「SSL 证书」→「新建证书」：
   - 填写 ACME 账号邮箱（用于注册 ACME 账号、接收 CA 紧急通知，请勿使用 `example.com` 等保留域名）；
   - 选择 DNS-01 验证配置（复用已有 DDNS 密钥，无需重复填写）；
   - 填写证书域名，每行一个，泛域名写作 `*.example.com`；
   - 默认 EC256 私钥、Let's Encrypt 正式环境，可先切换测试环境（staging）跑通流程。
3. 保存后自动异步申请，列表实时展示状态、到期时间与剩余天数。
4. 证书文件保存在配置目录下的 `.ddns_go_certs/`（`fullchain.pem`、`privkey.pem`、`cert.pem`）。
5. 「下载」菜单支持：
   - 直接下载 PEM / PFX / ZIP；
   - **终端部署命令**：选择服务器类型与终端类型（Linux bash / PowerShell / CMD），生成在当前目录写文件的命令，附对应服务器配置参考；
   - **查看证书内容**：按文件在线查看并复制 PEM 原文。

## 技术栈与构建

- 后端：Go（标准库 `net/http`，无 Web 框架）、[go-acme/lego/v4](https://github.com/go-acme/lego)、[go-pkcs12](https://software.sslmate.com/src/go-pkcs12)、YAML 单文件配置
- 前端：Vue 3、Element Plus、Vue Router、Vite，源码在 `web/ui/`
- 单文件分发：前端构建产物输出到 `web/dist`，通过 `go:embed` 嵌入二进制，CGO 关闭可静态编译

环境要求：Go ≥ 1.27、Node.js ≥ 20。

```bash
# 一键构建（自动构建前端后编译 Go）
make build

# 或手动执行
cd web/ui && npm install && npm run build && cd ../..
CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath -o ddns-go-acme .

# 前端热更新开发（另开终端运行 go run . -l :9876）
make ui-dev

# 运行测试
make test
```

## GitHub Actions 自动构建

[release.yml](.github/workflows/release.yml) 已配置：

- 推送 `v*` 标签（如 `git tag v1.0.0 && git push origin v1.0.0`）自动构建 Windows / Linux amd64 二进制，打包 zip / tar.gz 并发布 Release；
- 也可在 Actions 页面手动触发，仅构建并上传 artifacts（保留 7 天），不发布。

流程：Node 20 构建前端 → Go 交叉编译（注入 tag 版本号）→ 打包 → `softprops/action-gh-release` 发布，无需额外 secrets（仅使用内置 `GITHUB_TOKEN`）。

## 文档

架构与实现细节见 [doc/architecture.md](doc/architecture.md)，包含 DDNS 更新流程、ACME 签发/续期流程、配置结构与全部 HTTP 接口。

## 致谢与许可

- 原项目：[jeessy2/ddns-go](https://github.com/jeessy2/ddns-go)
- ACME 实现：[go-acme/lego](https://github.com/go-acme/lego)
- UI 组件：[Element Plus](https://element-plus.org/)

本项目遵循与原项目一致的开源协议，详见 [LICENSE](LICENSE)。
