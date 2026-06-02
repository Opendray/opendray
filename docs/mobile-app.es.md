# App móvil — compilar e instalar

La app móvil de opendray (`app/mobile/`) es un **cliente de control**, no un
segundo gateway. Hace el mismo trabajo que el panel de administración web en `/admin/`:
lanzar y dirigir sesiones, gestionar canales e integraciones, explorar
memoria, leer hosts de git. Los propios agentes siguen ejecutándose en tu
host gateway — el teléfono simplemente se conecta a ellos.

Por eso, la app es inútil por sí sola: se conecta a un
**gateway de opendray en marcha** sobre HTTPS. Pon primero el gateway en marcha
([getting-started](getting-started.md)), luego compila la app y apúntala
a la URL de tu gateway.

> **¿Por qué no hay descarga en App Store / Play Store?**
> opendray es software autoalojado y de un solo inquilino. Una compilación de tienda tendría
> que incorporar el backend de *alguien*, que es exactamente lo que opendray
> no es. Así que compilas la app tú mismo, firmada con tu propia identidad,
> y solo habla con tu gateway. Los dos caminos soportados a continuación son
> **(A)** un APK de Android que instalas manualmente (sideload), y **(B)** una compilación de iOS que
> instalas mediante Xcode.

---

## Paso 0 — hacer que el gateway sea accesible desde el teléfono

La app habla con el gateway por la red, así que el teléfono tiene que poder
alcanzarlo.

| Escenario | Qué introducir como Gateway URL |
|---|---|
| Teléfono en la misma LAN que el gateway | `http://<gateway-lan-ip>:8770` (p. ej. `http://192.168.1.50:8770`) |
| Gateway detrás de un reverse proxy con TLS | `https://opendray.yourdomain.com` |
| Acceso fuera de la LAN (datos móviles, viajes) | Un endpoint HTTPS público — Cloudflare Tunnel, Tailscale, o un reverse proxy nginx/Caddy |

