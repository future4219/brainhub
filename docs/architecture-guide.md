# AI Development Architecture Guide

このドキュメントは、Xroll backend と同じ考え方で他プロジェクトを開発するためのアーキテクチャ指針です。
AI に実装を任せるときは、このファイルを `AGENTS.md`、`CLAUDE.md`、またはプロジェクトの開発ルールとして渡してください。

目的は、AI が迷ったときに次を判断できるようにすることです。

- どのディレクトリにファイルを置くべきか
- どの層にどの責務を書くべきか
- どの依存は許可され、どの依存は禁止されるか
- API、DB、外部サービスの変更をどこで吸収するか
- 新機能を追加するときの実装順序

## brainhub固有の制約

このガイドは目標形であって、初期構成ではない。

1. 入口のない機能を作らない。ディレクトリも同じ
2. GBrainのCLI/APIでできることを、コードで再実装しない
3. GBrainへの呼び出しは adapter/gbrain の外に一切出さない

## Architecture Style

このプロジェクトは Clean Architecture を採用する。

中心に `domain` を置き、その外側に `usecase`、さらに外側に `adapter` と `api` を置く。
外側の層は内側の層に依存してよいが、内側の層は外側の層に依存してはいけない。

依存方向は原則として次の通り。

```text
api / adapter / main
  -> usecase
    -> domain
```

DB、HTTP framework、外部 API、認証、メール、ファイルストレージなどの技術詳細は外側の層に閉じ込める。
業務ルールは `domain` と `usecase` に置く。

## Standard Directory Structure

Go backend の標準構成は次の形を基本とする。

```text
api/
  main.go

  api/
    router/
    handler/
    middleware/
    schema/
    csv/

  domain/
    entity/
    constructor/
    validation/
    entconst/

  usecase/
    input_port/
    interactor/
    output_port/
    input_port_mock/
    output_port_mock/

  adapter/
    database/
      model/
      repository/
      initdata/
    authentication/
    email/
    file/
    cache/
    clock/
    ulid/
    aws/
    external_service_name/

  config/
  log/
  utils/
  mysql/
    migrations/
    init_data/
  integration_test/
  cmd/
```

プロジェクトによって不要なディレクトリは作らなくてよい。
ただし、責務の境界はこの構成に合わせる。

## Directory Coding Guide

この章は、AI が各ディレクトリでどのようにコードを書くべきかを示す。
Xroll backend の実ファイルを例にしながら、配置、責務、書き方を固定する。

### api/main.go

役割:

- アプリケーションの起動点
- DB、logger、外部 service、repository、usecase、router の組み立て
- 環境に応じた driver の切り替え
- server timeout など runtime 設定

書き方:

- `main.go` は composition root として扱う
- repository や adapter はここで生成して usecase に渡す
- usecase は `interactor.NewXxxUseCase(...)` で生成する
- router には input port interface を満たす usecase を渡す
- `main.go` に業務ロジックを書かない
- handler や usecase 内で DB 接続や AWS client を new しない

Xroll の例:

- `api/main.go` では `database.NewMySQLDB` で DB を作り、`repository.NewVideoRepository` を作り、`interactor.NewVideoUseCase` に渡している
- `VideoUseCase` はさらに `NewMetricsVideoUseCase` decorator で包まれている
- `router.NewServer(...)` に完成した usecase 群を渡している

AI への指示:

- 新しい feature を追加したら、最後に `main.go` で依存を wire する
- constructor の引数が増えた場合も、原則として `main.go` の組み立てを更新する
- 環境変数や credential の値を直接書かない

### api/api/router

役割:

- HTTP endpoint の定義
- middleware の適用
- handler の生成
- public、auth required、admin required などの route grouping

書き方:

- route は feature 単位でまとめる
- 認証必須 route は `auth.Group(...)` のように group で分ける
- 任意認証、cookie/header 認証、管理者権限などを route 上で明示する
- rate limit が必要な endpoint は route 定義時に付ける
- path 変更は frontend 影響が大きいため、明示指示なしに変えない

Xroll の例:

- `api/api/router/router.go` は `auth`, `authIfPossible`, `authCookieOrHeader`, `notAuth` を分けている
- `/videos` は public route と authenticated route が混在するため、`notAuth.Group("/videos")` と `auth.POST("/videos/...")` を使い分けている
- `/videos/feeds/...` や `/videos/:videoId/play-events` のように、同じ feature 内で読み取りと記録系 endpoint を近くに置いている

