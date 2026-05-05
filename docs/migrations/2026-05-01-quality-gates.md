# 2026-05-01: 品質ゲート (lint / CI / dependabot) の導入

## Why

- リポジトリに `Makefile` の lint / vet / test ターゲットも CI も無く、コードが静かに腐っていた（実例: Echo→chi 移行で integration テストがコンパイルできない状態のまま 2 ヶ月放置）
- 派生プロジェクトでも同じ事が起きるリスクがあるため、テンプレ側で恒常的なガードを入れる

## What

新規ファイル:
- `.golangci.yml` (golangci-lint 設定、v2 形式)
- `Makefile` の `fmt` `vet` `lint` `test` `vuln` `check` ターゲット
- `.github/workflows/ci.yml` (build / test / lint / govulncheck の最小 CI)
- `.github/dependabot.yml` (Go モジュール + GitHub Actions の週次自動更新)
- `CLAUDE.md` の「作業完了時に `make vet` と `make lint` を実行」ルール

## How

### `.golangci.yml`

```yaml
version: "2"

run:
  timeout: 5m

linters:
  default: standard  # errcheck/govet/ineffassign/staticcheck/unused
  enable:
    - gosec
  settings:
    errcheck:
      exclude-functions:
        - (io.Closer).Close
        - (*encoding/json.Encoder).Encode
        - (github.com/a-h/templ.Component).Render
        - (*github.com/starfederation/datastar-go/datastar.ServerSentEventGenerator).ExecuteScript
    gosec:
      excludes:
        - G104  # errcheck と機能重複
        - G706  # log injection、env 変数由来で誤検知が多い

formatters:
  enable:
    - gofmt
    - goimports
```

### `Makefile` 追加分

```makefile
.PHONY: fmt vet lint test vuln check

fmt:
	gofmt -w .

vet:
	go vet ./...

lint:
	golangci-lint run ./...

test:
	go test ./...

vuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

check: fmt vet lint test
```

### `.github/workflows/ci.yml`

主要ポイント:
- `go-version: '1.26'` + `check-latest: true` で stdlib 脆弱性を自動回避（マイナーまで指定し、パッチは常に最新）
- `golangci/golangci-lint-action@v8` + `version: v2.11.4` (v6 以前は golangci-lint v1 用)
- `go mod tidy` の検証ステップで go.mod の汚れも検出

完全な内容は本リポジトリの `.github/workflows/ci.yml` を参照。

### `.github/dependabot.yml`

```yaml
version: 2
updates:
  - package-ecosystem: "gomod"
    directory: "/"
    schedule:
      interval: "weekly"
    open-pull-requests-limit: 5
    commit-message:
      prefix: "deps"

  - package-ecosystem: "github-actions"
    directory: "/"
    schedule:
      interval: "weekly"
    open-pull-requests-limit: 3
    commit-message:
      prefix: "ci"
```

### CLAUDE.md ルール追加

```markdown
## 作業完了時

- Go ファイルの変更を伴う作業の最後に `make vet` と `make lint` を実行し、結果を報告する
  - `make lint` で `golangci-lint: command not found` が出たらスキップして報告のみでよい
  - 既存コードに由来する警告（自分が変更していない箇所）は、その旨を明記して区別する
```

## 派生プロジェクトへの適用

派生プロジェクトの Claude Code に投げるプロンプト例:

```
テンプレリポの docs/migrations/2026-05-01-quality-gates.md を参照して、
このリポジトリにも品質ゲートと CI を導入してください。
```

ライブラリプロジェクトの場合は `cmd/` や `make generate` 関連が無いので、`Makefile` から `generate` ターゲットを除外するなどの調整が必要。

## 検証

- ローカル: `make check` が緑
- 初回 push 後の GitHub Actions が緑

### 想定される躓きポイント (姉妹リポでの試行錯誤からの学び)

1. **`golangci-lint-action` を `@v6` 以前にしない**: v1 系を引いてしまう
2. **`version: latest` を避ける**: latest が v1 系を返すことがある。`v2.x.y` を明示
3. **`go-version-file: go.mod` よりも `go-version: '1.26'` + `check-latest: true`**: 後者の方が脆弱性のあるパッチを引かない

## 関連コミット

- `51f1e4d` lint と CI 環境を導入
- `d30c419` lint の些細な警告を解消（既存コード由来の指摘の整理）
