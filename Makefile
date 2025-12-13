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

# Version Info (git から自動取得)
VERSION    := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT     := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS    := -X 'github.com/naozine/project_crud_with_auth_tmpl/internal/version.Version=$(VERSION)' \
              -X 'github.com/naozine/project_crud_with_auth_tmpl/internal/version.Commit=$(COMMIT)' \
              -X 'github.com/naozine/project_crud_with_auth_tmpl/internal/version.BuildDate=$(BUILD_DATE)'

# VPS Connection Info (deploy.config または環境変数で上書き可能)
VPS_USER    ?= user
VPS_HOST    ?= 192.168.1.100
SSH_PORT    ?= 22
VPS_DIR     ?= /var/www/project_crud_with_auth_tmpl
SERVICE_NAME ?= my-app.service
APP_PORT     ?= 8080
ADMIN_EMAIL  ?=
ADMIN_NAME   ?= Admin

# Docker Settings
IMAGE_NAME    ?= project-crud-auth
IMAGE_TAG     ?= $(VERSION)
APP_NAME      ?= project-crud-auth

# Caddy Settings (共有リバースプロキシ)
PUBLIC_HOST   ?= localhost
CADDY_DIR     ?= /home/$(VPS_USER)/caddy
CADDY_NETWORK ?= caddy-net

# Server Configuration
# PUBLIC_HOST が設定されていれば https:// を付けて自動生成
# バイナリ直接デプロイ (make deploy) では SERVER_ADDR を直接指定も可
SERVER_ADDR   ?= https://$(PUBLIC_HOST)

# -----------------------------------------------------------------------------
# Targets
# -----------------------------------------------------------------------------
.PHONY: all build build-linux deploy sync restart logs clean generate \
        docker-build docker-push docker-up docker-down docker-logs docker-dev \
        caddy-setup caddy-status docker-deploy docker-restart docker-remote-logs \
        fly-deploy fly-logs fly-status

all: build-linux

# Generate all auto-generated code (sqlc, templ)
generate:
	@echo ">> Generating code..."
	sqlc generate
	templ generate

# 0. Local build (開発・テスト用)
build: generate
	@echo ">> Building $(VERSION)..."
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)

# 1. Cross-compile for Linux (amd64) using Pure Go (no CGO required)
#    modernc.org/sqlite (Pure Go) を使用しているため、Docker不要でクロスコンパイル可能です。
build-linux: generate
	@echo ">> Building $(VERSION) for Linux/amd64..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -v -o $(BUILD_DIR)/$(BINARY_NAME)-linux $(CMD_PATH)

# 2. Deploy: Build -> Push Binary -> Push Service Config -> Restart Service
deploy: build-linux
	@echo ">> Deploying to $(VPS_HOST) (Port: $(SSH_PORT))..."

	# 1. リモートディレクトリ構造を作成
	ssh -p $(SSH_PORT) $(VPS_USER)@$(VPS_HOST) "mkdir -p $(VPS_DIR)/web/static && mkdir -p ~/.config/systemd/user"

	# 2. ローカルで一時的なサービスファイルを作成
	@echo "[Unit]\nDescription=$(SERVICE_NAME)\nAfter=network.target\n\n[Service]\nWorkingDirectory=$(VPS_DIR)\nExecStart=$(VPS_DIR)/$(BINARY_NAME)\n\
	Environment=\"PORT=$(APP_PORT)\"\n\
	Environment=\"ADMIN_EMAIL=$(ADMIN_EMAIL)\"\n\
	Environment=\"ADMIN_NAME=$(ADMIN_NAME)\"\n\
	Environment=\"SERVER_ADDR=$(SERVER_ADDR)\"\n\
	SyslogIdentifier=$(SERVICE_NAME)\n\
	Restart=always\nRestartSec=5\nStandardOutput=journal\nStandardError=journal\n\n[Install]\nWantedBy=default.target" > $(BINARY_NAME).service

	# 3. バイナリとサービスファイルを転送
	rsync -avz -e "ssh -p $(SSH_PORT)" --progress $(BUILD_DIR)/$(BINARY_NAME)-linux $(VPS_USER)@$(VPS_HOST):$(VPS_DIR)/$(BINARY_NAME)
	rsync -avz -e "ssh -p $(SSH_PORT)" $(BINARY_NAME).service $(VPS_USER)@$(VPS_HOST):~/.config/systemd/user/$(SERVICE_NAME)
	
	# 4. 静的ファイルを転送
	rsync -avz -e "ssh -p $(SSH_PORT)" --delete web/static/ $(VPS_USER)@$(VPS_HOST):$(VPS_DIR)/web/static/

	# 5. サービス登録・有効化・再起動・永続化
	ssh -p $(SSH_PORT) $(VPS_USER)@$(VPS_HOST) "\
		loginctl enable-linger $(VPS_USER) && \
		systemctl --user daemon-reload && \
		systemctl --user enable $(SERVICE_NAME) && \
		systemctl --user restart $(SERVICE_NAME)"
	
	# 6. ローカルの一時ファイルを削除
	@rm $(BINARY_NAME).service
	@echo ">> Deployment complete!"

