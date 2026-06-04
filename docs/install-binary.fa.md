<div dir="rtl" lang="fa">

# نصب و اجرا از باینری از‌پیش‌ساخته‌شده

برای وقتی که فقط باینری `opendray` را می‌خواهید — بدون wizard نصب‌کننده که دست به تنظیمات ماشین شما بزند. این مسیر مناسب است برای:

- **`npm install -g opendray` / `npx opendray`** — پکیج npm باینری رسمی Go release را شامل می‌شود (ببینید [README → npm / npx](../README.fa.md)).
- **دانلود Release** — فایل `opendray_*_<os>_<arch>.tar.gz` را از [صفحه Releases](https://github.com/Opendray/opendray/releases) بردارید.
- **محیط‌های scripted / ephemeral** — CI runnerها، golden imageها، config management (Ansible، Nix، Docker)، یا هر hostی که Postgres و process supervisor خودتان را دارید.

باینری همان *کل* gateway است — web admin SPA داخلش embed شده، پس نه Node runtime لازم است، نه static server جداگانه، و نه چیزی برای build کردن. آنچه **نمی‌کند** این است که چیزی را برای شما setup نکند. این همان معامله است: شما یک دیتابیس PostgreSQL و روشی برای نگه‌داشتن فرایند می‌آورید، و در عوض هیچ‌چیزی بدون اطلاع شما نصب، تنظیم، یا ثبت نمی‌شود.

> **می‌خواهید همه‌چیز برایتان انجام شود؟** روی یک Linux / macOS تازه، installer تک‌خطی Postgres را provision می‌کند، AI CLIها را نصب می‌کند، config را می‌نویسد، و یک service را در ۵ تا ۱۰ دقیقه ثبت می‌کند. ببینید [README → نصب‌کننده تک‌خطی](../README.fa.md) یا راهنمای دستی [getting-started.fa.md](getting-started.fa.md).

این راهنما شما را از «باینری روی PATH» به «gateway در حال اجرا» در پنج گام می‌رساند، سپس نشان می‌دهد چطور آن را به‌صورت یک service اجرا کنید.

---

## گام ۱ — باینری را بگیرید

### از طریق npm (هر OS با Node ≥ 18)

<div dir="ltr">

```sh
npm install -g opendray        # نصب global، `opendray` را به PATH اضافه می‌کند
# یا بدون نصب:
npx opendray --help
# یا با package manager دیگر:
pnpm add -g opendray
yarn global add opendray
```

</div>

باینری پلتفرم متناظر (`opendray-{linux,darwin}-{x64,arm64}`) به‌صورت خودکار از طریق `optionalDependencies` انتخاب می‌شود — هیچ `postinstall` hookی وجود ندارد و هیچ network call هنگام نصب زده نمی‌شود. از دادن `--no-optional` **خودداری کنید**: این پکیج پلتفرم را رد می‌کند و launcher را بدون باینری‌ای که exec کند رها می‌کند.

### از طریق آرشیو release

<div dir="ltr">

```sh
# آرشیو متناسب با OS/arch خود را از صفحه Releases انتخاب کنید، سپس:
tar -xzf opendray_*_linux_amd64.tar.gz
sudo install -m 0755 opendray /usr/local/bin/opendray
```

</div>

### تأیید

<div dir="ltr">

```sh
opendray version          # نسخه، commit، تاریخ build را نشان می‌دهد
opendray --help           # همه subcommandها را فهرست می‌کند
```

</div>

پلتفرم‌های پشتیبانی‌شده: **Linux** (x64, arm64) و **macOS** (x64, arm64).
Windows بومی بسته‌بندی نشده — از WSL2 استفاده کنید و مسیر Linux را دنبال کنید.

---

## گام ۲ — PostgreSQL 15+ با pgvector را فراهم کنید

opendray همه‌چیز (سشن‌ها، حافظه، audit log) را در PostgreSQL ذخیره می‌کند، و زیرسیستم حافظه‌اش به extension [`pgvector`](https://github.com/pgvector/pgvector) نیاز دارد. نسخه‌های server پشتیبانی‌شده: **15، 16، 17**.

اگر از قبل Postgres دارید، یک دیتابیس و یک نقش CRUD-only بسازید، سپس extension را یک بار با superuser فعال کنید:

<div dir="ltr">

```sh
# 1. نصب pgvector (یک بار در هر host).
#    Ubuntu/Debian:  sudo apt install postgresql-16-pgvector
#    macOS (brew):   brew install pgvector
#    سایر / از source: https://github.com/pgvector/pgvector#installation

# 2. ساختن دیتابیس + یک نقش scoped به پروژه.
sudo -u postgres psql <<'SQL'
CREATE DATABASE opendray;
CREATE USER opendray WITH ENCRYPTED PASSWORD 'change-me';
GRANT ALL PRIVILEGES ON DATABASE opendray TO opendray;
SQL

# 3. فعال‌سازی pgvector داخل آن دیتابیس (یک بار، نیاز به superuser دارد).
sudo -u postgres psql -d opendray -c 'CREATE EXTENSION IF NOT EXISTS vector;'
sudo -u postgres psql -d opendray -c 'GRANT ALL ON SCHEMA public TO opendray;'
```

</div>

وقتی extension موجود باشد، نقش CRUD-only opendray بدون هیچ دسترسی superuser بیشتری migration اجرا می‌کند. **هرگز opendray را به یک نقش superuser در زمان اجرا وصل نکنید** — به آن یک حساب scoped به پروژه بدهید و رمز عبورش را جداگانه rotate کنید.

---

## گام ۳ — تنظیم کنید

opendray config خود را از یک فایل TOML **یا** صرفاً از متغیرهای محیط (12-factor) می‌خواند — env همیشه بر فایل ارجحیت دارد. تنها نیاز اجباری database URL است؛ بقیه مقادیر پیش‌فرض دارند.

### گزینه الف — متغیرهای محیط (مناسب برای containerها / hostهای ephemeral)

<div dir="ltr">

```sh
export OPENDRAY_DATABASE_URL="postgres://opendray:change-me@127.0.0.1:5432/opendray?sslmode=disable"
export OPENDRAY_ADMIN_PASSWORD="$(openssl rand -base64 24)"   # رمز ورود به ادمین
export OPENDRAY_LISTEN="127.0.0.1:8770"                       # اختیاری؛ این مقدار پیش‌فرض است
```

</div>

| متغیر | اجباری | پیش‌فرض | هدف |
|---|---|---|---|
| `OPENDRAY_DATABASE_URL` | **بله** | — | Postgres DSN |
| `OPENDRAY_ADMIN_PASSWORD` | توصیه می‌شود | — | رمز ادمین وب/موبایل |
| `OPENDRAY_ADMIN_USER` | خیر | `admin` | نام کاربری ادمین |
| `OPENDRAY_LISTEN` | خیر | `127.0.0.1:8770` | آدرس bind |
| `OPENDRAY_LOG_LEVEL` | خیر | `info` | `debug`/`info`/`warn`/`error` |
| `OPENDRAY_LOG_FORMAT` | خیر | `text` | `text`/`json` |

`opendray serve` را بدون flag مربوط به `-config` اجرا کنید و کاملاً از محیط بارگذاری می‌شود.

### گزینه ب — config.toml

<div dir="ltr">

```sh
curl -fsSLO https://raw.githubusercontent.com/Opendray/opendray/main/config.example.toml
mv config.example.toml config.toml
$EDITOR config.toml        # [database].url و [admin].password را تنظیم کنید
```

</div>

حداقل چیزی که باید ویرایش کنید:

<div dir="ltr">

```toml
listen = "127.0.0.1:8770"

[database]
url = "postgres://opendray:change-me@127.0.0.1:5432/opendray?sslmode=disable"

[admin]
user     = "admin"
password = "use-a-real-password"
```

</div>

فایل کاملاً annotated را در [`config.example.toml`](../config.example.toml) ببینید (logging، session idle detection، backupها، vault، MCP). آن را با `-config config.toml` به دستورات زیر بدهید. secretها را روی hostهای shared داخل TOML نگذارید — `OPENDRAY_DATABASE_URL` / `OPENDRAY_ADMIN_PASSWORD` را از طریق env تنظیم کنید و فایل را non-secret نگه دارید.

---

## گام ۴ — schema را اعمال کنید

<div dir="ltr">

```sh
opendray migrate                          # config فقط از env
# یا
opendray migrate -config config.toml
```

</div>

Idempotent است — اجرای مجدد وقتی schema جاری است no-op می‌شود. این باید قبل از اولین `serve` با موفقیت اجرا شود.

---

## گام ۵ — اجرا کنید

<div dir="ltr">

```sh
opendray serve                            # config فقط از env
# یا
opendray serve -config config.toml
```

</div>

این در **foreground** اجرا می‌شود (Ctrl-C آن را متوقف می‌کند). حالا باید داشته باشید:

| آدرس | کاربرد |
|---|---|
| `http://127.0.0.1:8770/admin/` | ادمین وب — با `admin` + رمز عبورتان وارد شوید |
| `http://127.0.0.1:8770/api/v1/...` | REST + WebSocket API |

این یک gateway کامل و در حال اجرا است. برای هر چیزی فراتر از یک تست سریع، آن را زیر یک supervisor اجرا کنید تا از reboots جان سالم به در ببرد و در صورت crash دوباره راه‌اندازی شود — ادامه می‌دهیم.

---

## اجرا به‌عنوان service

`opendray serve` دقیقاً همان دستوری است که start command یک service unit باید فراخوانی کند.
opendray unitهای hardened و آماده‌به‌استفاده ارائه می‌کند؛ مراحل زیر همان [README → انتشار عملیاتی](../README.fa.md#production-deploy) است که مرجع اصلی است (bootstrap کامل، نکات sandboxing، reverse-proxy/TLS).

### Linux — systemd

این repo یک unit hardened در
[`deploy/systemd/opendray.service`](../deploy/systemd/opendray.service)
ارائه می‌کند
(اجرای `migrate` به‌عنوان `ExecStartPre`، secretها از طریق `EnvironmentFile`،
restart با `on-failure`، syscall/filesystem sandboxing).

<div dir="ltr">

```sh
# باینری در /usr/local/bin/opendray، service user، state dir:
sudo useradd -r -s /usr/sbin/nologin -d /var/lib/opendray opendray
sudo install -d -o opendray -g opendray -m 0700 /var/lib/opendray

# فایل config (non-secret) + فایل secretها (env، mode 0640):
sudo install -D -m 0640 config.toml /etc/opendray/config.toml
sudo install -D -m 0640 -o root -g opendray /dev/null /etc/opendray/env.d/secrets
echo 'OPENDRAY_ADMIN_PASSWORD=use-a-real-password' | sudo tee -a /etc/opendray/env.d/secrets

# نصب + فعال‌سازی unit:
sudo cp deploy/systemd/opendray.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now opendray
sudo systemctl status opendray
journalctl -u opendray -f --no-pager
```

</div>

systemd ندارید؟ (LXC بدون آن، OpenRC، runit، s6، supervisord…) supervisor خودتان را به `opendray serve -config /etc/opendray/config.toml` اشاره دهید و `opendray migrate` را یک بار به‌عنوان pre-start step اجرا کنید. ببینید
[README → انتشار عملیاتی §B](../README.fa.md#production-deploy).

### macOS — launchd

این repo یک LaunchDaemon در
[`deploy/launchd/com.opendray.opendray.plist`](../deploy/launchd/com.opendray.opendray.plist)
ارائه می‌کند
(در boot شروع می‌شود، در صورت crash دوباره راه‌اندازی می‌شود، در `/usr/local/var/log/opendray/` لاگ می‌نویسد).

<div dir="ltr">

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

</div>

Restart: `sudo launchctl kickstart -k system/com.opendray.opendray`.
Unload: `sudo launchctl bootout system/com.opendray.opendray`.

> هر دو unit به‌طور کامل — شامل چیدمان secretها و دلیل خاموش گذاشتن `MemoryDenyWriteExecute` — در [`deploy/README.md`](../deploy/README.md) مستند شده‌اند.

---

## به‌روز نگه داشتن

نحوه به‌روزرسانی به روش نصب بستگی دارد:

- **از طریق npm نصب شده** — با package manager خود به‌روز کنید. `opendray update`
  باینری را *داخل* `node_modules` پشت npm جایگزین می‌کند و با اجرای بعدی install بازنویسی می‌شود؛ پس اینجا از آن استفاده نکنید.

<div dir="ltr">

  ```sh
  npm install -g opendray@latest
  ```

</div>

- **دانلود release / نصب wizard** — باینری خودش را in-place به‌روز می‌کند
  (آخرین release را دانلود می‌کند، SHA-256 آن را تأیید می‌کند، به‌صورت atomic خودش را swap می‌کند):

<div dir="ltr">

  ```sh
  opendray update --check          # فقط گزارش نسخه
  sudo opendray update --restart   # اعمال، سپس راه‌اندازی مجدد service
  ```

</div>

---

## عیب‌یابی

**`the matching platform package "opendray-…" was not installed`**
npm با `--no-optional` اجرا شده، یا نصب قطع شده است. `npm install -g opendray` را دوباره اجرا کنید (بدون `--no-optional`).

**`unsupported platform`**
پکیج npm فقط Linux/macOS روی x64/arm64 را پوشش می‌دهد. روی targetهای دیگر، از source بسازید — ببینید [quickstart.md](quickstart.md).

**`config: database.url is empty`**
نه `OPENDRAY_DATABASE_URL` و نه `[database].url` تنظیم نشده‌اند. یکی را تنظیم کنید (گام ۳).

**`connection refused` هنگام migrate/serve**
Postgres اجرا نمی‌شود یا DSN اشتباه است. مطمئن شوید server up است و host/port/credentials داخل DSN شما درست هستند.

**pgvector / `extension "vector" is not available`**
Extension روی server نصب نشده، یا داخل دیتابیس opendray فعال نشده است. گام ۲ را دوباره انجام دهید (پکیج OS را نصب کنید، سپس `CREATE EXTENSION vector` را به‌عنوان superuser اجرا کنید).

**Port در حال استفاده است**
`OPENDRAY_LISTEN` (یا `listen` در config.toml) را به یک port آزاد تغییر دهید.

---

## گام‌های بعدی

- [README → انتشار عملیاتی](../README.fa.md#production-deploy) — مرجع کامل deploy (systemd / launchd / own-supervisor، hardening، reverse proxy)
- [`docs/operator-guide.md`](operator-guide.md) — ops: توپولوژی reverse-proxy/TLS، backupهای رمزنگاری‌شده DB، data export/import
- [`docs/integration-guide.md`](integration-guide.md) — ساختن یک integration خارجی با REST + WebSocket API
- [`docs/getting-started.fa.md`](getting-started.fa.md) — راه‌اندازی راهنما و all-in-one اگر ترجیح می‌دهید قطعات را خودتان سر هم نکنید

</div>
