# task.plan.md — SalesPilot (Layer 3: TASK)

> **Induk:** [story.plan.md](./story.plan.md) → [epic.plan.md](./epic.plan.md) · **Sumber:** [PRD.md](./PRD.md) v1.3 · [Design.md](./Design.md) v1.3.
> **Layer:** Lapis terbawah & paling eksekutif. Tiap **Task** memecah satu **Story** jadi langkah mikro yang **tidak menyisakan keputusan arsitektur**. Ditujukan agar bisa dieksekusi model hemat (mis. Sonnet 4.6) namun hasil konsisten setara model besar — karena seluruh keputusan sudah dikunci di sini & di §A (Pola Reusable).
> **Warisan:** seluruh konvensi, stack, struktur repo, koreksi v1.3 (**tanpa `organization_id`**, tanpa self-signup) diwarisi dari `epic.plan.md §1`. **Pola kode reusable di §A wajib diikuti** — task yang berulang cukup menyebut "ikuti Pola P-x".

**Format task:** `- [ ] TK-NN.s.t — <aksi>. **File:** <path>. **Do:** <langkah>. **Done:** <cara verifikasi>.`
**Aturan eksekusi:** kerjakan task **berurutan** dalam satu story; jangan loncat dependency; setiap selesai, jalankan **Done-check** sebelum lanjut. Bila Done-check gagal, perbaiki dulu.

---

## §A. Pola Reusable (WAJIB dipakai — jangan reinvent)

### Perintah standar
- Backend run: `go run ./apps/api` · test: `go test ./...` · lint: `golangci-lint run` · vet: `go vet ./...`
- Migrasi: `migrate -path db/migrations -database "$DATABASE_URL" up` (golang-migrate).
- Frontend: `cd apps/web && npm run dev` · build: `npm run build` · `npm run check` (tsc+eslint).
- Full stack: `docker compose -f deploy/docker-compose.yml up`.

### P-1 — Migrasi SQL (pasangan up/down)
File: `db/migrations/000N_<name>.up.sql` + `.down.sql`. Aturan:
- Semua tabel punya `id UUID PRIMARY KEY DEFAULT gen_random_uuid()`, `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`, `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`. **TANPA `organization_id`.**
- Enum pakai `TEXT` + `CHECK (col IN (...))` (hindari Postgres ENUM type agar fleksibel).
- `jsonb` untuk evidence/risk_flags/content/tool_calls. Index pada FK & kolom filter (status, deadline, stage, dedup_key unik).
- `.down.sql` `DROP TABLE IF EXISTS ... CASCADE;` urutan terbalik.

### P-2 — Domain entity (`internal/domain/<entity>.go`)
Struct Go dengan tag `json` & `gorm`; enum sebagai `type XStatus string` + konstanta + `func (s XStatus) Valid() bool`. Tanpa import infra. Definisikan port interface repo di sini bila perlu (`type TenderRepository interface { ... }`).

### P-3 — Repository (`internal/repository/<entity>_repo.go`)
Struct `XRepo{ db *gorm.DB }` + `NewXRepo`. Method: `Create/GetByID/List(filter)/Update/Delete`. `List` terima struct filter + pagination `(page,pageSize)` → `(items, total, error)`. Pakai `ctx`. Error wrap `fmt.Errorf("...: %w", err)`.

### P-4 — DTO (`internal/http/dto/<entity>.go`)
`XCreateRequest`, `XUpdateRequest`, `XResponse`, `XListResponse{Items,Total,Page,PageSize}`. Validasi via tag `validate:"required,..."` (go-playground/validator) + `Bind`+`Validate` di handler.

### P-5 — Handler (`internal/http/handlers/<entity>_handler.go`)
Struct `XHandler{ svc *service.XService }`. Method per endpoint terima `echo.Context`:
1. bind+validate DTO → 2. ambil user dari ctx (RBAC sudah di middleware) → 3. panggil service → 4. map ke Response → 5. `return c.JSON(status, resp)`.
Error → helper `httperr.Write(c, err)` menghasilkan `{"error":{"code","message"}}`. Daftarkan route di `internal/http/router.go` dengan middleware RBAC yang sesuai matriks §3.1.

### P-6 — Service (`internal/service/<entity>_service.go`)
Logika bisnis & validasi lintas-entity; panggil repo; emit telemetry/audit/hook bila relevan. AI **non-blocking**: error AI dibungkus jadi pesan ramah, tidak menggagalkan CRUD.

### P-7 — React page CRUD (`apps/web/src/pages/<area>/`)
- Query hooks di `src/api/<area>.ts` pakai TanStack Query (`useXList`, `useX`, `useCreateX`, `useUpdateX`, `useDeleteX`); key `['x', params]`; mutation → `invalidateQueries`.
- List page: filter bar + `Table`/board + Empty/Skeleton + Error toast. Form: `Drawer` + komponen UI §2.4 + validasi inline. Optimistic update untuk aksi cepat (stage, dsb) dengan rollback `onError`.
- Semua teks Bahasa Indonesia; angka/tanggal via `src/lib/format.ts`.

### P-8 — AI service (`internal/ai/<feature>.go`)
1. Build prompt dari data + Company Profile (`get` via repo) + rubrik/template. 2. Panggil `hermes.GenerateJSON(ctx, prompt, schema, sessionKey)` (sessionKey = `cfg.WorkspaceSessionKey`). 3. Validasi output ke schema; retry 1x bila invalid. 4. Persist + audit (model, waktu, evidence). 5. Return typed result. Gagal → `ErrAIUnavailable` (pesan ramah).

### P-9 — MCP tool (`internal/mcp/tools_*.go`)
Daftarkan tool dengan nama stabil + JSON schema input/output (aditif, jangan rename). Read tool: query repo, kembalikan ringkas. Write tool: hanya bila di whitelist `tools.include`, tulis audit, jangan aksi final tanpa konteks human-in-the-loop. Tambah ke `deploy/hermes/config.yaml.example`.

### P-10 — Komponen UI (`apps/web/src/components/ui/<Name>.tsx`)
Props TS eksplisit; varian via prop union; state §2.4; pakai token Tailwind (jangan hardcode hex); ekspor dari `src/components/ui/index.ts`. Sertakan di katalog `/dev/components`.

### Done-check umum
Setiap task FE: `npm run check` hijau + render manual sesuai Design. Setiap task BE: `go build ./...` + `go vet` hijau + unit test (bila ada) hijau + endpoint diuji via `curl`/REST client.

---

## EP-00 — Fondasi Proyek, Monorepo & DevOps

### ST-00.1 — Inisialisasi monorepo
- [x] TK-00.1.1 — Buat struktur folder. **File:** seluruh tree epic §1. **Do:** buat folder `apps/api`, `apps/web`, `internal/{config,auth,domain,repository,service,http/{handlers,dto},hermes,ai,mcp,telemetry}`, `db/migrations`, `deploy/hermes`. **Done:** `tree -L 2` cocok dengan epic §1.
- [x] TK-00.1.2 — Init module Go. **File:** `go.mod`. **Do:** `go mod init salespilot`; set `go 1.22`. **Done:** `go mod verify` ok.
- [x] TK-00.1.3 — Tambah `.gitignore`, `.editorconfig`, `LICENSE`(internal). **File:** root. **Do:** ignore `node_modules`, `*.env`, `dist`, `tmp`, binari. **Done:** file ada.

### ST-00.2 — Echo bootstrap + healthz + config
- [x] TK-00.2.1 — Config loader. **File:** `internal/config/config.go`. **Do:** struct `Config{ Port, DatabaseURL, JWTSecret, HermesBaseURL, APIServerKey, WorkspaceSessionKey, SalesMCPToken, SeedAdminEmail, SeedAdminPassword }`; load dari env (godotenv); fungsi `MustLoad()` fatal bila wajib kosong. **Done:** unit test load dari env map.
- [x] TK-00.2.2 — Router + middleware. **File:** `internal/http/router.go`. **Do:** `New(echo)` daftarkan middleware Logger, Recover, CORS, RequestID; route `GET /healthz`. **Done:** kompil.
- [x] TK-00.2.3 — main.go. **File:** `apps/api/main.go`. **Do:** load config → init echo → router → `e.Start(:Port)`; graceful shutdown. **Done:** `go run ./apps/api` → `curl localhost:8080/healthz` = `{"status":"ok"}`.

