package integration

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/naozine/project_crud_with_auth_tmpl/internal/database"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/limits"
)

// G120 (DoS / メモリ・ディスク枯渇) 対策のテスト。
// 各ハンドラの直前で http.MaxBytesReader を仕掛け、上限超過時に 413 を返すこと。
// 上限値は internal/limits パッケージで定義（ProjectFormBody, UserImportBody）。

// ---------------------------------------------------------------------------
// Project: POST /projects/new
// ---------------------------------------------------------------------------

func TestProjectsCreate_OverLimit(t *testing.T) {
	conn := SetupTestDB(t)
	e := SetupTestServer(t, conn)
	seed := SeedTestData(t, conn)

	// 上限を確実に超えるサイズ（上限の 2 倍）
	body := "name=" + strings.Repeat("x", 2*limits.ProjectFormBody)
	rec := DoRequest(e, http.MethodPost, "/projects/new", &seed.AdminUser, body)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("ステータスコード = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}

	// 上限超過時はレコードが作成されないこと（seed の 1 件のまま）
	q := queryFromConn(conn)
	projects, err := q.ListProjects(t.Context())
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projects) != 1 {
		t.Errorf("Project 件数 = %d, want 1（上限超過後にレコードが追加されている）", len(projects))
	}
}

func TestProjectsCreate_UnderLimit(t *testing.T) {
	conn := SetupTestDB(t)
	e := SetupTestServer(t, conn)
	seed := SeedTestData(t, conn)

	body := "name=Normal"
	rec := DoRequest(e, http.MethodPost, "/projects/new", &seed.AdminUser, body)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("ステータスコード = %d, want %d", rec.Code, http.StatusSeeOther)
	}

	q := queryFromConn(conn)
	projects, err := q.ListProjects(t.Context())
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projects) != 2 {
		t.Errorf("Project 件数 = %d, want 2（seed の 1 件 + 新規 1 件）", len(projects))
	}
}

// ---------------------------------------------------------------------------
// Project: POST /projects/{id}/update
// ---------------------------------------------------------------------------

func TestProjectsUpdate_OverLimit(t *testing.T) {
	conn := SetupTestDB(t)
	e := SetupTestServer(t, conn)
	seed := SeedTestData(t, conn)

	originalName := seed.Project.Name

	body := "name=" + strings.Repeat("y", 2*limits.ProjectFormBody)
	rec := DoRequest(e, http.MethodPost,
		fmt.Sprintf("/projects/%d/update", seed.Project.ID),
		&seed.AdminUser, body)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("ステータスコード = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}

	// 名前が変更されていないこと
	q := queryFromConn(conn)
	p, err := q.GetProject(t.Context(), seed.Project.ID)
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if p.Name != originalName {
		t.Errorf("Name = %q, want %q（上限超過後に名前が変更されている）", p.Name, originalName)
	}
}

// ---------------------------------------------------------------------------
// Excel インポート: POST /admin/users/import
// ---------------------------------------------------------------------------

func TestUserImport_OverLimit(t *testing.T) {
	conn := SetupTestDB(t)
	e := SetupTestServer(t, conn)
	seed := SeedTestData(t, conn)

	// 上限を確実に超える multipart body を送る。
	// 中身は不正な Excel だが、上限超過は MaxBytesReader の段階で検知されるため到達しない。
	rec := doOversizedFileUpload(t, e, "/admin/users/import", &seed.AdminUser, limits.UserImportBody+(1<<20))

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("ステータスコード = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}

	// 上限超過時はユーザーが作成されていないこと（seed の 4 件のまま）
	q := queryFromConn(conn)
	users, err := q.ListUsers(t.Context())
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(users) != 4 {
		t.Errorf("User 件数 = %d, want 4（上限超過後にユーザーが追加されている）", len(users))
	}
}

// doOversizedFileUpload は指定サイズの multipart リクエストを送る。
// 中身は 'x' で埋めた偽 .xlsx で、ファイル形式の妥当性は保証しない。
func doOversizedFileUpload(t *testing.T, h http.Handler, path string, user *database.User, fileSize int) *httptest.ResponseRecorder {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "huge.xlsx")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := io.CopyN(part, strings.NewReader(strings.Repeat("x", fileSize)), int64(fileSize)); err != nil {
		t.Fatalf("CopyN: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, path, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if user != nil {
		req.Header.Set("X-Test-User-ID", fmt.Sprintf("%d", user.ID))
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}
