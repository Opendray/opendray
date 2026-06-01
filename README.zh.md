<p align="center">
  <a href="https://opendray.dev"><img src="docs/assets/logo.png" alt="opendray" width="180"></a>
</p>

<h1 align="center">opendray</h1>

<p align="center">
  <strong>自托管网关，统一接入 Claude Code · Codex · Gemini · shell —— 跨 CLI 共享的本地优先记忆层。</strong>
  <br/>
  <sub>在自己的服务器上运行会话。Web、移动端、聊天工具任意驾驭。开放 REST + WebSocket API 供集成。</sub>
</p>

<p align="center">
  <strong><a href="https://opendray.dev">🌐 opendray.dev</a></strong>
</p>

<p align="center">
  <a href="https://opendray.dev"><img alt="Website" src="https://img.shields.io/badge/website-opendray.dev-F43F5E"></a>
  <a href="https://github.com/Opendray/opendray/releases/latest"><img alt="Latest release" src="https://img.shields.io/github/v/release/Opendray/opendray?label=release&color=4f46e5"></a>
  <a href="LICENSE"><img alt="License Apache 2.0" src="https://img.shields.io/github/license/Opendray/opendray?color=blue"></a>
  <a href="https://github.com/Opendray/opendray/actions/workflows/ci.yml"><img alt="CI" src="https://img.shields.io/github/actions/workflow/status/Opendray/opendray/ci.yml?branch=main&label=CI"></a>
  <a href="https://github.com/Opendray/opendray/discussions"><img alt="Discussions" src="https://img.shields.io/github/discussions/Opendray/opendray?color=ec4899"></a>
  <br/>
  <img alt="Go" src="https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go&logoColor=white">
  <img alt="React" src="https://img.shields.io/badge/React-19-61DAFB?logo=react&logoColor=black">
  <img alt="Flutter" src="https://img.shields.io/badge/Flutter-mobile-02569B?logo=flutter&logoColor=white">
  <img alt="Postgres" src="https://img.shields.io/badge/PostgreSQL-15%2F16%2F17-336791?logo=postgresql&logoColor=white">
</p>

<p align="center">
  🌐 <a href="README.md">English</a> · <strong>简体中文</strong> · <a href="README.fa.md">فارسی</a> · <a href="README.es.md">Español</a> · <a href="README.pt-BR.md">Português</a> · <a href="README.ja.md">日本語</a> · <a href="README.ko.md">한국어</a> · <a href="README.fr.md">Français</a> · <a href="README.de.md">Deutsch</a> · <a href="README.ru.md">Русский</a>
</p>

---

## opendray 是什么?

**opendray** 把你已经在用的 AI 编程 CLI(Claude Code、Codex、Gemini,加上任意 shell)包起来,变成一个你能随处操控的东西。在家用服务器 / NAS / VPS 上跑会话,空闲时 Telegram 给你推送,在手机上回一句话就能把下一个 prompt 送回去 —— 整个网关你自己托管、端到端掌控。

- 🛰 **一个后端,三个 surface** —— 单一 Go 二进制同时提供 React Web 后台和 Flutter 移动端,每个操作也通过 REST + WebSocket API 暴露给第三方集成。
- 💬 **六大双向频道,不锁定平台** —— Telegram、Slack、Discord、飞书、钉钉、企业微信,外加 Bridge 适配器接入任意自定义协议。任意频道的回复都路由回正确的会话。
- 🧠 **本地优先的记忆系统** —— ONNX / Ollama / LM Studio 嵌入向量,三层作用域(用户 · 项目 · 会话)智能检索 + 智能排序 + 跨层冲突检测。向量数据不出你的内网。
- 🔌 **集成级 API** —— scope 化的 API key、每次调用审计日志、反向代理挂载。可以把 opendray 作为你产品后端的网关,也可以纯粹当个人指挥中心。
- 🔑 **多 Claude 账户调度** —— 把多个 `claude login` 账户丢进网关,面板通过文件系统 watcher 自动发现;新会话在已启用的账户间均衡分配,**切换正在运行的会话到另一个账户也不会丢失对话**(后台自动迁移 transcript)。每行会显示该账户的实时容量(订阅套餐、限速档位、活动会话数、最近使用时间、当前 Anthropic 邮箱),一眼就能挑对账户。
- 🔒 **自托管 + 许可证清晰** —— Apache 2.0、单一静态二进制、cosign 签名的 release 自带 SPDX SBOM。零遥测、不依赖云账号、无订阅。

