<div dir="rtl" lang="fa">

<p align="center">
  <a href="https://opendray.dev"><img src="docs/assets/logo.png" alt="opendray" width="180"></a>
</p>

<h1 align="center">opendray</h1>

<p align="center">
  <strong>درگاه self-hosted برای Claude Code · Codex · Gemini · shell، با یک لایه حافظه مشترک local-first بین همه آن‌ها.</strong>
  <br/>
  <sub>سشن‌ها را روی زیرساخت خودتان اجرا کنید. از وب، موبایل یا chat کنترلش کنید. یک API باز REST + WebSocket هم برای integrationها دارید.</sub>
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
  🌐 <strong>فارسی</strong> · <a href="README.zh.md">简体中文</a> · <a href="README.md">English</a>
</p>

---

## ابزار OpenDray چیست؟

ابزار **OpenDray** ابزارهای AI CLI مورد استفاده شما، از جمله Claude Code، Codex، Gemini و هر Shell دیگری را به یک بستر یکپارچه تبدیل می‌کند که از هر جایی قابل دسترسی است. سشن‌های کاری خود را روی سرور خانگی، NAS یا VPS اجرا کنید، هنگام توقف یا بیکار شدن آن‌ها در تلگرام اعلان دریافت کنید و تنها با پاسخ دادن از طریق گوشی، پرامپت بعدی را برای ادامه کار ارسال کنید. همه این قابلیت‌ها از طریق یک درگاه Self-Hosted تحت کنترل کامل شما ارائه می‌شوند؛ بدون وابستگی به سرویس‌های واسط و با مالکیت کامل بر داده‌ها و ارتباطات.

- 🛰 **یک بک‌اند، سه سطح:** یک باینری Go واحد که یک ادمین وب React و یک اپ موبایل فلاتر را سرویس می‌دهد، و هر عملیات را هم از طریق REST + WebSocket API برای integrationهای third-party در دسترس می‌گذارد.
- 💬 **شش کانال دوطرفه، بدون قفل‌شدن در اکوسیستم‌های بسته:** تلگرام، اسلک، دیسکورد، Feishu (飞书)، DingTalk (钉钉)، WeCom (企业微信)، به‌علاوه یک Bridge adapter برای هر چیز سفارشی. replyها در هر کانال به سشن درست برمی‌گردند.
- 🧠 **حافظهٔ محلی‌محور:** استفاده از مدل‌های تعبیه‌سازی ONNX، Ollama و LM Studio با قابلیت بازیابی در سه دامنهٔ کاربر، پروژه و سشن، رتبه‌بندی هوشمند نتایج، و شناسایی تعارض بین لایه‌های مختلف داده. همهٔ داده‌های برداری در شبکهٔ شما باقی می‌مانند و به بیرون از آن ارسال نمی‌شوند.
- 🔌 **API در سطح integration:** API keyهای scoped، audit log برای هر call، reverse-proxy mountها. opendray را می‌توانید هم به‌عنوان درگاه پشت محصول خودتان ببینید، هم فقط یک مرکز فرمان شخصی.
- 🔑 **مدیریت چند حساب Claude:** چند حساب `claude login` را داخل gateway بگذارید؛ پنل با filesystem watcher خودش آن‌ها را پیدا می‌کند، سشن‌های جدید را بین حساب‌های فعال balance می‌کند، و اجازه می‌دهد یک سشن زنده را بین حساب‌ها **بدون از دست دادن گفتگو** جابه‌جا کنید (transcript در پشت صحنه منتقل می‌شود). هر ردیف ظرفیت حساب را به صورت زنده نشان می‌دهد (subscription tier، rate-limit tier، active sessions، last-used، current Anthropic email) تا در یک نگاه حساب درست را انتخاب کنید.
- 🔒 **self-hosted با وضعیت مجوز شفاف:** آپاچی 2.0، یک باینری static، انتشارهای امضاشده با cosign همراه با SPDX SBOM. بدون telemetry، بدون حساب ابری، بدون subscription.

## وضعیت

