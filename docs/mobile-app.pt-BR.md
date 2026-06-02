# App mobile — build e instalação

O app mobile do opendray (`app/mobile/`) é um **cliente de controle**, não um
segundo gateway. Ele faz o mesmo trabalho do admin web em `/admin/`: cria e
conduz sessões, gerencia canais e integrações, navega pela memória, lê hosts
git. Os próprios agentes continuam rodando no seu host de gateway — o celular
apenas se conecta a eles.

Por isso, o app é inútil sozinho: ele se conecta a um **gateway opendray em
execução** via HTTPS. Suba o gateway primeiro
([getting-started](getting-started.md)), depois faça o build do app e aponte-o
para a URL do seu gateway.

> **Por que não há download na App Store / Play Store?**
> O opendray é um software self-hosted e single-tenant. Um build de loja teria
> que embutir o backend de *alguém*, que é exatamente o que o opendray não é.
> Então você faz o build do app por conta própria, assinado com sua própria
> identidade, e ele fala apenas com o seu gateway. Os dois caminhos suportados
> abaixo são **(A)** um APK Android que você instala via sideload, e **(B)** um
> build iOS que você instala pelo Xcode.

---

## Etapa 0 — Tornar o gateway acessível a partir do celular

O app fala com o gateway pela rede, então o celular precisa conseguir
alcançá-lo.

| Cenário | O que inserir como URL do Gateway |
|---|---|
| Celular na mesma LAN que o gateway | `http://<gateway-lan-ip>:8770` (ex.: `http://192.168.1.50:8770`) |
| Gateway atrás de um reverse proxy com TLS | `https://opendray.yourdomain.com` |
| Acesso fora da LAN (celular, viagem) | Um endpoint HTTPS público — Cloudflare Tunnel, Tailscale ou um reverse proxy nginx/Caddy |