AI への指示:

- endpoint を追加するときは、既存 feature の route group に追加する
- handler の中で middleware 相当の認証分岐を増やす前に、router 側で表現できないか確認する
- endpoint path、method、request/response の互換性を守る

### api/api/handler

役割:

- HTTP request を受け取る
- path/query/body を schema に bind する
- 認証 context から user を取り出す
- usecase を呼ぶ
- usecase error を HTTP status code に変換する
- response schema に変換して返す

書き方:

- handler struct は usecase input port interface を field に持つ
- constructor は `NewXxxHandler(...)` にする
- request parsing は `echo.QueryParamsBinder`、`echo.PathParamsBinder`、`c.Bind` などに寄せる
- response は `schema.XxxResFromEntity(...)` のような変換関数を使う
- DB、GORM、repository、external adapter を直接呼ばない
- 長い業務判断は interactor に移す

Xroll の例:

- `api/api/handler/video.go` の `VideoHandler` は `input_port.IVideoUseCase` と `input_port.IUserUseCase` を持つ
- `Search` は query を `schema.VideoSearchQueryReq` に bind し、`h.VideoUC.Search(...)` を呼ぶ
- `FindByID` は path param を取り、usecase error を `404` または `500` に変換する
- `Like` は context から user を取得できる場合だけ `LikeWithUser` を呼び、未ログインなら `Like` を呼ぶ

AI への指示:

- handler には「HTTP と usecase の変換」だけを書く
- 入力の trim、上限、権限、状態遷移などの業務判断が増えたら usecase に置く
- response JSON の shape は schema に閉じ込める

### api/api/schema

役割:

- HTTP request DTO
- HTTP response DTO
- JSON/query/path の field name 定義
- domain entity との変換

書き方:

- JSON tag、query tag は schema に書く
- entity に JSON tag を増やして API response に使い回さない
- request type は `XxxReq`、response type は `XxxRes` を基本にする
- entity への変換は `ToEntity`、entity から response への変換は `XxxResFromEntity` のような関数にする
- frontend contract なので field の削除、rename、型変更は慎重に扱う

Xroll の例:

- `api/api/schema/video.go` は `VideoRes`、`VideosRes`、`VideoSearchQueryReq`、`VideoCreateBulkReq` を定義している
- `VideoRes` は JSON response のための構造で、`domain/entity.Video` とは分離されている
- `TwitterMetaRes` や `CommentRes` のように nested response も schema 側に置いている

AI への指示:

- 新しい endpoint を作るときは、まず request/response DTO を schema に作る
- usecase/input_port に schema type を渡さない
- DB model を response に直接返さない

### api/api/middleware

役割:

- 認証
- 認可
- CSRF/trusted origin check
- request context への user 注入
- 共通の HTTP 横断処理

書き方:

- middleware は HTTP framework に依存してよい
- 認証 token/cookie の読み取りは middleware に閉じ込める
- handler では middleware が context に入れた user を取り出すだけにする
- 権限の最終判断が業務ルールの場合は usecase 側でも確認する

Xroll の例:

- `api/api/middleware/auth_middleware.go` は route group に適用される
- router では `Authenticate`、`AuthenticateIfPossible`、`AuthenticateCookieOrHeader` を使い分けている
- admin 系 route は middleware で system admin を要求している

AI への指示:

- endpoint 個別に token parsing を書かない
- 認証方式を増やす場合は middleware に追加し、handler の変更を最小にする

### api/api/csv

役割:

- CSV marshal/unmarshal など HTTP 入出力に近い変換処理

書き方:

- CSV の列名、文字コード、入出力形式など transport 固有の都合を置く
- domain/usecase に CSV の表現を漏らさない
- validation のうち業務ルールは domain/usecase に渡して判断する

Xroll の例:

- `api/api/csv/marchal.go`、`api/api/csv/unmarshal.go` が CSV 変換を担当する

### domain/entity

役割:

- アプリケーションの中核データ構造
- DB や HTTP に依存しない業務概念

書き方:

- field は業務概念として自然な名前にする
- JSON tag、GORM tag、Echo context、SQL type を持ち込まない
- entity は API response そのものではない
- DB の relation 表現ではなく、usecase が扱いやすい形にする

Xroll の例:

- `api/domain/entity/video.go` は `Video`、`Comment`、`VideoPlaylist`、`Tag` などを定義している
- `Video` は `VideoURL`、`ThumbnailURL`、`TwitterMeta`、`Taggings` など業務上必要な情報を持つが、GORM tag はない
- `api/domain/entity/user.go` は user の業務データを表現し、DB model とは別に存在する

