<p align="center">
  <a href="https://opendray.dev"><img src="docs/assets/logo.png" alt="opendray" width="180"></a>
</p>

<h1 align="center">opendray</h1>

<p align="center">
  <strong>Gateway self-hosted pour Claude Code · Codex · Gemini · shell — avec une couche de mémoire local-first partagée entre tous.</strong>
  <br/>
  <sub>Fais tourner tes sessions sur ta propre infra. Pilote depuis le web, le mobile ou le chat. API REST + WebSocket ouverte pour les intégrations.</sub>
</p>

<p align="center">
  <strong><a href="https://opendray.dev">🌐 opendray.dev</a></strong>
</p>

<p align="center">
  <a href="https://opendray.dev"><img alt="Website" src="https://img.shields.io/badge/website-opendray.dev-F43F5E"></a>
  <a href="https://github.com/Opendray/opendray/releases/latest"><img alt="Latest release" src="https://img.shields.io/github/v/release/Opendray/opendray?label=release&color=4f46e5"></a>
  <a href="LICENSE"><img alt="License Apache 2.0" src="https://img.shields.io/github/license/Opendray/opendray?color=blue"></a>
  <a href="https://github.com/Opendray/opendray/actions/workflows/ci.yml"><img alt="CI" src="https://img.shields.io/github/actions/workflow/status/Opendray/opendray/ci.yml?branch=main&label=CI"></a>
  <a href="https://github.com/Opendray/opendray/discussions"><img alt="Discussions" src="https://img.shields.io/github/discussions/Opendray/opendray?color=ec4899"></a>
  <br/>
  <img alt="Go" src="https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go&logoColor=white">
  <img alt="React" src="https://img.shields.io/badge/React-19-61DAFB?logo=react&logoColor=black">
  <img alt="Flutter" src="https://img.shields.io/badge/Flutter-mobile-02569B?logo=flutter&logoColor=white">
  <img alt="Postgres" src="https://img.shields.io/badge/PostgreSQL-15%2F16%2F17-336791?logo=postgresql&logoColor=white">
</p>

<p align="center">
  🌐 <a href="README.md">English</a> · <a href="README.zh.md">简体中文</a> · <a href="README.fa.md">فارسی</a> · <a href="README.es.md">Español</a> · <a href="README.pt-BR.md">Português</a> · <a href="README.ja.md">日本語</a> · <a href="README.ko.md">한국어</a> · <strong>Français</strong> · <a href="README.de.md">Deutsch</a> · <a href="README.ru.md">Русский</a>
</p>

---

## Pourquoi opendray existe

Trois frictions du quotidien avec les CLI d'IA pour le code que opendray vient corriger.

**Tes sessions meurent quand ton laptop se met en veille.** Faire tourner Claude Code ou Codex via SSH, c'est voir l'agent mourir à la seconde où tu rabats l'écran ou perds le Wi-Fi. Le contexte, les tool calls en cours, le diff partiel que tu allais relire. Envolés. opendray exécute l'agent sur un host qui ne dort jamais (un Mac mini sous ton bureau, un NAS, un VPS) et te laisse t'y rattacher depuis une admin web, une app mobile Flutter ou un message de chat. La session continue de s'exécuter, que quelqu'un soit connecté ou non.

**Atteindre une rate limit ne devrait pas tuer ce que tu étais en train de faire.** Si tu as plusieurs comptes Anthropic (pro + perso, plan famille + Pro), opendray les traite comme un pool : il expose tier, quota et nombre de sessions actives par compte, équilibre les nouvelles sessions entre eux, et te permet de basculer une session en cours vers un autre compte sans perdre la conversation. Le transcript te suit. Même logique pour les comptes Codex et Gemini.

**La mémoire est une couche de premier ordre, pas un ajout après coup.** La plupart des CLI d'IA ré-indexent le contexte projet à zéro à chaque session, brûlant des tokens en retrieval répété. opendray embarque un vector store local-first (embeddings ONNX / Ollama / LM Studio) avec un retrieval sur trois domaines (utilisateur, projet, session), plus une détection de drift entre les couches. Chaque octet reste sur ton réseau.

---

