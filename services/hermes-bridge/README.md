# hermes-bridge

A FastAPI wrapper around the Hermes `AIAgent` library that exposes an OpenAI-compatible `/v1` REST API. This service forms the anti-corruption boundary between the Go backend and internal Hermes library changes, allowing the backend to depend only on a stable `/v1` surface while Hermes versions are upgraded independently.

## Quick start

### Prerequisites

- Python ≥ 3.11
- Git (to fetch the pinned Hermes library from GitHub)

### Install & run

```bash
cd services/hermes-bridge

# Install the bridge + pinned Hermes version
pip install .

# Set the API_SERVER_KEY (must match the Go API's API_SERVER_KEY)
export API_SERVER_KEY=your-shared-secret-here

# Start the bridge (port 8642)
python -m app.main
```

Or using uvicorn directly:
```bash
uvicorn app.main:app --host 0.0.0.0 --port 8642
```

The bridge is now running and ready to accept requests from the Go API.

## Environment variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `API_SERVER_KEY` | **Yes** | — | Bearer token for authenticating requests from the Go API. Must match the value in the root `.env`. If empty, the bridge crashes at startup. |
| `HERMES_MODEL` | No | `default` | AI model to use (e.g., `gpt-4o`, `anthropic/claude-sonnet-4-6`, or `default` for Hermes' built-in selection). |
| `PORT` | No | `8642` | Port the bridge binds to. |
| `ENABLED_TOOLSETS` | No | `web` | Comma-separated list of toolsets to enable (e.g., `web,mcp`). |
| `OPENAI_API_KEY` | No | — | OpenAI API key (if using OpenAI as the LLM provider). |
| `OPENROUTER_API_KEY` | No | — | OpenRouter API key (if using OpenRouter as the LLM provider). |

At least one of `OPENAI_API_KEY` or `OPENROUTER_API_KEY` must be set to enable AI features. Without a provider key, the bridge boots successfully but AI requests fail gracefully.

## Endpoints

The bridge exposes OpenAI-compatible endpoints plus admin configuration:

| Endpoint | Method | Auth | Description |
|---|---|---|---|
| `/health` | GET | No | Health check; returns `{"status":"ok"}` |
| `/v1/capabilities` | GET | No | List bridge capabilities and available models |
| `/v1/chat/completions` | POST | Bearer | Chat endpoint (streaming or JSON); accepts `stream=true` or `stream=false` |
| `/v1/responses` | POST | Bearer | Deterministic JSON responses for structured extraction (e.g., scoring) |
| `/admin/config` | GET | Bearer | Retrieve current provider and model configuration |
| `/admin/config` | POST | Bearer | Set provider, model, or base URL (SSRF-guarded to `https://` + `openai.com` / `openrouter.ai`) |

All authenticated endpoints require:
```
Authorization: Bearer <API_SERVER_KEY>
```

Token validation is done with constant-time comparison to prevent timing attacks.

## Modes

The bridge instantiates the Hermes `AIAgent` in one of two modes per request:

### Chat mode
- **Memory:** ON — conversation history is stored and replayed
- **Tools:** ON — MCP data tools are available (e.g., `get_tenders`, `search_tenders`)
- **Use case:** `/v1/chat/completions` — conversational AI with context awareness
- **Endpoint:** `POST /v1/chat/completions` with `mode: "chat"`

### Responses mode
- **Memory:** OFF — each request is stateless
- **Tools:** OFF — no tool calls
- **Use case:** `/v1/responses` — deterministic JSON extraction (e.g., tender scoring)
- **Endpoint:** `POST /v1/responses` with `mode: "responses"`

The calling Go API specifies the mode in the request body.

## Running tests

```bash
# Run all tests (24 tests)
pytest

# Run a specific test file
pytest tests/test_chat.py

# Run with verbose output
pytest -v

# Run with coverage report
pytest --cov=app tests/
```

Tests are in `services/hermes-bridge/tests/` and use pytest.

## Upgrading the Hermes version

The Hermes library version is pinned in `pyproject.toml`:

```toml
hermes-agent = "git+https://github.com/NousResearch/hermes-agent.git@v2026.6.19"
```

### Process

1. **Bump the git tag** in `pyproject.toml`:
   ```toml
   hermes-agent = "git+https://github.com/NousResearch/hermes-agent.git@v2026.7.01"
   ```

2. **Reinstall the bridge** with the new version:
   ```bash
   cd services/hermes-bridge
   pip install .
   ```

3. **Run contract tests** (in the Go repository):
   ```bash
   go test ./internal/hermes -run TestContract -v
   ```

   These tests verify that the bridge's `/v1` surface still conforms to the OpenAI spec. **Contract tests must pass before committing.**

4. **If contract tests fail:**
   - The new Hermes version broke the `/v1` spec
   - Either revert the version bump or fix the bridge wrapper (`app/*.py`) to restore compatibility
   - Re-run contract tests

5. **If all tests pass:**
   - Commit the change: `git commit -m "upgrade: hermes-agent to v2026.7.01"`
   - Redeploy the bridge in your environment

### Why contract tests?

The Go API only depends on `/v1`. Contract tests ensure that if a Hermes upgrade changes `/v1` behavior, we catch it before production. This allows Hermes to evolve independently while keeping the bridge a stable boundary.

## Architecture

```
Bridge
  ├─ app/main.py          FastAPI app entry point
  ├─ app/routes/          Endpoint handlers
  │  ├─ health.py         /health
  │  ├─ chat.py           /v1/chat/completions
  │  ├─ responses.py      /v1/responses
  │  └─ admin.py          /admin/config
  ├─ app/config.py        Environment + provider setup
  ├─ app/auth.py          Bearer token validation
  ├─ app/schemas.py       Pydantic request/response models
  └─ app/agent_factory.py Hermes AIAgent instantiation

Hermes (pinned)
  └─ AIAgent              The actual LLM orchestration + tools
```

The bridge decouples the Go API from Hermes internals. Changes to Hermes that don't affect `/v1` require only a bridge restart; breaking changes to `/v1` are caught by contract tests.

## Logging

The bridge logs:
- Startup configuration (with secrets redacted)
- Incoming requests (method, path, auth status)
- Errors and exceptions (with stack traces)
- Upstream LLM provider calls and latencies

Logs go to stdout. In production, direct stdout to a log aggregator (e.g., CloudWatch, ELK).

## Troubleshooting

| Issue | Cause | Solution |
|---|---|---|
| `RuntimeError: API_SERVER_KEY not set` | `API_SERVER_KEY` env var is empty | Set it: `export API_SERVER_KEY=<shared-secret>` |
| `Connection refused: localhost:8642` | Bridge not running | Start with `python -m app.main` or check if port 8642 is in use |
| `Unauthorized` (401) | Bearer token invalid | Verify `API_SERVER_KEY` matches the Go API's value |
| `Model not found` or `OPENAI_API_KEY not set` | No LLM provider key | Set `OPENAI_API_KEY` or `OPENROUTER_API_KEY` in `.env` |
| `Tests fail: "no module named 'hermes'"` | Hermes not installed | Run `pip install .` from the bridge directory |

## License

Proprietary — Copyright (c) 2026 SalesPilot. All Rights Reserved. Internal use only.