## 当前状态

**v2.6.0**(最新)—— v2 代持续迭代。
参见 [`VERSIONING.md`](VERSIONING.md) 了解 major-as-generation 版本策略
(major = 产品代号,而不是严格的 SemVer "破坏性变更" 标记),
[`CHANGELOG.md`](CHANGELOG.md) 有完整 release 历史。

这一代产品包含:

- **一行命令安装/卸载 wizard**(Linux + macOS;Windows 经 WSL2)——
  引导操作者完成 Postgres bootstrap、AI CLI 安装、admin 凭据、监听地址、
  binary 安装、schema migration、service 注册。
- **可自管理的 binary** —— `opendray update / start / stop /
  restart / status / providers list / providers update`,日常运维
  不用碰 `systemctl` / `launchctl`。
- **goreleaser release 流水线** —— 交叉编译(linux/darwin ×
  amd64/arm64)、cosign 无密钥签名(Sigstore)、SPDX SBOM、原子校验
  自更新。

## 安装

### 一行命令安装

**Linux / macOS / WSL2**

```sh
curl -fsSL https://raw.githubusercontent.com/Opendray/opendray/main/scripts/install.sh | bash
```

**Windows** —— 先设置 WSL2,然后在 WSL2 里跑 Linux 安装器。[详情 →](scripts/README.md#windows)

```powershell
irm https://raw.githubusercontent.com/Opendray/opendray/main/scripts/install-windows.ps1 | iex
```

引导你完成 Postgres 设置、AI CLI 安装、admin 凭据、服务注册,5–10 分钟拉起一个运行中的网关。详见 [**`scripts/README.md`**](scripts/README.md):wizard 做什么、生成的文件布局、参数、排错。

> **想自己一步步来?** 看 [**docs/getting-started.zh.md**](docs/getting-started.zh.md) —— 15 分钟端到端 walkthrough,跟 wizard 做的是同样的事,但每一步你都自己确认。

### 卸载(Linux / macOS)

**默认模式** —— 停掉网关、删 binary,但**保留** `config.toml`、数据目录(bcrypt keyfile、sessions、notes、vault)、日志、PostgreSQL 数据库。重装时直接接上,数据不丢:

```sh
curl -fsSL https://raw.githubusercontent.com/Opendray/opendray/main/scripts/uninstall.sh | bash
```

**完整 purge** —— 还会 drop 数据库 + role、删 config / 数据 / 日志、移除服务用户。最后有 verification 步骤,有残留会大声报错:

```sh
curl -fsSL https://raw.githubusercontent.com/Opendray/opendray/main/scripts/uninstall.sh | OPENDRAY_PURGE=1 bash
```

### 日常运维命令

装完之后,`opendray` 二进制自己管自己的生命周期 —— 不用记 `systemctl` / `launchctl`:

```sh
sudo opendray update --restart   # 拉最新 release、SHA-256 校验、原子替换 + 重启
```

```sh
sudo opendray providers update   # 升级 AI CLIs(claude / codex / gemini)到 npm 最新版
```

```sh
opendray providers list          # 看装了哪些 AI CLI、各自版本
```

```sh
sudo opendray start              # start | stop | restart | status —— 封装 systemd / launchd
```

完整子命令列表:`opendray --help`。

### 选部署路径

所有支持的方案都包含 session spawn、AI CLI、加密备份、完整集成 API。opendray 是 host-resident 网关 —— PTY spawn AI CLI、跟它们共享进程状态(`~/.claude`、ssh-agent、项目文件),这跟生产级 Docker 的容器隔离模型不兼容,因此 v2.x 不支持 Docker 部署。

| 方式 | 适合 | 跳转到 |
|---|---|---|
| 📦 **预构建二进制** | "拿来就跑" — Linux / macOS,搭配任意进程管理器 | [Releases 页](https://github.com/Opendray/opendray/releases) → 见下方 [生产部署](#生产部署) |
| 🐧 **systemd unit** | 裸机 / VM / Linux LXC | [生产部署 §A](#方案-a--systemd裸机--vm--lxc) |
| 🍎 **macOS LaunchDaemon** | Mac mini / Mac Studio 当家用 server | [生产部署 §C](#方案-c--macos-launchdmac-mini--studio-当家用-server) |
| 🛠 **从源码构建** | 开发 / 贡献代码 / 定制构建 | [快速开始](#快速开始5-分钟开发版) |

## 快速开始(5 分钟开发版)

完整 walkthrough(含前置依赖、排错)见
[`docs/quickstart.md`](docs/quickstart.md)。压缩版:

```bash
# 1. 准备一个 Postgres 15+(127.0.0.1:5432),并启用 pgvector。
#    apt install postgresql-16 postgresql-16-pgvector  /  brew install postgresql@16 pgvector
#    用别的 DSN 也行 — 改 [database].url 就行。

# 2. 本地配置 — 已经 gitignored。
cp config.example.toml config.toml
$EDITOR config.toml          # 设置 [database].url 和 [admin].password

# 3. 构建 web bundle 到 embed 目录。
cd app/web && pnpm install && pnpm build && cd ../..

# 4. 应用 schema。
go run ./cmd/opendray migrate -config config.toml

# 5. 运行。
go run ./cmd/opendray serve -config config.toml
# → REST + WS:  http://127.0.0.1:8770/api/v1/...
# → Web admin:  http://127.0.0.1:8770/admin/
```

上面是前台运行 —— Ctrl-C 即可结束。如果要做成长期运行的守护进程,
看下面 **生产部署**。

## 生产部署

四种受支持的部署路径,按你的环境挑一种。每种都提供:
崩溃后自动重启、状态持久化、secrets 跟 config 分离。

### 方案 A — systemd(裸机 / VM / LXC)

Linux 推荐部署路径。
[`deploy/systemd/opendray.service`](deploy/systemd/opendray.service)
是一个加固过的 unit:沙箱(`ProtectSystem=strict`、`NoNewPrivileges`、
`MemoryDenyWriteExecute`、capability 收紧)、先 `migrate` 后 `serve`
的启动顺序、20 秒优雅退出窗口。

**先拿一个二进制。** 要么从
[Releases 页](https://github.com/Opendray/opendray/releases)
下载预构建归档(`opendray_*_linux_<arch>.tar.gz` — 解压就是
单一 `opendray` 二进制),要么按上面 [快速开始](#快速开始5-分钟开发版)
从源码 build(`go build ./cmd/opendray`)。

```bash
# 1. 安装刚拿到的(或刚 build 的)二进制。
sudo install -m 0755 /path/to/opendray /usr/local/bin/opendray

# 2. 创建服务用户和状态目录。
sudo useradd -r -s /usr/sbin/nologin -d /var/lib/opendray opendray
sudo install -d -o opendray -g opendray -m 0700 /var/lib/opendray

# 3. 放 config 和 secrets(root 所有,mode 0640)。
sudo install -D -m 0640 config.example.toml /etc/opendray/config.toml
sudo $EDITOR /etc/opendray/config.toml             # 设置 [database].url 等
sudo install -D -m 0640 -o root -g opendray /dev/null /etc/opendray/env.d/secrets
sudo $EDITOR /etc/opendray/env.d/secrets           # OPENDRAY_ADMIN_PASSWORD=…

# 4. 安装并启用 unit。
sudo cp deploy/systemd/opendray.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now opendray

# 5. 验证。
sudo systemctl status opendray
sudo journalctl -u opendray -f --no-pager
```

Unit 在 `ExecStartPre` 阶段跑 `opendray migrate`,首次启动会先把
所有 migration 应用了再启动 `serve`。Restart 策略是 `on-failure`,
5 秒退避,每分钟最多重启 5 次。

### 方案 B — 直接跑二进制 + 你自己的进程管理器

适合没装 systemd 的 LXC、FreeBSD `rc.d`、OpenRC,或其他任何环境。
一次构建,任意进程管理器拉起来:

```bash
# 本地交叉编译一个 release archive:
goreleaser release --clean --snapshot
ls dist/                  # opendray_*_linux_amd64.tar.gz 等

# 或者从已发布的 release 拿 artifact:
# https://github.com/Opendray/opendray/releases
```

让你的进程管理器(s6、runit、supervisord、runwhen)指向:

```
/usr/local/bin/opendray serve -config /etc/opendray/config.toml
```

预先动作:首次 `serve` 之前跑一次 `opendray migrate -config /etc/opendray/config.toml`,
或者把它做成进程管理器的 pre-start hook。

### 方案 C — macOS launchd(Mac mini / Studio 当家用 server)

适合 Apple Silicon 的 Mac mini / Mac Studio 24/7 跑。
[`deploy/launchd/com.opendray.opendray.plist`](deploy/launchd/com.opendray.opendray.plist)
是一个 LaunchDaemon:开机即启动(不需要用户登录),崩溃后 5 秒
节流重启,日志写到 `/usr/local/var/log/opendray/`。

```bash
# 1. 安装 darwin 二进制 + 配置 + 状态目录。
sudo install -m 0755 ./opendray /usr/local/bin/opendray
sudo install -d -m 0755 \
  /usr/local/etc/opendray \
  /usr/local/var/lib/opendray \
  /usr/local/var/log/opendray
sudo install -m 0640 config.example.toml /usr/local/etc/opendray/config.toml
sudo $EDITOR /usr/local/etc/opendray/config.toml    # 设置 [database].url 等

# 2. 跑一次 migrate。
sudo /usr/local/bin/opendray migrate \
  -config /usr/local/etc/opendray/config.toml

# 3. 安装并加载 LaunchDaemon。
sudo cp deploy/launchd/com.opendray.opendray.plist /Library/LaunchDaemons/
sudo chown root:wheel /Library/LaunchDaemons/com.opendray.opendray.plist
sudo chmod 0644 /Library/LaunchDaemons/com.opendray.opendray.plist
sudo launchctl bootstrap system /Library/LaunchDaemons/com.opendray.opendray.plist

# 4. 验证。
sudo launchctl print system/com.opendray.opendray
tail -f /usr/local/var/log/opendray/opendray.log
```

重启:`sudo launchctl kickstart -k system/com.opendray.opendray`;
完全卸载:`sudo launchctl bootout system/com.opendray.opendray`。

macOS 上的 Postgres — 用 Homebrew 装(`brew install postgresql@17 && brew services start postgresql@17`),把 `[database].url` 指向
`postgres://$USER@127.0.0.1:5432/opendray`。还要 `brew install pgvector`
+ 在 opendray 数据库里 `CREATE EXTENSION vector`。

---

Proxmox LXC 特定的说明(非特权容器里的 PTY、网络、cgroup 调整)
见 [`deploy/lxc/proxmox-pty-notes.md`](deploy/lxc/proxmox-pty-notes.md)。

反向代理 / TLS 终止(nginx、Caddy、Traefik、Cloudflare Tunnel)
见 [`docs/operator-guide.md`](docs/operator-guide.md) §Topology。

### 可选:启用加密 DB 备份 + 数据导出

```bash
# 主密码(只能用 env 传 — 永远不要写进 config.toml)。
export OPENDRAY_BACKUP_KEY="$(openssl rand -base64 32)"
export OPENDRAY_BACKUP_ENABLED=1

# pg_dump / pg_restore 必须跟 Postgres server 主版本一致。
# Apple Silicon 上指向 PG17 的示例:
export OPENDRAY_BACKUP_PG_DUMP_PATH=/opt/homebrew/opt/postgresql@17/bin/pg_dump
export OPENDRAY_BACKUP_PG_RESTORE_PATH=/opt/homebrew/opt/postgresql@17/bin/pg_restore
```

重启 opendray,侧栏会出现 Backups 页(`/backups`)用于加密的
PostgreSQL 备份 + 恢复,以及 `/export` 用于 zip 包数据导出 + 导入。
[`docs/operator-guide.md`](docs/operator-guide.md) §Backup 有完整生命周期说明。

一个 Go 二进制装着整个 web bundle —— 运行时不需要 Node,不需要单独的
静态文件服务器,不需要 Caddy/nginx。Cloudflare Tunnel 在 `:8770`
前面负责 TLS 终止。

## 项目结构

```
cmd/opendray/        二进制入口(按设计 §14 控制在 ≤100 LOC)
internal/
├── app/             composition root(组装所有子系统)
├── audit/           订阅事件总线,持久化到 audit_log
├── auth/            admin bearer token(M2.5)
├── backup/          加密 DB 备份 + admin 导出/导入├── catalog/         CLI provider manifest + 每个 id 的用户配置(M2)
├── channel/         channel hub + telegram 实现(M4)
├── config/          TOML 加载器,支持 OPENDRAY_* env 覆盖
├── eventbus/        进程内 pub/sub
├── gateway/         chi HTTP 路由 + 中间件 + slog
├── integration/     外部应用注册表 + 反向代理 + events WS(M3)
├── memory/          跨 CLI 持久化记忆├── session/         PTY 生命周期 + ring buffer + WS 流(M1)
├── store/           pgx pool + 自写迁移 runner(M0)
├── version/         build 时的身份标识
└── web/             web bundle 的 go:embed(W5)

app/web/             React 19 + TypeScript + Vite SPA(Phase 2 W0-W5)
app/mobile/          Flutter app(iOS + Android),跟 Web 同等功能集
docs/
├── design.md        SSOT north-star
└── adr/             架构决策,按日期排序
```

## Web 前端

`app/web/` 把单页 SPA 构建到 `internal/web/dist/`,Go 二进制 embed
后在 `/admin/*` 路径提供服务。Vite dev server 在 `:5173`,把 `/api`
代理到 `:8770` 用于 HMR 驱动的开发。

```bash
# dev(React 端热重载,另起 Go server 提供 API)
cd app/web && pnpm dev               # http://localhost:5173
go run ./cmd/opendray serve -config ../../config.toml   # 另一个终端

# prod(一个二进制提供一切)
cd app/web && pnpm build              # 写到 ../../internal/web/dist
cd ../..
go build ./cmd/opendray               # 把 dist 打进二进制
./opendray serve -config config.toml
```

前端技术栈细节(React + Vite + Tailwind v4 + shadcn/ui + TanStack
Router/Query + Zustand + xterm.js)和每个 W 里程碑笔记见
[`app/web/README.md`](app/web/README.md)。

## 文档

- [`docs/getting-started.zh.md`](docs/getting-started.zh.md) — **新手从这开始**:零到首次会话 15 分钟,含装 wrap 的 CLI + bootstrap Postgres + 收第一条 Telegram 通知
- [`docs/quickstart.md`](docs/quickstart.md) — 5 分钟开发环境(假设你已经懂各组件)
- [`docs/operator-guide.md`](docs/operator-guide.md) — 生产化部署 + 运维参考
- [`docs/integration-guide.md`](docs/integration-guide.md) — 用任意语言写外部集成
- [`VERSIONING.md`](VERSIONING.md) — 版本策略(major-as-generation)
- [`CHANGELOG.md`](CHANGELOG.md) — 发布历史

## 测试

```bash
go test -race ./...        # 后端
cd app/web && pnpm build   # web(TS strict + vite production build)
```

端到端 smoke flow 在每个 milestone 的 commit message 里追踪。
Playwright e2e harness 是计划中的后续工作。

## 跟 v1 的关系

v1(`Opendray/opendray`)是上一代代码库,已归档。v2 是当前活跃的
代号 —— 功能完整,是唯一接受开发的分支。v1 的 16 个 builtin 里有
4 个迁到了 v2 后端,其余的拆成了客户端功能、channel 适配器或集成
API 消费方。

## 许可证

Apache 2.0 — 见 [`LICENSE`](LICENSE)。(v1 是 MIT;v2 独立授权。)
