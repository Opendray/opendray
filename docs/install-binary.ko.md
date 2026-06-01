# 사전 빌드 바이너리로 설치 및 실행

설치 마법사가 머신에 개입하지 않고, `opendray` 바이너리만 단독으로 갖고 싶을 때의 경로입니다. 다음 경우에 적합합니다:

- **`npm install -g opendray` / `npx opendray`** — npm 패키지에는 공식 Go release 바이너리가 포함되어 있습니다([README → npm / npx](../README.ko.md#npm--npx-node--18) 참고).
- **릴리즈 아카이브 다운로드** — [Releases 페이지](https://github.com/Opendray/opendray/releases)에서 `opendray_*_<os>_<arch>.tar.gz`를 받아 쓰세요.
- **스크립트 / 일회성 환경** — CI 러너, golden 이미지, 구성 관리(Ansible, Nix, Docker), 또는 이미 자체 Postgres와 프로세스 supervisor를 운영 중인 호스트.

바이너리 하나가 *전체* 게이트웨이입니다 — 웹 어드민 SPA가 내장되어 있기 때문에 Node 런타임도, 별도의 정적 파일 서버도, 빌드할 것도 없습니다. **바이너리가 하지 않는** 것은 자동 설정입니다. 이것이 트레이드오프입니다: PostgreSQL 데이터베이스와 프로세스를 계속 실행할 방법을 직접 제공하는 대신, 아무것도 몰래 설치·설정·등록되지 않습니다.

> **모든 것을 자동으로 처리해 드리길 원하신다면?** 새 Linux / macOS 박스에서 원라인 설치 스크립트가 Postgres 프로비저닝, AI CLI 설치, 설정 작성, 서비스 등록을 약 5~10분 안에 마칩니다. [README → 원라인 설치 스크립트](../README.ko.md#원라인-설치-스크립트) 또는 수동 [getting-started.md](getting-started.md) 가이드를 참고하세요.

이 가이드는 "바이너리가 `PATH`에 있는 상태"에서 "실행 중인 게이트웨이"까지 다섯 단계로 안내하고, 이후 서비스로 계속 실행하는 방법을 설명합니다.

---

## Step 1 — 바이너리 확보

### npm을 통해 (Node ≥ 18이 있는 모든 OS)

```sh
npm install -g opendray        # 전역 설치, `opendray`를 PATH에 추가
# 또는 설치 없이:
npx opendray --help
# 또는 다른 패키지 매니저 사용:
pnpm add -g opendray
yarn global add opendray
```

`optionalDependencies`를 통해 올바른 플랫폼 바이너리(`opendray-{linux,darwin}-{x64,arm64}`)가 자동으로 선택됩니다 — `postinstall` 훅도 없고 설치 시 네트워크 호출도 없습니다. `--no-optional`을 **절대 전달하지 마세요**: 플랫폼 패키지를 건너뛰어 런처가 실행할 바이너리가 없는 상태가 됩니다.

### 릴리즈 아카이브를 통해

```sh
# Releases 페이지에서 OS/아키텍처에 맞는 아카이브를 선택한 뒤:
tar -xzf opendray_*_linux_amd64.tar.gz
sudo install -m 0755 opendray /usr/local/bin/opendray
```

### 확인

```sh
opendray version          # 버전, 커밋, 빌드 날짜 출력
opendray --help           # 모든 서브커맨드 목록
```

지원 플랫폼: **Linux** (x64, arm64)과 **macOS** (x64, arm64).
네이티브 Windows는 패키징되어 있지 않습니다 — WSL2를 사용해 Linux 경로로 진행하세요.

---

## Step 2 — pgvector가 설치된 PostgreSQL 15+ 제공

opendray는 모든 데이터(세션, 메모리, 감사 로그)를 PostgreSQL에 저장하며, 메모리 서브시스템은 [`pgvector`](https://github.com/pgvector/pgvector) 확장이 필요합니다. 지원 서버 버전: **15, 16, 17**.

이미 Postgres를 운영 중이라면, 데이터베이스와 CRUD 전용 role을 만들고 수퍼유저로 확장을 한 번 활성화합니다:

```sh
# 1. pgvector 설치 (호스트당 한 번).
#    Ubuntu/Debian:  sudo apt install postgresql-16-pgvector
#    macOS (brew):   brew install pgvector
#    기타 / 소스:    https://github.com/pgvector/pgvector#installation

# 2. 데이터베이스 + 프로젝트 전용 role 생성.
sudo -u postgres psql <<'SQL'
CREATE DATABASE opendray;
CREATE USER opendray WITH ENCRYPTED PASSWORD 'change-me';
GRANT ALL PRIVILEGES ON DATABASE opendray TO opendray;
SQL

# 3. 해당 데이터베이스 안에서 pgvector 활성화 (한 번만, 수퍼유저 필요).
sudo -u postgres psql -d opendray -c 'CREATE EXTENSION IF NOT EXISTS vector;'
sudo -u postgres psql -d opendray -c 'GRANT ALL ON SCHEMA public TO opendray;'
```

확장이 존재하면, opendray의 CRUD 전용 role이 수퍼유저 접근 없이도 마이그레이션을 실행합니다. **런타임에 opendray가 수퍼유저 role을 사용하게 하지 마세요** — 프로젝트 전용 계정을 부여하고 비밀번호는 별도로 교체하세요.

---

## Step 3 — 설정

opendray는 TOML 파일 **또는** 환경 변수(12-factor)만으로 설정을 읽습니다 — 환경 변수는 항상 파일보다 우선합니다. 필수 항목은 데이터베이스 URL 하나뿐이며, 나머지는 기본값이 있습니다.

### Option A — 환경 변수 (컨테이너 / 일회성 호스트에 적합)

```sh
export OPENDRAY_DATABASE_URL="postgres://opendray:change-me@127.0.0.1:5432/opendray?sslmode=disable"
export OPENDRAY_ADMIN_PASSWORD="$(openssl rand -base64 24)"   # 어드민 로그인
export OPENDRAY_LISTEN="127.0.0.1:8770"                       # 선택사항; 이것이 기본값
```

| 변수 | 필수 | 기본값 | 용도 |
|---|---|---|---|
| `OPENDRAY_DATABASE_URL` | **예** | — | Postgres DSN |
| `OPENDRAY_ADMIN_PASSWORD` | 권장 | — | 웹/모바일 어드민 비밀번호 |
| `OPENDRAY_ADMIN_USER` | 아니오 | `admin` | 어드민 사용자명 |
| `OPENDRAY_LISTEN` | 아니오 | `127.0.0.1:8770` | 바인드 주소 |
| `OPENDRAY_LOG_LEVEL` | 아니오 | `info` | `debug`/`info`/`warn`/`error` |
| `OPENDRAY_LOG_FORMAT` | 아니오 | `text` | `text`/`json` |

`-config` 플래그 없이 `opendray serve`를 실행하면 환경에서만 설정을 로드합니다.

### Option B — config.toml

```sh
curl -fsSLO https://raw.githubusercontent.com/Opendray/opendray/main/config.example.toml
mv config.example.toml config.toml
$EDITOR config.toml        # [database].url 과 [admin].password 설정
```

최소 편집 항목:

```toml
listen = "127.0.0.1:8770"

[database]
url = "postgres://opendray:change-me@127.0.0.1:5432/opendray?sslmode=disable"

[admin]
user     = "admin"
password = "use-a-real-password"
```

로깅, 세션 idle 감지, 백업, vault, MCP가 완전히 주석 처리된 파일은 [`config.example.toml`](../config.example.toml)을 참고하세요. 아래 명령어에 `-config config.toml`로 전달합니다. 공유 호스트에서는 시크릿을 TOML에 넣지 마세요 — `OPENDRAY_DATABASE_URL` / `OPENDRAY_ADMIN_PASSWORD`는 환경 변수로 설정하고 파일은 비공개가 필요 없는 상태로 유지하세요.

---

## Step 4 — 스키마 적용

```sh
opendray migrate                          # 환경 변수 전용 설정
# 또는
opendray migrate -config config.toml
```

멱등성 보장 — 스키마가 최신 상태라면 재실행해도 아무 변화가 없습니다. 첫 번째 `serve` 이전에 반드시 성공해야 합니다.

---

## Step 5 — 실행

```sh
opendray serve                            # 환경 변수 전용 설정
# 또는
opendray serve -config config.toml
```

이 명령은 **포그라운드**에서 실행됩니다(Ctrl-C로 중지). 이제 다음 URL을 사용할 수 있습니다:

| URL | 설명 |
|---|---|
| `http://127.0.0.1:8770/admin/` | 웹 어드민 — `admin` + 설정한 비밀번호로 로그인 |
| `http://127.0.0.1:8770/api/v1/...` | REST + WebSocket API |

완전히 작동하는 게이트웨이입니다. 빠른 테스트 이상의 용도라면, 재부팅 시 살아남고 크래시 시 자동으로 재시작하도록 supervisor 아래에서 실행하세요 — 다음 섹션에서 설명합니다.

---

## 서비스로 실행

`opendray serve`는 서비스 유닛의 시작 명령으로 그대로 사용할 수 있습니다.
opendray는 강화된 즉시 사용 가능한 유닛을 함께 제공합니다. 아래 단계는
[README → 프로덕션 deploy](../README.ko.md#프로덕션-deploy)와 동일하며, 그곳이 전체 부트스트랩, 샌드박싱 주의사항, reverse-proxy/TLS에 대한 권위 있는 레퍼런스입니다.

### Linux — systemd

저장소에는
[`deploy/systemd/opendray.service`](../deploy/systemd/opendray.service)에
강화된 유닛이 포함되어 있습니다
(`ExecStartPre`로 `migrate` 실행, `EnvironmentFile`로 시크릿 관리,
`on-failure` 재시작, syscall/파일시스템 샌드박싱).

```sh
# /usr/local/bin/opendray에 바이너리, 서비스 사용자, 상태 디렉터리 설정:
sudo useradd -r -s /usr/sbin/nologin -d /var/lib/opendray opendray
sudo install -d -o opendray -g opendray -m 0700 /var/lib/opendray

# 설정 파일(비시크릿) + 시크릿 파일(환경 변수, 모드 0640):
sudo install -D -m 0640 config.toml /etc/opendray/config.toml
sudo install -D -m 0640 -o root -g opendray /dev/null /etc/opendray/env.d/secrets
echo 'OPENDRAY_ADMIN_PASSWORD=use-a-real-password' | sudo tee -a /etc/opendray/env.d/secrets

# 유닛 설치 및 활성화:
sudo cp deploy/systemd/opendray.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now opendray
sudo systemctl status opendray
journalctl -u opendray -f --no-pager
```

systemd가 없는 경우? (systemd 없는 LXC, OpenRC, runit, s6, supervisord…) supervisor가 `opendray serve -config /etc/opendray/config.toml`을 실행하도록 지정하고, pre-start 단계로 `opendray migrate`를 한 번 실행하세요. [README → 프로덕션 deploy §B](../README.ko.md#option-b--바이너리-직접-실행--자체-프로세스-supervisor) 참고.

### macOS — launchd

저장소에는
[`deploy/launchd/com.opendray.opendray.plist`](../deploy/launchd/com.opendray.opendray.plist)에
LaunchDaemon이 포함되어 있습니다
(부팅 시 시작, 크래시 시 재시작, `/usr/local/var/log/opendray/`에 로그 기록).

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

재시작: `sudo launchctl kickstart -k system/com.opendray.opendray`.
언로드: `sudo launchctl bootout system/com.opendray.opendray`.

> 두 유닛 모두 — 시크릿 레이아웃과 `MemoryDenyWriteExecute`를 해제한 이유를 포함해 — 전체 문서는
> [`deploy/README.md`](../deploy/README.md)에 있습니다.

---

## 업데이트 유지

업데이트 방법은 설치 방법에 따라 다릅니다:

- **npm으로 설치한 경우** — 패키지 매니저로 업데이트합니다. `opendray update`는
  `node_modules` 안의 바이너리를 npm 모르게 교체하여 다음 설치 시 덮어씌워지므로, 이 경우에는 사용하지 마세요.

  ```sh
  npm install -g opendray@latest
  ```

- **릴리즈 다운로드 / 마법사 설치** — 바이너리가 자체 업데이트합니다
  (최신 릴리즈를 다운로드하고, SHA-256을 검증한 뒤, 원자적으로 교체합니다):

  ```sh
  opendray update --check          # 버전만 확인 (적용 없음)
  sudo opendray update --restart   # 적용 후 서비스 재시작
  ```

---

## 트러블슈팅

**`the matching platform package "opendray-…" was not installed`**
npm이 `--no-optional`로 실행되었거나 설치가 중단된 경우입니다. `npm install -g opendray`를 다시 실행하세요 (`--no-optional` 없이).

**`unsupported platform`**
npm 패키지는 Linux/macOS의 x64/arm64만 지원합니다. 다른 플랫폼에서는 소스에서 빌드하세요 — [quickstart.md](quickstart.md) 참고.

**`config: database.url is empty`**
`OPENDRAY_DATABASE_URL`과 `[database].url` 중 하나도 설정되어 있지 않습니다. Step 3에서 하나를 설정하세요.

**migrate/serve 시 `connection refused`**
Postgres가 실행 중이지 않거나 DSN이 잘못된 경우입니다. 서버가 올라와 있는지, DSN의 호스트/포트/자격증명이 맞는지 확인하세요.

**pgvector / `extension "vector" is not available`**
서버에 확장이 설치되어 있지 않거나, opendray 데이터베이스 안에서 활성화되지 않은 경우입니다. Step 2를 다시 진행하세요 (OS 패키지를 설치한 뒤, 수퍼유저로 `CREATE EXTENSION vector`를 실행합니다).

**포트가 이미 사용 중**
`OPENDRAY_LISTEN`(또는 config.toml의 `listen`)을 사용하지 않는 포트로 변경하세요.

---

## 다음 단계

- [README → 프로덕션 deploy](../README.ko.md#프로덕션-deploy) — 전체 deploy 레퍼런스(systemd / launchd / 자체 supervisor, 하드닝, reverse proxy)
- [`docs/operator-guide.md`](operator-guide.md) — 운영: reverse-proxy/TLS 토폴로지, 암호화 DB 백업, 데이터 export/import
- [`docs/integration-guide.md`](integration-guide.md) — REST + WebSocket API를 대상으로 외부 통합 구축
- [`docs/getting-started.md`](getting-started.md) — 직접 구성 요소를 조립하기보다 안내받고 싶다면 올인원 설정 가이드