### ST-00.3 — Vite + TS shell
- [x] TK-00.3.1 — Scaffold Vite. **File:** `apps/web/`. **Do:** `npm create vite@latest . -- --template react-ts`; install deps dasar. **Done:** `npm run dev` membuka halaman.
- [x] TK-00.3.2 — Dev proxy. **File:** `apps/web/vite.config.ts`. **Do:** proxy `/api`→`http://localhost:8080`. **Done:** fetch `/api/healthz` dari web sukses.
- [x] TK-00.3.3 — Bersihkan template + App placeholder. **File:** `src/App.tsx`, `src/main.tsx`. **Do:** hapus boilerplate, render "SalesPilot". **Done:** `npm run build` & `tsc --noEmit` hijau.

### ST-00.4 — Postgres + GORM + migrate
- [x] TK-00.4.1 — DB connect. **File:** `internal/repository/db.go`. **Do:** `Open(cfg)` GORM postgres + pool + retry 5x; `Ping`. **Done:** boot log "db connected".
- [x] TK-00.4.2 — Migrasi awal. **File:** `db/migrations/0001_init.up.sql`/`.down.sql`. **Do:** `CREATE EXTENSION IF NOT EXISTS pgcrypto;` (untuk `gen_random_uuid`). **Done:** `migrate ... up` sukses.
- [x] TK-00.4.3 — Healthz cek DB. **File:** router/handler. **Do:** healthz panggil `db.Ping`. **Done:** matikan DB → healthz 503.

### ST-00.5 — docker-compose + env + README
> **Koreksi v1.4:** service `hermes-gateway` di sini di-rework jadi `hermes-bridge` (build sendiri, bukan image publik) di TK-01.6.8 — lihat [hermes-bridge.plan.md](./hermes-bridge.plan.md).
- [x] TK-00.5.1 — Compose. **File:** `deploy/docker-compose.yml`. **Do:** services `postgres:16` (volume), `api` (build apps/api), `web` (build apps/web), `hermes-gateway` (image di-pin + mount config). Env wiring. **Done:** `docker compose config` valid.
- [x] TK-00.5.2 — Hermes config example. **File:** `deploy/hermes/config.yaml.example`, `.env.example`. **Do:** blok `mcp_servers.sales` (placeholder url `http://api:8080/mcp`), `memory.provider: holographic`, `API_SERVER_KEY`. **Done:** file ada + komentar.
- [x] TK-00.5.3 — Root `.env.example` + README. **File:** `.env.example`, `README.md`. **Do:** semua env (§ST-00.2.1) + run guide Docker & native Windows. **Done:** ikut README → stack hidup.

### ST-00.6 — Tooling
- [x] TK-00.6.1 — golangci-lint. **File:** `.golangci.yml`. **Do:** enable govet, staticcheck, errcheck, gofmt. **Done:** `golangci-lint run` hijau.
- [x] TK-00.6.2 — ESLint/Prettier/tsconfig strict. **File:** `apps/web/.eslintrc.cjs`, `.prettierrc`, `tsconfig.json`. **Do:** strict true. **Done:** `npm run check` hijau.
- [x] TK-00.6.3 — Script agregat. **File:** `Makefile`, `apps/web/package.json`. **Do:** `make check` = vet+lint+test; `npm run check`. **Done:** keduanya hijau.

---

## EP-01 — Anti-Corruption Layer & Jembatan Hermes

### ST-01.1 — Client interface + tipe
- [x] TK-01.1.1 — Tipe domain. **File:** `internal/hermes/client.go`. **Do:** definisikan `SessionKey string`; `Message{Role,Content,ToolCalls}`; `ChatRequest{Messages,Stream,SessionKey,SessionID}`; `ChatResponse{Content,ToolCalls}`; `Chunk{Delta,ToolCall,Done,Err}`; `Capabilities{Version,Models,Features}`. **Done:** kompil.
- [x] TK-01.1.2 — Interface. **File:** `internal/hermes/client.go`. **Do:** `type Client interface { Chat(ctx,ChatRequest)(ChatResponse,error); ChatStream(ctx,ChatRequest)(<-chan Chunk,error); GenerateJSON(ctx,prompt string,schema any,sk SessionKey)(json.RawMessage,error); Health(ctx)(Capabilities,error) }`. **Done:** kompil; tidak ada import internal Hermes.
- [x] TK-01.1.3 — Constructor impl. **File:** `internal/hermes/http_client.go`. **Do:** `type httpClient struct{ baseURL, apiKey string; hc *http.Client }`; `New(cfg) Client`; helper `newReq` set header `Authorization`, `X-Hermes-Session-Key`, `X-Hermes-Session-Id`. **Done:** kompil.

### ST-01.2 — Chat impl
- [x] TK-01.2.1 — Non-stream Chat. **File:** `internal/hermes/chat.go`. **Do:** POST `/v1/chat/completions` (`stream:false`), parse choice→`ChatResponse`. **Done:** test integrasi tandai `Contract`.
- [x] TK-01.2.2 — Stream ChatStream. **File:** `internal/hermes/chat.go`. **Do:** POST `stream:true`; baca SSE `data:` lines; kirim `Chunk` ke channel; tutup channel di akhir/`[DONE]`; hormati `ctx.Done()` (cancel/Stop). **Done:** stream channel menerima delta dari Hermes nyata.
- [x] TK-01.2.3 — Error & timeout. **File:** chat.go. **Do:** timeout via ctx; non-2xx → error berisi body; channel kirim `Chunk{Err}`. **Done:** matikan Hermes → error tertangani, tidak panic.

### ST-01.3 — GenerateJSON
- [x] TK-01.3.1 — Impl. **File:** `internal/hermes/generate.go`. **Do:** pakai `/v1/responses` atau chat dengan instruksi JSON + `response_format` json_schema; kembalikan `json.RawMessage`. **Done:** kembalikan JSON valid untuk schema contoh.
- [x] TK-01.3.2 — Validasi + retry. **File:** generate.go. **Do:** unmarshal ke `schema`; bila gagal, retry 1x dengan instruksi "valid JSON only". **Done:** output non-JSON dipulihkan/diberi error jelas.

### ST-01.4 — Health guard
- [x] TK-01.4.1 — Health. **File:** `internal/hermes/health.go`. **Do:** GET `/health` + `/v1/capabilities` → `Capabilities`. **Done:** kembalikan versi & model.
- [x] TK-01.4.2 — Startup guard. **File:** `apps/api/main.go`. **Do:** panggil `Health` saat boot; log status; simpan flag `aiAvailable`; jangan crash bila gagal. **Done:** boot tanpa Hermes tetap jalan, log warning.

### ST-01.5 — Contract tests + config
- [x] TK-01.5.1 — Contract test. **File:** `internal/hermes/contract_test.go`. **Do:** `//go:build contract`; uji ChatStream, GenerateJSON, Health terhadap `HERMES_BASE_URL` (= hermes-bridge). **Done:** `go test -tags contract ./internal/hermes/...` hijau bila bridge hidup.
- [x] TK-01.5.2 — Pin versi + env. **File:** `deploy/docker-compose.yml`, README. **Do:** pin versi lib `hermes-agent` di `services/hermes-bridge/pyproject.toml`; dokumen upgrade = ganti versi + run contract. **Done:** compose & pyproject pakai versi tetap.

