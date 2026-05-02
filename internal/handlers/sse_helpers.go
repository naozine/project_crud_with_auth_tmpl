package handlers

import (
	"net/http"

	"github.com/starfederation/datastar-go/datastar"
)

// newSSE は ResponseWriter と Request から SSE ジェネレーターを作成する。
func newSSE(w http.ResponseWriter, r *http.Request) *datastar.ServerSentEventGenerator {
	return datastar.NewSSE(w, r)
}
