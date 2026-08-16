# brainhub ドメイン設計

`docs/architecture-guide.md` の `domain` 層に置く定義。実運用を前提とする。

## 前提となる制約

brainhub の状態は **2つのシステムに分散する**。

| | 持つもの |
|---|---|
| brainhub DB | 人、所有、発行した鍵の記録 |
| GBrain | source、OAuth クライアント、知識そのもの |
| ファイルシステム | 正本 Markdown と git |

三者に跨る共通トランザクションは存在しない。したがってドメインは「両者がズレうる」ことを前提に設計する。ズレを検出できない設計は本番で必ず事故る。

具体的な事故:

- Brain 行はあるが GBrain に source が無い → 誰も書けない幽霊の脳
- GBrain にクライアントが存在するが `issued_clients` に記録が無い → **所有者不明の有効な鍵**
- Membership を消したが GBrain のクライアントを失効し忘れた → **退職者が読み続けられる**

これらを型と状態で防ぐ。

---

## entity

### SourceID

```go
// domain/entity/source_id.go
package entity

// GBrain の source id。引用キーとして永続に使われるため変更不可。
// GBrain 側の制約: [a-z0-9-]{1,32}
type SourceID string

func (s SourceID) String() string { return string(s) }
```

生成は `constructor` を経由する。`SourceID("任意の文字列")` を書ける場所を作らない。

### Brain

```go
// domain/entity/brain.go
package entity

type Brain struct {
    ID          string      // ULID。brainhub 内の識別子
    SourceID    SourceID    // GBrain 側の識別子。不変
    Name        string      // 表示名。変更可
    Description string
    Visibility  Visibility
    OwnerID     string
    State       BrainState
    StateReason string      // failed / degraded のときの理由
    CreatedAt   time.Time
    UpdatedAt   time.Time
    ArchivedAt  *time.Time
}

func (b Brain) IsReadable() bool { return b.State == BrainStateReady }
func (b Brain) IsArchived() bool { return b.ArchivedAt != nil }
```

**`ID` と `SourceID` を分ける理由。** 表示名は変えられるが source id は変えられない。ひとつのフィールドで両方を担うと、改名要求が来たときに引用が壊れるか、改名を断るかの二択になる。

**`State` を持つ理由。** 上記の分散状態の帰結。`sources add` の成否がそのまま状態になる。

### BrainState

```go
// domain/entity/entconst/brain_state.go
type BrainState string

const (
    BrainStateProvisioning BrainState = "provisioning" // DB 行はある。GBrain 側は未確認
    BrainStateReady        BrainState = "ready"        // 両者が揃っている
    BrainStateDegraded     BrainState = "degraded"     // 存在するが同期が古い等
    BrainStateFailed       BrainState = "failed"       // 作成に失敗。再試行または削除の対象
    BrainStateArchived     BrainState = "archived"     // 論理削除
)
```

遷移は一方向に限定する。

```
provisioning ──> ready ──> degraded ──> ready
      │            │            │
      └──> failed  └──────> archived <──┘
```

`failed` から `ready` へ直接は遷移しない。作り直して `provisioning` からやり直す。

### Visibility

```go
type Visibility string

const (
    VisibilityPrivate Visibility = "private" // 招待された者のみ
    VisibilityPublic  Visibility = "public"  // 誰でも
)
```

`unlisted` は現時点で作らない。要求が出てから足す。

### User

```go
// domain/entity/user.go
type User struct {
    ID        string
    Email     string
    Name      string
    State     UserState
    CreatedAt time.Time
    UpdatedAt time.Time
}

type UserState string

const (
    UserStateActive     UserState = "active"
    UserStateSuspended  UserState = "suspended"  // 全クライアントを失効させる
)
```

`suspended` は退職・不正利用時の一括遮断に使う。この状態を持たないと、鍵を1本ずつ探して失効させることになる。

### Membership

```go
// domain/entity/membership.go
type Membership struct {
    ID        string
    BrainID   string
    UserID    string
    Role      Role
    InvitedBy string
    CreatedAt time.Time
    UpdatedAt time.Time
    RevokedAt *time.Time
}

func (m Membership) IsActive() bool { return m.RevokedAt == nil }
func (m Membership) CanInvite() bool { return m.IsActive() && m.Role == RoleOwner }
func (m Membership) CanWrite() bool {
    return m.IsActive() && (m.Role == RoleOwner || m.Role == RoleEditor)
}
```

