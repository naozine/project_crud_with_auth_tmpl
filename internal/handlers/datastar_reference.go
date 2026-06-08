package handlers

import (
	"net/http"

	"github.com/naozine/project_crud_with_auth_tmpl/web/components"
)

// DatastarReferencePage は Datastar の主要フロントエンド機能を一覧する
// リファレンス兼回帰確認ページを描画する。認証不要だが、ルート登録自体を
// APP_ENV=dev でガードしているため本番では配信されない（cmd/server/main.go 参照）。
func DatastarReferencePage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	_ = components.DatastarReference().Render(r.Context(), w)
}
