<p align="center">
  <a href="https://opendray.dev"><img src="docs/assets/logo.png" alt="opendray" width="180"></a>
</p>

<h1 align="center">opendray</h1>

<p align="center">
  <strong>Self-hosted шлюз для Claude Code · Codex · Gemini · shell — с единым local-first слоем памяти, общим для всех инструментов.</strong>
  <br/>
  <sub>Запускайте сессии на собственной инфраструктуре. Управляйте из веба, с мобильного или из чата. Открытый REST + WebSocket API для интеграций.</sub>
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
  🌐 <a href="README.md">English</a> · <a href="README.zh.md">简体中文</a> · <a href="README.fa.md">فارسی</a> · <a href="README.es.md">Español</a> · <a href="README.pt-BR.md">Português</a> · <a href="README.ja.md">日本語</a> · <a href="README.ko.md">한국어</a> · <a href="README.fr.md">Français</a> · <a href="README.de.md">Deutsch</a> · <strong>Русский</strong>
</p>

---

## Зачем нужен opendray

Три ежедневных боли при работе с AI coding CLI, которые opendray призван закрыть.

**Сессии умирают, когда ноутбук уходит в сон.** Запускаете Claude Code или Codex по SSH, и агент умирает в тот же момент, когда вы закрываете крышку или теряете Wi-Fi. Контекст, незавершённые tool calls, тот самый частичный diff, который вы как раз собирались отсмотреть. Всё пропадает. opendray запускает агента на хосте, который не уходит в сон (Mac mini под столом, NAS, VPS), и позволяет переподключиться к сессии из веб-админки, мобильного приложения на Flutter или прямо из чата. Сессия продолжает выполняться, подключён к ней кто-нибудь или нет.

**Упереться в rate limit не должно убивать то, чем вы занимались.** Если у вас несколько аккаунтов Anthropic (рабочий и личный, семейный план и Pro), opendray воспринимает их как пул: показывает тариф, квоту и количество активных сессий по каждому аккаунту, балансирует новые сессии между ними и позволяет переключить живую сессию на другой аккаунт, не теряя диалог. Транскрипт переезжает вместе с вами. То же самое работает для аккаунтов Codex и Gemini.

**Память это полноценный слой, а не довесок.** Большинство AI CLI каждый раз заново индексируют контекст проекта с нуля, сжигая токены на повторных выборках. opendray приносит локальный векторный store (эмбеддинги через ONNX / Ollama / LM Studio) с трёхдоменным поиском (пользователь, проект, сессия), плюс детектирование расхождений между слоями. Каждый байт остаётся в вашей сети.

---

## Что такое opendray?

**opendray** оборачивает AI-CLI для кодинга, которыми вы уже пользуетесь — Claude Code, Codex, Gemini, плюс любой shell — и превращает их в то, чем можно управлять откуда угодно. Запускайте сессии на домашнем сервере / NAS / VPS, получайте уведомление в Telegram, когда сессия простаивает, отвечайте с телефона, чтобы скормить следующий промпт обратно — всё через self-hosted шлюз, который вы контролируете от и до.

- 🛰 **Один бэкенд, три поверхности** — единый Go-бинарник раздаёт React-админку и Flutter-приложение, а каждое действие также доступно через REST + WebSocket API для сторонних интеграций.
- 💬 **Шесть двусторонних каналов, без огороженных садов** — Telegram, Slack, Discord, Feishu (飞书), DingTalk (钉钉), WeCom (企业微信), плюс адаптер Bridge для чего угодно кастомного. Ответы в любом канале маршрутизируются обратно в нужную сессию.
- 🧠 **Local-first память** — эмбеддинги через ONNX / Ollama / LM Studio, поиск в трёх скоупах (пользователь · проект · сессия), умный ranking и детект конфликтов между слоями. Векторные данные не покидают вашу сеть.
- 🔌 **API уровня интеграции** — scoped API-ключи, аудит-лог по каждому вызову, маунты через reverse-proxy. Используйте opendray как шлюз за вашим собственным продуктом или просто как личный командный центр.
- 🔑 **Флот из нескольких Claude-аккаунтов** — подкладывайте в шлюз несколько аккаунтов `claude login`; панель автоматически подхватывает их через filesystem watcher, балансирует новые сессии между активными аккаунтами и позволяет переключить живую сессию между аккаунтами **без потери разговора** (транскрипт мигрируется под капотом). В каждой строке аккаунта видна актуальная загрузка (subscription tier, rate-limit tier, активные сессии, время последнего использования, текущий email Anthropic), чтобы выбрать нужный одним взглядом.
- 🔒 **Self-hosted, лицензия прозрачная** — Apache 2.0, один статический бинарник, релизы подписаны cosign, плюс SPDX SBOM. Без телеметрии, без облачного аккаунта, без подписки.

