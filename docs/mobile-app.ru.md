# Мобильное приложение — сборка и установка

Мобильное приложение opendray (`app/mobile/`) — это **клиент управления**, а
не второй шлюз. Оно выполняет ту же работу, что и веб-админка по адресу
`/admin/`: запускает и ведёт сессии, управляет каналами и интеграциями,
просматривает память, читает git-хосты. Сами агенты продолжают работать на
вашем хосте-шлюзе — телефон лишь подключается к ним.

Поэтому само по себе приложение бесполезно: оно подключается к **работающему
шлюзу opendray** по HTTPS. Сначала поднимите шлюз
([getting-started](getting-started.md)), затем соберите приложение и укажите
ему URL вашего шлюза.

> **Почему нет загрузки из App Store / Play Store?**
> opendray — это self-hosted, однотенантное ПО. Сборка для магазина была бы
> вынуждена зашить в себя *чей-то* бэкенд, а это ровно то, чем opendray не
> является. Поэтому вы собираете приложение сами, подписанное вашей
> собственной учётной записью, и оно общается только с вашим шлюзом. Два
> поддерживаемых пути ниже — **(A)** Android APK, который вы устанавливаете
> вручную (sideload), и **(B)** iOS-сборка, которую вы ставите через Xcode.

---

## Шаг 0 — сделайте шлюз доступным с телефона

Приложение общается со шлюзом по сети, поэтому телефон должен иметь
возможность достучаться до него.

| Сценарий | Что вводить как Gateway URL |
|---|---|
| Телефон в той же LAN, что и шлюз | `http://<gateway-lan-ip>:8770` (например, `http://192.168.1.50:8770`) |
| Шлюз за reverse-proxy с TLS | `https://opendray.yourdomain.com` |
| Доступ вне LAN (сотовая связь, в дороге) | Публичный HTTPS-эндпоинт — Cloudflare Tunnel, Tailscale или reverse-proxy на nginx/Caddy |

