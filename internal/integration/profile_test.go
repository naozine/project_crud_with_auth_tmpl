package integration

import (
	"net/http"
	"strings"
	"testing"
)

// TestProfile_UpdateProfile_PatchesSignalNoReload は、プロフィール名の更新が
// reload ではなく originalName signal の patch になっていることを担保する。
// （シェルは email 表示で名前を出さないため、signal 更新だけで足りる）
func TestProfile_UpdateProfile_PatchesSignalNoReload(t *testing.T) {
	conn := SetupTestDB(t)
	e := SetupTestServer(t, conn)
	seed := SeedTestData(t, conn)

	rec := DoSSERequest(e, http.MethodPut, "/api/sse/profile",
		&seed.ViewerUser,
		`{"profileName":"RenamedViewer"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	q := queryFromConn(conn)
	user, err := q.GetUserByID(t.Context(), seed.ViewerUser.ID)
	if err != nil {
		t.Fatalf("ユーザー取得に失敗: %v", err)
	}
	if user.Name != "RenamedViewer" {
		t.Errorf("Name = %q, want %q", user.Name, "RenamedViewer")
	}

	body := rec.Body.String()
	if strings.Contains(body, "location.reload") {
		t.Errorf("reload してはいけない。body: %s", body)
	}
	if !strings.Contains(body, "datastar-patch-signals") {
		t.Errorf("datastar-patch-signals が無い。body: %s", body)
	}
	if !strings.Contains(body, "originalName") {
		t.Errorf("originalName signal の patch が無い。body: %s", body)
	}
}
