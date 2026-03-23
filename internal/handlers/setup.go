package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/naozine/project_crud_with_auth_tmpl/internal/database"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/logger"
	"github.com/naozine/project_crud_with_auth_tmpl/web/components"
)

type SetupHandler struct {
	Queries *database.Queries
}

func NewSetupHandler(queries *database.Queries) *SetupHandler {
	return &SetupHandler{Queries: queries}
}

func (h *SetupHandler) hasUsers(r *http.Request) (bool, error) {
	count, err := h.Queries.CountUsers(r.Context())
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (h *SetupHandler) SetupPage(w http.ResponseWriter, r *http.Request) {
	hasUsers, err := h.hasUsers(r)
	if err != nil {
		logger.Error("Failed to count users", "error", err)
		httpError(w, r, http.StatusInternalServerError, "データベースエラーが発生しました")
		return
	}

	if hasUsers {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	renderGuest(w, r, "初期セットアップ", components.SetupForm(""))
}

func (h *SetupHandler) CreateInitialAdmin(w http.ResponseWriter, r *http.Request) {
	hasUsers, err := h.hasUsers(r)
	if err != nil {
		logger.Error("Failed to count users", "error", err)
		jsonError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if hasUsers {
		jsonError(w, http.StatusForbidden, "セットアップは既に完了しています")
		return
	}

	var req struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if req.Email == "" {
		jsonError(w, http.StatusBadRequest, "メールアドレスは必須です")
		return
	}

	if req.Name == "" {
		req.Name = "Admin"
	}

	_, err = h.Queries.CreateUser(r.Context(), database.CreateUserParams{
		Email:    req.Email,
		Name:     req.Name,
		Role:     "admin",
		IsActive: true,
	})
	if err != nil {
		logger.Error("Failed to create admin user", "error", err, "email", req.Email)
		jsonError(w, http.StatusInternalServerError, "ユーザーの作成に失敗しました")
		return
	}

	logger.Info("Initial admin user created", "email", req.Email)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"email":   req.Email,
		"message": "管理者ユーザーを作成しました。パスキーを登録してください。",
	})
}

// jsonError はJSONエラーレスポンスを返す。
func jsonError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
