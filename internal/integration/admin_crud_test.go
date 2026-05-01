package integration

import (
	"net/http"
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
