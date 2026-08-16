# brainhub

領域ごとの知識を、利用者が普段使っている AI クライアントへ MCP で直接挿せるようにする。

比喩としては「知識の GitHub」。リポジトリの代わりに脳が置かれ、public なら誰でも、private なら招待された者だけが接続できる。

## 掟

実装で迷ったら、他のどのルールよりも優先する。

1. **入口のない機能を作らない。** ディレクトリも同じ。
2. **GBrain の CLI / API でできることを、コードで再実装しない。**
3. **GBrain への呼び出しは `adapter/gbrain` の外に一切出さない。**
4. **決定的な処理はコードで書く。LLM には判断だけさせる。**

## 核

1. 知識の実体は人間が読める Markdown + git である。持ち出せる。
2. 脳は領域ごとに分かれる。ひとつの脳へ全領域を入れない。
3. 脳は組み合わせられる。複数を同時に読ませ、ひとつの窓から引ける（`--federated-read`）。
4. 読む側は Claude / ChatGPT / Codex / Claude Code である。ブラウザではない。

### 非目標

- ChatGPT の劣化版となるチャット機能
- 会話ログの蓄積とその検索
- 全領域を収めた単一の巨大な脳

チャット UI は既に世界に十分ある。不足しているのは中身の側である。

## 責務の分界

