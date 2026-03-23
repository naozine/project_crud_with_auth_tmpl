package handlers

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/starfederation/datastar-go/datastar"
)

// newSSE は ResponseWriter と Request から SSE ジェネレーターを作成する。
func newSSE(w http.ResponseWriter, r *http.Request) *datastar.ServerSentEventGenerator {
	return datastar.NewSSE(w, r)
}

// patchContent は #main-content の中身を差し替える。
func patchContent(sse *datastar.ServerSentEventGenerator, component templ.Component) error {
	return sse.PatchElementTempl(
		component,
		datastar.WithSelectorID("main-content"),
		datastar.WithModeInner(),
	)
}