### ST-01.6 — hermes-bridge (jembatan Python ke Hermes Agent) — detail: [hermes-bridge.plan.md](./hermes-bridge.plan.md)
> **Koreksi arsitektur v1.4:** Hermes Agent (Nous Research) diakses via **Python library** (`AIAgent`), bukan image gateway OpenAI-compatible. `hermes-bridge` (FastAPI) membungkus `AIAgent` & mengekspos HTTP `/v1` tiruan agar `internal/hermes` (ACL Go) tak berubah. Kontrak = OpenAI `/v1`; streaming = MVP blocking-then-send. Double isolation: perubahan Hermes hanya menyentuh bridge.
- [x] TK-01.6.1 — Scaffold + health. **File:** `services/hermes-bridge/` (`pyproject.toml`, `app/main.py`). **Do:** FastAPI + `GET /health`; pin `hermes-agent`, `fastapi`, `uvicorn`. **Done:** `uvicorn` hidup, `/health` ok. (BR-1)
- [x] TK-01.6.2 — Config + auth Bearer. **File:** `app/config.py`, `app/auth.py`. **Do:** env (`API_SERVER_KEY`, `HERMES_MODEL`, provider keys, toolsets, PORT); dependency cek `Authorization`. **Done:** Bearer salah → 401. (BR-2)
- [x] TK-01.6.3 — Agent factory. **File:** `app/agent_factory.py`. **Do:** bangun `AIAgent` per-request (`quiet_mode=True`, model, `enabled_toolsets`, matikan `terminal`). **Done:** unit test buat agent tanpa error. (BR-3)
- [x] TK-01.6.4 — Chat non-stream. **File:** `app/routes/chat.py`. **Do:** `POST /v1/chat/completions` stream=false → `run_conversation` (pisah messages→history+user) → OpenAI-shaped. **Done:** `curl` balas content. (BR-4)
- [x] TK-01.6.5 — Chat stream (MVP). **File:** chat.py. **Do:** stream=true → blocking → SSE `data:` chunk + `[DONE]`. **Done:** `curl -N` terima chunk lalu `[DONE]`; ACL `ChatStream` jalan. (BR-5)
- [x] TK-01.6.6 — Capabilities + error. **File:** `app/routes/health.py`. **Do:** `GET /v1/capabilities` (versi+model); bungkus error provider → JSON ramah, worker tak crash. **Done:** provider down → 5xx JSON, service hidup. (BR-6)
- [x] TK-01.6.7 — responses (GenerateJSON) skeleton. **File:** `app/routes/responses.py`. **Do:** `POST /v1/responses` minimal (ephemeral_system_prompt JSON, toolsets off). **Done:** balas JSON untuk schema contoh (disempurnakan di TK-01.3). (BR-7)
- [x] TK-01.6.8 — Dockerfile + compose wiring. **File:** `services/hermes-bridge/Dockerfile`, `deploy/docker-compose.yml`. **Do:** ganti service `hermes-gateway` → `hermes-bridge` (build + env); `HERMES_BASE_URL` Go → `http://hermes-bridge:PORT`. **Done:** `docker compose config` valid; stack up; Go `Health()` hijau. (BR-8)
- [x] TK-01.6.9 — Bridge `/admin/config` (override provider runtime). **File:** `app/routes/admin.py`, `app/agent_factory.py`. **Do:** `POST /admin/config` (auth Bearer) set in-memory provider ∈ {`openai`,`openrouter`}, model, base_url, api_key; `agent_factory` pakai config aktif, **fallback env** `OPENAI_API_KEY`/`OPENROUTER_API_KEY` + `HERMES_MODEL`. **Done:** set config → chat pakai provider itu; tanpa config → pakai env. (BR-9)
- [x] TK-01.6.10 — ACL `Configure`. **File:** `internal/hermes/client.go` (interface), `internal/hermes/configure.go`. **Do:** tambah `Configure(ctx, ProviderConfig) error` ke `Client` (additive) + impl `POST /admin/config`; `ProviderConfig{Provider,Model,BaseURL,APIKey}`. **Done:** `go build`/`vet` hijau; unit test httptest kirim header & body benar. (BR-10)

---

## EP-02 — Design System & Application Shell

### ST-02.1 — Tailwind + tokens
- [x] TK-02.1.1 — Install Tailwind. **File:** `apps/web` (`tailwind.config.ts`, `postcss.config.js`, `src/index.css`). **Do:** setup Tailwind; import. **Done:** kelas Tailwind bekerja.
- [x] TK-02.1.2 — Token warna. **File:** `tailwind.config.ts`. **Do:** `theme.extend.colors`: primary `#4F46E5`/`#4338CA`, accent violet `#7C3AED`, success `#10B981`, warning `#F59E0B`, danger `#EF4444`, info/sky `#0EA5E9`, neutral/surface/border per §2.1. **Done:** token resolve di kelas.
- [x] TK-02.1.3 — Tipografi & spacing & radius & shadow. **File:** `tailwind.config.ts`, `tokens.css`. **Do:** font Inter (import), skala H1–caption (§2.2), spacing 4/8/12/16/24/32/48, radius card 12/btn 8/pill 999, shadow subtle. **Done:** sesuai §2.3.
- [x] TK-02.1.4 — Skala warna helper. **File:** `src/lib/score.ts`. **Do:** `scoreColor(n)` (0–49 rose,50–64 sky,65–79 amber,80–100 emerald); `actionColor(action)` (Pursue emerald…Need Partner violet). **Done:** unit test mapping.

### ST-02.2 — Komponen form/atom (ikuti P-10)
- [x] TK-02.2.1 — Button. **File:** `ui/Button.tsx`. **Do:** varian primary/secondary/ghost/danger; size sm/md/lg; ikon; loading. **Done:** semua state §2.4 di `/dev/components`.
- [x] TK-02.2.2 — Input/Textarea/Select/DatePicker/Combobox. **File:** `ui/*.tsx`. **Do:** state error+helper, focus ring. **Done:** render + validasi visual.
- [x] TK-02.2.3 — Chip/Tag input (preset+tambah). **File:** `ui/ChipInput.tsx`. **Do:** multi-select preset + free add + remove. **Done:** tambah/hapus chip.
- [x] TK-02.2.4 — Toggle/Switch + Badge/Pill. **File:** `ui/Toggle.tsx`, `ui/Badge.tsx`. **Do:** Badge varian status/AI/confidence/recommended_action. **Done:** warna ikut token.

### ST-02.3 — Komponen struktur/feedback
- [x] TK-02.3.1 — Card, Tabs, Breadcrumb, Avatar, Tooltip. **File:** `ui/*`. **Done:** render.
- [x] TK-02.3.2 — Table (sortable/sticky/pagination/kebab). **File:** `ui/Table.tsx`. **Do:** generic columns + sort + pagination. **Done:** sort & paging jalan.
- [x] TK-02.3.3 — Modal, Drawer, Toast, Confirmation dialog. **File:** `ui/*`, `src/lib/toast.ts`. **Done:** open/close + focus trap.
- [x] TK-02.3.4 — Skeleton + Empty state. **File:** `ui/Skeleton.tsx`, `ui/EmptyState.tsx`. **Do:** EmptyState props (ikon, judul, manfaat, CTA). **Done:** render.

### ST-02.4 — Komponen AI/khusus
- [x] TK-02.4.1 — Score ring/gauge. **File:** `ui/ScoreRing.tsx`. **Do:** SVG ring, warna via `scoreColor`, angka tabular. **Done:** 48→rose, 86→emerald.
- [x] TK-02.4.2 — Stat card + AI panel/callout. **File:** `ui/StatCard.tsx`, `ui/AiCallout.tsx`. **Do:** AiCallout aksen violet + sparkles + "Lihat alasan". **Done:** render.
- [x] TK-02.4.3 — Streaming text + Risk-flag chip. **File:** `ui/StreamingText.tsx`, `ui/RiskFlag.tsx`. **Do:** StreamingText animasi caret; RiskFlag amber/rose ⚠. **Done:** render.
- [x] TK-02.4.4 — File dropzone + Stepper. **File:** `ui/FileDropzone.tsx`, `ui/Stepper.tsx`. **Do:** dropzone PDF (drag+pick), progress slot. **Done:** pilih file memicu callback.

