# -----------------------------------------------------------------------------
# Configuration
# -----------------------------------------------------------------------------
BUILD_DIR   ?= bin
CMD_PATH    ?= ./cmd/server

# Project Name (フォルダ名から自動取得、_ は - に変換)
PROJECT_NAME := $(subst _,-,$(notdir $(CURDIR)))

# Version Info (git から自動取得)
VERSION    := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT     := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS     = -X 'github.com/naozine/project_crud_with_auth_tmpl/internal/version.Version=$(VERSION)' \
              -X 'github.com/naozine/project_crud_with_auth_tmpl/internal/version.Commit=$(COMMIT)' \
              -X 'github.com/naozine/project_crud_with_auth_tmpl/internal/version.BuildDate=$(BUILD_DATE)' \
              -X 'github.com/naozine/project_crud_with_auth_tmpl/internal/version.ProjectName=$(PROJECT_NAME)' \
              -X 'github.com/naozine/project_crud_with_auth_tmpl/internal/version.ServerAddr=$(SERVER_ADDR)'

# Server Configuration（ローカル開発時は http://localhost:8080）
SERVER_ADDR   ?= http://localhost:8080

# Tool Versions
# tailwindcss はバージョンで minify 出力が変わるため、ここを唯一の定義とする。
# CI (.github/workflows/ci.yml) と make install はこの値を参照する。
TAILWIND_VERSION := v4.2.1

# -----------------------------------------------------------------------------
# Local Development Targets
# -----------------------------------------------------------------------------
.PHONY: build generate dev-build migrate-new tailwind-version

# Generate all auto-generated code (sqlc, templ, tailwind)
generate:
	@echo ">> Generating code..."
	sqlc generate
	templ generate
	tailwindcss -i web/static/css/input.css -o web/static/css/style.css --minify

# Local build (開発・テスト用)
build: generate
	@echo ">> Building $(VERSION)..."
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/server $(CMD_PATH)

# Air 用ビルド (ホットリロード開発用)
# generate は Air の include_ext で監視しているため省略
dev-build:
	go build -ldflags "$(LDFLAGS)" -o ./tmp/main $(CMD_PATH)

# CI が tailwindcss のバージョンを取得するための出力専用ターゲット
tailwind-version:
	@echo $(TAILWIND_VERSION)

# Utility: Create New Migration
# Usage: make migrate-new NAME=add_users_table
migrate-new:
	@if [ -z "$(NAME)" ]; then echo "Usage: make migrate-new NAME=description"; exit 1; fi
	go run github.com/pressly/goose/v3/cmd/goose -dir db/migrations create $(NAME) sql

# -----------------------------------------------------------------------------
# Code Quality Targets
# -----------------------------------------------------------------------------
.PHONY: fmt vet lint check-roles check-env-docs test vuln check cover cover-html

# Format code
fmt:
	@echo ">> Formatting..."
	gofmt -w .

# Run go vet
vet:
	@echo ">> Running go vet..."
	go vet ./...

# Run golangci-lint
# 要: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
lint:
	@echo ">> Running golangci-lint..."
	golangci-lint run ./...

# Check that role strings are not hardcoded outside internal/roles
# ロール文字列 ("admin"/"editor"/"viewer") のベタ書きを検出する。
# プロダクションコードのみ対象（テスト・生成物・定義元は除外）。
check-roles:
	@echo ">> Checking for hardcoded role strings..."
	@if grep -rn '"admin"\|"editor"\|"viewer"' internal/ web/ cmd/ --include="*.go" --include="*.templ" \
		| grep -v '_templ.go' | grep -v '_test.go' \
		| grep -v 'internal/roles/' | grep -v 'internal/integration/'; then \
		echo "ERROR: ロール文字列はベタ書きせず internal/roles の定数を使ってください（テストは対象外）"; \
		exit 1; \
	fi

# Check that .env.example and the env vars read in code stay in sync (both directions)
# .env.example は「変数の意味・デフォルト値」の正。コードの os.Getenv("X") と突き合わせ、
# 記載漏れと、どこからも読まれない残骸の両方を検出する。
check-env-docs:
	@echo ">> Checking .env.example is in sync with os.Getenv usage..."
	@code_vars=$$(grep -rhoE 'os\.Getenv\("[A-Z0-9_]+"\)' cmd/ internal/ web/ --include='*.go' --exclude='*_test.go' --exclude='*_templ.go' | sed -E 's/.*"([A-Z0-9_]+)".*/\1/' | sort -u); \
	doc_vars=$$(grep -oE '^#? ?[A-Z0-9_]+=' .env.example | tr -d '# =' | sort -u); \
	ok=1; \
	for v in $$code_vars; do echo "$$doc_vars" | grep -qx "$$v" || { echo "ERROR: コードが読む $$v が .env.example に記載されていません"; ok=0; }; done; \
	for v in $$doc_vars; do echo "$$code_vars" | grep -qx "$$v" || { echo "ERROR: .env.example の $$v はコードのどこからも読まれていません（残骸？）"; ok=0; }; done; \
	[ $$ok -eq 1 ]

# Run tests
test:
	@echo ">> Running tests..."
	go test ./...

# Check known vulnerabilities (要ネットワーク)
vuln:
	@echo ">> Running govulncheck..."
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

# Run tests with coverage and show overall total
# -coverpkg=./... で integration テスト等が他パッケージを呼んだ行もカバレッジに含める。
# 関数単位の詳細は cover-html で確認する。
cover:
	@echo ">> Running tests with coverage..."
	go test -coverpkg=./... -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | tail -1

# Open HTML coverage report in browser (depends on cover)
cover-html: cover
	@echo ">> Opening HTML coverage report..."
	go tool cover -html=coverage.out

# Run all quality checks (lint は要 golangci-lint インストール)
check: fmt vet lint check-roles check-env-docs test
