// Package limits は受信リクエストのサイズ上限など、HTTP 層のリソース制約値を一元管理する。
//
// ここに集約することで、上限を変更する際に middleware / handler / test の
// 複数箇所を grep して回る必要がなくなる。
package limits

const (
	// ProjectFormBody は /projects/* の POST/PUT 受信 body 上限。
	// プロジェクト名等の小さなフォーム送信のみを想定。
	ProjectFormBody = 1 << 20 // 1 MB

	// UserImportBody は /admin/users/import の multipart 受信 body 上限。
	// 5 MB の Excel ファイル + multipart オーバーヘッド分の余裕を見込む。
	UserImportBody = 6 << 20 // 6 MB
)
