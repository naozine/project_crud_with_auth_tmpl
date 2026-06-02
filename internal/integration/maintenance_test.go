package integration

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/naozine/project_crud_with_auth_tmpl/internal/database"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/maintenance"
)

// /admin/maintenance: アクセス権限
func TestMaintenance_PageAccess(t *testing.T) {
	conn := SetupTestDB(t)
	e := SetupTestServer(t, conn)
	seed := SeedTestData(t, conn)

	rec := DoRequest(e, http.MethodGet, "/admin/maintenance", &seed.AdminUser)
	if rec.Code != http.StatusOK {
		t.Errorf("admin: got %d, want %d", rec.Code, http.StatusOK)
	}
	rec = DoRequest(e, http.MethodGet, "/admin/maintenance", &seed.EditorUser)
	if rec.Code != http.StatusForbidden {
		t.Errorf("editor: got %d, want %d", rec.Code, http.StatusForbidden)
	}
	rec = DoRequest(e, http.MethodGet, "/admin/maintenance", &seed.ViewerUser)
	if rec.Code != http.StatusForbidden {
		t.Errorf("viewer: got %d, want %d", rec.Code, http.StatusForbidden)
	}
	rec = DoRequest(e, http.MethodGet, "/admin/maintenance", nil)
	if rec.Code != http.StatusSeeOther {
		t.Errorf("unauth: got %d, want %d", rec.Code, http.StatusSeeOther)
	}
}

// /api/sse/admin/maintenance/toggle: 権限と切替動作
func TestMaintenance_ToggleSSE(t *testing.T) {
	conn := SetupTestDB(t)
	e := SetupTestServer(t, conn)
	seed := SeedTestData(t, conn)
	q := database.New(conn)

	// 初期: OFF
	if maintenance.IsEnabled(t.Context(), q) {
		t.Fatalf("初期状態でメンテモードが ON になっている")
	}

	// admin がトグル → ON
	rec := DoSSERequest(e, http.MethodPost, "/api/sse/admin/maintenance/toggle", &seed.AdminUser, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("admin POST: got %d, want %d", rec.Code, http.StatusOK)
	}
	if !maintenance.IsEnabled(t.Context(), q) {
		t.Errorf("トグル後メンテモードが ON になっていない")
	}

	// もう一度トグル → OFF
	rec = DoSSERequest(e, http.MethodPost, "/api/sse/admin/maintenance/toggle", &seed.AdminUser, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("admin POST 2 回目: got %d", rec.Code)
	}
	if maintenance.IsEnabled(t.Context(), q) {
		t.Errorf("再トグル後メンテモードが OFF になっていない")
	}

	// editor / viewer / 未認証 は弾かれる
	rec = DoSSERequest(e, http.MethodPost, "/api/sse/admin/maintenance/toggle", &seed.EditorUser, "")
	if rec.Code != http.StatusForbidden {
		t.Errorf("editor: got %d, want %d", rec.Code, http.StatusForbidden)
	}
	rec = DoSSERequest(e, http.MethodPost, "/api/sse/admin/maintenance/toggle", &seed.ViewerUser, "")
	if rec.Code != http.StatusForbidden {
		t.Errorf("viewer: got %d, want %d", rec.Code, http.StatusForbidden)
	}
	rec = DoSSERequest(e, http.MethodPost, "/api/sse/admin/maintenance/toggle", nil, "")
	if rec.Code != http.StatusSeeOther {
		t.Errorf("unauth: got %d, want %d", rec.Code, http.StatusSeeOther)
	}
}

// メンテ ON 時、/auth/login (一般) はメンテ画面を表示し、ログインフォームは出ない
func TestMaintenance_LoginPageUnderMaintenance(t *testing.T) {
	conn := SetupTestDB(t)
	e := SetupTestServer(t, conn)
	_ = SeedTestData(t, conn)
	q := database.New(conn)

	// メンテ ON にしておく
	if err := maintenance.SetEnabled(t.Context(), q, true); err != nil {
		t.Fatalf("SetEnabled: %v", err)
	}

	rec := DoRequest(e, http.MethodGet, "/auth/login", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("/auth/login: got %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.Bytes()
	if !bytes.Contains(body, []byte("メンテナンス中です")) {
		t.Error("メンテ画面の文言が表示されていない")
	}
	if bytes.Contains(body, []byte(`id="login-form"`)) {
		t.Error("メンテ中なのにログインフォームが描画されている")
	}

	// 後始末
	_ = maintenance.SetEnabled(t.Context(), q, false)
}

// メンテ OFF 時は /auth/login が通常通りフォームを描画
func TestMaintenance_LoginPageWhenOff(t *testing.T) {
	conn := SetupTestDB(t)
	e := SetupTestServer(t, conn)
	_ = SeedTestData(t, conn)

	rec := DoRequest(e, http.MethodGet, "/auth/login", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.Bytes()
	if !bytes.Contains(body, []byte(`id="login-form"`)) {
		t.Error("OFF 時にログインフォームが出ていない")
	}
	if bytes.Contains(body, []byte("メンテナンス中です")) {
		t.Error("OFF なのにメンテ文言が表示されている")
	}
}
