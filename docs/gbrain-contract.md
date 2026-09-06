# GBrain 依存契約とアップグレード手順

brainhub は GBrain を上流 OSS として使う。GBrain は v0.x で動きが速く、**ドキュメントに書かれている機能が自分のバージョンに存在するとは限らない。**

実際に踏んだ例（2026-08-14〜15）:

| 期待 | 実際 |
|---|---|
| `mcp.publish_skills` を設定できる | 0.42.73.2 には存在せず `Config key not found` |
| `/admin` ダッシュボードが開く | 起動バナーには出るが 0.45.12.0 では 404 |
| `sources add --path` は CLI 専用 | MCP にも存在した（ただし `path` は remote から拒否） |
| `sources set-path` がある | 存在しない |

いずれも**推測で設計し、実行して初めて分かった。** この文書はそれを繰り返さないためにある。

---

## 依存している機能の一覧（＝契約）

brainhub が壊れる条件はこの表に尽きる。ここに無いものは使っていない。

### MCP（HTTP）

| 操作 | 用途 | 使用箇所 |
|---|---|---|
| `list_pages` | ページ一覧 | `GET /api/brains/{id}/pages` |
| `sources_list` | source の実在確認 | `POST /api/brains/{id}/adopt` |
| `get_page` | 単体取得 | `GET /api/brains/{id}/pages/{slug...}`、個別ページ閲覧 |
| `put_page` | 本文全体の書き込みとwrite-through | MCP経由のページ作成・更新 |

利用者向け `/mcp` はbrainhubがBearer tokenからUserを特定し、現在見られるsourceへrescopeした利用者別GBrain reader tokenに差し替える。`query` / `list_pages` / `get_page` は明示された `source_id` が範囲内かbrainhubで先に確認し、未指定時は `__all__` を注入する。sourceを指定できない操作も、利用者別readerのgrantを上限として転送する。

#### 利用者MCPのページ書き込み（2026-09-07）

OAuthは `read` / `read write` を受け付け、省略時は後者を要求範囲とする。画面で選んだ許可は要求範囲を超えられない。認可codeとaccess/refresh tokenそれぞれに `write_allowed` を保存する（Brainhub migration 000007）。既存行と旧画面からの許可はfalse。refreshは元の許可を引き継ぐ。

`tools/list` は利用者別readerの一覧を維持し、write許可と現在編集できるreadyの脳がある場合だけ、`source_id` 必須の `put_page` を追加する。呼び出しでは現在のowner/editor権限を照合し、対象脳専用writerに切り替える。ルーティング後は `source_id` を除く。その他の操作にwriterを渡さず、Source作成・削除を有効にしない。OAuthのwrite許可とSourceの編集権限は独立し、どちらかがなければ書けない。

#### `search` の source 境界（2026-08-23）

GBrain v0.46.28.0 の MCP `search` には `source_id` 引数がない。呼び出しごとに source を指定して絞ることはできず、検索対象は OAuth client の `federatedRead` grant で決まる。`gbrain-evals-amara-v1` の調査では、`federatedRead=["gbrain-evals-amara-v1"]` の専用 client を発行して source を限定した。

brainhubの `/mcp` は `search` をHTTP 200のMCP tool errorとして拒否し、`query` を案内する。GBrainの `search` は `source_id` を受け取らずgrant全体を検索するため、別の結果フィルタで代替しない。

### OAuth 2.1

| | 用途 |
|---|---|
| GBrain `client_credentials` grant | brainhub自身の内部読み取り、source固定writer、利用者別reader |
| GBrain `--public-url` discovery | adapterがpathだけ採用し、内部originへ差し替えてbackend tokenを取得 |
| brainhub `/.well-known/*` / `/authorize` / `/token` / `/revoke` | Claude Web向けOAuth 2.1。GBrainへ転送しない |
| brainhub `authorization_code` + PKCE S256 | public client。認可コード5分、access 1時間、refresh 30日ローテーション |
| Claude Web redirect URI | `https://claude.ai/api/mcp/auth_callback` の完全一致だけを許可 |
| `/register` | DCR未対応として拒否 |

### Admin API

