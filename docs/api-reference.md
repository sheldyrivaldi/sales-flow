# API Reference

## Overview

- **Base URL:** `http://localhost:8080` (dev), `http://api:8080` (Docker), `http://<api-host>` (prod)
- **All endpoints** use the `/api` prefix except `/healthz` (health check)
- **Authentication:** JWT Bearer token in the `Authorization` header: `Bearer <access_token>`
- **Content-Type:** All request/response bodies are JSON
- **Error shape:** `{"error":{"code":"<CODE>","message":"<message>"}}`
- **Pagination:** `GET` list endpoints support `?page=1&page_size=20` (defaults: page 1, size 20; max page_size 100). List responses always use the shape `{"items":[...],"total":N,"page":N,"page_size":N}`.

---

## Authentication

### Login (POST `/api/auth/login`)

No auth required.

**Request body:**
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**Response (200):**
```json
{
  "access_token": "<JWT access token, TTL 15 min>",
  "refresh_token": "<JWT refresh token, TTL 7 days>",
  "user": {
    "id": "uuid",
    "email": "user@example.com",
    "name": "Full Name",
    "role": "SALES",
    "active": true
  }
}
```

### Refresh (POST `/api/auth/refresh`)

No auth required (this endpoint is public, like login). Mints a new access and refresh token pair from a valid refresh token.

**Request body:**
```json
{
  "refresh_token": "<JWT refresh token>"
}
```

**Response (200):**
```json
{
  "access_token": "<new JWT access token>",
  "refresh_token": "<new JWT refresh token>"
}
```

**Note:** The refresh token sent in the request should be the value from the previous login or refresh response. Tokens are **typed** — a refresh token cannot be used as an access token and vice versa.

### Me (GET `/api/me`)

JWT required. Returns the authenticated user.

**Response (200):**
```json
{
  "id": "uuid",
  "email": "user@example.com",
  "name": "Full Name",
  "role": "ADMIN",
  "active": true
}
```

---

## Health (no auth)

### Health Check (GET `/healthz`)

No auth required.

**Response (200 — OK):**
```json
{
  "status": "ok"
}
```

**Response (503 — DB down):**
```json
{
  "status": "degraded",
  "db": "down"
}
```

---

## Users

All user endpoints require the `ManageUsers` capability → **ADMIN role only**.

### List Users (GET `/api/users`)

**Query params:**
- `page` (int, optional, default 1)
- `page_size` (int, optional, default 20)
- `role` (string, optional; filter by `SALES`, `OPS`, `MANAGER`, `ADMIN`)
- `active` (string, optional; `true` or `false`)
- `search` (string, optional)

**Response (200):**
```json
{
  "items": [
    {
      "id": "uuid",
      "email": "user1@example.com",
      "name": "User One",
      "role": "SALES",
      "active": true
    }
  ],
  "total": 5,
  "page": 1,
  "page_size": 20
}
```

### Create User (POST `/api/users`)

**Request body:**
```json
{
  "email": "newuser@example.com",
  "name": "New User",
  "password": "InitialPassword123",
  "role": "SALES"
}
```

**Response (201):**
```json
{
  "id": "uuid",
  "email": "newuser@example.com",
  "name": "New User",
  "role": "SALES",
  "active": true
}
```

### Update User (PATCH `/api/users/:id`)

**Request body** (all fields optional):
```json
{
  "name": "Updated Name",
  "role": "MANAGER",
  "active": false
}
```

**Response (200):**
```json
{
  "id": "uuid",
  "email": "user@example.com",
  "name": "Updated Name",
  "role": "MANAGER",
  "active": false
}
```

### Reset User Password (POST `/api/users/:id/reset-password`)

ADMIN only. The `password` field is optional — if omitted, the server generates a random password and returns it in the response. If provided, it must be 8–72 characters.

**Request body** (optional field):
```json
{
  "password": "NewPassword123"
}
```

**Response (200):**
```json
{
  "password": "NewPassword123"
}
```

If no password was supplied in the request, `password` in the response is the newly generated one — this is the only time it is returned, so capture it immediately.

---

## Tenders

All tender endpoints require the `CRUDData` capability → **all roles** (SALES, OPS, MANAGER, ADMIN).

### List Tenders (GET `/api/tenders`)

