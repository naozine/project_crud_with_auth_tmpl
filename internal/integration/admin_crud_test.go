package integration

import (
	"net/http"
	"strings"
	"testing"
)

// admin の CRUD は Datastar SSE モーダル化されており、
// /api/sse/admin/users/* の JSON シグナルを叩く形でテストする。

func TestAdminCRUD_CreateUser(t *testing.T) {
	conn := SetupTestDB(t)
	e := SetupTestServer(t, conn)
	seed := SeedTestData(t, conn)

	rec := DoSSERequest(e, http.MethodPost, "/api/sse/admin/users/create",
		&seed.AdminUser,
		`{"newName":"NewUser","newEmail":"newuser@test.com","newRole":"editor"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	q := queryFromConn(conn)
	user, err := q.GetUserByEmail(t.Context(), "newuser@test.com")
	if err != nil {
		t.Fatalf("作成されたユーザーが見つからない: %v", err)
	}
	if user.Name != "NewUser" {
		t.Errorf("Name = %q, want %q", user.Name, "NewUser")
	}
	if user.Role != "editor" {
		t.Errorf("Role = %q, want %q", user.Role, "editor")
	}
	if !user.IsActive {
		t.Error("IsActive = false, want true")
	}
}

func TestAdminCRUD_CreateUser_ValidationError(t *testing.T) {
	conn := SetupTestDB(t)
	e := SetupTestServer(t, conn)
	seed := SeedTestData(t, conn)

	// 必須フィールドが空 → ハンドラが http.ErrAbortHandler を返し、500 となる
	rec := DoSSERequest(e, http.MethodPost, "/api/sse/admin/users/create",
		&seed.AdminUser,
		`{"newName":"","newEmail":"","newRole":""}`)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("ステータスコード = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestAdminCRUD_UpdateUser(t *testing.T) {
	conn := SetupTestDB(t)
	e := SetupTestServer(t, conn)
	seed := SeedTestData(t, conn)

	targetID := seed.ViewerUser.ID

	rec := DoSSERequest(e, http.MethodPut, sprintf("/api/sse/admin/users/%d", targetID),
		&seed.AdminUser,
		`{"editName":"UpdatedName","editRole":"editor","editStatus":"active"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	q := queryFromConn(conn)
	user, err := q.GetUserByID(t.Context(), targetID)
	if err != nil {
		t.Fatalf("ユーザーの取得に失敗: %v", err)
	}
	if user.Name != "UpdatedName" {
		t.Errorf("Name = %q, want %q", user.Name, "UpdatedName")
	}
	if user.Role != "editor" {
		t.Errorf("Role = %q, want %q", user.Role, "editor")
	}
}

func TestAdminCRUD_UpdateUser_Deactivate(t *testing.T) {
	conn := SetupTestDB(t)
	e := SetupTestServer(t, conn)
	seed := SeedTestData(t, conn)

	targetID := seed.ViewerUser.ID

	rec := DoSSERequest(e, http.MethodPut, sprintf("/api/sse/admin/users/%d", targetID),
		&seed.AdminUser,
		`{"editName":"Viewer","editRole":"viewer","editStatus":"inactive"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d, want %d", rec.Code, http.StatusOK)
	}

	q := queryFromConn(conn)
	user, err := q.GetUserByID(t.Context(), targetID)
	if err != nil {
		t.Fatalf("ユーザーの取得に失敗: %v", err)
	}
	if user.IsActive {
		t.Error("IsActive = true, want false")
	}
}

// TestAdminCRUD_UpdateUser_PatchesCardNoReload は、編集保存が reload ではなく patch で
// 反映されることを担保する（UI 更新方針の回帰防止）。一覧コンテナ (#users-list) ごと
// 再描画する方式のため、更新後のユーザーが新しい内容に含まれることを確認する。
func TestAdminCRUD_UpdateUser_PatchesCardNoReload(t *testing.T) {
	conn := SetupTestDB(t)
	e := SetupTestServer(t, conn)
	seed := SeedTestData(t, conn)

	targetID := seed.ViewerUser.ID

	rec := DoSSERequest(e, http.MethodPut, sprintf("/api/sse/admin/users/%d", targetID),
		&seed.AdminUser,
		`{"editName":"PatchedName","editRole":"editor","editStatus":"active"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	body := rec.Body.String()
	if strings.Contains(body, "location.reload") {
		t.Errorf("reload してはいけない（patch 化されていない）。body: %s", body)
	}
	if !strings.Contains(body, "datastar-patch-elements") {
		t.Errorf("datastar-patch-elements イベントが無い。body: %s", body)
	}
	if !strings.Contains(body, sprintf("user-%d", targetID)) {
		t.Errorf("該当カード id (user-%d) が patch に含まれない。body: %s", targetID, body)
	}
	if !strings.Contains(body, "PatchedName") {
		t.Errorf("更新後の名前が patch に含まれない。body: %s", body)
	}
}

func TestAdminCRUD_DeleteUser(t *testing.T) {
	conn := SetupTestDB(t)
	e := SetupTestServer(t, conn)
	seed := SeedTestData(t, conn)

	targetID := seed.DeletableUser.ID

	rec := DoSSERequest(e, http.MethodDelete, sprintf("/api/sse/admin/users/%d", targetID),
		&seed.AdminUser, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d, want %d", rec.Code, http.StatusOK)
	}

	q := queryFromConn(conn)
	_, err := q.GetUserByID(t.Context(), targetID)
	if err == nil {
		t.Error("削除されたはずのユーザーが見つかった")
	}
}

func TestAdminCRUD_DeleteUser_PreventSelfDeletion(t *testing.T) {
	conn := SetupTestDB(t)
	e := SetupTestServer(t, conn)
	seed := SeedTestData(t, conn)

	rec := DoSSERequest(e, http.MethodDelete, sprintf("/api/sse/admin/users/%d", seed.AdminUser.ID),
		&seed.AdminUser, "")

	if rec.Code != http.StatusBadRequest {
		t.Errorf("ステータスコード = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	q := queryFromConn(conn)
	_, err := q.GetUserByID(t.Context(), seed.AdminUser.ID)
	if err != nil {
		t.Error("自分自身が削除されてしまった")
	}
}

// TestAdminCRUD_CreateUser_AppendsCardNoReload は、追加が reload ではなく patch で
// 反映されることを担保する。一覧コンテナ (#users-list) ごと再描画する方式のため、
// 追加したユーザーが新しい内容に含まれることを確認する。
func TestAdminCRUD_CreateUser_AppendsCardNoReload(t *testing.T) {
	conn := SetupTestDB(t)
	e := SetupTestServer(t, conn)
	seed := SeedTestData(t, conn)

	rec := DoSSERequest(e, http.MethodPost, "/api/sse/admin/users/create",
		&seed.AdminUser,
		`{"newName":"AppendedUser","newEmail":"appended@test.com","newRole":"editor"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	body := rec.Body.String()
	if strings.Contains(body, "location.reload") {
		t.Errorf("reload してはいけない。body: %s", body)
	}
	if !strings.Contains(body, "datastar-patch-elements") {
		t.Errorf("datastar-patch-elements が無い。body: %s", body)
	}
	if !strings.Contains(body, "AppendedUser") {
		t.Errorf("新ユーザーのカードが patch に含まれない。body: %s", body)
	}
}

// TestAdminCRUD_DeleteUser_RemovesCardNoReload は、削除が reload ではなく patch で
// 反映されることを担保する。レスポンシブ（テーブル+カード）対応のため、行単位の除去では
// なく一覧コンテナ (#users-list) ごと再描画する方式。削除後はそのユーザーが新しい内容に
// 含まれないことを確認する。
func TestAdminCRUD_DeleteUser_RemovesCardNoReload(t *testing.T) {
	conn := SetupTestDB(t)
	e := SetupTestServer(t, conn)
	seed := SeedTestData(t, conn)

	targetID := seed.DeletableUser.ID

	rec := DoSSERequest(e, http.MethodDelete, sprintf("/api/sse/admin/users/%d", targetID),
		&seed.AdminUser, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	if strings.Contains(body, "location.reload") {
		t.Errorf("reload してはいけない。body: %s", body)
	}
	if !strings.Contains(body, "datastar-patch-elements") {
		t.Errorf("datastar-patch-elements イベントが無い。body: %s", body)
	}
	if !strings.Contains(body, "#users-list") {
		t.Errorf("一覧コンテナ #users-list への patch になっていない。body: %s", body)
	}
	if strings.Contains(body, sprintf("id=\"user-%d\"", targetID)) {
		t.Errorf("削除したユーザーのカード id (user-%d) が再描画後も残っている。body: %s", targetID, body)
	}
}
