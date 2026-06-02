# モバイルアプリ — ビルドとインストール

opendray モバイルアプリ（`app/mobile/`）は **コントロールクライアント** であり、
2 台目のゲートウェイではありません。`/admin/` の web 管理画面と同じ仕事をします:
セッションの起動と操作、チャンネルと連携の管理、メモリの閲覧、
git ホストの読み取り。エージェント自体はゲートウェイホスト上で動き続け、
スマートフォンはそれにアタッチするだけです。

そのため、アプリ単体では役に立ちません: HTTPS 経由で
**動作中の opendray ゲートウェイ** に接続します。まずゲートウェイを立ち上げ
（[getting-started](getting-started.md)）、その後アプリをビルドして
ゲートウェイ URL を指し示します。

> **なぜ App Store / Play Store からダウンロードできないのか？**
> opendray はセルフホストかつシングルテナントのソフトウェアです。ストア配布版は
> *誰かの* バックエンドを焼き込まなければならず、それはまさに opendray が
> 目指していないものです。そのため、アプリは自分でビルドし、自分のアイデンティティで
> 署名し、自分のゲートウェイとだけ通信します。以下にサポートされている 2 つの経路があります:
> **(A)** サイドロードする Android APK、および **(B)** Xcode 経由でインストールする iOS ビルドです。

---

## ステップ 0 — スマートフォンからゲートウェイに到達できるようにする

アプリはネットワーク経由でゲートウェイと通信するため、
スマートフォンがゲートウェイに到達できる必要があります。

| シナリオ | Gateway URL に入力する値 |
|---|---|
| スマートフォンがゲートウェイと同じ LAN 上にある | `http://<gateway-lan-ip>:8770`（例: `http://192.168.1.50:8770`） |
| ゲートウェイが TLS 付きのリバースプロキシの背後にある | `https://opendray.yourdomain.com` |
| LAN 外からのアクセス（モバイル回線、外出時） | パブリックな HTTPS エンドポイント — Cloudflare Tunnel、Tailscale、または nginx/Caddy リバースプロキシ |