نسخه ۲٫۶٫۰ (آخرین انتشار): توسعه v2 همچنان ادامه دارد. برای policy مربوط به major-as-generation به [`VERSIONING.md`](VERSIONING.md) مراجعه کنید؛ اینجا ماژور به معنی نسل محصول است، نه لزوماً یک "breaking change" به معنای سخت‌گیرانه SemVer. برای تاریخچه کامل انتشارها هم [`CHANGELOG.md`](CHANGELOG.md) را ببینید.

این نسل شامل موارد زیر است:

- **ویزاردهای نصب و حذف نصب تک‌خطی** (لینوکس + مک؛ ویندوز از طریق WSL2 هدایت می‌شود). این ویزاردها اپراتور را مرحله‌به‌مرحله از راه‌اندازی Postgres، نصب AI-CLI، تنظیم credentialهای ادمین، تعیین listen address، نصب باینری، اجرای schema migration، و ثبت service عبور می‌دهند.
- **باینری خودمدیریت:** با `opendray update / start / stop / restart / status / providers list / providers update` اپراتورها برای کارهای روزمره دیگر نیازی ندارند مستقیم سراغ `systemctl` / `launchctl` بروند.
- **پایپ‌لاین انتشار با Goreleaser:** شامل باینری‌های cross-compiled (linux/darwin × amd64/arm64)، امضای keyless با cosign (Sigstore)، SPDX SBOM، و self-update اتمیِ verified است.

## نصب

### نصب‌کننده تک‌خطی

**لینوکس / مک / WSL2**

<div dir="ltr">

```sh
curl -fsSL https://raw.githubusercontent.com/Opendray/opendray/main/scripts/install.sh | bash
```

</div>

