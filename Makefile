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

# -----------------------------------------------------------------------------
# Local Development Targets
# -----------------------------------------------------------------------------
.PHONY: build generate dev-build migrate-new

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

# Utility: Create New Migration
# Usage: make migrate-new NAME=add_users_table
migrate-new:
	@if [ -z "$(NAME)" ]; then echo "Usage: make migrate-new NAME=description"; exit 1; fi
	go run github.com/pressly/goose/v3/cmd/goose -dir db/migrations create $(NAME) sql

# -----------------------------------------------------------------------------
# Code Quality Targets
# -----------------------------------------------------------------------------
.PHONY: fmt vet lint check-roles test vuln check cover cover-html

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
check: fmt vet lint check-roles test