## C'est quoi opendray ?

**opendray** enveloppe les CLI de coding IA que tu utilises déjà — Claude Code, Codex, Gemini, plus n'importe quel shell — et les transforme en quelque chose que tu peux piloter depuis n'importe où. Fais tourner tes sessions sur ton home server / NAS / VPS, reçois une notification Telegram quand l'une d'elles devient inactive, réponds depuis ton téléphone pour relancer le prochain prompt, le tout via un gateway self-hosted que tu contrôles de bout en bout.

- 🛰 **Un backend, trois surfaces** — un seul binaire Go qui sert un admin web React et une app mobile Flutter, avec chaque action également exposée via une API REST + WebSocket pour des intégrations tierces.
- 💬 **Six canaux bidirectionnels, pas de walled gardens** — Telegram, Slack, Discord, Feishu (飞书), DingTalk (钉钉), WeCom (企业微信), plus un adaptateur Bridge pour tout ce qui est custom. Les réponses sur n'importe quel canal sont routées vers la bonne session.
- 🧠 **Mémoire local-first** — embeddings ONNX / Ollama / LM Studio avec recherche sur trois scopes (user · projet · session), ranking intelligent et détection des conflits entre couches. Aucune donnée vectorielle ne quitte ton réseau.
- 🔌 **API de niveau intégration** — clés d'API scopées, audit log par appel, montages reverse-proxy. Traite opendray comme le gateway derrière ton propre produit, ou simplement comme un centre de commande personnel.
- 🔑 **Flotte multi-comptes Claude** — ajoute plusieurs comptes `claude login` dans le gateway ; le panel les découvre automatiquement via un filesystem watcher, répartit les nouvelles sessions entre les comptes activés, et te permet de basculer une session en vie d'un compte à l'autre **sans perdre la conversation** (le transcript est migré sous le capot). Chaque ligne de compte affiche la capacité en temps réel (subscription tier, rate-limit tier, sessions actives, dernière utilisation, email Anthropic courant) pour que tu choisisses le bon d'un coup d'œil.
- 🔒 **Self-hosted, licence claire** — Apache 2.0, un binaire statique, releases signées avec cosign + SBOM SPDX. Pas de télémétrie, pas de compte cloud, pas d'abonnement.

## État

**v2.7.0** (dernière) — la génération v2 continue d'itérer. Voir
[`VERSIONING.md`](VERSIONING.md) pour la politique major-comme-génération
(major = génération de produit, pas un « breaking change » strict au sens SemVer) et
[`CHANGELOG.md`](CHANGELOG.md) pour l'historique complet des releases.

Cette génération embarque :

- **Assistants d'installation et de désinstallation en une ligne** (Linux + macOS ;
  Windows passe par WSL2). Guide l'opérateur à travers le bootstrap de Postgres,
  l'installation des AI-CLI, les credentials admin, l'adresse d'écoute,
  l'installation du binaire, la migration du schéma et l'enregistrement du service.
- **Binaire auto-géré** — `opendray update / start / stop /
  restart / status / providers list / providers update`, pour que les opérateurs
  ne touchent plus à `systemctl` / `launchctl` pour les opérations courantes.
- **Pipeline de release Goreleaser** — binaires cross-compilés
  (linux/darwin × amd64/arm64), signature keyless cosign (Sigstore),
  SBOM SPDX, self-update vérifié atomiquement.

## Installation

### Installeur en une ligne

**Linux / macOS / WSL2**

```sh
curl -fsSL https://raw.githubusercontent.com/Opendray/opendray/main/scripts/install.sh | bash
```

**Windows** — met d'abord WSL2 en place, puis lance l'installeur Linux à l'intérieur. [détails →](scripts/README.md#windows)

```powershell
irm https://raw.githubusercontent.com/Opendray/opendray/main/scripts/install-windows.ps1 | iex
```

Déroule la mise en place de Postgres, l'installation des AI-CLI, les credentials admin et l'enregistrement du service — pour un gateway en marche en ~5-10 minutes. Voir [**`scripts/README.md`**](scripts/README.md) pour ce que fait l'assistant, le layout de fichiers qu'il crée, les options et le troubleshooting.

