# Tech Stack

> Back to [[00-index]] · Related: [[architecture]]

## Runtime & Language

| Component | Choice | Version |
|---|---|---|
| Language | Go | 1.26.2 |
| Module | `radline` | — |
| Server | `net/http` (stdlib) | — |
| Templates | `html/template` (stdlib) | — |

## Dependencies

### Direct

| Package | Purpose | Notes |
|---|---|---|
| `github.com/jmoiron/sqlx` | SQL extensions for `database/sql` | Struct scanning via `db:` tags |
| `github.com/xuri/excelize/v2` | Excel file reader and writer | Used for batch data importing |
| `modernc.org/sqlite` | Pure-Go SQLite driver | **No CGO required** — critical for Windows dev. Registered as `"sqlite"` not `"sqlite3"` |

### Frontend (CDN)

| Library | Version | Purpose |
|---|---|---|
| HTMX | 1.9.10 | Hypermedia-driven UI updates |
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

### Why `modernc.org/sqlite` over `mattn/go-sqlite3`?
`mattn/go-sqlite3` requires CGO and a C compiler (GCC). On default Windows setups without MinGW, the build fails. `modernc.org/sqlite` is a **pure Go** translation of SQLite, requiring zero external tooling. See [[database-schema]] for schema details.

### Why no JSON API?
The application follows a strict **hypermedia architecture**. All state is managed server-side and rendered as HTML. HTMX handles partial page updates. This eliminates the need for a separate frontend framework. See [[htmx-patterns]].

### Why no migration tool?
The project uses inline startup migrations in [[database-migrations|db/db.go]]. This keeps the deployment simple (single binary + SQLite file) without requiring a migration CLI.

## Learnings

### Context: BackOffice System Go Rewrite
**Problem**: Migrating from Python Flask to Go + HTMX. Ensuring adherence to the `golang-htmx-dev` rules.
**Enforced Solution**: 
- Handlers will strictly return HTML fragments.
- State is managed purely on the server via HTMX attributes.
- Use `html/template` combined with `sqlx` for database querying and rendering.
- No JSON endpoints are allowed unless strictly necessary for an external integration (none planned).

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
- [[database-schema]] — SQLite tables
- [[htmx-patterns]] — Frontend interaction model
