# =============================================================================
# Dockerfile - マルチステージビルド
# =============================================================================
# VPS (docker-deploy) と fly.io (fly-deploy) の両方で使用
# SQLite は modernc.org/sqlite (Pure Go) を使用
# =============================================================================

# -----------------------------------------------------------------------------
# Stage 1: Build
# -----------------------------------------------------------------------------
FROM golang:1.25-alpine AS builder

# ビルドに必要なツールをインストール
RUN apk add --no-cache git

WORKDIR /app

# 依存関係を先にコピーしてキャッシュを活用
COPY go.mod go.sum ./
RUN go mod download

# ソースコードをコピー
COPY . .

# バージョン情報とプロジェクト名を埋め込んでビルド
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown
ARG PROJECT_NAME=app

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags "-X 'github.com/naozine/project_crud_with_auth_tmpl/internal/version.Version=${VERSION}' \
              -X 'github.com/naozine/project_crud_with_auth_tmpl/internal/version.Commit=${COMMIT}' \
              -X 'github.com/naozine/project_crud_with_auth_tmpl/internal/version.BuildDate=${BUILD_DATE}' \
              -X 'github.com/naozine/project_crud_with_auth_tmpl/internal/version.ProjectName=${PROJECT_NAME}'" \
    -o /app/server ./cmd/server

# -----------------------------------------------------------------------------
# Stage 2: Runtime
# -----------------------------------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

# バイナリをコピー
COPY --from=builder /app/server /app/server

# 静的ファイルをコピー
COPY --from=builder /app/web/static /app/web/static

# データディレクトリ (SQLite DB用) - ボリュームマウント先
VOLUME ["/app/data"]

# 環境変数のデフォルト値
ENV PORT=8080

# ポートを公開
EXPOSE 8080

# 実行
ENTRYPOINT ["/app/server"]