### ST-02.5 — Application Shell
- [x] TK-02.5.1 — Layout. **File:** `src/layout/AppShell.tsx`. **Do:** grid sidebar+topbar+konten. **Done:** responsif desktop/tablet (§7).
- [x] TK-02.5.2 — Sidebar. **File:** `src/layout/Sidebar.tsx`. **Do:** item urut §3.1 + ikon lucide + badge (Penemuan AI/Chat) + collapsible + divider sebelum Otak Agent. **Done:** navigasi aktif highlight.
- [x] TK-02.5.3 — Topbar. **File:** `src/layout/Topbar.tsx`. **Do:** breadcrumb/judul, search ⌘K (placeholder), +New kontekstual, bell popover, avatar menu. **Done:** render.
- [x] TK-02.5.4 — Routing. **File:** `src/routes.tsx`. **Do:** route semua halaman (placeholder), wrap `RequireAuth` (diisi EP-03). **Done:** navigasi antar route.

### ST-02.6 — Format & i18n & a11y
- [x] TK-02.6.1 — Format. **File:** `src/lib/format.ts`. **Do:** `formatRupiah`, `formatRupiahShort` (`Rp 2,5 M`/`Rp 300 jt`), `formatTanggal` (`24 Jun 2026`), `formatRelative` ("2 jam lalu"). **Done:** unit test id-ID.
- [x] TK-02.6.2 — i18n strings + a11y. **File:** `src/lib/i18n.ts`, `src/lib/a11y.ts`. **Do:** kumpulan label ID; helper live-region. **Done:** kontras AA & focus ring tampak.

---

## EP-03 — Auth, RBAC & User Management

### ST-03.1 — Entity user + seed (P-1, P-2, P-3)
- [x] TK-03.1.1 — Migrasi user. **File:** `db/migrations/0002_users.up.sql`. **Do:** tabel `user` (email unik, password_hash, name, role CHECK in SALES/OPS/MANAGER/ADMIN, active bool). **Done:** migrate up/down ok.
- [x] TK-03.1.2 — Domain + repo. **File:** `internal/domain/user.go`, `internal/repository/user_repo.go`. **Do:** struct + `Role` type + repo CRUD + `GetByEmail`. **Done:** kompil.
- [x] TK-03.1.3 — Seed admin. **File:** `internal/repository/seed.go` (+ panggil di main). **Do:** buat admin dari `SEED_ADMIN_*` bila belum ada. **Done:** boot → admin tersedia.

### ST-03.2 — Login + JWT
- [x] TK-03.2.1 — Password util. **File:** `internal/auth/password.go`. **Do:** `Hash`, `Verify` (bcrypt cost 12). **Done:** unit test.
- [x] TK-03.2.2 — JWT. **File:** `internal/auth/jwt.go`. **Do:** `Issue(user)`/`Parse(token)` claims sub/role/exp; secret dari config; access + refresh. **Done:** unit test round-trip.
- [x] TK-03.2.3 — Login handler. **File:** `internal/http/handlers/auth_handler.go`. **Do:** `POST /api/auth/login`, `/api/auth/refresh`; **tidak ada register**. **Done:** `curl` login valid→token, salah→401.

### ST-03.3 — Middleware RBAC
- [x] TK-03.3.1 — Auth middleware. **File:** `internal/auth/middleware.go`. **Do:** verifikasi Bearer JWT → set user di ctx; 401 bila invalid. **Done:** route terproteksi menolak tanpa token.
- [x] TK-03.3.2 — RBAC map. **File:** `internal/auth/rbac.go`. **Do:** peta capability→roles dari matriks §3.1 (mis. `EditProfile: {OPS,MANAGER,ADMIN}`, `RunDiscovery: {OPS,MANAGER,ADMIN}`, `ManageUsers: {ADMIN}`); `RequireCapability(cap)` middleware. **Done:** unit test tiap capability per role.

### ST-03.4 — Admin user mgmt
- [x] TK-03.4.1 — Handler. **File:** `internal/http/handlers/user_handler.go`. **Do:** `GET/POST /api/users`, `PATCH /api/users/:id`, `POST /api/users/:id/reset-password` — middleware `RequireCapability(ManageUsers)`. **Done:** non-admin→403; admin sukses.
- [x] TK-03.4.2 — Guard last admin. **File:** user service. **Do:** cegah nonaktifkan/turunkan admin terakhir. **Done:** uji menolak.

### ST-03.5 — FE Login + guard (P-7)
- [x] TK-03.5.1 — API client. **File:** `src/lib/api.ts`. **Do:** fetch wrapper + inject token + handle 401→silent refresh→logout. **Done:** request ber-Authorization.
- [x] TK-03.5.2 — Auth store. **File:** `src/store/auth.ts`. **Do:** Zustand: token, user, login/logout, persist. **Done:** reload tetap login.
- [x] TK-03.5.3 — Login page. **File:** `src/pages/Login.tsx`. **Do:** Design §4.1 (gradient card, email/password toggle, Masuk, error inline, loading); **tanpa link daftar**; catatan "Akun dikelola Admin internal". **Done:** login → redirect dashboard.
- [x] TK-03.5.4 — RequireAuth + role UI. **File:** `src/components/RequireAuth.tsx`, helper `useCan(cap)`. **Do:** guard route; sembunyikan aksi sesuai role. **Done:** SALES tak melihat tombol terlarang.

---

## EP-04 — Chat Assistant

### ST-04.1 — Entity conversation/message
- [x] TK-04.1.1 — Migrasi. **File:** `0003_chat.up.sql`. **Do:** tabel `conversation`, `message` (P-1; tool_calls jsonb). **Done:** migrate ok.
- [x] TK-04.1.2 — Domain + repo. **File:** `domain/conversation.go`, `repository/chat_repo.go`. **Done:** kompil.

### ST-04.2 — Create conversation + session key
- [x] TK-04.2.1 — Handler create. **File:** `handlers/chat_handler.go`. **Do:** `POST /api/conversations` set `session_key=cfg.WorkspaceSessionKey`, `hermes_session_id=uuid`. **Done:** row tercipta.
- [x] TK-04.2.2 — Auto title. **File:** chat service. **Do:** judul dari 6 kata pertama pesan user. **Done:** judul terisi.

### ST-04.3 — Chat SSE relay
- [x] TK-04.3.1 — Endpoint relay. **File:** chat_handler.go. **Do:** `POST /api/conversations/:id/chat`: simpan msg user → `hermes.ChatStream` → set header SSE → flush tiap chunk → kumpulkan assistant content+tool_calls → simpan. **Done:** browser menerima stream.
- [x] TK-04.3.2 — Stop & error. **File:** chat_handler.go. **Do:** ctx cancel saat client disconnect; Hermes error → SSE event `error` ramah. **Done:** Stop menghentikan; down → banner.

### ST-04.4 — History
- [x] TK-04.4.1 — List/get. **File:** chat_handler.go. **Do:** `GET /api/conversations`, `GET /api/conversations/:id` (+messages); filter milik user. **Done:** data benar.

### ST-04.5 — FE Chat UI (P-7)
- [x] TK-04.5.1 — SSE parser. **File:** `src/lib/sse.ts`. **Do:** fetch-stream reader → yield event. **Done:** delta ter-render incremental.
- [x] TK-04.5.2 — Chat page. **File:** `src/pages/Chat.tsx`. **Do:** Design §4.12 layout (list percakapan+search, area pesan). **Done:** kirim→stream.
- [x] TK-04.5.3 — Bubble + tool-call chip. **File:** `src/components/chat/*`. **Do:** assistant violet+sparkles+markdown; tool-call chip "🔧 Membaca data tender…"→"✓ N" expandable. **Done:** chip tampil saat tool dipakai.
- [x] TK-04.5.4 — Input + Stop + suggested chips + context chip. **File:** Chat.tsx. **Do:** auto-grow, Enter kirim, Stop, chips ("Tender prioritas minggu ini?", "Ringkas pipeline", "Cari tender baru sekarang"), context chip dari detail. **Done:** semua interaktif.