# (Option) Sync Source Code Only (開発用: サーバー上でビルドする場合に使用)
sync:
	@echo ">> Syncing source code to VPS (Port: $(SSH_PORT))..."
	rsync -avz -e "ssh -p $(SSH_PORT)" --delete \
		--exclude '.git' \
		--exclude '.idea' \
		--exclude 'bin/' \
		--exclude 'tmp/' \
		--exclude '*.db*' \
		--exclude '.env' \
		--exclude 'deploy.config' \
		./ $(VPS_USER)@$(VPS_HOST):$(VPS_DIR)

# Utility: Create New Migration
# Usage: make migrate-new NAME=add_users_table
migrate-new:
	@if [ -z "$(NAME)" ]; then echo "Usage: make migrate-new NAME=description"; exit 1; fi
	go run github.com/pressly/goose/v3/cmd/goose -dir db/migrations create $(NAME) sql

# Utility: Restart Service
restart:
	ssh -p $(SSH_PORT) $(VPS_USER)@$(VPS_HOST) "systemctl --user restart $(SERVICE_NAME)"

# Utility: View Logs
logs:
	ssh -p $(SSH_PORT) $(VPS_USER)@$(VPS_HOST) "journalctl --user -u $(SERVICE_NAME) -f"

clean:
	rm -f $(BUILD_DIR)/$(BINARY_NAME)-linux

# -----------------------------------------------------------------------------
# Docker Targets
# -----------------------------------------------------------------------------

# Build Docker image for linux/amd64 (VPS deployment)
# M4 Mac (arm64) から x86 VPS へのデプロイに対応
# 方法: VPS上でビルドするか、マルチアーキテクチャ対応の場合は buildx を使用
docker-build: generate
	@echo ">> Building Docker image $(IMAGE_NAME):$(IMAGE_TAG)..."
	@if docker buildx version >/dev/null 2>&1; then \
		echo "Using buildx for cross-platform build (linux/amd64)..."; \
		docker buildx build \
			--platform linux/amd64 \
			--build-arg VERSION=$(VERSION) \
			--build-arg COMMIT=$(COMMIT) \
			--build-arg BUILD_DATE=$(BUILD_DATE) \
			-t $(IMAGE_NAME):$(IMAGE_TAG) \
			-t $(IMAGE_NAME):latest \
			--load \
			.; \
	else \
		echo "buildx not available, building for local architecture..."; \
		echo "Note: VPS上でイメージをビルドするか、buildxをインストールしてください"; \
		docker build \
			--build-arg VERSION=$(VERSION) \
			--build-arg COMMIT=$(COMMIT) \
			--build-arg BUILD_DATE=$(BUILD_DATE) \
			-t $(IMAGE_NAME):$(IMAGE_TAG) \
			-t $(IMAGE_NAME):latest \
			.; \
	fi

# Push Docker image to registry (要: DOCKER_REGISTRY 設定)
docker-push: docker-build
	@if [ -z "$(DOCKER_REGISTRY)" ]; then \
		echo "Error: DOCKER_REGISTRY is not set"; \
		exit 1; \
	fi
	@echo ">> Pushing to $(DOCKER_REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)..."
	docker tag $(IMAGE_NAME):$(IMAGE_TAG) $(DOCKER_REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)
	docker tag $(IMAGE_NAME):latest $(DOCKER_REGISTRY)/$(IMAGE_NAME):latest
	docker push $(DOCKER_REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)
	docker push $(DOCKER_REGISTRY)/$(IMAGE_NAME):latest