AI への指示:

- 新しい業務概念が必要なら、まず entity として表現できるか考える
- HTTP response の都合だけで entity field を増やさない
- DB の join や preload のための field は model 側に置く

### domain/constructor

役割:

- entity を正しい初期状態で生成する
- 必須 field、enum、初期値を保証する

書き方:

- `NewXxxCreate`、`NewXxxUpdate` のように用途がわかる名前にする
- 不正な入力は validation error として返す
- default value をここで設定してよい
- DB 保存や外部 API 呼び出しはしない

Xroll の例:

- `api/domain/constructor/user.go` の `NewUserCreate` は id、name、age、userType などを検証し、`LikesPublic: true` の初期値を設定している
- `NewUserUpdate` は name、bio、twitterURL の上限を検証する

AI への指示:

- entity の作成条件が複雑になったら constructor に寄せる
- interactor 内でバラバラに entity 初期化を増やしすぎない

### domain/validation

役割:

- ドメインとして再利用される形式チェック
- email、password、login ID などの制約

書き方:

- framework 非依存の pure function にする
- error は domain の validation error として返す
- 正規表現や文字数制限はここに閉じ込める
- request 固有の bind error は handler/schema 側に残す

Xroll の例:

- `api/domain/validation/user.go` は `ValidateEmail`、`ValidatePassword`、`ValidateLoginID` を持つ
- `entconst.NewValidationError(...)` を使って validation error を返している

AI への指示:

- 複数 usecase で使う入力制約は validation に置く
- 一つの usecase だけの都合なら interactor の helper に置いてもよい

### domain/entconst

役割:

- ドメイン定数
- enum 的な値
- validation error、共通 error
- sort や file type などの固定値

書き方:

- magic string を usecase や handler に散らさない
- exported const/type は意味がわかる名前にする
- HTTP status code はここに置かない

Xroll の例:

- `api/domain/entconst/user_detail.go` は user type などを定義している
- `api/domain/entconst/errors.go`、`errorvalidation.go` は domain error を定義している

AI への指示:

- 同じ文字列や状態値を複数箇所で使うなら entconst に寄せる
- API response 用の文字列だけなら schema 側に置く

### usecase/input_port

役割:

- handler から見える usecase interface
- usecase に渡す入力 DTO

書き方:

- interface は feature 単位で `IXxxUseCase` にする
- 引数は domain entity または input_port DTO にする
- schema type、Echo context、GORM model は使わない
- handler が必要とする操作だけを公開する

Xroll の例:

- `api/usecase/input_port/video.go` の `IVideoUseCase` は検索、作成、like、playlist、comment、report など handler から必要な操作を定義している
- `VideoSearch`、`VideoAddTags`、`TwitterMeta` のような usecase 入力 DTO も同じ file に置いている

AI への指示:

- handler に新しい操作が必要なら input_port に method を追加する
- interactor 固有の private helper は input_port に出さない

### usecase/interactor

役割:

- usecase の具体実装
- 業務フロー
- 権限チェック
- 入力正規化
- ID 発行、時刻取得
- repository/external adapter の呼び出し順序
- transaction 境界

書き方:

- struct は output port interface を field に持つ
- constructor は `NewXxxUseCase(...) input_port.IXxxUseCase` を返す
- `clock` や `ulid` も interface として受け取る
- `ErrKind.BadRequest`、`ErrKind.NotFound` など usecase error に変換する
- limit/offset の上限は usecase で守る
- helper function を切り出して handler を薄く保つ

Xroll の例:

- `api/usecase/interactor/video.go` の `VideoUseCase` は `videoRepo output_port.VideoRepository`、`ulid output_port.ULID`、`clock output_port.Clock` を持つ
- `Search` は現在時刻を使って期間条件を作り、repository に検索条件を渡す
- `FindByID` は `gorm.ErrRecordNotFound` を `ErrKind.NotFound` に変換している
- `api/usecase/interactor/contact.go` は trim、文字数上限、email parse、admin 権限、status 条件などの業務処理を持つ

AI への指示:

- 業務ルールは handler ではなく interactor に書く
- repository method をただ横流しするだけなら、本当に usecase method が必要か確認する
- DB query の詳細は repository に任せる

### usecase/output_port

役割:

- usecase から外部へ出る依存の interface
- repository、transaction、email、file、clock、ulid、external API の抽象化