> **Não exponha a `:8770` crua para a internet.** Coloque TLS e um ingress na
> frente dela. O Cloudflare Tunnel é a opção de menor atrito (sem
> port-forwarding, sem IP público). Snippets de nginx / Caddy — incluindo os
> **headers de upgrade de WebSocket** que o terminal de Sessões precisa —
> estão em [operator-guide §Topology](operator-guide.md#topology).

Verifique a acessibilidade a partir do celular *antes* de fazer o build, por
exemplo abrindo a URL do Gateway no navegador do celular — você deve obter a
página de login do admin web.

---

## Etapa 1 — Instalar o toolchain do Flutter

O app é construído com Flutter. Você precisa dele na máquina que faz o build
(não no celular).

```sh
# Siga https://docs.flutter.dev/get-started/install para o seu OS.
flutter --version          # precisa 3.41+ (Dart SDK ^3.11)
flutter doctor             # resolva qualquer ✗ para a plataforma alvo
```

O `flutter doctor` é o portão: ele te diz exatamente o que está faltando para
Android (Android SDK + um dispositivo/emulador) ou iOS (Xcode + CocoaPods).
Corrija as linhas com ✗ da sua plataforma alvo antes de continuar.

Busque as dependências uma vez:

```sh
cd app/mobile
flutter pub get
```

---

## Etapa 2A — Android: buildar um APK e instalar via sideload

Este é o caminho mais simples — sem conta de desenvolvedor, sem loja.

### Buildar o APK

```sh
cd app/mobile

# APK universal único (mais fácil de compartilhar / instalar via sideload):
flutter build apk --release

# — ou — APKs menores, por arquitetura (escolha o do seu celular):
flutter build apk --release --split-per-abi
```

A saída fica em:

```
app/mobile/build/app/outputs/flutter-apk/app-release.apk
```

> **Nota sobre assinatura.** Por padrão, o build de release é assinado com o
> **keystore de debug** (veja o `TODO` em
> `android/app/build.gradle.kts`). Isso é suficiente para sideload pessoal.
> Se você quiser uma chave de upload apropriada (obrigatória para a Play
> Store, e boa prática para um build que você vai manter atualizando), siga
> [Flutter — Sign the app](https://docs.flutter.dev/deployment/android#sign-the-app)
> e adicione um `signingConfig` para `release`.

### Colocar o APK no celular

Escolha o que for conveniente:

```sh
# Se o celular estiver conectado com depuração USB ativa, instale diretamente:
flutter install                 # builda + instala no dispositivo conectado
# ou, com um APK existente:
adb install -r build/app/outputs/flutter-apk/app-release.apk
```

Ou transfira o arquivo `.apk` para o celular (equivalente ao AirDrop, um
compartilhamento de arquivos, um link de download, e-mail para você mesmo) e
toque nele. O Android vai pedir para você permitir **"Instalar apps
desconhecidos"** para o app que estiver abrindo o arquivo (Arquivos, Chrome,
etc.) — conceda a permissão, depois confirme a instalação.

O app aparece como **Opendray** (`io.opendray.opendray`).

---

## Etapa 2B — iOS: buildar e instalar pelo Xcode

O iOS não tem equivalente a instalar um APK via sideload — toda instalação é
code-signed. Você precisa de um **Mac com Xcode** e um **Apple ID**. Um Apple
ID gratuito funciona (o app é re-assinado a cada 7 dias; você reinstala quando
o provisioning profile expira). Uma conta paga do Apple Developer (US$99/ano)
dá profiles válidos por um ano e TestFlight.

### Configuração de assinatura (uma vez)

```sh
cd app/mobile
flutter pub get
open ios/Runner.xcworkspace        # abra o WORKSPACE, não o .xcodeproj
```

No Xcode:

1. Selecione o target **Runner** → aba **Signing & Capabilities**.
2. Marque **Automatically manage signing**.
3. **Team**: escolha o time do seu Apple ID (adicione seu Apple ID em
   Xcode → Settings → Accounts se ele não estiver listado).
4. **Bundle Identifier**: ele vem como `io.opendray.opendray`. Com um Apple ID
   gratuito esse ID exato pode já estar em uso do lado da Apple — se o Xcode
   mostrar um erro de provisioning, mude-o para algo único como
   `io.opendray.opendray.<yourname>`.

### Buildar e instalar no iPhone

1. Conecte o iPhone via USB; **confie** no computador quando solicitado.
2. Habilite o **Developer Mode** no celular:
   Settings → Privacy & Security → Developer Mode → on → reinicie.
3. No dropdown de dispositivos do Xcode (barra superior), selecione seu iPhone.
4. Pressione **▶ Run** (ou `⌘R`). O Xcode builda, assina e instala.

Ou conduza pelo CLI e deixe o Xcode cuidar da assinatura:

```sh
flutter run --release -d <device-id>     # `flutter devices` lista os ids
```

### Primeiro lançamento no dispositivo

O iOS não vai rodar um app assinado por um time pessoal até você confiar no
developer profile:

- No celular: **Settings → General → VPN & Device Management →**
  seu Apple ID → **Trust**.

O app aparece como **Opendray** na tela inicial.

> **Expiração do Apple ID gratuito.** Após ~7 dias o app para de abrir
> ("could not verify app"). Rode o build novamente pelo Xcode para renovar o
> profile. Uma conta paga evita isso.

---

## Etapa 3 — Conectar o app ao seu gateway

O primeiro lançamento mostra a tela de onboarding:

1. **Gateway URL** — insira a URL da Etapa 0
   (ex.: `https://opendray.yourdomain.com`). Toque em **Continue**.
2. **Sign in** — `admin` + sua senha de admin (a que você definiu em
   `[admin].password`, ou alterou depois).

É isso — você chega nas mesmas superfícies do admin web: Sessões, Canais,
Integrações, Memória, Git, Configurações.

Para apontar o app para um gateway diferente depois, toque em **Change** na
tela de login (ou Settings → server) e insira a URL novamente.

---

## Atualizando o app

Não há atualização automática — você reinstala após puxar o código novo:

```sh
git pull
cd app/mobile
flutter pub get

# Android:
flutter build apk --release      # depois sideload / `flutter install`

# iOS:
open ios/Runner.xcworkspace      # ▶ Run de novo, ou `flutter run --release`
```

A string de versão do próprio app fica em `app/mobile/pubspec.yaml`
(`version: <semver>+<build>`).

---

## Solução de problemas

| Sintoma | Causa | Correção |
|---|---|---|
| Onboarding com "could not connect" | O celular não alcança a URL do Gateway | Abra a URL no navegador do celular; corrija IP da LAN / tunnel / TLS primeiro (Etapa 0) |
| Login funciona mas o terminal de Sessões nunca conecta | O reverse proxy está descartando o upgrade de WebSocket | Adicione os headers WS — [operator-guide §Topology](operator-guide.md#topology) |
| O Android bloqueia a instalação | "Instalar apps desconhecidos" não foi concedido | Permita para o app que abre o `.apk` (Arquivos / Chrome) |
| iOS com "Untrusted Developer" ao abrir | Profile de time pessoal ainda não confiado | Settings → General → VPN & Device Management → Trust |
| iOS com "Unable to install / signing" no Xcode | Conflito de Bundle ID com um Apple ID gratuito | Mude o Bundle Identifier para `io.opendray.opendray.<yourname>` |
| O app iOS para de abrir após uma semana | Profile do Apple ID gratuito expirou (7 dias) | Rode novamente pelo Xcode, ou use uma conta paga |
| `flutter doctor` mostra ✗ para sua plataforma | Falta Android SDK / Xcode / CocoaPods | Siga exatamente a linha que o `flutter doctor` imprime |

---

## Veja também

- [getting-started.md](getting-started.md) — suba o gateway ao qual o app se conecta
- [operator-guide.md](operator-guide.md) — topologia de reverse proxy / tunnel para acesso fora da LAN
- [Flutter — build & release Android](https://docs.flutter.dev/deployment/android)
- [Flutter — build & release iOS](https://docs.flutter.dev/deployment/ios)
