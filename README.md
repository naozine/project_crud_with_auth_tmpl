# GOTH Stack Template (Go, Echo, Templ, Htmx)

A production-ready template for building web applications with the GOTH stack.

## Tech Stack

- **Go** (Echo v4)
- **Templ** (Type-safe HTML templates)
- **Htmx** (Frontend interactivity)
- **Tailwind CSS** (Styling)
- **SQLite + sqlc** (Type-safe database access)
- **nz-magic-link** (Passwordless authentication)

## Features

- **Clean Architecture:** Standard Go project layout (`cmd`, `internal`, `web`).
- **Authentication:** Magic link authentication (passwordless).
- **Authorization:** Middleware for protecting routes.
- **Type-Safe SQL:** No ORM, just raw SQL with type safety via `sqlc`.
- **Progressive Enhancement:** Works with and without JavaScript (mostly).
- **Modern UI:** Clean & Minimal design inspired by Vercel/Stripe.

## How to Use

### Option A: Using `gonew` (Recommended)

If you have `gonew` installed:

```bash
go install golang.org/x/tools/cmd/gonew@latest
gonew github.com/naozine/project_crud_with_auth_tmpl example.com/my-new-app
cd my-new-app
go mod tidy
```

### Option B: Manual Clone & Setup

1. **Clone the repository:**
   ```bash
   git clone https://github.com/naozine/project_crud_with_auth_tmpl.git my-new-app
   cd my-new-app
   ```

2. **Run the setup tool:**
   This tool will rename the module and cleanup example files.
   ```bash
   go run cmd/setup/main.go
   ```

3. **Install dependencies & Generate code:**
   ```bash
   go mod tidy
   go run github.com/a-h/templ/cmd/templ@latest generate
   ```

4. **Run the application:**
   ```bash
   # Development (with Air)
   air
   
   # Or standard build
   go build -o app cmd/server/main.go
   ./app
   ```

## Project Structure

```
.
├── cmd/
│   ├── server/       # Application entry point
│   └── setup/        # Project initialization tool
├── internal/
│   ├── appcontext/   # Context helpers
│   ├── database/     # sqlc generated code
│   ├── handlers/     # HTTP handlers
│   └── middleware/   # Custom middleware
├── web/
│   ├── components/   # Templ components
│   ├── layouts/      # Page layouts
│   └── static/       # Static assets
├── db/               # SQL migrations and queries
└── sqlc.yaml         # sqlc configuration
```

## License

MIT
