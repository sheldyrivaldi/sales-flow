# hermes-bridge — Plan Implementasi (Jembatan Python ke Hermes Agent)

> **Status koreksi arsitektur (2026-06-19):** Hermes Agent (Nous Research) **bukan** HTTP gateway OpenAI-compatible seperti diasumsikan `epic.plan.md` awal. Hermes adalah **agentic engine** yang diakses via **Python library** (`from run_agent import AIAgent`). Dokumen ini mendefinisikan `hermes-bridge` — service Python tipis yang membungkus `AIAgent` dan mengekspos HTTP OpenAI-compatible agar backend Go (`internal/hermes`) bisa bicara seperti semula.

## Context

SalesPilot ingin memanfaatkan kapabilitas Hermes yang sudah jadi (memory persisten, 70+ tools, 28 toolsets, skills, browser automation) **tanpa** memaksa user sales memakai TUI Hermes yang teknikal. Solusinya: Hermes jadi **mesin di belakang layar**, SalesPilot jadi **UI ramah-sales** di depan.

Karena Hermes ke depan akan sering berubah (termasuk upgrade major), arsitektur memakai **double isolation**:
1. **`internal/hermes` (ACL Go)** — UI & backend hanya kenal interface `Client`.
2. **`hermes-bridge` (Python)** — satu-satunya yang menyentuh API `AIAgent`. Bila API Hermes berubah, **hanya file bridge** yang disesuaikan; Go + React tak tersentuh.

### Keputusan terkunci (dikonfirmasi user)
- **Kontrak HTTP bridge = tiru OpenAI `/v1`** → kode `internal/hermes` yang sudah ada (Chat/ChatStream non-stream & SSE) nyaris tak berubah.
- **Streaming = MVP blocking-then-send** → bridge panggil `AIAgent` (blocking), lalu kirim hasil sebagai SSE chunk + `[DONE]`. Upgrade ke real-time (callback) menyusul tanpa mengubah Go/UI.

### Fakta Hermes terverifikasi (dari docs resmi)
- Library: `pip install git+https://github.com/NousResearch/hermes-agent.git`; `from run_agent import AIAgent`.
- API utama:
  - `agent.chat(message) -> str` (loop penuh, balas teks final).
  - `agent.run_conversation(user_message, conversation_history=, task_id=) -> {"final_response", "messages"}`.
- Constructor params relevan: `model`, `quiet_mode`, `enabled_toolsets`, `disabled_toolsets`, `skip_memory`, `skip_context_files`, `api_key`, `base_url`, `max_iterations`, `ephemeral_system_prompt`, `platform`.
- **Sinkron, tanpa streaming native, TIDAK thread-safe** → **buat instance `AIAgent` baru per request**.
- Memory: SQLite + FTS5 + context compression, per-profile, persisten lintas restart. `skip_memory=False` (default) = memory terakumulasi.
- Provider model via env: `OPENROUTER_API_KEY` / `OPENAI_API_KEY` / `ANTHROPIC_API_KEY` (atau Nous Portal).

### Ketidakpastian yang harus diverifikasi saat implementasi (cek source/`pip show`)
1. **Format `conversation_history`** — diasumsikan list `{"role","content"}` (OpenAI/ShareGPT). Verifikasi field & apakah pesan `tool` didukung.
2. **Cara mengikat `WORKSPACE_SESSION_KEY` ke satu profile memory Hermes** — kemungkinan via konfigurasi profile/working dir, bukan param konstruktor. MVP: satu workspace = satu profile global (SalesPilot memang single-company per `epic.plan.md`).
3. **Ekstraksi tool-calls dari `result["messages"]`** untuk chip "🔧 Membaca data…" di UI — struktur pesan tool perlu dicek (opsional untuk MVP).
4. **Callback streaming** (untuk upgrade nanti) — arsitektur menyebut "callbacks"; cek signature di source.

---

## Arsitektur & alur request

```
React → Go API (internal/hermes ACL) → [HTTP /v1] → hermes-bridge (FastAPI) → AIAgent → OpenRouter/Nous
```

- Go mengirim: `POST /v1/chat/completions` + header `Authorization: Bearer <API_SERVER_KEY>`, `X-Hermes-Session-Key`, `X-Hermes-Session-Id`, body `{model, messages, stream}`.
- Bridge:
  1. Validasi `Authorization` == `API_SERVER_KEY` (env bridge).
  2. `X-Hermes-Session-Key` → tentukan profile memory (workspace tunggal).
  3. `X-Hermes-Session-Id` → `task_id` (isolasi eksekusi tool per percakapan).
  4. Pisah `messages`: `messages[-1].content` = `user_message`; `messages[:-1]` = `conversation_history`.
  5. Buat `AIAgent(...)` **baru**, panggil `run_conversation(...)`.
  6. Bentuk respons OpenAI-shaped.

