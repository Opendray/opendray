<div dir="rtl">

# شروع کار

راهنمای گام‌به‌گام از صفر تا اولین جلسه. اگر PostgreSQL از قبل روی میزبان نصب شده باشد، حدود ۱۵ دقیقه زمان بگذارید؛ در غیر این صورت حدود ۲۵ دقیقه طول می‌کشد.

این راهنما به صورت انتها-به-انتها نوشته شده — مواردی را پوشش می‌دهد که کنار opendray قرار دارند (نصب CLIهایی که opendray آن‌ها را اجرا می‌کند، آماده‌سازی Postgres) و مسیرهای استقرار را در [README](../README.md#install) پی‌می‌گیرد. اگر قبلاً از opendray استفاده کرده‌اید و فقط می‌خواهید مجدداً اجرا کنید، مسیرهای خلاصه در بخش «Production deploy» در README مناسب‌تر هستند.

> **قبلاً می‌دانید opendray مناسب شما هست یا نه؟**
> اگر نمی‌دانید، ابتدا بخش «What is opendray?» در [README](../README.md#what-is-opendray) را بخوانید — گلوله‌های آن ممکن است ۱۵ دقیقه از وقت شما را صرفه‌جویی کند اگر مورد استفادهٔ شما متناسب نباشد.

---

## گام ۰ — چه چیزهایی نیاز دارید

| ابزار | چرا | توضیح |
|---|---|---|
| حداقل یکی از موارد: Claude Code / Codex CLI / Gemini CLI | opendray یک «بسته‌کننده» است، نه مدل — آن‌ها یک CLI را روی میزبان شما اجرا می‌کند | گام ۱ پایین‌تر |
| PostgreSQL 15 / 16 / 17 + `pgvector` extension | نگهداری وضعیت، جلسات، و بردارهای حافظه | گام ۲ پایین‌تر |
| `go` 1.25+ و `pnpm` 10+ — تنها در صورتی که از سورس بسازید | اگر باینری انتشار را دانلود می‌کنید، لازم نیست | [صفحه انتشار ها](https://github.com/Opendray/opendray/releases) |
| یک پورت شبکه قابل دسترس (پیش‌فرض `:8770`) برای مدیریت وب | رابط کاربری + API + WebSocketها | در صورت استفاده از پروکسی معکوس، به `127.0.0.1` محدود کنید |

---

## گام ۱ — نصب حداقل یک AI CLI

ابزار opendray این CLIها را با استفاده از حساب‌های لوکال شما اجرا می‌کند. این ابزارها را همان‌طور که برای استفاده در ترمینال نصب می‌کنید، نصب کنید؛ opendray باینری‌ها را در متغیر محیطی `PATH` پیدا می‌کند.

### Claude Code (نقطهٔ شروع پیشنهادی)

```sh
npm install -g @anthropic-ai/claude-code
claude login        # احراز هویت از طریق مرورگر
```

بعد از لاگین، اطلاعات اعتباری در `~/.claude/credentials.json` قرار می‌گیرد.
ابزار opendray این فایل را به‌صورت خودکار هنگام انتخاب ارائه‌دهندهٔ **claude** می‌خواند.

### Codex CLI (OpenAI)

```sh
# راهنمایی در https://github.com/openai/codex
# بستهٔ دقیق npm یا pip بسته به نسخه متفاوت است؛ چیزی که نصب کنید باید `codex` را در PATH قرار دهد.
codex --version     # بررسی سالم بودن
```

### Gemini CLI (Google)

```sh
npm install -g @google/gemini-cli
gemini auth login
```

### بررسی اینکه حداقل یکی قابل دسترس است

```sh
which claude codex gemini      # حداقل باید یک خط نتیجه بدهد
```

> می‌توانید opendray را با تنها یک CLI نصب‌شده اجرا کنید و بعداً بقیه را اضافه کنید. فهرست ارائه‌دهنده‌ها پویا است — opendray در زمان اجرا باینری را پروب می‌کند و موارد گمشده در پنل Sessions به‌عنوان «executable file not found in $PATH» نمایش داده می‌شوند.

---

## گام ۲ — نصب Postgres + pgvector

ابزار opendray به PostgreSQL نسخهٔ **۱۵، ۱۶، یا ۱۷** به‌همراه افزونهٔ [`pgvector`](https://github.com/pgvector/pgvector) نیاز دارد. روش نصب را بر اساس توزیع یا سیستم‌عامل خود انتخاب کنید.

### macOS (Homebrew)

```sh
brew install postgresql@17 pgvector
brew services start postgresql@17
```

### اوبونتو / دبیان

```sh
sudo apt install postgresql-17 postgresql-17-pgvector
sudo systemctl enable --now postgresql
```

### سایر لینوکس‌ها

بسته‌های PostgreSQL توزیع خود را نصب کنید، سپس یا pkg مربوط به pgvector را نصب کنید یا از منبع [ساخت](https://github.com/pgvector/pgvector#installation) کنید.

### آماده‌سازی پایگاه‌دادهٔ opendray (یک‌بار)

در `psql` متصل به‌عنوان سوپربِوزر:

```sql
-- محلی (پیش‌فرض Homebrew): `psql postgres`
-- ریموت: `psql -h <host> -U postgres -d postgres`

CREATE DATABASE opendray;
CREATE USER opendray_user WITH ENCRYPTED PASSWORD '<یک گذرواژهٔ قوی انتخاب کنید>';
GRANT ALL PRIVILEGES ON DATABASE opendray TO opendray_user;

\c opendray
CREATE EXTENSION IF NOT EXISTS vector;
GRANT ALL ON SCHEMA public TO opendray_user;
```

> دستور `CREATE EXTENSION vector` نیاز به دسترسی سوپربِوزر دارد. پس از اجرا، `opendray_user` فقط به امتیازات CRUD نیاز دارد — opendray در زمان اجرا هرگز به‌عنوان سوپربِوزر مجدداً وصل نمی‌شود.

برای آزمایش اعتبارنامه‌ها از میزانی که opendray قرار است اجرا شود، دستور زیر را اجرا کنید:

```sh
PGPASSWORD='<password>' psql -h <pg-host> -U opendray_user -d opendray -c "SELECT 'ok' AS check;"
```

باید خروجی `check: ok` را بدون خطا ببینید.

---

## گام ۳ — انتخاب مسیر استقرار و نصب opendray

پرسش تصمیم‌گیرنده: آیا برای قابلیت «spawn session» آماده می‌شوید (کنترل Claude / Codex / Gemini از طریق صفحهٔ Sessions وب)؟

### اگر پاسخ بله است — مسیر «Full» لازم دارید

| میزبان شما | مسیر | بخش README |
|---|---|---|
| macOS به‌عنوان سرور ۲۴/۷ | macOS LaunchDaemon | [Option D](../README.md#option-d--macos-launchd-mac-mini--studio-as-home-server) |
| سرور/VM/LXC لینوکس | systemd | [Option B](../README.md#option-b--systemd-bare-metal--vm--lxc) |
| فقط آزمایشی در foreground | `go run` از سورس | [Quickstart](../README.md#quickstart-5-minute-dev-path) |
| ناظر روند دستی (s6 / runit / launchd Agent) | باینری مستقیم | [Option C](../README.md#option-c--direct-binary--your-own-process-supervisor) |

> از Docker بپرهیزید. ایمیج distroless است (نود، AI CLIها و `pg_dump` در آن نیست)؛ بنابراین تب Sessions در هر کلیک spawn خطا خواهد داد. دلیل معماری در بخش §A توضیح داده شده است.

### اگر پاسخ خیر است — تنها به کانال‌ها / انتگراسیون‌ها / یادداشت‌ها / API نیاز دارید

شما همچنان می‌توانید پیام‌ها را روی Telegram / Slack و دیگر کانال‌ها دریافت کنید، یادداشت بنویسید، از API استفاده کنید و مدیریت وب را ببینید؛ فقط نمی‌توانید از این استقرار برای spawn sessionهای لوکال استفاده کنید.

تمام مسیرها به هم می‌رسند روی:

```sh
# پیکربندی نمونه‌ای که gateway آن را می‌خواند
cp config.example.toml config.toml
$EDITOR config.toml            # مقدارهای [database].url و [admin].password را تنظیم کنید

# یک‌بار: ساخت شِما (idempotent در اجرای مجدد)
opendray migrate -config config.toml

# اجرای gateway
opendray serve -config config.toml
```

دو فیلد حداقلی در `config.toml`:

```toml
[database]
url = "postgres://opendray_user:<password>@<host>:5432/opendray?sslmode=disable"

[admin]
password = "<initial-bootstrap-password>"
```

سایر تنظیمات مقدارهای پیش‌فرض مناسبی دارند — برای مشاهدهٔ کامل سطح تنظیمات، نظرات داخلی در `config.example.toml` را ببینید.

---

## گام ۴ — اولین ورود + تغییر گذرواژهٔ admin

آدرس `http://localhost:8770/admin/` (یا هر host:port که در `config.toml` تنظیم کرده‌اید) را باز کنید.

1. با نام کاربری `admin` و گذرواژه‌ای که در `[admin].password` گذاشته‌اید وارد شوید.
2. بلافاصله به Settings → Admin → Change password بروید.

چرا بلافاصله: بعد از اولین تغییر گذرواژه، opendray یک فایل keyfile حاوی هش bcrypt را در `$HOME/.opendray/secrets/admin.key` می‌نویسد و مقدار متنی `[admin].password` در `config.toml` بی‌اثر می‌شود (keyfile ارجح است). تا قبل از تغییر، تنها حفاظتی که دارید مجوزهای فایل سیستم روی `config.toml` است.

زنجیرهٔ ارجحیت اعتبارنامه‌ها در [operator-guide §admin](operator-guide.md#admin) توضیح داده شده است.

---

## گام ۵ — پیکربندی یک Provider

Providers → روی ارائه‌دهنده‌ای که در گام ۱ نصب کرده‌اید کلیک کنید → موارد زیر را پر کنید:

- **Command path** — مسیر مطلق به باینری CLI (با `which claude` پیدا می‌شود؛ روی Apple Silicon، نصب‌های Homebrew معمولاً در `/opt/homebrew/bin/claude` قرار می‌گیرند).
- **Accounts dir** (فقط Claude، اختیاری) — دایرکتوری مجموعهٔ اعتبارنامه‌های نام‌گذاری‌شدهٔ Claude اگر می‌خواهید به‌ازای هر جلسه هویت متفاوت انتخاب کنید. خالی بگذارید تا از `~/.claude` پیش‌فرض استفاده شود.

ذخیره کنید. opendray یک بار `<cli> --version` اجرا می‌کند تا پروب انجام شود؛ کارت ارائه‌دهنده وقتی باینری قابل دسترس باشد به رنگ سبز درمی‌آید.

---

## گام ۶ — ایجاد اولین جلسه (spawn)

Sessions → New session → ارائه‌دهنده را انتخاب کنید → یک دایرکتوری کاری انتخاب کنید (هر پروژه‌ای روی ماشین شما) → Spawn کنید.

یک ترمینال سمتِ مرورگر باز می‌شود. مثل ترمینال واقعی پرامپت‌ها را تایپ کنید. تب را ببندید؛ جلسه روی میزبان به‌کارش ادامه می‌دهد و هنگام بازگشت، scrollback حفظ شده است.

---

## گام ۷ (اختیاری) — افزودن یک کانال Telegram

این ویژگی تفاوت opendray را با `tmux` + `ssh` نشان می‌دهد. با اتصال یک کانال، opendray وقتی جلسه idle می‌شود (CLI منتظر ورودی است) نوتیفیکیشن می‌فرستد و پاسخ شما در Telegram به‌عنوان ورودی بعدی stdin نوشته می‌شود.

### راه‌اندازی یک‌بارۀ Telegram

1. در Telegram دنبال **@BotFather** بگردید و چت را شروع کنید.
2. `/newbot` → BotFather شما را برای نام و username راهنمایی می‌کند و توکنی شبیه `123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11` صادر می‌کند.
3. شناسهٔ چت خود را پیدا کنید:
   - به‌صورت مستقیم به بات پیام بدهید (هر متن).
   - `https://api.telegram.org/bot<token>/getUpdates` را در مرورگر باز کنید — `chat.id` عددی در پاسخ JSON دیده می‌شود.

### در opendray

Channels → New channel → نوع **Telegram**:

- **Bot token**: از BotFather
- **Default chat ID**: `chat.id` از `getUpdates`
- **Notify on**: تیک `session.idle` را بزنید (یا هر سه موضوع را انتخاب کنید)

ذخیره کنید → روی کارت کانال «Test» را بزنید. ظرف چند ثانیه پیام آزمایشی را در Telegram دریافت خواهید کرد.

حالا یک جلسه را ۳۰ ثانیه (آستانۀ پیش‌فرض idle — قابل تنظیم در `[session].idle_threshold`) بدون فعالیت رها کنید؛ Telegram پیامی با آخرین خروجی CLI برایتان می‌فرستد. پاسخ دهید؛ متن به stdin جلسه نوشته می‌شود.

---

## قدم بعدی؟

- **کانال‌های بیشتر**: Slack / Discord / Feishu (飞书) / DingTalk (钉钉) / WeCom (企业微信) — هرکدام تنظیمات مخصوص خود را در آموزش درون‌برنامه‌ای در `/admin/tutorial/` دارند.
- **انتگراسیون‌های API**: [docs/integration-guide.md](integration-guide.md) — کلیدهای API با محدوده، mount پروکسی معکوس، و WebSocket رویدادها.
- **زیرسیستم حافظه**: برای بردارهای محلی ابتدا `"[memory.backend] = \"local\""` را فعال کنید یا Ollama / LM Studio را متصل کنید — آموزش درون‌برنامه → بخش Memory را ببینید.
- **پشتیبان‌گیری رمزنگاری‌شده**: `[backup]` را پیکربندی کنید تا dumpهای DB را به S3 / R2 / B2 / SFTP / rclone بفرستد — بخش [operator-guide §backup](operator-guide.md#backup) را ملاحظه کنید.

## رفع‌اشکال

| نشانه | دلیل | رفع |
|---|---|---|
| `relation "providers" does not exist` هنگام migrate | باینری قدیمی‌تر از v2.0.0 (issue #162) | آخرین باینری را بگیرید — اصلاح در v2.0.0 آمده است |
| `type "vector" does not exist` هنگام migrate | افزونهٔ pgvector در پایگاه‌دادهٔ opendray فعال نشده است | در دیتابیس `opendray` به‌عنوان سوپربِوزر `CREATE EXTENSION vector;` را اجرا کنید |
| `Spawn session failed: executable file not found in $PATH` | CLI بسته‌بندی‌شده روی میزبان opendray نصب نیست، یا Command Path در تنظیمات Provider اشتباه است | گام ۱ را ببینید؛ با `which claude` (یا هر CLI) بررسی کنید |
| بات Telegram به پاسخ‌ها جواب نمی‌دهد | حالت privacy بات به‌صورت پیش‌فرض روشن است (بات فقط دستورات را می‌بیند) | از BotFather با `/setprivacy` حالت privacy را غیرفعال کنید |
| `Bad gateway` از طریق پروکسی معکوس | پروکسی هدرهای ارتقاء WebSocket را فوروارد نمی‌کند | برای نمونه‌های nginx / Caddy به [operator-guide §Topology](operator-guide.md#topology) مراجعه کنید |
| تب Sessions خالی است اما کانال‌ها کار می‌کنند | به‌احتمال زیاد باینری قابل اجرا هست اما Provider پیکربندی نشده است | گام ۵ را بررسی کنید |

---

## منابع

- [README](../README.md) — جدول نصب، مسیرهای استقرار، وضعیت پروژه
- [README.zh.md](../README.zh.md) — نسخهٔ چینی ساده‌شده
- [docs/quickstart.md](quickstart.md) — محیط توسعهٔ ۵ دقیقه‌ای (متمرکزتر از این راهنما)
- [docs/operator-guide.md](operator-guide.md) — راهنمای اپراتور: توپولوژی، احراز هویت، پشتیبان، لاگینگ
- [docs/integration-guide.md](integration-guide.md) — سطح API انتگراسیون‌های طرف‌سوم
- [VERSIONING.md](../VERSIONING.md) — نسخه‌بندی major-as-generation
- [CHANGELOG.md](../CHANGELOG.md) — تاریخچهٔ انتشار

</div>
