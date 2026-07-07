# Development

## Project structure

```
apps/
  api/               Go API entrypoint (main.go) + Dockerfile
  web/               React 19 + Vite + Tailwind frontend + tests
    src/
      components/    React components
      lib/           Utilities, API client, RBAC helpers
      pages/         Route views
    __tests__/       Frontend unit tests (Vitest)
    Dockerfile       Production nginx build
    package.json     npm scripts: dev, build, test, lint, format, check

internal/            Go backend packages (no circular deps)
  auth/              JWT + password + RBAC
  config/            Environment loading + validation
  domain/            Domain models + interfaces (no external deps)
  hermes/            AI service client + contract tests
  http/              Handlers, DTOs, error shaping
    handlers/
    dto/
    httperr/
    router.go
  pagination/        Page/page_size normalization
  repository/        Data access + GORM + Postgres
  service/           Business logic layer

services/
  hermes-bridge/     Python FastAPI AI bridge
    app/             FastAPI app, routes, config
    tests/           pytest test suite
    Dockerfile       Python build
    pyproject.toml   Dependencies (Hermes pinned git tag)

db/migrations/       SQL migrations (golang-migrate)
  0001_init.up.sql   Postgres extensions
  0002_users.up.sql
  ...
  0007_prospects.up.sql
  (and .down.sql for each)

deploy/              Docker Compose + Hermes config
  docker-compose.yml All-in-one development stack
  hermes/
    .env.example     Hermes environment variables
    config.yaml      MCP server + memory provider config

docs/                Documentation (this directory)

Makefile            Development shortcuts
.env.example        Root environment template
go.mod / go.sum     Go dependencies
```

---

## Common commands

### Go (API)

| Command | Purpose |
|---|---|
| `go run ./apps/api` | Start the API locally (port 8080) |
| `make run` | Same (loads .env) |
| `go build ./...` | Compile the entire codebase |
| `go test ./...` | Run all Go tests |
| `make test` | Same as above |
| `go vet ./...` | Static analysis (go vet) |
| `make vet` | Same as above |
| `golangci-lint run ./apps/api/... ./internal/...` | Comprehensive lint |
| `make lint` | Same as above |
| `make check` | Run vet + lint + test (all quality gates) |

### Frontend (Node.js / React)

| Command | Purpose |
|---|---|
| `cd apps/web && npm install` | Install dependencies |
| `npm run dev` | Start Vite dev server (port 5173, proxies `/api` → 8080) |
| `make web` | Same as above |
| `npm run build` | Produce static bundle for nginx (dist/) |
| `npm run test` | Run Vitest suite |
| `npm run lint` | Run ESLint |
| `npm run format` | Format code with Prettier |
| `npm run check` | Type-check with TypeScript |

### Python (hermes-bridge)

| Command | Purpose |
|---|---|
| `cd services/hermes-bridge && pip install .` | Install bridge + dependencies (includes pinned Hermes version) |
| `python -m app.main` | Start the bridge (port 8642) |
| `uvicorn app.main:app --host 0.0.0.0 --port 8642` | Alternative startup |
| `pytest` | Run test suite (24 tests) |
| `pytest -v` | Verbose test output |

### Make shortcuts (Windows: run from Git Bash)

| Command | Purpose |
|---|---|
| `make check` | Run `go vet ./...` + `golangci-lint run` + `go test ./...` |
| `make vet` | Run `go vet ./...` |
| `make lint` | Run `golangci-lint run ./apps/api/... ./internal/...` |
| `make test` | Run `go test ./...` |
| `make run` | Run `go run ./apps/api` (starts API on 8080) |
| `make web` | Run `cd apps/web && npm run dev` (starts frontend on 5173) |
| `make migrate-up` | Apply all pending migrations (requires `DATABASE_URL` env var) |
| `make migrate-down` | Roll back one migration |

---

## Testing

### Go Tests

```bash
# Run all Go tests
go test ./...

# Run tests in a specific package
go test ./internal/service/...

# Run with coverage
go test -cover ./...

# Run with verbose output and coverage
go test -v -cover ./...
```

Test files are colocated with source (`*_test.go`). Use the standard `testing` package.

### Contract tests (Hermes version gating)

```bash
go test ./internal/hermes -run TestContract
```

These tests verify that the hermes-bridge's `/v1` OpenAI-compatible surface conforms to the spec. They **must pass** before upgrading the Hermes version. See "Upgrading Hermes" below.

### Frontend Tests (Vitest)

```bash
cd apps/web

# Run tests
npm run test

# Run in watch mode
npm run test -- --watch

# Run with coverage
npm run test -- --coverage
```