> **Tu préfères la procédure manuelle ?** Lis [**docs/getting-started.md**](docs/getting-started.md) — un guide end-to-end de 15 minutes qui reproduit ce que fait l'assistant pour que tu puisses vérifier chaque étape toi-même.

### npm / npx (Node ≥ 18)

Installe globalement et ajoute `opendray` au `PATH` :

```sh
npm install -g opendray
```

Ou exécute-le à la demande sans installer :

```sh
npx opendray
```

Installe **uniquement le binaire** — sans assistant, sans enregistrement de service, sans Postgres. Le paquet récupère le binaire de plateforme correspondant (`opendray-{linux,darwin}-{x64,arm64}`) via `optionalDependencies` (le pattern esbuild / Biome — pas de `postinstall`, pas d'appel réseau au moment de l'install). Adapté aux environnements scriptés, runners éphémères, ou si tu as déjà ton propre Postgres et ton propre superviseur de process.

Tu amènes quand même une base de données et tu démarres le gateway toi-même :

```sh
# 1. PostgreSQL 15+ avec pgvector — pointe un DSN dessus, définis un mot de passe admin.
export OPENDRAY_DATABASE_URL="postgres://opendray:pw@127.0.0.1:5432/opendray?sslmode=disable"
export OPENDRAY_ADMIN_PASSWORD="$(openssl rand -base64 24)"
# 2. Applique le schéma, puis lance (foreground).
opendray migrate
opendray serve        # → http://127.0.0.1:8770/admin/
```

Procédure complète — setup pgvector, `config.toml`, lancer comme service systemd / launchd, et mises à jour — dans [**docs/install-binary.fr.md**](docs/install-binary.fr.md).

### Désinstallation (Linux / macOS)

**Par défaut** — arrête le gateway et supprime le binaire, mais **conserve** ton `config.toml`, le répertoire de données (keyfile bcrypt, sessions, notes, vault), les logs et la base PostgreSQL pour qu'une réinstallation reprenne là où tu en étais :

```sh
curl -fsSL https://raw.githubusercontent.com/Opendray/opendray/main/scripts/uninstall.sh | bash
```

**Purge complète** — supprime aussi la base PG + le rôle, efface config / data / logs et retire le service user. Inclut une étape de vérification post-suppression qui plante bruyamment si quelque chose a survécu :

```sh
curl -fsSL https://raw.githubusercontent.com/Opendray/opendray/main/scripts/uninstall.sh | OPENDRAY_PURGE=1 bash
```

### Commandes du quotidien

Après l'installation, le binaire `opendray` gère son propre cycle de vie — pas besoin de te souvenir des incantations `systemctl` / `launchctl` :

```sh
sudo opendray update --restart   # download latest release, verify SHA, atomic replace + restart
```

```sh
sudo opendray providers update   # bump installed AI CLIs (claude / codex / gemini) to npm-latest
```

```sh
opendray providers list          # see which AI CLIs are installed + their versions
```

```sh
sudo opendray start              # start | stop | restart | status — wraps systemd / launchd
```

`opendray --help` liste l'ensemble des sous-commandes.

### Choisir son chemin de deploy

Chaque chemin supporté inclut le spawn de session, l'accès aux AI-CLI, les backups chiffrés et l'API d'intégration complète. opendray est un gateway host-resident — il spawn les AI CLI via des PTY et partage l'état des process (`~/.claude`, ssh-agent, fichiers projet) avec eux. Ce modèle est incompatible avec l'isolation conteneur qu'imposerait Docker en production, donc Docker n'est pas un chemin de déploiement supporté pour v2.x.