**Query params:**
- `page` (int, optional, default 1)
- `page_size` (int, optional, default 20)
- `status` (string, optional; filter by status: `IDENTIFIED`, `QUALIFYING`, `BIDDING`, `SUBMITTED`, `WON`, `LOST`)
- `buyer` (string, optional; substring search on `buyer_name`)
- `recommended_action` (string, optional; `PURSUE`, `REVIEW`, `WATCHLIST`, `REJECT`, `NEED_PARTNER`)
- `origin` (string, optional; `manual` or `discovery`)
- `deadline_from` (ISO 8601 date, optional)
- `deadline_to` (ISO 8601 date, optional)
- `search` (string, optional; full-text search on title + scope summary)

**Response (200):**
```json
{
  "items": [
    {
      "id": "uuid",
      "title": "IT Infrastructure Procurement",
      "buyer_name": "Ministry X",
      "buyer_country": "Indonesia",
      "buyer_industry": "Government",
      "value_estimate": 500000,
      "currency": "IDR",
      "published_date": "2026-07-01T00:00:00Z",
      "submission_deadline": "2026-08-15T17:00:00Z",
      "source_name": "LKPP",
      "source_url": "https://lkpp.go.id/tender/...",
      "service_category": "IT Services",
      "scope_summary": "Supply and installation of servers...",
      "status": "QUALIFYING",
      "fit_score": null,
      "recommended_action": null,
      "risk_flags": null,
      "reasoning_summary": null,
      "origin": "manual",
      "created_at": "2026-07-02T10:00:00Z",
      "updated_at": "2026-07-05T14:00:00Z"
    }
  ],
  "total": 42,
  "page": 1,
  "page_size": 20
}
```

### Create Tender (POST `/api/tenders`)

**Request body:**
```json
{
  "title": "New Tender Title",
  "buyer_name": "Buyer Corp",
  "buyer_country": "Indonesia",
  "buyer_industry": "Manufacturing",
  "value_estimate": 250000,
  "currency": "IDR",
  "published_date": "2026-07-05T00:00:00Z",
  "submission_deadline": "2026-08-20T17:00:00Z",
  "source_name": "Internal Inbox",
  "source_url": "https://example.com/tender/123",
  "service_category": "Consulting",
  "scope_summary": "Strategic advisory on digital transformation...",
  "eligibility_requirements": "ISO 27001 certified...",
  "technical_requirements": "Python, Go, Postgres experience required..."
}
```

**Response (201):**
```json
{
  "id": "new-uuid",
  "title": "New Tender Title",
  "status": "IDENTIFIED",
  "origin": "manual",
  "created_at": "2026-07-05T15:00:00Z",
  "updated_at": "2026-07-05T15:00:00Z",
  "...": "other fields as listed in List response"
}
```

### Get Tender (GET `/api/tenders/:id`)

**Response (200):**
```json
{
  "id": "uuid",
  "title": "IT Infrastructure Procurement",
  "buyer_name": "Ministry X",
  "status": "QUALIFYING",
  "...": "all fields"
}
```

### Update Tender (PUT `/api/tenders/:id`)

**Request body** (all fields optional):
```json
{
  "title": "Updated Title",
  "buyer_name": "Updated Buyer",
  "value_estimate": 600000,
  "scope_summary": "Updated scope..."
}
```

**Response (200):**
```json
{
  "id": "uuid",
  "title": "Updated Title",
  "...": "all fields with updates applied"
}
```

### Delete Tender (DELETE `/api/tenders/:id`)

**Response (204 — No Content)**

### Change Tender Status (PATCH `/api/tenders/:id/status`)

Transitions the tender through the status machine. Invalid transitions return 400 Bad Request.

**Request body:**
```json
{
  "status": "QUALIFYING"
}
```

**Valid transitions:**
- `IDENTIFIED` → `QUALIFYING`, `LOST`
- `QUALIFYING` → `BIDDING`, `LOST`
- `BIDDING` → `SUBMITTED`, `WON`, `LOST`
- `SUBMITTED` → `WON`, `LOST`
- `WON`, `LOST` → terminal (no further transitions)

**Response (200):**
```json
{
  "id": "uuid",
  "status": "QUALIFYING",
  "updated_at": "2026-07-05T15:30:00Z",
  "...": "all fields"
}
```

### Record Tender Outcome (POST `/api/tenders/:id/outcome`)

Records a final outcome (WON or LOST) and creates an `OutcomeEvent`. Sets the tender status to the outcome.

**Request body:**
```json
{
  "result": "WON",
  "notes": "Won on technical merit; partner collaboration model."
}
```

**Response (200):**

Returns the updated tender (an `OutcomeEvent` is created server-side but is not part of the response body). The tender's `status` is set to the outcome result:
```json
{
  "id": "tender-uuid",
  "title": "IT Infrastructure Procurement",
  "status": "WON",
  "...": "all tender fields"
}
```

