# opendray v2

> 🌐 **语言**: [English](README.md) · 简体中文

> AI 编程 CLI 的多路控制 + 集成网关。
> 通过 Web + 移动端远程控制 Claude Code、Codex、Gemini、shell 会话。
> 一份 Claude Pro 订阅就能服务整个个人应用生态,不必按 token 给 API 付费。

## 当前状态

**v2.0.0** — opendray v2 这一代产品的首次发布(2026-05-17)。
参见 [`VERSIONING.md`](VERSIONING.md) 了解 major-as-generation 版本策略
(major = 产品代号,而不是严格的 SemVer "破坏性变更" 标记)。

这一代产品包含:

- **后端(Go)** — sessions、channels、providers、memory、backup、
  集成 API。单一静态二进制,React SPA 通过 `go:embed` 嵌入。
- **Web admin**(React 19 + Vite + Tailwind v4 + shadcn/ui + TanStack
  Router/Query + Zustand + xterm.js)
- **移动端**(Flutter,iOS + Android,在 `app/mobile/`)— 在会话控制、
  频道管理、记忆、备份、笔记和集成 API 上跟 Web 完全对等
- **六大双向频道** — Telegram · Slack · Discord · 飞书(Feishu)·
  钉钉(DingTalk)· 企业微信(WeCom)— 外加 **Bridge** 用 WebSocket
  接入自定义平台
- **本地优先的记忆系统** — ONNX / Ollama / LM Studio 嵌入向量;
  跨层检索(用户 · 项目 · 会话)+ 智能排序 + 冲突检测;
  数据不出你的内网
- **自动化 release 流水线** — goreleaser 交叉编译(linux/darwin ×
  amd64/arm64)、cosign 无密钥签名(Sigstore)、SPDX SBOM、
  GHCR 多架构镜像

查看 [`CHANGELOG.md`](CHANGELOG.md) 了解 v2.0.0 详情和后续 Unreleased
段中即将落地的内容。

## 快速开始

完整 walkthrough(含前置依赖、排错、docker-compose 开发 DB)见
[`docs/quickstart.md`](docs/quickstart.md)。压缩版:

```bash
# 1. 启动本地开发用 Postgres(或把 [database].url 指向你自己的 DB)。
docker compose -f docker-compose.test.yml up -d   # 127.0.0.1:5432

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
ADR 0012 和应用内的 **Tutorial → Backups** 章节有完整生命周期说明。

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
├── backup/          加密 DB 备份 + admin 导出/导入(ADR 0012)
├── catalog/         CLI provider manifest + 每个 id 的用户配置(M2)
├── channel/         channel hub + telegram 实现(M4)
├── config/          TOML 加载器,支持 OPENDRAY_* env 覆盖
├── eventbus/        进程内 pub/sub
├── gateway/         chi HTTP 路由 + 中间件 + slog
├── integration/     外部应用注册表 + 反向代理 + events WS(M3)
├── memory/          跨 CLI 持久化记忆(ADR 0014)
├── session/         PTY 生命周期 + ring buffer + WS 流(M1)
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

- [`docs/quickstart.md`](docs/quickstart.md) — 完整 quickstart,含前置依赖、排错、docker-compose 开发 DB
- [`docs/design.md`](docs/design.md) — 任务、架构、子系统、API、数据模型、路线图
- [`docs/adr/`](docs/adr/) — 每个生效中的架构决策,按日期排序
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
代号 —— 功能完整,是唯一接受开发的分支。ADR 0001 记录了 greenfield
决策;ADR 0004 解释了哪些 v1 builtin 迁移过来(16 个里只迁了 4 个),
哪些变成了 v2 里的客户端 / channel / 集成工作。

## 许可证

Apache 2.0 — 见 [`LICENSE`](LICENSE)。(v1 是 MIT;v2 独立授权。)
