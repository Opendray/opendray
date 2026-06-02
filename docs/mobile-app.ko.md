# 모바일 앱 — 빌드 및 설치

opendray 모바일 앱(`app/mobile/`)은 **컨트롤 클라이언트**이며,
두 번째 게이트웨이가 아닙니다. `/admin/`의 웹 어드민과 동일한 일을 합니다:
세션을 spawn하고 조작하며, 채널과 통합을 관리하고, 메모리를 탐색하고,
git 호스트를 읽습니다. 에이전트 자체는 게이트웨이 호스트에서 계속
실행됩니다 — 휴대폰은 거기에 attach할 뿐입니다.

그렇기 때문에 앱은 단독으로는 쓸모가 없습니다: HTTPS를 통해
**실행 중인 opendray 게이트웨이**에 연결됩니다. 먼저 게이트웨이를
올린 뒤([getting-started](getting-started.md)), 앱을 빌드하고
게이트웨이 URL을 가리키도록 설정하세요.

> **왜 App Store / Play Store 다운로드가 없나요?**
> opendray는 self-hosted, single-tenant 소프트웨어입니다. 스토어 빌드는
> *누군가의* 백엔드를 박아 넣어야 하는데, 그것이야말로 opendray가
> 지향하지 않는 바입니다. 그래서 앱을 직접 빌드하고, 본인의 신원으로
> 서명하며, 오직 자신의 게이트웨이하고만 통신합니다. 아래 지원되는 두
> 경로는 **(A)** 사이드로드하는 Android APK와 **(B)** Xcode를 통해
> 설치하는 iOS 빌드입니다.

---

## Step 0 — 휴대폰에서 게이트웨이에 도달 가능하게 만들기

앱은 네트워크를 통해 게이트웨이와 통신하므로, 휴대폰이 게이트웨이에
도달할 수 있어야 합니다.

| 시나리오 | Gateway URL로 입력할 값 |
|---|---|
| 휴대폰이 게이트웨이와 같은 LAN에 있음 | `http://<gateway-lan-ip>:8770` (예: `http://192.168.1.50:8770`) |
| 게이트웨이가 TLS를 갖춘 reverse proxy 뒤에 있음 | `https://opendray.yourdomain.com` |
| LAN 외부 접근 (셀룰러, 이동 중) | 공개 HTTPS 엔드포인트 — Cloudflare Tunnel, Tailscale, 또는 nginx/Caddy reverse proxy |

