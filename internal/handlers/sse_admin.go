package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/appcontext"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/database"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/logger"
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
	if err := datastar.ReadSignals(r, &signals); err != nil {
		http.Error(w, "無効なリクエストです", http.StatusBadRequest)
		return
	}

	if err := h.createUser(r.Context(), signals.NewName, signals.NewEmail, signals.NewRole); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sse := newSSE(w, r)
	sse.ExecuteScript("document.getElementById('user-add-dialog')?.close(); window.location.reload()")
}

func (h *AdminSSEHandler) createUser(ctx context.Context, name, email, role string) error {
	if name == "" || email == "" || role == "" {
		return http.ErrAbortHandler
	}

	if role != "admin" && role != "editor" && role != "viewer" {
		return http.ErrAbortHandler
	}

	if _, err := h.Queries.CreateUser(ctx, database.CreateUserParams{
		Email:    email,
		Name:     name,
		Role:     role,
		IsActive: true,
	}); err != nil {
		logger.Error("ユーザー作成に失敗", "error", err, "email", email)
		return err
	}

	return nil
}

func (h *AdminSSEHandler) EditUserDialogSSE(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "無効なIDです", http.StatusBadRequest)
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
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "無効なIDです", http.StatusBadRequest)
		return
	}

	var signals struct {
		EditName   string `json:"editName"`
		EditRole   string `json:"editRole"`
		EditStatus string `json:"editStatus"`
	}
	if err := datastar.ReadSignals(r, &signals); err != nil {
		http.Error(w, "無効なリクエストです", http.StatusBadRequest)
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
	sse.ExecuteScript("document.getElementById('user-edit-dialog')?.close(); window.location.reload()")
}

func (h *AdminSSEHandler) DeleteUserSSE(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "無効なIDです", http.StatusBadRequest)
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
	sse.ExecuteScript("document.getElementById('user-edit-dialog')?.close(); window.location.reload()")
}
