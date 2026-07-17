# Architecture

SalesPilot is a three-service stack: a React web frontend, a Go API backend, and a Python AI bridge service.

## Target architecture: Hermes as AI engine, Web App as orchestrator

Hermes Agent is **not** being replaced or reimplemented by this project — it stays the unmodified, fully-capable agentic engine (TUI, tools, memory, workflows) that Nous Research ships. SalesPilot (Go API + React) is a **separate orchestration/business application** in front of it:

- **Web App responsibilities:** authentication, database, CRUD, business logic, role & permission (RBAC), audit trail, logging, configuration, and the integration glue to Hermes.
- **Prompt engineering lives entirely in the backend.** Users never prompt Hermes directly — they fill a form or click a button; the Go API (`internal/ai/scoring.go`, `playbook.go`, `report.go`, `learning.go`) composes the optimal prompt, calls Hermes via the ACL, persists the result to Postgres, and the React frontend just renders it. This keeps prompting consistent, efficient, accurate, and independent of end-user prompting skill.
- **Hermes is treated like an AI employee**, called on demand by the orchestrator — not a chat partner the user talks to freely (the one exception is the mediated Chat feature, EP-04, which still flows through hermes-bridge's session/memory model rather than a raw Hermes session).
- **`hermes-tui` is an admin-only escape hatch, not a rebuild.** The Settings → Hermes TUI tab embeds/proxies the real, unmodified Hermes CLI (via a `ttyd` sidecar, see `services/hermes-tui/`) so an administrator can reach the full native tool when needed (e.g. `hermes auth add codex --type oauth`). It is not a replacement UI and is not part of the primary user flow.

This is not a future plan — EP-01 through EP-18 (see `epic.plan.md`) already implement this split in full.

## Overview

| Service | Tech | Role |
|---|---|---|
| **Web** | React 19 + Vite | Browser SPA; handles login, tender/event/prospect management, chat UI |
| **API** | Go 1.25 + Echo v4 | REST backend; CRUD, auth, SSE chat relay, MCP data tools, Postgres integration |
| **hermes-bridge** | Python 3.11+ + FastAPI | AI service; wraps the Hermes `AIAgent`, exposes OpenAI-compatible `/v1` API |
| **Postgres 16** | — | Primary datastore; shared by API and seeded with migrations |

## Request diagram

```
┌──────────────────────────────────────────────────────────────────┐
│ Browser                                                          │
└────────────────────────────────┬─────────────────────────────────┘
                                 │
                   ┌─────────────▼──────────────┐
                   │  Vite (dev) / nginx (prod) │
                   │  Port 5173 / 80            │
                   └─────────────┬──────────────┘
                                 │
                   ┌─────────────▼──────────────┐
                   │  Go API                    │
                   │  Echo + GORM + Postgres    │
                   │  Port 8080                 │
                   │                            │
                   │  ├─ CRUD routes            │
                   │  ├─ Auth (JWT)             │
                   │  ├─ Chat SSE relay         │
                   │  └─ MCP data tools (/mcp)  │
                   └────────────┬───────────────┘
                                │
                ┌───────────────┼───────────────┐
                │               │               │
        ┌───────▼────────┐  ┌───▼─────────┐  ┌─▼──────────────┐
        │ Postgres 16    │  │ hermes-      │  │ External LLM   │
        │                │  │ bridge       │  │ (OpenAI /      │
        │ Tables:        │  │ FastAPI      │  │  OpenRouter)   │
        │ ├─ user        │  │ Port 8642    │  │                │
        │ ├─ tender      │  │              │  └────────────────┘
        │ ├─ event       │  │ ├─ AIAgent   │
        │ ├─ conversation│  │ ├─ /v1/chat  │
        │ └─ ...         │  │ └─ /admin/   │
        │                │  │    config    │
        └────────────────┘  └──────────────┘
```

## Chat message flow (SSE)

1. **Browser:** Opens SSE POST request to `/api/conversations/:id/chat` with the user message.
2. **API:** Persists the user message to `conversation.message` table.
3. **API:** Calls hermes-bridge POST `/v1/chat/completions` (stream mode) with conversation history + `X-Hermes-Session-Id` header.
4. **Bridge:** Instantiates a fresh `AIAgent` in chat mode (memory ON, toolsets ON), runs the conversation against the LLM provider.
5. **Bridge:** Streams sentence-chunked text deltas back to API (chunked HTTP/2).
6. **API:** Relays SSE frames to browser:
   - `data: {"type":"delta","content":"..."}`
   - `data: {"type":"tool_call","id":"...","name":"...","arguments":...}`
   - `data: {"type":"done"}` and `data: [DONE]` (terminator)
7. **API (best-effort):** Persists the assembled assistant message to `conversation.message` with a detached context (survives client disconnect).
8. **Browser:** Renders deltas in real time; stops at `[DONE]`.

If stream fails to start (bridge unavailable, auth denied, etc.) → API returns JSON error `{"error":{"code":"AI_UNAVAILABLE",...}}` (HTTP 400).

## Hermes anti-corruption boundary

The **hermes-bridge** service exists to isolate the Go backend from internal changes to the Hermes `AIAgent` library. This is the anti-corruption boundary:

- **Go API** only depends on the **OpenAI-compatible `/v1` surface** of hermes-bridge:
  - `/v1/chat/completions` — chat requests
  - `/v1/responses` — deterministic JSON (used for tender scoring)
  - `/v1/capabilities` — health check
  - `/admin/config` — provider/model setup

- **Hermes version is pinned** in `services/hermes-bridge/pyproject.toml`:
  ```
  hermes-agent @ git+https://github.com/NousResearch/hermes-agent.git@v2026.6.19
  ```
  This is the **single source of truth**. Changes to the Hermes library are absorbed by the bridge; the Go side sees no change.

- **Contract tests** in `internal/hermes/contract_test.go` verify that the bridge's `/v1` surface conforms to the OpenAI spec. Upgrading Hermes requires:
  1. Bump the tag in `pyproject.toml`.
  2. Run contract tests against the new bridge.
  3. Only commit if tests pass; failure signals a breaking change in the Hermes library's `/v1` behavior.

This **double isolation** protects the Go and React layers: only the bridge is rebuilt and redeployed.

## Backend architecture: layered packages

Go API is organized in layers:

```
domain/          Domain models (User, Tender, Event, etc.) + interfaces
                 ├─ No external dependencies; high cohesion
                 └─ Defines the data model and contract

repository/      Data access layer (GORM + Postgres)
                 ├─ Implements domain interfaces (UserRepository, TenderRepository, etc.)
                 └─ Handles schema and query construction

service/         Business logic layer
                 ├─ Implements domain services (AuthService, TenderService, etc.)
                 ├─ Orchestrates repository + external calls
                 └─ Enforces invariants (status transitions, RBAC, etc.)

http/            HTTP layer
├─ handlers/     Request handlers (receive, validate, transform, respond)
├─ dto/          Request/response DTOs (serializable)
├─ httperr/      Standardized error response shape
└─ router.go     Route registration + middleware

auth/            Authentication & RBAC
├─ jwt.go        JWT token signing/verification (typed tokens)
├─ middleware.go Request auth guard
├─ password.go   Password hashing + validation
└─ rbac.go       Capability-based RBAC

hermes/          AI service client
├─ client.go     HTTP wrapper for `/v1` calls
└─ contract_test.go Contract tests vs. the bridge

config/          Environment loading + validation
pagination/      Page/page_size normalization
```

No circular dependencies; higher layers depend only on lower layers.

## Authentication & RBAC

### JWT tokens (typed, short-lived)

- **Access token:** TTL 15 minutes. Sent as `Authorization: Bearer <token>` on every request. Typed as `typ=access`.
- **Refresh token:** TTL 7 days. Sent in POST `/api/auth/refresh` to mint a new token pair. Typed as `typ=refresh`.
- **Token type check:** A refresh token **cannot** be replayed as an access token and vice versa (they are cryptographically distinct).

### Capability-based RBAC

Four roles: `SALES`, `OPS`, `MANAGER`, `ADMIN`.

Each endpoint requires a **capability**, not a raw role. Capabilities are granted to roles:

| Capability | SALES | OPS | MANAGER | ADMIN |
|---|:--:|:--:|:--:|:--:|
| `ViewData` | ✅ | ✅ | ✅ | ✅ |
| `CRUDData` | ✅ | ✅ | ✅ | ✅ |
| `EditProfile` | ❌ | ✅ | ✅ | ✅ |
| `RunDiscovery` | ❌ | ✅ | ✅ | ✅ |
| `UseAI` | ✅ | ✅ | ✅ | ✅ |
| `MakeDecision` | ✅ | ✅ | ✅ | ✅ |
| `ManageUsers` | ❌ | ❌ | ❌ | ✅ |

Enforcement is done via `auth.RequireCapability(cap)` middleware, which returns 403 Forbidden if the user's role lacks the capability.

The frontend mirrors this check (`apps/web/src/lib/rbac.ts`) to gate navigation items.

## Data flow: tender management

1. **User (SALES/OPS/MANAGER)** creates a tender via POST `/api/tenders`.
2. **API (service layer)** validates inputs, ensures `status=IDENTIFIED` by default.
3. **API (repository)** inserts into `tender` table.
4. **User (OPS only)** can run discovery or promote a `discovery` tender to `QUALIFYING`.
5. **User** transitions tender through the status pipeline via PATCH `/api/tenders/:id/status` (enforced state machine).
6. **User (all roles)** records an outcome (WON/LOST) via POST `/api/tenders/:id/outcome`.
7. **API (background)** notifies the learning hook (placeholder; future EP-16).

All mutations are gated by capability and RBAC.

## Security model

- **No multi-tenancy:** Single company, single workspace. No `organization_id` or row-level authorization.
- **No public signup:** Accounts are created by ADMIN only. Initial admin is seeded on boot.
- **SQL injection:** All queries use parameterized statements (GORM `?` binding; no string concatenation).
- **Token expiry:** Access tokens have a short TTL (15 min); refresh tokens allow a grace period (7 days) before re-login is required.
- **Password hashing:** bcrypt with cost 12; max 72-byte password length.
- **SSRF protection:** Hermes bridge admin config restricts `base_url` to approved domains (openai.com, openrouter.ai).
- **MCP/data tools:** Authenticated with bearer token (`SALES_MCP_TOKEN`); constant-time comparison prevents timing attacks.

## Observability

- **Health endpoint:** `GET /healthz` (no auth) returns `{"status":"ok"}` or `{"status":"degraded","db":"..."}` (HTTP 503 if DB is down).
- **Logging:** Go API logs startup config (with secrets redacted), request IDs, errors. hermes-bridge logs upstream LLM calls.
- **Error shape:** All errors conform to `{"error":{"code":"...","message":"..."}}` for client consistency.

## Deployment topology

**Docker Compose (all-in-one):**
- Services communicate via compose network hostnames (`postgres`, `api`, `hermes-bridge`).
- Postgres is **not** published to host; only API and hermes-bridge publish their ports.
- Migrations must be run on the host (requires `golang-migrate` CLI).

**Native dev (Windows):**
- Postgres runs in Docker on host port 5433.
- API and hermes-bridge run locally (`:8080`, `:8642`).
- Vite dev server on `:5173` proxies `/api` → API.

**Production (not in this repo):**
- Services run in Kubernetes / cloud platform.
- Hermes version is gated by contract tests and must be vetted before deployment.
