# 2026-05-05: テストカバレッジ計測の導入

## Why

テストの **量** ではなく **時系列の変化** を観測したかった:

- 機能追加した際にテストが書かれているか / 削られていないか
- 派生プロジェクトでも初期からカバレッジを把握できる土台が欲しい

しきい値強制 (例: 「85% 以下なら CI 失敗」) は **過剰** と判断:
- 個人開発・テンプレ用途では人間関係よりも数字を上げる動機が弱い
- カバレッジは「テストがある」ことを示すだけで「テストが意味ある」を保証しない
- ノイズと無意味なテスト増加を招く

→ **計測 + 表示のみ** が現実的な落とし所。

## What

変更ファイル:
- `Makefile` (`cover` `cover-html` ターゲット追加)
- `.github/workflows/ci.yml` (Test ステップにカバレッジ計測 + Job Summary 表示)
- `.gitignore` (`coverage.out` を除外)

## How

### `Makefile`

```makefile
.PHONY: ... cover cover-html

# Run tests with coverage and show overall total
# 関数単位の詳細は cover-html で確認する
cover:
	@echo ">> Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | tail -1

# Open HTML coverage report in browser (depends on cover)
cover-html: cover
	@echo ">> Opening HTML coverage report..."
	go tool cover -html=coverage.out
```

意図:
- `make cover`: 数字を 1 行だけ出す（CI/手動どちらでも軽い）
- `make cover-html`: ブラウザで関数単位・行単位の赤緑表示

### `.github/workflows/ci.yml`

```yaml
- name: Test (with coverage)
  run: go test -coverprofile=coverage.out ./...

- name: Coverage summary
  if: always()
  run: |
    if [ -f coverage.out ]; then
      echo "## Test Coverage" >> $GITHUB_STEP_SUMMARY
      echo '```' >> $GITHUB_STEP_SUMMARY
      go tool cover -func=coverage.out | tail -1 >> $GITHUB_STEP_SUMMARY
      echo '```' >> $GITHUB_STEP_SUMMARY
      echo '<details><summary>Per-package detail</summary>' >> $GITHUB_STEP_SUMMARY
      echo '' >> $GITHUB_STEP_SUMMARY
      echo '```' >> $GITHUB_STEP_SUMMARY
      go tool cover -func=coverage.out >> $GITHUB_STEP_SUMMARY
      echo '```' >> $GITHUB_STEP_SUMMARY
      echo '</details>' >> $GITHUB_STEP_SUMMARY
    fi
```

意図:
- `if: always()` で **テストが失敗してもカバレッジを表示** する
- `Job Summary` に総カバレッジを大きく出し、関数別詳細は `<details>` で隠す
- Codecov 等の外部サービスは導入せず、GitHub Actions の標準機能のみで完結

### `.gitignore`

```
# Coverage output
coverage.out
```

## 数字の現実

導入時点 (テンプレリポ) のカバレッジは **2.6%**。低いように見えるが:

- `web/components/*_templ.go`, `web/layouts/*_templ.go` のような **生成ファイル** は実装側でカバーする意味が薄い
- `internal/database/*.go` (sqlc 自動生成) も同様
- 実質的にカバーすべきは `internal/handlers/`, `internal/middleware/`, `internal/routes/` あたり

**業務クリティカルな部分のカバレッジ** が分かれば十分。総計の数字を上げる目的でテストを増やすのは本末転倒。

派生プロジェクトでは `internal/handlers/business_*.go` など独自ロジックのカバレッジを意識的に上げると ROI が高い。

## 派生プロジェクトへの適用

派生プロジェクトの Claude Code に投げるプロンプト例:

```
テンプレリポの docs/migrations/2026-05-05-test-coverage.md を参照して、
このプロジェクトにテストカバレッジ計測を導入してください。
```

しきい値強制は不要。数字を見て、必要なら手動で `make cover-html` で追跡。

## 想定外の追加: 業務ドメインに合わせて対象を絞りたい場合

生成物のノイズが気になる場合、`-coverpkg` で対象を絞る:

```makefile
cover:
	go test -coverpkg=./internal/handlers/...,./internal/middleware/...,./internal/routes/... \
	       -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | tail -1
```

ただしこの調整は **派生プロジェクトの構造次第**。テンプレでは汎用的な `./...` のまま残す。

## 検証

- `make cover` でローカルで総カバレッジが出る
- `make cover-html` でブラウザに HTML レポートが開く
- CI 緑のままで、PR / push の Job Summary に Coverage セクションが表示される

## 関連コミット

- 後日付与
