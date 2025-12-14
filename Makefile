# -----------------------------------------------------------------------------
# Configuration
# -----------------------------------------------------------------------------
# プロジェクト固有の設定を読み込む（存在する場合）
-include deploy.config

BUILD_DIR   ?= bin
CMD_PATH    ?= ./cmd/server

# Project Name (フォルダ名から自動取得、_ は - に変換)
# Docker イメージ名、コンテナ名、fly.io app名、クッキー名などに使用
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

# VPS Connection Info (deploy.config または環境変数で上書き可能)
VPS_USER     ?= user
VPS_HOST     ?= 192.168.1.100
SSH_PORT     ?= 22
VPS_BASE_DIR ?= /var/www

# Docker Settings (デフォルトはフォルダ名から自動生成)
IMAGE_NAME    ?= $(PROJECT_NAME)
IMAGE_TAG     ?= $(VERSION)
APP_NAME      ?= $(PROJECT_NAME)

# Caddy Settings (共有リバースプロキシ)
PUBLIC_HOST   ?= localhost
CADDY_DIR     ?= /home/$(VPS_USER)/caddy
CADDY_NETWORK ?= caddy-net

# Server Configuration（ローカル開発時は http://localhost:8080）
SERVER_ADDR   ?= http://localhost:8080

# VPS上の実際のデプロイパス
VPS_DEPLOY_DIR := $(VPS_BASE_DIR)/$(PROJECT_NAME)

# -----------------------------------------------------------------------------
# Targets
# -----------------------------------------------------------------------------
.PHONY: build generate migrate-new \
        docker-build docker-push docker-up docker-down docker-logs docker-dev \
        caddy-setup caddy-status caddy-reload docker-deploy docker-restart docker-remote-logs \
        dns-setup fly-deploy fly-logs fly-status

# Generate all auto-generated code (sqlc, templ)
generate:
	@echo ">> Generating code..."
	sqlc generate
	templ generate

# Local build (開発・テスト用)
build: generate
	@echo ">> Building $(VERSION)..."
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/server $(CMD_PATH)

# Utility: Create New Migration
# Usage: make migrate-new NAME=add_users_table
migrate-new:
	@if [ -z "$(NAME)" ]; then echo "Usage: make migrate-new NAME=description"; exit 1; fi
	go run github.com/pressly/goose/v3/cmd/goose -dir db/migrations create $(NAME) sql

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
			--build-arg PROJECT_NAME=$(PROJECT_NAME) \
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
			--build-arg PROJECT_NAME=$(PROJECT_NAME) \
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
# Cloudflare DNS 設定
# =============================================================================

# DNS レコードを設定（Cloudflare）
# 必要な環境変数: CF_API_TOKEN, CF_ZONE_ID, VPS_IP
dns-setup:
	@if [ -z "$(CF_API_TOKEN)" ] || [ -z "$(CF_ZONE_ID)" ] || [ -z "$(VPS_IP)" ]; then \
		echo "Error: CF_API_TOKEN, CF_ZONE_ID, VPS_IP を deploy.config に設定してください"; \
		exit 1; \
	fi
	@echo ">> Setting up DNS for $(PUBLIC_HOST)..."
	@curl -s -X POST "https://api.cloudflare.com/client/v4/zones/$(CF_ZONE_ID)/dns_records" \
		-H "Authorization: Bearer $(CF_API_TOKEN)" \
		-H "Content-Type: application/json" \
		--data '{"type":"A","name":"$(PROJECT_NAME)","content":"$(VPS_IP)","proxied":true,"ttl":1}' \
		| jq -r 'if .success then "DNS record created: \(.result.name)" else "Error: \(.errors[0].message)" end'
	@echo ""
	@echo ">> DNS の伝播を待ってから make docker-deploy を実行してください（1-2分程度）"
	@echo ">> 確認: dig $(PUBLIC_HOST) で $(VPS_IP) ではなく Cloudflare の IP が返れば OK"

# =============================================================================
# Docker Deploy (VPS上でDockerビルド・実行 + Caddy 設定)
# =============================================================================

