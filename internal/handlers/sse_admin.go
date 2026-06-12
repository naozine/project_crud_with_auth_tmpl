package handlers

import (
	"context"
	"net/http"

	"github.com/naozine/project_crud_with_auth_tmpl/internal/appcontext"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/database"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/logger"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/roles"
	"github.com/naozine/project_crud_with_auth_tmpl/web/components"
	"github.com/starfederation/datastar-go/datastar"
)

type AdminSSEHandler struct {
	Queries *database.Queries
}

func NewAdminSSEHandler(queries *database.Queries) *AdminSSEHandler {
	return &AdminSSEHandler{Queries: queries}
}

func (h *AdminSSEHandler) CreateUserDialogSSE(w http.ResponseWriter, r *http.Request) {
	var signals struct {
		NewName  string `json:"newName"`
		NewEmail string `json:"newEmail"`
		NewRole  string `json:"newRole"`
	}
	if !readSignalsOr413(w, r, &signals) {
		return
	}

	if _, err := h.createUser(r.Context(), signals.NewName, signals.NewEmail, signals.NewRole); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sse := newSSE(w, r)
	// 一覧コンテナを再描画する（テーブル/カードの2系統を同期、reload しない）。
	if err := h.patchUserList(r.Context(), sse); err != nil {
		logger.Error("SSE PatchElementTempl failed", "error", err)
		return
	}
	sse.ExecuteScript("document.getElementById('user-add-dialog')?.close()")
	sendToast(sse, "ユーザーを追加しました")
}

// patchUserList は一覧コンテナ #users-list を最新の全ユーザーで inner 置換する。
// レスポンシブ（デスクトップ=テーブル / モバイル=カード）の2系統を同期させるため、
// 行単位ではなくコンテナごとまとめて差し替える。
func (h *AdminSSEHandler) patchUserList(ctx context.Context, sse *datastar.ServerSentEventGenerator) error {
	users, err := h.Queries.ListUsers(ctx)
	if err != nil {
		return err
	}
	return sse.PatchElementTempl(
		components.AdminUsersListBody(users),
		datastar.WithSelectorID("users-list"),
		datastar.WithModeInner(),
		datastar.WithViewTransitions(),
	)
}

func (h *AdminSSEHandler) createUser(ctx context.Context, name, email, role string) (database.User, error) {
	if name == "" || email == "" || role == "" {
		return database.User{}, http.ErrAbortHandler
	}

	if !roles.IsValid(role) {
		return database.User{}, http.ErrAbortHandler
	}

	user, err := h.Queries.CreateUser(ctx, database.CreateUserParams{
		Email:    email,
		Name:     name,
		Role:     role,
		IsActive: true,
	})
	if err != nil {
		logger.Error("ユーザー作成に失敗", "error", err, "email", email)
		return database.User{}, err
	}

	return user, nil
}

func (h *AdminSSEHandler) EditUserDialogSSE(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDOr400(w, r, "id")
	if !ok {
		return
	}

	user, err := h.Queries.GetUserByID(r.Context(), id)
	if err != nil {
		http.Error(w, "ユーザーが見つかりません", http.StatusNotFound)
		return
	}

	sse := newSSE(w, r)
	if err := sse.PatchElementTempl(
		components.AdminUserEditDialog(user),
		datastar.WithSelectorID("dialog-container"),
		datastar.WithModeInner(),
	); err != nil {
		logger.Error("SSE PatchElementTempl failed", "error", err)
		return
	}
	if err := sse.ExecuteScript("document.getElementById('user-edit-dialog')?.showModal();document.activeElement?.blur()"); err != nil {
		logger.Error("SSE ExecuteScript failed", "error", err)
		return
	}
}

func (h *AdminSSEHandler) UpdateUserSSE(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDOr400(w, r, "id")
	if !ok {
		return
	}

	var signals struct {
		EditName   string `json:"editName"`
		EditRole   string `json:"editRole"`
		EditStatus string `json:"editStatus"`
	}
	if !readSignalsOr413(w, r, &signals) {
		return
	}

	isActive := signals.EditStatus == "active"

	if _, err := h.Queries.UpdateUser(r.Context(), database.UpdateUserParams{
		Name:     signals.EditName,
		Role:     signals.EditRole,
		IsActive: isActive,
		ID:       id,
	}); err != nil {
		logger.Error("ユーザー更新に失敗", "error", err, "id", id)
		http.Error(w, "ユーザーの更新に失敗しました", http.StatusInternalServerError)
		return
	}

	sse := newSSE(w, r)
	// 一覧コンテナを再描画する（テーブル/カードの2系統を同期、reload しない）。
	if err := h.patchUserList(r.Context(), sse); err != nil {
		logger.Error("SSE PatchElementTempl failed", "error", err)
		return
	}
	sse.ExecuteScript("document.getElementById('user-edit-dialog')?.close()")
	sendToast(sse, "ユーザーを更新しました")
}

func (h *AdminSSEHandler) DeleteUserSSE(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDOr400(w, r, "id")
	if !ok {
		return
	}

	currentUserID := appcontext.GetUserID(r.Context())
	if id == currentUserID {
		http.Error(w, "自分自身を削除することはできません", http.StatusBadRequest)
		return
	}

	if err := h.Queries.DeleteUser(r.Context(), id); err != nil {
		logger.Error("ユーザー削除に失敗", "error", err, "id", id)
		http.Error(w, "ユーザーの削除に失敗しました", http.StatusInternalServerError)
		return
	}

	sse := newSSE(w, r)
	// 一覧コンテナを再描画する（テーブル/カードの2系統を同期、reload しない）。
	if err := h.patchUserList(r.Context(), sse); err != nil {
		logger.Error("SSE PatchElementTempl failed", "error", err)
	}
	sendToast(sse, "ユーザーを削除しました")
}