書き方:

- interface は usecase が必要とする操作だけに絞る
- domain entity または output_port DTO を返す
- GORM model、SQL row、AWS SDK 型などを返さない
- DB 検索条件は output_port 用 DTO にまとめる

Xroll の例:

- `api/usecase/output_port/video.go` の `VideoRepository` は `Search`、`FindByID`、`CreateLikeIfAbsent`、`ListCommentsByVideoID` など DB に必要な操作を定義している
- `VideoSearch` は repository に渡す検索条件として `Limit`、`Offset`、`Query`、`Start`、`End`、`OrderBy` などを持つ
- `api/usecase/output_port/email.go`、`clock.go`、`ulid.go` は外部依存を小さな interface にしている

AI への指示:

- interactor が新しい外部操作を必要としたら、まず output_port に interface を追加する
- adapter の都合で interface を大きくしない

### usecase/input_port_mock and usecase/output_port_mock

役割:

- unit test 用 mock
- usecase test で外部依存を差し替える

書き方:

- 原則として generator で作る
- 生成物を手で編集しない
- interface を変更したら mock を再生成する

Xroll の例:

- `api/usecase/output_port_mock/video.go`、`user.go`、`clock.go`、`ulid.go` などが usecase test で使われる
- `api/README.md` に `mockgen` の例がある

AI への指示:

- mock file を直接修正しない
- test が interface 変更で壊れたら generator の使い方を確認する

### adapter/database/model

役割:

- GORM model
- table、column、index、relation、constraint の表現
- DB model から domain entity への変換

書き方:

- GORM tag は model にだけ書く
- relation、foreign key、index はここに閉じ込める
- domain entity と 1:1 にしなくてよい
- `Entity()` や `modeltoentity` で domain entity に変換する

Xroll の例:

- `api/adapter/database/model/model.go` は `User`、`Video`、`VideoLike`、`VideoPlaylist`、`TwitterVideoMeta` などの GORM model を定義している
- `Video.CommentCount` は `gorm:"->;column:comment_count"` のように query result 用 field を持つ
- `api/adapter/database/model/modeltoentity.go` は model から entity への変換を担当している

AI への指示:

- DB schema や preload の都合は model に置く
- model を handler response として返さない
- schema 変更が必要なら migration も必要か確認する

### adapter/database/repository

役割:

- output port repository interface の実装
- GORM/SQL query
- transaction 内の DB 操作
- model/entity 変換

書き方:

- constructor は `NewXxxRepository(db, ...) output_port.XxxRepository` にする
- method は output_port interface を満たす
- query は parameterized にする
- search 条件、pagination、order、preload は repository に閉じ込める
- N+1 query を避け、必要なら join/preload を使う
- DB error は必要に応じて wrap/convert する

Xroll の例:

- `api/adapter/database/repository/video.go` の `VideoRepository.Search` は GORM で `videos` を検索し、comment count join、preload、fulltext search、order を repository に閉じ込めている
- `Create` は `clause.OnConflict` を使って重複時の更新を DB 層で扱っている
- `api/adapter/database/repository/contact.go` は contact inquiry の永続化を担当する

AI への指示:

- usecase から「この条件で検索したい」と言われたら、output_port DTO を受け取って repository で query に変換する
- raw SQL が必要な場合も、文字列連結で user input を混ぜない

### adapter/database/initdata and transaction

役割:

- 初期データ投入
- 開発環境用 seed
- transaction 実装

書き方:

- initdata は開発/test の補助として扱う
- production data を勝手に変更する処理を書かない
- transaction は usecase/output_port の transaction interface を満たす

Xroll の例:

- `api/adapter/database/initdata/user.go` は development 用 default user を作る
- `api/adapter/database/transaction.go` は GORM transaction を output port として提供する

### adapter/authentication

役割:

- JWT
- password hash
- authentication code
- TOTP
- user auth 実装

書き方:

- 認証技術の詳細を adapter に閉じ込める
- usecase には `output_port.UserAuth` や `AuthenticationCode` のような interface で渡す
- secret や token の読み取りは config 経由にする
- password や token を log に出さない

Xroll の例:

- `api/adapter/authentication/jwt.go`、`bcrypt.go`、`totp.go`、`user_auth.go` が認証処理を担当している
- `main.go` では `authentication.NewUserAuth()` を作って usecase に渡している

### adapter/email, adapter/file, adapter/twitter, adapter/gofile

役割:

- 外部サービスや外部I/Oの concrete implementation
- email、S3/file、Twitter/X、GoFile API など

書き方:

- external SDK の型を usecase/domain に漏らさない
- constructor は output port interface を返す
- timeout、credential、API error handling を adapter に閉じ込める
- mock driver が必要なら adapter 内または output_port_mock で用意する

Xroll の例:

- `api/adapter/email/email_aws.go` は Amazon SES の `ses.SendEmailInput` を adapter 内だけで扱い、`output_port.Email` を実装している
- `api/adapter/file/file.go` は file driver として pre-signed URL などを扱う
- `api/adapter/twitter/twitter.go` は Twitter/X 取得処理を閉じ込めている
- `api/adapter/gofile/gofile.go` は GoFile API との通信を担当する

AI への指示:

- 新しい外部 service を追加するときは、まず output_port に小さい interface を作る
- SDK response をそのまま entity にしない

### adapter/aws, adapter/cache, adapter/clock, adapter/ulid

役割:

- 横断的な infrastructure 実装
- AWS session/client
- cache singleton
- system clock
- ID generator

書き方:

- usecase から直接 `time.Now()` や random ID generator を呼ばせない
- clock や ulid は interface 化して test で差し替える
- AWS client の生成は adapter/aws に閉じ込める

Xroll の例:

- `api/adapter/clock/clock.go` は `output_port.Clock` を実装する
- `api/adapter/ulid/ulid.go` は `output_port.ULID` を実装する
- `main.go` で `clock.New()`、`ulid.NewULID()` を作って usecase に渡している

AI への指示:

- 時刻や ID が絡む usecase は `clock`、`ulid` を dependency として受け取る
- test で固定できない実装を interactor に直接書かない

### config

役割:

- 環境変数の読み取り
- 環境判定
- credential や URL の getter
- config validation

書き方:

- `os.Getenv` は config に集約する
- 各 package で環境変数名を直接参照しない
- secret value を log に出さない
- default 値を使う場合は意図を明確にする

Xroll の例:

- `api/config/config.go` は `SIG_KEY`、`ENV`、`AWS_REGION`、`S3_BUCKET`、`FRONTEND_URL` などを読み取り、getter を提供している
- `IsDevelopment`、`IsTest`、`IsAWSConfigFilled` のような環境判定関数を持つ

AI への指示:

- 新しい env var が必要な場合は config に getter を追加する
- ただし env var の追加は deploy 影響があるため、ユーザーの明示指示なしに増やさない

### log

役割:

- structured logger の生成
- 環境ごとの logger 切り替え

書き方:

- `zap` など structured logging を使う
- `fmt.Println` を runtime log に使わない
- test では noise を抑える logger を使う
- secret や token を log に出さない

Xroll の例:

- `api/log/log.go` は development で `zap.NewDevelopment`、test で `zap.NewNop`、それ以外で `zap.NewProduction` を返す

AI への指示:

- handler や main で log が必要な場合は logger を使う
- error は `zap.Error(err)` のように structured field として出す

### utils and testutil

役割:

- 汎用 helper
- set など小さな utility
- test 用 random data/helper

書き方:

- domain/usecase に置くべき業務ロジックを utils に逃がさない
- 汎用性が本当にあるものだけ置く
- test 専用 helper は `testutil` に置く

Xroll の例:

- `api/utils/set/set.go` は汎用 set helper
- `api/testutil/random.go` は test 用 random helper

AI への指示:

- 迷ったら utils に置かず、feature の近くに private helper として置く
- 複数箇所で必要になってから共通化する

### mysql/migrations and mysql/init_data

役割:

- DB migration
- 初期投入データ
- MySQL 設定

書き方:

- migration は up/down を揃える
- schema 変更は明示指示があるときだけ行う
- index、constraint、data migration の影響を考える
- application code と migration の整合性を保つ

Xroll の例:

- `api/mysql/migrations/000001_prod_schema_sync.up.sql` 以降に schema 変更が seq で管理されている
- `api/mysql/init_data/user.json` は初期 user data として使われる

AI への指示:

- model を変えても migration が不要とは限らない
- DB schema 変更を勝手に作らない
- 必要ならユーザーに確認する

### integration_test

役割:

- API contract test
- DB 込みの user flow test
- 認証、権限、error response、security test

書き方:

- frontend が依存する request/response は integration test で守る
- real DB を使う test と stub usecase test を分ける
- helper を使って fixture 作成を共通化する
- happy path だけでなく error path も書く

Xroll の例:

- `api/integration_test/realdb_video_api_test.go` は video API の DB 込み test
- `api/integration_test/realdb_auth_security_test.go` は認証/security 系 test
- `api/integration_test/stub_video_usecase_test.go` は stub usecase を使った handler/router test
- `api/integration_test/realdb_seed_helpers_test.go` は fixture helper を提供している

AI への指示:

- API response を変えたら integration test を確認、更新する
- bug fix では再発防止になる最小 test を追加する

### cmd

役割:

- batch
- admin/maintenance command
- 検証用 CLI
- ranking job など HTTP server とは別の entrypoint

書き方:

- HTTP server とは別の `main` package として置く
- 業務処理を cmd に直接書きすぎず、usecase や repository を再利用する
- 一時的な検証 command は production job と混同しない名前にする
- DB や external service を使う場合は config と adapter を通す

Xroll の例:

- `api/cmd/dailyranking/main.go`、`api/cmd/ranking7days.go` は ranking 系 job
- `api/cmd/verifyranking/main.go`、`api/cmd/scorecalc/main.go` は検証や計算用 command

AI への指示:

- 定期実行や one-shot job は `cmd` に置く
- 既存 API handler を無理に呼ばず、下の usecase/repository を使う

### docs and runbooks

役割:

- 設計メモ
- 運用手順
- rollback 手順
- AI 向け開発ルール

書き方:

- 実装とズレる古い説明を残さない
- 手順は command、前提、確認方法を明確にする
- architecture doc は AI が迷わない粒度で書く

Xroll の例:

- `docs/runbooks/rollback.md` は運用手順
- `docs/ai-clean-architecture-guide.md` は AI 向けの設計・実装ルール

## Layer Responsibilities

### domain

`domain` はアプリケーションの中核データとルールを置く層。
HTTP、DB、外部 API、認証ライブラリ、framework に依存してはいけない。

置くもの:

- `entity`: User、Video、Order などの中核データ構造
- `constructor`: entity を正しい初期状態で生成する処理
- `validation`: ドメイン上の入力制約、形式チェック
- `entconst`: ドメイン定数、エラー種別、列挙値

置かないもの:

- Echo、Gin、net/http などの handler 処理
- GORM、SQL、DB model
- JSON request/response DTO
- AWS、Stripe、Twitter などの外部 API 呼び出し

判断基準:

- そのコードは DB が MySQL から PostgreSQL に変わっても残るか
- そのコードは HTTP API ではなく CLI から呼んでも意味があるか
- そのコードは業務概念として説明できるか

答えが yes なら `domain` に置く候補。

### usecase

`usecase` はアプリケーション固有の処理手順を置く層。
handler から呼ばれ、repository や外部 service を interface 越しに使う。

#### input_port

handler から呼び出される usecase interface を定義する。

例:

```go
type IVideoUseCase interface {
    Search(search VideoSearch) ([]entity.Video, error)
    FindByID(id string) (entity.Video, error)
    LikeWithUser(user entity.User, videoID string) error
}
```

ルール:

- handler は concrete interactor ではなく input port interface に依存する
- request/response schema を input port に入れない
- 引数と戻り値は原則として domain entity または usecase 用 DTO にする

#### interactor

usecase の実装を置く。
業務フロー、権限判断、入力正規化、repository 呼び出し順序、トランザクション制御を担当する。

ルール:

- DB を直接触らない
- HTTP status code を返さない
- Echo context や request object を受け取らない
- 外部 API adapter を直接 new しない
- 必要な依存は constructor で interface として受け取る
- NotFound、BadRequest などの usecase error に変換する

#### output_port

DB、外部 API、時刻、ID 発行、メール、ファイル保存など、usecase から外へ出る依存の interface を定義する。

例:

```go
type VideoRepository interface {
    Search(search VideoSearch) ([]entity.Video, error)
    FindByID(id string) (entity.Video, error)
    Create(video entity.Video) error
}
```

ルール:

- interface は usecase が必要とする操作だけに絞る
- GORM model や SQL 固有型を返さない
- 外部サービスの SDK 型を返さない
- adapter はこの interface を実装する

### adapter

`adapter` は外部技術の実装を置く層。
MySQL、GORM、AWS、JWT、メール送信、外部 API、ID 生成などを担当する。

置くもの:

- `database/repository`: output port の repository 実装
- `database/model`: GORM model
- `database/transaction`: transaction 実装
- `authentication`: JWT、bcrypt、TOTP など
- `email`: メール送信 driver
- `file`: S3 やローカルファイル driver
- `clock`: 現在時刻 provider
- `ulid`: ID generator
- 外部 API client

