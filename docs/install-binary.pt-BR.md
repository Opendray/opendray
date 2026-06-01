# Instalar e rodar a partir de um binário pré-buildado

Para quando você já tem — ou quer — só o binário `opendray`, sem que nenhum
wizard de instalação mexa na sua máquina. Este é o caminho para:

- **`npm install -g opendray` / `npx opendray`** — o pacote npm traz o
  binário oficial do release Go (veja [README → npm / npx](../README.pt-BR.md#npm--npx-node--18)).
- **Downloads de release** — pegue `opendray_*_<os>_<arch>.tar.gz` na
  [página de Releases](https://github.com/Opendray/opendray/releases).
- **Ambientes com scripts / efêmeros** — runners de CI, imagens golden,
  gerenciamento de configuração (Ansible, Nix, Docker), ou qualquer host
  onde você já roda seu próprio Postgres e supervisor de processos.

O binário é o gateway *completo* — a SPA do painel web está embutida, então
não há runtime Node, nenhum servidor de arquivos estáticos separado, e nada
para buildar. O que ele **não** faz é configurar nada por você. Essa é a
troca: você traz um banco PostgreSQL e uma forma de manter o processo rodando,
e em troca nada é instalado, configurado ou registrado por baixo dos panos.

> **Prefere que tudo seja feito por você?** Num Linux / macOS novo, o
> instalador de uma linha provisiona o Postgres, instala as AI CLIs, escreve
> o config e registra um serviço em ~5–10 minutos. Veja
> [README → Instalador de uma linha](../README.pt-BR.md#instalação) ou o
> passo a passo manual em [getting-started.md](getting-started.md).

Este guia te leva de "binário no `PATH`" a "gateway rodando" em cinco
etapas, e depois mostra como mantê-lo rodando como serviço.

---

## Etapa 1 — Obter o binário

### Via npm (qualquer OS com Node ≥ 18)

```sh
npm install -g opendray        # instalação global, coloca `opendray` no PATH
# ou, sem instalar:
npx opendray --help
# ou com outro gerenciador de pacotes:
pnpm add -g opendray
yarn global add opendray
```

O binário correto para a plataforma (`opendray-{linux,darwin}-{x64,arm64}`)
é selecionado automaticamente via `optionalDependencies` — não há hook de
`postinstall` nem chamada de rede no momento da instalação. **Não** passe
`--no-optional`: isso pula o pacote de plataforma e deixa o launcher sem
binário para executar.

### Via arquivo de release

```sh
# Escolha o arquivo correspondente ao seu OS/arch na página de Releases, depois:
tar -xzf opendray_*_linux_amd64.tar.gz
sudo install -m 0755 opendray /usr/local/bin/opendray
```

### Verificar

```sh
opendray version          # exibe versão, commit, data do build
opendray --help           # lista todos os subcomandos
```

Plataformas suportadas: **Linux** (x64, arm64) e **macOS** (x64, arm64).
Windows nativo não tem pacote — use WSL2 e siga o caminho do Linux.

---

## Etapa 2 — Prover o PostgreSQL 15+ com pgvector

O opendray armazena tudo (sessões, memória, audit log) no PostgreSQL, e
seu subsistema de memória precisa da extensão [`pgvector`](https://github.com/pgvector/pgvector).
Versões de servidor suportadas: **15, 16, 17**.

Se você já roda o Postgres, crie um banco e uma role somente CRUD, depois
habilite a extensão uma vez com um superusuário:

```sh
# 1. Instale o pgvector (uma vez por host).
#    Ubuntu/Debian:  sudo apt install postgresql-16-pgvector
#    macOS (brew):   brew install pgvector
#    Outros / fonte: https://github.com/pgvector/pgvector#installation

# 2. Crie o banco + uma role com escopo de projeto.
sudo -u postgres psql <<'SQL'
CREATE DATABASE opendray;
CREATE USER opendray WITH ENCRYPTED PASSWORD 'change-me';
GRANT ALL PRIVILEGES ON DATABASE opendray TO opendray;
SQL

# 3. Habilite o pgvector dentro do banco (uma vez, requer superusuário).
sudo -u postgres psql -d opendray -c 'CREATE EXTENSION IF NOT EXISTS vector;'
sudo -u postgres psql -d opendray -c 'GRANT ALL ON SCHEMA public TO opendray;'
```

Após a extensão existir, a role CRUD do opendray executa as migrations sem
nenhum acesso de superusuário adicional. **Nunca aponte o opendray para uma
role de superusuário em produção** — forneça uma conta com escopo de projeto
e altere a senha fora do fluxo.

---

## Etapa 3 — Configurar

O opendray lê sua configuração de um arquivo TOML **ou** puramente de
variáveis de ambiente (12-factor) — a env sempre vence sobre o arquivo. O
único requisito obrigatório é a URL do banco; todo o resto tem um padrão.

### Opção A — variáveis de ambiente (bom para containers / hosts efêmeros)

```sh
export OPENDRAY_DATABASE_URL="postgres://opendray:change-me@127.0.0.1:5432/opendray?sslmode=disable"
export OPENDRAY_ADMIN_PASSWORD="$(openssl rand -base64 24)"   # login de admin
export OPENDRAY_LISTEN="127.0.0.1:8770"                       # opcional; este é o padrão
```

| Variável | Obrigatório | Padrão | Finalidade |
|---|---|---|---|
| `OPENDRAY_DATABASE_URL` | **sim** | — | DSN do Postgres |
| `OPENDRAY_ADMIN_PASSWORD` | recomendado | — | Senha de admin web/mobile |
| `OPENDRAY_ADMIN_USER` | não | `admin` | Nome de usuário do admin |
| `OPENDRAY_LISTEN` | não | `127.0.0.1:8770` | Endereço de bind |
| `OPENDRAY_LOG_LEVEL` | não | `info` | `debug`/`info`/`warn`/`error` |
| `OPENDRAY_LOG_FORMAT` | não | `text` | `text`/`json` |

Execute `opendray serve` sem a flag `-config` e ele carrega inteiramente
do ambiente.

### Opção B — config.toml

```sh
curl -fsSLO https://raw.githubusercontent.com/Opendray/opendray/main/config.example.toml
mv config.example.toml config.toml
$EDITOR config.toml        # defina [database].url e [admin].password
```

O mínimo a editar:

```toml
listen = "127.0.0.1:8770"

[database]
url = "postgres://opendray:change-me@127.0.0.1:5432/opendray?sslmode=disable"

[admin]
user     = "admin"
password = "use-a-real-password"
```

Veja [`config.example.toml`](../config.example.toml) para o arquivo
completamente anotado (logging, detecção de sessão ociosa, backups, vault,
MCP). Passe com `-config config.toml` nos comandos abaixo. Mantenha segredos
fora do TOML em hosts compartilhados — defina `OPENDRAY_DATABASE_URL` /
`OPENDRAY_ADMIN_PASSWORD` via env e deixe o arquivo sem segredos.

---

## Etapa 4 — Aplicar o schema

```sh
opendray migrate                          # config somente via env
# ou
opendray migrate -config config.toml
```

Idempotente — rodar novamente é um no-op quando o schema já está atualizado.
Isso precisa ter sucesso antes do primeiro `serve`.

---

## Etapa 5 — Rodar

```sh
opendray serve                            # config somente via env
# ou
opendray serve -config config.toml
```

Isso roda em **foreground** (Ctrl-C para). Você deve ter agora:

| URL | O que é |
|---|---|
| `http://127.0.0.1:8770/admin/` | Painel web — faça login com `admin` + sua senha |
| `http://127.0.0.1:8770/api/v1/...` | API REST + WebSocket |

Isso é um gateway completo e rodando. Para qualquer coisa além de um teste
rápido, rode-o sob um supervisor para que sobreviva a reinicializações e
reinicie em caso de crash — veja a seguir.

---

## Rodar como serviço

`opendray serve` é exatamente o que o comando de start de uma unit de serviço
deve chamar. O opendray traz units prontas e endurecidas; os passos abaixo são
os mesmos do [README → Deploy de produção](../README.pt-BR.md#deploy-de-produção),
que é a referência autoritativa (bootstrap completo, notas de sandboxing,
reverse-proxy/TLS).

### Linux — systemd

O repositório traz uma unit endurecida em
[`deploy/systemd/opendray.service`](../deploy/systemd/opendray.service)
(roda `migrate` como `ExecStartPre`, segredos via `EnvironmentFile`,
restart `on-failure`, sandboxing de syscall/filesystem).

```sh
# Binário em /usr/local/bin/opendray, service user, diretório de estado:
sudo useradd -r -s /usr/sbin/nologin -d /var/lib/opendray opendray
sudo install -d -o opendray -g opendray -m 0700 /var/lib/opendray

# Config (sem segredos) + arquivo de segredos (env, modo 0640):
sudo install -D -m 0640 config.toml /etc/opendray/config.toml
sudo install -D -m 0640 -o root -g opendray /dev/null /etc/opendray/env.d/secrets
echo 'OPENDRAY_ADMIN_PASSWORD=use-a-real-password' | sudo tee -a /etc/opendray/env.d/secrets

# Instale + habilite a unit:
sudo cp deploy/systemd/opendray.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now opendray
sudo systemctl status opendray
journalctl -u opendray -f --no-pager
```

Sem systemd? (LXC sem ele, OpenRC, runit, s6, supervisord…) Aponte seu
supervisor para `opendray serve -config /etc/opendray/config.toml` e rode
`opendray migrate` uma vez como etapa de pré-start. Veja
[README → Deploy de produção §B](../README.pt-BR.md#opção-b--binário-direto--seu-próprio-supervisor-de-processos).

### macOS — launchd

O repositório traz um LaunchDaemon em
[`deploy/launchd/com.opendray.opendray.plist`](../deploy/launchd/com.opendray.opendray.plist)
(sobe no boot, reinicia em caso de crash, loga em `/usr/local/var/log/opendray/`).

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

Reinicie: `sudo launchctl kickstart -k system/com.opendray.opendray`.
Desmonte: `sudo launchctl bootout system/com.opendray.opendray`.

> Ambas as units são documentadas por completo — incluindo o layout de
> segredos e por que `MemoryDenyWriteExecute` fica desabilitado — em
> [`deploy/README.md`](../deploy/README.md).

---

## Mantendo atualizado

Como você atualiza depende de como instalou:

- **Instalado via npm** — atualize com seu gerenciador de pacotes. `opendray update`
  substituiria o binário *dentro* de `node_modules` sem o conhecimento do npm
  e seria sobrescrito na próxima instalação, então não use aqui.

  ```sh
  npm install -g opendray@latest
  ```

- **Download de release / instalação via wizard** — o binário se auto-atualiza
  no lugar (baixa o release mais recente, verifica o SHA-256, troca atomicamente
  por si mesmo):

  ```sh
  opendray update --check          # verificação somente de versão, sem aplicar
  sudo opendray update --restart   # aplica e depois reinicia o serviço
  ```

---

## Solução de problemas

**`the matching platform package "opendray-…" was not installed`**
O npm foi rodado com `--no-optional`, ou a instalação foi interrompida. Rode
novamente `npm install -g opendray` (sem `--no-optional`).

**`unsupported platform`**
O pacote npm cobre somente Linux/macOS em x64/arm64. Em outros alvos, faça
o build a partir do código-fonte — veja [quickstart.md](quickstart.md).

**`config: database.url is empty`**
Nem `OPENDRAY_DATABASE_URL` nem `[database].url` está definido. Defina um
(Etapa 3).

**`connection refused` no migrate/serve**
O Postgres não está rodando ou o DSN está errado. Confirme que o servidor
está ativo e que o host/porta/credenciais no seu DSN estão corretos.

**pgvector / `extension "vector" is not available`**
A extensão não está instalada no servidor, ou não foi habilitada no banco do
opendray. Refaça a Etapa 2 (instale o pacote do SO, depois
`CREATE EXTENSION vector` como superusuário).

**Porta já em uso**
Altere `OPENDRAY_LISTEN` (ou `listen` em config.toml) para uma porta livre.

---

## Próximos passos

- [README → Deploy de produção](../README.pt-BR.md#deploy-de-produção) — referência
  completa de deploy (systemd / launchd / supervisor próprio, hardening, reverse proxy)
- [`docs/operator-guide.md`](operator-guide.md) — ops: topologia de reverse-proxy/TLS,
  backups criptografados do DB, exportação/importação de dados
- [`docs/integration-guide.md`](integration-guide.md) — escreva uma integração
  externa contra a API REST + WebSocket
- [`docs/getting-started.md`](getting-started.md) — o setup guiado e tudo-em-um
  se você preferir não montar as peças por conta própria
