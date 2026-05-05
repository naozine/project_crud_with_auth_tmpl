package integration

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// 初期セットアップフロー (/setup) のテスト。
// 派生プロジェクトでも初回管理者作成の手順は共通なので、ここで挙動を保証する。

// ---------------------------------------------------------------------------
// GET /setup
// ---------------------------------------------------------------------------

func TestSetup_PageWithNoUsers(t *testing.T) {
	conn := SetupTestDB(t)
	e := SetupTestServer(t, conn)

	rec := DoRequest(e, http.MethodGet, "/setup", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d, want %d", rec.Code, http.StatusOK)
	}
	// HTML フォームが返ること
	body := rec.Body.String()
	if !strings.Contains(body, "<form") {
		t.Errorf("レスポンスにフォームが含まれていない")
	}
}

func TestSetup_PageWithExistingUsers(t *testing.T) {
	conn := SetupTestDB(t)
	e := SetupTestServer(t, conn)
	SeedTestData(t, conn) // 既にユーザーがいる状態

	rec := DoRequest(e, http.MethodGet, "/setup", nil)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("ステータスコード = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if loc := rec.Header().Get("Location"); loc != "/auth/login" {
		t.Errorf("リダイレクト先 = %q, want /auth/login", loc)
	}
}

// ---------------------------------------------------------------------------
// POST /setup
// ---------------------------------------------------------------------------

func TestSetup_CreateAdminSuccess(t *testing.T) {
	conn := SetupTestDB(t)
	e := SetupTestServer(t, conn)

	rec := DoSSERequest(e, http.MethodPost, "/setup", nil,
		`{"email":"newadmin@test.com","name":"New Admin"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	// レスポンスが success:true であること
	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("レスポンスの JSON パースに失敗: %v", err)
	}
	if success, _ := resp["success"].(bool); !success {
		t.Errorf("success = %v, want true", resp["success"])
	}

	// DB に admin が作成されていること
	q := queryFromConn(conn)
	user, err := q.GetUserByEmail(t.Context(), "newadmin@test.com")
	if err != nil {
		t.Fatalf("作成された admin が見つからない: %v", err)
	}
	if user.Role != "admin" {
		t.Errorf("Role = %q, want %q", user.Role, "admin")
	}
	if user.Name != "New Admin" {
		t.Errorf("Name = %q, want %q", user.Name, "New Admin")
	}
	if !user.IsActive {
		t.Error("IsActive = false, want true")
	}
}

func TestSetup_CreateAdminDefaultName(t *testing.T) {
	conn := SetupTestDB(t)
	e := SetupTestServer(t, conn)

	// name を省略
	rec := DoSSERequest(e, http.MethodPost, "/setup", nil,
		`{"email":"noname@test.com"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d, want %d", rec.Code, http.StatusOK)
	}

	q := queryFromConn(conn)
	user, err := q.GetUserByEmail(t.Context(), "noname@test.com")
	if err != nil {
		t.Fatalf("作成された admin が見つからない: %v", err)
	}
	if user.Name != "Admin" {
		t.Errorf("Name = %q, want %q（name 省略時のデフォルト）", user.Name, "Admin")
	}
}

func TestSetup_CreateAdminAlreadySetup(t *testing.T) {
	conn := SetupTestDB(t)
	e := SetupTestServer(t, conn)
	SeedTestData(t, conn) // 既にユーザーがいる状態

	rec := DoSSERequest(e, http.MethodPost, "/setup", nil,
		`{"email":"another@test.com","name":"Another"}`)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("ステータスコード = %d, want %d", rec.Code, http.StatusForbidden)
	}

	// 新規ユーザーが作成されていないこと
	q := queryFromConn(conn)
	if _, err := q.GetUserByEmail(t.Context(), "another@test.com"); err == nil {
		t.Error("既にセットアップ済みなのにユーザーが作成されてしまった")
	}
}

func TestSetup_CreateAdminInvalidJSON(t *testing.T) {
	conn := SetupTestDB(t)
	e := SetupTestServer(t, conn)

	rec := DoSSERequest(e, http.MethodPost, "/setup", nil,
		`{not valid json`)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("ステータスコード = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestSetup_CreateAdminMissingEmail(t *testing.T) {
	conn := SetupTestDB(t)
	e := SetupTestServer(t, conn)

	rec := DoSSERequest(e, http.MethodPost, "/setup", nil,
		`{"name":"NoEmail"}`)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("ステータスコード = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	// ユーザーが作成されていないこと
	q := queryFromConn(conn)
	users, err := q.ListUsers(t.Context())
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(users) != 0 {
		t.Errorf("User 件数 = %d, want 0（email 必須エラー後にユーザーが作成された）", len(users))
	}
}