# Start production containers
docker-up:
	@echo ">> Starting containers..."
	docker compose up -d

# Stop production containers
docker-down:
	@echo ">> Stopping containers..."
	docker compose down

# View container logs
docker-logs:
	docker compose logs -f

# Start development environment with hot reload
docker-dev:
	@echo ">> Starting development environment..."
	docker compose -f docker-compose.dev.yaml up --build

# Stop development environment
docker-dev-down:
	docker compose -f docker-compose.dev.yaml down

# =============================================================================
# Caddy (共有リバースプロキシ) 関連
# =============================================================================

# Caddy 初回セットアップ（VPSに Caddy がなければ構築）
caddy-setup:
	@echo ">> Setting up Caddy on $(VPS_HOST)..."
	# 1. Caddy ディレクトリを作成
	ssh -p $(SSH_PORT) $(VPS_USER)@$(VPS_HOST) "mkdir -p $(CADDY_DIR)/sites"
	# 2. Caddy 設定ファイルを転送
	rsync -avz -e "ssh -p $(SSH_PORT)" caddy/docker-compose.yaml $(VPS_USER)@$(VPS_HOST):$(CADDY_DIR)/
	rsync -avz -e "ssh -p $(SSH_PORT)" caddy/Caddyfile $(VPS_USER)@$(VPS_HOST):$(CADDY_DIR)/
	# 3. Docker ネットワークを作成（存在しなければ）
	ssh -p $(SSH_PORT) $(VPS_USER)@$(VPS_HOST) "docker network create $(CADDY_NETWORK) 2>/dev/null || true"
	# 4. Caddy を起動
	ssh -p $(SSH_PORT) $(VPS_USER)@$(VPS_HOST) "cd $(CADDY_DIR) && docker compose up -d"
	@echo ">> Caddy setup complete!"

# Caddy の状態確認
caddy-status:
	@echo ">> Caddy status on $(VPS_HOST)..."
	ssh -p $(SSH_PORT) $(VPS_USER)@$(VPS_HOST) "cd $(CADDY_DIR) && docker compose ps"

# Caddy のログ確認
caddy-logs:
	ssh -p $(SSH_PORT) $(VPS_USER)@$(VPS_HOST) "cd $(CADDY_DIR) && docker compose logs -f"

# Caddy をリロード（設定変更を反映）
caddy-reload:
	ssh -p $(SSH_PORT) $(VPS_USER)@$(VPS_HOST) "docker exec caddy caddy reload --config /etc/caddy/Caddyfile"

# =============================================================================
# Docker Deploy (VPS上でDockerビルド・実行 + Caddy 設定)
# =============================================================================

