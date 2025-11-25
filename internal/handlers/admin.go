package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/appcontext"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/database"
	"github.com/naozine/project_crud_with_auth_tmpl/web/components"
	"github.com/naozine/project_crud_with_auth_tmpl/web/layouts"
)

type AdminHandler struct {
	Queries *database.Queries
}

func NewAdminHandler(queries *database.Queries) *AdminHandler {
	return &AdminHandler{Queries: queries}
}

// EnsureAdmin checks if the current user is an admin
func (h *AdminHandler) checkAdmin(c echo.Context) error {
	role := appcontext.GetUserRole(c.Request().Context())
	if role != "admin" {
		return echo.NewHTTPError(http.StatusForbidden, "Access denied: Admin role required")
	}
	return nil
}

func (h *AdminHandler) ListUsers(c echo.Context) error {
	if err := h.checkAdmin(c); err != nil {
		return err
	}

	users, err := h.Queries.ListUsers(c.Request().Context())
	if err != nil {
		log.Printf("Failed to list users: %v", err)
		return c.String(http.StatusInternalServerError, "Failed to list users")
	}

	// If HTMX request, render just the content part (e.g. after adding a user)
	// But for now, let's just re-render the whole page or list for simplicity
	// To make it smoother with HTMX, we might want to return just the list component
	// For now, full page render for GET /admin/users
	content := components.UserList(users)
	return layouts.Base("ユーザー管理", content).Render(c.Request().Context(), c.Response().Writer)
}

func (h *AdminHandler) NewUserPage(c echo.Context) error {
	if err := h.checkAdmin(c); err != nil {
		return err
	}
	return components.UserForm().Render(c.Request().Context(), c.Response().Writer)
}

func (h *AdminHandler) CreateUser(c echo.Context) error {
	if err := h.checkAdmin(c); err != nil {
		return err
	}

	name := c.FormValue("name")
	email := c.FormValue("email")
	role := c.FormValue("role")

	if name == "" || email == "" || role == "" {
		return c.String(http.StatusBadRequest, "Name, email and role are required")
	}

	// Validate role
	if role != "admin" && role != "editor" && role != "viewer" {
		return c.String(http.StatusBadRequest, "Invalid role")
	}

	_, err := h.Queries.CreateUser(c.Request().Context(), database.CreateUserParams{
		Email:    email,
		Name:     name,
		Role:     role,
		IsActive: true,
	})

	if err != nil {
		log.Printf("Failed to create user: %v", err)
		// In a real app, handle duplicate email error specifically
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to create user: %v", err))
	}

	// Return updated user list
	// Since HTMX target is "main" (from UserForm templ), we can just redirect or re-render the list page
	// Ideally we should return just the table or a fragment, but Base layout expects full page?
	// Actually, UserForm hx-target="main" implies replacing the main content.
	// Let's redirect to /admin/users which renders the full page (Base + UserList)
	// HTMX handles redirects by replacing the body if hx-boost is on or if we use hx-target="body".
	// If hx-target="main", it expects content to put inside main.

	// Let's just return the UserList component wrapped in Base? No, that would nest Base.
	// We want to re-render the UserList component and replace the current view.

	// Simple approach: Return the full page content again.
	// HTMX will put it in "main" if target is set.
	// But our Base template includes <main> tag.
	// If we return Base, we get <html ...> inside <main>.

	// Correct HTMX pattern:
	// If request is HTMX, return partial. If full load, return Base.
	// But here we are POSTing from a form inside the page.

	// Let's return the UserList component directly, but we need to fetch users again.
	users, err := h.Queries.ListUsers(c.Request().Context())
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to list users")
	}

	return components.UserList(users).Render(c.Request().Context(), c.Response().Writer)
}
