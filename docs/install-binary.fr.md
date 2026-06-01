# Installer et lancer depuis un binaire précompilé

Pour quand tu as déjà — ou que tu veux — juste le binaire `opendray`, sans
assistant d'installation qui touche ta machine. C'est le chemin pour :

- **`npm install -g opendray` / `npx opendray`** — le paquet npm embarque le
  binaire officiel de la release Go (voir [README → npm / npx](../README.fr.md#npm--npx-node--18)).
- **Téléchargement depuis les releases** — récupère `opendray_*_<os>_<arch>.tar.gz` depuis la
  [page des releases](https://github.com/Opendray/opendray/releases).
- **Environnements scriptés / éphémères** — runners CI, images golden, gestion
  de configuration (Ansible, Nix, Docker), ou tout hôte où tu as déjà ton propre
  Postgres et ton propre superviseur de process.

Le binaire *est* le gateway entier — le SPA d'administration web est embarqué, donc
il n'y a pas de runtime Node, pas de serveur de fichiers statiques séparé, et rien
à construire. Ce qu'il ne fait **pas**, c'est tout configurer pour toi. C'est le
compromis : tu amènes une base PostgreSQL et un moyen de maintenir le process en
marche, et en échange rien n'est installé, configuré ou enregistré dans ton dos.

> **Tu préfères tout faire faire à ta place ?** Sur une machine Linux / macOS
> fraîche, l'installeur en une ligne provisionne Postgres, installe les AI-CLI,
> écrit la config et enregistre un service en ~5–10 minutes. Voir
> [README → Installeur en une ligne](../README.fr.md#installation) ou le guide
> manuel [getting-started.md](getting-started.md).

Ce guide t'emmène de « binaire dans le `PATH` » à « gateway en marche » en cinq
étapes, puis te montre comment le faire tourner comme service.

---

## Étape 1 — Obtenir le binaire

### Via npm (n'importe quel OS avec Node ≥ 18)

```sh
npm install -g opendray        # installation globale, met `opendray` dans le PATH
# ou, sans installer :
npx opendray --help
# ou avec un autre gestionnaire de paquets :
pnpm add -g opendray
yarn global add opendray
```

Le bon binaire de plateforme (`opendray-{linux,darwin}-{x64,arm64}`) est sélectionné
automatiquement via `optionalDependencies` — il n'y a pas de hook `postinstall` et
pas d'appel réseau au moment de l'install. Ne passe **pas** `--no-optional` : ça
saute le paquet de plateforme et laisse le lanceur sans binaire à exécuter.

### Via une archive de release

```sh
# Choisis l'archive correspondant à ton OS/arch depuis la page des releases, puis :
tar -xzf opendray_*_linux_amd64.tar.gz
sudo install -m 0755 opendray /usr/local/bin/opendray
```

### Vérification

```sh
opendray version          # affiche la version, le commit, la date de build
opendray --help           # liste toutes les sous-commandes
```

Plateformes supportées : **Linux** (x64, arm64) et **macOS** (x64, arm64).
Windows natif n'est pas packagé — utilise WSL2 et suis le chemin Linux.

---

## Étape 2 — Fournir PostgreSQL 15+ avec pgvector

opendray stocke tout (sessions, mémoire, audit log) dans PostgreSQL, et son
sous-système de mémoire a besoin de l'extension
[`pgvector`](https://github.com/pgvector/pgvector).
Versions de serveur supportées : **15, 16, 17**.

Si tu as déjà Postgres, crée une base et un rôle limité au CRUD, puis active
l'extension une fois avec un superuser :

```sh
# 1. Installe pgvector (une fois par hôte).
#    Ubuntu/Debian :  sudo apt install postgresql-16-pgvector
#    macOS (brew) :   brew install pgvector
#    Autre / source : https://github.com/pgvector/pgvector#installation

# 2. Crée la base + un rôle scopé au projet.
sudo -u postgres psql <<'SQL'
CREATE DATABASE opendray;
CREATE USER opendray WITH ENCRYPTED PASSWORD 'change-me';
GRANT ALL PRIVILEGES ON DATABASE opendray TO opendray;
SQL

# 3. Active pgvector dans cette base (unique, nécessite un superuser).
sudo -u postgres psql -d opendray -c 'CREATE EXTENSION IF NOT EXISTS vector;'
sudo -u postgres psql -d opendray -c 'GRANT ALL ON SCHEMA public TO opendray;'
```

Une fois l'extension en place, le rôle CRUD d'opendray lance les migrations sans
aucun accès superuser supplémentaire. **Ne pointe jamais opendray sur un rôle
superuser en production** — donne-lui un compte scopé au projet et fais tourner
son mot de passe hors bande.

---

## Étape 3 — Configurer

opendray lit sa config depuis un fichier TOML **ou** uniquement depuis des
variables d'environnement (12-factor) — les variables d'environnement ont toujours
la priorité sur le fichier. La seule exigence absolue est l'URL de la base de
données ; tout le reste a une valeur par défaut.

### Option A — variables d'environnement (adapté aux conteneurs / hôtes éphémères)

```sh
export OPENDRAY_DATABASE_URL="postgres://opendray:change-me@127.0.0.1:5432/opendray?sslmode=disable"
export OPENDRAY_ADMIN_PASSWORD="$(openssl rand -base64 24)"   # mot de passe admin
export OPENDRAY_LISTEN="127.0.0.1:8770"                       # optionnel ; c'est la valeur par défaut
```

| Variable | Requis | Défaut | Rôle |
|---|---|---|---|
| `OPENDRAY_DATABASE_URL` | **oui** | — | DSN Postgres |
| `OPENDRAY_ADMIN_PASSWORD` | recommandé | — | Mot de passe admin web/mobile |
| `OPENDRAY_ADMIN_USER` | non | `admin` | Nom d'utilisateur admin |
| `OPENDRAY_LISTEN` | non | `127.0.0.1:8770` | Adresse d'écoute |
| `OPENDRAY_LOG_LEVEL` | non | `info` | `debug`/`info`/`warn`/`error` |
| `OPENDRAY_LOG_FORMAT` | non | `text` | `text`/`json` |

Lance `opendray serve` sans flag `-config` et il charge entièrement depuis
l'environnement.

### Option B — config.toml

```sh
curl -fsSLO https://raw.githubusercontent.com/Opendray/opendray/main/config.example.toml
mv config.example.toml config.toml
$EDITOR config.toml        # renseigne [database].url et [admin].password
```

Le minimum à éditer :

```toml
listen = "127.0.0.1:8770"

[database]
url = "postgres://opendray:change-me@127.0.0.1:5432/opendray?sslmode=disable"

[admin]
user     = "admin"
password = "use-a-real-password"
```

Voir [`config.example.toml`](../config.example.toml) pour le fichier entièrement
annoté (logging, détection d'inactivité de session, backups, vault, MCP). Passe-le
avec `-config config.toml` aux commandes ci-dessous. Garde les secrets hors du TOML
sur les hôtes partagés — renseigne `OPENDRAY_DATABASE_URL` / `OPENDRAY_ADMIN_PASSWORD`
via l'environnement et laisse le fichier non secret.

---

## Étape 4 — Appliquer le schéma

```sh
opendray migrate                          # config uniquement via env
# ou
opendray migrate -config config.toml
```

Idempotent — relancer est un no-op une fois le schéma à jour. Cette commande
doit réussir avant le premier `serve`.

---

## Étape 5 — Lancer

```sh
opendray serve                            # config uniquement via env
# ou
opendray serve -config config.toml
```

Cela tourne en **foreground** (Ctrl-C l'arrête). Tu devrais maintenant avoir :

| URL | Ce que c'est |
|---|---|
| `http://127.0.0.1:8770/admin/` | Admin web — connecte-toi avec `admin` + ton mot de passe |
| `http://127.0.0.1:8770/api/v1/...` | API REST + WebSocket |

C'est un gateway complet et opérationnel. Pour tout ce qui dépasse un test rapide,
lance-le sous un superviseur pour qu'il survive aux reboots et redémarre en cas
de crash — voir la suite.

---

## Lancer comme service

`opendray serve` est exactement la commande de démarrage qu'une unit de service
devrait appeler. opendray embarque des units robustes prêtes à l'emploi ; les étapes
ci-dessous sont les mêmes que
[README → Déploiement en production](../README.fr.md#déploiement-en-production),
qui est la référence faisant autorité (bootstrap complet, notes de sandboxing,
reverse-proxy/TLS).

### Linux — systemd

Le dépôt embarque une unit hardenée dans
[`deploy/systemd/opendray.service`](../deploy/systemd/opendray.service)
(lance `migrate` en `ExecStartPre`, secrets via un `EnvironmentFile`,
redémarrage `on-failure`, sandboxing des appels système et du système de fichiers).

```sh
# Binaire dans /usr/local/bin/opendray, service user, répertoire d'état :
sudo useradd -r -s /usr/sbin/nologin -d /var/lib/opendray opendray
sudo install -d -o opendray -g opendray -m 0700 /var/lib/opendray

# Config (non secrète) + fichier de secrets (env, mode 0640) :
sudo install -D -m 0640 config.toml /etc/opendray/config.toml
sudo install -D -m 0640 -o root -g opendray /dev/null /etc/opendray/env.d/secrets
echo 'OPENDRAY_ADMIN_PASSWORD=use-a-real-password' | sudo tee -a /etc/opendray/env.d/secrets

# Installer + activer l'unit :
sudo cp deploy/systemd/opendray.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now opendray
sudo systemctl status opendray
journalctl -u opendray -f --no-pager
```

Pas de systemd ? (LXC sans systemd, OpenRC, runit, s6, supervisord…) Pointe ton
superviseur sur `opendray serve -config /etc/opendray/config.toml` et lance
`opendray migrate` une fois en étape pre-start. Voir
[README → Déploiement en production §B](../README.fr.md#option-b--binaire-direct--ton-propre-superviseur-de-process).

### macOS — launchd

Le dépôt embarque un LaunchDaemon dans
[`deploy/launchd/com.opendray.opendray.plist`](../deploy/launchd/com.opendray.opendray.plist)
(démarre au boot, redémarre sur crash, logs dans `/usr/local/var/log/opendray/`).

```sh
sudo install -d -m 0755 /usr/local/etc/opendray /usr/local/var/lib/opendray /usr/local/var/log/opendray
sudo install -m 0640 config.toml /usr/local/etc/opendray/config.toml
sudo /usr/local/bin/opendray migrate -config /usr/local/etc/opendray/config.toml

sudo cp deploy/launchd/com.opendray.opendray.plist /Library/LaunchDaemons/
sudo chown root:wheel /Library/LaunchDaemons/com.opendray.opendray.plist
sudo chmod 0644 /Library/LaunchDaemons/com.opendray.opendray.plist
sudo launchctl bootstrap system /Library/LaunchDaemons/com.opendray.opendray.plist
sudo launchctl print system/com.opendray.opendray
```

Redémarre : `sudo launchctl kickstart -k system/com.opendray.opendray`.
Décharge : `sudo launchctl bootout system/com.opendray.opendray`.

> Les deux units sont documentées en détail — notamment le layout des secrets et
> pourquoi `MemoryDenyWriteExecute` est laissé désactivé — dans
> [`deploy/README.md`](../deploy/README.md).

---

## Mises à jour

La façon de mettre à jour dépend de la méthode d'installation :

- **Installé via npm** — mets à jour avec ton gestionnaire de paquets. `opendray update`
  remplacerait le binaire *à l'intérieur* de `node_modules` dans le dos de npm et serait
  écrasé au prochain install, donc ne l'utilise pas ici.

  ```sh
  npm install -g opendray@latest
  ```

- **Téléchargement de release / install par l'assistant** — le binaire se met à jour
  lui-même en place (télécharge la dernière release, vérifie son SHA-256, se remplace
  atomiquement) :

  ```sh
  opendray update --check          # sonde la version uniquement, sans appliquer
  sudo opendray update --restart   # applique, puis redémarre le service
  ```

---

## Dépannage

**`the matching platform package "opendray-…" was not installed`**
npm a été lancé avec `--no-optional`, ou l'installation a été interrompue. Relance
`npm install -g opendray` (sans `--no-optional`).

**`unsupported platform`**
Le paquet npm couvre Linux/macOS sur x64/arm64 uniquement. Sur d'autres cibles,
build depuis les sources — voir [quickstart.md](quickstart.md).

**`config: database.url is empty`**
Ni `OPENDRAY_DATABASE_URL` ni `[database].url` n'est renseigné. Définis l'un des deux (Étape 3).

**`connection refused` au moment de migrate/serve**
Postgres ne tourne pas ou le DSN est incorrect. Vérifie que le serveur est actif et
que le host/port/identifiants dans ton DSN sont corrects.

**pgvector / `extension "vector" is not available`**
L'extension n'est pas installée sur le serveur, ou n'a pas été activée dans la base
opendray. Recommence l'Étape 2 (installe le paquet OS, puis
`CREATE EXTENSION vector` en superuser).

**Port déjà utilisé**
Change `OPENDRAY_LISTEN` (ou `listen` dans config.toml) vers un port disponible.

---

## Prochaines étapes

- [README → Déploiement en production](../README.fr.md#déploiement-en-production) — référence
  de déploiement complète (systemd / launchd / superviseur custom, hardening, reverse proxy)
- [`docs/operator-guide.md`](operator-guide.md) — ops : topologie reverse-proxy/TLS,
  backups DB chiffrés, export/import de données
- [`docs/integration-guide.md`](integration-guide.md) — construire une intégration externe
  contre l'API REST + WebSocket
- [`docs/getting-started.md`](getting-started.md) — la mise en place guidée tout-en-un
  si tu préfères ne pas assembler les pièces toi-même
