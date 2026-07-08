# Data Model

## Overview

All entities have `id` (UUID), `created_at` (TIMESTAMPTZ), and `updated_at` (TIMESTAMPTZ) fields. There is no `organization_id` — this is a single-company workspace. GORM table names are singular (e.g., `user`, `tender`, not `users`, `tenders`).

---

## User

Table: `user`

| Field | Type | Notes |
|---|---|---|
| `id` | UUID | Primary key |
| `email` | TEXT | Unique, not null |
| `password_hash` | TEXT | JSON-hidden in responses |
| `name` | TEXT | Display name |
| `role` | TEXT | SALES, OPS, MANAGER, or ADMIN |
| `active` | BOOLEAN | Default true; deactivated users can't log in |
| `created_at` | TIMESTAMPTZ | — |
| `updated_at` | TIMESTAMPTZ | — |

**Enum: `Role`**
- `SALES`
- `OPS`
- `MANAGER`
- `ADMIN`

Passwords are bcrypt-hashed with cost 12; max 72 bytes. An initial admin user is seeded on boot from `SEED_ADMIN_EMAIL` and `SEED_ADMIN_PASSWORD`.

---

## Tender

Table: `tender` (central entity)

| Field | Type | Notes |
|---|---|---|
| `id` | UUID | Primary key |
| `title` | TEXT | Not null |
| `buyer_name` | TEXT | — |
| `buyer_country` | TEXT | — |
| `buyer_industry` | TEXT | — |
| `value_estimate` | DECIMAL | In the specified currency |
| `currency` | TEXT | Default `IDR` |
| `published_date` | TIMESTAMPTZ | — |
| `submission_deadline` | TIMESTAMPTZ | — |
| `source_name` | TEXT | e.g., "LKPP", "Company X Portal" |
| `source_url` | TEXT | — |
| `service_category` | TEXT | e.g., "IT Services", "Construction" |
| `scope_summary` | TEXT | — |
| `eligibility_requirements` | TEXT | — |
| `technical_requirements` | TEXT | — |
| `status` | TEXT | Default `IDENTIFIED`; see status machine below |
| `fit_score` | INT | 0–100, set by AI scoring (planned) |
| `recommended_action` | TEXT | PURSUE, REVIEW, WATCHLIST, REJECT, NEED_PARTNER |
| `risk_flags` | JSONB | Array of risk identifiers |
| `reasoning_summary` | TEXT | AI-generated explanation |
| `dedup_key` | TEXT | For deduplication across crawling runs |
| `origin` | TEXT | `manual` or `discovery` (set by crawling) |
| `created_at` | TIMESTAMPTZ | — |
| `updated_at` | TIMESTAMPTZ | — |

**Enums:**
- **`TenderStatus`:** `IDENTIFIED`, `QUALIFYING`, `BIDDING`, `SUBMITTED`, `WON`, `LOST`
- **`RecommendedAction`:** `PURSUE`, `REVIEW`, `WATCHLIST`, `REJECT`, `NEED_PARTNER`
- **`TenderOrigin`:** `manual`, `discovery`

### Tender Status Machine

```
IDENTIFIED ──→ QUALIFYING ──→ BIDDING ──→ SUBMITTED ──→ WON
    │              │             │            │
    └──────────────┴─────────────┴────────────┴──→ LOST

WON and LOST are terminal states (no outgoing transitions).
```

Operations on status:
- **Transition:** `PATCH /api/tenders/:id/status` validates state machine rules. Invalid transitions return 400 Bad Request.
- **Promote:** A discovery tender in `IDENTIFIED` can be promoted to `QUALIFYING` via `POST /api/tenders/:id/promote`.
- **Outcome:** Record a `WON` or `LOST` result via `POST /api/tenders/:id/outcome`, which sets the terminal status and creates an `OutcomeEvent`.

---

## Event

Table: `event`

| Field | Type | Notes |
|---|---|---|
| `id` | UUID | Primary key |
| `name` | TEXT | Not null |
| `type` | TEXT | EXPO, CONFERENCE, SEMINAR, WORKSHOP, NETWORKING, OTHER |
| `date` | TIMESTAMPTZ | Column name: `event_date` |
| `location` | TEXT | — |
| `organizer` | TEXT | — |
| `notes` | TEXT | — |
| `status` | TEXT | PLANNED, ATTENDED, CANCELLED (default PLANNED) |
| `created_at` | TIMESTAMPTZ | — |
| `updated_at` | TIMESTAMPTZ | — |

