package handlers

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
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

// CreateUserDialogSSE はダイアログからユーザーを作成し、一覧をリロードする。
func (h *AdminSSEHandler) CreateUserDialogSSE(c echo.Context) error {
	var signals struct {
		NewName  string `json:"newName"`
		NewEmail string `json:"newEmail"`
		NewRole  string `json:"newRole"`
	}
	if err := datastar.ReadSignals(c.Request(), &signals); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "無効なリクエストです")
	}

	if err := h.createUser(c, signals.NewName, signals.NewEmail, signals.NewRole); err != nil {
		return err
	}

	// ダイアログを閉じて一覧をリロード
	sse := newSSE(c)
	return sse.ExecuteScript("document.getElementById('user-add-dialog')?.hidePopover(); window.location.reload()")
}

func (h *AdminSSEHandler) createUser(c echo.Context, name, email, role string) error {
	if name == "" || email == "" || role == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "名前・メールアドレス・ロールは必須です")
	}

	if role != "admin" && role != "editor" && role != "viewer" {
		return echo.NewHTTPError(http.StatusBadRequest, "無効なロールです")
	}

	if _, err := h.Queries.CreateUser(c.Request().Context(), database.CreateUserParams{
		Email:    email,
		Name:     name,
		Role:     role,
		IsActive: true,
	}); err != nil {
		logger.Error("ユーザー作成に失敗", "error", err, "email", email)
		return echo.NewHTTPError(http.StatusInternalServerError, "ユーザーの作成に失敗しました")
	}

	return nil
}

// EditUserDialogSSE は編集ダイアログをSSEでパッチして表示する。
func (h *AdminSSEHandler) EditUserDialogSSE(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "無効なIDです")
	}

	user, err := h.Queries.GetUserByID(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "ユーザーが見つかりません")
	}

	sse := newSSE(c)
	if err := sse.ExecuteScript("document.getElementById('dialog-container').innerHTML = ''"); err != nil {
		return err
	}
	if err := sse.PatchElementTempl(
		components.AdminUserEditDialog(user),
		datastar.WithSelectorID("dialog-container"),
		datastar.WithModeInner(),
	); err != nil {
		return err
	}
	return sse.ExecuteScript("document.getElementById('user-edit-dialog').showPopover()")
}

// UpdateUserSSE はユーザーを更新し、ダイアログを閉じて一覧をリロードする。
func (h *AdminSSEHandler) UpdateUserSSE(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "無効なIDです")
	}

	var signals struct {
		EditName   string `json:"editName"`
		EditRole   string `json:"editRole"`
		EditStatus string `json:"editStatus"`
	}
	if err := datastar.ReadSignals(c.Request(), &signals); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "無効なリクエストです")
	}

	isActive := signals.EditStatus == "active"

	if _, err := h.Queries.UpdateUser(c.Request().Context(), database.UpdateUserParams{
		Name:     signals.EditName,
		Role:     signals.EditRole,
		IsActive: isActive,
		ID:       id,
	}); err != nil {
		logger.Error("ユーザー更新に失敗", "error", err, "id", id)
		return echo.NewHTTPError(http.StatusInternalServerError, "ユーザーの更新に失敗しました")
	}

	sse := newSSE(c)
	return sse.ExecuteScript("document.getElementById('user-edit-dialog')?.hidePopover(); window.location.reload()")
}

// DeleteUserSSE はユーザーを削除し、一覧にリダイレクトする。
func (h *AdminSSEHandler) DeleteUserSSE(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "無効なIDです")
	}

	// 自分自身の削除を防止
	currentUserID := appcontext.GetUserID(c.Request().Context())
	if id == currentUserID {
		return echo.NewHTTPError(http.StatusBadRequest, "自分自身を削除することはできません")
	}

	if err := h.Queries.DeleteUser(c.Request().Context(), id); err != nil {
		logger.Error("ユーザー削除に失敗", "error", err, "id", id)
		return echo.NewHTTPError(http.StatusInternalServerError, "ユーザーの削除に失敗しました")
	}

	sse := newSSE(c)
	return sse.ExecuteScript("document.getElementById('user-edit-dialog')?.hidePopover(); window.location.reload()")
}
