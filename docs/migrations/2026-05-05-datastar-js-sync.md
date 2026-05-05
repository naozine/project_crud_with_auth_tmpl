# 2026-05-05: Datastar JS のセルフホスト版を Go SDK と同期更新

## Why

Datastar の SDK は **2 系統** に分かれており、それぞれが独立してバージョニングされている:

| 種別 | 場所 | 自動更新 |
|---|---|---|
| Go SDK (`datastar-go`) | `go.mod` 経由 | ✅ dependabot で自動更新される |
| **JS 本体 (`datastar.js`)** | `web/static/js/datastar.js` (vendor 化) | ❌ **手動更新が必要** |

dependabot で `datastar-go` だけが上がると、**Go SDK と JS のバージョン乖離** が起きる。SSE プロトコルの細部やイベント名が変わると、サーバー → ブラウザ間で機能不整合になる。

実際: dependabot が `datastar-go 1.1.0 → 1.2.1` を merge した時点で、JS 側は `v1.0.0-RC.8` (Release Candidate) のままだった。

## What

更新ファイル:
- `web/static/js/datastar.js`

公式の最新 JS バンドル (`v1.0.1` 等) で上書きする。Go SDK のバージョンと完全一致する必要はないが、メジャーが揃っていれば実用上問題なし (Datastar の SSE プロトコルは安定している)。

## How

### 現在のバージョン確認

```bash
head -1 web/static/js/datastar.js
# 例: // Datastar v1.0.0-RC.8
```

### 最新バージョンの確認

```bash
curl -s https://api.github.com/repos/starfederation/datastar/releases/latest \
  | python3 -c "import json,sys;print(json.load(sys.stdin)['tag_name'])"
# 例: v1.0.1
```

### 更新

公式リポジトリの `bundles/` ディレクトリから raw ファイルをダウンロード:

```bash
LATEST_TAG=$(curl -s https://api.github.com/repos/starfederation/datastar/releases/latest \
  | python3 -c "import json,sys;print(json.load(sys.stdin)['tag_name'])")
curl -sL -o web/static/js/datastar.js \
  "https://raw.githubusercontent.com/starfederation/datastar/$LATEST_TAG/bundles/datastar.js"
head -1 web/static/js/datastar.js  # バージョン確認
```

### 動作確認

ローカルで `air` 起動 → ブラウザで以下を試す:
- ログイン
- データ表示・追加・編集・削除 (SSE モーダルが正しく開閉)
- ブラウザコンソールにエラーが出ていない

問題なければ commit:

```bash
git add web/static/js/datastar.js
git commit -m "Datastar JS を vX.Y.Z から vA.B.C に更新"
```

問題があれば巻き戻し:

```bash
git checkout HEAD -- web/static/js/datastar.js
```

## 派生プロジェクトへの適用

派生プロジェクトでも `web/static/js/datastar.js` をセルフホストしている場合、定期的にこの更新を行う。Go SDK がメジャー/マイナーで上がった時に **必ずペアで更新** するルールにすると安全。

派生プロジェクトの Claude Code に投げるプロンプト例:

```
テンプレリポの docs/migrations/2026-05-05-datastar-js-sync.md を参照して、
web/static/js/datastar.js を最新版に更新してください。
更新後は air で動作確認してから commit します。
```

## 構造的な改善案 (将来検討)

毎回手動で同期するより、以下の選択肢がある:

### A. CI で同期チェック
- `datastar-go` の go.mod バージョンと `datastar.js` ファイル冒頭のコメントを突合
- 乖離していたら CI を warn or fail させる

### B. Makefile に `make update-datastar-js` ターゲット
- 上記の curl 手順をコマンド化
- 派生プロジェクトでも `make update-datastar-js` で済む

### C. CDN 経由に戻す
- セルフホストのメリット (オフライン動作、CSP 厳格化) が要らないなら CDN で常に最新を取る
- ただし Datastar の vendor 化コミット (`56db7d7`) は VPS デプロイの権限問題回避が動機なので、戻すなら再度問題が起きないか要確認

現時点では **手動更新 + このハウツー参照** で運用。

## 関連コミット

- `91409d1` Datastar JS を v1.0.0-RC.8 から v1.0.1 に更新
- `56db7d7` Datastar JS を vendor 化（参考、過去のコミット）
