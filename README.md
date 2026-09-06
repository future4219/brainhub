# brainhub

領域ごとの知識を、利用者が普段使っている AI クライアントへ MCP で直接挿せるようにする。

比喩としては「知識の GitHub」。リポジトリの代わりに脳が置かれ、public なら誰でも、private なら招待された者だけが接続できる。

## 掟

実装で迷ったら、他のどのルールよりも優先する。

1. **入口のない機能を作らない。** ディレクトリも同じ。
2. **GBrain の CLI / API でできることを、コードで再実装しない。**
3. **GBrain への呼び出しは `api/adapter/gbrain` の外に一切出さない。**
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
| MCPツール、source grant、backend用OAuth client | 利用者向けOAuth 2.1、認可プロキシ、AIからのページ保存 |
| reader/writer発行・source grant・失効 | Membership照合と利用者MCP token |
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

**当面はすべて source 分離で運用する。** 複数ユーザーも source 分離で成立する（GBrain は全テーブルに RLS が有効）。brainhub 側は `users` / `memberships` を権限の正とし、利用者ごとのMCP tokenとGBrain reader clientを管理する。

runtime 分離が必要になるのは、他者のデータを同一 DB に置けないという要求が出たときだけである。分離を強めるほど横断読み取りが難しくなるため、必要のない分離は行わない。

## 技術構成

- **Go** — プロキシ、招待・権限 API
- **TypeScript / React** — 公開ページ、招待画面、接続設定
- **GBrain** — 知識基盤（`v0.45.18.0` にタグ固定）
- **Postgres + pgvector** — GBrain の索引・ベクトル

アプリ本体は `api/`（Go）と `web/`（React）に分ける。

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
# openssl rand -base64 32 で生成し、変更せず保持する
BRAINHUB_WRITER_CREDENTIAL_KEY=
SHIM_TOKEN=
# brainhub API / MCPの公開origin。末尾スラッシュはどちらでもよい
BRAINHUB_PUBLIC_URL=http://localhost:8080
# ブラウザで開く公開origin。脳のアドレス表示とOAuth画面遷移に使う
BRAINHUB_PUBLIC_WEB_URL=http://localhost:3000
# GBrainがadvertiseする公開origin
GBRAIN_PUBLIC_URL=http://localhost:8080
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

サイドバーの「AIとの接続」（`/settings/connections`）で、利用者共通の接続を設定する。以前の脳ごとの接続URLはここへ転送する。

Codexの標準手順はブラウザ認証:

1. 「Codexの接続を準備」を押す。利用者専用のpublic OAuth clientを発行し、再操作時は同じclientを使う。
2. 表示された追加コマンドをCodexを使う端末で実行する。

   ```sh
   codex mcp add brainhub --url '<画面のMCP URL>' --oauth-client-id '<画面のClient ID>'
   ```

3. 開いたブラウザでBrainhubへログインし、脳の一覧と接続権限を確認して許可する。読み取り・書き込みを標準とし、読み取りのみに変更できる。端末に接続完了が表示されたらCodexで新しい会話を開く。

既存の接続で書き込みを使うには `codex mcp login brainhub --scopes read,write` で再認可し、新しい会話を開く。既存のトークンは読み取り専用のままで、更新時にも権限は拡張しない。Codexがaccess/refresh tokenを管理するので、通常の接続でトークンをコピーしたり環境変数を設定したりする必要はない。refresh token自体が期限切れになった場合は再ログインする。

Webはページの閲覧に対応し、ページの作成・編集画面とREST保存APIは持たない。

認証・トークン・読み書きの許可の仕組みは[鍵とアクセス許可の入門](docs/mcp-access.md)を参照。

内部のGBrain接続は `api/adapter/gbrain` にまとめる。通常のMCP usecaseは利用者と権限だけを判断し、handlerは認可済み要求を渡す。GBrain用の鍵取得・引数変換・ヘッダー差し替えを上位へ持ち出さない。残す互換処理の理由は[依存契約](docs/gbrain-contract.md)を参照。

MCPの書き込みは `put_page` によるページの作成・更新に対応する。`source_id` を必須とし、呼び出すたびに接続の書き込み許可と対象脳の現在のowner/editor権限、ready状態を確認する。その脳に固定した既存writer clientでGBrainを呼び、GBrainが受け付けない `source_id` は転送前に取り除く。読み取りは従来の利用者別readerで行う。脳の作成・削除やその他の管理操作は公開しない。手動CLIトークンは引き続き読み取り専用。

`put_page` はページ全体を置き換える。編集前に同じ `source_id` と `include_content: true` で `get_page` し、既存内容を保持する。GBrain v0.46.28.0のAPI契約に合わせた引数のみ公開し、新しい書き込みツールを自動的に開放しない。

Codex clientは `http://127.0.0.1/callback` と、MCP URLのSHA-256先頭9バイトをbase64url化したIDを付けた `/callback/<ID>`（Codex 0.149系）の2種類を事前登録する。認可時はRFC 8252に従いloopback portのみ可変とし、token交換時は認可codeに保存されたredirect URIとの完全一致も要求する。Brainhubはissuerを認可応答の `iss` で返す。Codexの事前登録clientとissuer-bound callbackの仕様: https://developers.openai.com/codex/mcp/

- Claude Webは同じ画面の「Claude Webに接続する」からOAuth Client IDを発行してコネクタへ設定する。public clientのためsecretはない。
- ブラウザ認証を使えない環境には「手動トークンで接続する」を残す。生tokenは発行時だけ表示され、DBにはSHA-256 hashだけを保存する。有効期限は90日でrefreshはなく、期限切れ後は `401 token_expired` になるため再発行する。
- 手動tokenをCodexで使う場合は `BRAINHUB_MCP_TOKEN` に保存し、`mcp_servers.brainhub` の `url` と `bearer_token_env_var = "BRAINHUB_MCP_TOKEN"` を設定する。

