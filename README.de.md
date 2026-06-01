<p align="center">
  <a href="https://opendray.dev"><img src="docs/assets/logo.png" alt="opendray" width="180"></a>
</p>

<h1 align="center">opendray</h1>

<p align="center">
  <strong>Self-hosted Gateway für Claude Code · Codex · Gemini · Shell — mit einer gemeinsamen Local-First-Memory-Schicht über alle hinweg.</strong>
  <br/>
  <sub>Lass Sessions auf deiner eigenen Infrastruktur laufen. Steuere sie aus dem Web, vom Smartphone oder aus dem Chat. Offene REST- + WebSocket-API für Integrationen.</sub>
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
  🌐 <a href="README.md">English</a> · <a href="README.zh.md">简体中文</a> · <a href="README.fa.md">فارسی</a> · <a href="README.es.md">Español</a> · <a href="README.pt-BR.md">Português</a> · <a href="README.ja.md">日本語</a> · <a href="README.ko.md">한국어</a> · <a href="README.fr.md">Français</a> · <strong>Deutsch</strong> · <a href="README.ru.md">Русский</a>
</p>

---

## Was ist opendray?

**opendray** umschließt die KI-Coding-CLIs, die du eh schon nutzt — Claude Code, Codex, Gemini und jede beliebige Shell — und macht sie zu etwas, das du von überall aus steuern kannst. Lass Sessions auf deinem Heimserver / NAS / VPS laufen, lass dich per Telegram benachrichtigen, wenn eine in den Idle-Zustand geht, und antworte vom Handy aus, um den nächsten Prompt einzufüttern — alles über ein Self-hosted Gateway, das du Ende zu Ende kontrollierst.

- 🛰 **Ein Backend, drei Oberflächen** — ein einziges Go-Binary, das ein React-Webadmin und eine Flutter-Mobile-App ausliefert; jede Aktion ist zusätzlich über eine REST- + WebSocket-API für Drittanbieter-Integrationen verfügbar.
- 💬 **Sechs bidirektionale Channels, keine Walled Gardens** — Telegram, Slack, Discord, Feishu (飞书), DingTalk (钉钉), WeCom (企业微信), plus ein Bridge-Adapter für alles Eigene. Antworten auf jedem Channel werden zurück in die passende Session geroutet.
- 🧠 **Local-First Memory** — ONNX- / Ollama- / LM-Studio-Embeddings mit Retrieval in drei Scopes (User · Projekt · Session), smartes Ranking und Konflikterkennung über die Layer hinweg. Keine Vektordaten verlassen dein Netzwerk.
- 🔌 **API auf Integrations-Niveau** — gescopte API-Keys, Audit-Log pro Call, Reverse-Proxy-Mounts. Nutze opendray als Gateway hinter deinem eigenen Produkt oder einfach als persönliche Kommandozentrale.
- 🔑 **Flotte mit mehreren Claude-Accounts** — wirf mehrere `claude login`-Accounts ins Gateway; das Panel erkennt sie automatisch über einen Filesystem-Watcher, balanciert neue Sessions über die aktivierten Accounts und lässt dich eine laufende Session zwischen Accounts umschalten, **ohne die Konversation zu verlieren** (das Transcript wird unter der Haube migriert). Jede Account-Zeile zeigt die aktuelle Kapazität (Subscription-Tier, Rate-Limit-Tier, aktive Sessions, zuletzt verwendet, aktuelle Anthropic-E-Mail), sodass du auf einen Blick den richtigen Account auswählen kannst.
- 🔒 **Self-hosted, klare Lizenz** — Apache 2.0, ein statisches Binary, cosign-signierte Releases mit SPDX-SBOM. Keine Telemetrie, kein Cloud-Account, kein Abo.

## Status

**v2.6.0** (aktuell) — die v2-Generation iteriert weiter. Siehe
[`VERSIONING.md`](VERSIONING.md) für die Major-als-Generation-Policy
(Major = Produktgeneration, kein strikter SemVer-"Breaking Change") und
[`CHANGELOG.md`](CHANGELOG.md) für die vollständige Release-Historie.

Diese Generation liefert:

- **Einzeilige Installer- und Uninstaller-Wizards** (Linux + macOS;
  Windows läuft über WSL2). Führen den Operator durch Postgres-
  Bootstrap, AI-CLI-Installation, Admin-Credentials, Listen-Adresse,
  Binary-Installation, Schema-Migration und Service-Registrierung.
