# Configuration

Reference for all environment variables and configuration options.

## Root `.env`

Copy `.env.example` to `.env` and fill in the required variables.

| Variable | Required? | Default | Used by | Description |
|---|---|---|---|---|
| `DATABASE_URL` | **YES** | — | API | PostgreSQL connection string. Docker: `postgres://salespilot:salespilot@postgres:5432/salespilot?sslmode=disable`. Native: `postgres://salespilot:salespilot@127.0.0.1:5433/salespilot?sslmode=disable`. |
| `JWT_SECRET` | **YES** | — | API | Secret key for signing JWT tokens. Generate: `openssl rand -hex 32`. |
| `API_SERVER_KEY` | **YES** | — | API, hermes-bridge | Bearer token for hermes-bridge API authentication. Generate: `openssl rand -hex 32`. Must match `deploy/hermes/.env`. |
| `WORKSPACE_SESSION_KEY` | **YES** | — | API | Session key for Hermes workspace context. Generate: `openssl rand -hex 32`. |
| `SALES_MCP_TOKEN` | **YES** | — | API, hermes-bridge | Bearer token for MCP (Model Context Protocol) data tools. Generate: `openssl rand -hex 32`. Must match `deploy/hermes/.env`. |
| `SEED_ADMIN_EMAIL` | **YES** | — | API | Email address of the initial admin user (seeded on boot). Example: `admin@company.com`. |
| `SEED_ADMIN_PASSWORD` | **YES** | — | API | Password for the initial admin user. Use a strong password. |
| `PORT` | No | `8080` | API | Port the API binds to. |
| `HERMES_BASE_URL` | No | `http://localhost:8642` | API | URL of the hermes-bridge service (used by API to call the AI bridge). |
| `HERMES_MODEL` | No | `default` | hermes-bridge | AI model to use (e.g., `gpt-4o`, `anthropic/claude-sonnet-4-6`). |
| `OPENAI_API_KEY` | No | — | hermes-bridge | OpenAI API key (if using OpenAI as provider). |
| `OPENROUTER_API_KEY` | No | — | hermes-bridge | OpenRouter API key (if using OpenRouter as provider). |
| `POSTGRES_USER` | No | `salespilot` | Docker compose | Postgres username (compose only). |
| `POSTGRES_PASSWORD` | No | `salespilot` | Docker compose | Postgres password (compose only). |
| `POSTGRES_DB` | No | `salespilot` | Docker compose | Postgres database name (compose only). |

## hermes-bridge `.env` (`deploy/hermes/.env`)

Copy `deploy/hermes/.env.example` to `deploy/hermes/.env`.

| Variable | Required? | Default | Description |
|---|---|---|---|
| `API_SERVER_KEY` | **YES** | — | Bearer token for API authentication. Must **match** the value in root `.env`. |
| `SALES_MCP_TOKEN` | **YES** | — | Bearer token for MCP tools. Must **match** the value in root `.env`. |
| `PORT` | No | `8642` | Port the bridge binds to. |
| `HERMES_MODEL` | No | `default` | AI model (e.g., `gpt-4o`). Overrides the root `.env` value if set. |
| `OPENAI_API_KEY` | No | — | OpenAI provider key (if using OpenAI). |
| `OPENROUTER_API_KEY` | No | — | OpenRouter provider key (if using OpenRouter). |

## Hermes gateway config (`deploy/hermes/config.yaml`)

Copy `deploy/hermes/config.yaml.example` to `deploy/hermes/config.yaml`.

This YAML file configures:
- The MCP (Model Context Protocol) `sales` server endpoint (`url: http://api:8080/mcp`, bearer token, tool whitelist)
- Memory provider (`holographic`)
- Workspace session context injection

**Known limitation:** The current `docker-compose.yml` does not mount `config.yaml` into the bridge container, so this file is not yet active. It is documented for future use.

## Ports

| Component | Port | Context | Notes |
|---|---|---|---|
| Go API | 8080 | Both | Binds to `:8080`; Docker compose exposes `8080:8080`; Vite/nginx proxy target. |
| Web (Vite dev) | 5173 | Native dev only | Dev server; proxies `/api/*` → `http://localhost:8080`. |
| Web (nginx prod) | 80 | Docker | Production; served from built static files; proxies `/api/*` → Go API. |
| hermes-bridge | 8642 | Both | Default; set `PORT=8642` in `deploy/hermes/.env` if changing. |
| Postgres (Docker Compose) | 5432 | Compose internal | Not published to host; internal to compose network. API connects via `postgres:5432`. |
| Postgres (native dev) | 5433 → 5432 | Native dev only | Run `docker run -p 5433:5432 postgres:16`. API's `DATABASE_URL` uses `127.0.0.1:5433`. |

## Secrets guidance

Generate all secret values with:

```bash
openssl rand -hex 32
```

This produces a 64-character hex string (~256 bits of entropy).

Required secrets:
- `JWT_SECRET` — Access and refresh token signing key. Each token must be distinct; tokens are typed so one cannot be replayed as the other.
- `API_SERVER_KEY` — Used by hermes-bridge and API to authenticate each other. Must be identical in root `.env` and `deploy/hermes/.env`.
- `WORKSPACE_SESSION_KEY` — Workspace-wide context key passed to Hermes for conversation isolation.
- `SALES_MCP_TOKEN` — Bearer token for MCP tools (data access). Must be identical in root `.env` and `deploy/hermes/.env`.

**Never:**
- Commit `.env` files (they are in `.gitignore`)
- Log or expose these secrets
- Reuse the same secret for multiple variables

## Example full `.env`

```bash
# Database (required)
DATABASE_URL=postgres://salespilot:salespilot@postgres:5432/salespilot?sslmode=disable

# Secrets (required)
JWT_SECRET=a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0e1f
API_SERVER_KEY=b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0e1f2
WORKSPACE_SESSION_KEY=c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0e1f2g3
SALES_MCP_TOKEN=d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0e1f2g3h4

# Admin user (required)
SEED_ADMIN_EMAIL=admin@company.com
SEED_ADMIN_PASSWORD=YourStrongPassword123!

# Optional: API port (default 8080)
PORT=8080

# Optional: Hermes bridge (default http://localhost:8642)
HERMES_BASE_URL=http://hermes-bridge:8642

# Optional: AI provider (pick one)
HERMES_MODEL=gpt-4o
OPENAI_API_KEY=sk-...
# OR:
# OPENROUTER_API_KEY=sk-or-...
```

## Database URL formats

### Docker Compose (host: postgres service)

```
postgres://salespilot:salespilot@postgres:5432/salespilot?sslmode=disable
```

- `postgres` — hostname (Docker network)
- `5432` — standard Postgres port (inside container)

### Native dev on Windows (local Postgres)

```
postgres://salespilot:salespilot@127.0.0.1:5433/salespilot?sslmode=disable
```

- `127.0.0.1` — localhost
- `5433` — host port (maps to container 5432: `docker run -p 5433:5432 postgres:16`)

## Env vars are loaded at startup

- **Go API** (`main.go`): calls `config.MustLoad()` on boot. Missing required vars → `log.Fatal`.
- **hermes-bridge** (`app/config.py`): missing `API_SERVER_KEY` → `RuntimeError`.
- Changes to `.env` require a restart.
