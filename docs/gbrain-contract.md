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
| `put_page` | 本文全体の書き込みとwrite-through | ページ作成・編集 |
| `schema_graph` | sourceに適用されたschema packの型一覧 | `GET /api/brains/{id}/page-types` |
| `get_links` | brainhubが作成した状態辺の取得 | 編集画面の `superseded_by` |
| `add_link` / `remove_link` | 状態辺の同期 | ページ保存後の `superseded_by` |

### OAuth 2.1

| | 用途 |
|---|---|
| `/.well-known/oauth-authorization-server` | ディスカバリ |
| `/authorize` `/token` `/revoke` | ブラウザからの接続 |
| `client_credentials` grant | brainhub 自身の読み取り、およびsource固定writerのread/write |
| `authorization_code` + PKCE + `token_endpoint_auth_method=none` | Claude / ChatGPT からの接続 |
| `--public-url` が discovery に反映される | 外部到達 |

### Admin API

| 操作 | 用途 | 使用箇所 |
|---|---|---|
| `POST /admin/login` | bootstrap tokenを24時間のadmin cookieへ交換 | `adapter/gbrain/admin_client.go` |
| `POST /admin/api/register-client` | 利用者向けpublic clientと、脳ごとのconfidential writer client発行 | client発行、Brain作成/adopt、writer再発行 |
| `POST /admin/api/revoke-client` | 利用者clientの失効、writer再発行前の旧client失効 | `DELETE /api/clients/{id}`、`POST /api/brains/{id}/writer/reissue` |

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

### 依存している「挙動」

機能ではなく振る舞いに依存している箇所。**上流が方針を変えたら黙って壊れる。**

1. **MCP 経由の `sources_add` は `path` を拒否する** — シムが存在する理由そのもの
2. **`sources_add` の `url` は `https://` のみ** — `file://` は拒否（実測）
3. **`default` source は削除できない** — 予約語として弾いている
4. **source id は `[a-z0-9-]{1,32}` で不変** — URL に使っている
5. **source は git リポジトリである必要がある** — シムが `git init` する理由
6. **`serve --http` が SSE を返す** — プロキシがバッファリングしない理由
7. **`put_page` は `{slug, content}` で本文全体を受け取る** — `compiled_truth` と `timeline` の分離引数はない
8. **`get_page` は `compiled_truth` / `timeline` / `frontmatter` / `content_hash` を分離して返す** — ただし `put_page` にhash/versionの事前条件はなく、楽観ロックは実装できない
9. **`put_page` のwrite-throughは正本Markdownを書き、git commitする** — brainhubは `written` と `committed` の両方を成功条件にする
10. **`schema_graph` は0ページのsourceでも適用packの型を返す** — 既存ページから型候補を推測しない
11. **状態は明示的な `superseded_by` 辺が正** — `link_source=brainhub-web` の辺、frontmatter、本文の順で解釈し、本文中のStatus宣言は使わない

writer clientは `issued_clients` に入れない。`issued_clients.write_source_id` は利用者へ渡すclientの権限境界であり、User/Membershipと共に失効する。Web編集用writerはbrainhub自身が脳ごとに1本保持し、暗号化secretと復旧状態を `brain_writer_clients` で管理する。

---

## 契約テスト

上の表を、実際に動く GBrain に対して検証するテストを持つ。**アップグレードの可否をこれで判断する。**

```
test/contract/gbrain_contract_test.go
```

要件:

- 実際に動いている GBrain（compose の `gbrain` サービス）に対して実行する
- モックを使わない。**モックで通っても意味がない**
- 通常の `go test ./...` からは除外し、タグかフラグで明示的に走らせる
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
go test -tags=contract ./test/contract/...
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

`shim/main.go` は、**GBrain が意図的に閉じた境界を、brainhub の責任で 1 箇所だけ開けたもの**である。存在理由は契約表の挙動 1 に依存している。

### 現在の位置づけ

- brainhub で唯一、GBrain の CLI を直接実行する場所
- `usecase/output_port/source_provisioner.go` の背後にあり、**実装の差し替えは 1 ファイルで済む**
- パスを外部から受け取らないため、GBrain が防いでいる攻撃（任意パスの登録）は成立しない

### 削除条件

以下のいずれかが成立したら、シムを削除して `adapter/gbrain` の HTTP 実装に差し替える。

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