### ST-04.6 — Floating Tanya AI + degrade
- [x] TK-04.6.1 — Drawer. **File:** `src/components/AskAIDrawer.tsx`. **Do:** floating button → slide-over chat; terima context (tender/prospect). **Done:** dari detail muncul context chip.
- [x] TK-04.6.2 — Degrade banner. **File:** Chat/Drawer. **Do:** bila status Hermes down → banner "Agent tidak tersedia"; CRUD tetap jalan. **Done:** matikan Hermes → banner.

---

## EP-05 — Tender Management

### ST-05.1 — Entity tender (P-1,P-2,P-3)
- [x] TK-05.1.1 — Migrasi. **File:** `0004_tenders.up.sql`. **Do:** tabel `tender` semua field §10; enum `status`,`recommended_action`,`origin` via CHECK; `risk_flags jsonb`; index status/deadline/`dedup_key`(unique). **Done:** migrate ok.
- [x] TK-05.1.2 — Domain + enum valid. **File:** `domain/tender.go`. **Do:** struct + `TenderStatus`/`RecommendedAction`/`Origin` + `Valid()`. **Done:** unit test enum.
- [x] TK-05.1.3 — Repo + filter. **File:** `repository/tender_repo.go`. **Do:** P-3 + filter struct (status,buyer,deadline range,recommended_action,origin,search). **Done:** list terfilter.

### ST-05.2 — CRUD endpoints (P-4,P-5,P-6)
- [x] TK-05.2.1 — DTO + validasi. **File:** `dto/tender.go`. **Do:** Create min `title`; value ≥0. **Done:** invalid→422.
- [x] TK-05.2.2 — Service. **File:** `service/tender_service.go`. **Do:** CRUD + list. **Done:** kompil.
- [x] TK-05.2.3 — Handler + route. **File:** `handlers/tender_handler.go`, router. **Do:** `GET/POST /api/tenders`, `GET/PUT/DELETE /api/tenders/:id`; RBAC CRUD (semua role). **Done:** `curl` CRUD sukses; delete→204.

### ST-05.3 — Status + outcome
- [x] TK-05.3.1 — Transisi status. **File:** tender_service.go. **Do:** `PATCH /api/tenders/:id/status` validasi transisi sah. **Done:** transisi ilegal→400.
- [x] TK-05.3.2 — Outcome. **File:** tender_handler.go + `0005_outcome_events.up.sql` + `domain/outcome.go` + `repository/outcome_repo.go`. **Do:** `POST /api/tenders/:id/outcome` (WON/LOST+notes) → INSERT `outcome_event` + set status terminal + emit hook learning (no-op). **Catatan:** TK-16.1.1 superseded; migrasi EP-06+ bergeser +1. **Done:** row outcome tercipta, `go test ./... hijau`.

### ST-05.4 — Promote
- [x] TK-05.4.1 — Service promote. **File:** tender_service.go. **Do:** `Promote(id)` set keluar inbox (mis. flag/`status=QUALIFYING`) tanpa hapus field AI; endpoint `POST /api/tenders/:id/promote`. **Done:** tender muncul di pipeline.

### ST-05.5 — FE List (P-7)
- [x] TK-05.5.1 — Query hooks. **File:** `src/api/tenders.ts`. **Done:** hooks jalan.
- [x] TK-05.5.2 — List page. **File:** `src/pages/tenders/TenderList.tsx`. **Do:** Design §4.4 filter+kolom (Fit Score mini ring, Rekomendasi badge, Origin ✨, kebab). **Done:** render + filter + empty/loading.

### ST-05.6 — FE Detail
- [x] TK-05.6.1 — Detail layout + tabs. **File:** `src/pages/tenders/TenderDetail.tsx`. **Do:** Design §4.5 ringkasan + tabs Ringkasan/Analisa AI/Playbook/Timeline + tombol Edit/Analisa/Playbook/WON-LOST. **Done:** render data.
- [x] TK-05.6.2 — Panel Analisa AI placeholder. **File:** Detail. **Do:** slot AiScorePanel (diisi EP-10) + origin link sumber asli + risk chips. **Done:** placeholder rapi.

### ST-05.7 — FE Form drawer
- [x] TK-05.7.1 — Form. **File:** `src/pages/tenders/TenderFormDrawer.tsx`. **Do:** Design §4.6 field + "Simpan & Analisa AI" (hook ke EP-10). **Done:** create/edit tersimpan.

---

## EP-06 — Event Management

### ST-06.1–06.2 — Entity + CRUD (ikuti P-1..P-6, pola EP-05)
- [x] TK-06.1.1 — Migrasi `0006_events.up.sql` (`event` field §10; type EXPO/CONFERENCE/SEMINAR/WORKSHOP/NETWORKING/OTHER; status PLANNED/ATTENDED/CANCELLED). **Done:** migrate ok. (**Koreksi:** nomor 0006 karena 0005 sudah dipakai outcome_events di ST-05.3.2)
- [x] TK-06.1.2 — Domain+repo `event`. **Done:** kompil.
- [x] TK-06.2.1 — DTO+service+handler `GET/POST /api/events`,`GET/PUT/DELETE /api/events/:id` (min name+type). **Done:** CRUD `curl` ok.

### ST-06.3 — Konversi → prospect
- [x] TK-06.3.1 — Endpoint convert. **File:** event_handler.go. **Do:** `POST /api/events/:id/convert` buat prospect (`source_type=event`,`source_id`). **Done:** prospect tertaut. (**Catatan:** sekaligus membuat migrasi `0007_prospects.up.sql` + domain Prospect minimal + ProspectRepo; EP-07 memperluas)

### ST-06.4 — FE (P-7)
- [x] TK-06.4.1 — List+Detail+Form. **File:** `src/pages/events/*`. **Do:** Design §4.7; tombol "+ Konversi ke Prospek". **Done:** konversi membuat prospect.

---

## EP-07 — Prospect Management & Pipeline

### ST-07.1–07.2 — Entity + CRUD
- [ ] TK-07.1.1 — Migrasi `0006_prospects.up.sql` (`prospect` field §10; stage CHECK). **Done:** migrate ok.
- [ ] TK-07.1.2 — Domain+repo (`ProspectStage` valid). **Done:** kompil.
- [ ] TK-07.2.1 — DTO+service+handler CRUD + `PATCH /api/prospects/:id/stage`; WON/LOST emit outcome hook. **Done:** stage tersimpan.

### ST-07.3 — FE Kanban
- [ ] TK-07.3.1 — Board. **File:** `src/pages/prospects/ProspectBoard.tsx`. **Do:** Design §4.8 kolom+header(jumlah+total nilai)+kartu(score ring/owner/badge sumber). **Done:** render board.
- [ ] TK-07.3.2 — Drag-drop. **File:** Board (dnd-kit). **Do:** drag pindah stage → optimistic `PATCH` + rollback `onError`. **Done:** pindah tersimpan; gagal→rollback.
- [ ] TK-07.3.3 — Toggle Board↔Table + filter. **File:** Board. **Do:** filter owner/sumber/min skor. **Done:** toggle & filter jalan.

### ST-07.4 — FE Detail drawer
- [ ] TK-07.4.1 — Drawer. **File:** `src/pages/prospects/ProspectDrawer.tsx`. **Do:** Design §4.9 sections + aksi cepat (stage, WON/LOST, "Tanya AI tentang prospek ini"→AskAIDrawer context). **Done:** render + aksi.

---

## EP-08 — Knowledge / Company Profile

### ST-08.1 — Entities + versioning
- [ ] TK-08.1.1 — Migrasi `0007_profile.up.sql`. **Do:** `company_profile`,`target_criteria`,`nogo_rule`,`source`,`keyword_set` (field §10) + `version int` + `is_current bool`. **Done:** migrate ok.
- [ ] TK-08.1.2 — Domain+repo. **File:** `domain/profile.go`, `repository/profile_repo.go`. **Do:** simpan versi baru = clone+increment, set `is_current`. **Done:** versi bertambah.