# Deploy to VPS using Docker
# ソースを転送し、VPS上でDockerマルチステージビルド
docker-deploy: generate
	@echo ">> Deploying Docker to $(VPS_HOST)..."
	@echo ">> App: $(APP_NAME) -> $(PUBLIC_HOST)"
	@echo ">> Deploy path: $(VPS_DEPLOY_DIR)"
	# 1. Caddy が起動しているか確認（なければセットアップ）
	@ssh -p $(SSH_PORT) $(VPS_USER)@$(VPS_HOST) "docker ps -q -f name=caddy" | grep -q . || $(MAKE) caddy-setup
	# 2. ソースコードを転送（ホワイトリスト方式：必要なものだけ）
	ssh -p $(SSH_PORT) $(VPS_USER)@$(VPS_HOST) "mkdir -p $(VPS_DEPLOY_DIR)"
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
		./ $(VPS_USER)@$(VPS_HOST):$(VPS_DEPLOY_DIR)/
	# 3. 本番用 .env を転送（SMTP設定など。SERVER_ADDR, WEBAUTHN_* はビルド時注入）
	@if [ -f ".env.production" ]; then \
		rsync -avz -e "ssh -p $(SSH_PORT)" .env.production $(VPS_USER)@$(VPS_HOST):$(VPS_DEPLOY_DIR)/.env; \
	else \
		ssh -p $(SSH_PORT) $(VPS_USER)@$(VPS_HOST) "touch $(VPS_DEPLOY_DIR)/.env"; \
	fi
	# 4. Docker イメージをビルド（マルチステージビルド）
	@echo ">> Building Docker image on VPS..."
	ssh -p $(SSH_PORT) $(VPS_USER)@$(VPS_HOST) "cd $(VPS_DEPLOY_DIR) && \
		docker build \
			--build-arg VERSION=$(VERSION) \
			--build-arg COMMIT=$(COMMIT) \
			--build-arg BUILD_DATE=$(BUILD_DATE) \
			--build-arg PROJECT_NAME=$(PROJECT_NAME) \
			--build-arg SERVER_ADDR=$(SERVER_ADDR) \
			-t $(APP_NAME):latest ."
	# 5. データディレクトリの作成と権限設定（nonroot ユーザー用）
	@echo ">> Setting data directory permissions..."
	ssh -p $(SSH_PORT) $(VPS_USER)@$(VPS_HOST) "mkdir -p $(VPS_DEPLOY_DIR)/data && chmod 777 $(VPS_DEPLOY_DIR)/data"
	# 6. コンテナ起動（イメージが更新されたので再作成）
	@echo ">> Starting container..."
	ssh -p $(SSH_PORT) $(VPS_USER)@$(VPS_HOST) "cd $(VPS_DEPLOY_DIR) && APP_NAME=$(APP_NAME) docker compose up -d --force-recreate"
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
	ssh -p $(SSH_PORT) $(VPS_USER)@$(VPS_HOST) "cd $(VPS_DEPLOY_DIR) && docker compose restart"

# View container logs on VPS
docker-remote-logs:
	ssh -p $(SSH_PORT) $(VPS_USER)@$(VPS_HOST) "cd $(VPS_DEPLOY_DIR) && docker compose logs -f"

# =============================================================================
# fly.io Deploy
# =============================================================================

# fly.io へデプロイ
# アプリ名はフォルダ名 (PROJECT_NAME) を使用
# 初回は fly apps create $(PROJECT_NAME) でアプリを作成しておくこと
# SERVER_ADDR は https://$(PROJECT_NAME).fly.dev を使用
fly-deploy: generate
	@echo ">> Deploying to fly.io..."
	@echo ">> App: $(PROJECT_NAME)"
	fly deploy -a $(PROJECT_NAME) \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--build-arg PROJECT_NAME=$(PROJECT_NAME) \
		--build-arg SERVER_ADDR=https://$(PROJECT_NAME).fly.dev

# fly.io のログを表示
fly-logs:
	fly logs -a $(PROJECT_NAME)

# fly.io のステータス確認
fly-status:
	fly status -a $(PROJECT_NAME)