---

## Struktur folder

```
services/hermes-bridge/
├── pyproject.toml          # deps: fastapi, uvicorn, hermes-agent (pinned)
├── app/
│   ├── __init__.py
│   ├── main.py             # FastAPI app + lifespan
│   ├── config.py           # env: API_SERVER_KEY, HERMES_MODEL, provider keys, toolsets, PORT
│   ├── auth.py             # dependency cek Bearer
│   ├── agent_factory.py    # build AIAgent per-request (toolsets, model, profile)
│   ├── schemas.py          # pydantic: ChatCompletionRequest/Response, dst.
│   └── routes/
│       ├── chat.py         # POST /v1/chat/completions (stream & non-stream)
│       ├── health.py       # GET /health, GET /v1/capabilities
│       └── responses.py    # POST /v1/responses (GenerateJSON; dipakai TK-01.3)
├── tests/
│   └── test_chat.py        # test endpoint dgn AIAgent di-mock
├── Dockerfile
└── README.md
```

---

## Endpoint yang diekspos (cocok dengan ACL Go yang sudah ada)

### 1. `POST /v1/chat/completions`
- **Auth:** Bearer `API_SERVER_KEY` (401 bila salah).
- **Body:** `{ "model": str, "messages": [{role,content}], "stream": bool }` (field `model` diabaikan; bridge pakai `HERMES_MODEL` dari env).
- **stream=false:** panggil `run_conversation`; balas:
  ```json
  { "choices": [ { "message": { "content": "...", "tool_calls": [...] } } ] }
  ```
- **stream=true (MVP):** header `Content-Type: text/event-stream`; panggil blocking, lalu emit:
  ```
  data: {"choices":[{"delta":{"content":"<hasil penuh / dipotong>"}}]}

  data: [DONE]
  ```
  (Boleh emit beberapa chunk dengan memotong teks agar UI terasa mengalir; cukup untuk MVP.)
- **Error/timeout:** non-2xx body JSON `{"error":{...}}`; saat stream, kirim `data: {"error":...}` lalu tutup. Jangan crash worker.

### 2. `GET /health`
- Balas `{"status":"ok"}`. Opsional: cek ketersediaan provider key.

### 3. `GET /v1/capabilities`
- Balas `{"version": "<hermes lib version>", "models": ["<HERMES_MODEL>"], "features": ["chat","memory","tools"]}` (disintesis dari versi lib + config). Dipakai `Health()` ACL (TK-01.4) & Settings (EP-18).

### 4. `POST /v1/responses` (untuk `GenerateJSON`, TK-01.3 — boleh stub dulu)
- Terima `{prompt, response_format(json_schema)}`; bridge set `ephemeral_system_prompt` "balas JSON valid sesuai schema", `enabled_toolsets=[]` (matikan tools agar deterministik), panggil `chat`, balas `json.RawMessage`. MVP bodoh: kembalikan teks, validasi JSON di sisi Go (TK-01.3.2 sudah retry).

### 5. `POST /admin/config` (override provider runtime — dipakai EP-18)
- **Auth:** Bearer `API_SERVER_KEY`.
- **Body:** `{ "provider": "openai"|"openrouter", "model": str, "base_url": str|null, "api_key": str }`.
- Simpan **in-memory** (config aktif); `agent_factory` memakai config ini, **fallback ke env** (`OPENAI_API_KEY`/`OPENROUTER_API_KEY`, `HERMES_MODEL`) bila belum di-set.
- **Stateless terhadap restart:** kalau bridge restart, config in-memory hilang → SalesPilot Go **re-push** saat boot (TK-18.4.4). Ini menjaga ACL chat tetap kontrak OpenAI murni (tak perlu kirim key tiap request).
- **Provider mapping:** `openai` → base_url default `https://api.openai.com/v1`; `openrouter` → `https://openrouter.ai/api/v1`. `model` mengikuti format provider (mis. `gpt-4o` vs `anthropic/claude-sonnet-4.6`).

---

## Pemetaan ke `AIAgent`

| Konsep SalesPilot | Ke Hermes |
|---|---|
| `messages` (history dari Postgres) | `conversation_history` (semua kecuali pesan terakhir) + `user_message` (pesan terakhir) |
| `WORKSPACE_SESSION_KEY` | profile memory tunggal (workspace) — `skip_memory=False` |
| `X-Hermes-Session-Id` (conversation) | `task_id` (isolasi tool execution) |
| toolset sales (web/browser) | `enabled_toolsets=["web", ...]`; matikan `terminal` untuk keamanan default |
| model | `HERMES_MODEL` (env), mis. `anthropic/claude-sonnet-4.6` atau model Nous via OpenRouter |