# Deploy to VPS using Docker
# ソースを転送し、VPS上でDockerマルチステージビルド
docker-deploy: generate
	@echo ">> Deploying Docker to $(VPS_HOST)..."
	@echo ">> App: $(APP_NAME) -> $(PUBLIC_HOST)"
	# 1. Caddy が起動しているか確認（なければセットアップ）
	@ssh -p $(SSH_PORT) $(VPS_USER)@$(VPS_HOST) "docker ps -q -f name=caddy" | grep -q . || $(MAKE) caddy-setup
	# 2. ソースコードを転送（ホワイトリスト方式：必要なものだけ）
	ssh -p $(SSH_PORT) $(VPS_USER)@$(VPS_HOST) "mkdir -p $(VPS_DIR)"
	rsync -avz -e "ssh -p $(SSH_PORT)" --delete \
		--include='go.mod' \
		--include='go.sum' \
		--include='Dockerfile' \
		--include='.dockerignore' \
		--include='docker-compose.yaml' \
		--include='cmd/***' \
		--include='internal/***' \
		--include='db/***' \
		--include='web/***' \
		--exclude='*' \
		./ $(VPS_USER)@$(VPS_HOST):$(VPS_DIR)/
	# 3. 本番用 .env を転送（APP_NAME, SERVER_ADDR, WebAuthn設定を追加）
	@if [ -f ".env.production" ]; then \
		echo "APP_NAME=$(APP_NAME)" > /tmp/.env.deploy && \
		echo "SERVER_ADDR=$(SERVER_ADDR)" >> /tmp/.env.deploy && \
		echo "WEBAUTHN_RP_ID=$(PUBLIC_HOST)" >> /tmp/.env.deploy && \
		echo "WEBAUTHN_ALLOWED_ORIGINS=$(SERVER_ADDR)" >> /tmp/.env.deploy && \
		cat .env.production >> /tmp/.env.deploy && \
		rsync -avz -e "ssh -p $(SSH_PORT)" /tmp/.env.deploy $(VPS_USER)@$(VPS_HOST):$(VPS_DIR)/.env && \
		rm /tmp/.env.deploy; \
	else \
		echo "APP_NAME=$(APP_NAME)\nSERVER_ADDR=$(SERVER_ADDR)\nWEBAUTHN_RP_ID=$(PUBLIC_HOST)\nWEBAUTHN_ALLOWED_ORIGINS=$(SERVER_ADDR)" | \
		ssh -p $(SSH_PORT) $(VPS_USER)@$(VPS_HOST) "cat > $(VPS_DIR)/.env"; \
	fi
	# 4. Docker イメージをビルド（マルチステージビルド）
	@echo ">> Building Docker image on VPS..."
	ssh -p $(SSH_PORT) $(VPS_USER)@$(VPS_HOST) "cd $(VPS_DIR) && \
		docker build \
			--build-arg VERSION=$(VERSION) \
			--build-arg COMMIT=$(COMMIT) \
			--build-arg BUILD_DATE=$(BUILD_DATE) \
			-t $(APP_NAME):latest ."
	# 5. データディレクトリの作成と権限設定（nonroot ユーザー用）
	@echo ">> Setting data directory permissions..."
	ssh -p $(SSH_PORT) $(VPS_USER)@$(VPS_HOST) "mkdir -p $(VPS_DIR)/data && chmod 777 $(VPS_DIR)/data"
	# 6. コンテナ起動（イメージが更新されたので再作成）
	@echo ">> Starting container..."
	ssh -p $(SSH_PORT) $(VPS_USER)@$(VPS_HOST) "cd $(VPS_DIR) && docker compose up -d --force-recreate"
	# 7. Caddy 設定を追加
	@echo ">> Configuring Caddy for $(PUBLIC_HOST)..."
	@echo '$(PUBLIC_HOST) {\n    reverse_proxy $(APP_NAME):8080\n}' | \
		ssh -p $(SSH_PORT) $(VPS_USER)@$(VPS_HOST) "cat > $(CADDY_DIR)/sites/$(APP_NAME).caddy"
	# 8. Caddy をリロード
	ssh -p $(SSH_PORT) $(VPS_USER)@$(VPS_HOST) "docker exec caddy caddy reload --config /etc/caddy/Caddyfile"
	@echo ">> Docker deployment complete!"
	@echo ">> Access: https://$(PUBLIC_HOST)"

# Restart containers on VPS
docker-restart:
	ssh -p $(SSH_PORT) $(VPS_USER)@$(VPS_HOST) "cd $(VPS_DIR) && docker compose restart"

# View container logs on VPS
docker-remote-logs:
	ssh -p $(SSH_PORT) $(VPS_USER)@$(VPS_HOST) "cd $(VPS_DIR) && docker compose logs -f"

# =============================================================================
# fly.io Deploy
# =============================================================================

# fly.io へデプロイ
# 事前準備: fly.toml.example を fly.toml にコピーし、app 名を設定
fly-deploy: generate
	@echo ">> Deploying to fly.io..."
	@if [ ! -f "fly.toml" ]; then \
		echo "Error: fly.toml not found. Copy fly.toml.example to fly.toml and configure it."; \
		exit 1; \
	fi
	fly deploy --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) --build-arg BUILD_DATE=$(BUILD_DATE)

# fly.io のログを表示
fly-logs:
	fly logs

# fly.io のステータス確認
fly-status:
	fly status
