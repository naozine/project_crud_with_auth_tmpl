// Package appconfig はアプリケーション全体で共有される設定定数を定義する。
//
// 派生プロジェクトでは、このファイルを直接編集してアプリ固有の値に変更する。
// （A 戦略: Core / Business の境界を強制せず、派生で必要なファイルは自由に編集してよい）
package appconfig

const (
	// AppName は UI 各所に表示されるアプリ名。
	AppName = "プロジェクト管理"

	// LandingPath はログイン成功後・WebAuthn 検証成功後の既定リダイレクト先。
	// magiclink の RedirectURL / WebAuthnRedirectURL の両方にデフォルトとして渡される。
	LandingPath = "/projects"
)
