# EP-16 — Continuous Learning: Memory Persistence Verification (TK-16.4.2)

> Scope: verify that WON/LOST outcomes and Discovery "Tolak" rejections actually reach
> Hermes workspace memory, and that memory survives a `hermes-bridge` restart — per
> epic.plan.md §4 verification item 8 ("Tandai WON → restart Hermes → chat tetap ingat
> konteks").

## Contract (how the loop is supposed to work)

1. WON/LOST on a tender/prospect, or "Tolak" on a Discovery Inbox tender with a reason,
   calls `ai.LearningHermes.RecordOutcome` / `RecordDiscoveryReject`
   (`internal/ai/learning.go`, EP-16 TK-16.2.1).
2. That method sends a short Bahasa Indonesia note via `hc.Chat(...)` with
   `SessionKey = cfg.WorkspaceSessionKey` — a plain chat call, not a dedicated
   "write to memory" API, because `hermes-bridge`'s `mode="chat"` path already runs
   with `skip_memory=False` (`services/hermes-bridge/app/agent_factory.py`), so every
   chat turn is itself a memory-write.
3. On the Hermes side, the configured `memory.provider: holographic`
   (`deploy/hermes/config.yaml.example`) persists facts to a SQLite file at
   `get_hermes_home()/memory_store.db` (confirmed by reading the pinned
   `hermes-agent` source at tag `v2026.6.19`, see TK-16.3.1 for how that was
   investigated).
4. A later chat in the same workspace (same `SessionKey`) should have that fact
   available for retrieval/recall by the agent's own memory tools.

## Infrastructure gap found and fixed during this verification

Reading `get_hermes_home()`'s actual resolution (`HERMES_HOME` env var, else
`~/.hermes`) against `services/hermes-bridge/Dockerfile` (no `USER` directive → runs
as **root**, so home = `/root`) against `deploy/docker-compose.yml` **before this
task** revealed: the `hermes-bridge` service had **no volume mounted for `/root`** —
memory (`MEMORY.md`, `USER.md`, `memory_store.db`) lived on the container's ephemeral
filesystem. Any `docker compose restart hermes-bridge` (or image rebuild/recreate)
would silently wipe all workspace memory, defeating this entire epic's premise before
any request-level bug could even be reached.

**Fixed** (this session): `deploy/docker-compose.yml` now sets
`HERMES_HOME=/root/.hermes` and mounts a named volume `hermes-memory:/root/.hermes`
for the `hermes-bridge` service, alongside the pre-existing `uploads`/`pgdata`
volumes. Validated with `docker compose -f deploy/docker-compose.yml config` (parses
cleanly, `hermes-memory` volume resolves) — Docker Desktop itself is not running in
this environment, so the compose stack was not actually brought up.

## What was verified in this session (automated, real)

- **Go ACL** (`internal/hermes/reset_test.go`, `internal/ai/learning_test.go`,
  `internal/service/tender_service_test.go`): `RecordOutcome`/`RecordDiscoveryReject`
  call `hc.Chat` with the correct `SessionKey` and a note containing the
  target/result/reason; `TenderService.Review` only fires the reject hook when a
  reason is present (Watchlist, no reason → no hook call); Hermes/network failures
  are logged, never panic or block the CRUD response.
- **hermes-bridge** (`services/hermes-bridge/tests/`, run for real via the project's
  own `.venv` + pytest — 28/28 pass): `/admin/reset-memory` requires Bearer auth,
  deletes the expected files, degrades to a friendly 502 when `hermes-agent` isn't
  installed (the actual condition in this environment) rather than crashing.
- **docker-compose config**: the new volume/env wiring is syntactically valid.

## What could NOT be verified live, and why

- **Docker Desktop is not running** in this environment, so the full stack
  (`postgres` + `api` + `web` + `hermes-bridge`) was never brought up together.
- **`hermes-agent` is not installed** here (confirmed via `pip show hermes-agent`) —
  only its source was inspected via a pinned-tag git clone for the TK-16.3.1
  investigation. No real `AIAgent` process, no real LLM provider call, no real
  holographic SQLite write was exercised end-to-end.
- Consequently the literal scenario — record WON with a distinctive note, chat about
  it, restart `hermes-bridge`, chat again, confirm the answer still reflects that
  note — was **not run**. Doing so requires: Docker running, a real
  `OPENAI_API_KEY`/`OPENROUTER_API_KEY`, and the `hermes-agent` git dependency
  actually building inside the bridge image.

## How to complete this verification when the above is available

1. `docker compose -f deploy/docker-compose.yml up -d` with a real provider key set
   in `.env`.
2. Create a tender/prospect, mark it WON with a distinctive note (e.g. a made-up buyer
   name that can't appear by chance).
3. In Chat, ask something that would only be answerable if the agent recalls that
   note (e.g. "Apa yang kamu tahu soal <distinctive buyer name>?").
4. `docker compose restart hermes-bridge`.
5. Ask the same question again in a **new** conversation (still same
   `WORKSPACE_SESSION_KEY`, since that's what scopes Hermes memory, not the
   conversation id) — the answer should still reflect the recorded note.
6. If it doesn't: check the `hermes-memory` volume actually persisted
   (`docker volume inspect deploy_hermes-memory`) and that `memory.provider:
   holographic` in the mounted `config.yaml` matches what `agent_factory.py` is
   actually reading (TK-09.1.2's open question about whether `config.yaml` is even
   consulted by the bridge is still unresolved — see `task.plan.md` EP-09 notes).
