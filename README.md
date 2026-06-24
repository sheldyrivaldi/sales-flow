# SalesPilot

Internal sales intelligence platform — Go + React + Hermes AI.

## Prerequisites

| Tool | Required | Install |
|---|---|---|
| Docker Desktop | Yes | [docker.com](https://www.docker.com/products/docker-desktop) |
| Go 1.22+ | Native dev | [go.dev](https://go.dev/dl/) |
| Node.js 20+ | Native dev | [nodejs.org](https://nodejs.org) |
| golang-migrate CLI | Native dev | `go install -tags 'pgx5' github.com/golang-migrate/migrate/v4/cmd/migrate@latest` |
| golangci-lint | CI/lint | `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest` |
| GNU make | Optional | `winget install ezwinports.make` or `choco install make` |

**Windows PATH:** add `%GOPATH%\bin` (default: `C:\Users\<you>\go\bin`) to your PATH so `migrate` and `golangci-lint` are available in your shell.

---

## Run with Docker (recommended)

```bash
# 1. Copy and fill environment variables
cp .env.example .env
# Edit .env — fill JWT_SECRET, API_SERVER_KEY, WORKSPACE_SESSION_KEY,
# SALES_MCP_TOKEN, SEED_ADMIN_EMAIL, SEED_ADMIN_PASSWORD

# 2. Copy Hermes config
cp deploy/hermes/.env.example deploy/hermes/.env
cp deploy/hermes/config.yaml.example deploy/hermes/config.yaml
# Edit both files — fill API_SERVER_KEY, SALES_MCP_TOKEN

# 3. Start the stack (postgres + api + web)
docker compose -f deploy/docker-compose.yml up

# API:  http://localhost:8080/healthz
# Web:  http://localhost:80
```

> **Hermes Bridge:** the AI service (`hermes-bridge`) is built from source in
> `services/hermes-bridge/` — a FastAPI wrapper around the Hermes `AIAgent` Python library.
> Set `OPENAI_API_KEY` or `OPENROUTER_API_KEY` + `HERMES_MODEL` in `.env` to activate AI features.
> CRUD features remain available even if the bridge is unavailable (graceful degrade).

---

## Run natively on Windows

> This guide assumes a local Postgres on port **5433** to avoid conflicts with other
> installations. Adjust `DATABASE_URL` in `.env` if yours runs on a different port.

### 1. Start Postgres

```powershell
docker run -d --name sp-pg `
  -e POSTGRES_USER=salespilot `
  -e POSTGRES_PASSWORD=salespilot `
  -e POSTGRES_DB=salespilot `
  -e POSTGRES_HOST_AUTH_METHOD=trust `
  -p 5433:5432 postgres:16
```

### 2. Run migrations

```bash
export DATABASE_URL="postgres://salespilot:salespilot@127.0.0.1:5433/salespilot?sslmode=disable"
migrate -path db/migrations -database "$DATABASE_URL" up
```

### 3. Start the API

```bash
# Copy .env.example → .env and fill in values first, then:
export $(cat .env | grep -v '#' | xargs)
# Override DATABASE_URL for local port:
export DATABASE_URL="postgres://salespilot:salespilot@127.0.0.1:5433/salespilot?sslmode=disable"
go run ./apps/api
# → http://localhost:8080/healthz should return {"status":"ok"}
```

### 4. Start the frontend

```bash
cd apps/web
npm install
npm run dev
# → http://localhost:5173
```

---

## Development commands

```bash
# Go
go build ./...                   # compile
go test ./...                    # run tests
go vet ./...                     # vet
golangci-lint run                # lint

# With Makefile (requires GNU make):
make check                       # vet + lint + test
make migrate-up                  # apply migrations (DATABASE_URL must be set)
make migrate-down                # rollback last migration

# Frontend
cd apps/web
npm run dev                      # dev server + proxy
npm run build                    # type-check + bundle
npm run check                    # tsc -b + eslint
npm run lint                     # eslint only
npm run format                   # prettier write
```

---

## Project structure

```
sales_pilot/
├── apps/
│   ├── api/          # Go Echo API (main.go)
│   └── web/          # React + Vite frontend
├── internal/
│   ├── config/       # env loading
│   ├── repository/   # GORM, DB connect
│   └── http/         # router, handlers, DTOs
├── db/migrations/    # golang-migrate SQL files
├── deploy/
│   ├── docker-compose.yml
│   └── hermes/       # Hermes gateway config
├── .env.example      # copy → .env and fill values
└── PRD.md · Design.md · *.plan.md
```

## Health check

```bash
curl http://localhost:8080/healthz
# {"status":"ok"}          — API + DB running
# {"status":"degraded",...} — DB unreachable (503)
```

---

## Upgrading Hermes

The Hermes Agent version is pinned in `services/hermes-bridge/pyproject.toml`
(field `hermes-agent @ git+https://...@<TAG>`). This is the **single source of truth** —
do not change it without running the contract test suite.

### Upgrade procedure

1. Update the pinned tag in `services/hermes-bridge/pyproject.toml`.
2. Rebuild the bridge image:
   ```bash
   docker compose -f deploy/docker-compose.yml build hermes-bridge
   ```
3. Start the bridge and run contract tests against it:
   ```bash
   HERMES_BASE_URL=http://localhost:8642 \
   API_SERVER_KEY=<your-key> \
   WORKSPACE_SESSION_KEY=<your-sk> \
   go test -tags contract ./internal/hermes/...
   ```
4. All tests green → commit the version bump.
   Tests fail → fix **only** `services/hermes-bridge/` (double isolation: Go + React stay untouched).

> **Why double isolation?** `internal/hermes` (ACL Go) speaks only the OpenAI `/v1` contract.
> Changes in Hermes Agent's Python API are absorbed entirely by the bridge.
> See `hermes-bridge.plan.md` for full architecture.
