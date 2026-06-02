# opendray

<div dir="rtl" lang="fa">

یک gateway خودمیزبان برای اجرای Claude Code، Codex، Gemini و shell؛ همراه با یک لایه memory مشترک و local-first بین همه آن‌ها.

سشن‌های کاری را روی زیرساخت خودتان اجرا کنید، از طریق web، mobile یا chat کنترلشان کنید، و برای integrationهای دیگر هم یک REST + WebSocket API باز داشته باشید.

[opendray.dev](https://opendray.dev)

[English](README.md) · [简体中文](README.zh.md) · فارسی · [Español](README.es.md) · [Português](README.pt-BR.md) · [日本語](README.ja.md) · [한국어](README.ko.md) · [Français](README.fr.md) · [Deutsch](README.de.md) · [Русский](README.ru.md)

---

## چرا opendray وجود دارد؟

opendray برای حل چند اصطکاک جدی در کار روزمره با AI coding CLIها ساخته شده است.

### وقتی laptop sleep می‌شود، session نباید از بین برود

اگر Claude Code یا Codex را از طریق SSH روی laptop اجرا کنید، به‌محض این‌که درِ دستگاه را ببندید یا Wi‑Fi قطع شود، agent هم عملاً از بین می‌رود. context، tool callهای در حال اجرا، diff نیمه‌کاره‌ای که قرار بوده review کنید، و هر چیزی که وسط کار بوده ممکن است از دست برود.

opendray agent را روی hostای اجرا می‌کند که قرار نیست sleep شود؛ مثلاً یک Mac mini زیر میز، یک NAS یا یک VPS. بعد می‌توانید از web panel، mobile app مبتنی بر Flutter یا حتی از طریق chat دوباره به همان session وصل شوید. session به کارش ادامه می‌دهد؛ چه online باشید، چه نباشید.

### rate limit نباید کارتان را خراب کند

اگر چند Anthropic account دارید، مثلاً work account و personal account یا Pro و Family، opendray آن‌ها را مثل یک capacity pool مدیریت می‌کند. status مربوط به quota، tier و تعداد active sessionهای هر account را نشان می‌دهد، sessionهای جدید را بین accountهای فعال balance می‌کند، و حتی اجازه می‌دهد یک live session را بدون از دست دادن conversation به account دیگری منتقل کنید. transcript همراه session جابه‌جا می‌شود. همین مدل برای Codex و Gemini accountها هم در نظر گرفته شده است.

### memory باید جزء اصلی سیستم باشد، نه یک قابلیت اضافه‌شده در آخر

بیشتر AI CLIها در هر session دوباره project context را از صفر index می‌کنند و tokenها را صرف retrievalهای تکراری می‌کنند. opendray یک local-first vector store ارائه می‌دهد که با ONNX، Ollama و LM Studio embeddings کار می‌کند و retrieval را در سه scope انجام می‌دهد: user، project و session. همچنین drift و ناسازگاری بین memory layerها را تشخیص می‌دهد. داده‌ها روی network و infrastructure خودتان باقی می‌مانند.

---

## OpenDray چیست؟

**OpenDray** ابزارهای AI CLI شما، مثل Claude Code، Codex، Gemini و هر shell دیگری را به یک platform واحد تبدیل می‌کند؛ platformای که از هرجا قابل دسترسی است و کنترلش دست خودتان می‌ماند.

می‌توانید working sessionها را روی home server، NAS یا VPS اجرا کنید، وقتی idle می‌شوند یا نیاز به input دارند در Telegram notification بگیرید، و فقط با reply کردن از روی گوشی، prompt بعدی را برای ادامه کار ارسال کنید.

همه این قابلیت‌ها از طریق یک self-hosted gateway ارائه می‌شود؛ بدون وابستگی به سرویس واسط، بدون lock-in، و با مالکیت کامل روی data و communication.

- **یک backend، چند interface:** یک Go binary واحد که React web admin و Flutter mobile app را سرویس می‌دهد. تمام operationها از طریق REST و WebSocket API هم برای third-party integrationها قابل استفاده است.
- **کانال‌های دوطرفه، بدون lock-in:** Telegram، Slack، Discord، Feishu، DingTalk، WeCom و یک bridge adapter برای اتصال‌های custom. replyهایی که در هر channel می‌فرستید به session درست برمی‌گردند.
- **local-first memory:** استفاده از ONNX، Ollama و LM Studio embeddings، retrieval در سطح user/project/session، smart ranking برای resultها، و تشخیص conflict بین memory layerها. vector data روی network خودتان می‌ماند و به بیرون ارسال نمی‌شود.
- **API مناسب integration:** scoped API key، audit log برای هر call و reverse-proxy mountها. می‌توانید opendray را به‌عنوان gateway پشت محصول خودتان استفاده کنید یا فقط آن را command centre شخصی خودتان بدانید.
- **multi-account management برای Claude:** چند `claude login` را داخل gateway قرار دهید. panel با filesystem watcher خودش آن‌ها را پیدا می‌کند، sessionهای جدید را بین accountهای فعال balance می‌کند، و اجازه می‌دهد یک live session را **بدون از دست دادن conversation** بین accountها جابه‌جا کنید. transcript در پشت صحنه منتقل می‌شود.
- **ظرفیت accountها در یک نگاه:** هر ردیف، وضعیت account را live نشان می‌دهد؛ از جمله `subscription tier`، `rate-limit tier`، `active sessions`، `last-used` و `current Anthropic email` تا سریع‌تر account درست را انتخاب کنید.
- **self-hosted با license شفاف:** Apache 2.0، یک static binary، releaseهای امضاشده با cosign و SPDX SBOM. بدون telemetry، بدون cloud account و بدون subscription.

---

## architecture در یک نگاه

یک Go binary روی host شما همه‌چیز را اداره می‌کند. clientها از طریق HTTP/WebSocket، sessionها را راه می‌اندازند، session manager هر AI CLI را در یک PTY مستقل spawn می‌کند، و memory layer شامل shared state در Postgres با vector embeddingsای است که از provider خودتان می‌آید.

</div>

```mermaid
flowchart LR
    subgraph clients [Clients]
        web[Web admin<br/>React SPA]
        mob[Mobile app<br/>Flutter]
        chat[Chat<br/>Telegram, Slack,<br/>Discord, Feishu,<br/>DingTalk, WeCom]
        api[Third-party apps<br/>REST + WS]
    end

    subgraph gw [opendray gateway · single Go binary on your host]
        direction TB
        http[HTTP + WS<br/>chi · auth · audit]
        sess[Session manager<br/>PTY · ring buffer]
        mem[Memory layer<br/>three-domain retrieval]
    end

    subgraph cli [AI CLIs · spawned via PTY]
        cc[Claude Code]
        co[Codex]
        ge[Gemini]
        sh[Shell]
    end

    subgraph data [Persistence · stays on your network]
        pg[(PostgreSQL<br/>+ pgvector)]
        em[ONNX · Ollama<br/>LM Studio embeddings]
    end

    clients --> http
    http --> sess
    http --> mem
    sess --> cc
    sess --> co
    sess --> ge
    sess --> sh
    sess -.-> mem
    mem --> pg
    mem --> em
```

<div dir="rtl" lang="fa">

هرچه در دیاگرام می‌بینید روی network خودتان اجرا می‌شود. بدون وابستگی به cloud، بدون inference بیرون از کنترل شما.

---

## وضعیت نسخه

نسخه ۲٫۷٫۱ آخرین release فعلی است و development روی v2 همچنان ادامه دارد.

برای policy مربوط به major-as-generation، فایل [`VERSIONING.md`](VERSIONING.md) را ببینید. در این پروژه، major الزاماً به معنی breaking change به معنای سخت‌گیرانه SemVer نیست؛ بیشتر به نسل محصول اشاره دارد. برای تاریخچه کامل releaseها هم [`CHANGELOG.md`](CHANGELOG.md) را بخوانید.

این generation شامل موارد زیر است:

- **one-line install و uninstall wizard** برای Linux و macOS. مسیر Windows از طریق WSL2 انجام می‌شود. wizard، operator را مرحله‌به‌مرحله از Postgres setup، نصب AI CLIها، تنظیم admin credentialها، تعیین listen address، نصب binary، اجرای schema migration و ثبت service عبور می‌دهد.
- **self-managing binary:** با دستورهای `opendray update / start / stop / restart / status / providers list / providers update` برای کارهای روزمره لازم نیست مستقیم سراغ `systemctl` یا `launchctl` بروید.
- **release pipeline با Goreleaser:** شامل cross-compiled binaryها برای `linux/darwin × amd64/arm64`، keyless signing با cosign و Sigstore، SPDX SBOM و atomic verified self-update.

---

## نصب

### one-line installer

**Linux / macOS / WSL2**

</div>

```sh
curl -fsSL https://raw.githubusercontent.com/Opendray/opendray/main/scripts/install.sh | bash
```

<div dir="rtl" lang="fa">

**Windows:** ابتدا WSL2 را راه‌اندازی می‌کند و سپس Linux installer را داخل آن اجرا می‌کند. [جزئیات →](scripts/README.md#windows)

</div>

```powershell
irm https://raw.githubusercontent.com/Opendray/opendray/main/scripts/install-windows.ps1 | iex
```

<div dir="rtl" lang="fa">

این مسیر Postgres setup، نصب AI CLIها، admin credentialها و service registration را انجام می‌دهد و معمولاً بعد از حدود ۵ تا ۱۰ دقیقه یک gateway آماده تحویل می‌دهد. اگر می‌خواهید دقیقاً بدانید wizard چه کاری انجام می‌دهد، چه layoutای می‌سازد، چه optionهایی دارد و چطور باید debug کنید، به [**`scripts/README.md`**](scripts/README.md) سر بزنید.

> **راهنمای دستی و قدم‌به‌قدم می‌خواهید؟**  
> [`docs/getting-started.md`](docs/getting-started.md) را بخوانید. این راهنما همان مسیر wizard را در قالب یک walkthrough حدوداً ۱۵ دقیقه‌ای باز می‌کند تا بتوانید هر مرحله را خودتان بررسی کنید.

---

### npm / npx

نیازمند Node نسخه ۱۸ یا بالاتر.

نصب global و اضافه شدن `opendray` به `PATH`:

</div>

```sh
npm install -g opendray
```

<div dir="rtl" lang="fa">

یا اجرای on-demand بدون نصب دائمی:

</div>

```sh
npx opendray
```

<div dir="rtl" lang="fa">

این روش فقط **binary** را نصب می‌کند؛ نه wizard، نه service و نه Postgres.

binary package مناسب platform شما، مثل `opendray-{linux,darwin}-{x64,arm64}` از طریق `optionalDependencies` دریافت می‌شود. این مدل شبیه esbuild و Biome است: بدون `postinstall` و بدون network call هنگام نصب. این روش برای scripted environmentها، ephemeral runnerها یا زمانی مناسب است که Postgres و process supervisor خودتان را از قبل دارید.

در این حالت، database و gateway را خودتان آماده و اجرا می‌کنید:

</div>

```sh
# 1. PostgreSQL 15+ with pgvector
# یک DSN تعریف کنید و برای admin هم password بگذارید.

export OPENDRAY_DATABASE_URL="postgres://opendray:pw@127.0.0.1:5432/opendray?sslmode=disable"
export OPENDRAY_ADMIN_PASSWORD="$(openssl rand -base64 24)"

# 2. schema را اعمال کنید و بعد gateway را در foreground اجرا کنید.

opendray migrate
opendray serve        # → http://127.0.0.1:8770/admin/
```

<div dir="rtl" lang="fa">

راهنمای کامل راه‌اندازی pgvector، تنظیم `config.toml`، اجرای systemd یا launchd service و update در [`docs/install-binary.fa.md`](docs/install-binary.fa.md) آمده است.

---

## حذف نصب در Linux و macOS

### حذف پیش‌فرض

gateway را stop می‌کند و binary را حذف می‌کند، اما `config.toml`، مسیر data مثل bcrypt keyfile، sessionها، noteها و vault، logها و PostgreSQL database را نگه می‌دارد تا اگر دوباره نصب کردید، از همان‌جا ادامه دهید.

</div>

```sh
curl -fsSL https://raw.githubusercontent.com/Opendray/opendray/main/scripts/uninstall.sh | bash
```

<div dir="rtl" lang="fa">

### حذف کامل

علاوه بر موارد بالا، PostgreSQL database و role مربوطه، config، data، logها و service user را هم حذف می‌کند. بعد از حذف هم یک verification step اجرا می‌شود تا اگر چیزی باقی مانده بود، بی‌سروصدا نادیده گرفته نشود.

</div>

```sh
curl -fsSL https://raw.githubusercontent.com/Opendray/opendray/main/scripts/uninstall.sh | OPENDRAY_PURGE=1 bash
```

<div dir="rtl" lang="fa">

---

## دستورات روزمره

بعد از نصب، خود binary `opendray` lifecycle سرویس را مدیریت می‌کند. لازم نیست دستورهای طولانی `systemctl` یا `launchctl` را حفظ کنید.

</div>

```sh
sudo opendray update --restart   # download latest release, verify SHA, atomic replace + restart
sudo opendray providers update   # bump installed AI CLIs: claude / codex / gemini
opendray providers list          # see installed AI CLIs and their versions
sudo opendray start              # start | stop | restart | status via systemd / launchd
opendray --help                  # show all subcommands
```

<div dir="rtl" lang="fa">

---

## انتخاب روش راه‌اندازی

همه مسیرهای پشتیبانی‌شده امکان spawn کردن session، دسترسی به AI CLIها، encrypted backup و integration API کامل را دارند.

opendray یک host-resident gateway است. AI CLIها را از طریق PTY اجرا می‌کند و process state مثل `~/.claude`، `ssh-agent` و project fileها را با آن‌ها share می‌کند. این مدل با container isolationای که production Docker تحمیل می‌کند خوب جور درنمی‌آید؛ بنابراین Docker در v2.x مسیر پشتیبانی‌شده محسوب نمی‌شود.

</div>

| Path | Best for | Next |
|---|---|---|
| **ready-made binary** | Run-only setup on Linux/macOS with your own process manager | [release page](https://github.com/Opendray/opendray/releases), then [production deployment](#انتشار-عملیاتی) |
| **systemd unit** | bare-metal, VM, or Linux LXC | [Option 1: systemd](#گزینه-۱-systemd-برای-bare-metal--vm--lxc) |
| **macOS LaunchDaemon** | Mac mini or Mac Studio used as a home server | [Option 3: macOS launchd](#گزینه-۳-macos-launchd-برای-mac-mini--mac-studio-بهعنوان-home-server) |
| **build from source** | development, contribution, or custom builds | [Quickstart](#quickstart-مسیر-۵-دقیقهای-برای-dev) |

<div dir="rtl" lang="fa">

---

## Quickstart: مسیر ۵ دقیقه‌ای برای dev

برای راهنمای کامل، همراه با prerequisiteها و troubleshooting، فایل [`docs/quickstart.md`](docs/quickstart.md) را ببینید.

مسیر خلاصه development:

</div>

```bash
# 1. Have a Postgres 15+ running on 127.0.0.1:5432 with pgvector enabled.
#    Example:
#    apt install postgresql-16 postgresql-16-pgvector
#    or:
#    brew install postgresql@16 pgvector
#
#    اگر remote PG را ترجیح می‌دهید،
#    [database].url را به DSN همان database اشاره دهید.

# 2. Local config. این فایل از قبل gitignored است.

cp config.example.toml config.toml
$EDITOR config.toml          # set [database].url, [admin].password

# 3. Build the web bundle into the embed tree.

cd app/web && pnpm install && pnpm build && cd ../..

# 4. Apply schema.

go run ./cmd/opendray migrate -config config.toml

# 5. Run.

go run ./cmd/opendray serve -config config.toml

# → REST + WS:  http://127.0.0.1:8770/api/v1/...
# → Web admin:  http://127.0.0.1:8770/admin/
```

<div dir="rtl" lang="fa">

این روش opendray را در foreground اجرا می‌کند و با `Ctrl-C` متوقف می‌شود. اگر daemonای می‌خواهید که برای مدت طولانی بالا بماند، بخش **انتشار عملیاتی** را ببینید.

---

## انتشار عملیاتی

چهار مسیر deployment پشتیبانی‌شده داریم؛ هرکدام که به environment شما می‌خورد همان را انتخاب کنید. همه آن‌ها auto-restart هنگام crash، state پایدار، و جداسازی secrets از config را فراهم می‌کنند.

### گزینه ۱: systemd برای bare-metal / VM / LXC

این مسیر، روش پیشنهادی برای Linux deployment است.

یک hardened unit در [`deploy/systemd/opendray.service`](deploy/systemd/opendray.service) ارائه می‌شود که شامل sandboxing مثل `ProtectSystem=strict`، `NoNewPrivileges`، `MemoryDenyWriteExecute` و capability scrub است. boot flow آن `migrate` سپس `serve` است و یک graceful stop window بیست‌ثانیه‌ای دارد.

**اول binary را بگیرید.** یا یک archive آماده را از [release page](https://github.com/Opendray/opendray/releases) بردارید، مثل `opendray_*_linux_.tar.gz` که به یک single `opendray` binary unpack می‌شود، یا از طریق [Quickstart](#quickstart-مسیر-۵-دقیقهای-برای-dev) از source build بگیرید: `go build ./cmd/opendray`.

</div>

```bash
# 1. Install the binary you just grabbed or built.

sudo install -m 0755 /path/to/opendray /usr/local/bin/opendray

# 2. Create the service user and state dir.

sudo useradd -r -s /usr/sbin/nologin -d /var/lib/opendray opendray
sudo install -d -o opendray -g opendray -m 0700 /var/lib/opendray

# 3. Drop config and secrets. root-owned; mode 0640.

sudo install -D -m 0640 config.example.toml /etc/opendray/config.toml
sudo $EDITOR /etc/opendray/config.toml     # set [database].url etc.

sudo install -D -m 0640 -o root -g opendray /dev/null /etc/opendray/env.d/secrets
sudo $EDITOR /etc/opendray/env.d/secrets   # OPENDRAY_ADMIN_PASSWORD=...

# 4. Install and enable the unit.

sudo cp deploy/systemd/opendray.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now opendray

# 5. Verify.

sudo systemctl status opendray
sudo journalctl -u opendray -f --no-pager
```

<div dir="rtl" lang="fa">

این unit، دستور `opendray migrate` را به‌عنوان `ExecStartPre` اجرا می‌کند؛ بنابراین در اولین boot، همه migrationها قبل از شروع `serve` اعمال می‌شوند. restartها هم `on-failure` هستند، با back-off پنج‌ثانیه‌ای و limit پنج‌بار در دقیقه.

---

### گزینه ۲: direct binary + process manager دلخواه

برای LXC بدون systemd، FreeBSD `rc.d`، OpenRC یا هر process manager دیگری.

یک بار build بگیرید و با process managerی که از قبل دارید اجرا کنید:

</div>

```bash
# Cross-compile a release archive locally:

goreleaser release --clean --snapshot
ls dist/        # opendray_*_linux_amd64.tar.gz etc.

# Or grab a published release artefact:
# https://github.com/Opendray/opendray/releases
```

<div dir="rtl" lang="fa">

بعد process manager خودتان مثل s6، runit، supervisord یا runwhen را به این command اشاره دهید:

</div>

```sh
/usr/local/bin/opendray serve -config /etc/opendray/config.toml
```

<div dir="rtl" lang="fa">

قبل از اولین `serve`، یک بار این command را اجرا کنید:

</div>

```sh
opendray migrate -config /etc/opendray/config.toml
```

<div dir="rtl" lang="fa">

یا آن را به‌عنوان pre-start hook در process manager دلخواهتان قرار دهید.

---

### گزینه ۳: macOS launchd برای Mac mini / Mac Studio به‌عنوان home server

برای Apple Silicon Mac mini یا Mac Studio که ۲۴/۷ روشن است.

یک LaunchDaemon در [`deploy/launchd/com.opendray.opendray.plist`](deploy/launchd/com.opendray.opendray.plist) ارائه می‌شود که هنگام boot و قبل از user login بالا می‌آید، در صورت crash با throttle پنج‌ثانیه‌ای دوباره start می‌شود، و در `/usr/local/var/log/opendray/` log می‌نویسد.

</div>

```bash
# 1. Install the darwin binary, config and state dirs.

sudo install -m 0755 ./opendray /usr/local/bin/opendray
sudo install -d -m 0755 \
  /usr/local/etc/opendray \
  /usr/local/var/lib/opendray \
  /usr/local/var/log/opendray

sudo install -m 0640 config.example.toml /usr/local/etc/opendray/config.toml
sudo $EDITOR /usr/local/etc/opendray/config.toml   # set [database].url etc.

# 2. Apply migrations once.

sudo /usr/local/bin/opendray migrate \
  -config /usr/local/etc/opendray/config.toml

# 3. Install and load the LaunchDaemon.

sudo cp deploy/launchd/com.opendray.opendray.plist /Library/LaunchDaemons/
sudo chown root:wheel /Library/LaunchDaemons/com.opendray.opendray.plist
sudo chmod 0644 /Library/LaunchDaemons/com.opendray.opendray.plist
sudo launchctl bootstrap system /Library/LaunchDaemons/com.opendray.opendray.plist

# 4. Verify.

sudo launchctl print system/com.opendray.opendray
tail -f /usr/local/var/log/opendray/opendray.log
```

```sh
# Restart
sudo launchctl kickstart -k system/com.opendray.opendray

# Full unload
sudo launchctl bootout system/com.opendray.opendray
```

```sh
# PostgreSQL on macOS
brew install postgresql@17 && brew services start postgresql@17

# pgvector
brew install pgvector
```

```text
postgres://$USER@127.0.0.1:5432/opendray
```

```sql
CREATE EXTENSION vector;
```

<div dir="rtl" lang="fa">

برای noteهای مخصوص Proxmox LXC، مثل PTY داخل unprivileged containerها، networking و cgroup settingها، به [`deploy/lxc/proxmox-pty-notes.md`](deploy/lxc/proxmox-pty-notes.md) نگاه کنید.

برای reverse-proxy و TLS termination، مثل nginx، Caddy، Traefik و Cloudflare Tunnel، بخش Topology در [`docs/operator-guide.md`](docs/operator-guide.md) را ببینید.

---

## اختیاری: فعال‌سازی encrypted DB backup و data export

</div>

```bash
# Master passphrase. env-only: never write it into config.toml.

export OPENDRAY_BACKUP_KEY="$(openssl rand -base64 32)"
export OPENDRAY_BACKUP_ENABLED=1

# pg_dump / pg_restore must match the server's major version.
# Example for Apple Silicon dev machines pointing at a PG17 server:

export OPENDRAY_BACKUP_PG_DUMP_PATH=/opt/homebrew/opt/postgresql@17/bin/pg_dump
export OPENDRAY_BACKUP_PG_RESTORE_PATH=/opt/homebrew/opt/postgresql@17/bin/pg_restore
```

<div dir="rtl" lang="fa">

opendray را restart کنید. sidebar یک صفحه Backups در مسیر `/backups` برای encrypted PostgreSQL dump و restore اضافه می‌کند. مسیر `/export` هم برای zip-bundle data export و import استفاده می‌شود. برای چرخه کامل، بخش Backup در [`docs/operator-guide.md`](docs/operator-guide.md) را ببینید.

کل web UI داخل یک Go binary قرار گرفته است؛ بنابراین در runtime نیازی به Node.js، static file server جداگانه یا ابزارهایی مثل Caddy و Nginx ندارید. البته Cloudflare Tunnel همچنان می‌تواند TLS را قبل از رسیدن traffic به پورت `8770` terminate کند.

---

## ساختار پروژه

</div>

```text
cmd/opendray/
  entrypoint اصلی binary

internal/
├── app/           bootstrap و اتصال componentهای اصلی برنامه
├── audit/         دریافت eventها از bus و ذخیره در audit_log
├── auth/          admin Bearer tokenها
├── backup/        encrypted DB backup و data import/export
├── catalog/       provider manifestها و per-user config
├── channel/       channel management و Telegram integration
├── config/        load کردن TOML config و OPENDRAY_* environment variables
├── eventbus/      internal Pub/Sub
├── gateway/       HTTP routing با chi، middleware و slog
├── integration/   external app registration، Reverse Proxy و WebSocket eventها
├── memory/        shared persistent memory بین CLIها
├── session/       session management، PTY، ring buffer و WebSocket stream
├── store/         PostgreSQL connection با pgx و migrationها
├── version/       version و build metadata
└── web/           embedded web UI با go:embed

app/web/
  React 19 + TypeScript + Vite single-page app

app/mobile/
  Flutter app برای Android و iOS با featureهای مشابه web version

docs/
├── design.md      سند اصلی طراحی و مرجع پروژه
└── adr/           Architecture Decision Recordها
```

<div dir="rtl" lang="fa">

---

## web frontend

`app/web/` یک SPA واحد را در `internal/web/dist/` build می‌کند. Go binary آن را embed می‌کند و از مسیر `/admin/*` سرو می‌کند. Vite dev server روی `:5173` مسیر `/api` را به `:8770` proxy می‌کند تا development با HMR انجام شود.

</div>

```bash
# dev: hot reload on React side, separate Go server for API

cd app/web && pnpm dev       # http://localhost:5173

go run ./cmd/opendray serve -config ../../config.toml   # other terminal

# prod: one binary delivers everything

cd app/web && pnpm build     # writes ../../internal/web/dist
cd ../..
go build ./cmd/opendray      # bakes dist into the binary
./opendray serve -config config.toml
```

<div dir="rtl" lang="fa">

برای frontend stack، یعنی `React` + `Vite` + `Tailwind v4` + `shadcn/ui` + `TanStack Router/Query` + `Zustand` + `xterm.js`، و noteهای مربوط به milestoneهای W، به [`app/web/README.md`](app/web/README.md) نگاه کنید.

---

## مستندات

- [`docs/getting-started.md`](docs/getting-started.md): اگر تازه شروع کرده‌اید، از اینجا شروع کنید؛ از صفر تا اولین session در حدود ۱۵ دقیقه، شامل نصب CLIهای موردنیاز و bootstrap کردن Postgres.
- [`docs/install-binary.fa.md`](docs/install-binary.fa.md): نصب از npm package یا release binary، با فرض این‌که Postgres را خودتان آماده می‌کنید، و اجرا به‌عنوان systemd یا launchd service.
- [`docs/quickstart.md`](docs/quickstart.md): مسیر پنج‌دقیقه‌ای برای dev environment؛ مناسب وقتی که moving partها را از قبل می‌شناسید.
- [`docs/operator-guide.md`](docs/operator-guide.md): مرجع deploy و ops برای production-like setupها.
- [`docs/integration-guide.md`](docs/integration-guide.md): راهنمای نوشتن external integration با هر زبان برنامه‌نویسی.
- [`VERSIONING.md`](VERSIONING.md): استراتژی versioning مبتنی بر major-as-generation.
- [`CHANGELOG.md`](CHANGELOG.md): تاریخچه releaseها.

---

## تست‌ها

</div>

```bash
go test -race ./...          # backend
cd app/web && pnpm build     # web: TS strict + Vite production build
```

<div dir="rtl" lang="fa">

smoke flowهای end-to-end را بر اساس هر milestone در commit messageها track می‌کنیم. یک Playwright harness هم به‌عنوان follow-up برنامه‌ریزی شده است.

---

## نسبت به v1

v1 یعنی `Opendray/opendray` codebase قدیمی پروژه است که حالا archived شده. v2 نسل فعلی و active پروژه است؛ feature-complete است و تنها branchای است که development می‌گیرد.

از ۱۶ builtin موجود در v1، چهار مورد به backend در v2 منتقل شده‌اند. بقیه به سمت client-side featureها، channel adapterها یا integration API consumerها منتقل شده‌اند.

---

## License

Apache 2.0. فایل [`LICENSE`](LICENSE) را ببینید.

v1 تحت MIT license بود؛ v2 license جداگانه خودش را دارد.

</div>
