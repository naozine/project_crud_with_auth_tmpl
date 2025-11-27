# CLAUDE.md

## Commands
- **Build:** `go build -o bin/server ./cmd/server/main.go`
- **Run:** `go run ./cmd/server/main.go` (Only for manual testing, not for AI execution loops)
- **Test:** `go test ./...`
- **Gen Templ:** `templ generate`
- **Gen SQL:** `sqlc generate`
- **Format:** `gofmt -s -w . && templ fmt .`
- **Lint:** `golangci-lint run`

## Tech Stack & Principles
- **Stack:** Go (Echo v4), templ, htmx (Sparingly), Tailwind CSS, SQLite (WAL mode).
- **Data:** `sqlc` for type-safe SQL. **NO ORMs.**
- **Architecture:** Progressive Enhancement, PRG Pattern (Post-Redirect-Get), Type-Safe SQL.
- **Project Structure:** Standard Go Layout (`cmd/`, `internal/`, `web/`).

## Architectural Rules (STRICT)

### 1. Routing & State
- **Page-Based:** CRUD requires dedicated pages (`/items`, `/items/:id/edit`). No Modals for CRUD.
- **State:** Use URL Query Parameters (`?q=...`, `?return_to=...`) to preserve state.
- **Forms:** Use standard `<form>` (application/x-www-form-urlencoded). No JSON APIs for UI.

### 2. Rendering (The Hybrid Handler)
- **Pattern:** Dual-Mode Rendering.
    1. Check `c.Request().Header.Get("HX-Request") == "true"`.
    2. **True (htmx):** Render only the `templ` component.
    3. **False (Standard):** Render the full page layout wrapping the component.
- **Context:** Always pass `c.Request().Context()` to `templ` components.
- **No standard `c.Render`:** Use `component.Render(ctx, w)` directly.

### 3. Htmx Usage Policy
- **Limit Scope:** Do NOT build a full SPA. Use `hx-boost` only if requested.
- **Allowed Cases:** Active Search, Inline Validation, Dependent Dropdowns, Simple Toggles.
- **Deletes:** Handle via POST (or `_method`).

### 4. Database (SQLite + sqlc)
- **Config:** `file:app.db?_busy_timeout=5000&_journal_mode=WAL&_foreign_keys=on`
- **Pattern:** Write raw SQL in `query.sql`. Use `RETURNING` clause for INSERT/UPDATE.
- **DI:** Inject `*database.Queries` into handlers.
- **sqlc Note:** Hardcoding Japanese (non-ASCII) in SQL causes sqlc parse errors. Use constants from `internal/status` package and pass values as parameters from Go code.

## UI/UX Guidelines
- **Style:** Vercel/Stripe-inspired minimal monochrome.
- **Font:** Inter (Sans-serif).
- **Components:**
    - **Cards:** `bg-white`, thin border `border-gray-200`, `hover:shadow-md`.
    - **Buttons:** Primary `bg-black text-white hover:bg-gray-800`.
- **Language:** Japanese for UI text and code comments. English for technical terms.

## Code Example: Handler Pattern
```go
func (h *Handler) ListItems(c echo.Context) error {
    ctx := c.Request().Context()
    // 1. Data Access (sqlc)
    items, err := h.DB.ListItems(ctx)
    if err != nil { return echo.NewHTTPError(500, err) }

    // 2. Prepare Component
    content := components.ItemList(items)

    // 3. Dual-Mode Rendering
    c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTML)
    if c.Request().Header.Get("HX-Request") == "true" {
        return content.Render(ctx, c.Response().Writer)
    }
    return layouts.Base(content).Render(ctx, c.Response().Writer)
}