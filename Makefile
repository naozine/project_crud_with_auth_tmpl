# -----------------------------------------------------------------------------
# Configuration
# -----------------------------------------------------------------------------
# プロジェクト固有の設定を読み込む（存在する場合）
-include deploy.config

BINARY_NAME ?= server
BUILD_DIR   ?= bin
CMD_PATH    ?= ./cmd/server
# プロジェクトの go.mod に合わせる
GO_VERSION  ?= 1.25

# VPS Connection Info (deploy.config または環境変数で上書き可能)
VPS_USER    ?= user
VPS_HOST    ?= 192.168.1.100
VPS_DIR     ?= /var/www/project_crud_with_auth_tmpl
SERVICE_NAME ?= my-app.service

# -----------------------------------------------------------------------------
# Targets
# -----------------------------------------------------------------------------
.PHONY: all build-linux deploy sync restart logs clean

all: build-linux

# 1. Cross-compile for Linux (amd64) using Pure Go (no CGO required)
#    modernc.org/sqlite (Pure Go) を使用しているため、Docker不要でクロスコンパイル可能です。
build-linux:
	@echo ">> Generating templ components..."
	templ generate
	@echo ">> Building binary for Linux/amd64 (Pure Go)..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o $(BUILD_DIR)/$(BINARY_NAME)-linux $(CMD_PATH)

# 2. Deploy: Build -> Push Binary -> Restart Service
deploy: build-linux
	@echo ">> Deploying to $(VPS_HOST)..."
	# バイナリを転送 (サーバー上では 'server' という名前で配置)
	rsync -avz --progress $(BUILD_DIR)/$(BINARY_NAME)-linux $(VPS_USER)@$(VPS_HOST):$(VPS_DIR)/$(BINARY_NAME)
	# 静的ファイル(CSS/JS/Images)を転送
	rsync -avz --delete web/static/ $(VPS_USER)@$(VPS_HOST):$(VPS_DIR)/web/static/
	# サービスの再起動 (ユーザーサービスとして再起動)
	ssh $(VPS_USER)@$(VPS_HOST) "systemctl --user restart $(SERVICE_NAME)"
	@echo ">> Deployment complete!"

# (Option) Sync Source Code Only (開発用: サーバー上でビルドする場合に使用)
sync:
	@echo ">> Syncing source code to VPS..."
	rsync -avz --delete \
		--exclude '.git' \
		--exclude '.idea' \
		--exclude 'bin/' \
		--exclude 'tmp/' \
		--exclude '*.db*' \
		--exclude '.env' \
		--exclude 'deploy.config' \
		./ $(VPS_USER)@$(VPS_HOST):$(VPS_DIR)

# Utility: Restart Service
restart:
	ssh $(VPS_USER)@$(VPS_HOST) "systemctl --user restart $(SERVICE_NAME)"

# Utility: View Logs
logs:
	ssh $(VPS_USER)@$(VPS_HOST) "journalctl --user -u $(SERVICE_NAME) -f"

clean:
	rm -f $(BUILD_DIR)/$(BINARY_NAME)-linux
