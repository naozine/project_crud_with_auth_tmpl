package integration

import (
	"net/http"
	"strings"
	"testing"
)

// projects の作成・更新・削除が reload/replace ではなく SSE patch になっていることを担保する。

func TestProjects_CreateSSE_PatchesGridNoReload(t *testing.T) {
	conn := SetupTestDB(t)
	e := SetupTestServer(t, conn)
	seed := SeedTestData(t, conn)

	rec := DoSSERequest(e, http.MethodPost, "/api/sse/projects/new",
		&seed.AdminUser, `{"name":"PatchedProject"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	body := rec.Body.String()
	if strings.Contains(body, "location.replace") || strings.Contains(body, "location.reload") {
		t.Errorf("reload/replace してはいけない。body: %s", body)
	}
	if !strings.Contains(body, "datastar-patch-elements") {
		t.Errorf("datastar-patch-elements が無い。body: %s", body)
	}
	if !strings.Contains(body, "projects-grid") {
		t.Errorf("projects-grid への patch が無い。body: %s", body)
	}
	if !strings.Contains(body, "PatchedProject") {
		t.Errorf("新プロジェクトが patch に含まれない。body: %s", body)
	}
}

func TestProjects_UpdateSSE_PatchesCardNoReload(t *testing.T) {
	conn := SetupTestDB(t)
	e := SetupTestServer(t, conn)
	seed := SeedTestData(t, conn)

	rec := DoSSERequest(e, http.MethodPut, sprintf("/api/sse/projects/%d", seed.Project.ID),
		&seed.AdminUser, `{"name":"RenamedProject"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	body := rec.Body.String()
	if strings.Contains(body, "location.replace") || strings.Contains(body, "location.reload") {
		t.Errorf("reload/replace してはいけない。body: %s", body)
	}
	if !strings.Contains(body, sprintf("project-%d", seed.Project.ID)) {
		t.Errorf("該当カード id (project-%d) が patch に含まれない。body: %s", seed.Project.ID, body)
	}
	if !strings.Contains(body, "RenamedProject") {
		t.Errorf("更新名が patch に含まれない。body: %s", body)
	}
}

func TestProjects_DeleteSSE_PatchesGridNoReload(t *testing.T) {
	conn := SetupTestDB(t)
	e := SetupTestServer(t, conn)
	seed := SeedTestData(t, conn)

	rec := DoSSERequest(e, http.MethodDelete, sprintf("/api/sse/projects/%d", seed.Project.ID),
		&seed.AdminUser, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	if strings.Contains(body, "location.replace") || strings.Contains(body, "location.reload") {
		t.Errorf("reload/replace してはいけない。body: %s", body)
	}
	if !strings.Contains(body, "datastar-patch-elements") {
		t.Errorf("datastar-patch-elements が無い。body: %s", body)
	}
	if !strings.Contains(body, "projects-grid") {
		t.Errorf("projects-grid への patch が無い。body: %s", body)
	}

	q := queryFromConn(conn)
	if _, err := q.GetProject(t.Context(), seed.Project.ID); err == nil {
		t.Error("削除されたはずのプロジェクトが見つかった")
	}
}
