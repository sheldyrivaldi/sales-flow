# Getting Started

This guide walks you through setting up and running SalesPilot locally.

## Prerequisites

| Tool | Version | For | Install |
|---|---|---|---|
| Docker Desktop | current | Docker path | [docker.com](https://www.docker.com/products/docker-desktop) |
| Go | 1.25 | native API dev | [go.dev](https://go.dev/dl/) |
| Node.js | 24 | native web dev | [nodejs.org](https://nodejs.org) |
| Python | ≥3.11 | native hermes-bridge | [python.org](https://www.python.org/downloads/) |
| golang-migrate | latest | migrations | `go install -tags 'pgx5' github.com/golang-migrate/migrate/v4/cmd/migrate@latest` |
| golangci-lint | latest | linting | `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest` |
| GNU make | optional | shortcuts | `winget install ezwinports.make` or `choco install make` |

**Windows PATH:** Add `%GOPATH%\bin` (default: `C:\Users\<you>\go\bin`) to your PATH so `migrate` and `golangci-lint` are available globally.

**Make on Windows:** Run `make` commands from Git Bash, not MSYS2.

## Environment setup

### 1. Copy env files

```bash
cp .env.example .env
cp deploy/hermes/.env.example deploy/hermes/.env
cp deploy/hermes/config.yaml.example deploy/hermes/config.yaml
```

### 2. Fill required variables in `.env`

The 7 **required** variables (app will crash if empty):

| Variable | Example value |
|---|---|
| `DATABASE_URL` | `postgres://salespilot:salespilot@postgres:5432/salespilot?sslmode=disable` |
| `JWT_SECRET` | `openssl rand -hex 32` |
| `API_SERVER_KEY` | `openssl rand -hex 32` |
| `WORKSPACE_SESSION_KEY` | `openssl rand -hex 32` |
| `SALES_MCP_TOKEN` | `openssl rand -hex 32` |
| `SEED_ADMIN_EMAIL` | your admin email (e.g., `admin@company.com`) |
| `SEED_ADMIN_PASSWORD` | a strong password for the admin user |

Generate secure secrets with: `openssl rand -hex 32`

Optional variables (with defaults):
- `PORT` — API port (default 8080)
- `HERMES_BASE_URL` — Hermes bridge URL (default `http://localhost:8642`)
- `HERMES_MODEL` — AI model (e.g., `gpt-4o`, `anthropic/claude-sonnet-4-6`)
- `OPENAI_API_KEY` or `OPENROUTER_API_KEY` — provider key for AI features

See [Configuration](configuration.md) for the full reference.

### 3. Fill Hermes config

In `deploy/hermes/.env`:
- `API_SERVER_KEY` — must **match** the value in root `.env`
- `SALES_MCP_TOKEN` — must **match** the value in root `.env`

In `deploy/hermes/config.yaml`:
- This file defines the MCP server and toolsets; no edits needed unless you're customizing tool access.

## Path A — Run with Docker (recommended)

### 1. Start the stack

```bash
docker compose -f deploy/docker-compose.yml up -d
```

This starts: `postgres` (16), `api` (Go, port 8080), `web` (nginx, port 80), `hermes-bridge` (Python, port 8642).

### 2. Apply database schema

```bash
migrate -path db/migrations \
  -database "postgres://salespilot:salespilot@127.0.0.1:5432/salespilot?sslmode=disable" up
```

**Important:** Migrations are **not automatic**. If you skip this step, the API will fail on startup.

### 3. Verify

```bash
curl http://localhost:8080/healthz
```

Expected response:
- `{"status":"ok"}` — API and database are running
- `{"status":"degraded","db":"down"}` — API is up but database is unavailable

### 4. Log in

Open http://localhost in your browser. Log in with the `SEED_ADMIN_EMAIL` and `SEED_ADMIN_PASSWORD` you set in `.env`.

## Path B — Run natively (Windows)

### 1. Start Postgres

```bash
docker run --name salespilot-pg \
  -e POSTGRES_USER=salespilot \
  -e POSTGRES_PASSWORD=salespilot \
  -e POSTGRES_DB=salespilot \
  -p 5433:5432 -d postgres:16
```

This starts Postgres on **host port 5433** (to avoid conflicts).

### 2. Run migrations

```bash
export DATABASE_URL="postgres://salespilot:salespilot@127.0.0.1:5433/salespilot?sslmode=disable"
migrate -path db/migrations -database "$DATABASE_URL" up
```

### 3. Start the API

```bash
# Load .env variables (from root of the repo):
export $(cat .env | grep -v '#' | xargs)
# Override DATABASE_URL for the local Postgres:
export DATABASE_URL="postgres://salespilot:salespilot@127.0.0.1:5433/salespilot?sslmode=disable"
# Start the API:
go run ./apps/api
```

Verify: `curl http://localhost:8080/healthz` should return `{"status":"ok"}`.

### 4. Start the frontend

```bash
cd apps/web
npm install
npm run dev
```

Opens http://localhost:5173 (dev server, proxies `/api` calls to `:8080`).

## Running hermes-bridge (AI service)

The AI service is optional. CRUD features work without it (AI features degrade gracefully).

### Run natively

```bash
cd services/hermes-bridge
pip install .                    # install from pyproject.toml
export API_SERVER_KEY=...        # must match root .env
python -m app.main               # → :8642
```

### Run via Docker Compose

Already running as part of `docker compose up` (no extra steps needed).

Note: Set `OPENAI_API_KEY` or `OPENROUTER_API_KEY` in `.env` to enable AI. Without a provider key, the bridge boots but AI requests fail gracefully.

## First login

An **admin user** is automatically seeded on boot using `SEED_ADMIN_EMAIL` and `SEED_ADMIN_PASSWORD` from your `.env`.

- **Email:** the value from `SEED_ADMIN_EMAIL`
- **Password:** the value from `SEED_ADMIN_PASSWORD`

After logging in, you can create additional users from the settings page (or via the `/api/users` endpoint).

No public signup — this is an internal tool.

## Verifying the install

### API health

```bash
curl http://localhost:8080/healthz
```

Responses:
- `{"status":"ok"}` — API + database running normally
- `{"status":"degraded","db":"down"}` — API is running but database is unreachable (HTTP 503)

### Web

Open http://localhost (Docker) or http://localhost:5173 (native dev Vite) and log in.

### Chat (optional)

If hermes-bridge is running and has a provider key:
1. Go to the Chat page.
2. Start a conversation; messages should stream in.

If you see "Agent tidak tersedia" (Agent unavailable), either:
- hermes-bridge is not running, or
- `OPENAI_API_KEY` / `OPENROUTER_API_KEY` is not set in `.env`

CRUD (tenders, events) will still work.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| `config: required env <NAME> is empty` | A required env var is missing from `.env` | Check all 7 required vars are filled; see Environment setup above |
| Queries fail with `column does not exist` | Database schema not applied | Run `migrate -path db/migrations -database "$DATABASE_URL" up` |
| Connection refused | Postgres not running | For Docker: `docker compose up -d postgres`. For native: start the postgres container on port 5433 |
| Postgres connection to `localhost:5432` | Wrong port in Docker native setup | Use `127.0.0.1:5433` (host port 5433 maps to container 5432) |
| "Agent tidak tersedia" in chat | hermes-bridge down or no provider key | Set `OPENAI_API_KEY` or `OPENROUTER_API_KEY`; start bridge if needed. CRUD still works |
| Port 8080 / 5173 already in use | Another app is using the port | Change `PORT` in `.env` (API) or run web on a different port with `npm run dev -- --port 3000` |

## Next steps

- [API Reference](api-reference.md) — learn the REST endpoints
- [Configuration](configuration.md) — deep dive into environment variables
- [Architecture](architecture.md) — understand the three services
- [Development](development.md) — set up your dev workflow