**ویندوز:** ابتدا WSL2 را راه‌اندازی می‌کند، سپس installer لینوکس را داخل آن اجرا می‌کند. [جزئیات →](scripts/README.md#windows)

<div dir="ltr">

```powershell
irm https://raw.githubusercontent.com/Opendray/opendray/main/scripts/install-windows.ps1 | iex
```

</div>

این مسیر Postgres setup، نصب AI-CLI، credentialهای ادمین، و service registration را جلو می‌برد و حدود ۵ تا ۱۰ دقیقه بعد یک gateway آماده تحویل می‌دهد. اگر خواستید ببینید ویزارد دقیقاً چه می‌کند، چه چیدمانی می‌سازد، چه optionهایی دارد و چطور باید debug کنید، به [**`scripts/README.md`**](scripts/README.md) سر بزنید.

> **راهنمای گام‌به‌گام دستی می‌خواهید؟** [**docs/getting-started.md**](docs/getting-started.md) را بخوانید: یک راهنمای سرتاسری پانزده‌دقیقه‌ای که همان مسیر ویزارد را قدم‌به‌قدم باز می‌کند تا خودتان هر مرحله را چک کنید.

### حذف نصب (لینوکس / مک)

**پیش‌فرض:** درگاه را متوقف می‌کند و باینری را حذف می‌کند، اما `config.toml`، data directory شما (bcrypt keyfile، sessions، notes، vault)، logs، و PostgreSQL دیتابیس را **نگه می‌دارد** تا نصب مجدد از همان‌جایی که رها کرده بودید ادامه پیدا کند:

<div dir="ltr">

```sh
curl -fsSL https://raw.githubusercontent.com/Opendray/opendray/main/scripts/uninstall.sh | bash
```

</div>

**حذف کامل:** همچنین PG دیتابیس + نقش را پاک می‌کند، config / data / logs را می‌زند، و service user را برمی‌دارد. یک مرحله verification بعد از حذف هم دارد که اگر چیزی جا مانده باشد بی‌سر‌وصدا رد نمی‌شود:

<div dir="ltr">

```sh
curl -fsSL https://raw.githubusercontent.com/Opendray/opendray/main/scripts/uninstall.sh | OPENDRAY_PURGE=1 bash
```

</div>

### دستورات روزمره

بعد از نصب، باینری `opendray` لایف‌سایکل خودش را مدیریت می‌کند؛ دیگر لازم نیست incantationهای `systemctl` / `launchctl` را حفظ کنید:

<div dir="ltr">

```sh
sudo opendray update --restart   # download latest release, verify SHA, atomic replace + restart
```

</div>

<div dir="ltr">

```sh
sudo opendray providers update   # bump installed AI CLIs (claude / codex / gemini) to npm-latest
```

</div>

<div dir="ltr">

```sh
opendray providers list          # see which AI CLIs are installed + their versions
```

</div>

<div dir="ltr">

```sh
sudo opendray start              # start | stop | restart | status: wraps systemd / launchd
```

</div>

`opendray --help` فهرست کامل subcommandها را نشان می‌دهد.

### انتخاب روش راه‌اندازی

هر مسیر پشتیبانی‌شده امکان spawn کردن سشن، دسترسی AI-CLI، backupهای رمزنگاری‌شده، و API کامل integration را دارد. opendray یک gateway host-resident است؛ AI CLIها را از طریق PTY اجرا می‌کند و process state (`~/.claude`، ssh-agent، project files) را با آن‌ها share می‌کند. همین مدل با container isolationای که production Docker تحمیل می‌کند جور درنمی‌آید، بنابراین Docker برای v2.x مسیر پشتیبانی‌شده‌ای نیست.

| مسیر | مناسب برای | برو به |
|---|---|---|
| 📦 **نسخهٔ باینری آماده** | کافی است اجرا کنید؛ روی لینوکس و مک با هر ابزار مدیریت فرایند | [صفحه انتشارها](https://github.com/Opendray/opendray/releases) → ببینید [انتشار عملیاتی](#production-deploy) |
| 🐧 **systemd unit** | bare-metal / VM / LXC Linux box | [انتشار عملیاتی §A](#option-a--systemd-bare-metal--vm--lxc) |
| 🍎 **مک LaunchDaemon** | مک مینی / مک استودیو به‌عنوان سرور خانگی | [انتشار عملیاتی §C](#option-c--macos-launchd-mac-mini--studio-as-home-server) |
| 🛠 **ایجاد نسخه از کد منبع** | dev / مشارکت / نسخهٔ سفارشی | [راه‌اندازی سریع](#quickstart-5-minute-dev-path) در ادامه |

## Quickstart (5-minute dev path)

برای یک راهنمای گام به گام کامل همراه با پیش‌نیازها و عیب یابی، به [`docs/quickstart.md`](docs/quickstart.md) نگاه کنید. مسیر فشرده توسعه:

<div dir="ltr">

```bash
# 1. Have a Postgres 15+ running on 127.0.0.1:5432 with pgvector enabled
#    (apt install postgresql-16 postgresql-16-pgvector / brew install postgresql@16 pgvector).
#    Point [database].url at any other DSN if you'd rather use a remote PG.

# 2. Local config: already gitignored.
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

</div>

این opendray را در پیش‌زمینه اجرا می‌کند؛ Ctrl-C هم می‌بنددش. برای یک daemon که قرار است مدت زیادی بالا بماند، پایین‌تر به **انتشار عملیاتی** نگاه کنید.

<a id="production-deploy"></a>

## انتشار عملیاتی

چهار مسیر استقرار پشتیبانی‌شده داریم؛ هرکدام که به محیط شما می‌خورد همان را انتخاب کنید.
همه‌شان auto-restart هنگام crash، state پایدار، و جدا نگه‌داشتن secrets از config را فراهم می‌کنند.

<a id="option-a--systemd-bare-metal--vm--lxc"></a>

### گزینه ۱: systemd (bare-metal / VM / LXC)

مسیر پیشنهادی استقرار روی لینوکس. یک unit harden‌شده در
[`deploy/systemd/opendray.service`](deploy/systemd/opendray.service)
ارائه می‌کند با sandboxing (`ProtectSystem=strict`، `NoNewPrivileges`،
`MemoryDenyWriteExecute`، capability scrub)، بوت `migrate`-then-`serve`،
و یک graceful stop window بیست‌ثانیه‌ای.

**اول یک باینری بگیرید.** یا یک آرشیو از پیش ساخته را از
[صفحه انتشارها](https://github.com/Opendray/opendray/releases)
بردارید (`opendray_*_linux_<arch>.tar.gz`، که به یک `opendray` single-binary
unpack می‌شود)، یا از طریق [راه‌اندازی سریع](#quickstart-5-minute-dev-path)
ایجاد نسخه از کد منبع(`go build ./cmd/opendray`).

<div dir="ltr">

```bash
# 1. Install the binary you just grabbed (or built).
sudo install -m 0755 /path/to/opendray /usr/local/bin/opendray

# 2. Create the service user + state dir.
sudo useradd -r -s /usr/sbin/nologin -d /var/lib/opendray opendray
sudo install -d -o opendray -g opendray -m 0700 /var/lib/opendray

# 3. Drop config + secrets (root-owned; mode 0640).
sudo install -D -m 0640 config.example.toml /etc/opendray/config.toml
sudo $EDITOR /etc/opendray/config.toml             # set [database].url etc.
sudo install -D -m 0640 -o root -g opendray /dev/null /etc/opendray/env.d/secrets
sudo $EDITOR /etc/opendray/env.d/secrets           # OPENDRAY_ADMIN_PASSWORD=…

# 4. Install + enable the unit.
sudo cp deploy/systemd/opendray.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now opendray

# 5. Verify.
sudo systemctl status opendray
sudo journalctl -u opendray -f --no-pager
```

</div>

این unit `opendray migrate` را به‌عنوان `ExecStartPre` اجرا می‌کند، پس اولین boot همه migrationها را قبل از شروع `serve` اعمال می‌کند. restartها هم `on-failure` هستند با back-off پنج‌ثانیه‌ای و limit پنج‌بار در دقیقه.

### گزینه ۲: باینری مستقیم + ابزار مدیریت فرایند خودتان

برای LXC بدون systemd، FreeBSD `rc.d`، OpenRC، یا هر چیز دیگر.
یک بار build کنید و با هر ابزار مدیریت فرایندی که از قبل دارید اجرا کنید:

<div dir="ltr">

```bash
# Cross-compile a release archive locally:
goreleaser release --clean --snapshot
ls dist/                  # opendray_*_linux_amd64.tar.gz etc.

# Or grab a published release artefact:
# https://github.com/Opendray/opendray/releases
```

</div>

بعد ابزار مدیریت فرایند خودتان (s6، runit، supervisord، runwhen) را به این مسیر اشاره دهید:

<div dir="ltr">

```
/usr/local/bin/opendray serve -config /etc/opendray/config.toml
```

</div>

پیش از اولین `serve` یک بار `opendray migrate -config /etc/opendray/config.toml`
را اجرا کنید، یا آن را به‌عنوان pre-start hook در ابزار مدیریت فرایند دلخواهتان بگذارید.

<a id="option-c--macos-launchd-mac-mini--studio-as-home-server"></a>

### گزینه ۳: مک launchd (مک مینی / مک استودیو به‌عنوان سرور خانگی)

برای اپل سیلیکون مک مینی / مک استودیو که ۲۴/۷ روشن است. یک
LaunchDaemon در
[`deploy/launchd/com.opendray.opendray.plist`](deploy/launchd/com.opendray.opendray.plist)
ارائه می‌کند که در بوت و قبل از هر لاگین کاربر بالا می‌آید، هنگام کرش با
throttle پنج‌ثانیه‌ای دوباره start می‌شود، و در `/usr/local/var/log/opendray/`
لاگ می‌نویسد.

<div dir="ltr">

```bash
# 1. Install the darwin binary + config + state dirs.
sudo install -m 0755 ./opendray /usr/local/bin/opendray
sudo install -d -m 0755 \
  /usr/local/etc/opendray \
  /usr/local/var/lib/opendray \
  /usr/local/var/log/opendray
sudo install -m 0640 config.example.toml /usr/local/etc/opendray/config.toml
sudo $EDITOR /usr/local/etc/opendray/config.toml    # set [database].url etc.

# 2. Apply migrations once.
sudo /usr/local/bin/opendray migrate \
  -config /usr/local/etc/opendray/config.toml

# 3. Install + load the LaunchDaemon.
sudo cp deploy/launchd/com.opendray.opendray.plist /Library/LaunchDaemons/
sudo chown root:wheel /Library/LaunchDaemons/com.opendray.opendray.plist
sudo chmod 0644 /Library/LaunchDaemons/com.opendray.opendray.plist
sudo launchctl bootstrap system /Library/LaunchDaemons/com.opendray.opendray.plist

# 4. Verify.
sudo launchctl print system/com.opendray.opendray
tail -f /usr/local/var/log/opendray/opendray.log
```

</div>

با `sudo launchctl kickstart -k system/com.opendray.opendray` دوباره start کنید؛
با `sudo launchctl bootout system/com.opendray.opendray` هم کامل unload کنید.

Postgres روی مک: از طریق Homebrew نصبش کنید (`brew install postgresql@17 && brew services start postgresql@17`) و `[database].url` را روی
`postgres://$USER@127.0.0.1:5432/opendray` بگذارید. `pgvector` را هم با
`brew install pgvector` اضافه کنید و `CREATE EXTENSION vector` را داخل
دیتابیس opendray اجرا کنید.

---

برای نوت‌های مخصوص Proxmox LXC (PTY در containerهای unprivileged،
networking، تنظیمات cgroup)، به [`deploy/lxc/proxmox-pty-notes.md`](deploy/lxc/proxmox-pty-notes.md) نگاه کنید.

برای reverse-proxy / TLS termination (nginx، Caddy، Traefik، Cloudflare
Tunnel)، به [`docs/operator-guide.md`](docs/operator-guide.md) §Topology نگاه کنید.

### اختیاری: فعال‌سازی backupهای رمزنگاری‌شده DB و data exportها

<div dir="ltr">

```bash
# Master passphrase (env-only: never write into config.toml).
export OPENDRAY_BACKUP_KEY="$(openssl rand -base64 32)"
export OPENDRAY_BACKUP_ENABLED=1

# pg_dump / pg_restore must match the server's major version. On
# Apple Silicon dev machines pointing at a PG17 server:
export OPENDRAY_BACKUP_PG_DUMP_PATH=/opt/homebrew/opt/postgresql@17/bin/pg_dump
export OPENDRAY_BACKUP_PG_RESTORE_PATH=/opt/homebrew/opt/postgresql@17/bin/pg_restore
```

</div>

opendray را restart کنید؛ sidebar یک صفحه Backups (`/backups`) برای
PostgreSQL dumpهای رمزنگاری‌شده + restore و `/export` برای
zip-bundle data export + import اضافه می‌کند. برای چرخه کامل به
[`docs/operator-guide.md`](docs/operator-guide.md) §Backup نگاه کنید.

تمام رابط وب در یک باینری Go قرار گرفته است؛ بنابراین در زمان اجرا نیازی به
Node.js، سرور جداگانه برای فایل‌های استاتیک یا ابزارهایی مانند Caddy و Nginx
نخواهید داشت. Cloudflare Tunnel نیز اتصال TLS را پیش از رسیدن ترافیک به پورت `8770` مدیریت می‌کند.

## چیدمان

<div dir="ltr">

```
cmd/opendray/        نقطه ورود اصلی باینری (مطابق design §14 با حداکثر 100 خط کد)

internal/
├── app/             راه‌اندازی و اتصال همه بخش‌های برنامه
├── audit/           دریافت رویدادها از bus و ذخیره آن‌ها در audit_log
├── auth/            توکن‌های Bearer مدیر (M2.5)
├── backup/          پشتیبان‌گیری رمزنگاری‌شده از پایگاه داده و درون‌ریزی/برون‌بری داده‌ها
├── catalog/         مانیفست ارائه‌دهندگان CLI و تنظیمات هر کاربر
├── channel/         مدیریت کانال‌ها و یکپارچه‌سازی با تلگرام (M4)
├── config/          بارگذاری فایل‌های TOML و متغیرهای OPENDRAY_*
├── eventbus/        سیستم Pub/Sub داخلی
├── gateway/         مسیریابی HTTP با chi، middleware و slog
├── integration/     ثبت برنامه‌های خارجی، Reverse Proxy و رویدادهای WebSocket (M3)
├── memory/          حافظه پایدار مشترک بین CLIها
├── session/         مدیریت نشست‌ها، PTY، بافر حلقوی و جریان WebSocket (M1)
├── store/           اتصال به PostgreSQL با pgx و اجرای Migrationها (M0)
├── version/         اطلاعات نسخه و زمان ساخت
└── web/             جاسازی رابط وب با go:embed (W5)

app/web/             اپلیکیشن تک‌صفحه‌ای (SPA) مبتنی بر React 19، TypeScript و Vite (فاز 2، W0-W5)

app/mobile/          اپلیکیشن Flutter برای اندروید و iOS با قابلیت‌هایی مشابه نسخه وب

docs/
├── design.md        سند اصلی طراحی و مرجع پروژه (SSOT)
└── adr/             تصمیم‌های معماری ثبت‌شده به همراه تاریخ
```

</div>

## فرانت‌اند وب

`app/web/` یک SPA واحد را در `internal/web/dist/` build می‌کند، که باینری Go آن را embed می‌کند و در `/admin/*` سرو می‌کند. Vite dev server روی `:5173`
`/api` را به `:8770` proxy می‌کند تا development با HMR انجام شود.

<div dir="ltr">

```bash
# dev (hot reload on the React side, separate Go server for the API)
cd app/web && pnpm dev               # http://localhost:5173
go run ./cmd/opendray serve -config ../../config.toml   # other terminal

# prod (one binary delivers everything)
cd app/web && pnpm build              # writes ../../internal/web/dist
cd ../..
go build ./cmd/opendray               # bakes dist into the binary
./opendray serve -config config.toml
```

</div>

برای frontend stack
(`React` + `Vite` + `Tailwind v4` + `shadcn/ui` + `TanStack Router/Query` +
`Zustand` + `xterm.js`) و یادداشت‌های milestone هر W به
[`app/web/README.md`](app/web/README.md) نگاه کنید.

## مستندات

- [`docs/getting-started.md`](docs/getting-started.md): **اگر تازه‌کارید، از اینجا شروع کنید**؛ از صفر تا اولین سشن در ۱۵ دقیقه، شامل نصب CLIهای دربرگرفته‌شده و bootstrap کردن Postgres
- [`docs/quickstart.md`](docs/quickstart.md): محیط dev پنج‌دقیقه‌ای (فرض می‌کند از قبل moving parts را می‌شناسید)
- [`docs/operator-guide.md`](docs/operator-guide.md): مرجع deploy + ops برای setupهای production-ish
- [`docs/integration-guide.md`](docs/integration-guide.md): چطور یک integration خارجی را در هر زبانی بنویسید
- [`VERSIONING.md`](VERSIONING.md): استراتژی versioning (major-as-generation)
- [`CHANGELOG.md`](CHANGELOG.md): تاریخچه releaseها

## تست‌ها

<div dir="ltr">

```bash
go test -race ./...        # backend
cd app/web && pnpm build   # web (TS strict + vite production build)
```

</div>

smoke flowهای سرتاسری را در commit messageها بر اساس هر milestone track می‌کنیم.
یک Playwright harness هم به‌عنوان follow-up برنامه‌ریزی شده است.

## نسبت به v1

v1 (`Opendray/opendray`) codebase قدیمی است که حالا archived شده. v2 نسل
فعلی و active است؛ feature-complete و تنها branchی که development می‌گیرد.
از 16 builtin در v1، چهار مورد به بک‌اند v2 منتقل شدند؛ بقیه رفتند سمت
featureهای client-side، channel adapterها، یا integration-API consumerها.

## مجوز

آپاچی ۲.۰: به [`لایسنس`](LICENSE) نگاه کنید. (v1 تحت مجوز MIT بود؛ v2 مجوز جداگانهٔ خودش را دارد.)

</div>