Test files: `*.test.tsx` and `*.test.ts` in `__tests__/` or colocated with components.

### Python Tests (pytest)

```bash
cd services/hermes-bridge

# Run all tests
pytest

# Run a specific test file
pytest tests/test_chat.py

# Run with verbose output
pytest -v

# Run with coverage
pytest --cov=app tests/
```

Test files in `services/hermes-bridge/tests/`.

---

## Linting & formatting

### Go

```bash
# Lint (part of make check)
golangci-lint run ./apps/api/... ./internal/...

# Auto-format code (gofmt is built in; golangci-lint respects .golangci.yml)
go fmt ./...
```

### Frontend

```bash
cd apps/web

# Lint with ESLint
npm run lint

# Auto-format with Prettier
npm run format

# Type check (TypeScript)
npm run check
```

### Python

```bash
cd services/hermes-bridge

# Lint with ruff or flake8 (if configured in pyproject.toml)
# Note: currently pytest only; linting config TBD
```

---

## Database migrations

### Applying migrations

```bash
# Set DATABASE_URL (required)
export DATABASE_URL="postgres://salespilot:salespilot@127.0.0.1:5432/salespilot?sslmode=disable"

# Apply all pending migrations
migrate -path db/migrations -database "$DATABASE_URL" up

# Apply one specific migration
migrate -path db/migrations -database "$DATABASE_URL" up 1

# Roll back one migration
migrate -path db/migrations -database "$DATABASE_URL" down 1

# Roll back all migrations
migrate -path db/migrations -database "$DATABASE_URL" down
```

### Creating a new migration

```bash
migrate create -ext sql -dir db/migrations -seq <name>
```

This creates:
- `db/migrations/000N_<name>.up.sql` — apply logic
- `db/migrations/000N_<name>.down.sql` — rollback logic

Example:
```bash
migrate create -ext sql -dir db/migrations -seq add_user_active_flag
```

Creates `0008_add_user_active_flag.up.sql` and `.down.sql`.

### Migration naming

- Use `-seq` flag to auto-increment sequence numbers (0001, 0002, etc.)
- Filenames: `NNNN_descriptive_name.up.sql` and `NNNN_descriptive_name.down.sql`
- Use parameterized queries in `.up.sql`; ensure `.down.sql` is idempotent (use `IF EXISTS` clauses)

---

## Coding conventions

### Layered architecture

The Go backend is organized in layers; each layer only depends on layers below it:

```
http/ (handlers, DTOs, error responses)
  ↓
service/ (business logic, orchestration)
  ↓
repository/ (data access, GORM queries)
  ↓
domain/ (models, interfaces, no external dependencies)
```

**Rule:** Do not have `service/` import `http/`, `http/` import `service/` backwards, or `domain/` import anything else.

### Capability-based RBAC

Always use `auth.RequireCapability(cap)` middleware instead of raw role checks:

```go
// Good: capability-based
users := authd.Group("/users", auth.RequireCapability(auth.CapManageUsers))

// Avoid: role-based
if user.Role != "ADMIN" { ... }
```

See `internal/auth/rbac.go` for the capability matrix.

### Pagination

Always use `pagination.Normalize()` to clamp `page` and `page_size` before querying:

```go
page, pageSize := pagination.Normalize(reqPage, reqPageSize)
// page >= 1; pageSize in [1, pagination.MaxSize], falling back to pagination.DefaultSize (20) if out of range
```

### HTTP error shape

Use `httperr` package to shape all errors consistently:

```go
import "salespilot/internal/http/httperr"

// Generic error with a specific code
return httperr.Write(c, httperr.NewBadRequest("INVALID_STATUS_TRANSITION", "cannot transition from BIDDING to IDENTIFIED"))

// Common constructors: NewUnauthorized, NewForbidden, NewBadRequest(code, msg),
// NewValidation, NewNotFound, NewConflict(code, msg), NewInternal
```

All errors are serialized via `httperr.Write` as `{"error":{"code":"...","message":"..."}}`.

### DTOs & validation

Request/response bodies are DTOs (data transfer objects) in `internal/http/dto/`, with struct tags for validation, e.g.:

```go
type TenderCreateRequest struct {
    Title         string   `json:"title" validate:"required"`
    ValueEstimate *float64 `json:"value_estimate" validate:"omitempty,gte=0"`
    Status        *string  `json:"status" validate:"omitempty,oneof=IDENTIFIED QUALIFYING BIDDING SUBMITTED WON LOST"`
    // ...
}
```