知識の保存・検索・権限・MCP 配信は、すべて上流 OSS の [GBrain](https://github.com/garrytan/gbrain) が担う。

| GBrain が持つ | brainhub が作る |
|---|---|
| Markdown 保存と git 自動 commit | 公開ページ（接続前に中身を読む場所） |
| 検索・埋め込み・関係グラフ | 招待画面（`auth register-client` の Web 版） |
| MCP エンドポイントと OAuth 2.1 | 編集画面（`put_page` の Web 版） |
| クライアント発行・スコープ・失効 | 自ドメインのプロキシ |
| 領域ごとの source と横断読み取り | 脳の一覧・作成画面 |
| スキル配信（`mcp.publish_skills`） | **ユーザーという概念** |
| 夜間の自動整理・健康診断 | 誰がどの脳を所有するかの記録 |

GBrain のテーブルに `users` は存在しない。認証の単位は「クライアント」であって「人」ではない。

> **GBrain は鍵を作る。brainhub はその鍵に人の名前を貼る。**

遮断は GBrain が行う。brainhub は誰に何を渡したかを覚えるだけなので、DB は小さい。

## アーキテクチャ

```
Browser ── web（静的配信・/apiプロキシ）
                    │
Codex / Claude ── brainhub（API・MCPプロキシ）
                    │ Docker内部ネットワーク
        ┌───────────┼───────────┐
        │           │           │
     Postgres    gbrain       source作成shim
                    │
                autopilot
                    │ write-through
                    ▼
          brains/<source>/*.md（正本・git）
                    │ 自動 push
                    ▼
                  GitHub
```

- Docker の中身は使い捨て。何度でも作り直せる。
- 正本と DB のデータはホスト側（`~/dev/brainhub-data/`）にあり、コンテナが消えても残る。
- 通常の正本への書き込み経路は GBrain のみ。source作成時だけ、シムが空のgitリポジトリとREADMEを初期化する。brainhub APIは正本をマウントしない。

## テナント分離

脳の分離には二段階がある。

- **source 分離** — ひとつの GBrain 内で領域ごとに source を分ける。git リポジトリも分かれる。組み合わせは `--federated-read` で行う。
- **runtime 分離** — 脳ごとに DB / MCP ランタイム / 認証基盤を分ける。

**当面はすべて source 分離で運用する。** 複数ユーザーも source 分離で成立する（GBrain は全テーブルに RLS が有効）。brainhub 側は `users` / `source_ownership` / `issued_clients` を持ち、誰にどのクライアントを発行したかを記録する。

runtime 分離が必要になるのは、他者のデータを同一 DB に置けないという要求が出たときだけである。分離を強めるほど横断読み取りが難しくなるため、必要のない分離は行わない。

## 技術構成

- **Go** — プロキシ、招待・権限 API
- **TypeScript / React** — 公開ページ、招待画面、編集画面
- **GBrain** — 知識基盤（`v0.45.12.0` にタグ固定）
- **Postgres + pgvector** — GBrain の索引・ベクトル

Go の設計指針は [`docs/architecture-guide.md`](docs/architecture-guide.md)、フロントエンドの配置と実装規約は [`docs/frontend-guide.md`](docs/frontend-guide.md)。どちらも目標形であって、掟 1 に従い不要なディレクトリは作らない。

## セットアップ

### 前提

`.env` に以下を置く（`.gitignore` 済み、`chmod 600`）。

```
OPENAI_API_KEY=
ANTHROPIC_API_KEY=
GBRAIN_POSTGRES_PASSWORD=
BRAINHUB_DATABASE_URL=postgresql://brainhub:<password>@postgres:5432/brainhub
BRAINHUB_GBRAIN_CLIENT_ID=
BRAINHUB_GBRAIN_CLIENT_SECRET=
GBRAIN_ADMIN_BOOTSTRAP_TOKEN=
SHIM_TOKEN=
# 公開先を変える場合は2つを同じoriginに揃える
GBRAIN_PUBLIC_URL=http://localhost:8080
PUBLIC_MCP_URL=http://localhost:8080/mcp
```

データ用のホストディレクトリを作る。

```bash
mkdir -p ~/dev/brainhub-data/{pgdata,gbrain-home,brains}
```

### 起動

```bash
docker compose up -d
```

ブラウザは `http://localhost:3000`、brainhub APIとMCPは `http://localhost:8080` で待ち受ける。ホストへ公開するのはこの2サービスだけで、Postgres、GBrain、source作成シムはDocker内部ネットワークからのみ到達できる。

brainhub APIは起動時に未適用のマイグレーションを適用する。将来brainhubを複数インスタンスにする場合は、migrationを起動から分離して独立ジョブにする。

source作成シムは共有する `SHIM_TOKEN` を要求する。子プロセスには `PATH` / `HOME` / `USER` / `LANG` / `TZ` だけを渡し、実行記録は `/var/lib/gbrain-home/.gbrain/audit/source-shim.jsonl` に追記する。

### 初回のみ

```bash
docker compose run --rm gbrain \
  gbrain init --non-interactive \
    --url "postgresql://gbrain:${GBRAIN_POSTGRES_PASSWORD}@postgres:5432/gbrain" \
    --embedding-model openai:text-embedding-3-small \
    --embedding-dimensions 1536
```

source は git リポジトリである必要がある。

```bash
cd ~/dev/brainhub-data/brains/default && git init && git add -A && git commit -m "initial"

cd ~/dev/brainhub
docker compose exec gbrain gbrain sources add brainhub --path /brain
docker compose exec gbrain gbrain sources default brainhub
docker compose exec gbrain gbrain sources federate brainhub
```

### 取り込み

```bash
docker compose exec gbrain gbrain sync --source brainhub
docker compose exec gbrain gbrain embed --stale
docker compose exec gbrain gbrain extract links --source db
docker compose exec gbrain gbrain extract timeline --source db
docker compose exec gbrain gbrain stats
```

### バックアップ（git remote への自動 push）

```bash
cd ~/dev/brainhub-data/brains/default
git remote add origin <private repo url>

cd ~/dev/brainhub
docker compose exec gbrain gbrain sources harden brainhub --pat-file /var/lib/gbrain-home/pat.txt
```

`harden` は post-commit フック、commit-push ヘルパ、認証情報の配線を入れる。cron はコンテナ内では設定されないため、定期 pull が必要な環境では永続ホスト側で実行する。

### クライアントの接続

```bash
docker compose exec gbrain gbrain auth create "codex"
codex mcp add gbrain --url http://localhost:8080/mcp --bearer-token-env-var GBRAIN_TOKEN
```

`list_skills` が件数を返せば、スキル（知識の引き方）も配信されている。

### brainhub API

brainhub は GBrain と同じ Postgres クラスタ内の専用 role / database を使う。`gbrain` database への接続権限は与えない。

```sql
CREATE ROLE brainhub LOGIN;
\password brainhub
CREATE DATABASE brainhub OWNER brainhub;
REVOKE CONNECT ON DATABASE gbrain FROM PUBLIC;
GRANT CONNECT ON DATABASE gbrain TO gbrain;
REVOKE ALL PRIVILEGES ON DATABASE gbrain FROM brainhub;
```

APIコンテナはGBrainへ `http://gbrain:3131`、シムへ `http://shim:8081`、brainhub databaseへ `postgres:5432` で接続する。これらの内部URLはComposeが設定するため、`.env` でホスト用URLを二重管理しない。

GBrainのOAuth discoveryには `GBRAIN_PUBLIC_URL` がissuerとして載る。公開先を変更するときは、`PUBLIC_MCP_URL` を同じoriginの `/mcp` に揃える。

脳の作成と参照:

```text
POST /api/brains                         ログイン必須
POST /api/brains/{sourceID}/adopt        既存GBrain sourceを所有する（ログイン必須）
GET  /api/brains                         閲覧可能なBrainの配列
GET  /api/brains/{sourceID}              閲覧不可も404
GET  /api/brains/{sourceID}/pages        閲覧不可も404
POST /api/brains/{sourceID}/invitations  ownerのみ。招待tokenは作成時だけ返す
GET  /api/brains/{sourceID}/invitations  ownerのみ。生tokenは返さない
POST /api/invitations/{token}/accept     招待を受諾してMembershipを作る
POST /api/brains/{sourceID}/clients      Claude向けpublic OAuth clientを発行
GET  /api/brains/{sourceID}/clients      自分の発行済みclient一覧
DELETE /api/clients/{id}                 自分のclientを失効
DELETE /api/brains/{sourceID}/members/{userID} ownerのみ。clientも連鎖失効
```

`GET /api/brains` はGBrainのsource形式ではなく、`id` / `source_id` / `name` / `description` / `visibility` / `state` などBrain固有の情報を返す。GBrain側だけにあるsourceは返さない。

本番では `BRAINHUB_ENV=production` を設定し、session cookie に `Secure` を付ける。

Claude Webへ接続するときはMCP URLだけでなく、brainhubで発行したOAuth Client IDをコネクタ編集画面へ設定する。発行するclientはpublic clientなのでsecretはない。

### 開発

初回またはDockerfile・依存関係を変更した後は、対象サービスを再ビルドする。

```bash
docker compose up -d --build brainhub web
```

フロントをホットリロードしたい場合は、APIコンテナを起動したままViteをホストで動かす。Viteの `/api` プロキシは `http://localhost:8080` を使う。

```bash
npm --prefix web ci
npm --prefix web run dev
```

Goの変更を自動検知してコンテナを再ビルド・再起動する場合はCompose Watchを使う。常駐する開発専用サービスや追加のリローダーは使わない。

```bash
docker compose watch brainhub
```

## ハマりどころ

- `GBRAIN_SKILLS_DIR` を設定しないと `list_skills` が空になり、繋いだ AI が道具の使い方を知らないまま動く。値は `/home/bun/.bun/install/global/node_modules/gbrain/skills`。
- `gbrain init` を Dockerfile の `RUN` で実行しない。ビルドキャッシュにより黙ってスキップされ、DB が未初期化のまま進む。
- `sources add --path` の対象は git リポジトリでなければならない。
- volume は名前付きではなくホストパスを使う。`docker compose down -v` で正本が消えないようにするため。
- `sync` が滞ると検索結果が古いまま返る。`autopilot` を必ず起動しておく。
- `BRAINHUB_DATABASE_URL` はDocker内部の `postgres:5432` に固定する。ホスト用URLを別に持たない。

## 進め方

1. プロキシ（Go） — 配布 URL を自ドメインに固定する。中身は素通しでよい
2. 公開ページ（React） — `list_pages` を並べる
3. ユーザーと所有（Go + DB）
4. 招待（Go + React）
5. 編集画面（React）

1 と 2 が繋がった時点で外部に見せられる。

## 参照

- 核と適用範囲: `docs/core.md`
- 実装指針: `docs/architecture-guide.md`
- 上流 GBrain: https://github.com/garrytan/gbrain
- 上流の Docker/CI 手順: `gbrain/docs/operations/headless-install.md`