### Promote Discovery Tender (POST `/api/tenders/:id/promote`)

Moves a discovery tender from `IDENTIFIED` to `QUALIFYING`. The tender must have `origin: discovery`.

**Request body** (empty):
```json
{}
```

**Response (200):**
```json
{
  "id": "uuid",
  "status": "QUALIFYING",
  "origin": "discovery",
  "...": "all fields"
}
```

---

## Events

All event endpoints require the `CRUDData` capability → **all roles**.

### List Events (GET `/api/events`)

**Query params:**
- `page` (int, optional, default 1)
- `page_size` (int, optional, default 20)
- `type` (string, optional; `EXPO`, `CONFERENCE`, `SEMINAR`, `WORKSHOP`, `NETWORKING`, `OTHER`)
- `status` (string, optional; `PLANNED`, `ATTENDED`, `CANCELLED`)
- `search` (string, optional)

**Response (200):**
```json
{
  "items": [
    {
      "id": "uuid",
      "name": "Tech Conference 2026",
      "type": "CONFERENCE",
      "date": "2026-09-15T09:00:00Z",
      "location": "Jakarta Convention Center",
      "organizer": "Tech Indonesia",
      "notes": "Focus on AI and digital transformation.",
      "status": "PLANNED",
      "created_at": "2026-07-01T10:00:00Z",
      "updated_at": "2026-07-01T10:00:00Z"
    }
  ],
  "total": 10,
  "page": 1,
  "page_size": 20
}
```

### Create Event (POST `/api/events`)

**Request body:**
```json
{
  "name": "Procurement Expo 2026",
  "type": "EXPO",
  "date": "2026-08-20T10:00:00Z",
  "location": "Bali, Indonesia",
  "organizer": "Procurement Board",
  "notes": "Meet vendors and buyers in the region."
}
```

**Response (201):**
```json
{
  "id": "new-uuid",
  "name": "Procurement Expo 2026",
  "type": "EXPO",
  "date": "2026-08-20T10:00:00Z",
  "location": "Bali, Indonesia",
  "status": "PLANNED",
  "created_at": "2026-07-05T15:00:00Z",
  "updated_at": "2026-07-05T15:00:00Z"
}
```

### Get Event (GET `/api/events/:id`)

**Response (200):**
```json
{
  "id": "uuid",
  "name": "Tech Conference 2026",
  "...": "all fields"
}
```

### Update Event (PUT `/api/events/:id`)

**Request body** (all fields optional):
```json
{
  "name": "Updated Event Name",
  "status": "ATTENDED",
  "notes": "Event was well-attended; met 15+ prospects."
}
```

**Response (200):**
```json
{
  "id": "uuid",
  "name": "Updated Event Name",
  "status": "ATTENDED",
  "...": "all fields with updates"
}
```

### Delete Event (DELETE `/api/events/:id`)

**Response (204 — No Content)**

### Convert Event to Prospect (POST `/api/events/:id/convert`)

Creates a Prospect from an Event. No request body is required — the new prospect is derived from the event and owned by the authenticated user.

**Request body:** none.

**Response (201):**
```json
{
  "id": "prospect-uuid",
  "name": "Tech Conference 2026",
  "company": null,
  "contact_info": null,
  "source_type": "event",
  "source_id": "event-uuid",
  "stage": "NEW",
  "est_value": null,
  "owner_user_id": "current-user-uuid",
  "created_at": "2026-07-05T15:40:00Z",
  "updated_at": "2026-07-05T15:40:00Z"
}
```

---

## Conversations & Chat

All conversation endpoints require the `UseAI` capability → **all roles**.

### Create Conversation (POST `/api/conversations`)

**Request body** (both fields optional):
```json
{
  "title": "Tender Strategy Q3",
  "first_message": "Which tenders should we prioritize this quarter?"
}
```

**Response (201):**
```json
{
  "id": "conversation-uuid",
  "title": "Tender Strategy Q3",
  "created_at": "2026-07-05T15:00:00Z",
  "updated_at": "2026-07-05T15:00:00Z"
}
```

If no title is provided, one is auto-derived from `first_message` (or left blank if neither is provided).

### List Conversations (GET `/api/conversations`)

**Query params:**
- `page` (int, optional, default 1)
- `page_size` (int, optional, default 20)

**Response (200):**
```json
{
  "items": [
    {
      "id": "uuid",
      "title": "Tender Analysis",
      "created_at": "2026-07-01T10:00:00Z",
      "updated_at": "2026-07-05T12:00:00Z"
    }
  ],
  "total": 5,
  "page": 1,
  "page_size": 20
}
```

