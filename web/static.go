package web

import "embed"

// StaticFS は web/static 配下の全ファイルをバイナリに同梱する。
//
//go:embed static
var StaticFS embed.FS
