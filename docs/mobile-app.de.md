# Mobile App — bauen & installieren

Die opendray-Mobile-App (`app/mobile/`) ist ein **Control-Client**, kein
zweites Gateway. Sie erledigt dieselbe Aufgabe wie das Web-Admin unter
`/admin/`: Sessions starten und steuern, Channels und Integrationen
verwalten, Memory durchsuchen, Git-Hosts lesen. Die Agents selbst laufen
weiterhin auf deinem Gateway-Host — das Telefon hängt sich nur an sie an.

Deshalb ist die App für sich allein nutzlos: Sie verbindet sich mit einem
**laufenden opendray-Gateway** über HTTPS. Bring zuerst das Gateway zum
Laufen ([getting-started](getting-started.md)), baue dann die App und
richte sie auf deine Gateway-URL.

> **Warum kein App-Store- / Play-Store-Download?**
> opendray ist selbst-gehostete, mandantenfreie Software. Ein Store-Build
> müsste *irgendjemandes* Backend einbacken, was genau das ist, was
> opendray nicht ist. Also baust du die App selbst, signiert mit deiner
> eigenen Identität, und sie spricht nur mit deinem Gateway. Die zwei
> unterstützten Wege unten sind **(A)** ein Android-APK, das du sideloadest,
> und **(B)** ein iOS-Build, das du über Xcode installierst.

---

## Schritt 0 — das Gateway vom Telefon aus erreichbar machen

Die App spricht über das Netzwerk mit dem Gateway, daher muss das Telefon
es erreichen können.

| Szenario | Was als Gateway URL eingeben |
|---|---|
| Telefon im selben LAN wie das Gateway | `http://<gateway-lan-ip>:8770` (z. B. `http://192.168.1.50:8770`) |
| Gateway hinter einem Reverse-Proxy mit TLS | `https://opendray.yourdomain.com` |
| Zugriff außerhalb des LAN (Mobilfunk, unterwegs) | Ein öffentlicher HTTPS-Endpunkt — Cloudflare Tunnel, Tailscale oder ein nginx/Caddy-Reverse-Proxy |

