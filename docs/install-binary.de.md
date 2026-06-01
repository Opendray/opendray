# Installieren und Ausführen aus einem vorgefertigten Binary

Für den Fall, dass du bereits — oder nur — das `opendray`-Binary haben möchtest,
ohne dass ein Installer-Wizard dein System anfasst. Dies ist der Pfad für:

- **`npm install -g opendray` / `npx opendray`** — das npm-Paket liefert das
  offizielle Go-Release-Binary (siehe [README → npm / npx](../README.de.md#installation)).
- **Release-Downloads** — schnapp dir `opendray_*_<os>_<arch>.tar.gz` von der
  [Releases-Seite](https://github.com/Opendray/opendray/releases).
- **Geskriptete / ephemere Umgebungen** — CI-Runner, Golden Images, Config-
  Management (Ansible, Nix, Docker) oder jeder Host, auf dem du schon dein
  eigenes Postgres und deinen eigenen Process-Supervisor betreibst.

Das Binary ist das *komplette* Gateway — die Web-Admin-SPA ist eingebettet, daher
wird keine Node-Runtime, kein separater Static-Server und nichts zum Bauen
benötigt. Was es **nicht** tut: irgendetwas für dich einrichten. Das ist der
Deal: Du bringst eine PostgreSQL-Datenbank und eine Methode, den Prozess am
Laufen zu halten — und im Gegenzug wird nichts hinter deinem Rücken installiert,
konfiguriert oder registriert.

> **Lieber alles fertig eingerichtet?** Auf einer frischen Linux- / macOS-Box
> stellt der Einzeiler-Installer Postgres bereit, installiert die AI-CLIs,
> schreibt die Konfiguration und registriert einen Service in ~5–10 Minuten. Siehe
> [README → Einzeiliger Installer](../README.de.md#installation) oder den
> manuellen [getting-started.md](getting-started.md)-Walkthrough.

Diese Anleitung führt dich in fünf Schritten von „Binary auf `PATH`" zu
„laufendes Gateway" und zeigt anschließend, wie du es als Service betreibst.

---

## Schritt 1 — Binary beschaffen

### Via npm (beliebiges OS mit Node ≥ 18)

```sh
npm install -g opendray        # globale Installation, legt `opendray` auf den PATH
# oder, ohne zu installieren:
npx opendray --help
# oder mit einem anderen Package-Manager:
pnpm add -g opendray
yarn global add opendray
```

Das richtige Plattform-Binary (`opendray-{linux,darwin}-{x64,arm64}`) wird
automatisch via `optionalDependencies` ausgewählt — es gibt keinen `postinstall`-
Hook und keinen Netzwerk-Call beim Install. Übergib **nicht** `--no-optional`:
das überspringt das Plattform-Paket und lässt den Launcher ohne Binary zum Ausführen
zurück.

### Via Release-Archiv

```sh
# Wähle das Archiv passend zu deinem OS/Arch von der Releases-Seite, dann:
tar -xzf opendray_*_linux_amd64.tar.gz
sudo install -m 0755 opendray /usr/local/bin/opendray
```

### Verifizieren

```sh
opendray version          # gibt Version, Commit, Build-Datum aus
opendray --help           # listet alle Subcommands
```

Unterstützte Plattformen: **Linux** (x64, arm64) und **macOS** (x64, arm64).
Natives Windows ist nicht verpackt — nutze WSL2 und folge dem Linux-Pfad.

---

## Schritt 2 — PostgreSQL 15+ mit pgvector bereitstellen

opendray speichert alles (Sessions, Memory, Audit-Log) in PostgreSQL, und sein
Memory-Subsystem benötigt die [`pgvector`](https://github.com/pgvector/pgvector)-
Extension. Unterstützte Server-Versionen: **15, 16, 17**.

Wenn du bereits Postgres betreibst, erstelle eine Datenbank und eine CRUD-only-
Role, dann aktiviere die Extension einmalig mit einem Superuser:

```sh
# 1. pgvector installieren (einmalig pro Host).
#    Ubuntu/Debian:  sudo apt install postgresql-16-pgvector
#    macOS (brew):   brew install pgvector
#    Other / source: https://github.com/pgvector/pgvector#installation

# 2. Datenbank + eine projektspezifische Role erstellen.
sudo -u postgres psql <<'SQL'
CREATE DATABASE opendray;
CREATE USER opendray WITH ENCRYPTED PASSWORD 'change-me';
GRANT ALL PRIVILEGES ON DATABASE opendray TO opendray;
SQL

# 3. pgvector in der Datenbank aktivieren (einmalig, benötigt Superuser).
sudo -u postgres psql -d opendray -c 'CREATE EXTENSION IF NOT EXISTS vector;'
sudo -u postgres psql -d opendray -c 'GRANT ALL ON SCHEMA public TO opendray;'
```

Sobald die Extension existiert, kann die CRUD-only-Role von opendray Migrationen
durchführen, ohne weiteren Superuser-Zugriff zu benötigen. **Weise opendray zur
Laufzeit niemals eine Superuser-Role zu** — gib ihm ein projektspezifisches Konto
und rotiere dessen Passwort separat.

---

## Schritt 3 — Konfigurieren

opendray liest seine Konfiguration aus einer TOML-Datei **oder** rein aus
Umgebungsvariablen (12-Factor) — Env-Variablen haben immer Vorrang vor der Datei.
Die einzige harte Anforderung ist die Datenbank-URL; alles andere hat einen
Standardwert.

### Option A — Umgebungsvariablen (gut für Container / ephemere Hosts)

```sh
export OPENDRAY_DATABASE_URL="postgres://opendray:change-me@127.0.0.1:5432/opendray?sslmode=disable"
export OPENDRAY_ADMIN_PASSWORD="$(openssl rand -base64 24)"   # Admin-Login
export OPENDRAY_LISTEN="127.0.0.1:8770"                       # optional; das ist der Standardwert
```

| Variable | Erforderlich | Standard | Zweck |
|---|---|---|---|
| `OPENDRAY_DATABASE_URL` | **ja** | — | Postgres-DSN |
| `OPENDRAY_ADMIN_PASSWORD` | empfohlen | — | Web-/Mobile-Admin-Passwort |
| `OPENDRAY_ADMIN_USER` | nein | `admin` | Admin-Benutzername |
| `OPENDRAY_LISTEN` | nein | `127.0.0.1:8770` | Bind-Adresse |
| `OPENDRAY_LOG_LEVEL` | nein | `info` | `debug`/`info`/`warn`/`error` |
| `OPENDRAY_LOG_FORMAT` | nein | `text` | `text`/`json` |

Führe `opendray serve` ohne `-config`-Flag aus und es lädt alles aus der
Umgebung.

### Option B — config.toml

```sh
curl -fsSLO https://raw.githubusercontent.com/Opendray/opendray/main/config.example.toml
mv config.example.toml config.toml
$EDITOR config.toml        # [database].url und [admin].password setzen
```

Das Minimum, das bearbeitet werden muss:

```toml
listen = "127.0.0.1:8770"

[database]
url = "postgres://opendray:change-me@127.0.0.1:5432/opendray?sslmode=disable"

[admin]
user     = "admin"
password = "use-a-real-password"
```

Die vollständig annotierte Datei (Logging, Session-Idle-Erkennung, Backups, Vault,
MCP) findest du in [`config.example.toml`](../config.example.toml). Übergib sie
mit `-config config.toml` an die nachfolgenden Befehle. Halte Secrets auf
gemeinsam genutzten Hosts aus der TOML-Datei heraus — setze `OPENDRAY_DATABASE_URL`
/ `OPENDRAY_ADMIN_PASSWORD` via Env und lass die Datei nicht-geheim.

---

## Schritt 4 — Schema anwenden

```sh
opendray migrate                          # Env-only-Konfiguration
# oder
opendray migrate -config config.toml
```

Idempotent — ein erneuter Aufruf ist ein No-op, sobald das Schema aktuell ist.
Dies muss vor dem ersten `serve` erfolgreich abgeschlossen sein.

---

## Schritt 5 — Ausführen

```sh
opendray serve                            # Env-only-Konfiguration
# oder
opendray serve -config config.toml
```

Dies läuft im **Vordergrund** (Ctrl-C stoppt es). Jetzt solltest du Folgendes haben:

| URL | Inhalt |
|---|---|
| `http://127.0.0.1:8770/admin/` | Web-Admin — melde dich mit `admin` + deinem Passwort an |
| `http://127.0.0.1:8770/api/v1/...` | REST + WebSocket-API |

Das ist ein vollständiges, laufendes Gateway. Für alles über einen schnellen Test
hinaus führe es unter einem Supervisor aus, damit es Neustarts und Abstürze
überlebt — weiter unten.

---

## Als Service betreiben

`opendray serve` ist genau das, was der Start-Befehl einer Service-Unit aufrufen
sollte. opendray liefert gehärtete, betriebsfertige Units mit; die folgenden
Schritte entsprechen dem
[README → Produktions-Deployment](../README.de.md#produktions-deployment), das die
maßgebliche Referenz ist (vollständiger Bootstrap, Sandboxing-Hinweise,
Reverse-Proxy/TLS).

### Linux — systemd

Das Repo liefert eine gehärtete Unit unter
[`deploy/systemd/opendray.service`](../deploy/systemd/opendray.service)
(führt `migrate` als `ExecStartPre` aus, Secrets via `EnvironmentFile`,
`on-failure`-Restart, Syscall-/Filesystem-Sandboxing).

```sh
# Binary unter /usr/local/bin/opendray, Service-User, State-Verzeichnis:
sudo useradd -r -s /usr/sbin/nologin -d /var/lib/opendray opendray
sudo install -d -o opendray -g opendray -m 0700 /var/lib/opendray

# Config (nicht-geheim) + Secrets-Datei (Env, Modus 0640):
sudo install -D -m 0640 config.toml /etc/opendray/config.toml
sudo install -D -m 0640 -o root -g opendray /dev/null /etc/opendray/env.d/secrets
echo 'OPENDRAY_ADMIN_PASSWORD=use-a-real-password' | sudo tee -a /etc/opendray/env.d/secrets

# Unit installieren + aktivieren:
sudo cp deploy/systemd/opendray.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now opendray
sudo systemctl status opendray
journalctl -u opendray -f --no-pager
```

Kein systemd? (LXC ohne es, OpenRC, runit, s6, supervisord…) Weise deinen
Supervisor auf `opendray serve -config /etc/opendray/config.toml` hin und führe
`opendray migrate` einmalig als Pre-Start-Schritt aus. Siehe
[README → Produktions-Deployment §B](../README.de.md#option-b--direktes-binary--dein-eigener-process-supervisor).

### macOS — launchd

Das Repo liefert einen LaunchDaemon unter
[`deploy/launchd/com.opendray.opendray.plist`](../deploy/launchd/com.opendray.opendray.plist)
(startet beim Boot, startet bei Abstürzen neu, loggt nach `/usr/local/var/log/opendray/`).

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

Neustart: `sudo launchctl kickstart -k system/com.opendray.opendray`.
Entladen: `sudo launchctl bootout system/com.opendray.opendray`.

> Beide Units sind vollständig dokumentiert — einschließlich des Secrets-Layouts
> und warum `MemoryDenyWriteExecute` deaktiviert ist — in
> [`deploy/README.md`](../deploy/README.md).

---

## Aktuell halten

Wie du aktualisierst, hängt davon ab, wie du installiert hast:

- **Via npm installiert** — aktualisiere mit deinem Package-Manager. `opendray update`
  würde das Binary *innerhalb* von `node_modules` hinter dem Rücken von npm
  ersetzen und beim nächsten Install überschrieben werden — also nicht hier verwenden.

  ```sh
  npm install -g opendray@latest
  ```

- **Release-Download / Wizard-Installation** — das Binary aktualisiert sich selbst
  in-place (lädt das neueste Release herunter, verifiziert dessen SHA-256, ersetzt
  sich atomar):

  ```sh
  opendray update --check          # nur Versions-Überprüfung ohne Änderung
  sudo opendray update --restart   # anwenden, dann Service neu starten
  ```

---

## Fehlerbehebung

**`the matching platform package "opendray-…" was not installed`**
npm wurde mit `--no-optional` ausgeführt, oder die Installation wurde unterbrochen.
Führe `npm install -g opendray` erneut aus (ohne `--no-optional`).

**`unsupported platform`**
Das npm-Paket deckt nur Linux/macOS auf x64/arm64 ab. Für andere Ziele baue aus
dem Source — siehe [quickstart.md](quickstart.md).

**`config: database.url is empty`**
Weder `OPENDRAY_DATABASE_URL` noch `[database].url` ist gesetzt. Setze eines
davon (Schritt 3).

**`connection refused` bei migrate/serve**
Postgres läuft nicht oder der DSN ist falsch. Bestätige, dass der Server läuft
und Host/Port/Zugangsdaten in deinem DSN korrekt sind.

**pgvector / `extension "vector" is not available`**
Die Extension ist nicht auf dem Server installiert oder wurde nicht in der
opendray-Datenbank aktiviert. Führe Schritt 2 erneut durch (OS-Paket installieren,
dann `CREATE EXTENSION vector` als Superuser).

**Port already in use**
Ändere `OPENDRAY_LISTEN` (oder `listen` in config.toml) auf einen freien Port.

---

## Nächste Schritte

- [README → Produktions-Deployment](../README.de.md#produktions-deployment) — vollständige
  Deploy-Referenz (systemd / launchd / eigener Supervisor, Hardening, Reverse-Proxy)
- [`docs/operator-guide.md`](operator-guide.md) — Ops: Reverse-Proxy-/TLS-
  Topologie, verschlüsselte DB-Backups, Daten-Export/Import
- [`docs/integration-guide.md`](integration-guide.md) — externe Integration gegen
  die REST- + WebSocket-API erstellen
- [`docs/getting-started.md`](getting-started.md) — die geführte All-in-One-
  Einrichtung, wenn du die Teile lieber nicht selbst zusammensetzen möchtest
