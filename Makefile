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
SSH_PORT    ?= 22
VPS_DIR     ?= /var/www/project_crud_with_auth_tmpl
SERVICE_NAME ?= my-app.service
APP_PORT     ?= 8080

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

# 2. Deploy: Build -> Push Binary -> Push Service Config -> Restart Service
deploy: build-linux
	@echo ">> Deploying to $(VPS_HOST) (Port: $(SSH_PORT))..."
	
	# 1. リモートディレクトリ構造を作成
	ssh -p $(SSH_PORT) $(VPS_USER)@$(VPS_HOST) "mkdir -p $(VPS_DIR)/web/static && mkdir -p ~/.config/systemd/user"

	# 2. ローカルで一時的なサービスファイルを作成 (環境変数PORTを指定)
	@echo "[Unit]\nDescription=$(SERVICE_NAME)\nAfter=network.target\n\n[Service]\nWorkingDirectory=$(VPS_DIR)\nExecStart=$(VPS_DIR)/$(BINARY_NAME)\nEnvironment=\"PORT=$(APP_PORT)\"\nRestart=always\nRestartSec=5\nStandardOutput=journal\nStandardError=journal\n\n[Install]\nWantedBy=default.target" > $(BINARY_NAME).service

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

# Utility: Restart Service
restart:
	ssh -p $(SSH_PORT) $(VPS_USER)@$(VPS_HOST) "systemctl --user restart $(SERVICE_NAME)"

# Utility: View Logs
logs:
	ssh -p $(SSH_PORT) $(VPS_USER)@$(VPS_HOST) "journalctl --user -u $(SERVICE_NAME) -f"

clean:
	rm -f $(BUILD_DIR)/$(BINARY_NAME)-linux