| 操作 | 用途 | 使用箇所 |
|---|---|---|
| `POST /admin/login` | bootstrap tokenを24時間のadmin cookieへ交換 | `api/adapter/gbrain/admin_client.go` |
| `POST /admin/api/register-client` | 旧利用者向けpublic client、脳ごとのwriter、利用者別readerを発行 | 旧client発行、Brain作成/adopt、writer/reader再発行 |
| `POST /admin/api/revoke-client` | 旧利用者client、writer、readerの失効 | 旧client削除、writer/reader再発行 |
| `POST /admin/api/rescope-client` | 利用者別readerの `federatedRead` を現在の閲覧範囲へ変更 | `/mcp` 転送前、起動時照合 |

0.45.18.0ではadmin cookieに `Secure` と `Path=/admin` が付く。brainhubはCompose内部のHTTPで通信するため、レスポンスから取得したcookieをadmin APIリクエストへ明示的に付与する。

### CLI（シム経由）

| | 用途 |
|---|---|
| `sources add <id> --path <p> --federated` | 脳の作成 |

### 設定

| | 影響 |
|---|---|
| `search.mode` | 検索精度とコスト |
| `mcp.publish_skills` | 繋いだ AI にスキルが配られるか |
| `GBRAIN_SKILLS_DIR` | 未設定だと `list_skills` が空になる |
| `autopilot.auto_drain.enabled=false` | source単位の除外機能がないv0.46.28.0で、全sourceへのatom自動生成を止める |
| `mcp.strict_params=reject` | 未知引数を無視せず拒否し、scopeを絞ったつもりになる事故を防ぐ |

### Autopilotのマルチテナント運用方針（2026-08-23）

v0.46.28.0のauto-drainはfederationを参照せず、条件を満たす全sourceを`extract-atoms-drain`の対象にする。source単位のallow/deny設定がないため、マルチテナントでユーザーのsourceへ意図しないatomページとLLMコストを発生させないよう、brain全体で無効にする。

```bash
gbrain config set autopilot.auto_drain.enabled false
```

2026-08-23に設定値が`false`であることを確認し、停止中だった`extract-atoms-drain` job #5565をcancelしてからautopilotを再開した。10分間の観察では新しい`extract-atoms-drain` jobは0件で、`gbrain-evals-amara-v1`は467ページのままだった。ただし、この設定が止めるのはauto-drainだけである。通常の`sync`とsource fan-outの`autopilot-cycle`はeval sourceにも投入された。

上流にはsource単位の除外設定を求めるissueを出す予定である。除外設定が実装されたら、必要なsourceだけauto-drainを有効に戻せるか再検討する。

既知の運用状態として、`brainhub-new`、`sakaihayate`、`ui-check-20260815`は未同期のままで、`gbrain doctor`の`sync_freshness`をFAILにする。owner判断まではsyncも削除もしない。

