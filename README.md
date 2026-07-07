# SalesPilot

Internal sales-intelligence platform — an AI agent finds, scores, and helps you win tenders, with a human always in the loop.

## What it is

SalesPilot is an internal, single-company sales/tender-management application. Tenders are scattered across procurement portals and hard to prioritize; this tool centralizes them, uses an AI agent (Hermes) to find and score new opportunities, and provides playbooks and data-driven insights to your team. The app runs three services (React web, Go API, Python AI bridge) plus PostgreSQL.

## Key features

**Implemented:**
- Tender management with a status pipeline (IDENTIFIED → QUALIFYING → BIDDING → SUBMITTED → WON/LOST)
- Event management and convert-to-prospect workflow
- Streaming AI chat assistant with tool awareness and conversation memory
- JWT auth and capability-based RBAC (SALES, OPS, MANAGER, ADMIN roles)
- User management and admin controls

**Planned (not yet available as HTTP routes):**
- Tender discovery via AI crawling
- AI scoring and recommendation endpoints
- Company Knowledge Profile ("Otak Agent")
- Prospect Kanban board
- Playbook and report generators
- Dashboard

## Architecture at a glance

| Component | Tech | Purpose |
|---|---|---|
| Web | React 19 + Vite + Tailwind | Browser SPA (port 5173 dev, port 80 prod) |
| API | Go 1.25 + Echo v4 | CRUD, auth, chat relay (port 8080) |
| hermes-bridge | Python 3.11+ + FastAPI | AI agent wrapper, `/v1` OpenAI-compatible API (port 8642) |
| Postgres 16 | — | Primary datastore |

## Quickstart (Docker)

```bash
# 1. Copy and fill env files (7 required vars: see documentation)
cp .env.example .env
cp deploy/hermes/.env.example deploy/hermes/.env
cp deploy/hermes/config.yaml.example deploy/hermes/config.yaml

# 2. Start the stack
docker compose -f deploy/docker-compose.yml up -d

# 3. Apply database schema (golang-migrate must be installed on host)
migrate -path db/migrations \
  -database "postgres://salespilot:salespilot@127.0.0.1:5432/salespilot?sslmode=disable" up

# 4. Verify
curl http://localhost:8080/healthz         # API health
# → {"status":"ok"}
```

Then open:
- **Web:** http://localhost (port 80)
- **API:** http://localhost:8080

Log in with the `SEED_ADMIN_EMAIL` and `SEED_ADMIN_PASSWORD` from your `.env` file.

## Documentation

- **[Getting Started](docs/getting-started.md)** — full setup guide (Docker + native Windows paths, migrations, troubleshooting)
- **[Configuration](docs/configuration.md)** — complete environment-variable reference
- **[Architecture](docs/architecture.md)** — the three services, request flow, anti-corruption boundary
- **[Data Model](docs/data-model.md)** — entities, enums, tender status machine
- **[API Reference](docs/api-reference.md)** — REST endpoints (implemented only; planned endpoints listed)
- **[Development](docs/development.md)** — dev workflow, commands, testing, conventions
- **[hermes-bridge README](services/hermes-bridge/README.md)** — AI service setup and upgrade guide

For the full documentation index, see [docs/README.md](docs/README.md).

## Tech stack

- **API:** Go 1.25, Echo v4, GORM + Postgres (pgx), JWT auth, bcrypt
- **Web:** React 19, Vite, TypeScript, Tailwind CSS 4, lucide-react, Vitest
- **hermes-bridge:** Python ≥3.11, FastAPI, Uvicorn, Hermes `AIAgent` (pinned git tag)

## License

Proprietary — Copyright (c) 2026 SalesPilot. All Rights Reserved. Internal use only.