### ST-08.2 — Read/write + defaults
- [ ] TK-08.2.1 — Defaults/preset. **File:** profile service. **Do:** default value_min Rp 1e9, deadline_min_days 7, countries=[Indonesia], procurement preset. **Done:** profil baru terisi default.
- [ ] TK-08.2.2 — Endpoints. **File:** `handlers/profile_handler.go`. **Do:** `GET /api/profile` (current), `PUT /api/profile` (versi baru) — RBAC `EditProfile`. **Done:** SALES read-only, OPS+ bisa edit.

### ST-08.3 — Source mgmt
- [ ] TK-08.3.1 — CRUD source. **File:** `handlers/source_handler.go`. **Do:** CRUD + validasi URL + access enum; preset Indonesia (SPSE/LKPP, eProc PLN, Pertamina, Telkom/SMILE, PaDi) 1-klik aktif; tandai Login/Manual. **Done:** preset bisa diaktifkan.

### ST-08.4 — Keyword + auto-generate
- [ ] TK-08.4.1 — Generator. **File:** `service/keyword_service.go`. **Do:** generate keyword dari `service_categories`; `negative_keywords` preset; endpoint CRUD. **Done:** pilih kapabilitas→keyword muncul (editable).

### ST-08.5 — FE Onboarding lean
- [ ] TK-08.5.1 — Onboarding. **File:** `src/pages/onboarding/Onboarding.tsx`. **Do:** Design §4.2 dua jalur (Upload PDF[placeholder→EP-13] / Isi manual) + Stepper + "Lewati atur nanti". **Done:** alur jalan.
- [ ] TK-08.5.2 — Aktifkan Agent. **File:** Onboarding. **Do:** tombol → simpan profil → trigger discovery pertama (EP-12; no-op bila belum) → redirect Penemuan AI; skip → banner Dashboard. **Done:** redirect benar.

### ST-08.6 — FE Otak Agent (6 kartu)
- [ ] TK-08.6.1 — Halaman + 6 kartu. **File:** `src/pages/profile/OtakAgent.tsx`. **Do:** Design §4.13 kartu 1–6 (chip preset, toggle no-go, slider bobot collapsed), Simpan sticky, badge "diperbarui {waktu}", tooltip per field. **Done:** simpan→toast + versi baru.
- [ ] TK-08.6.2 — Sub-tab Sumber. **File:** OtakAgent. **Do:** tabel sumber (Nama/URL/Negara/Akses/Legal note/Aktif) + badge Login/Manual. **Done:** kelola sumber dari UI.

---

## EP-09 — MCP Server & Sales Data Tools (P-9)

### ST-09.1 — Bootstrap
- [ ] TK-09.1.1 — Server. **File:** `internal/mcp/server.go`. **Do:** mark3labs/mcp-go HTTP di `/mcp`, auth Bearer `SalesMCPToken`. **Done:** `/mcp` menolak tanpa token.
- [ ] TK-09.1.2 — Register Hermes. **File:** `deploy/hermes/config.yaml.example`. **Do:** `mcp_servers.sales` + `tools.include` daftar tool + `supports_parallel_tool_calls:true`. **Done:** Hermes connect.

### ST-09.2 — Read tools
- [ ] TK-09.2.1 — Tender/event/prospect read. **File:** `internal/mcp/tools_read.go`. **Do:** `list_tenders,get_tender,search_tenders,list_events,get_event,list_prospects,get_prospect`. **Done:** tool kembalikan data.
- [ ] TK-09.2.2 — Summary + profile. **File:** tools_read.go. **Do:** `get_pipeline_summary,get_revenue_summary,get_company_profile`. **Done:** agent baca profil terbaru.

### ST-09.3 — Write tools (gated)
- [ ] TK-09.3.1 — Write. **File:** `internal/mcp/tools_write.go`. **Do:** `update_prospect_stage`,`save_playbook_draft`; hanya whitelist; tulis audit. **Done:** hanya tool whitelist aktif.

### ST-09.4 — Contract test
- [ ] TK-09.4.1 — Test. **File:** `internal/mcp/contract_test.go`. **Do:** chat "tender prioritas?" memicu `list_tenders`. **Done:** `go test -tags contract` hijau.

---

## EP-10 — AI Scoring & Recommendation (P-8)

### ST-10.1 — Service + schema
- [ ] TK-10.1.1 — Prompt builder. **File:** `internal/ai/scoring.go`. **Do:** rakit prompt (data + profil + rubrik §8 8-dimensi+bobot). **Done:** prompt berisi semua dimensi.
- [ ] TK-10.1.2 — Schema + call. **File:** scoring.go. **Do:** schema `{fit_score,recommended_action,confidence,reasoning,evidence[],risk_flags[]}` via `GenerateJSON`. **Done:** output valid schema.

### ST-10.2 — Threshold + no-go
- [ ] TK-10.2.1 — Recommend. **File:** `internal/ai/recommend.go`. **Do:** map skor→action (§8 ambang) + no-go rule → Need Partner/Auto No-Go. **Done:** unit test tiap ambang & no-go.

### ST-10.3 — Persist + endpoints
- [ ] TK-10.3.1 — Migrasi `0008_prospect_score.up.sql` (`prospect_score` field §10). **Done:** migrate ok.
- [ ] TK-10.3.2 — Endpoints. **File:** `handlers/score_handler.go`. **Do:** `POST /api/tenders/:id/score`, `POST /api/prospects/:id/score`; simpan + update skor tender; "Analisa ulang" idempotent; gagal AI→pesan ramah (data utuh); audit model/waktu. **Done:** row score + skor tampil.

### ST-10.4 — FE panel
- [ ] TK-10.4.1 — AiScorePanel. **File:** `src/components/AiScorePanel.tsx`. **Do:** score ring + recommended_action badge + evidence per dimensi (✓/⚠) + risk chips + "Dibuat AI • {confidence} • {waktu}" + "Lihat alasan" + "Analisa ulang" (streaming). **Done:** terpasang di TenderDetail & ProspectDrawer.

---

## EP-11 — Dashboard

### ST-11.1 — Endpoint
- [ ] TK-11.1.1 — Summary. **File:** `handlers/dashboard_handler.go`. **Do:** `GET /api/dashboard/summary` agregasi pipeline per stage + revenue(sum est_value) + prioritas(skor tinggi) + penemuan AI hari ini(count, degrade bila EP-12 belum). **Done:** angka cocok DB; indeks ada.

### ST-11.2 — FE
- [ ] TK-11.2.1 — Dashboard. **File:** `src/pages/Dashboard.tsx`. **Do:** stat cards + pipeline + revenue + prioritas (score ring) + AI insight callout + banner "Lengkapi Otak Agent" bila profil kosong; empty/loading. **Done:** render data nyata.

---

## EP-12 — Tender Discovery & Inbox

### ST-12.1 — Entity
- [ ] TK-12.1.1 — Migrasi `0009_discovery.up.sql` (`discovery_run` + `correlation_key` unique). **Done:** migrate ok.

### ST-12.2 — Orchestrator + compliance
- [ ] TK-12.2.1 — Orchestrator. **File:** `internal/ai/discovery.go`. **Do:** ambil profil (sumber enabled/keyword/target/no-go) → instruksikan Hermes crawl/browse hanya sumber legal → ekstrak field tender. **Done:** menghasilkan kandidat tender.
- [ ] TK-12.2.2 — Compliance guard. **File:** discovery.go. **Do:** §9 — tidak bypass CAPTCHA/login/paywall; sumber access=login/manual hanya ditandai (skip crawl), prioritas API/RSS/portal resmi; audit akses. **Done:** sumber Login tidak di-crawl, ditandai.
- [ ] TK-12.2.3 — Score + simpan. **File:** discovery.go. **Do:** tiap kandidat → scoring (EP-10) → simpan tender `origin=discovery`, belum direview. **Done:** tender masuk inbox dengan skor.

### ST-12.3 — Dedup + idempotency
- [ ] TK-12.3.1 — Dedup. **File:** discovery.go + repo. **Do:** `dedup_key=hash(buyer+title+deadline)`; gabung sumber ("ditemukan di N sumber"). **Done:** duplikat tidak dobel.
- [ ] TK-12.3.2 — Idempotent run. **File:** discovery service. **Do:** correlation/idempotency key cegah run ganda. **Done:** run ulang aman.