> **No expongas `:8770` directamente a internet.** Pon TLS y un
> ingress por delante. Cloudflare Tunnel es la opción de menor fricción
> (sin port-forwarding, sin IP pública). Los snippets de nginx / Caddy —
> incluyendo las **cabeceras de upgrade de WebSocket** que necesita la terminal de
> Sessions — están en [operator-guide §Topology](operator-guide.md#topology).

Verifica la accesibilidad desde el teléfono *antes* de compilar, p. ej. abre la
Gateway URL en el navegador del teléfono — deberías obtener la página de inicio de
sesión del panel de administración web.

---

## Paso 1 — instalar la cadena de herramientas de Flutter

La app se compila con Flutter. Lo necesitas en la máquina que hace la
compilación (no en el teléfono).

```sh
# Sigue https://docs.flutter.dev/get-started/install para tu SO.
flutter --version          # se necesita 3.41+ (Dart SDK ^3.11)
flutter doctor             # resuelve cualquier ✗ para la plataforma a la que apuntas
```

`flutter doctor` es el filtro: te dice exactamente qué falta para
Android (Android SDK + un dispositivo/emulador) o iOS (Xcode + CocoaPods).
Corrige las líneas ✗ para tu plataforma objetivo antes de continuar.

Obtén las dependencias una vez:

```sh
cd app/mobile
flutter pub get
```

---

## Paso 2A — Android: compilar un APK e instalarlo manualmente

Este es el camino más simple — sin cuenta de desarrollador, sin tienda.

### Compilar el APK

```sh
cd app/mobile

# APK universal único (lo más fácil de compartir / instalar manualmente):
flutter build apk --release

# — o — APKs más pequeños, por arquitectura (elige el de tu teléfono):
flutter build apk --release --split-per-abi
```

La salida queda en:

```
app/mobile/build/app/outputs/flutter-apk/app-release.apk
```

> **Nota sobre la firma.** De fábrica, la compilación de release se firma con
> el **keystore de depuración** (ver el `TODO` en
> `android/app/build.gradle.kts`). Eso está bien para sideload personal.
> Si quieres una clave de subida adecuada (requerida para Play Store, y buena
> higiene para una compilación que mantendrás actualizada), sigue
> [Flutter — Sign the app](https://docs.flutter.dev/deployment/android#sign-the-app)
> y añade un `signingConfig` para `release`.

### Llevar el APK al teléfono

Elige el que te resulte conveniente:

```sh
# Si el teléfono está conectado con depuración USB activada, instala directamente:
flutter install                 # compila + instala en el dispositivo conectado
# o, con un APK existente:
adb install -r build/app/outputs/flutter-apk/app-release.apk
```

O transfiere el archivo `.apk` al teléfono (equivalente a AirDrop, un
recurso compartido de archivos, un enlace de descarga, un correo a ti mismo) y púlsalo. Android te pedirá
permitir **"Instalar apps desconocidas"** para la app que esté abriendo el
archivo (Files, Chrome, etc.) — concédelo, luego confirma la instalación.

La app aparece como **Opendray** (`io.opendray.opendray`).

---

## Paso 2B — iOS: compilar e instalar mediante Xcode

iOS no tiene un equivalente a instalar-un-APK manualmente — cada instalación está firmada con código.
Necesitas un **Mac con Xcode** y un **Apple ID**. Un Apple ID gratuito
funciona (la app se vuelve a firmar cada 7 días; reinstalas cuando el
perfil de aprovisionamiento expira). Una cuenta de pago de Apple Developer
(US$99/año) da perfiles de un año de duración y TestFlight.

### Configuración de firma única

```sh
cd app/mobile
flutter pub get
open ios/Runner.xcworkspace        # abre el WORKSPACE, no el .xcodeproj
```

En Xcode:

1. Selecciona el target **Runner** → pestaña **Signing & Capabilities**.
2. Marca **Automatically manage signing**.
3. **Team**: elige el equipo de tu Apple ID (añade tu Apple ID en
   Xcode → Settings → Accounts si no está listado).
4. **Bundle Identifier**: viene como `io.opendray.opendray`. Con un
   Apple ID gratuito, este ID exacto puede estar ya ocupado del lado de Apple —
   si Xcode muestra un error de aprovisionamiento, cámbialo a algo único
   como `io.opendray.opendray.<tunombre>`.

### Compilar e instalar en el iPhone

1. Conecta el iPhone vía USB; **confía** en el ordenador cuando se te pida.
2. Activa el **Modo de desarrollador** en el teléfono:
   Settings → Privacy & Security → Developer Mode → activar → reiniciar.
3. En el desplegable de dispositivos de Xcode (barra superior), selecciona tu iPhone.
4. Pulsa **▶ Run** (o `⌘R`). Xcode compila, firma e instala.

O contrólalo desde la CLI y deja que Xcode gestione la firma:

```sh
flutter run --release -d <device-id>     # `flutter devices` lista los ids
```

### Primer lanzamiento en el dispositivo

iOS no ejecutará una app firmada por un equipo personal hasta que confíes en el
perfil del desarrollador:

- En el teléfono: **Settings → General → VPN & Device Management →**
  tu Apple ID → **Trust**.

La app aparece como **Opendray** en la pantalla de inicio.

> **Expiración del Apple ID gratuito.** Tras ~7 días la app deja de lanzarse
> ("could not verify app"). Vuelve a ejecutar la compilación desde Xcode para refrescar el
> perfil. Una cuenta de pago evita esto.

---

## Paso 3 — conectar la app a tu gateway

El primer lanzamiento muestra la pantalla de incorporación:

1. **Gateway URL** — introduce la URL del Paso 0
   (p. ej. `https://opendray.yourdomain.com`). Pulsa **Continue**.
2. **Sign in** — `admin` + tu contraseña de administración (la que estableciste en
   `[admin].password`, o la que cambiaste después).

Eso es todo — aterrizas en las mismas superficies que el panel de administración web: Sessions,
Channels, Integrations, Memory, Git, Settings.

Para apuntar la app a un gateway diferente más tarde, pulsa **Change** en la
pantalla de inicio de sesión (o Settings → server) y vuelve a introducir la URL.

---

## Actualizar la app

No hay actualización automática — reinstalas tras obtener el nuevo código:

```sh
git pull
cd app/mobile
flutter pub get

# Android:
flutter build apk --release      # luego sideload / `flutter install`

# iOS:
open ios/Runner.xcworkspace      # ▶ Run de nuevo, o `flutter run --release`
```

La cadena de versión propia de la app vive en `app/mobile/pubspec.yaml`
(`version: <semver>+<build>`).

---

## Solución de problemas

| Síntoma | Causa | Solución |
|---|---|---|
| "could not connect" en la incorporación | El teléfono no puede alcanzar la Gateway URL | Abre la URL en el navegador del teléfono; corrige primero la IP de LAN / túnel / TLS (Paso 0) |
| El inicio de sesión funciona pero la terminal de Sessions nunca conecta | El reverse proxy está descartando el upgrade de WebSocket | Añade las cabeceras WS — [operator-guide §Topology](operator-guide.md#topology) |
| Android bloquea la instalación | "Instalar apps desconocidas" no concedido | Permítelo para la app que abre el `.apk` (Files / Chrome) |
| "Untrusted Developer" en iOS al lanzar | El perfil de equipo personal aún no es de confianza | Settings → General → VPN & Device Management → Trust |
| "Unable to install / signing" en iOS dentro de Xcode | Conflicto de Bundle ID con un Apple ID gratuito | Cambia el Bundle Identifier a `io.opendray.opendray.<tunombre>` |
| La app de iOS deja de abrirse tras una semana | El perfil del Apple ID gratuito expiró (7 días) | Vuelve a ejecutar desde Xcode, o usa una cuenta de pago |
| `flutter doctor` muestra ✗ para tu plataforma | Falta Android SDK / Xcode / CocoaPods | Sigue la línea exacta que imprime `flutter doctor` |

---

## Consulta también

- [getting-started.md](getting-started.md) — pon en marcha el gateway al que se conecta la app
- [operator-guide.md](operator-guide.md) — topología de reverse proxy / túnel para acceso fuera de la LAN
- [Flutter — build & release Android](https://docs.flutter.dev/deployment/android)
- [Flutter — build & release iOS](https://docs.flutter.dev/deployment/ios)
