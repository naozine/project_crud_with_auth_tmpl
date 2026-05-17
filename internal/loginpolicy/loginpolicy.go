// Package loginpolicy は magiclink の AllowLogin フックに渡すログイン許可ロジックを提供する。
// main.go のクロージャから切り出すことで、テストから直接呼べるようにしている。
package loginpolicy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/naozine/project_crud_with_auth_tmpl/internal/database"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/logger"
)

// AllowLogin は magiclink の AllowLogin フックの本体。
// 戻り値が nil のときマジックリンクの送信を許可し、error の場合は
// その error 文字列がフォームに表示される（メール送信は行われない）。
//
// 判定順:
//  1. honeypot に値が入っている → ボット扱いで拒否
//  2. users にメアドが無い場合 → 「登録されていません」拒否
//  3. is_active=false の場合 → 「ご利用いただけません」拒否
//  4. アクティブな登録済みユーザー → 許可
func AllowLogin(ctx context.Context, q *database.Queries, email, honeypot string) error {
	if strings.TrimSpace(honeypot) != "" {
		logger.Warn("Honeypot triggered", "email", email)
		return fmt.Errorf("不正なリクエストです")
	}

	user, err := q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logger.Warn("Login attempt with unregistered email", "email", email)
			return fmt.Errorf("このメールアドレスは登録されていません")
		}
		logger.Error("Database error in AllowLogin", "error", err, "email", email)
		return fmt.Errorf("システムエラーが発生しました。")
	}
	if !user.IsActive {
		logger.Warn("Login attempt with inactive account", "email", email)
		return fmt.Errorf("このアカウントは現在ご利用いただけません")
	}
	return nil
}
