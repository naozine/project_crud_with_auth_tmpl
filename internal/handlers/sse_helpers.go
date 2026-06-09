package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/starfederation/datastar-go/datastar"

	"github.com/naozine/project_crud_with_auth_tmpl/web/components"
)

// newSSE は ResponseWriter と Request から SSE ジェネレーターを作成する。
func newSSE(w http.ResponseWriter, r *http.Request) *datastar.ServerSentEventGenerator {
	return datastar.NewSSE(w, r)
}

// readSignalsOr413 は Datastar signals を読み込む。body 上限超過なら 413、
// その他のパースエラーなら 400 を返して false を返す（呼び出し元は return すること）。
// 上限は SSE ルートの MaxBodySize ミドルウェアで設定する。
func readSignalsOr413(w http.ResponseWriter, r *http.Request, signals any) bool {
	if err := datastar.ReadSignals(r, signals); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			http.Error(w, "リクエストが大きすぎます", http.StatusRequestEntityTooLarge)
			return false
		}
		http.Error(w, "無効なリクエストです", http.StatusBadRequest)
		return false
	}
	return true
}

// sendToast は #toast-container に通知を append し、数秒後に自動で取り除く。
// patch 化により reload しなくなった操作の成功フィードバックに使う。
func sendToast(sse *datastar.ServerSentEventGenerator, message string) {
	id := fmt.Sprintf("toast-%d", time.Now().UnixNano())
	_ = sse.PatchElementTempl(
		components.Toast(id, message),
		datastar.WithSelectorID("toast-container"),
		datastar.WithModeAppend(),
	)
	_ = sse.ExecuteScript(fmt.Sprintf("setTimeout(() => document.getElementById('%s')?.remove(), 3000)", id))
}