**Enums:**
- **`EventType`:** `EXPO`, `CONFERENCE`, `SEMINAR`, `WORKSHOP`, `NETWORKING`, `OTHER`
- **`EventStatus`:** `PLANNED`, `ATTENDED`, `CANCELLED`

Events can be converted to Prospects via `POST /api/events/:id/convert`.

---

## Prospect

Table: `prospect`

| Field | Type | Notes |
|---|---|---|
| `id` | UUID | Primary key |
| `name` | TEXT | Not null |
| `company` | TEXT | — |
| `contact_info` | TEXT | Phone, email, or other |
| `source_type` | TEXT | `manual`, `event`, `tender` (default `manual`) |
| `source_id` | UUID | Points to the source event or tender |
| `stage` | TEXT | NEW, QUALIFIED, ENGAGED, PROPOSAL, WON, LOST (default `NEW`) |
| `est_value` | DECIMAL | Estimated contract value |
| `owner_user_id` | UUID | Foreign key to `user.id` |
| `created_at` | TIMESTAMPTZ | — |
| `updated_at` | TIMESTAMPTZ | — |

**Enums:**
- **`ProspectStage`:** `NEW`, `QUALIFIED`, `ENGAGED`, `PROPOSAL`, `WON`, `LOST`
- **`ProspectSource`:** `manual`, `event`, `tender`

Current repository methods: `Create`, `GetByID`, `GetBySource`. List/Update/Delete are not yet implemented.

---

## Conversation & Message

### Conversation

Table: `conversation`

| Field | Type | Notes |
|---|---|---|
| `id` | UUID | Primary key |
| `owner_user_id` | UUID | Foreign key to `user.id` (not null) |
| `title` | TEXT | Conversation title (auto-derived from first message if empty) |
| `session_key` | TEXT | Workspace-wide session key (hidden in responses) |
| `hermes_session_id` | TEXT | Per-conversation session ID for Hermes; UUIDv4 |
| `created_at` | TIMESTAMPTZ | — |
| `updated_at` | TIMESTAMPTZ | — |

### Message

Table: `message`

| Field | Type | Notes |
|---|---|---|
| `id` | UUID | Primary key |
| `conversation_id` | UUID | Foreign key to `conversation.id` (not null); ON DELETE CASCADE |
| `role` | TEXT | `user`, `assistant`, `system`, `tool` |
| `content` | TEXT | Message text (empty for tool calls) |
| `tool_calls` | JSONB | Array of ToolCall objects (nullable) |
| `created_at` | TIMESTAMPTZ | — |
| `updated_at` | TIMESTAMPTZ | — |

**Enum: `MessageRole`**
- `user`
- `assistant`
- `system`
- `tool`

Messages are stored atomically; the chat SSE relay persists both user and assistant messages.

---

## OutcomeEvent

Table: `outcome_event`

| Field | Type | Notes |
|---|---|---|
| `id` | UUID | Primary key |
| `target_type` | TEXT | `tender` or `prospect` |
| `target_id` | UUID | ID of the tender or prospect |
| `result` | TEXT | `WON` or `LOST` |
| `notes` | TEXT | Free-text outcome notes |
| `created_at` | TIMESTAMPTZ | — |
| `updated_at` | TIMESTAMPTZ | — |

**Enum: `OutcomeResult`**
- `WON`
- `LOST`

**Enum: `OutcomeTargetType`**
- `tender`
- `prospect`

When a tender or prospect closes with an outcome, an `OutcomeEvent` is created. A background goroutine notifies the learning hook (currently a no-op placeholder; future EP-16 feeds this to Hermes memory).

---

## CompanyProfile & knowledge base ("Otak Agent")

The knowledge base guides discovery (EP-12) and scoring (EP-10). `CompanyProfile`
is the versioned root; `TargetCriteria`, `NoGoRule`, and `KeywordSet` are its
children, each carrying a `profile_id` FK. Every `PUT /api/profile` clones the
current version's children into a brand-new version (full snapshot) rather
than mutating them in place, so prior versions remain queryable as history.
Only one `CompanyProfile` row has `is_current=true` at a time, enforced by a
partial unique index — `ProfileRepo.CreateVersion` flips the old row to
`is_current=false` before inserting the new one, inside a transaction.

### CompanyProfile

Table: `company_profile`

