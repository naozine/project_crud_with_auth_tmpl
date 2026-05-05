package main

import (
	"os"

	"github.com/naozine/nz-magic-link/magiclink"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/appconfig"
)

// ConfigureBusinessSettings は magiclink の設定を派生プロジェクト固有の値に上書きする。
//
// アプリ名やランディング先など、UI/挙動の値は internal/appconfig/config.go を
// 直接編集する。ここでは magiclink ライブラリ側の設定だけを行う。
func ConfigureBusinessSettings(config *magiclink.Config) {
	config.RedirectURL = appconfig.LandingPath
	config.WebAuthnRedirectURL = appconfig.LandingPath

	// 負荷テスト用: DISABLE_RATE_LIMITING=true でレート制限を無効化
	if os.Getenv("DISABLE_RATE_LIMITING") == "true" {
		config.DisableRateLimiting = true
	}
}