## Статус

**v2.7.0** (последний) — поколение v2 продолжает итерироваться. См.
[`VERSIONING.md`](VERSIONING.md) для политики major-как-поколение
(major = поколение продукта, а не строгий SemVer "breaking change") и
[`CHANGELOG.md`](CHANGELOG.md) для полной истории релизов.

В это поколение входит:

- **Однострочные мастера установки и удаления** (Linux + macOS;
  Windows проходит через WSL2). Проводят оператора через bootstrap
  Postgres, установку AI-CLI, учётные данные админа, адрес прослушивания,
  установку бинарника, миграцию схемы и регистрацию сервиса.
- **Самообслуживаемый бинарник** — `opendray update / start / stop /
  restart / status / providers list / providers update`, чтобы операторы
  не лезли в `systemctl` / `launchctl` ради рутинных операций.
- **Релизный пайплайн на Goreleaser** — кросс-компилируемые бинарники
  (linux/darwin × amd64/arm64), keyless-подпись cosign (Sigstore),
  SPDX SBOM, атомарно верифицируемый self-update.

## Установка

### Однострочный установщик

**Linux / macOS / WSL2**

```sh
curl -fsSL https://raw.githubusercontent.com/Opendray/opendray/main/scripts/install.sh | bash
```

**Windows** — сначала поднимает WSL2, а затем запускает Linux-инсталлятор внутри него. [подробнее →](scripts/README.md#windows)

```powershell
irm https://raw.githubusercontent.com/Opendray/opendray/main/scripts/install-windows.ps1 | iex
```

Проводит через настройку Postgres, установку AI-CLI, учётные данные админа и регистрацию сервиса — поднимает рабочий шлюз примерно за 5–10 минут. См. [**`scripts/README.md`**](scripts/README.md) — что именно делает мастер, какую раскладку файлов он создаёт, доступные опции и troubleshooting.

> **Хотите пройти всё руками?** Прочитайте [**docs/getting-started.md**](docs/getting-started.md) — 15-минутный сквозной гайд, который повторяет действия мастера, чтобы вы могли проверить каждый шаг самостоятельно.

### npm / npx (Node ≥ 18)

Установить глобально и добавить `opendray` в `PATH`:

```sh
npm install -g opendray
```

Или запустить по требованию без установки:

```sh
npx opendray
```

Устанавливает **только бинарник** — без мастера, без регистрации сервиса, без настройки Postgres. Пакет подтягивает соответствующий платформенный бинарь (`opendray-{linux,darwin}-{x64,arm64}`) через `optionalDependencies` (паттерн esbuild / Biome — никакого `postinstall`, никаких сетевых вызовов на момент установки). Подходит для скриптовых окружений, эфемерных runner-ов или когда у вас уже есть собственный Postgres и супервизор процессов.

Базу данных и запуск шлюза вы берёте на себя:

```sh
# 1. PostgreSQL 15+ с pgvector — укажите DSN, задайте пароль администратора.
export OPENDRAY_DATABASE_URL="postgres://opendray:pw@127.0.0.1:5432/opendray?sslmode=disable"
export OPENDRAY_ADMIN_PASSWORD="$(openssl rand -base64 24)"
# 2. Примените схему, затем запустите (на переднем плане).
opendray migrate
opendray serve        # → http://127.0.0.1:8770/admin/
```

Подробный гайд — настройка pgvector, `config.toml`, запуск как systemd / launchd-сервис и обновление — в [**docs/install-binary.ru.md**](docs/install-binary.ru.md).

### Удаление (Linux / macOS)

**По умолчанию** — останавливает шлюз и удаляет бинарник, но **сохраняет** ваш `config.toml`, директорию с данными (bcrypt keyfile, сессии, заметки, vault), логи и базу PostgreSQL, чтобы переустановка продолжила с того же места:

```sh
curl -fsSL https://raw.githubusercontent.com/Opendray/opendray/main/scripts/uninstall.sh | bash
```

**Полная очистка** — дополнительно сносит базу PG и роль, удаляет config / data / logs, убирает сервисного пользователя. Включает пост-проверку, которая громко падает, если что-то выжило:

```sh
curl -fsSL https://raw.githubusercontent.com/Opendray/opendray/main/scripts/uninstall.sh | OPENDRAY_PURGE=1 bash
```

### Повседневные команды

После установки бинарник `opendray` сам управляет своим жизненным циклом — не нужно вспоминать заклинания `systemctl` / `launchctl`:

```sh
sudo opendray update --restart   # download latest release, verify SHA, atomic replace + restart
```

```sh
sudo opendray providers update   # bump installed AI CLIs (claude / codex / gemini) to npm-latest
```

```sh
opendray providers list          # see which AI CLIs are installed + their versions
```

```sh
sudo opendray start              # start | stop | restart | status — wraps systemd / launchd
```

`opendray --help` показывает полный набор подкоманд.

### Выбор способа развёртывания

Любой поддерживаемый путь включает спавн сессий, доступ к AI-CLI, шифрованные бэкапы и полный API интеграций. opendray — это host-resident шлюз: он спавнит AI-CLI через PTY и делит состояние процесса (`~/.claude`, ssh-agent, файлы проекта) с ними. Такая модель несовместима с изоляцией контейнеров, которую навязал бы продакшен Docker, поэтому Docker не является поддерживаемым способом развёртывания в v2.x.

| Путь | Подходит для | Перейти к |
|---|---|---|
| 📦 **Готовый бинарник** | "Просто запусти" — Linux / macOS, любой супервизор | [Страница релизов](https://github.com/Opendray/opendray/releases) → см. [Развёртывание в продакшене](#production-deploy) |
| 🐧 **systemd-юнит** | Bare-metal / VM / LXC-машина на Linux | [Развёртывание в продакшене §A](#option-a--systemd-bare-metal--vm--lxc) |
| 🍎 **LaunchDaemon на macOS** | Mac mini / Mac Studio как домашний сервер | [Развёртывание в продакшене §C](#option-c--macos-launchd-mac-mini--studio-as-home-server) |
| 🛠 **Сборка из исходников** | Разработка / контрибьют / кастомные сборки | [Quickstart](#quickstart-5-minute-dev-path) ниже |

<a id="quickstart-5-minute-dev-path"></a>

## Quickstart (dev-путь за 5 минут)

Полный walkthrough с пререквизитами и troubleshooting — в [`docs/quickstart.md`](docs/quickstart.md). Сжатый dev-путь:

```bash
# 1. Have a Postgres 15+ running on 127.0.0.1:5432 with pgvector enabled
#    (apt install postgresql-16 postgresql-16-pgvector / brew install postgresql@16 pgvector).
#    Point [database].url at any other DSN if you'd rather use a remote PG.

# 2. Local config — already gitignored.
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

Так OpenDray работает на переднем плане — Ctrl-C его убивает. Для долгоживущего демона смотрите **Развёртывание в продакшене** ниже.

<a id="production-deploy"></a>

## Развёртывание в продакшене

Четыре поддерживаемых способа развёртывания, выбирайте подходящий под ваше окружение.
Каждый даёт авто-рестарт при крэше, персистентное состояние и
разделение секретов и конфига.

<a id="option-a--systemd-bare-metal--vm--lxc"></a>

### Вариант A — systemd (bare-metal / VM / LXC)

Рекомендуемый способ деплоя в Linux. Поставляется захардененный юнит в
[`deploy/systemd/opendray.service`](deploy/systemd/opendray.service)
с sandboxing (`ProtectSystem=strict`, `NoNewPrivileges`,
`MemoryDenyWriteExecute`, очистка capabilities), порядком запуска
`migrate`-затем-`serve` и окном graceful-stop в 20 секунд.

**Сначала добудьте бинарник.** Либо скачайте готовый архив со
[страницы релизов](https://github.com/Opendray/opendray/releases)
(`opendray_*_linux_<arch>.tar.gz` — распаковывается в единственный
бинарник `opendray`), либо соберите из исходников через [Quickstart](#quickstart-5-minute-dev-path)
выше (`go build ./cmd/opendray`).

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

Юнит запускает `opendray migrate` в `ExecStartPre`, так что первый
старт применит все миграции до того, как `serve` вообще начнёт работать.
Рестарты — `on-failure` с back-off 5 секунд и лимитом 5 попыток в минуту.

### Вариант B — Прямой бинарник + ваш собственный супервизор процессов

Для LXC без systemd, FreeBSD `rc.d`, OpenRC или чего угодно ещё.
Собрали один раз — запускайте под тем супервизором, которым уже пользуетесь:

```bash
# Cross-compile a release archive locally:
goreleaser release --clean --snapshot
ls dist/                  # opendray_*_linux_amd64.tar.gz etc.

# Or grab a published release artefact:
# https://github.com/Opendray/opendray/releases
```

Затем направьте свой супервизор (s6, runit, supervisord, runwhen) на:

```
/usr/local/bin/opendray serve -config /etc/opendray/config.toml
```

Pre-flight: один раз перед первым `serve` запустите
`opendray migrate -config /etc/opendray/config.toml`, либо повесьте
это в pre-start хук вашего супервизора.

<a id="option-c--macos-launchd-mac-mini--studio-as-home-server"></a>

### Вариант C — macOS launchd (Mac mini / Studio как домашний сервер)

Для Mac mini / Mac Studio на Apple Silicon, работающих 24/7. Поставляется
LaunchDaemon в
[`deploy/launchd/com.opendray.opendray.plist`](deploy/launchd/com.opendray.opendray.plist),
который стартует при загрузке до логина любого пользователя,
рестартит при крэше с throttle 5 секунд и пишет логи в
`/usr/local/var/log/opendray/`.

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

Рестарт — `sudo launchctl kickstart -k system/com.opendray.opendray`;
полная выгрузка — `sudo launchctl bootout system/com.opendray.opendray`.

Postgres на macOS — ставится через Homebrew (`brew install postgresql@17 && brew services start postgresql@17`), а `[database].url` направляется на
`postgres://$USER@127.0.0.1:5432/opendray`. Добавьте `pgvector` командой
`brew install pgvector` и выполните `CREATE EXTENSION vector` внутри
базы opendray.

---

По специфике LXC в Proxmox (PTY в unprivileged-контейнерах,
сеть, тюнинг cgroup) — см. [`deploy/lxc/proxmox-pty-notes.md`](deploy/lxc/proxmox-pty-notes.md).

По reverse-proxy / терминации TLS (nginx, Caddy, Traefik, Cloudflare
Tunnel) — см. [`docs/operator-guide.md`](docs/operator-guide.md) §Topology.

### Опционально: включить шифрованные бэкапы БД + экспорт данных

```bash
# Master passphrase (env-only — never write into config.toml).
export OPENDRAY_BACKUP_KEY="$(openssl rand -base64 32)"
export OPENDRAY_BACKUP_ENABLED=1

# pg_dump / pg_restore must match the server's major version. On
# Apple Silicon dev machines pointing at a PG17 server:
export OPENDRAY_BACKUP_PG_DUMP_PATH=/opt/homebrew/opt/postgresql@17/bin/pg_dump
export OPENDRAY_BACKUP_PG_RESTORE_PATH=/opt/homebrew/opt/postgresql@17/bin/pg_restore
```

Перезапустите opendray; в сайдбаре появится страница Backups (`/backups`)
для шифрованных дампов PostgreSQL и restore, а также `/export` для
экспорта данных в zip-bundle и импорта. Полный жизненный цикл — в
[`docs/operator-guide.md`](docs/operator-guide.md) §Backup.

Один Go-бинарник несёт весь веб-бандл — никакого Node-рантайма во время
работы, отдельного сервера статики или Caddy/nginx не требуется.
Cloudflare Tunnel терминирует TLS перед `:8770`.

## Раскладка

```
cmd/opendray/        binary entry point (≤100 LOC per design §14)
internal/
├── app/             composition root (wires every subsystem)
├── audit/           subscribes to bus topics, persists to audit_log
├── auth/            admin bearer tokens (M2.5)
├── backup/          encrypted DB dumps + admin export/import├── catalog/         CLI provider manifests + per-id user config (M2)
├── channel/         channel hub + telegram impl (M4)
├── config/          TOML loader with OPENDRAY_* env overrides
├── eventbus/        in-process pub/sub
├── gateway/         chi HTTP router + middleware + slog
├── integration/     external-app registry + reverse proxy + events WS (M3)
├── memory/          cross-CLI persistent memory├── session/         PTY lifecycle + ring buffer + WS stream (M1)
├── store/           pgx pool + hand-rolled migration runner (M0)
├── version/         build-time identification
└── web/             go:embed of the web bundle (W5)

app/web/             React 19 + TypeScript + Vite SPA (Phase 2 W0-W5)
app/mobile/          Flutter app (iOS + Android), feature parity with web
docs/
├── design.md        SSOT north-star
└── adr/             architecture decisions, dated
```

## Веб-фронтенд

`app/web/` собирается в единую SPA в `internal/web/dist/`, которую Go-бинарник
встраивает через `go:embed` и раздаёт по `/admin/*`. Dev-сервер Vite на `:5173`
проксирует `/api` на `:8770` для разработки с HMR.

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

См. [`app/web/README.md`](app/web/README.md) — там стек фронтенда
(React + Vite + Tailwind v4 + shadcn/ui + TanStack Router/Query +
Zustand + xterm.js) и заметки по W-майлстоунам.

## Документация

- [`docs/getting-started.md`](docs/getting-started.md) — **начинайте отсюда**, если вы новичок: от нуля до первой сессии за 15 минут, включая установку оборачиваемых CLI и bootstrap Postgres
- [`docs/install-binary.ru.md`](docs/install-binary.ru.md) — установка из npm-пакета или release-бинарника (собственный Postgres) и запуск как systemd / launchd-сервис
- [`docs/quickstart.md`](docs/quickstart.md) — dev-окружение за 5 минут (предполагается, что вы уже знаете, из чего всё состоит)
- [`docs/operator-guide.md`](docs/operator-guide.md) — справочник по деплою и эксплуатации для околопродакшен-сетапов
- [`docs/integration-guide.md`](docs/integration-guide.md) — как написать внешнюю интеграцию на любом языке
- [`VERSIONING.md`](VERSIONING.md) — стратегия версионирования (major-as-generation)
- [`CHANGELOG.md`](CHANGELOG.md) — история релизов

## Тесты

```bash
go test -race ./...        # backend
cd app/web && pnpm build   # web (TS strict + vite production build)
```

End-to-end smoke-флоу отслеживаются в commit-сообщениях по майлстоунам.
Playwright-harness запланирован как follow-up.

## Связь с v1

v1 (`Opendray/opendray`) — это легаси-кодбейс, теперь архивный. v2 —
текущее и активное поколение, feature-complete и единственная ветка,
в которой ведётся разработка. Из 16 builtin'ов v1 четыре переехали в
бэкенд v2; остальные стали клиентскими фичами, channel-адаптерами или
потребителями API интеграций.

## Лицензия

Apache 2.0 — см. [`LICENSE`](LICENSE). (v1 был под MIT; v2 лицензирован
независимо.)