### ST-12.4 — Endpoints + async + rate limit
- [ ] TK-12.4.1 — Endpoints. **File:** `handlers/discovery_handler.go`. **Do:** `POST /api/discovery/run` (async→run id), `GET /api/discovery/runs`, `GET /api/discovery/inbox`; RBAC `RunDiscovery`. **Done:** run async, status live.
- [ ] TK-12.4.2 — Rate limit/backoff. **File:** discovery.go. **Do:** per-sumber rate limit + backoff. **Done:** tidak melebihi batas.

### ST-12.5 — Scheduling
- [ ] TK-12.5.1 — Scheduler. **File:** `internal/ai/scheduler.go`. **Do:** `crawl_frequency`→jadwal (cron Hermes atau ticker internal); bisa dimatikan. **Done:** run terjadwal tercatat.

### ST-12.6 — FE Inbox
- [ ] TK-12.6.1 — Inbox page. **File:** `src/pages/discovery/DiscoveryInbox.tsx`. **Do:** Design §4.3 header(Jalankan+status), filter, kartu(score ring+badge+risk chips+alasan 1 baris). **Done:** render hasil.
- [ ] TK-12.6.2 — State crawl + empty. **File:** Inbox. **Do:** progress "AI sedang mencari di N sumber…", empty (profil kosong→CTA / tidak ada hasil). **Done:** state benar.

### ST-12.7 — Promote/Tolak
- [ ] TK-12.7.1 — Aksi. **File:** Inbox. **Do:** Pursue→`POST /promote` (ST-05.4); Watchlist→tandai; Tolak→modal alasan→simpan (learning EP-16); optimistic. **Done:** aksi memindahkan/menyimpan benar.

---

## EP-13 — PDF Ingest

### ST-13.1 — Upload
- [ ] TK-13.1.1 — Endpoint. **File:** profile_handler.go. **Do:** `POST /api/profile/ingest` multipart PDF → simpan file (volume) + ref `source_doc_refs`; batas ukuran/tipe. **Done:** upload tersimpan.

### ST-13.2 — Extraction
- [ ] TK-13.2.1 — PDF→teks. **File:** `internal/ai/profile_extract.go`. **Do:** ekstrak teks PDF (lib Go). **Done:** teks keluar.
- [ ] TK-13.2.2 — Hermes extract. **File:** profile_extract.go. **Do:** `GenerateJSON` field profil; kembalikan draft (tak auto-simpan); gagal→error "coba isi manual". **Done:** draft field keluar.

### ST-13.3 — FE review
- [ ] TK-13.3.1 — Dropzone+progress+review. **File:** Onboarding & OtakAgent. **Do:** dropzone, progress "AI membaca dokumen…" (live-region), field hasil chip "diisi AI ✨", edit+konfirmasi→simpan versi. **Done:** alur PDF→konfirmasi→tersimpan.

---

## EP-14 — Playbook Generator (P-8)

### ST-14.1 — Entity
- [ ] TK-14.1.1 — Migrasi `0010_playbook.up.sql` (`playbook` jsonb content + version). **Done:** migrate ok.

### ST-14.2 — Service
- [ ] TK-14.2.1 — Generator. **File:** `internal/ai/playbook.go`. **Do:** prompt (konteks + playbook menang dari memory) → schema sections (Ringkasan/Value Prop/Stakeholders/Strategi checklist/Timeline/Risiko/Next Actions). **Done:** output sections lengkap.

### ST-14.3 — Endpoints
- [ ] TK-14.3.1 — Endpoints. **File:** `handlers/playbook_handler.go`. **Do:** `POST /api/{tenders|prospects}/:id/playbook` (versi+1 immutable), `GET .../playbooks`, `GET /api/playbooks/:id`, compare. **Done:** versi bertambah, lama tetap.

### ST-14.4 — FE
- [ ] TK-14.4.1 — Viewer. **File:** `src/pages/playbooks/*`. **Do:** Design §4.10 list+viewer terstruktur+generate streaming+footer(versi/model/waktu, generate versi baru, bandingkan)+salin/export markdown. **Done:** generate→render→export.

---

## EP-15 — Report Generator (P-8)

### ST-15.1 — Entity
- [ ] TK-15.1.1 — Migrasi `0011_report.up.sql` (`report`; validasi period). **Done:** migrate ok.

### ST-15.2 — Service
- [ ] TK-15.2.1 — Generator 3 tipe. **File:** `internal/ai/report.go`. **Do:** agregasi pipeline/aktivitas → Hermes markdown untuk Daily Digest/Weekly Pipeline/Per-peluang. **Done:** 3 tipe menghasilkan markdown < 2 menit.

### ST-15.3 — Endpoints
- [ ] TK-15.3.1 — Endpoints. **File:** `handlers/report_handler.go`. **Do:** `POST /api/reports`(type+period, validasi period_start≤end), `GET /api/reports`, `GET/:id`, `DELETE/:id`. **Done:** CRUD ok.

### ST-15.4 — FE
- [ ] TK-15.4.1 — Reports. **File:** `src/pages/reports/*`. **Do:** Design §4.11 list+generate modal(tipe+periode) streaming+viewer(ringkasan+tabel pipeline+prospek prioritas+insight)+export/salin/hapus+empty. **Done:** generate→viewer→export.

---

## EP-16 — Continuous Learning

### ST-16.1 — Entity
- [ ] TK-16.1.1 — Migrasi `0012_outcome.up.sql` (`outcome_event`). **Done:** migrate ok.

### ST-16.2 — Outcome hook
- [ ] TK-16.2.1 — Learning hook. **File:** `internal/ai/learning.go`. **Do:** dari outcome (EP-05/07) & Tolak discovery (EP-12) kirim catatan ringkas ke memory Hermes (session-key workspace) via `internal/hermes`; non-blocking. **Done:** memory write terkirim (cek log).

### ST-16.3 — Reset memory
- [ ] TK-16.3.1 — Endpoint. **File:** `handlers/admin_handler.go`. **Do:** `POST /api/admin/hermes/reset-memory` (ADMIN) → hermes reset. **Done:** non-admin 403; admin sukses.

### ST-16.4 — FE cue + verifikasi
- [ ] TK-16.4.1 — Cue. **File:** Chat + modal WON/LOST/Tolak. **Do:** microcopy "Asisten belajar dari aktivitas & hasil kamu" / "AI akan belajar dari ini". **Done:** teks tampil.
- [ ] TK-16.4.2 — Verifikasi persist. **Do:** WON → restart hermes-gateway → chat tanya serupa → jawaban mempertimbangkan konteks. **Done:** memory bertahan.

---

## EP-17 — Telemetry, Observability, Audit & NFR

### ST-17.1 — Telemetry
- [ ] TK-17.1.1 — Infra. **File:** `internal/telemetry/telemetry.go`. **Do:** `Emit(event, props)` simpan ke tabel/log. **Done:** event tersimpan.
- [ ] TK-17.1.2 — Emit titik metrik. **File:** handler terkait. **Do:** `chat_opened`, review pursue, durasi report/scoring, outcome WON/total. **Done:** event muncul saat aksi.

### ST-17.2 — Logging + audit
- [ ] TK-17.2.1 — Structured log. **File:** `internal/log`. **Do:** logger JSON + request id. **Done:** log terstruktur.
- [ ] TK-17.2.2 — Audit trail. **File:** ai/discovery + scoring/playbook/report. **Do:** simpan sumber/waktu/akses/data + reasoning/evidence/model tiap output AI. **Done:** audit row per output AI & crawl.

### ST-17.3 — Performance + health panel
- [ ] TK-17.3.1 — Indeks & p95. **File:** migrasi indeks + review query. **Do:** indeks status/deadline/stage/dedup_key; ukur CRUD p95<300ms. **Done:** benchmark lulus.
- [ ] TK-17.3.2 — Health endpoint. **File:** `GET /api/health/hermes`. **Do:** status + versi. **Done:** dipakai Settings.

