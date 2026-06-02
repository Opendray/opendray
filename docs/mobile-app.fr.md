# Application mobile — build & installation

L'application mobile opendray (`app/mobile/`) est un **client de contrôle**, pas
un second gateway. Elle fait le même travail que l'admin web à `/admin/` :
lancer et piloter des sessions, gérer les channels et les intégrations, parcourir
la mémoire, lire les hôtes git. Les agents eux-mêmes continuent de tourner sur ton
hôte gateway — le téléphone ne fait que s'y rattacher.

Pour cette raison, l'application est inutile seule : elle se connecte à un
**gateway opendray en marche** via HTTPS. Mets d'abord le gateway en route
([getting-started](getting-started.md)), puis build l'application et pointe-la
vers l'URL de ton gateway.

> **Pourquoi pas de téléchargement App Store / Play Store ?**
> opendray est un logiciel auto-hébergé, mono-locataire. Un build de store devrait
> embarquer le backend de *quelqu'un*, ce qui est exactement ce qu'opendray
> n'est pas. Tu build donc l'application toi-même, signée avec ta propre identité,
> et elle ne parle qu'à ton gateway. Les deux chemins supportés ci-dessous sont
> **(A)** un APK Android que tu sideload, et **(B)** un build iOS que tu
> installes via Xcode.

---

## Étape 0 — rendre le gateway joignable depuis le téléphone

L'application parle au gateway via le réseau, donc le téléphone doit pouvoir
l'atteindre.

| Scénario | Quoi saisir comme Gateway URL |
|---|---|
| Téléphone sur le même LAN que le gateway | `http://<gateway-lan-ip>:8770` (ex. `http://192.168.1.50:8770`) |
| Gateway derrière un reverse proxy avec TLS | `https://opendray.yourdomain.com` |
| Accès hors-LAN (cellulaire, en déplacement) | Un endpoint HTTPS public — Cloudflare Tunnel, Tailscale, ou un reverse proxy nginx/Caddy |