ルール:

- adapter は `usecase/output_port` の interface を実装する
- DB model と domain entity の変換は adapter 内で行う
- SQL は parameterized query または ORM の安全な API を使う
- string concatenation で SQL を組み立てない
- N+1 query を避ける
- 外部 SDK の型を usecase や domain に漏らさない

### api

`api` は HTTP API の入口を置く層。
Echo/Gin などの framework はこの層に閉じ込める。

#### router

route 定義、middleware 適用、handler の接続を担当する。

ルール:

- endpoint path の変更は互換性に注意する
- route は feature 単位で整理する
- 認証必須、任意認証、管理者権限などの middleware を明示する

#### handler

HTTP request を受け取り、schema に bind し、usecase を呼び、HTTP response に変換する。

ルール:

- DB に直接アクセスしない
- adapter を直接呼ばない
- 業務ロジックを厚くしない
- HTTP status code への変換は handler で行う
- request validation のうち HTTP 入力形式に関するものは handler/schema 側で行う
- 業務上の validation は usecase/domain に寄せる

#### schema

HTTP request/response DTO を置く。

ルール:

- JSON tag、query tag は schema に閉じ込める
- domain entity との変換関数を用意する
- API response format を変えると frontend に影響するため慎重に扱う

### main.go

`main.go` は composition root として扱う。
repository、adapter、usecase、handler、router を組み立てる。

ルール:

- 依存の wiring をここに集約する
- usecase 内で adapter を new しない
- 環境変数、DB 接続、logger 初期化、server 起動はここで扱う
- main に業務ロジックを書かない

## Dependency Rules

許可される依存:

```text
api/handler -> usecase/input_port
api/schema -> domain/entity
api/router -> api/handler, usecase/input_port
usecase/interactor -> domain/entity, usecase/input_port, usecase/output_port
adapter/database/repository -> usecase/output_port, domain/entity, adapter/database/model
main.go -> all outer constructors
```

禁止される依存:

```text
domain -> api
domain -> adapter
domain -> usecase
usecase -> api/handler
usecase -> api/schema
usecase -> adapter/database/model
usecase -> gorm.DB
handler -> adapter/database/repository
handler -> gorm.DB
repository -> api/schema
```

AI は実装中に依存方向で迷ったら、内側の層を汚さない選択をする。

## Data Model Separation

このアーキテクチャでは、似た構造体が複数存在してよい。
むしろ責務が違うなら分ける。

```text
api/schema        HTTP request/response 用 DTO
domain/entity    アプリケーションの中核データ
adapter/model     DB/ORM 用 model
usecase DTO       usecase 固有の入力条件
```

分離する理由:

- API response の都合を domain に持ち込まない
- DB schema の都合を usecase に持ち込まない
- 外部 service の型変更が中核層に波及しないようにする

変換の置き場所:

- schema -> entity: `api/schema` 側
- entity -> schema: `api/schema` 側
- model -> entity: `adapter/database/model` または repository 側
- entity -> model: repository 側

## Error Handling

エラーは層ごとに責務を分ける。

repository:

- DB error を受け取る
- NotFound、duplicate、constraint violation などを必要に応じて output port/usecase error に変換できる形にする
- DB 固有 error をそのまま上位に漏らしすぎない

usecase:

- 業務上の `BadRequest`、`NotFound`、`Forbidden`、`Conflict` を判断する
- 外部依存の失敗は文脈を維持して返す

handler:

- usecase error を HTTP status code に変換する
- internal error の詳細を response に出しすぎない

ルール:

- error は必ず確認して返す
- `_` で error を捨てない
- log は structured logging を使う
- `fmt.Println` を logging に使わない

## Feature Implementation Flow

AI が新機能を追加するときは、原則として次の順序で進める。

1. 既存の似た feature を探す
2. domain entity に必要な概念を追加する
3. usecase/input_port に操作を追加する
4. usecase/output_port に必要な repository/external dependency を追加する
5. usecase/interactor に業務フローを実装する
6. adapter/database/model または external adapter を実装する
7. adapter/database/repository で output port を満たす
8. api/schema に request/response DTO を追加する
9. api/handler で bind、usecase 呼び出し、response 変換を実装する
10. api/router に route を追加する
11. main.go で依存を wire する
12. unit test と integration test を追加または更新する
13. `gofmt`、`go test ./...` を実行する