### ST-17.4 — Contract + NFR verifikasi
- [ ] TK-17.4.1 — Suite + checklist. **File:** test + `docs/nfr-checklist.md`. **Do:** contract `/v1`+MCP hijau; checklist §11 + verifikasi epic §4. **Done:** semua centang.

---

## EP-18 — Settings, Admin & Hermes Ops

### ST-18.1 — Status/test endpoint
- [ ] TK-18.1.1 — Endpoint. **File:** `handlers/settings_handler.go`. **Do:** `GET /api/settings/hermes` (status+versi+memory), `POST /api/settings/hermes/test`. **Done:** kembalikan status.

### ST-18.2 — FE Settings
- [ ] TK-18.2.1 — Tabs. **File:** `src/pages/settings/*`. **Do:** Design §4.14 tab Profil user/Workspace/Users(admin CRUD)/AI-Hermes (status "Connected • vX", memory aktif, reset memory[admin], Test koneksi); gagal→badge merah+petunjuk. **Done:** semua tab fungsional.

### ST-18.3 — Reset memory wiring
- [ ] TK-18.3.1 — Wiring. **File:** Settings. **Do:** tombol reset(ADMIN)→konfirmasi→ST-16.3→toast. **Done:** reset jalan.

### ST-18.4 — AI Provider Config dari UI (OpenAI / OpenRouter)
> Atur provider/model/API-key AI dari UI (fleksibel seperti TUI Hermes). DB = source of truth → push ke `hermes-bridge` via `internal/hermes.Configure` (ST-01.6). API key disimpan terenkripsi, tak pernah dikirim balik plaintext.
- [ ] TK-18.4.1 — Migrasi `00xx_ai_setting.up.sql` (P-1). **Do:** tabel `ai_provider_setting` (`provider` TEXT CHECK in (`openai`,`openrouter`), `model` TEXT, `base_url` TEXT NULL, `api_key_encrypted` TEXT, `enabled_toolsets` jsonb, `is_active` bool). **Done:** migrate up/down ok.
- [ ] TK-18.4.2 — Domain+repo+enkripsi. **File:** `domain/ai_setting.go`, `repository/ai_setting_repo.go`, `internal/auth/crypto.go`. **Do:** struct + repo get-active/upsert; AES-GCM encrypt/decrypt key (`CONFIG_ENC_KEY` dari config; tambah ke `internal/config`). **Done:** unit test round-trip enkripsi; key tersimpan terenkripsi.
- [ ] TK-18.4.3 — Service + endpoints (P-5,P-6). **File:** `handlers/ai_settings_handler.go`, `service/ai_setting_service.go`, router. **Do:** `GET /api/settings/ai` (key **masked** `sk-...abcd`), `PUT /api/settings/ai` (RBAC `ManageUsers`/ADMIN → simpan DB + `hermes.Configure`), `POST /api/settings/ai/test` (uji koneksi via bridge `Health`/chat ringan). **Done:** non-admin→403; PUT→bridge dapat config; test→status.
- [ ] TK-18.4.4 — Rehydrate saat boot. **File:** `apps/api/main.go`. **Do:** setelah health bridge ok, baca config aktif dari DB → `hermes.Configure` (agar restart bridge tidak kehilangan provider). **Done:** restart `hermes-bridge` → Go re-push otomatis.
- [ ] TK-18.4.5 — FE tab AI Provider (P-7). **File:** `src/pages/settings/AiProvider.tsx`, `src/api/settings.ts`. **Do:** Design §4.14: pilih provider (OpenAI/OpenRouter), model (preset per provider + input custom), base_url opsional, API key (write-only, placeholder masked), enabled toolsets, Simpan + **Test koneksi** (badge hasil). **Done:** set dari UI → test hijau → chat berikutnya pakai provider baru tanpa restart.

---

## Verifikasi akhir (end-to-end, dari epic §4)
1. `docker compose up` → semua service hidup.
2. Hermes `/v1/models` 200 + `/health` ok.
3. Login → buat tender → tersimpan.
4. Chat "Tender prioritas minggu ini?" → tool-call `list_tenders` → ranking.
5. Score tender/prospect → `prospect_score` + skor tampil.
6. Isi Otak Agent → Aktifkan Agent → discovery run → tender baru di Inbox → Pursue → pipeline.
7. Generate playbook & 3 tipe report → tersimpan & export markdown.
8. Tandai WON → restart Hermes → chat tetap ingat konteks (continuous learning).
9. `go test -tags contract ./...` hijau.

## Status (3-lapis selesai)
- [x] **Layer 1 (EPIC)** — epic.plan.md (19 epic).
- [x] **Layer 2 (STORY)** — story.plan.md (≈90 story).
- [x] **Layer 3 (TASK)** — file ini (task mikro siap eksekusi + §A pola reusable).

## Progress Eksekusi

### EP-03 — Auth, RBAC & User Management
- [x] **ST-03.1** — Entity user + seed · TK-03.1.1 ✓ · TK-03.1.2 ✓ · TK-03.1.3 ✓
- [x] **ST-03.2** — Login + JWT · TK-03.2.1 ✓ · TK-03.2.2 ✓ · TK-03.2.3 ✓
- [x] **ST-03.3** — Middleware RBAC · TK-03.3.1 ✓ · TK-03.3.2 ✓
- [x] **ST-03.4** — Admin user mgmt · TK-03.4.1 ✓ · TK-03.4.2 ✓
- [x] **ST-03.5** — FE Login + auth store + guard · TK-03.5.1 ✓ · TK-03.5.2 ✓ · TK-03.5.3 ✓ · TK-03.5.4 ✓

### EP-04 — Chat Assistant (backend)
- [x] **ST-04.1** — Entity conversation/message · TK-04.1.1 ✓ · TK-04.1.2 ✓
- [x] **ST-04.2** — Create conversation + session key · TK-04.2.1 ✓ · TK-04.2.2 ✓
- [x] **ST-04.3** — Chat SSE relay · TK-04.3.1 ✓ · TK-04.3.2 ✓
- [x] **ST-04.4** — History · TK-04.4.1 ✓

### EP-04 — Chat Assistant (frontend)
- [x] **ST-04.5** — FE Chat UI · TK-04.5.1 ✓ · TK-04.5.2 ✓ · TK-04.5.3 ✓ · TK-04.5.4 ✓
- [x] **ST-04.6** — Floating Tanya AI + degrade · TK-04.6.1 ✓ · TK-04.6.2 ✓

### EP-05 — Tender Management
- [x] **ST-05.1** — Entity tender · TK-05.1.1 ✓ · TK-05.1.2 ✓ · TK-05.1.3 ✓
- [x] **ST-05.2** — CRUD endpoints · TK-05.2.1 ✓ · TK-05.2.2 ✓ · TK-05.2.3 ✓
- [x] **ST-05.3** — Status + outcome · TK-05.3.1 ✓ · TK-05.3.2 ✓ (**Catatan:** `outcome_event` dibuat di sini; TK-16.1.1 superseded; migrasi EP-06+ bergeser +1)
- [x] **ST-05.4** — Promote · TK-05.4.1 ✓
- [x] **ST-05.5** — FE List · TK-05.5.1 ✓ · TK-05.5.2 ✓
- [x] **ST-05.6** — FE Detail · TK-05.6.1 ✓ · TK-05.6.2 ✓
- [x] **ST-05.7** — FE Form drawer · TK-05.7.1 ✓

### EP-06 — Event Management
- [x] **ST-06.1–06.2** — Entity + CRUD · TK-06.1.1 ✓ · TK-06.1.2 ✓ · TK-06.2.1 ✓ (**Koreksi migrasi:** 0006_events karena 0005 sudah dipakai outcome_events; EP-07 prospect = 0007)
- [x] **ST-06.3** — Konversi → prospect · TK-06.3.1 ✓ (entity prospect minimal + migrasi 0007 dibuat di sini; EP-07 expand)
- [x] **ST-06.4** — FE Events · TK-06.4.1 ✓