- **Self-managing Binary** — `opendray update / start / stop /
  restart / status / providers list / providers update`, damit
  Operatoren für Routine-Ops nicht an `systemctl` / `launchctl` ran müssen.
- **Goreleaser-Release-Pipeline** — cross-kompilierte Binaries
  (linux/darwin × amd64/arm64), keyless cosign-Signing (Sigstore),
  SPDX-SBOM, atomar verifiziertes Self-Update.

## Installation

### Einzeiliger Installer

**Linux / macOS / WSL2**

```sh
curl -fsSL https://raw.githubusercontent.com/Opendray/opendray/main/scripts/install.sh | bash
```

**Windows** — richtet zuerst WSL2 ein und führt darin dann den Linux-Installer aus. [Details →](scripts/README.md#windows)

```powershell
irm https://raw.githubusercontent.com/Opendray/opendray/main/scripts/install-windows.ps1 | iex
```

Führt dich durch Postgres-Setup, AI-CLI-Installation, Admin-Credentials und Service-Registrierung — am Ende läuft ein Gateway in ~5–10 Minuten. Siehe [**`scripts/README.md`**](scripts/README.md) dafür, was der Wizard macht, welches File-Layout er anlegt, welche Optionen es gibt und für Troubleshooting.

> **Lieber den manuellen Walkthrough?** Lies [**docs/getting-started.md**](docs/getting-started.md) — eine 15-minütige End-to-End-Anleitung, die dasselbe wie der Wizard macht, sodass du jeden Schritt selbst nachvollziehen kannst.

### npm / npx (Node ≥ 18)

```sh
npm install -g opendray   # legt `opendray` ins PATH
# oder
npx opendray --help       # ohne Install, lädt on-demand
```

Für den Fall, dass du nur das statische Binary auf dem `PATH` willst — kein Wizard, keine Service-Registrierung, kein Postgres-Setup. Nützlich in geskripteten Umgebungen, ephemeren Runnern, oder wenn du schon dein eigenes Deployment-System hast. Das Paket zieht das passende Plattform-Binary (`opendray-{linux,darwin}-{x64,arm64}`) via `optionalDependencies` (das esbuild / Biome-Pattern — kein `postinstall`, kein Netzwerk-Call beim Install).

### Uninstall (Linux / macOS)

**Standard** — stoppt das Gateway und entfernt das Binary, **behält** aber deine `config.toml`, das Data-Verzeichnis (bcrypt-Keyfile, Sessions, Notes, Vault), die Logs und die PostgreSQL-Datenbank, sodass eine Neuinstallation dort weitermacht, wo du aufgehört hast:

```sh
curl -fsSL https://raw.githubusercontent.com/Opendray/opendray/main/scripts/uninstall.sh | bash
```

**Full Purge** — entfernt zusätzlich PG-Datenbank + Role, löscht Config / Data / Logs und entfernt den Service-User. Inklusive Verifikationsschritt nach dem Löschen, der lautstark Alarm schlägt, falls irgendetwas überlebt hat:

```sh
curl -fsSL https://raw.githubusercontent.com/Opendray/opendray/main/scripts/uninstall.sh | OPENDRAY_PURGE=1 bash
```

### Day-to-Day-Befehle

Nach der Installation kümmert sich das `opendray`-Binary selbst um seinen Lifecycle — keine `systemctl`- / `launchctl`-Beschwörungsformeln mehr nötig:

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

`opendray --help` listet das komplette Subcommand-Set.

### Deploy-Path-Picker

Jeder unterstützte Pfad enthält Session-Spawn, AI-CLI-Zugriff, verschlüsselte Backups und die vollständige Integrations-API. opendray ist ein host-resident Gateway — es spawnt AI-CLIs über PTYs und teilt Prozess-State (`~/.claude`, ssh-agent, Projektdateien) mit ihnen. Dieses Modell ist inkompatibel mit der Container-Isolierung, die ein produktives Docker erzwingen würde — daher ist Docker in v2.x kein unterstützter Deployment-Pfad.

| Pfad | Am besten für | Springe zu |
|---|---|---|
| 📦 **Vorgefertigtes Binary** | "Einfach laufen lassen" — Linux / macOS, beliebiger Supervisor | [Releases-Seite](https://github.com/Opendray/opendray/releases) → siehe [Produktions-Deployment](#production-deploy) |
| 🐧 **systemd-Unit** | Bare-Metal- / VM- / LXC-Linux-Box | [Produktions-Deployment §A](#option-a--systemd-bare-metal--vm--lxc) |
| 🍎 **macOS LaunchDaemon** | Mac mini / Mac Studio als Heimserver | [Produktions-Deployment §C](#option-c--macos-launchd-mac-mini--studio-as-home-server) |
| 🛠 **Build from Source** | Dev / Contributing / Custom Builds | [Quickstart](#quickstart-5-minute-dev-path) weiter unten |

## Quickstart (5-Minuten-Dev-Path)

Den kompletten Walkthrough mit Voraussetzungen und Troubleshooting findest du in [`docs/quickstart.md`](docs/quickstart.md). Der verdichtete Dev-Path:

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

Damit läuft OpenDray im Vordergrund — Ctrl-C beendet es. Für einen langlaufenden
Daemon siehe **Produktions-Deployment** unten.

## Produktions-Deployment

Vier unterstützte Deploy-Pfade, such dir den passenden für deine Umgebung aus.
Jeder davon liefert dir Auto-Restart bei Crash, persistenten State und
Trennung von Secrets und Config.

### Option A — systemd (Bare-Metal / VM / LXC)

Der empfohlene Linux-Deploy-Pfad. Liefert eine gehärtete Unit unter
[`deploy/systemd/opendray.service`](deploy/systemd/opendray.service)
mit Sandboxing (`ProtectSystem=strict`, `NoNewPrivileges`,
`MemoryDenyWriteExecute`, Capability-Scrub), `migrate`-dann-`serve`-
Boot und einem 20-s-Graceful-Stop-Fenster.

**Hol dir zuerst ein Binary.** Schnapp dir entweder ein vorgefertigtes Archiv von der
[Releases-Seite](https://github.com/Opendray/opendray/releases)
(`opendray_*_linux_<arch>.tar.gz` — entpackt sich zu einem einzigen `opendray`-
Binary) oder baue es aus dem Source via [Quickstart](#quickstart-5-minute-dev-path)
oben (`go build ./cmd/opendray`).

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

Die Unit führt `opendray migrate` als `ExecStartPre` aus, sodass der erste Boot
alle Migrations anwendet, bevor `serve` überhaupt startet. Restarts laufen
`on-failure` mit 5 s Back-off und einem Limit von 5 Bursts pro Minute.

### Option B — Direktes Binary + dein eigener Process-Supervisor

Für LXC ohne systemd, FreeBSD `rc.d`, OpenRC oder sonst irgendwas.
Einmal bauen, mit dem Supervisor laufen lassen, den du eh schon einsetzt:

```bash
# Cross-compile a release archive locally:
goreleaser release --clean --snapshot
ls dist/                  # opendray_*_linux_amd64.tar.gz etc.

# Or grab a published release artefact:
# https://github.com/Opendray/opendray/releases
```

Dann zeigt dein Supervisor (s6, runit, supervisord, runwhen) auf:

```
/usr/local/bin/opendray serve -config /etc/opendray/config.toml
```

Pre-Flight: führe `opendray migrate -config /etc/opendray/config.toml`
einmal vor dem ersten `serve` aus, oder als Pre-Start-Hook im
Supervisor deiner Wahl.

### Option C — macOS launchd (Mac mini / Studio als Heimserver)

Für Apple-Silicon-Mac-mini / Mac Studio im 24/7-Betrieb. Liefert einen
LaunchDaemon unter
[`deploy/launchd/com.opendray.opendray.plist`](deploy/launchd/com.opendray.opendray.plist),
der vor jedem User-Login beim Boot startet, bei Crashes mit 5-s-Throttle
neu startet und nach `/usr/local/var/log/opendray/` loggt.

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

Restart mit `sudo launchctl kickstart -k system/com.opendray.opendray`;
komplett entladen mit `sudo launchctl bootout system/com.opendray.opendray`.

Postgres auf macOS — installiere via Homebrew (`brew install postgresql@17 && brew services start postgresql@17`) und zeige mit `[database].url` auf
`postgres://$USER@127.0.0.1:5432/opendray`. `pgvector` ergänzt du mit
`brew install pgvector` und `CREATE EXTENSION vector` innerhalb der
opendray-Datenbank.

---

Proxmox-spezifische LXC-Notes (PTY in Unprivileged Containers,
Networking, cgroup-Tweaks) findest du in [`deploy/lxc/proxmox-pty-notes.md`](deploy/lxc/proxmox-pty-notes.md).

Für Reverse-Proxy / TLS-Termination (nginx, Caddy, Traefik, Cloudflare
Tunnel) siehe [`docs/operator-guide.md`](docs/operator-guide.md) §Topology.

### Optional: verschlüsselte DB-Backups + Daten-Exporte aktivieren

```bash
# Master passphrase (env-only — never write into config.toml).
export OPENDRAY_BACKUP_KEY="$(openssl rand -base64 32)"
export OPENDRAY_BACKUP_ENABLED=1

# pg_dump / pg_restore must match the server's major version. On
# Apple Silicon dev machines pointing at a PG17 server:
export OPENDRAY_BACKUP_PG_DUMP_PATH=/opt/homebrew/opt/postgresql@17/bin/pg_dump
export OPENDRAY_BACKUP_PG_RESTORE_PATH=/opt/homebrew/opt/postgresql@17/bin/pg_restore
```

Starte opendray neu; in der Sidebar erscheint dann eine Backups-Seite (`/backups`)
für verschlüsselte PostgreSQL-Dumps + Restore und `/export` für
Daten-Exporte als Zip-Bundle + Import. Den vollständigen Lifecycle beschreibt [`docs/operator-guide.md`](docs/operator-guide.md) §Backup.

Ein einziges Go-Binary trägt das komplette Web-Bundle in sich — keine Node-Runtime
zur Laufzeit nötig, kein separater Static-File-Server, kein Caddy/nginx
erforderlich. Cloudflare Tunnel terminiert TLS vor `:8770`.

## Layout

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

## Web-Frontend

`app/web/` baut eine einzelne SPA nach `internal/web/dist/`, die das Go-
Binary einbettet und unter `/admin/*` ausliefert. Der Vite-Dev-Server auf `:5173`
proxied `/api` nach `:8770` für HMR-getriebene Entwicklung.

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

Den Frontend-Stack (React + Vite + Tailwind v4 + shadcn/ui + TanStack
Router/Query + Zustand + xterm.js) und Notes pro W-Milestone findest du in
[`app/web/README.md`](app/web/README.md).

## Dokumentation

- [`docs/getting-started.md`](docs/getting-started.md) — **fang hier an**, wenn du neu bist: von null bis zur ersten Session in 15 Minuten, inklusive Installation der gewrappten CLIs und Postgres-Bootstrap
- [`docs/quickstart.md`](docs/quickstart.md) — 5-Minuten-Dev-Umgebung (setzt voraus, dass du die beweglichen Teile schon kennst)
- [`docs/operator-guide.md`](docs/operator-guide.md) — Deploy- und Ops-Referenz für produktionsnahe Setups
- [`docs/integration-guide.md`](docs/integration-guide.md) — wie du eine externe Integration in beliebiger Sprache schreibst
- [`VERSIONING.md`](VERSIONING.md) — Versioning-Strategie (Major-als-Generation)
- [`CHANGELOG.md`](CHANGELOG.md) — Release-Historie

## Tests

```bash
go test -race ./...        # backend
cd app/web && pnpm build   # web (TS strict + vite production build)
```

End-to-End-Smoke-Flows werden pro Milestone in den Commit-Messages getrackt.
Ein Playwright-Harness ist als Follow-up geplant.

## Verhältnis zu v1

v1 (`Opendray/opendray`) ist die Legacy-Codebase, inzwischen archiviert. v2 ist
die aktuelle und aktive Generation — feature-complete und der einzige
Branch, der noch Entwicklung sieht. Von den 16 v1-Builtins sind vier ins
v2-Backend gewandert; der Rest wurde zu Client-seitigen Features, Channel-
Adaptern oder Konsumenten der Integrations-API.

## Lizenz

Apache 2.0 — siehe [`LICENSE`](LICENSE). (v1 war MIT; v2 wird unabhängig
davon lizenziert.)
