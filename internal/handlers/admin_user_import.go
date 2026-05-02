package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strings"

	"github.com/naozine/project_crud_with_auth_tmpl/internal/database"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/logger"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/models"
	"github.com/naozine/project_crud_with_auth_tmpl/web/components"
	"github.com/xuri/excelize/v2"
)

type UserImportHandler struct {
	Queries *database.Queries
}

func NewUserImportHandler(queries *database.Queries) *UserImportHandler {
	return &UserImportHandler{Queries: queries}
}

func (h *UserImportHandler) ImportPage(w http.ResponseWriter, r *http.Request) {
	renderShell(w, r, "ユーザー一括インポート", components.AdminUserImport(nil))
}

func (h *UserImportHandler) TemplateDownload(w http.ResponseWriter, r *http.Request) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	sheet := "Sheet1"
	_ = f.SetCellValue(sheet, "A1", "名前")
	_ = f.SetCellValue(sheet, "B1", "メールアドレス")
	_ = f.SetCellValue(sheet, "C1", "ロール")

	style, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E5E7EB"}, Pattern: 1},
	})
	_ = f.SetCellStyle(sheet, "A1", "C1", style)

	_ = f.SetCellValue(sheet, "A2", "田中太郎")
	_ = f.SetCellValue(sheet, "B2", "tanaka@example.com")
	_ = f.SetCellValue(sheet, "C2", "viewer")

	_ = f.SetColWidth(sheet, "A", "A", 20)
	_ = f.SetColWidth(sheet, "B", "B", 30)
	_ = f.SetColWidth(sheet, "C", "C", 15)

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=users_import_template.xlsx")
	_ = f.Write(w)
}

func (h *UserImportHandler) ExecuteImport(w http.ResponseWriter, r *http.Request) {
	// MaxBodySize ミドルウェアで body 全体は 6 MB に制限済み。
	// maxMemory も 6 MB にしておけば一時ファイルへの書き出しは発生しない。
	if err := r.ParseMultipartForm(6 << 20); err != nil { //nolint:gosec // body 上限は MaxBodySize ミドルウェアで設定済み
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			httpError(w, r, http.StatusRequestEntityTooLarge, "ファイルサイズが大きすぎます")
			return
		}
		httpError(w, r, http.StatusBadRequest, "リクエストの解析に失敗しました")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		httpError(w, r, http.StatusBadRequest, "ファイルを選択してください")
		return
	}
	defer func() { _ = file.Close() }()

	if header.Size > 5*1024*1024 {
		httpError(w, r, http.StatusBadRequest, "ファイルサイズは5MB以下にしてください")
		return
	}

	f, err := excelize.OpenReader(file)
	if err != nil {
		httpError(w, r, http.StatusBadRequest, "Excel ファイルの読み取りに失敗しました。.xlsx 形式のファイルを使用してください")
		return
	}
	defer func() { _ = f.Close() }()

	sheet := f.GetSheetName(0)
	rows, err := f.GetRows(sheet)
	if err != nil {
		httpError(w, r, http.StatusBadRequest, "シートの読み取りに失敗しました")
		return
	}

	if len(rows) < 2 {
		httpError(w, r, http.StatusBadRequest, "データ行がありません（1行目はヘッダー、2行目以降にデータを入力してください）")
		return
	}

	if len(rows) > 1001 {
		httpError(w, r, http.StatusBadRequest, "一度にインポートできるのは1000件までです")
		return
	}

	ctx := r.Context()
	result := &models.ImportResult{}
	seenEmails := make(map[string]int)

	for i, row := range rows[1:] {
		rowNum := i + 2

		if isEmptyRow(row) {
			continue
		}

		name := cellValue(row, 0)
		email := strings.ToLower(cellValue(row, 1))
		role := strings.ToLower(cellValue(row, 2))

		if name == "" {
			result.Errors = append(result.Errors, models.ImportRowError{Row: rowNum, Message: "名前は必須です"})
			continue
		}
		if email == "" {
			result.Errors = append(result.Errors, models.ImportRowError{Row: rowNum, Message: "メールアドレスは必須です"})
			continue
		}
		if _, err := mail.ParseAddress(email); err != nil {
			result.Errors = append(result.Errors, models.ImportRowError{Row: rowNum, Message: "メールアドレスの形式が不正です"})
			continue
		}
		if role != "viewer" && role != "editor" && role != "admin" {
			result.Errors = append(result.Errors, models.ImportRowError{Row: rowNum, Message: "ロールは viewer, editor, admin のいずれかを指定してください"})
			continue
		}

		if firstRow, exists := seenEmails[email]; exists {
			result.Errors = append(result.Errors, models.ImportRowError{
				Row:     rowNum,
				Message: fmt.Sprintf("ファイル内でメールアドレスが重複しています（%d行目と重複）", firstRow),
			})
			continue
		}
		seenEmails[email] = rowNum

		_, err := h.Queries.GetUserByEmail(ctx, email)
		if err == nil {
			result.Errors = append(result.Errors, models.ImportRowError{Row: rowNum, Message: "このメールアドレスは既に登録されています"})
			continue
		}
		if err != sql.ErrNoRows {
			logger.Error("インポート中の DB エラー", "error", err, "email", email, "row", rowNum)
			result.Errors = append(result.Errors, models.ImportRowError{Row: rowNum, Message: "データベースエラーが発生しました"})
			continue
		}

		_, err = h.Queries.CreateUser(ctx, database.CreateUserParams{
			Email:    email,
			Name:     name,
			Role:     role,
			IsActive: true,
		})
		if err != nil {
			logger.Error("ユーザー作成に失敗", "error", err, "email", email, "row", rowNum)
			result.Errors = append(result.Errors, models.ImportRowError{Row: rowNum, Message: "ユーザーの作成に失敗しました"})
			continue
		}

		result.SuccessCount++
	}

	if len(result.Errors) > 50 {
		total := len(result.Errors)
		result.Errors = result.Errors[:50]
		result.Errors = append(result.Errors, models.ImportRowError{
			Row:     0,
			Message: fmt.Sprintf("他にも %d 件のエラーがあります", total-50),
		})
	}

	renderShell(w, r, "ユーザー一括インポート", components.AdminUserImport(result))
}

func cellValue(row []string, idx int) string {
	if idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func isEmptyRow(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}
