# Установка и запуск из готового бинарника

Для тех, у кого уже есть — или кто хочет получить — просто бинарник `opendray`,
без мастера установки, который трогает вашу машину. Этот путь подходит для:

- **`npm install -g opendray` / `npx opendray`** — npm-пакет поставляется с
  официальным Go release-бинарником (см. [README → npm / npx](../README.ru.md#npm--npx-node--18)).
- **Загрузки со страницы релизов** — скачайте `opendray_*_<os>_<arch>.tar.gz` со
  [страницы релизов](https://github.com/Opendray/opendray/releases).
- **Скриптовые / эфемерные окружения** — CI runner-ы, golden image-ы, управление
  конфигурацией (Ansible, Nix, Docker), или любой хост, где у вас уже работает
  собственный Postgres и супервизор процессов.

Бинарник — это *весь* шлюз целиком: React-админка встроена в него, поэтому
Node-рантайм, отдельный сервер статики и сборка не нужны. Чего он **не** делает —
так это ничего за вас не настраивает. Таков компромисс: вы приносите базу данных
PostgreSQL и способ держать процесс запущенным, а взамен ничего не устанавливается,
не конфигурируется и не регистрируется за вашей спиной.

> **Хотите, чтобы всё сделали за вас?** На свежей Linux / macOS-машине
> однострочный установщик поднимает Postgres, устанавливает AI-CLI, прописывает
> конфиг и регистрирует сервис примерно за 5–10 минут. См.
> [README → Однострочный установщик](../README.ru.md#установка) или подробный
> гайд [getting-started.md](getting-started.md).

Этот гайд проведёт вас от «бинарник есть в `PATH`» до «шлюз работает» за пять
шагов, а затем покажет, как запустить его как сервис.

---

## Шаг 1 — Получите бинарник

### Через npm (любая ОС с Node ≥ 18)

```sh
npm install -g opendray        # глобальная установка, добавляет `opendray` в PATH
# или без установки:
npx opendray --help
# или через другой пакетный менеджер:
pnpm add -g opendray
yarn global add opendray
```

Нужный платформенный бинарник (`opendray-{linux,darwin}-{x64,arm64}`) выбирается
автоматически через `optionalDependencies` — никакого `postinstall`-хука и никаких
сетевых вызовов на момент установки. **Не передавайте** `--no-optional`:
это пропустит платформенный пакет и оставит launcher без бинарника для запуска.

### Через архив со страницы релизов

```sh
# Выберите архив для вашей ОС/архитектуры на странице релизов, затем:
tar -xzf opendray_*_linux_amd64.tar.gz
sudo install -m 0755 opendray /usr/local/bin/opendray
```

### Проверка

```sh
opendray version          # выводит версию, коммит, дату сборки
opendray --help           # перечисляет все подкоманды
```

Поддерживаемые платформы: **Linux** (x64, arm64) и **macOS** (x64, arm64).
Нативный Windows не упакован — используйте WSL2 и следуйте пути для Linux.

---

## Шаг 2 — Обеспечьте PostgreSQL 15+ с pgvector

opendray хранит всё (сессии, память, аудит-лог) в PostgreSQL, и его
подсистема памяти требует расширения [`pgvector`](https://github.com/pgvector/pgvector).
Поддерживаемые версии сервера: **15, 16, 17**.

Если у вас уже работает Postgres, создайте базу данных и роль только для CRUD,
затем однократно включите расширение от суперпользователя:

```sh
# 1. Установите pgvector (один раз на хост).
#    Ubuntu/Debian:  sudo apt install postgresql-16-pgvector
#    macOS (brew):   brew install pgvector
#    Другое / из исходников: https://github.com/pgvector/pgvector#installation

# 2. Создайте базу данных + роль, ограниченную проектом.
sudo -u postgres psql <<'SQL'
CREATE DATABASE opendray;
CREATE USER opendray WITH ENCRYPTED PASSWORD 'change-me';
GRANT ALL PRIVILEGES ON DATABASE opendray TO opendray;
SQL

# 3. Включите pgvector внутри этой базы (однократно, нужен суперпользователь).
sudo -u postgres psql -d opendray -c 'CREATE EXTENSION IF NOT EXISTS vector;'
sudo -u postgres psql -d opendray -c 'GRANT ALL ON SCHEMA public TO opendray;'
```

После того как расширение установлено, роль с правами только на CRUD выполняет
миграции без каких-либо дополнительных прав суперпользователя. **Никогда не указывайте
в opendray роль суперпользователя для рантайма** — дайте ему проектную учётку и
меняйте пароль независимо.

---

## Шаг 3 — Настройте конфигурацию

opendray читает конфиг из TOML-файла **или** только из переменных окружения
(12-factor) — env всегда имеет приоритет над файлом. Единственное жёсткое
требование — URL базы данных; всё остальное имеет значение по умолчанию.

### Вариант A — переменные окружения (хорошо для контейнеров / эфемерных хостов)

```sh
export OPENDRAY_DATABASE_URL="postgres://opendray:change-me@127.0.0.1:5432/opendray?sslmode=disable"
export OPENDRAY_ADMIN_PASSWORD="$(openssl rand -base64 24)"   # пароль для входа в админку
export OPENDRAY_LISTEN="127.0.0.1:8770"                       # опционально; это значение по умолчанию
```

| Переменная | Обязательна | По умолчанию | Назначение |
|---|---|---|---|
| `OPENDRAY_DATABASE_URL` | **да** | — | DSN для Postgres |
| `OPENDRAY_ADMIN_PASSWORD` | рекомендуется | — | Пароль для веб/мобильной админки |
| `OPENDRAY_ADMIN_USER` | нет | `admin` | Имя пользователя администратора |
| `OPENDRAY_LISTEN` | нет | `127.0.0.1:8770` | Адрес привязки |
| `OPENDRAY_LOG_LEVEL` | нет | `info` | `debug`/`info`/`warn`/`error` |
| `OPENDRAY_LOG_FORMAT` | нет | `text` | `text`/`json` |

Запустите `opendray serve` без флага `-config`, и он полностью загрузится из
переменных окружения.

### Вариант B — config.toml

```sh
curl -fsSLO https://raw.githubusercontent.com/Opendray/opendray/main/config.example.toml
mv config.example.toml config.toml
$EDITOR config.toml        # задайте [database].url и [admin].password
```

Минимум для редактирования:

```toml
listen = "127.0.0.1:8770"

[database]
url = "postgres://opendray:change-me@127.0.0.1:5432/opendray?sslmode=disable"

[admin]
user     = "admin"
password = "use-a-real-password"
```

См. [`config.example.toml`](../config.example.toml) — полностью аннотированный
файл (логирование, обнаружение простоя сессий, бэкапы, vault, MCP). Передайте его
с `-config config.toml` командам ниже. Держите секреты вне TOML на общих хостах —
задайте `OPENDRAY_DATABASE_URL` / `OPENDRAY_ADMIN_PASSWORD` через env и оставьте
файл без секретов.

---

## Шаг 4 — Примените схему

```sh
opendray migrate                          # конфиг только из env
# или
opendray migrate -config config.toml
```

Идемпотентно — повторный запуск не даёт эффекта, если схема актуальна. Это должно
завершиться успешно до первого запуска `serve`.

---

## Шаг 5 — Запустите

```sh
opendray serve                            # конфиг только из env
# или
opendray serve -config config.toml
```

Это запускается на **переднем плане** (Ctrl-C останавливает). Теперь у вас есть:

| URL | Что |
|---|---|
| `http://127.0.0.1:8770/admin/` | Веб-админка — войдите с `admin` + ваш пароль |
| `http://127.0.0.1:8770/api/v1/...` | REST + WebSocket API |

Это полноценный работающий шлюз. Для всего, что выходит за рамки быстрой проверки,
запускайте его под супервизором, чтобы он выдерживал перезагрузки и рестартовал
при краше — далее.

---

## Запуск как сервис

`opendray serve` — именно та команда, которую должен вызывать start-команда юнита
сервиса. opendray поставляется с захардёненными, готовыми к использованию юнитами;
шаги ниже аналогичны
[README → Развёртывание в продакшене](../README.ru.md#развёртывание-в-продакшене),
который является авторитетным справочником (полный bootstrap, заметки по sandboxing,
reverse-proxy/TLS).

### Linux — systemd

Репозиторий поставляется с захардёненным юнитом в
[`deploy/systemd/opendray.service`](../deploy/systemd/opendray.service)
(запускает `migrate` как `ExecStartPre`, секреты через `EnvironmentFile`,
рестарт `on-failure`, sandboxing syscall/filesystem).

```sh
# Бинарник в /usr/local/bin/opendray, сервисный пользователь, директория состояния:
sudo useradd -r -s /usr/sbin/nologin -d /var/lib/opendray opendray
sudo install -d -o opendray -g opendray -m 0700 /var/lib/opendray

# Файл конфига (без секретов) + файл секретов (env, режим 0640):
sudo install -D -m 0640 config.toml /etc/opendray/config.toml
sudo install -D -m 0640 -o root -g opendray /dev/null /etc/opendray/env.d/secrets
echo 'OPENDRAY_ADMIN_PASSWORD=use-a-real-password' | sudo tee -a /etc/opendray/env.d/secrets

# Установка + включение юнита:
sudo cp deploy/systemd/opendray.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now opendray
sudo systemctl status opendray
journalctl -u opendray -f --no-pager
```

Нет systemd? (LXC без него, OpenRC, runit, s6, supervisord…) Направьте ваш
супервизор на `opendray serve -config /etc/opendray/config.toml` и запустите
`opendray migrate` один раз как шаг pre-start. См.
[README → Развёртывание в продакшене §B](../README.ru.md#вариант-b--прямой-бинарник--ваш-собственный-супервизор-процессов).

### macOS — launchd

Репозиторий поставляется с LaunchDaemon в
[`deploy/launchd/com.opendray.opendray.plist`](../deploy/launchd/com.opendray.opendray.plist)
(запускается при загрузке, рестартует при краше, логи в `/usr/local/var/log/opendray/`).

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

Рестарт: `sudo launchctl kickstart -k system/com.opendray.opendray`.
Выгрузка: `sudo launchctl bootout system/com.opendray.opendray`.

> Оба юнита полностью задокументированы — включая раскладку секретов и почему
> `MemoryDenyWriteExecute` оставлен выключенным — в
> [`deploy/README.md`](../deploy/README.md).

---

## Обновление

Способ обновления зависит от того, как вы установили:

- **Установлено через npm** — обновляйте через ваш пакетный менеджер. `opendray update`
  заменил бы бинарник *внутри* `node_modules` за спиной npm и был бы затёрт
  при следующей установке, поэтому здесь его не используйте.

  ```sh
  npm install -g opendray@latest
  ```

- **Загрузка со страницы релизов / установка через мастер** — бинарник самообновляется
  на месте (скачивает последний релиз, верифицирует SHA-256, атомарно заменяет себя):

  ```sh
  opendray update --check          # только проверка версии, без применения
  sudo opendray update --restart   # применить, затем перезапустить сервис
  ```

---

## Устранение неполадок

**`the matching platform package "opendray-…" was not installed`**
npm был запущен с `--no-optional`, или установка была прервана. Повторно запустите
`npm install -g opendray` (без `--no-optional`).

**`unsupported platform`**
npm-пакет покрывает только Linux/macOS на x64/arm64. На других платформах — соберите
из исходников: см. [quickstart.md](quickstart.md).

**`config: database.url is empty`**
Не задан ни `OPENDRAY_DATABASE_URL`, ни `[database].url`. Задайте одно из двух (Шаг 3).

**`connection refused` при migrate/serve**
Postgres не запущен или DSN неверный. Убедитесь, что сервер работает и
хост/порт/учётные данные в вашем DSN верны.

**pgvector / `extension "vector" is not available`**
Расширение не установлено на сервере или не включено в базе opendray.
Повторите Шаг 2 (установите OS-пакет, затем выполните `CREATE EXTENSION vector`
от суперпользователя).

**Port already in use**
Измените `OPENDRAY_LISTEN` (или `listen` в config.toml) на свободный порт.

---

## Дальнейшие шаги

- [README → Развёртывание в продакшене](../README.ru.md#развёртывание-в-продакшене) — полный
  справочник по деплою (systemd / launchd / собственный супервизор, hardening, reverse-proxy)
- [`docs/operator-guide.md`](operator-guide.md) — эксплуатация: топология reverse-proxy/TLS,
  шифрованные бэкапы БД, экспорт/импорт данных
- [`docs/integration-guide.md`](integration-guide.md) — написание внешней интеграции
  через REST + WebSocket API
- [`docs/getting-started.md`](getting-started.md) — управляемый сквозной гайд, если вы
  предпочитаете не собирать всё по кускам самостоятельно