`gbrain-evals-amara-v1`はMIT Licenseの[gbrain-evals](https://github.com/garrytan/gbrain-evals)由来のテストデータであり、本番データではない。ベンチマークと負荷確認のため467ページを残す。このうち27ページのatomは、auto-drainがマルチテナントsourceへ自動生成した挙動の実測データとして意図的に残す。

#### Upstream issue draft（未投稿）

**Title: Allow autopilot auto-drain to exclude specific sources**

```markdown
## Problem

In GBrain v0.46.28.0, autopilot auto-drain considers every eligible,
non-archived source returned by `loadAllSources()`. Source federation is not
part of the selection criteria, and the available auto-drain configuration
only covers `enabled`, `window_seconds`, `threshold`, and
`max_usd_per_day`.

This is problematic when GBrain is hosted as a multi-tenant service, with one
source per customer or team. A tenant may create or import a source without
expecting the host's autopilot to generate additional atom pages or incur LLM
cost on that source.

## Observed behavior

We imported a 424-page MIT-licensed evaluation corpus into a dedicated test
source. Although that source was not intended for automatic enrichment,
autopilot dispatched:

    [dispatch] job #5565 extract-atoms-drain (auto-drain: gbrain-evals-amara-v1; backlog=98)

Before autopilot was stopped, the job generated 27 atom pages in the test
source. This was a real multi-tenant hosting setup, not a single-repository
local brain: Brainhub hosts multiple independently permissioned sources for
different users and teams in one GBrain deployment.

## Current mitigation

Because v0.46.28.0 has no source-level exclusion, we currently disable
auto-drain for the entire brain:

    gbrain config set autopilot.auto_drain.enabled false

This prevents unintended pages and cost, but also disables auto-drain for the
production source where it may be desirable.

## Proposal

Please add a source-level allow/deny mechanism, either in source config or in
the autopilot auto-drain config. For example:

    autopilot.auto_drain.exclude_sources:
      - gbrain-evals-amara-v1

An allow-list would also work. The important property is that a multi-tenant
host can opt individual sources out without disabling auto-drain brain-wide.
```

### 依存している「挙動」

機能ではなく振る舞いに依存している箇所。**上流が方針を変えたら黙って壊れる。**

1. **MCP 経由の `sources_add` は `path` を拒否する** — シムが存在する理由そのもの
2. **`sources_add` の `url` は `https://` のみ** — `file://` は拒否（実測）
3. **`default` source は削除できない** — 予約語として弾いている
4. **source id は `[a-z0-9-]{1,32}` で不変** — URL に使っている
5. **source は git リポジトリである必要がある** — シムが `git init` する理由
6. **`serve --http` が SSE を返す** — 通常の応答はストリーミング転送する。書き込みツールを追加する `tools/list` だけ、現在の1行JSON形式のSSEを読み取って変換する
7. **`put_page` は `{slug, content}` で本文全体を受け取る** — `compiled_truth` と `timeline` の分離引数はない
8. **`get_page` は `compiled_truth` / `timeline` / `frontmatter` / `content_hash` を分離して返す** — ただし `put_page` にhash/versionの事前条件はなく、楽観ロックは実装できない
9. **`put_page` のwrite-throughは正本Markdownを書き、git commitする** — MCPではGBrainの結果をそのままAIクライアントへ返す

Web編集の廃止に伴い、型選択用の `schema_graph` と、`link_source=brainhub-web` の `superseded_by` 辺を読み書きする専用処理への依存は削除した。既存の本文や辺は変更しない。

writer clientは `issued_clients` に入れない。`issued_clients.write_source_id` は利用者へ渡すclientの権限境界であり、User/Membershipと共に失効する。脳専用writerはbrainhub自身が脳ごとに1本保持し、暗号化secretと復旧状態を `brain_writer_clients` で管理する。

### REST の読み取り資格情報（2026-08-22）

`GET /api/brains/{source}/pages` と個別ページは、`brain_writer_clients` のsource別client（`read write`、write sourceとfederated readを同じsourceに固定）で読む。2026-09-07にWeb編集画面とREST作成・更新、page-types APIを削除した。Markdownの作成・更新はMCPの `put_page` に集約する。

読み取り順序は次で固定する。

1. `api/usecase/interactor/page.go` の `readAccess` がBrainの公開状態とmembershipを判定する
2. 判定を通過した場合だけ、`api/adapter/gbrain/page_reader.go` がsource別clientを取得して `list_pages` / `get_page` を呼ぶ
3. `list_pages` と `get_page` には `source_id` を明示する
4. 非メンバーには設定されたpublic typeだけを返し、それ以外は404として扱う

したがって、REST経路の認可の門は `readAccess` である。source別clientが `write` scopeも持つことを理由に、ハンドラーやadapterから直接呼び出してはならない。共通read clientはページ読み取りには使わないが、`POST /api/brains/{id}/adopt` の `sources_list` によるsource実在確認に引き続き必要なため残す。

source別clientのGBrain上の名前は現在 `brainhub-writer-<source>` である。v0.46.28.0の利用中のAdmin APIにはclientのrename操作がなく、Brainhubのadapterもregister/revokeだけを契約としている。既存clientの名称変更にはrevokeとsecret再発行が必要で、認可境界は変わらない一方で全Brainを一時的にdegradedにし得る。このため名称だけの移行は行わず、DB上の名前は維持する。

AIクライアントの `/mcp` は、Brainhub OAuth tokenを毎回hash照合する。読むときはMembershipと `public + ready` から現在のsource一覧を作り、利用者別readerをGBrainでrescopeする。書くときはtokenのwrite許可と対象脳のowner/editor権限を照合して、その脳のsource別clientを使う。rescopeやwriter取得の失敗時は503で閉じる。旧 `issued_clients` のGBrain tokenとBrainhub tokenは `/mcp` で相互に代用できない。

---

## 契約テスト

上の表を、実際に動く GBrain に対して検証するテストを持つ。**アップグレードの可否をこれで判断する。**

```
api/test/contract/gbrain_contract_test.go
```

要件:

- 実際に動いている GBrain（compose の `gbrain` サービス）に対して実行する
- モックを使わない。**モックで通っても意味がない**
- 通常の `go -C api test ./...` からは除外し、タグかフラグで明示的に走らせる
- 各テストは上の表の項目番号を参照するコメントを持つ
- 「拒否されること」も検証する（例: `file://` が 400 を返す、`path` が MCP 経由で拒否される）

拒否側の検証が重要である。**上流が制限を緩めた場合、それは壊れたのではなく、シムが不要になったという合図になる。**

---

## アップグレード手順

### 1. 現状を固定する

```bash
cd ~/dev/brainhub
git add -A && git commit -m "chore: pre-upgrade snapshot"
git tag before-gbrain-<新バージョン>
```

正本の Markdown は git remote へ push 済みであることを確認する。

### 2. 差分を読む

```bash
cd ~/dev/gbrain && git pull
git log --oneline v<現在>..v<新> -- CHANGELOG.md
```

CHANGELOG から、上の契約表に触れる項目だけを抜き出す。**全部読まない。** 見るべきキーワード:

```
sources_add / sources / path / url / scheme
oauth / authorize / token / dcr / public-url / redirect
publish_skills / skills_dir / surface
search.mode / list_pages / put_page
admin / register-client
```

### 3. 上流が提供する道具を先に使う

自前でやる前にこれを叩く。掟 2（GBrain でできることを再実装しない）。

```bash
docker compose exec gbrain gbrain advisor
docker compose exec gbrain gbrain doctor --json
```

`gbrain advisor` はバージョンのずれや保留中のマイグレーションを含む「次にやること」を返す。`skills/gbrain-upgrade` にアップグレード用のスキルもある。

### 4. 上げる

`docker/gbrain/Dockerfile` のタグを変更し、ビルドし直す。

```bash
docker compose build gbrain
docker compose up -d gbrain
docker compose exec gbrain gbrain doctor --json
```

### 5. 契約テストを走らせる

```bash
go -C api test -tags=contract ./test/contract/...
```

**ここが判断点。**

- 全部通る → 上げてよい
- 「拒否されるはず」が通ってしまった → **制限が緩んだ。シムなどの回避策を削れる可能性がある**
- 期待した機能が落ちた → タグを戻す

### 6. 戻し方

```bash
git checkout before-gbrain-<バージョン>
docker compose build gbrain && docker compose up -d gbrain
```

正本の Markdown と Postgres のデータはホスト側の bind mount にあるため、イメージを戻してもデータは失われない。

---

## シムの扱い

`api/shim/main.go` は、**GBrain が意図的に閉じた境界を、brainhub の責任で 1 箇所だけ開けたもの**である。存在理由は契約表の挙動 1 に依存している。

### 現在の位置づけ

- brainhub で唯一、GBrain の CLI を直接実行する場所
- `api/usecase/output_port/source_provisioner.go` の背後にあり、**実装の差し替えは 1 ファイルで済む**
- パスを外部から受け取らないため、GBrain が防いでいる攻撃（任意パスの登録）は成立しない

### 削除条件

以下のいずれかが成立したら、シムを削除して `api/adapter/gbrain` の HTTP 実装に差し替える。

1. `/admin/api/` にローカルパスで source を作成できるエンドポイントが存在する
2. MCP の `sources_add` が、ホスト内の許可されたルート配下に限って `path` を受け付けるようになる
3. `sources_add` の `url` が `file://` またはローカルパスを受け付けるようになる

**確認済み:** 0.45.18.0の `serve-http.ts` にはsourceを作成するadmin APIが存在しない。`/admin/api/register-client` の `source` と `federatedRead` は既存sourceへのOAuth権限を指定するものであり、source作成ではない。

### 削除条件を満たしていない間

シムは維持する。ただし以下を守る。

- 実行できる操作を増やさない。**「ついでに他のコマンドも」は禁止**
- パスを外部から受け取る変更をしない
- 監査 JSONL を止めない
- ホストへポートを公開しない

シムに機能を足したくなったら、それは GBrain 側の API を探すべき合図である。

---

## この文書の更新

アップグレードのたびに、契約表の該当行を更新する。**表と実装がずれた状態を残さない。**

契約表に無い GBrain の機能を使い始めたら、使い始めた時点で表に追加し、契約テストも足す。

- Dockerfileのバージョンを上げる時は3箇所セット:
  1. bun install -g の #vX.Y.Z
  2. RUN test "$(gbrain --version)" = "gbrain X.Y.Z"
  3. compose.yaml の image: brainhub-gbrain:vX.Y.Z