> **Не выставляйте `:8770` голым в интернет.** Поставьте перед ним TLS и
> ingress. Cloudflare Tunnel — вариант с наименьшим трением (без
> проброса портов, без публичного IP). Сниппеты для nginx / Caddy —
> включая **заголовки WebSocket upgrade**, которые нужны терминалу сессий, —
> приведены в [operator-guide §Topology](operator-guide.md#topology).

Проверьте доступность с телефона *до* сборки — например, откройте Gateway URL
в браузере телефона: вы должны увидеть страницу входа веб-админки.

---

## Шаг 1 — установите тулчейн Flutter

Приложение собирается с помощью Flutter. Он нужен на машине, которая делает
сборку (не на телефоне).

```sh
# Следуйте https://docs.flutter.dev/get-started/install для вашей ОС.
flutter --version          # нужен 3.41+ (Dart SDK ^3.11)
flutter doctor             # устраните любые ✗ для целевой платформы
```

`flutter doctor` — это контрольная точка: он точно сообщает, чего не хватает
для Android (Android SDK + устройство/эмулятор) или iOS (Xcode + CocoaPods).
Исправьте строки с ✗ для вашей целевой платформы, прежде чем продолжить.

Один раз получите зависимости:

```sh
cd app/mobile
flutter pub get
```

---

## Шаг 2A — Android: соберите APK и установите его вручную

Это самый простой путь — без аккаунта разработчика, без магазина.

### Соберите APK

```sh
cd app/mobile

# Единый универсальный APK (проще всего поделиться / установить вручную):
flutter build apk --release

# — или — меньшие APK по архитектурам (выберите подходящий вашему телефону):
flutter build apk --release --split-per-abi
```

Результат окажется в:

```
app/mobile/build/app/outputs/flutter-apk/app-release.apk
```

> **Замечание о подписи.** По умолчанию release-сборка подписывается
> **debug-keystore** (см. `TODO` в
> `android/app/build.gradle.kts`). Для личной установки вручную этого
> достаточно. Если вам нужен полноценный upload-ключ (обязателен для Play
> Store и хорош как гигиена для сборки, которую вы будете обновлять),
> следуйте [Flutter — Sign the app](https://docs.flutter.dev/deployment/android#sign-the-app)
> и добавьте `signingConfig` для `release`.

### Перенесите APK на телефон

Выберите, что удобнее:

```sh
# Если телефон подключён по USB с включённой отладкой, установите напрямую:
flutter install                 # собирает + устанавливает на подключённое устройство
# или, с уже готовым APK:
adb install -r build/app/outputs/flutter-apk/app-release.apk
```

Либо перенесите файл `.apk` на телефон (аналог AirDrop, файловая шара, ссылка
для скачивания, письмо самому себе) и нажмите на него. Android попросит
разрешить **«Установка неизвестных приложений»** для того приложения, которое
открывает файл (Files, Chrome и т. д.) — выдайте разрешение, затем подтвердите
установку.

Приложение появляется как **Opendray** (`io.opendray.opendray`).

---

## Шаг 2B — iOS: сборка и установка через Xcode

В iOS нет аналога установки APK вручную — каждая установка подписана кодом.
Вам нужен **Mac с Xcode** и **Apple ID**. Бесплатный Apple ID подойдёт
(приложение переподписывается каждые 7 дней; вы переустанавливаете его, когда
provisioning-профиль истекает). Платный аккаунт Apple Developer
(99 долл. США в год) даёт профили на год и TestFlight.

### Однократная настройка подписи

```sh
cd app/mobile
flutter pub get
open ios/Runner.xcworkspace        # откройте WORKSPACE, а не .xcodeproj
```

В Xcode:

1. Выберите таргет **Runner** → вкладка **Signing & Capabilities**.
2. Отметьте **Automatically manage signing**.
3. **Team**: выберите команду вашего Apple ID (добавьте Apple ID в
   Xcode → Settings → Accounts, если его нет в списке).
4. **Bundle Identifier**: поставляется как `io.opendray.opendray`. С
   бесплатным Apple ID этот точный ID может быть уже занят на стороне Apple —
   если Xcode показывает ошибку provisioning, измените его на что-то
   уникальное, например `io.opendray.opendray.<yourname>`.

### Сборка и установка на iPhone

1. Подключите iPhone по USB; **доверьте** компьютеру по запросу.
2. Включите **Developer Mode** на телефоне:
   Settings → Privacy & Security → Developer Mode → включить → перезагрузка.
3. В выпадающем списке устройств Xcode (верхняя панель) выберите ваш iPhone.
4. Нажмите **▶ Run** (или `⌘R`). Xcode соберёт, подпишет и установит.

Либо управляйте этим из CLI, доверив подпись Xcode:

```sh
flutter run --release -d <device-id>     # `flutter devices` перечисляет id
```

### Первый запуск на устройстве

iOS не запустит приложение, подписанное персональной командой, пока вы не
доверите профиль разработчика:

- На телефоне: **Settings → General → VPN & Device Management →**
  ваш Apple ID → **Trust**.

Приложение появляется на домашнем экране как **Opendray**.

> **Истечение бесплатного Apple ID.** Примерно через 7 дней приложение
> перестаёт запускаться («could not verify app»). Перезапустите сборку из
> Xcode, чтобы обновить профиль. Платный аккаунт избавляет от этого.

---

## Шаг 3 — подключите приложение к вашему шлюзу

Первый запуск показывает экран онбординга:

1. **Gateway URL** — введите URL из Шага 0
   (например, `https://opendray.yourdomain.com`). Нажмите **Continue**.
2. **Вход** — `admin` + ваш пароль администратора (тот, что вы задали в
   `[admin].password`, или сменили позже).

Вот и всё — вы попадаете на те же поверхности, что и в веб-админке: Sessions,
Channels, Integrations, Memory, Git, Settings.

Чтобы позже направить приложение на другой шлюз, нажмите **Change** на экране
входа (или Settings → server) и заново введите URL.

---

## Обновление приложения

Авто-обновления нет — вы переустанавливаете приложение после получения нового
кода:

```sh
git pull
cd app/mobile
flutter pub get

# Android:
flutter build apk --release      # затем установка вручную / `flutter install`

# iOS:
open ios/Runner.xcworkspace      # снова ▶ Run, или `flutter run --release`
```

Собственная строка версии приложения находится в `app/mobile/pubspec.yaml`
(`version: <semver>+<build>`).

---

## Устранение неполадок

| Симптом | Причина | Решение |
|---|---|---|
| Онбординг «could not connect» | Телефон не может достучаться до Gateway URL | Откройте URL в браузере телефона; сначала исправьте LAN IP / туннель / TLS (Шаг 0) |
| Вход работает, но терминал сессий не подключается | Reverse-proxy отбрасывает WebSocket upgrade | Добавьте WS-заголовки — [operator-guide §Topology](operator-guide.md#topology) |
| Android блокирует установку | Не выдано «Установка неизвестных приложений» | Разрешите её для приложения, открывающего `.apk` (Files / Chrome) |
| iOS «Untrusted Developer» при запуске | Профиль персональной команды ещё не доверен | Settings → General → VPN & Device Management → Trust |
| iOS «Unable to install / signing» в Xcode | Конфликт Bundle ID с бесплатным Apple ID | Измените Bundle Identifier на `io.opendray.opendray.<yourname>` |
| iOS-приложение перестаёт открываться через неделю | Профиль бесплатного Apple ID истёк (7 дней) | Перезапустите из Xcode или используйте платный аккаунт |
| `flutter doctor` показывает ✗ для вашей платформы | Отсутствует Android SDK / Xcode / CocoaPods | Следуйте точной строке, которую печатает `flutter doctor` |

---

## См. также

- [getting-started.md](getting-started.md) — поднимите шлюз, к которому подключается приложение
- [operator-guide.md](operator-guide.md) — топология reverse-proxy / туннеля для доступа вне LAN
- [Flutter — build & release Android](https://docs.flutter.dev/deployment/android)
- [Flutter — build & release iOS](https://docs.flutter.dev/deployment/ios)
