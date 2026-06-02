# opendray mobile

The Flutter control client for an opendray gateway — feature parity
with the web admin (Sessions, Channels, Integrations, Memory, Git,
Settings). It attaches to a **running gateway**; it is not a gateway
itself.

## Build & install

Full walkthrough — Android APK sideload and iOS install via Xcode,
plus making the gateway reachable from the phone:

- [`docs/mobile-app.md`](../../docs/mobile-app.md) (English)
- [`docs/mobile-app.zh.md`](../../docs/mobile-app.zh.md) (简体中文)

## Quick reference

```sh
flutter pub get

# Android
flutter build apk --release        # build/app/outputs/flutter-apk/app-release.apk
flutter install                    # build + install to an attached device

# iOS (needs a Mac + Xcode)
open ios/Runner.xcworkspace        # set a signing Team, then ▶ Run
flutter run --release -d <device>  # or drive from the CLI

# run against a dev gateway
flutter run
```

App identity: bundle id `io.opendray.opendray`, display name **Opendray**.
Version string lives in [`pubspec.yaml`](pubspec.yaml)
(`version: <semver>+<build>`).

New to Flutter? Start at the
[Flutter docs](https://docs.flutter.dev/).