> **Exponiere `:8770` nicht roh ins Internet.** Setze TLS und einen Ingress
> davor. Cloudflare Tunnel ist die reibungsärmste Option (kein
> Port-Forwarding, keine öffentliche IP). nginx-/Caddy-Snippets —
> einschließlich der **WebSocket-Upgrade-Header**, die das Sessions-Terminal
> benötigt — findest du in
> [operator-guide §Topology](operator-guide.md#topology).

Verifiziere die Erreichbarkeit vom Telefon aus *bevor* du baust, z. B. indem
du die Gateway URL im Browser des Telefons öffnest — du solltest die
Web-Admin-Login-Seite erhalten.

---

## Schritt 1 — die Flutter-Toolchain installieren

Die App wird mit Flutter gebaut. Du brauchst es auf der Maschine, die den
Build durchführt (nicht auf dem Telefon).

```sh
# Folge https://docs.flutter.dev/get-started/install für dein OS.
flutter --version          # need 3.41+ (Dart SDK ^3.11)
flutter doctor             # resolve any ✗ for the platform you target
```

`flutter doctor` ist die Hürde: Es sagt dir genau, was für Android
(Android SDK + ein Gerät/Emulator) oder iOS (Xcode + CocoaPods) fehlt.
Behebe die ✗-Zeilen für deine Zielplattform, bevor du fortfährst.

Hol die Abhängigkeiten einmalig:

```sh
cd app/mobile
flutter pub get
```

---

## Schritt 2A — Android: ein APK bauen und sideloaden

Das ist der einfachste Weg — kein Entwicklerkonto, kein Store.

### Das APK bauen

```sh
cd app/mobile

# Single universal APK (easiest to share / sideload):
flutter build apk --release

# — or — smaller, per-architecture APKs (pick the one for your phone):
flutter build apk --release --split-per-abi
```

Die Ausgabe landet in:

```
app/mobile/build/app/outputs/flutter-apk/app-release.apk
```

> **Signierungs-Hinweis.** Out of the box wird das Release-Build mit dem
> **Debug-Keystore** signiert (siehe das `TODO` in
> `android/app/build.gradle.kts`). Das ist für persönliches Sideloading in
> Ordnung. Wenn du einen richtigen Upload-Key möchtest (erforderlich für den
> Play Store, und gute Hygiene für ein Build, das du fortlaufend
> aktualisierst), folge
> [Flutter — Sign the app](https://docs.flutter.dev/deployment/android#sign-the-app)
> und füge eine `signingConfig` für `release` hinzu.

### Das APK aufs Telefon bekommen

Wähle, was immer praktisch ist:

```sh
# If the phone is plugged in with USB debugging on, install directly:
flutter install                 # builds + installs to the attached device
# or, with an existing APK:
adb install -r build/app/outputs/flutter-apk/app-release.apk
```

Oder übertrage die `.apk`-Datei aufs Telefon (AirDrop-Äquivalent, ein
File-Share, ein Download-Link, eine E-Mail an dich selbst) und tippe sie an.
Android fragt dich, ob du **„Install unknown apps"** für die App erlaubst,
die die Datei öffnet (Files, Chrome usw.) — gewähre es, dann bestätige die
Installation.

Die App erscheint als **Opendray** (`io.opendray.opendray`).

---

## Schritt 2B — iOS: über Xcode bauen und installieren

iOS hat kein APK-Sideload-Äquivalent — jede Installation ist Code-signiert.
Du brauchst einen **Mac mit Xcode** und eine **Apple ID**. Eine kostenlose
Apple ID funktioniert (die App wird alle 7 Tage neu signiert; du
installierst neu, wenn das Provisioning-Profil abläuft). Ein
kostenpflichtiges Apple-Developer-Konto (99 US$/Jahr) gibt dir
jahresgültige Profile und TestFlight.

### Einmalige Signierungs-Einrichtung

```sh
cd app/mobile
flutter pub get
open ios/Runner.xcworkspace        # open the WORKSPACE, not the .xcodeproj
```

In Xcode:

1. Wähle das **Runner**-Target → Tab **Signing & Capabilities**.
2. Hake **Automatically manage signing** an.
3. **Team**: wähle dein Apple-ID-Team (füge deine Apple ID unter
   Xcode → Settings → Accounts hinzu, falls sie nicht gelistet ist).
4. **Bundle Identifier**: er wird als `io.opendray.opendray` ausgeliefert.
   Mit einer kostenlosen Apple ID kann genau diese ID auf Apples Seite
   bereits vergeben sein — wenn Xcode einen Provisioning-Fehler zeigt,
   ändere sie auf etwas Einzigartiges wie
   `io.opendray.opendray.<yourname>`.

### Bauen & auf das iPhone installieren

1. Verbinde das iPhone via USB; **vertraue** dem Computer, wenn du gefragt
   wirst.
2. Aktiviere **Developer Mode** auf dem Telefon:
   Settings → Privacy & Security → Developer Mode → an → Neustart.
3. Wähle in Xcodes Geräte-Dropdown (obere Leiste) dein iPhone.
4. Drücke **▶ Run** (oder `⌘R`). Xcode baut, signiert und installiert.

Oder steuere es von der CLI aus und lass Xcode die Signierung übernehmen:

```sh
flutter run --release -d <device-id>     # `flutter devices` lists ids
```

### Erster Start auf dem Gerät

iOS führt eine von einem persönlichen Team signierte App nicht aus, bis du
dem Entwicklerprofil vertraust:

- Auf dem Telefon: **Settings → General → VPN & Device Management →**
  deine Apple ID → **Trust**.

Die App erscheint als **Opendray** auf dem Home-Screen.

> **Ablauf bei kostenloser Apple ID.** Nach ~7 Tagen startet die App nicht
> mehr („could not verify app"). Führe das Build aus Xcode erneut aus, um
> das Profil aufzufrischen. Ein kostenpflichtiges Konto vermeidet das.

---

## Schritt 3 — die App mit deinem Gateway verbinden

Der erste Start zeigt den Onboarding-Screen:

1. **Gateway URL** — gib die URL aus Schritt 0 ein
   (z. B. `https://opendray.yourdomain.com`). Tippe **Continue**.
2. **Sign in** — `admin` + dein Admin-Passwort (das, das du in
   `[admin].password` gesetzt oder danach geändert hast).

Das war's — du landest auf denselben Surfaces wie im Web-Admin: Sessions,
Channels, Integrations, Memory, Git, Settings.

Um die App später auf ein anderes Gateway zu richten, tippe **Change** auf
dem Login-Screen (oder Settings → server) und gib die URL erneut ein.

---

## Die App aktualisieren

Es gibt kein Auto-Update — du installierst neu, nachdem du neuen Code
gezogen hast:

```sh
git pull
cd app/mobile
flutter pub get

# Android:
flutter build apk --release      # then sideload / `flutter install`

# iOS:
open ios/Runner.xcworkspace      # ▶ Run again, or `flutter run --release`
```

Der eigene Versions-String der App liegt in `app/mobile/pubspec.yaml`
(`version: <semver>+<build>`).

---

## Fehlerbehebung

| Symptom | Ursache | Behebung |
|---|---|---|
| Onboarding „could not connect" | Telefon erreicht die Gateway URL nicht | Öffne die URL im Browser des Telefons; behebe zuerst LAN-IP / Tunnel / TLS (Schritt 0) |
| Login funktioniert, aber das Sessions-Terminal verbindet nie | Reverse-Proxy verwirft das WebSocket-Upgrade | Füge WS-Header hinzu — [operator-guide §Topology](operator-guide.md#topology) |
| Android blockiert die Installation | „Install unknown apps" nicht gewährt | Erlaube es für die App, die das `.apk` öffnet (Files / Chrome) |
| iOS „Untrusted Developer" beim Start | Persönliches-Team-Profil noch nicht vertraut | Settings → General → VPN & Device Management → Trust |
| iOS „Unable to install / signing" in Xcode | Bundle-ID-Kollision mit einer kostenlosen Apple ID | Ändere den Bundle Identifier auf `io.opendray.opendray.<yourname>` |
| iOS-App startet nach einer Woche nicht mehr | Profil der kostenlosen Apple ID abgelaufen (7 Tage) | Erneut aus Xcode ausführen oder ein kostenpflichtiges Konto verwenden |
| `flutter doctor` zeigt ✗ für deine Plattform | Fehlendes Android SDK / Xcode / CocoaPods | Folge der genauen Zeile, die `flutter doctor` ausgibt |

---

## Siehe auch

- [getting-started.md](getting-started.md) — das Gateway aufsetzen, mit dem sich die App verbindet
- [operator-guide.md](operator-guide.md) — Reverse-Proxy-/Tunnel-Topologie für Zugriff außerhalb des LAN
- [Flutter — build & release Android](https://docs.flutter.dev/deployment/android)
- [Flutter — build & release iOS](https://docs.flutter.dev/deployment/ios)