> **`:8770` をインターネットに生のまま公開しないでください。** その前段に TLS と
> ingress を配置してください。Cloudflare Tunnel が最も手間のかからない選択肢です
> （ポートフォワーディング不要、パブリック IP 不要）。nginx / Caddy のスニペット —
> Sessions ターミナルが必要とする **WebSocket アップグレードヘッダー** を含む —
> は [operator-guide §Topology](operator-guide.md#topology) にあります。

ビルドする *前に* スマートフォンから到達性を確認してください。例えば
スマートフォンのブラウザで Gateway URL を開きます — web 管理画面のログインページが
表示されるはずです。

---

## ステップ 1 — Flutter ツールチェーンをインストールする

アプリは Flutter でビルドされています。ビルドを行うマシン（スマートフォンではない）に
インストールする必要があります。

```sh
# お使いの OS については https://docs.flutter.dev/get-started/install に従ってください。
flutter --version          # 3.41+ が必要（Dart SDK ^3.11）
flutter doctor             # 対象プラットフォームの ✗ をすべて解消する
```

`flutter doctor` がゲートになります: Android（Android SDK + デバイス/エミュレータ）
または iOS（Xcode + CocoaPods）に何が不足しているかを正確に教えてくれます。
続行する前に、対象プラットフォームの ✗ 行を修正してください。

依存関係を一度取得します:

```sh
cd app/mobile
flutter pub get
```

---

## ステップ 2A — Android: APK をビルドしてサイドロードする

これが最もシンプルな経路です — 開発者アカウントもストアも不要です。

### APK をビルドする

```sh
cd app/mobile

# 単一のユニバーサル APK（共有 / サイドロードが最も簡単）:
flutter build apk --release

# — または — アーキテクチャごとの小さい APK（スマートフォンに合うものを選択）:
flutter build apk --release --split-per-abi
```

出力先:

```
app/mobile/build/app/outputs/flutter-apk/app-release.apk
```

> **署名についての注意。** 初期状態では、リリースビルドは **デバッグキーストア** で
> 署名されます（`android/app/build.gradle.kts` の `TODO` を参照）。これは個人での
> サイドロードには問題ありません。適切なアップロードキー（Play Store には必須であり、
> 継続的に更新するビルドには良い習慣）が欲しい場合は、
> [Flutter — Sign the app](https://docs.flutter.dev/deployment/android#sign-the-app)
> に従って `release` 用の `signingConfig` を追加してください。

### APK をスマートフォンに入れる

都合の良い方法を選んでください:

```sh
# スマートフォンが USB デバッグを有効にして接続されている場合、直接インストール:
flutter install                 # ビルドして接続中のデバイスにインストール
# または、既存の APK を使う場合:
adb install -r build/app/outputs/flutter-apk/app-release.apk
```

または `.apk` ファイルをスマートフォンに転送し（AirDrop 相当、ファイル共有、
ダウンロードリンク、自分宛のメール）、タップします。Android は、ファイルを開くアプリ
（ファイル、Chrome など）に対して **「不明なアプリのインストール」** を許可するよう
求めてきます — 許可してから、インストールを確認してください。

アプリは **Opendray**（`io.opendray.opendray`）として表示されます。

---

## ステップ 2B — iOS: Xcode 経由でビルドしてインストールする

iOS には APK をサイドロードする相当の手段はありません — すべてのインストールはコード署名されます。
**Xcode を搭載した Mac** と **Apple ID** が必要です。無料の Apple ID でも
動作します（アプリは 7 日ごとに再署名され、プロビジョニングプロファイルが
期限切れになったら再インストールします）。有料の Apple Developer アカウント
（年間 US$99）なら、1 年間有効なプロファイルと TestFlight が利用できます。

### 一度だけの署名セットアップ

```sh
cd app/mobile
flutter pub get
open ios/Runner.xcworkspace        # .xcodeproj ではなく WORKSPACE を開く
```

Xcode で:

1. **Runner** ターゲットを選択 → **Signing & Capabilities** タブ。
2. **Automatically manage signing** にチェックを入れる。
3. **Team**: あなたの Apple ID チームを選択する（リストにない場合は
   Xcode → Settings → Accounts で Apple ID を追加する）。
4. **Bundle Identifier**: 初期値は `io.opendray.opendray` です。無料の
   Apple ID では、この正確な ID が Apple 側ですでに使用されている場合があります —
   Xcode がプロビジョニングエラーを表示する場合は、`io.opendray.opendray.<yourname>`
   のような一意なものに変更してください。

### iPhone へのビルドとインストール

1. iPhone を USB で接続し、求められたらコンピュータを **信頼** する。
2. スマートフォンで **デベロッパモード** を有効にする:
   設定 → プライバシーとセキュリティ → デベロッパモード → オン → 再起動。
3. Xcode のデバイスドロップダウン（上部バー）で iPhone を選択する。
4. **▶ Run**（または `⌘R`）を押す。Xcode がビルド、署名、インストールを行う。

または CLI から実行し、署名を Xcode に任せることもできます:

```sh
flutter run --release -d <device-id>     # `flutter devices` で id を一覧表示
```

### デバイスでの初回起動

iOS は、開発者プロファイルを信頼するまで、個人チームで署名されたアプリを実行しません:

- スマートフォンで: **設定 → 一般 → VPN とデバイス管理 →**
  あなたの Apple ID → **信頼**。

アプリはホーム画面に **Opendray** として表示されます。

> **無料 Apple ID の有効期限。** 約 7 日後、アプリは起動しなくなります
> （「App を検証できませんでした」）。Xcode からビルドを再実行して
> プロファイルを更新してください。有料アカウントならこれを回避できます。

---

## ステップ 3 — アプリをゲートウェイに接続する

初回起動時にオンボーディング画面が表示されます:

1. **Gateway URL** — ステップ 0 の URL を入力します
   （例: `https://opendray.yourdomain.com`）。**Continue** をタップします。
2. **Sign in** — `admin` + 管理者パスワード（`[admin].password` で設定したもの、
   またはその後変更したもの）。

これで完了です — web 管理画面と同じ画面に到達します: Sessions、
Channels、Integrations、Memory、Git、Settings。

後でアプリを別のゲートウェイに向けるには、ログイン画面の **Change** をタップ
（または Settings → server）し、URL を入力し直してください。

---

## アプリを更新する

自動更新はありません — 新しいコードを取得した後に再インストールします:

```sh
git pull
cd app/mobile
flutter pub get

# Android:
flutter build apk --release      # その後サイドロード / `flutter install`

# iOS:
open ios/Runner.xcworkspace      # 再度 ▶ Run、または `flutter run --release`
```

アプリ自体のバージョン文字列は `app/mobile/pubspec.yaml` にあります
（`version: <semver>+<build>`）。

---

## トラブルシューティング

| 症状 | 原因 | 対処 |
|---|---|---|
| オンボーディングで「接続できませんでした」 | スマートフォンが Gateway URL に到達できない | スマートフォンのブラウザで URL を開き、まず LAN IP / トンネル / TLS を修正する（ステップ 0） |
| ログインはできるが Sessions ターミナルが接続しない | リバースプロキシが WebSocket アップグレードを落としている | WS ヘッダーを追加する — [operator-guide §Topology](operator-guide.md#topology) |
| Android がインストールをブロックする | 「不明なアプリのインストール」が許可されていない | `.apk` を開くアプリ（ファイル / Chrome）に対して許可する |
| 起動時に iOS が「信頼されていないデベロッパ」と表示する | 個人チームのプロファイルがまだ信頼されていない | 設定 → 一般 → VPN とデバイス管理 → 信頼 |
| Xcode で iOS が「インストール / 署名できません」と表示する | 無料 Apple ID との Bundle ID の衝突 | Bundle Identifier を `io.opendray.opendray.<yourname>` に変更する |
| 1 週間後に iOS アプリが開かなくなる | 無料 Apple ID のプロファイルが期限切れ（7 日） | Xcode から再実行する、または有料アカウントを使用する |
| `flutter doctor` が対象プラットフォームに ✗ を表示する | Android SDK / Xcode / CocoaPods が不足 | `flutter doctor` が表示する正確な行に従う |

---

## 関連項目

- [getting-started.md](getting-started.md) — アプリが接続するゲートウェイを立ち上げる
- [operator-guide.md](operator-guide.md) — LAN 外アクセス向けのリバースプロキシ / トンネルトポロジー
- [Flutter — build & release Android](https://docs.flutter.dev/deployment/android)
- [Flutter — build & release iOS](https://docs.flutter.dev/deployment/ios)