**物理削除しない。** 「誰がいつ外れたか」は監査で必要になる。

### Role

```go
type Role string

const (
    RoleOwner  Role = "owner"
    RoleEditor Role = "editor"
    RoleReader Role = "reader"
)

// GBrain の --scopes へ落とす。ここが唯一の対応表。
func (r Role) GBrainScopes() []string {
    switch r {
    case RoleOwner, RoleEditor:
        return []string{"read", "write"}
    default:
        return []string{"read"}
    }
}
```

対応表を1箇所に閉じ込める。usecase や adapter で分岐を書かない。

### Invitation

```go
// domain/entity/invitation.go
type Invitation struct {
    ID        string
    BrainID   string
	Email     *string     // NULL なら誰でも受諾できる
    Role      Role
    TokenHash string      // 平文は保存しない
    InvitedBy string
    State     InvitationState
    ExpiresAt time.Time
    CreatedAt time.Time
	AcceptedAt *time.Time
	AcceptedBy *string
}

type InvitationState string

const (
    InvitationStatePending  InvitationState = "pending"
    InvitationStateAccepted InvitationState = "accepted"
    InvitationStateRevoked  InvitationState = "revoked"
    InvitationStateExpired  InvitationState = "expired"
)

func (i Invitation) IsAcceptable(now time.Time) bool {
    return i.State == InvitationStatePending && now.Before(i.ExpiresAt)
}
```

**Membership とは別の entity にする理由。** 「招待したがまだ受けていない」は所属ではない。Membership に `pending` を混ぜると、権限判定のたびに状態を確認する分岐が全経路に散る。

`Email` を持つのは、まだアカウントの無い相手を招待できるようにするため。

### IssuedClient

```go
// domain/entity/issued_client.go
type IssuedClient struct {
    ID             string
    UserID         string
    BrainID        string
    GBrainClientID *string     // issuing 中や発行失敗時はまだ存在しない（DBもNULL可）
    Label          string      // "codex", "claude-desktop" 等
	WriteSourceID  *SourceID   // reader は書き込み先を持たない
    ReadSourceIDs  []SourceID  // GBrain の --federated-read
    Scopes         []string
	State          ClientState
    StateReason    string      // 外部発行・失効の失敗理由（DBのstate_reason）
    IssuedAt       time.Time
    LastVerifiedAt *time.Time  // GBrain 側に実在することを最後に確認した時刻
    RevokedAt      *time.Time
}

type ClientState string

const (
    ClientStateIssuing ClientState = "issuing" // GBrain へ発行要求中
    ClientStateActive  ClientState = "active"
    ClientStateRevoked ClientState = "revoked"
    ClientStateOrphan  ClientState = "orphan"  // GBrain に無い、または記録と食い違う
)
```

**secret は保存しない。** confidential clientで返る場合も保持しない。現在発行する `token_endpoint_auth_method=none` のpublic clientにはsecret自体がない。

`GBrainClientID` をnullableにするのは、`issuing` 行を先にコミットしてからGBrainへ発行要求するためである。発行失敗時はclient IDが無いまま `orphan` になり、失敗理由は `StateReason` に残る。

**`LastVerifiedAt` を持つ理由。** GBrain の `auth list` と突き合わせて、記録に無いクライアントや失効済みのはずのクライアントを検出するため。これが無いと、上に挙げた「所有者不明の有効な鍵」を見つける手段が無い。

### Page

```go
// domain/entity/page.go
type Page struct {
    Slug      string
    Title     string
    Type      string
    UpdatedAt time.Time
    Body      string
}
```

**DB に持たない。** GBrain から取得してそのまま返す。テーブルもマイグレーションも作らない。正本を二重に持つと、必ずどちらかが古くなる。

---

## constructor

```go
// domain/constructor/source_id.go
var sourceIDPattern = regexp.MustCompile(`^[a-z0-9-]{1,32}$`)

var reservedSourceIDs = map[string]bool{
    "default": true, // GBrain が予約。削除不可
}

func NewSourceID(s string) (entity.SourceID, error) {
    if !sourceIDPattern.MatchString(s) {
        return "", entconst.NewValidationError("source_id",
            "英小文字・数字・ハイフンのみ、1〜32文字")
    }
    if reservedSourceIDs[s] {
        return "", entconst.NewValidationError("source_id", "予約語は使用できません")
    }
    return entity.SourceID(s), nil
}
```

`default` を弾くのは実地の知見。GBrain の `default` source は `sources remove` を拒否する。

