package integration

import (
	"strings"
	"testing"

	"github.com/naozine/project_crud_with_auth_tmpl/internal/database"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/loginpolicy"
)

// AllowLogin の判定をユニットレベルで網羅する。HTTP / magiclink を介さない。
func TestAllowLogin(t *testing.T) {
	conn := SetupTestDB(t)
	seed := SeedTestData(t, conn)
	q := database.New(conn)
	ctx := t.Context()

	// inactive ユーザー
	if _, err := q.CreateUser(ctx, database.CreateUserParams{
		Email:    "inactive@test.com",
		Name:     "Inactive",
		Role:     "viewer",
		IsActive: false,
	}); err != nil {
		t.Fatalf("inactive seed: %v", err)
	}

	cases := []struct {
		name       string
		email      string
		honeypot   string
		wantErrSub string // 部分一致。空なら許可（nil）を期待
	}{
		{"未登録メアド → 拒否", "nobody@test.com", "", "登録されていません"},
		{"inactive ユーザー → 拒否", "inactive@test.com", "", "ご利用いただけません"},
		{"アクティブな admin → 許可", seed.AdminUser.Email, "", ""},
		{"アクティブな viewer → 許可", seed.ViewerUser.Email, "", ""},
		{"Honeypot に値あり → 拒否（メアドが正規でも）", seed.AdminUser.Email, "https://attack.example", "不正なリクエスト"},
		{"Honeypot 空白のみ → 許可（空文字と同等）", seed.AdminUser.Email, "  ", ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := loginpolicy.AllowLogin(ctx, q, c.email, c.honeypot)
			if c.wantErrSub == "" {
				if err != nil {
					t.Errorf("got %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("got nil, want error containing %q", c.wantErrSub)
			}
			if !strings.Contains(err.Error(), c.wantErrSub) {
				t.Errorf("error = %q, want に %q を含む", err.Error(), c.wantErrSub)
			}
		})
	}
}
