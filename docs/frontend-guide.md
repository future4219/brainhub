# brainhub フロントエンド開発指針

`docs/architecture-guide.md` が Go 側の指針であるのと同じ役割を、`web/` に対して持つ。AI に実装を任せるときは、このファイルを一緒に読ませる。

`README.md` の「掟」4項目はここでも最優先である。特に **掟 1（入口のない機能を作らない）** は、コンポーネントとフックにもそのまま適用する。

---

## 技術構成

| | |
|---|---|
| React + TypeScript + Vite | |
| Tailwind + shadcn/ui | トークンは `docs/claude-design/` から抽出 |
| Fetch API | HTTP クライアント。共通処理は `lib/api.ts` |
| ルーティング | react-router-dom（declarative mode）。App.tsx で定義 |
| 状態管理 | ライブラリを使わない。React 組み込みの state のみ |

状態管理ライブラリを入れないのは意図的な選択である。画面数が 10 に満たず、URL の階層も 2 段までのため。**必要になったら入れる。それまでは入れない。**
framework mode（ファイルベースルーティング）は使わない。
Vite の SPA 構成を維持する。

---

## ディレクトリ構成

```text
web/src/
├── App.tsx                       # BrowserRouter と Routes
├── main.tsx                      # エントリーポイント
├── components/
│   ├── pages/                    # Route が参照する薄い入口
│   ├── features/                 # 機能単位の Container / Presenter
│   │   └── Feature/
│   │       ├── FeatureContainer.tsx
│   │       └── FeaturePresenter.tsx
│   └── ui/                       # 複数機能で使う共通 UI
├── config/url.ts                 # パス定数と URL 組み立て
├── entities/<domain>/entity.ts   # API と画面が共有する型
├── hooks/                        # 複数機能で使う hook
├── lib/                          # API client、endpoint 関数、pure helper
└── styles/                       # design token
```


不要なディレクトリは作らない。空のディレクトリを置かない（掟 1）。

---

## Container / Presenter

**Container** — データ取得、状態、イベントハンドラ。副作用を持つ。
**Presenter** — props を受けて描画するだけ。副作用を持たない。

原則:
- Presenter は API を呼ばない。useEffect でデータを取らない
- Presenter は props だけに依存する。同じ props なら常に同じ描画
- Container は描画をほとんど持たない。Presenter に渡すだけ
- タブが独立した画面なら、タブごとに Presenter を作る。全タブの props を1つの Presenter に集めない
- **props を受けて描画するだけのものに Container を作らない。**
  データ取得か状態を持つ場合にのみ Container を作る

ルートに対応する Container が、その画面のデータ取得の唯一の入口になる。

## 依存の方向

App.tsx              → components/pages, config
pages                → components/features
feature Container    → feature Presenter, hooks, lib, entities
feature Presenter    → components/ui, config, entities, pure helper
hooks                → lib, entities
lib                  → entities

禁止:
Presenter → API endpoint 関数
Presenter → Container
lib       → components

### コンポーネントは通信しない

**データ取得は feature の `Container` で行い、`Presenter` には props で渡す。**

Go 側で handler が repository を直接呼ばないのと同じ理由である。コンポーネントが通信を始めると、再利用したときに意図しないリクエストが飛び、テストのたびにモックが要る。

```tsx
// 悪い
function BrainRow({ sourceId }) {
  const [pages, setPages] = useState([]);
  useEffect(() => { getPages(sourceId).then(setPages); }, [sourceId]);
}

// 良い
function BrainRow({ brain, pageCount }) { ... }
```

例外はない。「1 箇所だけだから」で崩さない。

---

## API 層

### 型を推測しない

**動いている API を実際に叩いて、返ってきた形から型を書く。**

この規約は実際の失敗から来ている。`GET /api/brains` のレスポンス形状を推測して実装した結果、API 側の変更で 3 箇所が同時に壊れた。

```bash
curl -s localhost:8080/api/brains | jq
curl -s localhost:8080/api/brains/brainhub/pages | jq '.[0]'
```

型は `entities/<domain>/entity.ts` に置く。コンポーネントや画面の中で API レスポンスの型を定義しない。

### HTTP の共通処理は 1 つ

```ts
// lib/api.ts
export async function requestJSON<T>(path: string, init?: RequestInit): Promise<T>
```

- base URL、cookie、JSON、HTTP error の変換を `lib/api.ts` に閉じ込める
- 認証が必要な画面は `?next=` で元の場所に戻れるようにする
- **セッションは HttpOnly cookie。** JS からトークンを読まない・保存しない
- `localStorage` / `sessionStorage` に認証情報を置かない

### エンドポイントごとに関数を切る

Container から `requestJSON("/brains/...")` を直接呼ばない。

```ts
// lib/brainApi.ts
export async function listBrains(): Promise<Brain[]>
export async function getPages(sourceId: string): Promise<Page[]>
```

