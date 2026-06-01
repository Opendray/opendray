# Instalar y ejecutar desde un binario precompilado

Para cuando ya tienes — o quieres — solo el binario `opendray`, sin que ningún
asistente de instalación toque tu máquina. Este es el camino para:

- **`npm install -g opendray` / `npx opendray`** — el paquete npm incluye el
  binario de release oficial de Go (ver [README → npm / npx](../README.es.md#npm--npx-node--18)).
- **Descargas de release** — descarga `opendray_*_<os>_<arch>.tar.gz` desde la
  [página de releases](https://github.com/Opendray/opendray/releases).
- **Entornos con scripts / efímeros** — runners de CI, imágenes doradas, gestión
  de configuración (Ansible, Nix, Docker), o cualquier host donde ya ejecutes tu
  propio Postgres y supervisor de procesos.

El binario es el *gateway completo* — la SPA de administración web está embebida, así que
no hay runtime de Node, no hay servidor de archivos estáticos separado y nada que compilar. Lo que
**no** hace es configurar nada por ti. Ese es el trato: tú aportas una
base de datos PostgreSQL y una forma de mantener el proceso en marcha, y a cambio
nada se instala, configura o registra sin que lo sepas.

> **¿Prefieres que todo lo haga por ti?** En una máquina Linux / macOS recién instalada, el
> instalador de una sola línea provisiona Postgres, instala las AI-CLIs, escribe la
> configuración y registra un servicio en ~5–10 minutos. Consulta
> [README → Instalador de una sola línea](../README.es.md#instalación) o el recorrido manual
> [getting-started.md](getting-started.md).

Esta guía te lleva de "binario en el `PATH`" a "gateway en marcha" en cinco
pasos, y luego muestra cómo mantenerlo en ejecución como servicio.

---

## Paso 1 — Obtener el binario

### Vía npm (cualquier SO con Node ≥ 18)

```sh
npm install -g opendray        # instalación global, pone `opendray` en el PATH
# o, sin instalar:
npx opendray --help
# o con otro gestor de paquetes:
pnpm add -g opendray
yarn global add opendray
```

El binario de plataforma correcto (`opendray-{linux,darwin}-{x64,arm64}`) se selecciona
automáticamente vía `optionalDependencies` — no hay hook `postinstall` ni
llamada de red durante la instalación. **No** uses `--no-optional`: omite el
paquete de plataforma y deja al lanzador sin binario que ejecutar.

### Vía archivo de release

```sh
# Elige el archivo correspondiente a tu SO/arquitectura en la página de releases, luego:
tar -xzf opendray_*_linux_amd64.tar.gz
sudo install -m 0755 opendray /usr/local/bin/opendray
```

### Verificar

```sh
opendray version          # imprime versión, commit, fecha de compilación
opendray --help           # lista todos los subcomandos
```

Plataformas soportadas: **Linux** (x64, arm64) y **macOS** (x64, arm64).
Windows nativo no está empaquetado — usa WSL2 y sigue el camino de Linux.

---

## Paso 2 — Proporcionar PostgreSQL 15+ con pgvector

opendray almacena todo (sesiones, memoria, log de auditoría) en PostgreSQL, y
su subsistema de memoria necesita la extensión [`pgvector`](https://github.com/pgvector/pgvector).
Versiones de servidor soportadas: **15, 16, 17**.

Si ya ejecutas Postgres, crea una base de datos y un rol solo-CRUD, luego
habilita la extensión una vez con un superusuario:

```sh
# 1. Instalar pgvector (una vez por host).
#    Ubuntu/Debian:  sudo apt install postgresql-16-pgvector
#    macOS (brew):   brew install pgvector
#    Otros / fuente: https://github.com/pgvector/pgvector#installation

# 2. Crear la base de datos + un rol con scope al proyecto.
sudo -u postgres psql <<'SQL'
CREATE DATABASE opendray;
CREATE USER opendray WITH ENCRYPTED PASSWORD 'change-me';
GRANT ALL PRIVILEGES ON DATABASE opendray TO opendray;
SQL

# 3. Habilitar pgvector dentro de esa base de datos (una vez, requiere superusuario).
sudo -u postgres psql -d opendray -c 'CREATE EXTENSION IF NOT EXISTS vector;'
sudo -u postgres psql -d opendray -c 'GRANT ALL ON SCHEMA public TO opendray;'
```

Una vez que la extensión existe, el rol solo-CRUD de opendray ejecuta migraciones sin
ningún acceso adicional de superusuario. **Nunca apuntes opendray a un rol superusuario en
runtime** — dale una cuenta con scope al proyecto y rota su contraseña fuera
de banda.

---

## Paso 3 — Configurar

opendray lee su configuración desde un archivo TOML **o** puramente desde variables de
entorno (12-factor) — las variables de entorno siempre tienen prioridad sobre el archivo. El único requisito
estricto es la URL de la base de datos; todo lo demás tiene un valor por defecto.

### Opción A — variables de entorno (recomendado para contenedores / hosts efímeros)

```sh
export OPENDRAY_DATABASE_URL="postgres://opendray:change-me@127.0.0.1:5432/opendray?sslmode=disable"
export OPENDRAY_ADMIN_PASSWORD="$(openssl rand -base64 24)"   # contraseña de acceso al panel de administración
export OPENDRAY_LISTEN="127.0.0.1:8770"                       # opcional; este es el valor por defecto
```

| Variable | Requerida | Por defecto | Propósito |
|---|---|---|---|
| `OPENDRAY_DATABASE_URL` | **sí** | — | DSN de Postgres |
| `OPENDRAY_ADMIN_PASSWORD` | recomendada | — | Contraseña de administración web/móvil |
| `OPENDRAY_ADMIN_USER` | no | `admin` | Nombre de usuario administrador |
| `OPENDRAY_LISTEN` | no | `127.0.0.1:8770` | Dirección de escucha |
| `OPENDRAY_LOG_LEVEL` | no | `info` | `debug`/`info`/`warn`/`error` |
| `OPENDRAY_LOG_FORMAT` | no | `text` | `text`/`json` |

Ejecuta `opendray serve` sin el flag `-config` y cargará todo desde el
entorno.

### Opción B — config.toml

```sh
curl -fsSLO https://raw.githubusercontent.com/Opendray/opendray/main/config.example.toml
mv config.example.toml config.toml
$EDITOR config.toml        # establece [database].url y [admin].password
```

El mínimo a editar:

```toml
listen = "127.0.0.1:8770"

[database]
url = "postgres://opendray:change-me@127.0.0.1:5432/opendray?sslmode=disable"

[admin]
user     = "admin"
password = "use-a-real-password"
```

Consulta [`config.example.toml`](../config.example.toml) para el archivo
completamente anotado (logging, detección de sesión inactiva, backups, vault, MCP). Pásalo con
`-config config.toml` a los comandos de abajo. Mantén los secretos fuera del TOML en
hosts compartidos — establece `OPENDRAY_DATABASE_URL` / `OPENDRAY_ADMIN_PASSWORD` vía entorno
y deja el archivo sin secretos.

---

## Paso 4 — Aplicar el esquema

```sh
opendray migrate                          # configuración solo por entorno
# o
opendray migrate -config config.toml
```

Idempotente — volver a ejecutarlo no tiene efecto una vez que el esquema está actualizado. Esto debe
ejecutarse con éxito antes del primer `serve`.

---

## Paso 5 — Ejecutarlo

```sh
opendray serve                            # configuración solo por entorno
# o
opendray serve -config config.toml
```

Esto se ejecuta en **primer plano** (Ctrl-C lo detiene). Deberías tener ahora:

| URL | Qué es |
|---|---|
| `http://127.0.0.1:8770/admin/` | Panel de administración web — inicia sesión con `admin` + tu contraseña |
| `http://127.0.0.1:8770/api/v1/...` | API REST + WebSocket |

Eso es un gateway completo y en marcha. Para cualquier cosa más allá de una prueba rápida, ejecútalo
bajo un supervisor para que sobreviva a reinicios y se reinicie ante fallos — a continuación.

---

## Ejecutarlo como servicio

`opendray serve` es exactamente el comando de inicio que debería llamar la unidad de un servicio.
opendray incluye unidades endurecidas y listas para usar; los pasos a continuación son los mismos que
[README → Despliegue de producción](../README.es.md#despliegue-de-producción), que es la
referencia autorizada (bootstrap completo, notas de sandboxing, reverse-proxy/TLS).

### Linux — systemd

El repositorio incluye una unidad endurecida en
[`deploy/systemd/opendray.service`](../deploy/systemd/opendray.service)
(ejecuta `migrate` como `ExecStartPre`, secretos vía un `EnvironmentFile`,
reinicio `on-failure`, sandboxing de syscall/filesystem).

```sh
# Binario en /usr/local/bin/opendray, usuario del servicio, directorio de estado:
sudo useradd -r -s /usr/sbin/nologin -d /var/lib/opendray opendray
sudo install -d -o opendray -g opendray -m 0700 /var/lib/opendray

# Configuración (sin secretos) + archivo de secretos (entorno, modo 0640):
sudo install -D -m 0640 config.toml /etc/opendray/config.toml
sudo install -D -m 0640 -o root -g opendray /dev/null /etc/opendray/env.d/secrets
echo 'OPENDRAY_ADMIN_PASSWORD=use-a-real-password' | sudo tee -a /etc/opendray/env.d/secrets

# Instalar + habilitar la unidad:
sudo cp deploy/systemd/opendray.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now opendray
sudo systemctl status opendray
journalctl -u opendray -f --no-pager
```

¿Sin systemd? (LXC sin él, OpenRC, runit, s6, supervisord…) Apunta tu
supervisor a `opendray serve -config /etc/opendray/config.toml` y ejecuta
`opendray migrate` una vez como paso previo al inicio. Consulta
[README → Despliegue de producción §B](../README.es.md#opción-b--binario-directo--tu-propio-supervisor-de-procesos).

### macOS — launchd

El repositorio incluye un LaunchDaemon en
[`deploy/launchd/com.opendray.opendray.plist`](../deploy/launchd/com.opendray.opendray.plist)
(arranca al inicio, se reinicia ante fallos, registra en `/usr/local/var/log/opendray/`).

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

Reiniciar: `sudo launchctl kickstart -k system/com.opendray.opendray`.
Descargar: `sudo launchctl bootout system/com.opendray.opendray`.

> Ambas unidades están documentadas en detalle — incluyendo el layout de secretos y por qué
> `MemoryDenyWriteExecute` está desactivado — en
> [`deploy/README.md`](../deploy/README.md).

---

## Mantenerlo actualizado

La forma de actualizar depende de cómo lo instalaste:

- **Instalado vía npm** — actualiza con tu gestor de paquetes. `opendray update`
  reemplazaría el binario *dentro de* `node_modules` a espaldas de npm y se
  sobreescribiría en la próxima instalación, así que no lo uses aquí.

  ```sh
  npm install -g opendray@latest
  ```

- **Descarga de release / instalación con asistente** — el binario se autoactualiza en su sitio
  (descarga el último release, verifica su SHA-256, se intercambia a sí mismo atómicamente):

  ```sh
  opendray update --check          # sondeo de versión solo informativo
  sudo opendray update --restart   # aplicar y luego reiniciar el servicio
  ```

---

## Solución de problemas

**`the matching platform package "opendray-…" was not installed`**
npm se ejecutó con `--no-optional`, o la instalación se interrumpió. Vuelve a ejecutar
`npm install -g opendray` (sin `--no-optional`).

**`unsupported platform`**
El paquete npm cubre Linux/macOS en x64/arm64 únicamente. En otros objetivos, compila
desde el código fuente — consulta [quickstart.md](quickstart.md).

**`config: database.url is empty`**
Ni `OPENDRAY_DATABASE_URL` ni `[database].url` están configurados. Establece uno (Paso 3).

**`connection refused` en migrate/serve**
Postgres no está en ejecución o el DSN es incorrecto. Confirma que el servidor está activo y que
el host/puerto/credenciales en tu DSN son correctos.

**pgvector / `extension "vector" is not available`**
La extensión no está instalada en el servidor, o no se habilitó en la
base de datos opendray. Repite el Paso 2 (instala el paquete del SO, luego
`CREATE EXTENSION vector` como superusuario).

**Puerto ya en uso**
Cambia `OPENDRAY_LISTEN` (o `listen` en config.toml) a un puerto libre.

---

## Próximos pasos

- [README → Despliegue de producción](../README.es.md#despliegue-de-producción) — referencia completa
  de despliegue (systemd / launchd / supervisor propio, hardening, reverse proxy)
- [`docs/operator-guide.md`](operator-guide.md) — ops: topología de reverse-proxy/TLS,
  backups cifrados de DB, exportación/importación de datos
- [`docs/integration-guide.md`](integration-guide.md) — construye una integración externa
  contra la API REST + WebSocket
- [`docs/getting-started.md`](getting-started.md) — la configuración guiada y todo-en-uno
  si prefieres no ensamblar las piezas tú mismo
