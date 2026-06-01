# 安装并运行预构建二进制

适合已经有 —— 或只想要 —— `opendray` 二进制本身的场景，无需安装向导触碰你的机器。以下情况走这条路：

- **`npm install -g opendray` / `npx opendray`** —— npm 包附带官方 Go release 二进制（见 [README → npm / npx](../README.zh.md#npm--npx-node--18)）。
- **Release 下载** —— 从 [Releases 页](https://github.com/Opendray/opendray/releases) 拿 `opendray_*_<os>_<arch>.tar.gz`。
- **脚本化 / 临时环境** —— CI runner、Golden Image、配置管理（Ansible、Nix、Docker），或任何你已经有自己的 Postgres 和进程管理器的主机。

这个二进制就是*整个*网关 —— Web admin SPA 已经内嵌，所以不需要 Node runtime，不需要单独的静态文件服务器，也没有任何东西需要构建。它**不会**替你做任何设置。这就是它的取舍：你自带 PostgreSQL 数据库和保活进程的方式，换来的是不会有任何东西在背后悄悄安装、配置或注册。

> **想让一切自动搞定？** 在全新 Linux / macOS 机器上，一行安装命令会帮你 provision Postgres、安装 AI CLI、写好配置、并在 5–10 分钟内注册好服务。见 [README → 一行命令安装](../README.zh.md#一行命令安装)，或手动版 [getting-started.zh.md](getting-started.zh.md) walkthrough。

本指南带你从"二进制在 `PATH` 上"走到"gateway 运行中"，只需五步，然后介绍如何把它跑成服务。

---

## 第 1 步 —— 获取二进制

### 通过 npm（任意 OS，Node ≥ 18）

```sh
npm install -g opendray        # 全局安装，把 `opendray` 加进 PATH
# 或者，不安装直接用：
npx opendray --help
# 或者用其他包管理器：
pnpm add -g opendray
yarn global add opendray
```

正确的平台二进制（`opendray-{linux,darwin}-{x64,arm64}`）会通过 `optionalDependencies` 自动选择 —— 没有 `postinstall` hook，安装时不会发网络请求。**不要**加 `--no-optional`：那会跳过平台包，导致启动器找不到二进制可执行。

### 通过 Release 归档

```sh
# 从 Releases 页选对应 OS/arch 的归档，然后：
tar -xzf opendray_*_linux_amd64.tar.gz
sudo install -m 0755 opendray /usr/local/bin/opendray
```

### 验证

```sh
opendray version          # 输出版本号、commit、构建日期
opendray --help           # 列出所有子命令
```

支持的平台：**Linux**（x64、arm64）和 **macOS**（x64、arm64）。
原生 Windows 不打包 —— 请用 WSL2，然后走 Linux 路径。

---

## 第 2 步 —— 准备 PostgreSQL 15+（含 pgvector）

opendray 把所有数据（session、memory、audit log）存在 PostgreSQL 里，其记忆子系统需要 [`pgvector`](https://github.com/pgvector/pgvector) 扩展。支持的服务端版本：**15、16、17**。

如果你已经有 Postgres，创建一个数据库和仅有 CRUD 权限的角色，然后用 superuser 启用扩展：

```sh
# 1. 安装 pgvector（每台主机执行一次）。
#    Ubuntu/Debian:  sudo apt install postgresql-16-pgvector
#    macOS (brew):   brew install pgvector
#    其他 / 源码编译: https://github.com/pgvector/pgvector#installation

# 2. 创建数据库 + 项目专属角色。
sudo -u postgres psql <<'SQL'
CREATE DATABASE opendray;
CREATE USER opendray WITH ENCRYPTED PASSWORD 'change-me';
GRANT ALL PRIVILEGES ON DATABASE opendray TO opendray;
SQL

# 3. 在该数据库中启用 pgvector（一次性，需 superuser）。
sudo -u postgres psql -d opendray -c 'CREATE EXTENSION IF NOT EXISTS vector;'
sudo -u postgres psql -d opendray -c 'GRANT ALL ON SCHEMA public TO opendray;'
```

扩展启用后，opendray 的 CRUD 专属角色就能跑 migration，不再需要 superuser。**运行时绝对不要让 opendray 用 superuser 角色连接** —— 给它一个项目专属账号，密码带外轮换。

---

## 第 3 步 —— 配置

opendray 从 TOML 文件**或**纯环境变量（12-factor 风格）读取配置 —— 环境变量始终优先于文件。唯一的硬性要求是数据库 URL；其他配置都有默认值。

### 方案 A —— 环境变量（适合容器 / 临时主机）

```sh
export OPENDRAY_DATABASE_URL="postgres://opendray:change-me@127.0.0.1:5432/opendray?sslmode=disable"
export OPENDRAY_ADMIN_PASSWORD="$(openssl rand -base64 24)"   # admin 登录密码
export OPENDRAY_LISTEN="127.0.0.1:8770"                       # 可选；这是默认值
```

| 变量 | 是否必填 | 默认值 | 用途 |
|---|---|---|---|
| `OPENDRAY_DATABASE_URL` | **是** | — | Postgres DSN |
| `OPENDRAY_ADMIN_PASSWORD` | 建议设置 | — | Web/移动端 admin 密码 |
| `OPENDRAY_ADMIN_USER` | 否 | `admin` | Admin 用户名 |
| `OPENDRAY_LISTEN` | 否 | `127.0.0.1:8770` | 监听地址 |
| `OPENDRAY_LOG_LEVEL` | 否 | `info` | `debug`/`info`/`warn`/`error` |
| `OPENDRAY_LOG_FORMAT` | 否 | `text` | `text`/`json` |

运行 `opendray serve` 时不带 `-config` 参数，即完全从环境变量加载配置。

### 方案 B —— config.toml

```sh
curl -fsSLO https://raw.githubusercontent.com/Opendray/opendray/main/config.example.toml
mv config.example.toml config.toml
$EDITOR config.toml        # 设置 [database].url 和 [admin].password
```

最少需要编辑的部分：

```toml
listen = "127.0.0.1:8770"

[database]
url = "postgres://opendray:change-me@127.0.0.1:5432/opendray?sslmode=disable"

[admin]
user     = "admin"
password = "use-a-real-password"
```

完整注释版见 [`config.example.toml`](../config.example.toml)（含日志、session 空闲检测、备份、vault、MCP 等配置）。用 `-config config.toml` 把配置文件传给下面的命令。在共享主机上请把 secret 从 TOML 中移出 —— 通过环境变量设置 `OPENDRAY_DATABASE_URL` / `OPENDRAY_ADMIN_PASSWORD`，让文件本身不含密钥。

---

## 第 4 步 —— 应用 Schema

```sh
opendray migrate                          # 纯环境变量配置
# 或
opendray migrate -config config.toml
```

幂等操作 —— schema 已是最新时重复运行是 no-op。必须在第一次 `serve` 之前成功执行。

---

## 第 5 步 —— 运行

```sh
opendray serve                            # 纯环境变量配置
# 或
opendray serve -config config.toml
```

这会在**前台**运行（Ctrl-C 停止）。运行后应可访问：

| URL | 内容 |
|---|---|
| `http://127.0.0.1:8770/admin/` | Web admin —— 用 `admin` + 你的密码登录 |
| `http://127.0.0.1:8770/api/v1/...` | REST + WebSocket API |

至此 gateway 已完整运行。如果不只是快速测试，建议用进程管理器保活，让它在重启后自动恢复、崩溃后自动重启 —— 见下一节。

---

## 跑成服务

`opendray serve` 就是服务 unit 启动命令该调用的东西。
opendray 附带已加固、开箱即用的 unit 文件；下面的步骤与
[README → 生产部署](../README.zh.md#生产部署) 一致，后者是权威参考（含完整 bootstrap、沙箱说明、反向代理/TLS）。

### Linux —— systemd

仓库附带一个加固过的 unit，路径为
[`deploy/systemd/opendray.service`](../deploy/systemd/opendray.service)
（`ExecStartPre` 跑 `migrate`、通过 `EnvironmentFile` 传 secret、
`on-failure` 重启策略、syscall/filesystem 沙箱）。

```sh
# 把二进制放到 /usr/local/bin/opendray，创建服务用户和状态目录：
sudo useradd -r -s /usr/sbin/nologin -d /var/lib/opendray opendray
sudo install -d -o opendray -g opendray -m 0700 /var/lib/opendray

# 非密钥配置 + 密钥文件（env 格式，mode 0640）：
sudo install -D -m 0640 config.toml /etc/opendray/config.toml
sudo install -D -m 0640 -o root -g opendray /dev/null /etc/opendray/env.d/secrets
echo 'OPENDRAY_ADMIN_PASSWORD=use-a-real-password' | sudo tee -a /etc/opendray/env.d/secrets

# 安装并启用 unit：
sudo cp deploy/systemd/opendray.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now opendray
sudo systemctl status opendray
journalctl -u opendray -f --no-pager
```

没有 systemd？（没有 systemd 的 LXC、OpenRC、runit、s6、supervisord……）让你的进程管理器指向 `opendray serve -config /etc/opendray/config.toml`，并把 `opendray migrate` 做成 pre-start 步骤跑一次。见
[README → 生产部署 §方案 B](../README.zh.md#方案-b--直接跑二进制--你自己的进程管理器)。

### macOS —— launchd

仓库附带一个 LaunchDaemon，路径为
[`deploy/launchd/com.opendray.opendray.plist`](../deploy/launchd/com.opendray.opendray.plist)
（开机启动、崩溃后重启、日志写到 `/usr/local/var/log/opendray/`）。

```sh
sudo install -d -m 0755 /usr/local/etc/opendray /usr/local/var/lib/opendray /usr/local/var/log/opendray
sudo install -m 0640 config.toml /usr/local/etc/opendray/config.toml
sudo /usr/local/bin/opendray migrate -config /usr/local/etc/opendray/config.toml

sudo cp deploy/launchd/com.opendray.opendray.plist /Library/LaunchDaemons/
sudo chown root:wheel /Library/LaunchDaemons/com.opendray.opendray.plist
sudo chmod 0644 /Library/LaunchDaemons/com.opendray.opendray.plist
sudo launchctl bootstrap system /Library/LaunchDaemons/com.opendray.opendray.plist
sudo launchctl print system/com.opendray.opendray
```

重启：`sudo launchctl kickstart -k system/com.opendray.opendray`。
卸载：`sudo launchctl bootout system/com.opendray.opendray`。

> 两个 unit 的完整文档 —— 包括 secret 布局说明以及为什么关掉 `MemoryDenyWriteExecute` —— 见
> [`deploy/README.md`](../deploy/README.md)。

---

## 保持更新

如何更新取决于你的安装方式：

- **通过 npm 安装** —— 用包管理器更新。`opendray update` 会在 npm 背后替换 `node_modules` 里的二进制，下次 install 时会被覆盖，所以这里不要用它。

  ```sh
  npm install -g opendray@latest
  ```

- **通过 Release 下载 / wizard 安装** —— 二进制自我更新（下载最新 release、校验 SHA-256、原子替换）：

  ```sh
  opendray update --check          # 仅探测版本，不更新
  sudo opendray update --restart   # 应用更新，然后重启服务
  ```

---

## 排错

**`the matching platform package "opendray-…" was not installed`**
npm 运行时带了 `--no-optional`，或安装被中断。重新运行
`npm install -g opendray`（不带 `--no-optional`）。

**`unsupported platform`**
npm 包只覆盖 Linux/macOS 的 x64/arm64。其他目标请从源码构建 —— 见 [quickstart.md](quickstart.md)。

**`config: database.url is empty`**
`OPENDRAY_DATABASE_URL` 和 `[database].url` 都没有设置。设置其中一个（第 3 步）。

**`connection refused` —— migrate/serve 时**
Postgres 没有运行，或者 DSN 有误。确认服务已启动，并检查 DSN 中的 host/port/凭据是否正确。

**pgvector / `extension "vector" is not available`**
扩展未安装在服务器上，或未在 opendray 数据库中启用。重做第 2 步（安装 OS 包，然后以 superuser 执行 `CREATE EXTENSION vector`）。

**端口已被占用**
修改 `OPENDRAY_LISTEN`（或 config.toml 里的 `listen`）为空闲端口。

---

## 下一步

- [README → 生产部署](../README.zh.md#生产部署) —— 完整部署参考（systemd / launchd / 自定义进程管理器、加固、反向代理）
- [`docs/operator-guide.md`](operator-guide.md) —— 运维：反向代理/TLS 拓扑、加密 DB 备份、数据导出/导入
- [`docs/integration-guide.md`](integration-guide.md) —— 基于 REST + WebSocket API 构建外部集成
- [`docs/getting-started.zh.md`](getting-started.zh.md) —— 如果你更想要一步步引导式的端到端 setup，看这里