API のパスが変わったとき、直す場所が `lib/*Api.ts` だけで済む。

### サーバーが知っていることをフロントで組み立てない

**MCP URL は `GET /api/config` から取得する。`window.location` から作らない。**

これも実際の失敗から来ている。配布した MCP URL を変更すると、接続済みの利用者全員が繋ぎ直すことになる。フロントの origin に依存させると、公開ページと MCP が別ドメインになった時点で壊れる。

同じ原則が他にも適用される。**外部に配る値、権限に関わる値、サーバーが正を持つ値は、フロントで導出しない。**

---

## コンポーネント

### 作る基準

**実際に 2 箇所以上で使うものだけを `components/ui/` に置く。**

1 機能だけで使うものは、その機能の `components/features/<Feature>/` に置く。将来使いそう、は理由にならない（掟 1）。

新しく共通コンポーネントを作ったら、**何箇所で使っているかを報告する。** 1 箇所なら画面側へ戻す。

### トークン以外の値を書かない

```tsx
// 悪い
<div className="p-[13px] text-[#eae6e0] rounded-[5px]">

// 良い
<div className="p-3 text-fg rounded-md">
```

hex、任意 px、font-family をコンポーネント内に書かない。`styles/tokens.css` と Tailwind の theme を経由する。

トークンに無い値が必要になったら、**それはトークンを増やすべきか、既存で足りるかを先に判断する。** 勝手に任意値を書かない。

### 状態は色だけで示さない

`ready` / `provisioning` / `failed` / `archived` は、色に加えて `[failed]` のようなテキストを必ず併記する。色覚と、モノクロ印刷と、スクリーンリーダーの 3 つに同時に効く。

---

## 画面が必ず持つ 4 状態

すべての画面で以下を実装する。**忘れやすいので、実装前にこの 4 つを列挙してから書く。**

| | |
|---|---|
| loading | 取得中 |
| error | 失敗。**何が起きて次に何をするかを書く。謝らない** |
| empty | データが 0 件。**次の行動への誘導にする** |
| ready | 通常 |

`empty` は特に手を抜きやすい。「データがありません」で終わらせない。

---

## URL とルーティング

ルート定義は `App.tsx`、パス定数と組み立ては `config/url.ts` に集約する。画面の中にパス文字列を直接書かない。

```ts
export const appUrl = { brainList: "/", brainDetail: "/brains/:sourceID" }
export function brainUrl(sourceID: string): string
```

**URL に使う識別子は `source_id`。** 内部の ULID (`brains.id`) を URL に出さない。source id は不変の引用キーであり、人間が読める。GitHub がリポジトリ名で辿れるのと同じ。

戻る・進む・リロードで壊れないこと。ログインが必要な画面は `?next=` で元の位置へ戻す。

---

## 品質の床

宣言せずに満たす。

- モバイルまでレスポンシブ
- キーボードのフォーカスが目視できる
- `prefers-reduced-motion` を尊重
- 押せる要素は `button` か `a`。`div` に `onClick` を付けない
- 無効な行はリンク要素にせず、`aria-disabled` を付ける
- フォームは `label` と入力を関連付ける
- コントラスト比 4.5:1 以上（本文）

---

## やらないこと

- `localStorage` / `sessionStorage` への認証情報の保存
- `div` に `onClick`
- コンポーネント内での API 呼び出し
- API レスポンス型の推測
- `window.location` からの MCP URL 組み立て
- トークン外の hex / px / font-family
- 使う場所が 1 つしかない共通コンポーネント
- 既存機能で足りるライブラリの追加
- 意味のないスクロール連動アニメーション

ライブラリを足したくなったら、**それが解く問題を 1 文で書いてから提案する。** 「あった方が便利」は理由にならない。

---

## 実装の進め方

1. 既存の似た画面を探す
   - 配置や分割で迷ったら `/home/futur/dev/xroll-frontend` の同種機能を見る
2. **動いている API を叩いてレスポンスを確認する**
3. `entities/<domain>/entity.ts` に型を書く
4. `lib/*Api.ts` に呼び出し関数を追加する
5. `components/pages` に薄い入口を追加する
6. `components/features` に Container / Presenter を追加する
7. 2 機能以上で使う要素が出たら `components/ui` へ切り出す
8. `config/url.ts` と `App.tsx` にルートを追加する

実装後:

```bash
npm --prefix web run build
npm --prefix web run check
npm --prefix web test
```

- 型エラーがないこと
- コンソールに警告が出ていないこと
- 4 状態すべてを実際に表示して確認したこと
- 新しいライブラリを足していないこと

---

## 変更時のチェック

- API のレスポンス形状を推測していないか
- コンポーネントが通信していないか
- トークン外の値を書いていないか
- 新しい共通コンポーネントが実際に 2 箇所以上で使われているか
- `empty` と `error` を実装したか
- URL に ULID を出していないか
