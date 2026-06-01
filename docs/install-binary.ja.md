# バイナリをインストールして実行する

すでに `opendray` バイナリだけを持っている、または欲しいだけで、
インストールウィザードにマシンを触らせたくないという場合のための手順です。この経路は次のケースを対象としています:

- **`npm install -g opendray` / `npx opendray`** — npm パッケージには公式 Go リリースバイナリが同梱されています（[README → npm / npx](../README.ja.md#npm--npx-node--18) を参照）。
- **リリースアーカイブからのダウンロード** — [リリースページ](https://github.com/Opendray/opendray/releases) から `opendray_*_<os>_<arch>.tar.gz` を取得します。
- **スクリプト化・エフェメラル環境** — CI ランナー、ゴールデンイメージ、構成管理（Ansible、Nix、Docker）、または独自の Postgres とプロセススーパーバイザーをすでに運用しているホスト。

バイナリは *ゲートウェイそのもの* です — web 管理 SPA が埋め込まれているため、Node ランタイムも別の静的サーバーもビルドも不要です。バイナリが **しない** ことは、何かをセットアップすることです。それがトレードオフです: PostgreSQL データベースとプロセスを動かし続ける手段を自分で用意する代わりに、裏で何かがインストール・設定・登録されることはありません。

> **すべて自動でやってほしい場合は？** 新しい Linux / macOS マシンなら、ワンラインインストーラーが Postgres のプロビジョニング、AI CLI のインストール、設定の書き込み、サービスの登録まで約 5〜10 分で完了します。[README → ワンライナーインストーラー](../README.ja.md#インストール) または手動手順の [getting-started.md](getting-started.md) を参照してください。

このガイドでは、「`PATH` にバイナリがある」状態から「ゲートウェイが動いている」状態まで 5 ステップで到達し、その後サービスとして常駐させる方法を説明します。

---

## ステップ 1 — バイナリを入手する

### npm 経由（Node ≥ 18 の任意の OS）

```sh
npm install -g opendray        # グローバルインストール、`opendray` を PATH に追加
# またはインストールせずに:
npx opendray --help
# または別のパッケージマネージャーで:
pnpm add -g opendray
yarn global add opendray
```

適切なプラットフォームバイナリ（`opendray-{linux,darwin}-{x64,arm64}`）は
`optionalDependencies` 経由で自動的に選択されます — `postinstall` フックも、
インストール時のネットワーク呼び出しもありません。`--no-optional` は **渡さないでください**:
プラットフォームパッケージがスキップされ、ランチャーが実行すべきバイナリを見つけられなくなります。

### リリースアーカイブ経由

```sh
# リリースページから OS/アーキテクチャに合ったアーカイブを選択し:
tar -xzf opendray_*_linux_amd64.tar.gz
sudo install -m 0755 opendray /usr/local/bin/opendray
```

### 確認

```sh
opendray version          # バージョン、コミット、ビルド日時を表示
opendray --help           # 全サブコマンドの一覧
```

サポート対象プラットフォーム: **Linux**（x64、arm64）および **macOS**（x64、arm64）。
ネイティブ Windows はパッケージ化されていません — WSL2 を使用して Linux の手順に従ってください。

---

## ステップ 2 — pgvector 付きの PostgreSQL 15+ を用意する

opendray はすべて（セッション、メモリ、audit log）を PostgreSQL に保存し、
そのメモリサブシステムには [`pgvector`](https://github.com/pgvector/pgvector) 拡張が必要です。
サポートされているサーバーバージョン: **15、16、17**。

すでに Postgres を運用している場合は、データベースと CRUD のみの role を作成し、
スーパーユーザーで拡張を一度有効にします:

```sh
# 1. pgvector をインストール（ホストごとに一度）。
#    Ubuntu/Debian:  sudo apt install postgresql-16-pgvector
#    macOS (brew):   brew install pgvector
#    その他 / ソース: https://github.com/pgvector/pgvector#installation

# 2. データベースとプロジェクトスコープの role を作成。
sudo -u postgres psql <<'SQL'
CREATE DATABASE opendray;
CREATE USER opendray WITH ENCRYPTED PASSWORD 'change-me';
GRANT ALL PRIVILEGES ON DATABASE opendray TO opendray;
SQL

# 3. そのデータベース内で pgvector を有効化（一度だけ、スーパーユーザーが必要）。
sudo -u postgres psql -d opendray -c 'CREATE EXTENSION IF NOT EXISTS vector;'
sudo -u postgres psql -d opendray -c 'GRANT ALL ON SCHEMA public TO opendray;'
```

拡張が有効になると、opendray の CRUD のみの role はスーパーユーザーアクセスなしにマイグレーションを実行できます。**実行時に opendray をスーパーユーザー role に向けないでください** — プロジェクトスコープのアカウントを使用し、パスワードはアウトオブバンドでローテーションしてください。

---

## ステップ 3 — 設定する

opendray は TOML ファイル **または** 環境変数のみ（12-factor）から設定を読み込みます
— 環境変数は常にファイルより優先されます。唯一の必須要件はデータベース URL で、
それ以外はすべてデフォルト値があります。

### オプション A — 環境変数（コンテナ / エフェメラルホスト向け）

```sh
export OPENDRAY_DATABASE_URL="postgres://opendray:change-me@127.0.0.1:5432/opendray?sslmode=disable"
export OPENDRAY_ADMIN_PASSWORD="$(openssl rand -base64 24)"   # 管理者ログイン
export OPENDRAY_LISTEN="127.0.0.1:8770"                       # 任意; これがデフォルト
```

| 変数 | 必須 | デフォルト | 用途 |
|---|---|---|---|
| `OPENDRAY_DATABASE_URL` | **必須** | — | Postgres DSN |
| `OPENDRAY_ADMIN_PASSWORD` | 推奨 | — | Web / モバイル管理者パスワード |
| `OPENDRAY_ADMIN_USER` | 任意 | `admin` | 管理者ユーザー名 |
| `OPENDRAY_LISTEN` | 任意 | `127.0.0.1:8770` | バインドアドレス |
| `OPENDRAY_LOG_LEVEL` | 任意 | `info` | `debug`/`info`/`warn`/`error` |
| `OPENDRAY_LOG_FORMAT` | 任意 | `text` | `text`/`json` |

`-config` フラグなしで `opendray serve` を実行すると、環境変数のみから読み込みます。

### オプション B — config.toml

```sh
curl -fsSLO https://raw.githubusercontent.com/Opendray/opendray/main/config.example.toml
mv config.example.toml config.toml
$EDITOR config.toml        # [database].url と [admin].password を設定
```

最低限編集が必要な箇所:

```toml
listen = "127.0.0.1:8770"

[database]
url = "postgres://opendray:change-me@127.0.0.1:5432/opendray?sslmode=disable"

[admin]
user     = "admin"
password = "use-a-real-password"
```

ロギング、セッションアイドル検出、バックアップ、vault、MCP を含む完全アノテーション付きファイルは
[`config.example.toml`](../config.example.toml) を参照してください。以下のコマンドに
`-config config.toml` を渡して使用します。共有ホストでは TOML にシークレットを書かず、
`OPENDRAY_DATABASE_URL` / `OPENDRAY_ADMIN_PASSWORD` を環境変数で設定してファイルを非シークレットのままにしてください。

---

## ステップ 4 — スキーマを適用する

```sh
opendray migrate                          # 環境変数のみの設定
# または
opendray migrate -config config.toml
```

冪等性あり — スキーマが最新であれば再実行しても何も起きません。最初の `serve` の前に必ず成功させてください。

---

## ステップ 5 — 実行する

```sh
opendray serve                            # 環境変数のみの設定
# または
opendray serve -config config.toml
```

これは **フォアグラウンド** で実行されます（Ctrl-C で停止）。以下が利用可能になります:

| URL | 内容 |
|---|---|
| `http://127.0.0.1:8770/admin/` | Web 管理画面 — `admin` とパスワードでログイン |
| `http://127.0.0.1:8770/api/v1/...` | REST + WebSocket API |

これで完全に動作するゲートウェイです。簡単なテスト以上のことをするなら、
再起動時も存続しクラッシュ時に再起動するよう、スーパーバイザー下で実行しましょう — 次へ。

---

## サービスとして実行する

`opendray serve` がまさにサービスユニットの開始コマンドで呼ぶべきものです。
opendray にはハードニング済みのすぐに使えるユニットが同梱されています。以下の手順は
[README → 本番環境へのデプロイ](../README.ja.md#本番環境へのデプロイ) と同じです。
そちらが権威的なリファレンスです（完全なブートストラップ、サンドボックスのメモ、リバースプロキシ/TLS）。

### Linux — systemd

リポジトリにはハードニング済みのユニットが
[`deploy/systemd/opendray.service`](../deploy/systemd/opendray.service) として同梱されています
（`ExecStartPre` で `migrate` を実行、`EnvironmentFile` 経由でシークレットを渡す、
`on-failure` での再起動、syscall / ファイルシステムのサンドボックス化）。

```sh
# /usr/local/bin/opendray にバイナリ、サービスユーザー、状態ディレクトリ:
sudo useradd -r -s /usr/sbin/nologin -d /var/lib/opendray opendray
sudo install -d -o opendray -g opendray -m 0700 /var/lib/opendray

# 設定ファイル（非シークレット）+ シークレットファイル（env、モード 0640）:
sudo install -D -m 0640 config.toml /etc/opendray/config.toml
sudo install -D -m 0640 -o root -g opendray /dev/null /etc/opendray/env.d/secrets
echo 'OPENDRAY_ADMIN_PASSWORD=use-a-real-password' | sudo tee -a /etc/opendray/env.d/secrets

# ユニットをインストールして有効化:
sudo cp deploy/systemd/opendray.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now opendray
sudo systemctl status opendray
journalctl -u opendray -f --no-pager
```

systemd がない場合（systemd のない LXC、OpenRC、runit、s6、supervisord…）は、
スーパーバイザーを `opendray serve -config /etc/opendray/config.toml` に向け、
pre-start ステップとして `opendray migrate` を一度実行してください。
[README → 本番環境へのデプロイ §B](../README.ja.md#option-b--直接バイナリ--任意のプロセススーパーバイザー) を参照してください。

### macOS — launchd

リポジトリには LaunchDaemon が
[`deploy/launchd/com.opendray.opendray.plist`](../deploy/launchd/com.opendray.opendray.plist) として同梱されています
（ブート時に起動、クラッシュ時に再起動、`/usr/local/var/log/opendray/` にログ出力）。

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

再起動: `sudo launchctl kickstart -k system/com.opendray.opendray`。
アンロード: `sudo launchctl bootout system/com.opendray.opendray`。

> 両ユニットは完全にドキュメント化されています — シークレットレイアウトや
> `MemoryDenyWriteExecute` をオフにしている理由を含めて —
> [`deploy/README.md`](../deploy/README.md) を参照してください。

---

## 更新し続ける

更新方法はインストール方法によって異なります:

- **npm 経由でインストール** — パッケージマネージャーで更新します。`opendray update` は
  npm の管理外で `node_modules` 内のバイナリを置き換えてしまい、次回インストール時に上書きされてしまうため、ここでは使用しないでください。

  ```sh
  npm install -g opendray@latest
  ```

- **リリースダウンロード / ウィザードインストール** — バイナリが自己更新します
  （最新リリースをダウンロードし、SHA-256 を検証してアトミックに置き換えます）:

  ```sh
  opendray update --check          # バージョン確認のみ
  sudo opendray update --restart   # 適用後、サービスを再起動
  ```

---

## トラブルシューティング

**`the matching platform package "opendray-…" was not installed`**
npm が `--no-optional` で実行されたか、インストールが中断されました。`npm install -g opendray`
を（`--no-optional` なしで）再実行してください。

**`unsupported platform`**
npm パッケージがカバーするのは Linux / macOS の x64 / arm64 のみです。その他のターゲットでは
ソースからビルドしてください — [quickstart.md](quickstart.md) を参照。

**`config: database.url is empty`**
`OPENDRAY_DATABASE_URL` も `[database].url` も設定されていません。どちらかを設定してください（ステップ 3）。

**`connection refused` (migrate / serve 時)**
Postgres が起動していないか、DSN が間違っています。サーバーが起動していることと、
DSN 内のホスト / ポート / 認証情報が正しいことを確認してください。

**pgvector / `extension "vector" is not available`**
拡張がサーバーにインストールされていないか、opendray データベース内で有効化されていません。
ステップ 2 をやり直してください（OS パッケージをインストールし、スーパーユーザーで
`CREATE EXTENSION vector` を実行）。

**Port already in use（ポートが使用中）**
`OPENDRAY_LISTEN`（または config.toml の `listen`）を空きポートに変更してください。

---

## 次のステップ

- [README → 本番環境へのデプロイ](../README.ja.md#本番環境へのデプロイ) — 完全なデプロイリファレンス（systemd / launchd / 独自スーパーバイザー、ハードニング、リバースプロキシ）
- [`docs/operator-guide.md`](operator-guide.md) — 運用: リバースプロキシ/TLS トポロジー、暗号化 DB バックアップ、データのエクスポート / インポート
- [`docs/integration-guide.md`](integration-guide.md) — REST + WebSocket API を使った外部連携の構築
- [`docs/getting-started.md`](getting-started.md) — 構成要素を自分で組み立てたくない場合のガイド付きオールインワンセットアップ
