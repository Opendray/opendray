<div dir="rtl" lang="fa">

# اپ موبایل — build و نصب

اپ موبایل opendray (`app/mobile/`) یک **کلاینت کنترل** است، نه یک gateway دوم. همان کاری را می‌کند که web admin روی `/admin/` انجام می‌دهد: ساختن و راندن سشن‌ها، مدیریت channelها و integrationها، مرور حافظه، خواندن git hostها. خودِ agentها همچنان روی gateway host شما در حال اجرا می‌مانند — گوشی فقط به آن‌ها وصل می‌شود.

به همین خاطر، اپ به‌تنهایی به‌دردنخور است: از طریق HTTPS به یک **gateway فعالِ opendray** وصل می‌شود. اول gateway را بالا بیاورید ([getting-started](getting-started.md))، بعد اپ را build کنید و آن را به URL gateway خودتان نشانه بگیرید.

> **چرا دانلود از App Store / Play Store در کار نیست؟**
> opendray نرم‌افزاری self-hosted و single-tenant است. یک build فروشگاهی مجبور بود backend *یک نفرِ مشخص* را داخلش جا بدهد، و این دقیقاً همان چیزی است که opendray نیست. پس خودتان اپ را build می‌کنید، با هویت خودتان امضایش می‌کنید، و فقط با gateway شما حرف می‌زند. دو مسیر پشتیبانی‌شده در ادامه عبارت‌اند از **(الف)** یک APK اندرویدی که sideload می‌کنید، و **(ب)** یک build برای iOS که از طریق Xcode نصبش می‌کنید.

---

## گام ۰ — کاری کنید gateway از گوشی در دسترس باشد

اپ از طریق شبکه با gateway حرف می‌زند، پس گوشی باید بتواند به آن برسد.

| سناریو | چه چیزی به‌عنوان Gateway URL وارد کنید |
|---|---|
| گوشی روی همان LANِ gateway | `http://<gateway-lan-ip>:8770` (مثلاً `http://192.168.1.50:8770`) |
| gateway پشت یک reverse proxy با TLS | `https://opendray.yourdomain.com` |
| دسترسی خارج از LAN (دیتای موبایل، در سفر) | یک endpoint عمومی HTTPS — Cloudflare Tunnel، Tailscale، یا یک reverse proxy روی nginx/Caddy |

