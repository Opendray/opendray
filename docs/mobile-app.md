# Mobile app — build & install

The opendray mobile app (`app/mobile/`) is a **control client**, not a
second gateway. It does the same job as the web admin at `/admin/`:
spawn and drive sessions, manage channels and integrations, browse
memory, read git hosts. The agents themselves keep running on your
gateway host — the phone just attaches to them.

Because of that, the app is useless on its own: it connects to a
**running opendray gateway** over HTTPS. Get the gateway up first
([getting-started](getting-started.md)), then build the app and point
it at your gateway URL.

> **Why no App Store / Play Store download?**
> opendray is self-hosted, single-tenant software. A store build would
> have to bake in *someone's* backend, which is exactly what opendray
> is not. So you build the app yourself, signed with your own identity,
> and it talks only to your gateway. The two supported paths below are
> **(A)** an Android APK you sideload, and **(B)** an iOS build you
> install through Xcode.

---

## Step 0 — make the gateway reachable from the phone

The app talks to the gateway over the network, so the phone has to be
able to reach it.

| Scenario | What to enter as Gateway URL |
|---|---|
| Phone on the same LAN as the gateway | `http://<gateway-lan-ip>:8770` (e.g. `http://192.168.1.50:8770`) |
| Gateway behind a reverse proxy with TLS | `https://opendray.yourdomain.com` |
| Off-LAN access (cellular, travelling) | A public HTTPS endpoint — Cloudflare Tunnel, Tailscale, or an nginx/Caddy reverse proxy |