DB schema 変更が必要な場合は、migration を明示的に追加する。
ただし、DB schema 変更は影響が大きいため、ユーザーの明示指示がない限り行わない。

## Adding a New Resource

例として `Bookmark` 機能を追加する場合の配置。

```text
domain/entity/bookmark.go
usecase/input_port/bookmark.go
usecase/output_port/bookmark.go
usecase/interactor/bookmark.go
adapter/database/model/model.go
adapter/database/repository/bookmark.go
api/schema/bookmark.go
api/handler/bookmark.go
api/router/router.go
main.go
```

handler は `IBookmarkUseCase` だけを見る。
interactor は `BookmarkRepository` interface だけを見る。
repository が GORM と MySQL を扱う。

## Testing Policy

テストはリスクと責務に合わせて置く。

usecase/interactor:

- 業務ロジック、権限、状態遷移、エラー分岐を unit test する
- output port mock を使う

adapter/database/repository:

- 複雑な query、join、transaction、unique constraint を test する
- DB 依存がある場合は integration test に寄せる

api/handler/router:

- request binding、status code、response format、認証 middleware を test する

integration_test:

- frontend が依存する API contract
- DB を含む主要 user flow
- 認証、権限、セキュリティ境界

必須:

```sh
go test ./...
```

## API Contract Rules

API path、request JSON、response JSON は frontend と外部 client が依存する contract として扱う。

ルール:

- 明示指示なしに endpoint path を変えない
- 明示指示なしに response field を削除、rename しない
- 既存 field の意味を変えない
- 互換性を壊す場合は migration plan または versioning を検討する
- schema 変更時は integration test を更新する

## Database Rules

DB は adapter 層の責務。

ルール:

- handler や usecase から `gorm.DB` を直接使わない
- SQL は parameterized query を使う
- N+1 query を避ける
- pagination の limit と offset に上限を設ける
- soft delete、公開範囲、認可条件を query で漏らさない
- migration 追加は明示指示があるときだけ行う
- index 影響を考える

## External Service Rules

外部サービスは adapter に閉じ込める。

例:

- AWS S3
- SES
- Twitter/X API
- Stripe
- SendGrid
- OpenAI API
- GoFile API

ルール:

- SDK client を usecase や handler で直接 new しない
- output port に必要最小限の interface を定義する
- 外部 API の response 型を domain に漏らさない
- retry、timeout、rate limit、credential は adapter/config で扱う

## AI Implementation Checklist

AI は実装前に次を確認する。

- 似た feature はどこにあるか
- 変更対象はどの層か
- API contract を壊していないか
- DB schema 変更が必要か
- migration が必要なら明示指示があるか
- usecase に業務ロジックが入り、handler が薄いままか
- repository に DB 詳細が閉じ込められているか
- domain が framework や DB に依存していないか
- test をどこに追加すべきか

実装後に次を確認する。

- `gofmt` 済みか
- `go test ./...` が通るか
- error を握りつぶしていないか
- structured logging を使っているか
- SQL injection の余地がないか
- N+1 query を増やしていないか
- request/response format を意図せず変えていないか

## Anti Patterns

避けること:

- handler から repository を直接呼ぶ
- usecase で GORM model を使う
- domain entity に JSON tag や GORM tag を大量に付ける
- repository で HTTP response DTO を返す
- main.go 以外で依存を勝手に new する
- business logic を handler に書く
- DB schema 変更を migration なしで行う
- 外部 API SDK の型を domain/usecase に漏らす
- test のためだけに production code の責務を崩す
- unrelated refactor を混ぜる

## Naming Conventions

Go の標準に従う。

- exported: `PascalCase`
- unexported: `camelCase`
- interface は利用者側の package に置く
- repository interface は usecase/output_port に置く
- handler constructor は `NewXxxHandler`
- usecase constructor は `NewXxxUseCase`
- repository constructor は `NewXxxRepository`

## Practical Rule for AI

AI は「どこに書くべきか」で迷ったら、次の基準で決める。

```text
HTTP の話か
  -> api/handler, api/schema, api/router

業務フローの話か
  -> usecase/interactor

中核データや業務概念の話か
  -> domain/entity, domain/validation

DB query や永続化の話か
  -> adapter/database/repository, adapter/database/model

外部サービスの話か
  -> adapter/external_service_name

依存の組み立ての話か
  -> main.go
```

内側の層に外側の都合を入れない。
このルールを守ることを、実装速度より優先する。