> **N'expose pas `:8770` brut sur internet.** Mets du TLS et un ingress devant.
> Cloudflare Tunnel est l'option la plus simple (pas de port-forwarding, pas
> d'IP publique). Les snippets nginx / Caddy — y compris les
> **headers d'upgrade WebSocket** dont le terminal Sessions a besoin — sont dans
> [operator-guide §Topology](operator-guide.md#topology).

Vérifie la joignabilité depuis le téléphone *avant* de build, par exemple ouvre
la Gateway URL dans le navigateur du téléphone — tu devrais obtenir la page de
connexion de l'admin web.

---

## Étape 1 — installer la toolchain Flutter

L'application est construite avec Flutter. Tu en as besoin sur la machine qui fait
le build (pas sur le téléphone).

```sh
# Suis https://docs.flutter.dev/get-started/install pour ton OS.
flutter --version          # besoin de 3.41+ (Dart SDK ^3.11)
flutter doctor             # résous chaque ✗ pour la plateforme que tu vises
```

`flutter doctor` est le filtre : il te dit exactement ce qui manque pour
Android (Android SDK + un device/émulateur) ou iOS (Xcode + CocoaPods).
Corrige les lignes ✗ de ta plateforme cible avant de continuer.

Récupère les dépendances une fois :

```sh
cd app/mobile
flutter pub get
```

---

## Étape 2A — Android : build un APK et sideload-le

C'est le chemin le plus simple — pas de compte développeur, pas de store.

### Build l'APK

```sh
cd app/mobile

# APK universel unique (le plus simple à partager / sideload) :
flutter build apk --release

# — ou — des APK plus petits, par architecture (choisis celui de ton téléphone) :
flutter build apk --release --split-per-abi
```

Le résultat se trouve dans :

```
app/mobile/build/app/outputs/flutter-apk/app-release.apk
```

> **Note sur la signature.** Tel quel, le build release est signé avec le
> **keystore de debug** (voir le `TODO` dans
> `android/app/build.gradle.kts`). C'est très bien pour du sideloading personnel.
> Si tu veux une vraie upload key (requise pour le Play Store, et bonne
> hygiène pour un build que tu vas mettre à jour régulièrement), suis
> [Flutter — Sign the app](https://docs.flutter.dev/deployment/android#sign-the-app)
> et ajoute une `signingConfig` pour `release`.

### Mettre l'APK sur le téléphone

Prends ce qui est le plus pratique :

```sh
# Si le téléphone est branché avec le débogage USB activé, installe directement :
flutter install                 # build + installe sur le device attaché
# ou, avec un APK existant :
adb install -r build/app/outputs/flutter-apk/app-release.apk
```

Ou transfère le fichier `.apk` sur le téléphone (équivalent AirDrop, un partage
de fichiers, un lien de téléchargement, un email à toi-même) et tape dessus.
Android te demandera d'autoriser **« Installer des applis inconnues »** pour
l'appli qui ouvre le fichier (Fichiers, Chrome, etc.) — accorde-le, puis confirme
l'installation.

L'application apparaît sous le nom **Opendray** (`io.opendray.opendray`).

---

## Étape 2B — iOS : build et installation via Xcode

iOS n'a pas d'équivalent au sideload d'un APK — chaque installation est signée par
code. Il te faut un **Mac avec Xcode** et un **Apple ID**. Un Apple ID gratuit
fonctionne (l'application est re-signée tous les 7 jours ; tu réinstalles quand le
provisioning profile expire). Un compte Apple Developer payant
(99 $US/an) donne des profils valables un an et TestFlight.

### Configuration de signature (une fois)

```sh
cd app/mobile
flutter pub get
open ios/Runner.xcworkspace        # ouvre le WORKSPACE, pas le .xcodeproj
```

Dans Xcode :

1. Sélectionne la target **Runner** → onglet **Signing & Capabilities**.
2. Coche **Automatically manage signing**.
3. **Team** : choisis ton équipe Apple ID (ajoute ton Apple ID sous
   Xcode → Settings → Accounts s'il n'apparaît pas).
4. **Bundle Identifier** : il est livré en `io.opendray.opendray`. Avec un
   Apple ID gratuit, cet ID exact peut déjà être pris côté Apple —
   si Xcode affiche une erreur de provisioning, change-le pour quelque chose
   d'unique comme `io.opendray.opendray.<yourname>`.

### Build & installation sur l'iPhone

1. Connecte l'iPhone via USB ; **fais confiance** à l'ordinateur quand on te le demande.
2. Active le **Developer Mode** sur le téléphone :
   Settings → Privacy & Security → Developer Mode → on → reboot.
3. Dans le menu déroulant des devices de Xcode (barre du haut), sélectionne ton iPhone.
4. Appuie sur **▶ Run** (ou `⌘R`). Xcode build, signe, et installe.

Ou pilote-le depuis la CLI et laisse Xcode gérer la signature :

```sh
flutter run --release -d <device-id>     # `flutter devices` liste les ids
```

### Premier lancement sur le device

iOS ne lancera pas une application signée par une équipe personnelle tant que tu
n'as pas fait confiance au profil développeur :

- Sur le téléphone : **Settings → General → VPN & Device Management →**
  ton Apple ID → **Trust**.

L'application apparaît sous le nom **Opendray** sur l'écran d'accueil.

> **Expiration de l'Apple ID gratuit.** Après ~7 jours, l'application cesse de se
> lancer (« could not verify app »). Relance le build depuis Xcode pour rafraîchir
> le profil. Un compte payant évite ça.

---

## Étape 3 — connecter l'application à ton gateway

Le premier lancement affiche l'écran d'onboarding :

1. **Gateway URL** — saisis l'URL de l'Étape 0
   (ex. `https://opendray.yourdomain.com`). Tape **Continue**.
2. **Sign in** — `admin` + ton mot de passe admin (celui que tu as défini dans
   `[admin].password`, ou modifié par la suite).

C'est tout — tu arrives sur les mêmes surfaces que l'admin web : Sessions,
Channels, Integrations, Memory, Git, Settings.

Pour pointer l'application vers un autre gateway plus tard, tape **Change** sur
l'écran de connexion (ou Settings → server) et saisis à nouveau l'URL.

---

## Mettre à jour l'application

Il n'y a pas d'auto-update — tu réinstalles après avoir récupéré le nouveau code :

```sh
git pull
cd app/mobile
flutter pub get

# Android :
flutter build apk --release      # puis sideload / `flutter install`

# iOS :
open ios/Runner.xcworkspace      # ▶ Run à nouveau, ou `flutter run --release`
```

La chaîne de version propre à l'application vit dans `app/mobile/pubspec.yaml`
(`version: <semver>+<build>`).

---

## Dépannage

| Symptôme | Cause | Correctif |
|---|---|---|
| Onboarding « could not connect » | Le téléphone ne peut pas atteindre la Gateway URL | Ouvre l'URL dans le navigateur du téléphone ; corrige d'abord l'IP LAN / le tunnel / le TLS (Étape 0) |
| La connexion marche mais le terminal Sessions ne se connecte jamais | Le reverse proxy laisse tomber l'upgrade WebSocket | Ajoute les headers WS — [operator-guide §Topology](operator-guide.md#topology) |
| Android bloque l'installation | « Installer des applis inconnues » non accordé | Autorise-le pour l'appli qui ouvre le `.apk` (Fichiers / Chrome) |
| iOS « Untrusted Developer » au lancement | Profil d'équipe personnelle pas encore approuvé | Settings → General → VPN & Device Management → Trust |
| iOS « Unable to install / signing » dans Xcode | Conflit de Bundle ID avec un Apple ID gratuit | Change le Bundle Identifier en `io.opendray.opendray.<yourname>` |
| L'application iOS cesse de s'ouvrir après une semaine | Profil Apple ID gratuit expiré (7 jours) | Relance depuis Xcode, ou utilise un compte payant |
| `flutter doctor` montre ✗ pour ta plateforme | Android SDK / Xcode / CocoaPods manquant | Suis la ligne exacte qu'affiche `flutter doctor` |

---

## Voir aussi

- [getting-started.md](getting-started.md) — mettre en route le gateway auquel l'application se connecte
- [operator-guide.md](operator-guide.md) — topologie reverse proxy / tunnel pour l'accès hors-LAN
- [Flutter — build & release Android](https://docs.flutter.dev/deployment/android)
- [Flutter — build & release iOS](https://docs.flutter.dev/deployment/ios)