> **`:8770` را خام به اینترنت expose نکنید.** جلویش TLS و یک ingress بگذارید. Cloudflare Tunnel کم‌دردسرترین گزینه است (نه port-forwarding، نه IP عمومی). snippetهای nginx / Caddy — شامل **هدرهای WebSocket upgrade** که ترمینال سشن‌ها لازم دارد — در [operator-guide §Topology](operator-guide.md#topology) آمده‌اند.

دسترسی از روی گوشی را *پیش از* build تأیید کنید، مثلاً Gateway URL را در مرورگر گوشی باز کنید — باید صفحه ورود web admin را بگیرید.

---

## گام ۱ — toolchainِ Flutter را نصب کنید

اپ با Flutter ساخته می‌شود. آن را روی ماشینی که build را انجام می‌دهد لازم دارید (نه روی گوشی).

<div dir="ltr">

```sh
# برای OS خودتان https://docs.flutter.dev/get-started/install را دنبال کنید.
flutter --version          # need 3.41+ (Dart SDK ^3.11)
flutter doctor             # resolve any ✗ for the platform you target
```

</div>

`flutter doctor` همان دروازه است: دقیقاً به شما می‌گوید برای اندروید (Android SDK + یک device/emulator) یا iOS (Xcode + CocoaPods) چه چیزی کم است. قبل از ادامه، خطوط ✗ مربوط به پلتفرم هدف خود را رفع کنید.

یک بار dependencyها را بگیرید:

<div dir="ltr">

```sh
cd app/mobile
flutter pub get
```

</div>

---

## گام ۲الف — اندروید: یک APK بسازید و sideloadش کنید

این ساده‌ترین مسیر است — نه حساب developer، نه فروشگاه.

### ساختن APK

<div dir="ltr">

```sh
cd app/mobile

# یک APK یونیورسال (راحت‌ترین برای share / sideload):
flutter build apk --release

# — یا — APKهای کوچک‌تر و per-architecture (آن را که برای گوشی شماست بردارید):
flutter build apk --release --split-per-abi
```

</div>

خروجی اینجا می‌نشیند:

<div dir="ltr">

```
app/mobile/build/app/outputs/flutter-apk/app-release.apk
```

</div>

> **نکته امضا.** به‌صورت پیش‌فرض، build رِلیز با **debug keystore** امضا می‌شود (به `TODO` داخل `android/app/build.gradle.kts` نگاه کنید). این برای sideload شخصی اشکالی ندارد. اگر یک upload key درست‌وحسابی می‌خواهید (برای Play Store الزامی است، و برای buildی که قرار است مدام به‌روزش کنید بهداشت خوبی است)، [Flutter — Sign the app](https://docs.flutter.dev/deployment/android#sign-the-app) را دنبال کنید و یک `signingConfig` برای `release` اضافه کنید.

### رساندن APK به گوشی

هر کدام که راحت‌تر است را بردارید:

<div dir="ltr">

```sh
# اگر گوشی با USB debugging روشن وصل است، مستقیم نصب کنید:
flutter install                 # build می‌کند + روی device متصل نصب می‌کند
# یا، با یک APK موجود:
adb install -r build/app/outputs/flutter-apk/app-release.apk
```

</div>

یا فایل `.apk` را به گوشی منتقل کنید (معادل AirDrop، یک file share، یک لینک دانلود، ایمیل به خودتان) و رویش بزنید. اندروید از شما می‌خواهد برای هر اپی که فایل را باز می‌کند (Files، Chrome و غیره) اجازه **«Install unknown apps»** را بدهید — اجازه‌اش را بدهید، بعد نصب را تأیید کنید.

اپ با نام **Opendray** (`io.opendray.opendray`) ظاهر می‌شود.

---

## گام ۲ب — iOS: build و نصب از طریق Xcode

iOS معادل sideloadِ APK ندارد — هر نصب code-signed است. به یک **Mac با Xcode** و یک **Apple ID** نیاز دارید. یک Apple ID رایگان کار می‌کند (اپ هر ۷ روز دوباره امضا می‌شود؛ وقتی provisioning profile منقضی شد دوباره نصبش می‌کنید). یک حساب پولی Apple Developer (سالی ۹۹ دلار) profileهای یک‌ساله و TestFlight می‌دهد.

### تنظیم یک‌باره‌ی امضا

<div dir="ltr">

```sh
cd app/mobile
flutter pub get
open ios/Runner.xcworkspace        # WORKSPACE را باز کنید، نه .xcodeproj را
```

</div>

در Xcode:

1. تارگت **Runner** را انتخاب کنید → تب **Signing & Capabilities**.
2. تیک **Automatically manage signing** را بزنید.
3. **Team**: تیم Apple ID خود را انتخاب کنید (اگر در فهرست نیست، Apple ID خود را زیر Xcode → Settings → Accounts اضافه کنید).
4. **Bundle Identifier**: به‌صورت پیش‌فرض `io.opendray.opendray` است. با یک Apple ID رایگان ممکن است همین ID دقیق از سمت Apple قبلاً گرفته شده باشد — اگر Xcode خطای provisioning نشان داد، آن را به چیزی یکتا مثل `io.opendray.opendray.<yourname>` تغییر دهید.

### build و نصب روی iPhone

1. iPhone را با USB وصل کنید؛ وقتی پرسیده شد کامپیوتر را **trust** کنید.
2. روی گوشی **Developer Mode** را فعال کنید:
   Settings → Privacy & Security → Developer Mode → روشن → ری‌بوت.
3. در dropdownِ device در Xcode (نوار بالا)، iPhone خود را انتخاب کنید.
4. **▶ Run** را بزنید (یا `⌘R`). Xcode build می‌کند، امضا می‌کند و نصب می‌کند.

یا از CLI رانش کنید و بگذارید Xcode امضا را مدیریت کند:

<div dir="ltr">

```sh
flutter run --release -d <device-id>     # `flutter devices` idها را فهرست می‌کند
```

</div>

### اولین اجرا روی device

iOS اپی را که با یک تیم شخصی امضا شده اجرا نمی‌کند تا وقتی profile توسعه‌دهنده را trust کنید:

- روی گوشی: **Settings → General → VPN & Device Management →**
  Apple ID خودتان → **Trust**.

اپ با نام **Opendray** روی صفحه home ظاهر می‌شود.

> **انقضای Apple ID رایگان.** بعد از ~۷ روز اپ دیگر اجرا نمی‌شود («could not verify app»). build را از Xcode دوباره اجرا کنید تا profile تازه شود. یک حساب پولی این مشکل را ندارد.

---

## گام ۳ — اپ را به gateway خود وصل کنید

اولین اجرا صفحه onboarding را نشان می‌دهد:

1. **Gateway URL** — URL گام ۰ را وارد کنید
   (مثلاً `https://opendray.yourdomain.com`). روی **Continue** بزنید.
2. **Sign in** — `admin` + رمز ادمین شما (همانی که در `[admin].password` تنظیم کردید، یا بعداً تغییرش دادید).

همین — روی همان surfaceهای web admin فرود می‌آیید: Sessions، Channels، Integrations، Memory، Git، Settings.

برای اینکه بعداً اپ را به یک gateway دیگر نشانه بگیرید، در صفحه ورود روی **Change** بزنید (یا Settings → server) و URL را دوباره وارد کنید.

---

## به‌روزرسانی اپ

به‌روزرسانی خودکار وجود ندارد — بعد از pull کردن کد جدید دوباره نصب می‌کنید:

<div dir="ltr">

```sh
git pull
cd app/mobile
flutter pub get

# اندروید:
flutter build apk --release      # سپس sideload / `flutter install`

# iOS:
open ios/Runner.xcworkspace      # دوباره ▶ Run، یا `flutter run --release`
```

</div>

رشته‌ی نسخه‌ی خود اپ در `app/mobile/pubspec.yaml` زندگی می‌کند (`version: <semver>+<build>`).

---

## عیب‌یابی

| نشانه | علت | راه‌حل |
|---|---|---|
| onboarding می‌گوید «could not connect» | گوشی نمی‌تواند به Gateway URL برسد | URL را در مرورگر گوشی باز کنید؛ اول LAN IP / tunnel / TLS را درست کنید (گام ۰) |
| ورود کار می‌کند ولی ترمینال Sessions هیچ‌وقت وصل نمی‌شود | reverse proxy دارد WebSocket upgrade را drop می‌کند | هدرهای WS را اضافه کنید — [operator-guide §Topology](operator-guide.md#topology) |
| اندروید جلوی نصب را می‌گیرد | اجازه‌ی «Install unknown apps» داده نشده | برای اپی که `.apk` را باز می‌کند (Files / Chrome) اجازه‌اش را بدهید |
| هنگام اجرا روی iOS «Untrusted Developer» | profile تیم شخصی هنوز trust نشده | Settings → General → VPN & Device Management → Trust |
| در Xcode «Unable to install / signing» روی iOS | برخورد Bundle ID با یک Apple ID رایگان | Bundle Identifier را به `io.opendray.opendray.<yourname>` تغییر دهید |
| اپ iOS بعد از یک هفته دیگر باز نمی‌شود | profile رایگان Apple ID منقضی شده (۷ روز) | از Xcode دوباره اجرا کنید، یا از حساب پولی استفاده کنید |
| `flutter doctor` برای پلتفرم شما ✗ نشان می‌دهد | Android SDK / Xcode / CocoaPods کم است | دقیقاً همان خطی را که `flutter doctor` چاپ می‌کند دنبال کنید |

---

## همچنین ببینید

- [getting-started.md](getting-started.md) — gatewayی که اپ به آن وصل می‌شود را بالا بیاورید
- [operator-guide.md](operator-guide.md) — توپولوژی reverse proxy / tunnel برای دسترسی خارج از LAN
- [Flutter — build & release Android](https://docs.flutter.dev/deployment/android)
- [Flutter — build & release iOS](https://docs.flutter.dev/deployment/ios)

</div>