接続は脳ごとではなく利用者ごとに1本である。各MCPリクエストで現在のMembershipと `public + ready` を読み直すため、所属や脳が増減しても接続し直さない。`list_skills` が件数を返せば、スキル（知識の引き方）も配信されている。

GBrainの未知パラメータ無視による権限漏れを防ぐため、DB planeの設定は次で固定する。

```bash
docker compose exec gbrain gbrain config set mcp.strict_params reject
```

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

GBrainのOAuth discoveryはbackend用clientが内部通信で使う。利用者向けのdiscovery、`/authorize`、`/token`、`/revoke` はbrainhubが提供する。公開先を変更するときは `BRAINHUB_PUBLIC_URL` に公開originを設定する。`/api/config` のMCP URLとOAuth discoveryのissuer・各endpointはこの値から導出される。

脳の作成と参照:

```text
POST /api/brains                         ログイン必須
POST /api/brains/{sourceID}/adopt        既存GBrain sourceを所有する（ログイン必須）
GET  /api/brains                         閲覧可能なBrainの配列
GET  /api/brains/{sourceID}              閲覧不可も404
GET  /api/brains/{sourceID}/pages        閲覧不可も404
GET  /api/brains/{sourceID}/pages/{slug} 本文・Timelineを含むページ1件
POST /api/brains/{sourceID}/writer/reissue ownerのみ。脳専用clientを再発行
POST /api/brains/{sourceID}/invitations  ownerのみ。招待tokenは作成時だけ返す
GET  /api/brains/{sourceID}/invitations  ownerのみ。生tokenは返さない
POST /api/invitations/{token}/accept     招待を受諾してMembershipを作る
POST /api/brains/{sourceID}/clients      旧GBrain client発行API（新接続では不使用）
GET  /api/brains/{sourceID}/clients      旧issued client一覧
DELETE /api/clients/{id}                 旧issued clientを失効
DELETE /api/brains/{sourceID}/members/{userID} ownerのみ。clientも連鎖失効
GET  /api/mcp/connection                 利用者共通の接続情報と閲覧可能な脳
POST /api/mcp/client                     Claude Web用Client IDを1人1件発行（互換用）
POST /api/mcp/clients/{name}             codex / claude-webの利用者別Client IDを発行
POST /api/mcp/tokens                     CLI用Bearer tokenを発行。生tokenはこの応答だけ
DELETE /api/mcp/tokens/{id}              自分のCLI用Bearer tokenを失効
POST /api/mcp/reader/reissue             orphanになったGBrain readerを手動再発行
GET  /authorize                          brainhub OAuth認可開始
POST /token                              認可コード交換・refresh tokenローテーション
POST /revoke                             brainhub MCP tokenを失効
```

`GET /api/brains` はGBrainのsource形式ではなく、`id` / `source_id` / `name` / `description` / `visibility` / `state` などBrain固有の情報を返す。GBrain側だけにあるsourceは返さない。

本番では `BRAINHUB_ENV=production` を設定し、session cookie に `Secure` を付ける。

Claude Webへ接続するときはMCP URLだけでなく、brainhubで発行したOAuth Client IDをコネクタ編集画面へ設定する。発行するclientはpublic clientなのでsecretはない。旧 `issued_clients` の行とGBrain clientは移行履歴として残すが、brainhubの `/mcp` はそのtokenを受け付けず、GBrain OAuthへフォールバックしない。

脳の閲覧とMCP書き込み用には、脳の作成・adopt時にsourceへ固定したconfidential writer clientを1本だけ発行する。secretは `BRAINHUB_WRITER_CREDENTIAL_KEY` によるAES-GCM暗号文として `brain_writer_clients` に保存し、平文をDBへ置かない。利用者へ配るpublic clientを記録する `issued_clients` は流用しない。後者はUser/Membershipに従って失効する鍵でsecretを保持しない一方、writerはbrainhub自身が脳ごとに恒久保持する別ライフサイクルだからである。

### 開発

初回またはDockerfile・依存関係を変更した後は、対象サービスを再ビルドする。

```bash
docker compose up -d --build brainhub web
```

WebコンテナはVite開発サーバーで動き、`web/src` の変更をホットリロードする。依存関係やDockerfileを変更した場合だけ再ビルドする。

Goの変更を自動検知してコンテナを再ビルド・再起動する場合はCompose Watchを使い、Go用の常駐リローダーは追加しない。

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
- `BRAINHUB_WRITER_CREDENTIAL_KEY` を失うと、既存の全writer client secretを復号できない。新しい鍵を `openssl rand -base64 32` で `.env` に設定してbrainhubを再起動し、各脳の「設定」タブから「脳専用の接続を再発行」を実行する（またはowner sessionで `POST /api/brains/{sourceID}/writer/reissue`）。GBrain側の旧clientを失効して新規発行し、`brain_writer_clients` の記録を作り直す。脳、正本Markdown、git履歴、GBrainの知識索引は失われない。鍵のローテーション自体は未実装なので、平常時に値を変更しない。

## 進め方

1. プロキシ（Go） — 配布 URL を自ドメインに固定する。中身は素通しでよい
2. 公開ページ（React） — `list_pages` を並べる
3. ユーザーと所有（Go + DB）
4. 招待（Go + React）
5. AIからのページ作成・更新（MCP）

1 と 2 が繋がった時点で外部に見せられる。

## 参照

- 核と適用範囲: `docs/core.md`
- 実装指針: `docs/architecture-guide.md`
- 上流 GBrain: https://github.com/garrytan/gbrain
- 上流の Docker/CI 手順: `gbrain/docs/operations/headless-install.md`