| Chemin | Idéal pour | Aller à |
|---|---|---|
| 📦 **Binaire pré-construit** | « Just run it » — Linux / macOS, n'importe quel superviseur | [Page des releases](https://github.com/Opendray/opendray/releases) → voir [Déploiement en production](#production-deploy) |
| 🐧 **Unit systemd** | Linux bare-metal / VM / LXC | [Déploiement en production §A](#option-a--systemd-bare-metal--vm--lxc) |
| 🍎 **LaunchDaemon macOS** | Mac mini / Mac Studio en serveur maison | [Déploiement en production §C](#option-c--macos-launchd-mac-mini--studio-as-home-server) |
| 🛠 **Build depuis les sources** | Dev / contribution / builds custom | [Quickstart](#quickstart-5-minute-dev-path) plus bas |

<a id="quickstart-5-minute-dev-path"></a>

## Quickstart (chemin dev en 5 minutes)

Pour la procédure complète avec prérequis et troubleshooting, voir [`docs/quickstart.md`](docs/quickstart.md). La version condensée pour les devs :

```bash
# 1. Have a Postgres 15+ running on 127.0.0.1:5432 with pgvector enabled
#    (apt install postgresql-16 postgresql-16-pgvector / brew install postgresql@16 pgvector).
#    Point [database].url at any other DSN if you'd rather use a remote PG.

# 2. Local config — already gitignored.
cp config.example.toml config.toml
$EDITOR config.toml          # set [database].url, [admin].password

# 3. Build the web bundle into the embed tree.
cd app/web && pnpm install && pnpm build && cd ../..

# 4. Apply schema.
go run ./cmd/opendray migrate -config config.toml

# 5. Run.
go run ./cmd/opendray serve -config config.toml
# → REST + WS:  http://127.0.0.1:8770/api/v1/...
# → Web admin:  http://127.0.0.1:8770/admin/
```

Ça fait tourner OpenDray en foreground — Ctrl-C l'arrête. Pour un daemon
long-running, voir **Déploiement en production** plus bas.

<a id="production-deploy"></a>

## Déploiement en production

Quatre chemins de déploiement supportés, choisis celui qui colle à ton environnement.
Chacun te donne auto-restart en cas de crash, état persistant et
séparation des secrets et du config.

### Option A — systemd (bare-metal / VM / LXC)

Le chemin de déploiement recommandé sur Linux. Embarque une unit hardenée dans
[`deploy/systemd/opendray.service`](deploy/systemd/opendray.service)
avec sandboxing (`ProtectSystem=strict`, `NoNewPrivileges`,
`MemoryDenyWriteExecute`, capability scrub), boot `migrate`-puis-`serve`,
et une fenêtre de graceful stop de 20 s.

**Récupère d'abord un binaire.** Soit attrape une archive pré-construite depuis la
[page des releases](https://github.com/Opendray/opendray/releases)
(`opendray_*_linux_<arch>.tar.gz` — décompresse en un seul binaire `opendray`),
soit build depuis les sources via le [Quickstart](#quickstart-5-minute-dev-path)
plus haut (`go build ./cmd/opendray`).

```bash
# 1. Install the binary you just grabbed (or built).
sudo install -m 0755 /path/to/opendray /usr/local/bin/opendray

# 2. Create the service user + state dir.
sudo useradd -r -s /usr/sbin/nologin -d /var/lib/opendray opendray
sudo install -d -o opendray -g opendray -m 0700 /var/lib/opendray

# 3. Drop config + secrets (root-owned; mode 0640).
sudo install -D -m 0640 config.example.toml /etc/opendray/config.toml
sudo $EDITOR /etc/opendray/config.toml             # set [database].url etc.
sudo install -D -m 0640 -o root -g opendray /dev/null /etc/opendray/env.d/secrets
sudo $EDITOR /etc/opendray/env.d/secrets           # OPENDRAY_ADMIN_PASSWORD=…

# 4. Install + enable the unit.
sudo cp deploy/systemd/opendray.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now opendray

# 5. Verify.
sudo systemctl status opendray
sudo journalctl -u opendray -f --no-pager
```

L'unit lance `opendray migrate` en `ExecStartPre`, donc le premier boot
applique toutes les migrations avant que `serve` ne démarre. Les redémarrages sont
en `on-failure` avec un back-off de 5 s et une limite de 5 rafales par minute.

### Option B — Binaire direct + ton propre superviseur de process

Pour LXC sans systemd, FreeBSD `rc.d`, OpenRC ou autre chose.
Build une fois, lance avec le superviseur que tu utilises déjà :

```bash
# Cross-compile a release archive locally:
goreleaser release --clean --snapshot
ls dist/                  # opendray_*_linux_amd64.tar.gz etc.

# Or grab a published release artefact:
# https://github.com/Opendray/opendray/releases
```

Pointe ensuite ton superviseur (s6, runit, supervisord, runwhen) sur :

```
/usr/local/bin/opendray serve -config /etc/opendray/config.toml
```

Pre-flight : lance `opendray migrate -config /etc/opendray/config.toml`
une fois avant le premier `serve`, ou en hook pre-start dans le superviseur
de ton choix.

<a id="option-c--macos-launchd-mac-mini--studio-as-home-server"></a>

### Option C — launchd macOS (Mac mini / Studio en serveur maison)

Pour les Mac mini / Mac Studio Apple Silicon qui tournent 24/7. Embarque un
LaunchDaemon dans
[`deploy/launchd/com.opendray.opendray.plist`](deploy/launchd/com.opendray.opendray.plist)
qui démarre au boot avant tout login utilisateur, redémarre sur crash avec
un throttle de 5 s, et log dans `/usr/local/var/log/opendray/`.

```bash
# 1. Install the darwin binary + config + state dirs.
sudo install -m 0755 ./opendray /usr/local/bin/opendray
sudo install -d -m 0755 \
  /usr/local/etc/opendray \
  /usr/local/var/lib/opendray \
  /usr/local/var/log/opendray
sudo install -m 0640 config.example.toml /usr/local/etc/opendray/config.toml
sudo $EDITOR /usr/local/etc/opendray/config.toml    # set [database].url etc.

# 2. Apply migrations once.
sudo /usr/local/bin/opendray migrate \
  -config /usr/local/etc/opendray/config.toml

# 3. Install + load the LaunchDaemon.
sudo cp deploy/launchd/com.opendray.opendray.plist /Library/LaunchDaemons/
sudo chown root:wheel /Library/LaunchDaemons/com.opendray.opendray.plist
sudo chmod 0644 /Library/LaunchDaemons/com.opendray.opendray.plist
sudo launchctl bootstrap system /Library/LaunchDaemons/com.opendray.opendray.plist

# 4. Verify.
sudo launchctl print system/com.opendray.opendray
tail -f /usr/local/var/log/opendray/opendray.log
```

Redémarre avec `sudo launchctl kickstart -k system/com.opendray.opendray` ;
unload complètement avec `sudo launchctl bootout system/com.opendray.opendray`.

Postgres sur macOS — installe via Homebrew (`brew install postgresql@17 && brew services start postgresql@17`) et pointe `[database].url` sur
`postgres://$USER@127.0.0.1:5432/opendray`. Ajoute `pgvector` avec
`brew install pgvector` puis `CREATE EXTENSION vector` à l'intérieur de la
base opendray.

---

Pour les notes spécifiques à LXC sur Proxmox (PTY dans des conteneurs unprivileged,
networking, tweaks cgroup), voir [`deploy/lxc/proxmox-pty-notes.md`](deploy/lxc/proxmox-pty-notes.md).

Pour la terminaison reverse-proxy / TLS (nginx, Caddy, Traefik, Cloudflare
Tunnel), voir [`docs/operator-guide.md`](docs/operator-guide.md) §Topology.

### Optionnel : activer les backups DB chiffrés + les exports de données

```bash
# Master passphrase (env-only — never write into config.toml).
export OPENDRAY_BACKUP_KEY="$(openssl rand -base64 32)"
export OPENDRAY_BACKUP_ENABLED=1

# pg_dump / pg_restore must match the server's major version. On
# Apple Silicon dev machines pointing at a PG17 server:
export OPENDRAY_BACKUP_PG_DUMP_PATH=/opt/homebrew/opt/postgresql@17/bin/pg_dump
export OPENDRAY_BACKUP_PG_RESTORE_PATH=/opt/homebrew/opt/postgresql@17/bin/pg_restore
```

Redémarre opendray ; la sidebar fait apparaître une page Backups (`/backups`)
pour les dumps PostgreSQL chiffrés + restore, et `/export` pour les
exports de données en bundle zip + import. Voir [`docs/operator-guide.md`](docs/operator-guide.md) §Backup pour le cycle complet.

Un seul binaire Go embarque tout le bundle web — pas de runtime Node
à l'exécution, pas de serveur de fichiers statiques séparé, pas de Caddy/nginx
nécessaire. Cloudflare Tunnel termine TLS devant `:8770`.

## Layout

```
cmd/opendray/        binary entry point (≤100 LOC per design §14)
internal/
├── app/             composition root (wires every subsystem)
├── audit/           subscribes to bus topics, persists to audit_log
├── auth/            admin bearer tokens (M2.5)
├── backup/          encrypted DB dumps + admin export/import├── catalog/         CLI provider manifests + per-id user config (M2)
├── channel/         channel hub + telegram impl (M4)
├── config/          TOML loader with OPENDRAY_* env overrides
├── eventbus/        in-process pub/sub
├── gateway/         chi HTTP router + middleware + slog
├── integration/     external-app registry + reverse proxy + events WS (M3)
├── memory/          cross-CLI persistent memory├── session/         PTY lifecycle + ring buffer + WS stream (M1)
├── store/           pgx pool + hand-rolled migration runner (M0)
├── version/         build-time identification
└── web/             go:embed of the web bundle (W5)

app/web/             React 19 + TypeScript + Vite SPA (Phase 2 W0-W5)
app/mobile/          Flutter app (iOS + Android), feature parity with web
docs/
├── design.md        SSOT north-star
└── adr/             architecture decisions, dated
```

## Frontend web

`app/web/` construit une SPA unique dans `internal/web/dist/`, que le binaire Go
embarque et sert sur `/admin/*`. Le dev server Vite sur `:5173` proxy `/api`
vers `:8770` pour un développement avec HMR.

```bash
# dev (hot reload on the React side, separate Go server for the API)
cd app/web && pnpm dev               # http://localhost:5173
go run ./cmd/opendray serve -config ../../config.toml   # other terminal

# prod (one binary delivers everything)
cd app/web && pnpm build              # writes ../../internal/web/dist
cd ../..
go build ./cmd/opendray               # bakes dist into the binary
./opendray serve -config config.toml
```

Voir [`app/web/README.md`](app/web/README.md) pour la stack frontend
(React + Vite + Tailwind v4 + shadcn/ui + TanStack Router/Query +
Zustand + xterm.js) et les notes par milestone W.

## Documentation

- [`docs/getting-started.md`](docs/getting-started.md) — **commence ici** si tu débutes : de zéro à ta première session en 15 minutes, installation des CLI wrappées et bootstrap Postgres compris
- [`docs/install-binary.fr.md`](docs/install-binary.fr.md) — installer depuis le paquet npm ou un binaire de release (amène ton propre Postgres) et le lancer comme service systemd / launchd
- [`docs/quickstart.md`](docs/quickstart.md) — environnement de dev en 5 minutes (suppose que tu connais déjà les morceaux)
- [`docs/operator-guide.md`](docs/operator-guide.md) — référence deploy + ops pour les setups quasi-production
- [`docs/integration-guide.md`](docs/integration-guide.md) — comment écrire une intégration externe dans n'importe quel langage
- [`VERSIONING.md`](VERSIONING.md) — stratégie de versioning (major-as-generation)
- [`CHANGELOG.md`](CHANGELOG.md) — historique des releases

## Tests

```bash
go test -race ./...        # backend
cd app/web && pnpm build   # web (TS strict + vite production build)
```

Les smoke flows end-to-end sont trackés dans les messages de commit par milestone.
Un harness Playwright est prévu en follow-up.

## Lien avec la v1

v1 (`Opendray/opendray`) est le codebase legacy, désormais archivé. v2 est
la génération actuelle et active — feature-complete et la seule branche qui
reçoit du développement. Sur les 16 builtins de v1, quatre ont migré dans
le backend v2 ; le reste est devenu des features côté client, des adaptateurs
de canaux ou des consommateurs de l'API d'intégration.

## Licence

Apache 2.0 — voir [`LICENSE`](LICENSE). (v1 était sous MIT ; v2 est licenciée
indépendamment.)
