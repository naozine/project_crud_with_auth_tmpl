package handlers

import (
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
