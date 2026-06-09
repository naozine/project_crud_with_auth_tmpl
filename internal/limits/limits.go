// Package limits は受信リクエストのサイズ上限など、HTTP 層のリソース制約値を一元管理する。
//
// ここに集約することで、上限を変更する際に middleware / handler / test の
// 複数箇所を grep して回る必要がなくなる。
package limits

const (
	// SSESignalBody は Datastar SSE（@post/@put）の signals JSON 受信 body 上限。
	// プロジェクト名・ユーザー名等の小さな signals のみを想定。
	SSESignalBody = 1 << 20 // 1 MB

	// UserImportBody は /admin/users/import の multipart 受信 body 上限。
	// 5 MB の Excel ファイル + multipart オーバーヘッド分の余裕を見込む。
	UserImportBody = 6 << 20 // 6 MB
)
