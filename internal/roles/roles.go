// Package roles はユーザーのロール定数と検証を一元管理する。
//
// ロール文字列 ("admin" / "editor" / "viewer") をコード中に直接書かず、
// 必ずこのパッケージの定数を使うこと。文字列リテラルの混入は
// `make check-roles` で検出される（CLAUDE.md のコード規約も参照）。
package roles

// ロール定数。users.role カラムに保存される値と一致する。
const (
	Admin  = "admin"
	Editor = "editor"
	Viewer = "viewer"
)

// All は有効なロールの一覧。UI のセレクトボックスや一括検証で使う。
// 権限の弱い順（表示順）に並べる。
var All = []string{Viewer, Editor, Admin}

// IsValid は r が有効なロール文字列かを返す。
func IsValid(r string) bool {
	switch r {
	case Admin, Editor, Viewer:
		return true
	default:
		return false
	}
}
