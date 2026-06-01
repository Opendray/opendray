<p align="center">
  <a href="https://opendray.dev"><img src="docs/assets/logo.png" alt="opendray" width="180"></a>
</p>

<h1 align="center">opendray</h1>

<p align="center">
  <strong>Claude Code · Codex · Gemini · shell을 위한 self-hosted 게이트웨이 — 이 모두를 가로지르는 단일 local-first 메모리 레이어 제공.</strong>
  <br/>
  <sub>세션을 본인 인프라에서 실행하고, 웹·모바일·채팅 어디서든 제어하세요. 통합을 위한 개방형 REST + WebSocket API를 제공합니다.</sub>
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
  🌐 <a href="README.md">English</a> · <a href="README.zh.md">简体中文</a> · <a href="README.fa.md">فارسی</a> · <a href="README.es.md">Español</a> · <a href="README.pt-BR.md">Português</a> · <a href="README.ja.md">日本語</a> · <strong>한국어</strong> · <a href="README.fr.md">Français</a> · <a href="README.de.md">Deutsch</a> · <a href="README.ru.md">Русский</a>
</p>

---

## opendray란?

**opendray**는 이미 쓰고 있는 AI 코딩 CLI들 — Claude Code, Codex, Gemini, 그리고 임의의 shell — 을 감싸서, 어디서든 제어 가능한 형태로 바꿔 줍니다. 홈 서버 / NAS / VPS에서 세션을 돌리고, idle 상태가 되면 Telegram으로 알림을 받고, 휴대폰에서 곧바로 다음 prompt를 흘려 넣을 수 있습니다. 모든 흐름이 처음부터 끝까지 본인이 통제하는 self-hosted 게이트웨이를 통해 이뤄집니다.

- 🛰 **하나의 backend, 세 가지 표면** — 단일 Go 바이너리가 React 웹 어드민과 Flutter 모바일 앱을 함께 서빙하며, 모든 동작은 서드파티 통합을 위해 REST + WebSocket API로도 노출됩니다.
- 💬 **6개의 양방향 채널, 닫힌 정원 없음** — Telegram, Slack, Discord, Feishu (飞书), DingTalk (钉钉), WeCom (企业微信), 그리고 커스텀 용도를 위한 Bridge 어댑터. 어느 채널에서 답장을 보내든 알맞은 세션으로 다시 라우팅됩니다.
- 🧠 **Local-first 메모리** — ONNX / Ollama / LM Studio 임베딩 기반에 3개 스코프 검색(사용자 · 프로젝트 · 세션), 스마트 랭킹, 레이어 간 충돌 감지까지 갖췄습니다. 벡터 데이터는 네트워크 밖으로 나가지 않습니다.
- 🔌 **통합 등급 API** — 스코프가 지정된 API 키, 호출 단위 audit log, reverse-proxy 마운트를 지원합니다. opendray를 자체 제품의 뒤를 받치는 게이트웨이로 쓰든, 개인용 command centre로 쓰든 자유입니다.
- 🔑 **Multi-Claude 계정 플릿** — 여러 `claude login` 계정을 게이트웨이에 넣어두면, 패널이 파일시스템 워처로 자동 감지하고 활성화된 계정들 사이에서 새 세션을 균형 있게 분배합니다. 실행 중인 세션을 **대화 흐름을 잃지 않고** 다른 계정으로 전환할 수도 있습니다(transcript가 내부적으로 마이그레이션됩니다). 각 계정 행에는 현재 capacity(subscription tier, rate-limit tier, 활성 세션 수, 마지막 사용 시각, 현재 Anthropic 이메일)가 실시간으로 표시되어 한눈에 적절한 계정을 고를 수 있습니다.
- 🔒 **Self-hosted, 명확한 라이선스** — Apache 2.0, 단일 정적 바이너리, cosign 서명된 release와 SPDX SBOM을 제공합니다. 텔레메트리 없음, 클라우드 계정 없음, 구독 없음.

## 현황

**v2.6.0** (최신) — v2 세대는 계속 이터레이션 중입니다.
[`VERSIONING.md`](VERSIONING.md)에서 major-as-generation 정책
(major = 제품 세대, 엄격한 SemVer "breaking change"가 아님)을,
[`CHANGELOG.md`](CHANGELOG.md)에서 전체 release 이력을 확인하세요.

이 세대에서 제공되는 것:

- **원라인 설치 / 제거 마법사** (Linux + macOS;
  Windows는 WSL2를 거쳐 진행). 운영자에게 Postgres 부트스트랩,
  AI-CLI 설치, 어드민 자격증명, 리스닝 주소,
  바이너리 설치, 스키마 마이그레이션, 서비스 등록까지 단계별로 안내합니다.
- **자기 관리형 바이너리** — `opendray update / start / stop /
  restart / status / providers list / providers update`를 통해
  일상 운영 작업에서 `systemctl` / `launchctl`에 손댈 일을 없앴습니다.
- **Goreleaser release 파이프라인** — 크로스 컴파일된 바이너리
  (linux/darwin × amd64/arm64), cosign keyless 서명(Sigstore),
  SPDX SBOM, 원자적으로 검증되는 self-update를 제공합니다.

## 설치

### 원라인 설치 스크립트

**Linux / macOS / WSL2**

```sh
curl -fsSL https://raw.githubusercontent.com/Opendray/opendray/main/scripts/install.sh | bash
```

**Windows** — 먼저 WSL2를 세팅한 다음 그 안에서 Linux용 설치 스크립트를 실행합니다. [상세 →](scripts/README.md#windows)

```powershell
irm https://raw.githubusercontent.com/Opendray/opendray/main/scripts/install-windows.ps1 | iex
```

Postgres 설정, AI-CLI 설치, 어드민 자격증명, 서비스 등록까지 차례로 진행하며 — 약 5~10분이면 게이트웨이가 떠 있는 상태가 됩니다. 마법사가 무엇을 하는지, 어떤 파일 레이아웃을 만드는지, 옵션과 트러블슈팅은 [**`scripts/README.md`**](scripts/README.md)를 참고하세요.

> **수동 절차로 진행하고 싶다면?** [**docs/getting-started.md**](docs/getting-started.md)를 읽어보세요 — 마법사가 하는 일을 그대로 풀어 놓은 15분짜리 엔드투엔드 가이드라서 각 단계를 직접 검증할 수 있습니다.

### 제거 (Linux / macOS)

**기본** — 게이트웨이를 중지하고 바이너리를 제거하지만, `config.toml`, 데이터 디렉터리(bcrypt 키파일, 세션, 노트, vault), 로그, PostgreSQL 데이터베이스는 **그대로 유지**합니다. 다시 설치하면 멈췄던 지점에서 이어집니다:

```sh
curl -fsSL https://raw.githubusercontent.com/Opendray/opendray/main/scripts/uninstall.sh | bash
```

**완전 삭제** — PG 데이터베이스와 role까지 drop하고, config / data / logs를 지우며, 서비스 사용자도 제거합니다. 삭제 후에 무엇이라도 살아 있으면 시끄럽게 실패하는 검증 단계가 포함되어 있습니다:

```sh
curl -fsSL https://raw.githubusercontent.com/Opendray/opendray/main/scripts/uninstall.sh | OPENDRAY_PURGE=1 bash
```

### 일상 운영 명령어

설치 후에는 `opendray` 바이너리가 자기 라이프사이클을 직접 다룹니다 — `systemctl` / `launchctl` 주문을 외울 필요가 없습니다:

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

`opendray --help`로 전체 서브커맨드 목록을 볼 수 있습니다.

### Deploy 경로 선택기

지원되는 모든 경로는 세션 spawn, AI-CLI 접근, 암호화 백업, 통합 API 전체를 포함합니다. opendray는 호스트에 상주하는 게이트웨이로 — PTY를 통해 AI CLI를 spawn하고, 그들과 프로세스 상태(`~/.claude`, ssh-agent, 프로젝트 파일)를 공유합니다. 이 모델은 프로덕션 Docker가 강제하는 컨테이너 격리와 맞지 않기 때문에, v2.x에서는 Docker가 지원되는 deploy 경로가 아닙니다.

| 경로 | 추천 대상 | 이동 |
|---|---|---|
| 📦 **사전 빌드 바이너리** | "그냥 실행" — Linux / macOS, 임의의 supervisor | [Releases page](https://github.com/Opendray/opendray/releases) → [프로덕션 deploy](#production-deploy) 참고 |
| 🐧 **systemd 유닛** | 베어메탈 / VM / LXC Linux 박스 | [프로덕션 deploy §A](#option-a--systemd-bare-metal--vm--lxc) |
| 🍎 **macOS LaunchDaemon** | 홈 서버용 Mac mini / Mac Studio | [프로덕션 deploy §C](#option-c--macos-launchd-mac-mini--studio-as-home-server) |
| 🛠 **소스에서 빌드** | 개발 / 기여 / 커스텀 빌드 | 아래의 [Quickstart](#quickstart-5-minute-dev-path) |

## Quickstart (5분 개발 경로)

사전 준비물과 트러블슈팅이 포함된 전체 가이드는 [`docs/quickstart.md`](docs/quickstart.md)를 참고하세요. 압축된 개발 경로는 다음과 같습니다:

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

이 방식은 OpenDray를 포그라운드에서 실행합니다 — Ctrl-C로 종료됩니다. 장기 실행 데몬으로 띄우려면 아래의 **프로덕션 deploy**를 참고하세요.

## 프로덕션 deploy

지원되는 deploy 경로는 네 가지이며, 각자의 환경에 맞는 것을 고르면 됩니다.
어느 쪽이든 crash 시 auto-restart, 영구 상태 유지, 시크릿과 config의
분리를 보장합니다.

### Option A — systemd (베어메탈 / VM / LXC)

권장 Linux deploy 경로입니다. [`deploy/systemd/opendray.service`](deploy/systemd/opendray.service)에
샌드박싱(`ProtectSystem=strict`, `NoNewPrivileges`,
`MemoryDenyWriteExecute`, capability scrub), `migrate` 이후 `serve`로 이어지는
부팅 순서, 20초의 graceful-stop 윈도우가 적용된 hardened 유닛이 들어 있습니다.

**먼저 바이너리부터 확보하세요.** [Releases 페이지](https://github.com/Opendray/opendray/releases)에서
사전 빌드 아카이브(`opendray_*_linux_<arch>.tar.gz` — 압축 해제 시 단일 `opendray`
바이너리 한 개로 풀립니다)를 받거나, 위의 [Quickstart](#quickstart-5-minute-dev-path)를
참고해 소스에서 빌드(`go build ./cmd/opendray`)하면 됩니다.

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

이 유닛은 `opendray migrate`를 `ExecStartPre`로 실행하기 때문에, 최초 부팅에서
`serve`가 시작되기 전에 모든 마이그레이션이 적용됩니다. 재시작 정책은
`on-failure`이며 5초 back-off에 분당 5회 burst 제한이 걸려 있습니다.

### Option B — 바이너리 직접 실행 + 자체 프로세스 supervisor

systemd가 없는 LXC, FreeBSD `rc.d`, OpenRC 등 다른 환경을 위한 경로입니다.
한 번 빌드해두고, 이미 쓰고 있는 supervisor로 띄우면 됩니다:

```bash
# Cross-compile a release archive locally:
goreleaser release --clean --snapshot
ls dist/                  # opendray_*_linux_amd64.tar.gz etc.

# Or grab a published release artefact:
# https://github.com/Opendray/opendray/releases
```

그런 다음 supervisor(s6, runit, supervisord, runwhen 등)가 다음을 실행하도록
설정합니다:

```
/usr/local/bin/opendray serve -config /etc/opendray/config.toml
```

Pre-flight: 최초 `serve` 이전에 `opendray migrate -config /etc/opendray/config.toml`을
한 번 실행하거나, 사용 중인 supervisor의 pre-start 훅으로 걸어두세요.

### Option C — macOS launchd (홈 서버용 Mac mini / Studio)

24/7 가동되는 Apple Silicon Mac mini / Mac Studio를 위한 경로입니다.
[`deploy/launchd/com.opendray.opendray.plist`](deploy/launchd/com.opendray.opendray.plist)에
사용자 로그인 이전 부팅 시점에 시작하고, crash 시 5초 throttle로 재시작하며,
`/usr/local/var/log/opendray/`에 로그를 남기는 LaunchDaemon이 들어 있습니다.

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

재시작은 `sudo launchctl kickstart -k system/com.opendray.opendray`로,
완전 언로드는 `sudo launchctl bootout system/com.opendray.opendray`로 수행합니다.

macOS의 Postgres — Homebrew로 설치(`brew install postgresql@17 && brew services start postgresql@17`)하고 `[database].url`을
`postgres://$USER@127.0.0.1:5432/opendray`로 지정하세요. `pgvector`는
`brew install pgvector`로 추가한 뒤 opendray 데이터베이스 안에서
`CREATE EXTENSION vector`를 실행하면 됩니다.

---

Proxmox에서의 LXC 관련 노트(unprivileged 컨테이너에서의 PTY,
네트워킹, cgroup 조정)는 [`deploy/lxc/proxmox-pty-notes.md`](deploy/lxc/proxmox-pty-notes.md)를 참고하세요.

reverse-proxy / TLS 종단(nginx, Caddy, Traefik, Cloudflare
Tunnel) 관련 내용은 [`docs/operator-guide.md`](docs/operator-guide.md) §Topology에 있습니다.

### 선택: 암호화 DB 백업 + 데이터 export 활성화

```bash
# Master passphrase (env-only — never write into config.toml).
export OPENDRAY_BACKUP_KEY="$(openssl rand -base64 32)"
export OPENDRAY_BACKUP_ENABLED=1

# pg_dump / pg_restore must match the server's major version. On
# Apple Silicon dev machines pointing at a PG17 server:
export OPENDRAY_BACKUP_PG_DUMP_PATH=/opt/homebrew/opt/postgresql@17/bin/pg_dump
export OPENDRAY_BACKUP_PG_RESTORE_PATH=/opt/homebrew/opt/postgresql@17/bin/pg_restore
```

opendray를 재시작하면 사이드바에 암호화된 PostgreSQL dump + restore를 위한
Backups 페이지(`/backups`)와, zip 번들 데이터 export + import를 위한
`/export`가 생깁니다. 전체 라이프사이클은 [`docs/operator-guide.md`](docs/operator-guide.md) §Backup을 참고하세요.

웹 번들 전체를 단일 Go 바이너리가 들고 다닙니다 — 런타임에 Node 런타임이
필요 없고, 별도의 정적 파일 서버도, Caddy/nginx도 필요 없습니다.
Cloudflare Tunnel이 `:8770` 앞에서 TLS를 종단합니다.

## 레이아웃

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

## 웹 frontend

`app/web/`는 단일 SPA를 `internal/web/dist/`로 빌드하고, Go 바이너리가 이를
embed해 `/admin/*`에서 서빙합니다. `:5173`의 Vite 개발 서버는 HMR 기반
개발을 위해 `/api`를 `:8770`으로 프록시합니다.

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

프론트엔드 스택(React + Vite + Tailwind v4 + shadcn/ui + TanStack Router/Query +
Zustand + xterm.js) 및 W 마일스톤별 노트는 [`app/web/README.md`](app/web/README.md)에서
확인할 수 있습니다.

## 문서

- [`docs/getting-started.md`](docs/getting-started.md) — 처음이라면 **여기서 시작**: 감싸는 CLI 설치와 Postgres 부트스트랩을 포함해 15분 만에 첫 세션까지
- [`docs/quickstart.md`](docs/quickstart.md) — 5분 개발 환경 (구성 요소를 이미 안다고 가정)
- [`docs/operator-guide.md`](docs/operator-guide.md) — 프로덕션급 셋업을 위한 deploy + 운영 레퍼런스
- [`docs/integration-guide.md`](docs/integration-guide.md) — 어떤 언어로든 외부 통합을 작성하는 방법
- [`VERSIONING.md`](VERSIONING.md) — 버저닝 전략 (major-as-generation)
- [`CHANGELOG.md`](CHANGELOG.md) — release 이력

## 테스트

```bash
go test -race ./...        # backend
cd app/web && pnpm build   # web (TS strict + vite production build)
```

엔드투엔드 smoke flow는 마일스톤별 커밋 메시지에서 추적됩니다.
Playwright harness는 후속 작업으로 계획되어 있습니다.

## v1과의 관계

v1 (`Opendray/opendray`)은 legacy 코드베이스이며, 현재는 아카이브된 상태입니다.
v2가 현재이자 활성 세대로 — feature-complete이고 개발이 진행되는 유일한 브랜치입니다.
v1의 16개 builtin 중 4개가 v2 backend로 이주했으며, 나머지는 클라이언트 측
기능, 채널 어댑터, 통합 API 소비자로 옮겨갔습니다.

## 라이선스

Apache 2.0 — [`LICENSE`](LICENSE) 참고. (v1은 MIT였으며, v2는 별도로
라이선스됩니다.)
