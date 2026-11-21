# Tech Stack

> Back to [[00-index]] · Related: [[architecture]]

## Runtime & Language

| Component | Choice | Version |
|---|---|---|
| Application | Radline BackOffice | — |
| Language | Go | 1.26.2 |
| Module | `rav` | — |
| Server | `net/http` (stdlib, `http.CrossOriginProtection`) | — |
| Templates | `html/template` (stdlib) | — |

## Dependencies

### Direct

| Package | Purpose | Notes |
|---|---|---|
| `github.com/jmoiron/sqlx` | SQL extensions for `database/sql` | Wrapped by `*db.DBConn` and `*db.TxConn` for automatic `?`→`$1` rebinding |
| `github.com/jackc/pgx/v5` (`pgx/v5/stdlib`) | PostgreSQL driver | Pure-Go production database driver |
| `modernc.org/sqlite` | Pure-Go SQLite driver | **No CGO required** — local dev (`WAL`, `busy_timeout=5000`, `foreign_keys=ON`, `SetMaxOpenConns(1)`). Registered as `"sqlite"` |
| `golang.org/x/crypto/bcrypt` | Password hashing | Used in `domain/auth.go` (`HashPassword`, `CheckPasswordHash`) and `db/db.go` (`seedDefaults`) |
| `github.com/xuri/excelize/v2` | Excel file reader and writer | Used for batch data importing (`handlers/import.go`) |

### Frontend Assets (`static/`)

| Asset | Version / Path | Purpose |
|---|---|---|
| HTMX | 1.9.10 (`/static/htmx.min.js`) | Locally hosted hypermedia-driven UI updates |
| Design Tokens & Styles | `/static/index.css?v=6.5` | Vanilla CSS shadcn/QuickBooks Royal Indigo design tokens |
| Base UI Script | `/static/js/base-ui.js?v=6.5` | Sidebar, quick actions, alert dialog, toasts, dropdown helpers |
| Universal Table Engine | `/static/js/table-engine.js?v=6.5` | Column collapse/expand, resizing, grouping, reordering, header themes |
| Inter (Google Fonts) | 400/500/700 | Typography |

### Indirect (auto-pulled)

| Package | Purpose |
|---|---|
| `github.com/dustin/go-humanize` | modernc dependency |
| `github.com/google/uuid` | modernc dependency |
| `github.com/mattn/go-isatty` | Terminal detection |
| `github.com/ncruces/go-strftime` | Time formatting |
| `github.com/remyoudompheng/bigfft` | Big number FFT |
| `modernc.org/libc`, `memory`, `mathutil` | Pure-Go C runtime |
| `golang.org/x/sys` | System calls |

## Why These Choices

### Why Dual SQLite (`modernc.org/sqlite`) + PostgreSQL (`pgx/v5`)?
`mattn/go-sqlite3` requires CGO and a C compiler (GCC). `modernc.org/sqlite` is a **pure Go** SQLite implementation requiring zero external tooling for local development and tests (configured with `journal_mode=WAL`, `busy_timeout=5000`, `foreign_keys=ON`, `SetMaxOpenConns(1)`). In production (`APP_ENV=production`), `DATABASE_URL` connects via `github.com/jackc/pgx/v5` (`pgx/v5/stdlib`). `*db.DBConn` and `*db.TxConn` transparently rebind `?` placeholders to `$1, $2, ...`. See [[database-schema]].

### Why SSR HTML/HTMX with Targeted JSON Endpoints?
Radline BackOffice follows a **hypermedia-first architecture** where application state is rendered server-side as HTML via `html/template` and swapped via HTMX (`/static/htmx.min.js`). Targeted JSON endpoints are provided only where client-side persistence or autocomplete requires them: user preferences (`GET/POST /api/user-preferences*`) and fast item/customer search (`?format=json` on `/items/search` and `/customers/search`). See [[htmx-patterns]].

### Why Inline Startup Migrations?
The project uses idempotent inline startup migrations in [[database-migrations|`db/db.go`]] (`runSQLiteMigrations` and `runPostgresMigrations`). This keeps deployment simple (single binary) without requiring an external migration CLI.

## Learnings

### Context: BackOffice System Go Rewrite
**Problem**: Migrating from Python Flask to Go + HTMX while keeping the architecture lightweight and maintainable.
**Enforced Solution**: 
- Handlers primarily return HTML pages and fragments rendered via `html/template` and `sqlx`.
- State is managed on the server via HTMX attributes, with targeted JSON endpoints for user preferences (`/api/user-preferences*`) and search autocomplete (`?format=json`).

### Context: SQLite Database Driver on Windows/CGO-disabled Environments
**Problem**: Compiling the project on environments without a C compiler (like a default Windows/VS Code setup) fails because `github.com/mattn/go-sqlite3` requires CGO (`CGO_ENABLED=1`) and a GCC toolchain.
**Enforced Solution**:
- Use `modernc.org/sqlite`, a pure Go implementation of SQLite that does not require CGO.
- Register/connect the driver in `sqlx` as `"sqlite"` instead of `"sqlite3"`.

### Context: Continue MCP Server Configuration (Playwright & Filesystem)
**Problem**: The `.continue/mcpServers/` folder contained a placeholder template (`new-mcp-server.yaml`) with `<your-mcp-server>` as the command argument, causing the Continue extension to fail when trying to run `npx <your-mcp-server>` — which is not a real npm package.
**Enforced Solution**:
- **Deleted** the invalid placeholder `new-mcp-server.yaml` file.
- **Created** two proper MCP server config files:
  1. `playwright-mcp.yaml` — Runs `npx -y @playwright/mcp` for browser automation MCP tools.
  2. `filesystem-mcp.yaml` — Runs `npx -y @modelcontextprotocol/server-filesystem .` (with `.` as the root argument) for file system access MCP tools.
- **Config format**: Each `.yaml` follows the `schema: v1` format with `name`, `command: npx`, `args` as a list including `-y` and the server package name, and `env: {}`.
- **Important**: The Filesystem MCP server requires a directory argument — passing `.` (the project root) ensures it can access all project files from within the project root.

## Related
- [[architecture]] — System layering
- [[database-schema]] — Dual SQLite/PostgreSQL 21-table schema
- [[htmx-patterns]] — Frontend interaction model
