# SalesPilot Documentation

**New to SalesPilot?** Start here: **[Getting Started](getting-started.md)** — a complete setup guide for Docker or native Windows development.

---

## Documentation Index

| Document | Audience | Purpose |
|---|---|---|
| **[Getting Started](getting-started.md)** | First-time setup | Complete setup guide: Docker path, native Windows path, migrations, verification, troubleshooting. Start here. |
| **[Configuration](configuration.md)** | DevOps / deployment | Full reference for all environment variables, secrets, ports, and database URLs. |
| **[Architecture](architecture.md)** | Engineers | The three services (Web, API, hermes-bridge), request flow, the Hermes anti-corruption boundary, layered backend design, auth/RBAC. |
| **[Data Model](data-model.md)** | Engineers / analysts | Database entities (User, Tender, Event, Prospect, Conversation/Message, OutcomeEvent), enums, tender status machine, migrations. |
| **[API Reference](api-reference.md)** | API consumers / frontend devs | REST endpoint reference (implemented only; planned endpoints listed). Includes auth flow, all CRUD routes, chat SSE, and error shapes. |
| **[Development](development.md)** | Contributors | Dev workflow, commands (Go, npm, Python, Make), testing, linting, database migrations, coding conventions, Hermes upgrade runbook. |
| **[hermes-bridge README](../services/hermes-bridge/README.md)** | AI service operators | The Python FastAPI bridge service: setup, endpoints, modes (chat vs. responses), testing, Hermes version upgrades. |

---

## Quick links by role

### I want to **set up and run the app**
→ [Getting Started](getting-started.md)

### I want to **deploy or configure the stack**
→ [Configuration](configuration.md)

### I want to **understand the architecture**
→ [Architecture](architecture.md)

### I want to **consume the API** (frontend or external client)
→ [API Reference](api-reference.md)

### I want to **contribute code**
→ [Development](development.md)

### I want to **work on the AI service** (Hermes bridge)
→ [hermes-bridge README](../services/hermes-bridge/README.md)

### I want to **understand the data model**
→ [Data Model](data-model.md)

---

## Key concepts

- **Single-company workspace:** SalesPilot is internal, not multi-tenant. No `organization_id`.
- **Tender-centric workflow:** Track opportunities from IDENTIFIED through to WON/LOST via a status machine.
- **AI-assisted:** Hermes agent scores tenders, generates playbooks, and learns from outcomes. The bridge isolates the backend from Hermes library changes.
- **JWT auth + RBAC:** Capability-based access control (7 capabilities × 4 roles). Typed tokens (access/refresh, each 15 min / 7 days).
- **Layered backend:** Go packages organized by concern (domain → repository → service → http). No circular deps.

---

## Troubleshooting

**General:** Check [Getting Started → Troubleshooting](getting-started.md#troubleshooting).

**Config/environment:** Check [Configuration](configuration.md).

**Architecture/design:** Check [Architecture](architecture.md).

**API:** Check [API Reference](api-reference.md).

**Dev issues:** Check [Development → Troubleshooting dev issues](development.md#troubleshooting-dev-issues).

**Hermes/bridge:** Check [hermes-bridge README](../services/hermes-bridge/README.md#troubleshooting).

---

## Contributing

Before opening a PR, ensure:
1. All Go tests pass: `make check`
2. Frontend tests pass: `cd apps/web && npm run test`
3. Python tests pass: `cd services/hermes-bridge && pytest`
4. Code follows conventions in [Development → Coding conventions](development.md#coding-conventions)

See [Development](development.md) for the full contributor guide.

---

## License

Proprietary — Copyright (c) 2026 SalesPilot. All Rights Reserved. Internal use only.