> **`:8770`을 인터넷에 그대로 노출하지 마세요.** 앞에 TLS와 ingress를
> 두세요. Cloudflare Tunnel이 가장 마찰이 적은 옵션입니다(포트 포워딩
> 불필요, 공개 IP 불필요). nginx / Caddy 스니펫은 — Sessions 터미널에
> 필요한 **WebSocket upgrade 헤더**를 포함해 —
> [operator-guide §Topology](operator-guide.md#topology)에 있습니다.

빌드하기 *전에* 휴대폰에서 도달 가능한지 확인하세요. 예를 들어
휴대폰 브라우저에서 Gateway URL을 열어 보면 웹 어드민 로그인 페이지가
나타나야 합니다.

---

## Step 1 — Flutter 툴체인 설치

앱은 Flutter로 빌드됩니다. 빌드를 수행하는 머신에 설치해야 합니다
(휴대폰이 아닙니다).

```sh
# 사용 중인 OS에 맞춰 https://docs.flutter.dev/get-started/install 를 따르세요.
flutter --version          # 3.41+ 필요 (Dart SDK ^3.11)
flutter doctor             # 대상 플랫폼의 ✗ 항목을 모두 해결
```

`flutter doctor`가 관문입니다: Android(Android SDK + 디바이스/에뮬레이터)
또는 iOS(Xcode + CocoaPods)에 무엇이 빠졌는지 정확히 알려줍니다.
계속하기 전에 대상 플랫폼의 ✗ 라인을 해결하세요.

의존성을 한 번 가져옵니다:

```sh
cd app/mobile
flutter pub get
```

---

## Step 2A — Android: APK 빌드 후 사이드로드

가장 간단한 경로입니다 — 개발자 계정도, 스토어도 필요 없습니다.

### APK 빌드

```sh
cd app/mobile

# 단일 universal APK (공유 / 사이드로드가 가장 쉬움):
flutter build apk --release

# — 또는 — 더 작은 아키텍처별 APK (자신의 휴대폰에 맞는 것을 선택):
flutter build apk --release --split-per-abi
```

출력 위치:

```
app/mobile/build/app/outputs/flutter-apk/app-release.apk
```

> **서명 참고.** 기본 상태에서 release 빌드는 **debug keystore**로
> 서명됩니다(`android/app/build.gradle.kts`의 `TODO` 참고). 개인
> 사이드로딩에는 문제없습니다. 제대로 된 upload key를 원한다면(Play
> Store에 필요하며, 계속 업데이트할 빌드라면 좋은 위생 습관입니다)
> [Flutter — Sign the app](https://docs.flutter.dev/deployment/android#sign-the-app)
> 을 따르고 `release`용 `signingConfig`를 추가하세요.

### APK를 휴대폰에 넣기

편한 방법을 고르세요:

```sh
# 휴대폰이 USB로 연결되어 있고 USB 디버깅이 켜져 있으면, 바로 설치:
flutter install                 # 빌드 후 연결된 디바이스에 설치
# 또는, 기존 APK로:
adb install -r build/app/outputs/flutter-apk/app-release.apk
```

또는 `.apk` 파일을 휴대폰으로 전송하고(AirDrop 상당, 파일 공유,
다운로드 링크, 본인에게 이메일) 탭하세요. Android는 파일을 여는 앱
(Files, Chrome 등)에 대해 **"출처를 알 수 없는 앱 설치"** 허용을
요청합니다 — 허용한 뒤 설치를 확인하세요.

앱은 **Opendray**(`io.opendray.opendray`)로 나타납니다.

---

## Step 2B — iOS: Xcode를 통해 빌드 및 설치

iOS에는 APK 사이드로드에 해당하는 것이 없습니다 — 모든 설치는
코드 서명됩니다. **Xcode가 설치된 Mac**과 **Apple ID**가 필요합니다.
무료 Apple ID도 작동합니다(앱은 7일마다 재서명되며, provisioning
profile이 만료되면 다시 설치합니다). 유료 Apple Developer 계정
(연 US$99)은 1년짜리 profile과 TestFlight을 제공합니다.

### 일회성 서명 설정

```sh
cd app/mobile
flutter pub get
open ios/Runner.xcworkspace        # .xcodeproj가 아니라 WORKSPACE를 여세요
```

Xcode에서:

1. **Runner** 타겟 선택 → **Signing & Capabilities** 탭.
2. **Automatically manage signing** 체크.
3. **Team**: 자신의 Apple ID 팀 선택(목록에 없으면
   Xcode → Settings → Accounts에서 Apple ID 추가).
4. **Bundle Identifier**: `io.opendray.opendray`로 출하됩니다. 무료
   Apple ID에서는 이 정확한 ID가 Apple 측에서 이미 사용 중일 수
   있습니다 — Xcode가 provisioning 오류를 표시하면
   `io.opendray.opendray.<yourname>` 같은 고유한 값으로 변경하세요.

### iPhone에 빌드 및 설치

1. iPhone을 USB로 연결하고, 요청 시 컴퓨터를 **신뢰**하세요.
2. 휴대폰에서 **개발자 모드**를 활성화하세요:
   Settings → Privacy & Security → Developer Mode → 켜기 → 재부팅.
3. Xcode의 디바이스 드롭다운(상단 바)에서 자신의 iPhone을 선택하세요.
4. **▶ Run**(또는 `⌘R`)을 누르세요. Xcode가 빌드, 서명, 설치합니다.

또는 CLI에서 구동하고 서명은 Xcode에 맡기세요:

```sh
flutter run --release -d <device-id>     # `flutter devices`가 id 목록을 출력
```

### 디바이스에서 첫 실행

iOS는 개인 팀이 서명한 앱을, 개발자 profile을 신뢰하기 전까지는
실행하지 않습니다:

- 휴대폰에서: **Settings → General → VPN & Device Management →**
  자신의 Apple ID → **Trust**.

앱은 홈 화면에 **Opendray**로 나타납니다.

> **무료 Apple ID 만료.** 약 7일 후 앱이 실행되지 않습니다
> ("could not verify app"). Xcode에서 빌드를 다시 실행해 profile을
> 갱신하세요. 유료 계정은 이를 피할 수 있습니다.

---

## Step 3 — 앱을 게이트웨이에 연결

첫 실행 시 온보딩 화면이 나타납니다:

1. **Gateway URL** — Step 0의 URL을 입력하세요
   (예: `https://opendray.yourdomain.com`). **Continue**를 탭하세요.
2. **Sign in** — `admin` + 어드민 비밀번호(`[admin].password`에서
   설정했거나 이후 변경한 값).

이게 전부입니다 — 웹 어드민과 동일한 surface로 이동합니다: Sessions,
Channels, Integrations, Memory, Git, Settings.

나중에 앱을 다른 게이트웨이로 가리키려면, 로그인 화면에서 **Change**를
탭하거나(또는 Settings → server) URL을 다시 입력하세요.

---

## 앱 업데이트

자동 업데이트는 없습니다 — 새 코드를 pull한 뒤 다시 설치합니다:

```sh
git pull
cd app/mobile
flutter pub get

# Android:
flutter build apk --release      # 이후 사이드로드 / `flutter install`

# iOS:
open ios/Runner.xcworkspace      # ▶ Run 다시 실행, 또는 `flutter run --release`
```

앱 자체의 버전 문자열은 `app/mobile/pubspec.yaml`에 있습니다
(`version: <semver>+<build>`).

---

## 트러블슈팅

| 증상 | 원인 | 해결 |
|---|---|---|
| 온보딩 "could not connect" | 휴대폰이 Gateway URL에 도달하지 못함 | 휴대폰 브라우저에서 URL을 열고, LAN IP / 터널 / TLS를 먼저 고치세요 (Step 0) |
| 로그인은 되지만 Sessions 터미널이 연결되지 않음 | reverse proxy가 WebSocket upgrade를 드롭함 | WS 헤더를 추가하세요 — [operator-guide §Topology](operator-guide.md#topology) |
| Android가 설치를 차단함 | "출처를 알 수 없는 앱 설치"가 허용되지 않음 | `.apk`를 여는 앱(Files / Chrome)에 대해 허용하세요 |
| 실행 시 iOS "Untrusted Developer" | 개인 팀 profile이 아직 신뢰되지 않음 | Settings → General → VPN & Device Management → Trust |
| Xcode에서 iOS "Unable to install / signing" | 무료 Apple ID와 Bundle ID 충돌 | Bundle Identifier를 `io.opendray.opendray.<yourname>`로 변경 |
| 일주일 후 iOS 앱이 열리지 않음 | 무료 Apple ID profile 만료 (7일) | Xcode에서 다시 실행하거나, 유료 계정을 사용하세요 |
| `flutter doctor`가 플랫폼에 대해 ✗를 표시함 | Android SDK / Xcode / CocoaPods 누락 | `flutter doctor`가 출력하는 정확한 라인을 따르세요 |

---

## 함께 보기

- [getting-started.md](getting-started.md) — 앱이 연결할 게이트웨이를 올리기
- [operator-guide.md](operator-guide.md) — LAN 외부 접근을 위한 reverse proxy / 터널 토폴로지
- [Flutter — build & release Android](https://docs.flutter.dev/deployment/android)
- [Flutter — build & release iOS](https://docs.flutter.dev/deployment/ios)