> **Don't expose `:8770` raw to the internet.** Put TLS and an
> ingress in front of it. Cloudflare Tunnel is the lowest-friction
> option (no port-forwarding, no public IP). nginx / Caddy snippets —
> including the **WebSocket upgrade headers** the Sessions terminal
> needs — are in [operator-guide §Topology](operator-guide.md#topology).

Verify reachability from the phone *before* building, e.g. open the
Gateway URL in the phone's browser — you should get the web admin
login page.

---

## Step 1 — install the Flutter toolchain

The app is built with Flutter. You need it on the machine that does the
build (not the phone).

```sh
# Follow https://docs.flutter.dev/get-started/install for your OS.
flutter --version          # need 3.41+ (Dart SDK ^3.11)
flutter doctor             # resolve any ✗ for the platform you target
```

`flutter doctor` is the gate: it tells you exactly what's missing for
Android (Android SDK + a device/emulator) or iOS (Xcode + CocoaPods).
Fix the ✗ lines for your target platform before continuing.

Fetch dependencies once:

```sh
cd app/mobile
flutter pub get
```

---

## Step 2A — Android: build an APK and sideload it

This is the simplest path — no developer account, no store.

### Build the APK

```sh
cd app/mobile

# Single universal APK (easiest to share / sideload):
flutter build apk --release

# — or — smaller, per-architecture APKs (pick the one for your phone):
flutter build apk --release --split-per-abi
```

Output lands in:

```
app/mobile/build/app/outputs/flutter-apk/app-release.apk
```

> **Signing note.** Out of the box, the release build is signed with
> the **debug keystore** (see the `TODO` in
> `android/app/build.gradle.kts`). That's fine for personal sideloading.
> If you want a proper upload key (required for Play Store, and good
> hygiene for a build you'll keep updating), follow
> [Flutter — Sign the app](https://docs.flutter.dev/deployment/android#sign-the-app)
> and add a `signingConfig` for `release`.

### Get the APK onto the phone

Pick whichever is convenient:

```sh
# If the phone is plugged in with USB debugging on, install directly:
flutter install                 # builds + installs to the attached device
# or, with an existing APK:
adb install -r build/app/outputs/flutter-apk/app-release.apk
```

Or transfer the `.apk` file to the phone (AirDrop-equivalent, a file
share, a download link, email to yourself) and tap it. Android will ask
you to allow **"Install unknown apps"** for whichever app is opening the
file (Files, Chrome, etc.) — grant it, then confirm the install.

The app appears as **Opendray** (`io.opendray.opendray`).

---

## Step 2B — iOS: build and install through Xcode

iOS has no sideload-an-APK equivalent — every install is code-signed.
You need a **Mac with Xcode** and an **Apple ID**. A free Apple ID
works (the app is re-signed every 7 days; you reinstall when the
provisioning profile expires). A paid Apple Developer account
(US$99/yr) gives year-long profiles and TestFlight.

### One-time signing setup

```sh
cd app/mobile
flutter pub get
open ios/Runner.xcworkspace        # open the WORKSPACE, not the .xcodeproj
```

In Xcode:

1. Select the **Runner** target → **Signing & Capabilities** tab.
2. Tick **Automatically manage signing**.
3. **Team**: pick your Apple ID team (add your Apple ID under
   Xcode → Settings → Accounts if it's not listed).
4. **Bundle Identifier**: it ships as `io.opendray.opendray`. With a
   free Apple ID this exact ID may already be taken on Apple's side —
   if Xcode shows a provisioning error, change it to something unique
   like `io.opendray.opendray.<yourname>`.

### Build & install onto the iPhone

1. Connect the iPhone via USB; **trust** the computer when prompted.
2. Enable **Developer Mode** on the phone:
   Settings → Privacy & Security → Developer Mode → on → reboot.
3. In Xcode's device dropdown (top bar), select your iPhone.
4. Press **▶ Run** (or `⌘R`). Xcode builds, signs, and installs.

Or drive it from the CLI and let Xcode handle signing:

```sh
flutter run --release -d <device-id>     # `flutter devices` lists ids
```

### First launch on the device

iOS won't run an app signed by a personal team until you trust the
developer profile:

- On the phone: **Settings → General → VPN & Device Management →**
  your Apple ID → **Trust**.

The app appears as **Opendray** on the home screen.

> **Free Apple ID expiry.** After ~7 days the app stops launching
> ("could not verify app"). Re-run the build from Xcode to refresh the
> profile. A paid account avoids this.

---

## Step 3 — connect the app to your gateway

First launch shows the onboarding screen:

1. **Gateway URL** — enter the URL from Step 0
   (e.g. `https://opendray.yourdomain.com`). Tap **Continue**.
2. **Sign in** — `admin` + your admin password (the one you set in
   `[admin].password`, or changed to afterward).

That's it — you land on the same surfaces as the web admin: Sessions,
Channels, Integrations, Memory, Git, Settings.

To point the app at a different gateway later, tap **Change** on the
login screen (or Settings → server) and re-enter the URL.

---

## Updating the app

There's no auto-update — you reinstall after pulling new code:

```sh
git pull
cd app/mobile
flutter pub get

# Android:
flutter build apk --release      # then sideload / `flutter install`

# iOS:
open ios/Runner.xcworkspace      # ▶ Run again, or `flutter run --release`
```

The app's own version string lives in `app/mobile/pubspec.yaml`
(`version: <semver>+<build>`).

---

## Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| Onboarding "could not connect" | Phone can't reach the Gateway URL | Open the URL in the phone's browser; fix LAN IP / tunnel / TLS first (Step 0) |
| Login works but Sessions terminal never connects | Reverse proxy is dropping the WebSocket upgrade | Add WS headers — [operator-guide §Topology](operator-guide.md#topology) |
| Android blocks the install | "Install unknown apps" not granted | Allow it for the app that opens the `.apk` (Files / Chrome) |
| iOS "Untrusted Developer" on launch | Personal-team profile not trusted yet | Settings → General → VPN & Device Management → Trust |
| iOS "Unable to install / signing" in Xcode | Bundle ID clash with a free Apple ID | Change Bundle Identifier to `io.opendray.opendray.<yourname>` |
| iOS app stops opening after a week | Free Apple ID profile expired (7 days) | Re-run from Xcode, or use a paid account |
| `flutter doctor` shows ✗ for your platform | Missing Android SDK / Xcode / CocoaPods | Follow the exact line `flutter doctor` prints |

---

## See also

- [getting-started.md](getting-started.md) — stand up the gateway the app connects to
- [operator-guide.md](operator-guide.md) — reverse proxy / tunnel topology for off-LAN access
- [Flutter — build & release Android](https://docs.flutter.dev/deployment/android)
- [Flutter — build & release iOS](https://docs.flutter.dev/deployment/ios)
