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

## Related
- [[architecture]] — System layering
- [[database-schema]] — SQLite tables
- [[htmx-patterns]] — Frontend interaction model