| Field | Type | Notes |
|---|---|---|
| `id` | UUID | Primary key |
| `company_name` | TEXT | Not null |
| `one_liner` | TEXT | — |
| `service_categories` | JSONB | Array of strings |
| `tech_stack` | JSONB | Array of strings |
| `source_doc_refs` | JSONB | Array of strings (PDF ingest refs, EP-13) |
| `version` | INT | Increments on every `PUT` |
| `is_current` | BOOLEAN | Exactly one `true` row at a time (partial unique index) |
| `created_at` / `updated_at` | TIMESTAMPTZ | — |

### TargetCriteria

Table: `target_criteria` — one row per `CompanyProfile` version (`profile_id` unique).

| Field | Type | Notes |
|---|---|---|
| `countries` | JSONB | Array of strings; default `["Indonesia"]` |
| `industries` | JSONB | Array of strings |
| `value_min` / `value_ideal` / `value_max` | NUMERIC | ≥ 0; default `value_min` = 1,000,000,000 (Rp 1 miliar) |
| `currency` | TEXT | Default `IDR` |
| `deadline_min_days` | INT | ≥ 0; default 7 |
| `procurement_types` | JSONB | Array of strings; default preset (Barang, Jasa Konsultansi, Jasa Lainnya, Pekerjaan Konstruksi) |

### NoGoRule

Table: `nogo_rule` — one row per `CompanyProfile` version (`profile_id` unique).

| Field | Type | Notes |
|---|---|---|
| `preset_flags` | JSONB | Array of strings |
| `custom` | JSONB | Array of free-text rules |

### KeywordSet

Table: `keyword_set` — zero or more rows per `CompanyProfile` version (one per category).

| Field | Type | Notes |
|---|---|---|
| `category` | TEXT | Nullable |
| `keywords` | JSONB | Array of strings |
| `negative_keywords` | JSONB | Array of strings |
| `language` | TEXT | Default `id` |

### Source

Table: `source` — global, **not** versioned with `CompanyProfile`.

| Field | Type | Notes |
|---|---|---|
| `name` | TEXT | Not null |
| `url` | TEXT | Not null; validated as a URL on write |
| `country` | TEXT | — |
| `access` | TEXT | `publik`, `login`, or `manual` (default `publik`) |
| `legal_note` | TEXT | — |
| `enabled` | BOOLEAN | Default false |
| `priority` | INT | Default 0 |
| `preset_key` | TEXT | Set when the row came from the hardcoded Indonesia preset catalog; unique when non-null (makes preset activation idempotent) |

**Enum: `SourceAccess`** — `publik`, `login`, `manual`. Sources with `access != publik` are marked but not auto-crawled (PRD §9 compliance).

The preset catalog (`GET /api/sources/presets`) is hardcoded in
`internal/service/source_service.go`: SPSE/Inaproc (LKPP), eProc PLN, eProc
Pertamina, Telkom SMILE, PaDi UMKM. `POST /api/sources/presets {key}` activates
one — idempotent: re-activating an already-active preset just re-asserts
`enabled=true` on the existing row instead of creating a duplicate.

Endpoints: `GET/PUT /api/profile` (RBAC: read = all roles, write = `EditProfile`
i.e. OPS/MANAGER/ADMIN); `GET/POST/PUT/DELETE /api/sources` + `GET/POST
/api/sources/presets` (RBAC: read = all roles, write = `EditProfile`).

---

## Migrations

Migrations are applied **manually** (not automatically on boot) via golang-migrate CLI:

```bash
migrate -path db/migrations -database "$DATABASE_URL" up
```

Migration files (in order):

1. **0001_init** — Creates `pgcrypto` extension for `gen_random_uuid()`.
2. **0002_users** — Creates `user` table + indexes.
3. **0003_chat** — Creates `conversation` and `message` tables + indexes.
4. **0004_tenders** — Creates `tender` table + indexes.
5. **0005_outcome_events** — Creates `outcome_event` table + indexes.
6. **0006_events** — Creates `event` table + indexes.
7. **0007_prospects** — Creates `prospect` table + indexes.
8. **0008_profile** — Creates `company_profile`, `target_criteria`, `nogo_rule`, `keyword_set`, `source` tables + indexes.

Each migration has `.up.sql` (apply) and `.down.sql` (rollback) files.

---

## Planned entities

These are specified in the PRD and supported by the Hermes bridge, but not yet implemented as tables or HTTP routes:

- **discovery_run** — A scheduled or on-demand crawling job
- **prospect_score** — Per-prospect AI scoring (distinct from tender scoring)
- **playbook** — Versioned playbook template (structured bid strategy)
- **report** — Generated report (daily digest, pipeline, per-opportunity)

When these are wired in, they will follow the same data model principles: UUID PKs, timestamps, no `organization_id`, and GORM singular table names.
