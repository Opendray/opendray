# 快速上手

零到首次会话的完整 walkthrough。Postgres 已经在 host 上的话 ~15
分钟,需要顺便装 Postgres 的话 25 分钟。

这份指南是**端到端**的 —— 把 opendray 外围的事情(装 opendray
wrap 的 CLI、bootstrap Postgres)跟 README 里的部署路径串到一起。
如果你之前用过 opendray、只是想重新部署,直接看 README 生产部署
里压缩版的步骤就够。

> **还没确定 "opendray 适合我吗"?**
> 先看 README 顶部的 [opendray 是什么?](../README.zh.md#opendray-是什么)。
> 如果用例不 match,那一段会让你省 15 分钟。

---

## 第 0 步 —— 你需要什么

| 工具 | 为什么 | 备注 |
|---|---|---|
| 至少一个:Claude Code / Codex CLI / Gemini CLI | opendray 是 **wrapper**,不是模型 —— 它在 host 上 spawn 你装好的 CLI | 第 1 步 |
| PostgreSQL 15 / 16 / 17 + **pgvector** 扩展 | 状态、会话、记忆向量 | 第 2 步 |
| `go` 1.25+ 和 `pnpm` 10+ —— *只在* 从源码 build 时需要 | 用 release binary 的话跳过 | [Releases 页](https://github.com/Opendray/opendray_v2/releases) |
| 一个可达的端口(默认 `:8770`)给 Web 后台 | UI + API + WebSocket | 没反向代理就绑 `127.0.0.1` |

---

## 第 1 步 —— 至少装一个 AI CLI

opendray 用你本地的账户 spawn 这些 CLI。你像平时终端用一样装它们,
opendray 在 `$PATH` 上找。

### Claude Code(推荐起点)

```sh
npm install -g @anthropic-ai/claude-code
claude login        # 浏览器 OAuth
```

登录后,凭据落在 `~/.claude/credentials.json`,opendray 选 **claude**
provider 时会自动读到。

### Codex CLI(OpenAI)

```sh
# 按 https://github.com/openai/codex 的官方安装步骤
# 不同版本 npm 包/pip target 可能不同,装完 `codex` 应该在 $PATH 上
codex --version     # 验证一下
```

### Gemini CLI(Google)

```sh
npm install -g @google/gemini-cli
gemini auth login
```

### 验证至少一个能找到

```sh
which claude codex gemini      # 至少一行能 resolve
```

> 只装 **一个** CLI 就能跑 opendray,其他以后再加。Provider 列表是
> 动态的 —— spawn 时 opendray 现场探测 binary,装漏的会在 Sessions
> 错误面板里显示 "command not found"。

---

## 第 2 步 —— 装 Postgres + pgvector

opendray 要求 PostgreSQL **15、16 或 17**,并装
[`pgvector`](https://github.com/pgvector/pgvector) 扩展。按 host
选安装方式。

### macOS(Homebrew)

```sh
brew install postgresql@17 pgvector
brew services start postgresql@17
```

### Ubuntu / Debian

```sh
sudo apt install postgresql-17 postgresql-17-pgvector
sudo systemctl enable --now postgresql
```

### 其他 Linux

用你发行版的 PG 包,然后 pgvector 走包管理器装,或者从
[源码 build](https://github.com/pgvector/pgvector#installation)。

### Bootstrap opendray 数据库(一次性)

用 superuser 进 psql:

```sql
-- 本地(Homebrew 默认): `psql postgres`
-- 远程: `psql -h <host> -U postgres -d postgres`

CREATE DATABASE opendray;
CREATE USER opendray_user WITH ENCRYPTED PASSWORD '<选一个强密码>';
GRANT ALL PRIVILEGES ON DATABASE opendray TO opendray_user;

\c opendray
CREATE EXTENSION IF NOT EXISTS vector;
GRANT ALL ON SCHEMA public TO opendray_user;
```

> `CREATE EXTENSION vector` 需要 **superuser**。装完之后,
> `opendray_user` 只需要上面 grant 的 CRUD 权限就够 —— opendray
> 运行时不会再以 superuser 连接。

从将来要跑 opendray 的 host 上测一下凭据:

```sh
PGPASSWORD='<密码>' psql -h <pg-host> -U opendray_user -d opendray -c "SELECT 'ok' AS check;"
```

应该看到 `check: ok`、没有错误。

---

## 第 3 步 —— 选部署路径、装 opendray

**先问自己**:你来这里是为了 session spawn 功能吗(在 web Sessions
页 spawn Claude / Codex / Gemini)?

### 如果是 —— 需要 "完整" 路径

| 你的 host | 路径 | README 章节 |
|---|---|---|
| macOS 24/7 家用 server | macOS LaunchDaemon | [方案 D](../README.zh.md#方案-d--macos-launchdmac-mini--studio-当家用-server) |
| Linux 机器 / VPS / LXC | systemd | [方案 B](../README.zh.md#方案-b--systemd裸机--vm--lxc) |
| 前台测试 | 源码 `go run` | [快速开始](../README.zh.md#快速开始5-分钟开发版) |
| 自己的进程管理器(s6 / runit / launchd Agent) | 直接二进制 | [方案 C](../README.zh.md#方案-c--直接跑二进制--你自己的进程管理器) |

> 跳过 Docker。镜像是 distroless(没 Node、没 AI CLI、没 `pg_dump`),
> Sessions tab 每次 spawn 都会报错。架构原因见 §A 的 callout。

### 如果不是 —— 只需要 channels / integrations / notes / API

| 你的 host | 路径 | README 章节 |
|---|---|---|
|

仍然可以收 Telegram / Slack 等消息、写 notes、调集成 API、看 Web 后台。
只是不能从这个部署里 spawn 本地 AI CLI 会话。

每条路径都汇聚到这里:

```sh
# 准备 gateway 读的配置
cp config.example.toml config.toml
$EDITOR config.toml            # 填 [database].url、[admin].password

# 一次性:创建 schema(再次跑是 no-op)
opendray migrate -config config.toml

# 跑 gateway
opendray serve -config config.toml
```

`config.toml` 最少必填的两个字段:

```toml
[database]
url = "postgres://opendray_user:<密码>@<host>:5432/opendray?sslmode=disable"

[admin]
password = "<初始 bootstrap 密码>"
```

其他都有合理默认 —— `config.example.toml` 行内注释覆盖了所有 surface。

---

## 第 4 步 —— 首次登录 + 立刻改 admin 密码

打开 `http://localhost:8770/admin/`(或者按 `config.toml` 里
`listen` 绑定的 host:port)。

1. 用 `admin` + 你 `[admin].password` 里写的密码登录。
2. **立刻** 去 Settings → Admin → Change password。

为什么要立刻:第一次改密码后,opendray 会写一个 bcrypt-hash 的
keyfile 到 `$HOME/.opendray/secrets/admin.key`,从此 `config.toml`
里的 `[admin].password` 明文就失效了(keyfile 优先)。改密码前,
保护层只有 `config.toml` 的文件权限。

完整凭据优先级链见
[operator-guide §admin](operator-guide.md#admin)。

---

## 第 5 步 —— 配 Provider

Providers → 点你第 1 步装的那个 CLI → 填:

- **Command path** —— CLI 二进制的绝对路径
  (`which claude` 可以找到;Apple Silicon Homebrew 装的一般在
  `/opt/homebrew/bin/claude`)。
- **Accounts dir**(只 Claude,可选) —— 多套命名的 Claude 凭据
  目录,做"每个 session 不同账号"切换时用。留空就用默认
  `~/.claude`。

Save。opendray 会跑一次 `<cli> --version` 探测;binary 能找到就
显示绿点。

---

## 第 6 步 —— spawn 第一个 session

Sessions → New session → 选 provider → 选工作目录(随便一个项目) → Spawn。

浏览器内开一个终端 pane。像在真实终端一样输入 prompt —— 每个字节都
通过 opendray 的 PTY wrapper 中转。关掉浏览器 tab,session 在 host
上继续跑;回头来,scrollback 完整。

---

## 第 7 步(可选) —— 加 Telegram 频道

这就是 opendray 跟 `tmux + ssh` 不同的地方。配好频道后,session
变 idle(CLI 在等你输入)时 opendray 推 Telegram 通知,你在 Telegram
回复,文本就 flow back 到 session 的 stdin。

### Telegram 一次性设置

1. Telegram → 搜 **@BotFather** → 开个 chat。
2. `/newbot` → BotFather 引导你选 name + username → 给你一个 token,
   长得像 `123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11`。
3. 找你要让 opendray 发到哪个 chat 的 ID:
   - 给 bot DM 一条消息(任意文字)。
   - 浏览器开 `https://api.telegram.org/bot<token>/getUpdates`,
     JSON response 里有 `chat.id`。

### 在 opendray 里

Channels → New channel → kind **Telegram**:

- **Bot token**:BotFather 给的
- **Default chat ID**:`getUpdates` 拿到的 `chat.id`
- **Notify on**:勾 `session.idle`(或者三个 topic 都勾)

Save → 点 channel 卡上的 **Test** 按钮。几秒内 Telegram 会收到测试
消息。

让一个 session idle 30 秒(默认 idle 阈值,可通过
`[session].idle_threshold` 改)。Telegram 会推送 CLI 最近的输出。你
回复,文本就 flow back 到 session 的 stdin。

---

## 下一步

- **更多频道**:Slack / Discord / 飞书 / 钉钉 / 企业微信 —— 每个的
  设置都在应用内 Tutorial(`/admin/tutorial/`)里。
- **API 集成**:[docs/integration-guide.md](integration-guide.md)
  —— scope 化的 API key、反向代理挂载、events WebSocket。
- **记忆子系统**:用 `[memory.backend] = "local"` 启用本地优先
  嵌入向量,或者接 Ollama / LM Studio —— 见应用内 Tutorial →
  Memory 章节。
- **加密备份**:配 `[backup]` 把 DB 备份推到 S3 / R2 / B2 / SFTP /
  rclone —— 见 [operator-guide §backup](operator-guide.md#backup)。

## 排错

| 症状 | 原因 | 修复 |
|---|---|---|
| migrate 报 `relation "providers" does not exist` | v2.0.0 之前的二进制(issue #162) | 拉最新二进制 —— v2.0.0 已修 |
| migrate 报 `type "vector" does not exist` | opendray 数据库里 pgvector 扩展没启用 | 用 superuser 在 `opendray` 库里跑 `CREATE EXTENSION vector;` |
| `Spawn session failed: executable file not found in $PATH` | wrap 的 CLI 没装在 opendray host 上,或者 Provider 配置里 Command Path 不对 | 回第 1 步;`which claude`(或对应 CLI)验证 |
| Telegram bot 不回复 | Bot 默认 privacy mode(bot 只看到 commands) | BotFather → `/setprivacy` → Disable |
| 反向代理后面浏览器报 `Bad gateway` | 代理没转发 WebSocket upgrade header | 见 [operator-guide §Topology](operator-guide.md#topology) 里的 nginx / Caddy 片段 |
| Sessions 页是空的但 Channels 工作 | binary 能 spawn 但没配 Provider | 第 5 步 |

---

## 另见

- [README.zh.md](../README.zh.md) —— 安装表、部署路径、项目状态
- [README.md](../README.md) —— English version
- [docs/quickstart.md](quickstart.md) —— 5 分钟开发环境(更聚焦)
- [docs/operator-guide.md](operator-guide.md) —— 运维参考:topology、认证、备份、日志
- [docs/integration-guide.md](integration-guide.md) —— 第三方 API surface
- [VERSIONING.md](../VERSIONING.md) —— major-as-generation 版本策略
- [CHANGELOG.md](../CHANGELOG.md) —— 发布历史