```go
// domain/constructor/brain.go
func NewBrainCreate(
    id string, sourceID entity.SourceID, name string,
    description string, visibility entity.Visibility, ownerID string,
    now time.Time,
) (entity.Brain, error) {
    if err := validation.ValidateBrainName(name); err != nil {
        return entity.Brain{}, err
    }
    if visibility == "" {
        visibility = entity.VisibilityPrivate // 既定は private
    }
    return entity.Brain{
        ID: id, SourceID: sourceID, Name: name,
        Description: description, Visibility: visibility, OwnerID: ownerID,
        State:     entity.BrainStateProvisioning, // ready から始めない
        CreatedAt: now, UpdatedAt: now,
    }, nil
}
```

**既定を `private` にする。** 公開は明示的な操作でのみ起きる。

**`provisioning` から始める。** GBrain 側の作成が済むまで `ready` にしない。

---

## validation

```go
// domain/validation/brain.go
func ValidateBrainName(s string) error        // 1〜80文字、制御文字なし
func ValidateBrainDescription(s string) error // 0〜500文字
func ValidateClientLabel(s string) error      // 1〜32文字

// domain/validation/slug_prefix.go
// GBrain の --bound-slug-prefixes は末尾が '/' または '/*' でないと
// 境界が効かず、"emp-alice" が "emp-alice-2/..." も含んでしまう。
func ValidateSlugPrefix(s string) error
```

最後のものは GBrain のヘルプに明記された落とし穴。ドメインの制約として持たないと、adapter か usecase のどこかで書き忘れて越境を許す。

---

## 不変条件

usecase で守る。ドメインのコメントに書き残す。

1. Brain の `SourceID` は作成後に変更されない。
2. `Visibility` が `public` へ変わるのは明示的な操作のみ。既定は `private`。
3. Membership を revoke したら、その User の当該 Brain 向け IssuedClient をすべて revoke する。
4. User が `suspended` になったら、その User の IssuedClient をすべて revoke する。
5. `ready` でない Brain は公開ページに出さない。
6. IssuedClient の secret はどの層にも保存しない。
7. 権限判定は Membership のみを根拠とする。Invitation は根拠にしない。

3 と 4 を守らないと、退職者が読み続けられる。

---

## 照合（reconciliation）

分散状態を運用可能にする最小の仕組み。

```go
// usecase/output_port/reconciler.go
type BrainReconciler interface {
    // GBrain の sources list と brains 表を突き合わせる
    ListGBrainSources(ctx context.Context) ([]entity.SourceID, error)
    // GBrain の auth list と issued_clients を突き合わせる
    ListGBrainClients(ctx context.Context) ([]GBrainClientRef, error)
}
```

定期実行で以下を検出する。

| 検出 | 意味 | 対処 |
|---|---|---|
| Brain あり / source なし | 作成が途中で失敗 | `failed` へ |
| source あり / Brain なし | 手動で作られた source | 記録のみ。消さない |
| IssuedClient あり / GBrain client なし | 記録が古い | `orphan` へ |
| GBrain client あり / 記録なし | **所有者不明の鍵** | 警告。手動で失効判断 |

最後の行が一番重要である。自動失効はしない — 運用者が手で作った鍵を消してしまうため。検出して知らせるところまでを責務とする。

---

## 今回作らないもの

掟 1（入口のない機能を作らない）に従う。以下は入口ができるまで作らない。

| | 理由 |
|---|---|
| `BrainRuntime`（cluster_ref / database_ref） | runtime 分離が必要になるまで不要。状態機械は Brain が持つ |
| Export / DataJob | 前のリポジトリで入口の無いまま 3,000 行が死んだ領域 |
| Session / Message | チャットを作らない |
| 課金・使用量 | GBrain の `--budget-usd-per-day` で上限は掛けられる。請求は後 |
| 監査ログ専用テーブル | まず各 entity の `RevokedAt` / `InvitedBy` で足りる |

---

## 実装順

| フェーズ | 必要な entity |
|---|---|
| 1. プロキシ | なし |
| 2. 公開ページ | `Page`, `Brain`, `SourceID` |
| 3. ユーザーと所有 | `User`, `Membership`, `Role` |
| 4. 招待 | `Invitation`, `IssuedClient`, 照合 |
| 5. 編集画面 | 追加なし |

フェーズ 2 で `Brain` を DB に持つ。公開ページには名前・説明・公開範囲が必要で、GBrain の source はそれらを持たないため。