Optional fields use pointer types (`*string`, `*float64`) so "not provided" can be distinguished from a zero value. Validate with the `go-playground/validator` package (already in use) via `c.Validate(&req)`.

### Comments

- English doc-comments for all exported types and functions
- Occasional inline comments in Indonesian for complex or non-obvious logic
- No comment duplication (e.g., `// Check if user is admin` above `if user.Role == "ADMIN"`)

---

## Upgrading Hermes

The Hermes `AIAgent` library is pinned in `services/hermes-bridge/pyproject.toml`. Upgrades are gated by contract tests to ensure the `/v1` OpenAI-compatible surface remains stable.

### Runbook

1. **Update the pinned version:**

   Edit `services/hermes-bridge/pyproject.toml`:
   ```toml
   hermes-agent = "git+https://github.com/NousResearch/hermes-agent.git@v2026.7.01"
   ```

   Replace `v2026.7.01` with the new tag.

2. **Install the new version:**

   ```bash
   cd services/hermes-bridge
   pip install .
   ```

3. **Run contract tests:**

   ```bash
   go test ./internal/hermes -run TestContract -v
   ```

   These tests call the bridge's `/v1` endpoints and verify OpenAI spec compliance.

   - **If tests pass:** Proceed to step 4.
   - **If tests fail:** The Hermes version introduced a breaking change. Investigate the failure, either revert the version bump or fix the bridge wrapper (`services/hermes-bridge/app/*`) to restore compatibility.

4. **Run the full test suite:**

   ```bash
   go test ./...
   cd apps/web && npm run test
   cd services/hermes-bridge && pytest
   ```

   Ensure all tests still pass.

5. **Commit and deploy:**

   ```bash
   git add services/hermes-bridge/pyproject.toml
   git commit -m "upgrade: hermes-agent to v2026.7.01"
   git push
   ```

   Redeploy the bridge service:
   ```bash
   docker build -t hermes-bridge:latest services/hermes-bridge/
   docker push hermes-bridge:latest
   ```

### Why contract tests?

The Go API only depends on the bridge's `/v1` surface (not the internal `AIAgent` library directly). Contract tests verify this boundary: if the Hermes library changes its `/v1` behavior, the tests catch it before the change reaches production.

---

## IDE & Editor setup

### Go (recommended: VS Code + Go extension)

Install the official [Go extension](https://marketplace.visualstudio.com/items?itemName=golang.go). It provides:
- Syntax highlighting
- IntelliSense (code completion)
- Debugging (delve)
- Go vet / golangci-lint integration

### React (recommended: VS Code + ES7+ React extension)

Install [ES7+ React/Redux/React-Native snippets](https://marketplace.visualstudio.com/items?itemName=dsznajder.es7-react-js-snippets) for JSX syntax highlighting and snippets.

### Python (recommended: VS Code + Python extension)

Install the official [Python extension](https://marketplace.visualstudio.com/items?itemName=ms-python.python).

---

## Troubleshooting dev issues

| Issue | Cause | Solution |
|---|---|---|
| `go: go.mod file not found` | Running `go` commands from wrong directory | Run from repo root (where `go.mod` is) |
| `make: command not found` | Make not installed or not in PATH | Install with `winget install ezwinports.make` or `choco install make` (Windows) |
| `migrate: command not found` | golang-migrate not installed | `go install -tags 'pgx5' github.com/golang-migrate/migrate/v4/cmd/migrate@latest`; ensure `%GOPATH%\bin` is in PATH |
| `npm ERR! ERESOLVE unable to resolve dependency tree` | Conflicting Node packages | `npm install --legacy-peer-deps` or update packages |
| `Connection refused: 127.0.0.1:5432` | Postgres not running | Start with `docker run -p 5433:5432 postgres:16`; override `DATABASE_URL` to `127.0.0.1:5433` |
| `Test fails with "config: required env <NAME> is empty"` | `.env` not loaded | Copy `.env.example` to `.env` and fill required vars |

---

## Performance profiling (optional)

### Go

```bash
# CPU profile
go test -cpuprofile=cpu.prof ./...
go tool pprof cpu.prof

# Memory profile
go test -memprofile=mem.prof ./...
go tool pprof mem.prof
```

### Frontend (Chrome DevTools)

1. Start the dev server: `npm run dev`
2. Open DevTools (F12 → Performance tab)
3. Record a session, then analyze the flame chart

### Python (cProfile)

```bash
cd services/hermes-bridge
python -m cProfile -s cumulative -m app.main > profile.txt
head -50 profile.txt
```
