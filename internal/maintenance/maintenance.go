// Package maintenance はメンテナンスモード（一般ユーザーのログイン受付停止）の
// オン／オフを管理するヘルパーを提供する。app_settings テーブルの 1 行
// （key="maintenance_mode"）に "true" / "false" を格納する。
package maintenance

import (
	"context"

	"github.com/naozine/project_crud_with_auth_tmpl/internal/database"
)

// Key は app_settings テーブルでメンテモード状態を保存する key 名。
const Key = "maintenance_mode"

// IsEnabled は現在メンテモードが ON かを返す。
// 行が無い／DB エラーなどの場合は false（安全側＝サービス継続）を返す。
func IsEnabled(ctx context.Context, q *database.Queries) bool {
	s, err := q.GetAppSetting(ctx, Key)
	if err != nil {
		return false
	}
	return s.Value == "true"
}

// SetEnabled はメンテモードを ON / OFF に切り替える。
func SetEnabled(ctx context.Context, q *database.Queries, on bool) error {
	val := "false"
	if on {
		val = "true"
	}
	return q.UpsertAppSetting(ctx, database.UpsertAppSettingParams{
		Key:   Key,
		Value: val,
	})
}