### Get Conversation (GET `/api/conversations/:id`)

**Response (200):**
```json
{
  "id": "uuid",
  "title": "Tender Analysis",
  "messages": [
    {
      "id": "msg-uuid",
      "conversation_id": "uuid",
      "role": "user",
      "content": "What tenders match our IT capabilities?",
      "created_at": "2026-07-05T15:10:00Z"
    },
    {
      "id": "msg-uuid",
      "conversation_id": "uuid",
      "role": "assistant",
      "content": "Based on your company profile...",
      "created_at": "2026-07-05T15:10:05Z"
    }
  ],
  "created_at": "2026-07-05T15:00:00Z",
  "updated_at": "2026-07-05T15:10:05Z"
}
```

`tool_calls` on a message is omitted from the JSON entirely when empty (not `null`).

### Post Message (Chat SSE) (POST `/api/conversations/:id/chat`)

Streams the AI response as Server-Sent Events (SSE). The request initiates a streaming connection.

**Request body:**
```json
{
  "content": "Which tenders in our list should we prioritize?"
}
```

**Response (200) — text/event-stream:**

The response is a stream of SSE events. Each event is a single line starting with `data: ` followed by JSON:

**Event frames:**

1. **Delta (streaming text):**
   ```
   data: {"type":"delta","content":"Based on..."}
   ```

2. **Tool Call:**
   ```
   data: {"type":"tool_call","id":"tool-id","name":"get_tenders","arguments":{"status":"QUALIFYING"}}
   ```

3. **Done (assembly complete):**
   ```
   data: {"type":"done"}
   ```

4. **Error (mid-stream failure):**
   ```
   data: {"type":"error","message":"Agent AI sedang tidak tersedia. Coba lagi sebentar lagi."}
   ```
   Note: this frame has no `code` field — it's a plain `type`/`message` pair, distinct from the standard `{"error":{"code",...}}` shape used by non-streaming endpoints.

5. **Stream terminator (always last):**
   ```
   data: [DONE]
   ```

The client reads events line-by-line, parses each `data: {...}` as JSON, and stops when it sees `data: [DONE]`.

If the hermes-bridge is unavailable *before* the stream starts (connection refused, no provider key, etc.), the endpoint instead returns a normal JSON error response with `AI_UNAVAILABLE` (HTTP 400) — no SSE stream is opened at all. The `error` SSE frame above only occurs if the stream had already started and then failed mid-way.

**Example client usage (JavaScript):**
```javascript
const response = await fetch('/api/conversations/conv-id/chat', {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${accessToken}`,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({ content: "User's message here" })
});

const reader = response.body.getReader();
const decoder = new TextDecoder();

while (true) {
  const { done, value } = await reader.read();
  if (done) break;

  const text = decoder.decode(value);
  const lines = text.split('\n');
  for (const line of lines) {
    if (line.startsWith('data: ')) {
      const data = line.slice(6);
      if (data === '[DONE]') {
        console.log('Stream finished');
        break;
      }
      const event = JSON.parse(data);
      console.log('Event:', event);
    }
  }
}
```

---

## Planned Endpoints

The following features are documented in the PRD and supported by the Hermes bridge, but are **not yet available as HTTP routes**:

- **Discovery** — automated tender crawling and ingestion
- **AI Scoring** — deterministic JSON scoring for tenders and prospects
- **Company Knowledge Profile** — "Otak Agent" — company capabilities and history
- **Playbooks** — structured bid strategy generation
- **Reports** — daily digest, pipeline, per-opportunity analysis
- **Prospect Management (advanced)** — list/update prospects (only `Create`, `GetByID`, `GetBySource` are implemented)

These will be wired in as HTTP endpoints in future releases, with contract tests gating the Hermes version dependency.

---

## Error Responses

All errors follow the standard shape:

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable description"
  }
}
```

Common error codes:
- `UNAUTHORIZED` (401) — not authenticated, or token invalid/expired
- `FORBIDDEN` (403) — user lacks required capability for this endpoint
- `NOT_FOUND` (404) — resource not found
- `VALIDATION_ERROR` (422) — validation failed (e.g., invalid status transition)
- `AI_UNAVAILABLE` (400) — hermes-bridge is not responding or no provider key configured
- `INTERNAL_ERROR` (500) — server error

Other codes (e.g., invalid-transition or conflict cases) may use a specific code with a 400 or 409 status; the `code` field is always present and the shape above never changes.