**Thread-safety:** endpoint pakai `def` (sync) → FastAPI jalankan di threadpool; `AIAgent` baru tiap request. Untuk batch/parallel, ikuti pola docs (`skip_memory=True` bila stateless seperti scoring/report; `False` untuk chat agar belajar).

---

## Task breakdown (urut; tiap selesai jalankan Done-check)

- **BR-1 — Scaffold service.** `services/hermes-bridge/` + `pyproject.toml` (pin `hermes-agent`, fastapi, uvicorn) + `app/main.py` (FastAPI + `/health`). **Done:** `uvicorn app.main:app` hidup, `GET /health` → ok.
- **BR-2 — Config + auth.** `config.py` (env) + `auth.py` (Bearer dependency). **Done:** request tanpa/with-wrong Bearer → 401.
- **BR-3 — Agent factory.** `agent_factory.py` bangun `AIAgent` dari config (model, toolsets, quiet_mode=True). **Done:** unit test buat agent tanpa error (model di-mock/skip jaringan).
- **BR-4 — Chat non-stream.** `routes/chat.py` stream=false → `run_conversation` → OpenAI-shaped. **Done:** `curl` dgn 1 pesan → balas content; integrasi nyata balas jawaban.
- **BR-5 — Chat stream (MVP).** stream=true → blocking → SSE chunk + `[DONE]`. **Done:** `curl -N` menerima `data:` lalu `[DONE]`; ACL Go `ChatStream` test integrasi jalan.
- **BR-6 — Capabilities + error handling.** `/v1/capabilities`; bungkus error provider → JSON ramah; tidak crash saat provider down. **Done:** matikan provider key → 5xx JSON, worker tetap hidup.
- **BR-7 — responses (GenerateJSON) skeleton.** `routes/responses.py` minimal. **Done:** balas JSON untuk schema contoh (boleh kasar; disempurnakan di TK-01.3).
- **BR-8 — Dockerfile + compose wiring.** `Dockerfile`; ganti service `hermes-gateway` di `deploy/docker-compose.yml` → `hermes-bridge` (build context + env provider key + `API_SERVER_KEY` + `HERMES_MODEL`); `HERMES_BASE_URL` Go → `http://hermes-bridge:PORT`. **Done:** `docker compose config` valid; stack up; Go `Health()` hijau.
- **BR-9 — `/admin/config` + agent_factory dinamis.** `routes/admin.py`; config aktif in-memory override env (OpenAI/OpenRouter). **Done:** set config → chat pakai provider itu; tanpa config → fallback env.
- **BR-10 — ACL `Configure` (Go).** `internal/hermes/configure.go` + tambah `Configure(ctx, ProviderConfig)` ke interface `Client` (additive). **Done:** `go build`/`vet` hijau; unit test httptest. (Jembatan untuk EP-18 ST-18.4.)

---

## Penyesuaian dokumen plan yang diperlukan (di luar bridge)

- `epic.plan.md` baris 33 & EP-01 scope: deskripsi Hermes dikoreksi (engine via Python lib + bridge, bukan image gateway publik).
- `task.plan.md`:
  - TK-00.5.1 / TK-00.5.2 / TK-01.5.2: service `hermes-gateway` → `hermes-bridge` (build sendiri, pin versi `hermes-agent`).
  - Tambah **ST-01.6 — hermes-bridge** (task BR-1..BR-8) di EP-01.
- ACL Go (`internal/hermes`, 6 task selesai): **tetap valid**, tidak dibongkar (kontrak `/v1` OpenAI-compatible).

---

## Verifikasi end-to-end (setelah bridge + ACL tersambung)

1. `docker compose up` → `postgres`, `api`, `web`, `hermes-bridge` hidup.
2. `curl http://localhost:<bridge>/health` → `{"status":"ok"}`; `/v1/capabilities` → versi+model.
3. Dari Go: `go test -tags contract ./internal/hermes/...` (HERMES_BASE_URL → bridge) → `Chat`, `ChatStream`, `Health` hijau.
4. UI Chat SalesPilot: kirim "Tender prioritas minggu ini?" → bridge → AIAgent (tool) → jawaban tampil (EP-04/EP-09).
5. Continuous learning: tandai WON → restart `hermes-bridge` → tanya ulang → jawaban mempertimbangkan konteks (memory persisten).
