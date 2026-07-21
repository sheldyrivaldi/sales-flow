package handlers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"

	"salespilot/internal/auth"
	"salespilot/internal/config"
	"salespilot/internal/domain"
	"salespilot/internal/hermestui"
	"salespilot/internal/http/httperr"
)

// Idle/hard-cap sweep constants (see plan §Session management). The hard
// cap deliberately matches auth.TUISessionTTL — once the session cookie
// itself expires, the registry should have already reaped the session, not
// leave it open past the point the cookie could still authorize it.
const (
	hermesTuiIdleTimeout = 30 * time.Minute
	hermesTuiHardCap     = auth.TUISessionTTL
	hermesTuiSweepTick   = 60 * time.Second
)

const sessionIDContextKey = "hermesTuiSessionID"

// HermesTuiHandler backs the admin-only Hermes TUI feature: a ticket ->
// session-cookie handoff, then a reverse proxy (HTTP + several WS
// sub-paths) into the hermes-tui sidecar (the official hermes-agent image
// running `hermes dashboard`), which itself serves Hermes's real,
// unmodified config/API-key/OAuth/session web UI. See the feature plan for
// the full connection lifecycle, authentication matrix, and security
// boundary this handler implements.
type HermesTuiHandler struct {
	tickets  *hermestui.TicketStore
	registry *hermestui.Registry
	repo     domain.HermesTuiSessionRepository
	cfg      *config.Config
	upgrader websocket.Upgrader
	proxy    *httputil.ReverseProxy
}

func NewHermesTuiHandler(repo domain.HermesTuiSessionRepository, cfg *config.Config) (*HermesTuiHandler, error) {
	target, err := url.Parse(cfg.HermesTuiBaseURL)
	if err != nil {
		return nil, fmt.Errorf("hermes tui handler: parse HermesTuiBaseURL: %w", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	baseDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		baseDirector(req)
		// The dashboard has no ttyd-style base-path flag — it always serves
		// its own routes rooted at "/" and expects X-Forwarded-Prefix to
		// know what prefix the browser sees it mounted under (see
		// TUIBasePath doc comment). Both halves of this contract must be
		// applied together: strip the prefix so the upstream route matches,
		// and tell it what that prefix was so it can re-embed it into the
		// HTML/CSS it serves back.
		req.URL.Path = strings.TrimPrefix(req.URL.Path, hermestui.TUIBasePath)
		if req.URL.Path == "" {
			req.URL.Path = "/"
		}
		req.Header.Set("X-Forwarded-Prefix", hermestui.TUIBasePath)

		// Autentikasi ke dashboard. Dashboard punya dua mode:
		//   - token/--insecure: menerima X-Hermes-Session-Token (atau Bearer
		//     legacy) yang nilainya = HERMES_DASHBOARD_SESSION_TOKEN.
		// Menyuntikkan token di sini membuat SEMUA rute (bukan cuma yang
		// dipegang SPA) lolos, termasuk saat dashboard berada di host lain.
		// Tanpa ini, dashboard remote dalam mode token akan menolak sebagian
		// rute. Di mode gated/OAuth token diabaikan (butuh cookie login), jadi
		// menyetelnya tidak berbahaya.
		if cfg.HermesTuiSessionToken != "" {
			req.Header.Set("X-Hermes-Session-Token", cfg.HermesTuiSessionToken)
			req.Header.Set("Authorization", "Bearer "+cfg.HermesTuiSessionToken)
		}
	}

	// BUG "iframe menampilkan SalesFlow": dashboard yang gated membalas
	// 302 ke "/auth/login" (root-relative). Di dalam iframe ber-origin
	// SalesFlow, "/auth/login" diresolusi ke SalesFlow — SPA fallback
	// menyajikan aplikasi ini sendiri, bukan halaman login Hermes. Perbaikan:
	// beri prefix TUIBasePath pada Location root-relative apa pun, sehingga
	// setiap redirect tetap berada di dalam jalur proxy (halaman login Hermes
	// yang tampil, bukan SalesFlow). Dashboard yang menghormati
	// X-Forwarded-Prefix sudah memberi prefix; ini jaring pengaman untuk
	// redirect auth yang kerap melewatinya.
	proxy.ModifyResponse = func(resp *http.Response) error {
		if loc := resp.Header.Get("Location"); loc != "" {
			loc = redirectAwayFromBrokenOAuthLogin(loc)
			resp.Header.Set("Location", prefixTuiLocation(loc, hermestui.TUIBasePath))
		}

		// BUG "login basic-auth balik ke dashboard SalesFlow": halaman statis
		// GET /login yang dipakai provider=basic (beda dari SPA lain di
		// dashboard — sudah dikonfirmasi via curl langsung dengan
		// X-Forwarded-Prefix bahwa redirect "/" -> "/auth/login" SUDAH
		// diprefix dengan benar oleh upstream) TIDAK menghormati
		// X-Forwarded-Prefix sama sekali: hidden input next="/",
		// fetch('/auth/password-login'), dan window.location.assign(next
		// || '/') semuanya root-relative absolut. Di dalam proxy kita
		// (origin selalu salesflow.moonlay.com), itu bikin submit gagal
		// nyampe Go (jatuh ke SPA fallback nginx) lalu window.location
		// balik ke root -> render dashboard SalesFlow sendiri, bukan
		// Hermes. Vendor bug (nousresearch/hermes-agent v2026.7.7.2),
		// bukan sesuatu yang bisa kita ubah di sumbernya — rewrite body di
		// sini sebagai jaring pengaman, sama prinsipnya dengan
		// prefixTuiLocation di atas untuk Location header. Re-cek kalau
		// image di-bump: fix upstream mungkin bikin blok ini jadi no-op
		// (aman) atau perlu disesuaikan lagi kalau markup-nya berubah.
		if strings.HasPrefix(resp.Header.Get("Content-Type"), "text/html") {
			body, err := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if err != nil {
				return fmt.Errorf("hermes tui: baca body login utk rewrite prefix: %w", err)
			}
			rewritten := rewriteHermesLoginHTML(body, hermestui.TUIBasePath)
			resp.Body = io.NopCloser(bytes.NewReader(rewritten))
			resp.ContentLength = int64(len(rewritten))
			resp.Header.Set("Content-Length", strconv.Itoa(len(rewritten)))
		}
		return nil
	}

	return &HermesTuiHandler{
		tickets:  hermestui.NewTicketStore(),
		registry: hermestui.NewRegistry(),
		repo:     repo,
		cfg:      cfg,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			// Subprotocols intentionally left unset here — the dashboard
			// exposes several WS endpoints (/api/pty, /api/ws, /api/pub,
			// /api/events) that may each negotiate differently. ProxyWS
			// sets this per-request from whatever the upstream dial
			// actually negotiates, rather than assuming one fixed protocol
			// (as the old ttyd-specific "tty" subprotocol did).
			//
			// Cross-origin WS clients are not a realistic threat here: the
			// actual boundary is the SameSite=Strict session cookie (never
			// attached to a cross-site request in the first place) checked
			// by cookieGate before Upgrade is ever called.
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		proxy: proxy,
	}, nil
}

// IssueTicket handles POST /api/admin/hermes/tui/ticket (JWT + CapManageUsers,
// wired under the existing admin group in router.go).
func (h *HermesTuiHandler) IssueTicket(c echo.Context) error {
	user, ok := auth.UserFromContext(c)
	if !ok {
		return httperr.Write(c, httperr.NewUnauthorized("user tidak terautentikasi"))
	}

	ticket := h.tickets.Issue(user.ID)
	return c.JSON(http.StatusOK, map[string]any{
		"ticket":     ticket,
		"expires_in": int(hermestui.TicketTTL.Seconds()),
		"tui_url":    hermestui.TUIBasePath + "/enter?ticket=" + url.QueryEscape(ticket),
	})
}

// Enter handles GET /api/admin/hermes/tui/enter?ticket=… — a plain browser
// navigation (the iframe's src), deliberately NOT behind JWTMiddleware: a
// navigation can't carry a custom Authorization header. The single-use
// ticket minted by IssueTicket is the proof of authorization for this one
// exchange; everything past this point is authorized by the session cookie
// set here.
func (h *HermesTuiHandler) Enter(c echo.Context) error {
	ticket := c.QueryParam("ticket")
	userID, ok := h.tickets.Consume(ticket)
	if !ok {
		return httperr.Write(c, httperr.NewUnauthorized("ticket tidak valid atau sudah kedaluwarsa"))
	}

	sessionID := uuid.NewString()
	h.registry.Open(sessionID, userID)

	dbCtx, dbCancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	dbErr := h.repo.Create(dbCtx, &domain.HermesTuiSession{
		ID:        sessionID,
		UserID:    userID,
		StartedAt: time.Now(),
		RemoteIP:  c.RealIP(),
	})
	dbCancel()
	if dbErr != nil {
		// Audit write failing must not block the session itself — best
		// effort, consistent with this codebase's non-blocking principle
		// (PRD §8). The live registry entry is unaffected either way.
		c.Logger().Errorf("hermes tui: gagal mencatat audit session %s: %v", sessionID, dbErr)
	}

	token, err := auth.IssueTUISession(userID, sessionID, h.cfg.JWTSecret)
	if err != nil {
		h.registry.Close(sessionID)
		return httperr.Write(c, httperr.NewInternal())
	}

	c.SetCookie(&http.Cookie{
		Name:     hermestui.SessionCookieName,
		Value:    token,
		Path:     hermestui.TUIBasePath,
		HttpOnly: true,
		// Secure is derived from the live request, not hardcoded — this
		// deployment currently terminates plain HTTP (see apps/web/nginx.conf),
		// and a hardcoded Secure:true cookie would never be resent by the
		// browser over HTTP, silently breaking every request past this one.
		// c.Scheme() already accounts for X-Forwarded-Proto if a future
		// TLS-terminating proxy sits in front.
		Secure:   c.Scheme() == "https",
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(auth.TUISessionTTL.Seconds()),
	})

	return c.Redirect(http.StatusFound, hermestui.TUIBasePath+"/")
}

// CookieGate validates the TUI session cookie on every proxied request
// (HTML, assets, the WS upgrade) — deliberately separate from
// auth.JWTMiddleware, since a plain navigation and a same-origin WS
// handshake can only carry a cookie, not a custom Authorization header (see
// router.go wiring comment for why these routes sit outside authd).
func (h *HermesTuiHandler) CookieGate(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		cookie, err := c.Cookie(hermestui.SessionCookieName)
		if err != nil || cookie.Value == "" {
			return httperr.Write(c, httperr.NewUnauthorized("sesi Hermes TUI tidak ditemukan"))
		}

		claims, err := auth.Parse(cookie.Value, h.cfg.JWTSecret, auth.TokenTUISession)
		if err != nil {
			return httperr.Write(c, httperr.NewUnauthorized("sesi Hermes TUI tidak valid atau kedaluwarsa"))
		}

		if !h.registry.IsOpen(claims.SessionID) {
			return httperr.Write(c, httperr.NewUnauthorized("sesi Hermes TUI sudah berakhir"))
		}

		c.Set(sessionIDContextKey, claims.SessionID)
		return next(c)
	}
}

// ProxyHTTP reverse-proxies everything under TUIBasePath (except /enter and
// /end) to the hermes-tui sidecar — the Director (see NewHermesTuiHandler)
// strips TUIBasePath from the request path and sets X-Forwarded-Prefix, so
// the dashboard's own routes (rooted at "/") and its base-path-aware
// HTML/CSS rewriting both resolve correctly.
func (h *HermesTuiHandler) ProxyHTTP(c echo.Context) error {
	h.proxy.ServeHTTP(c.Response(), c.Request())
	return nil
}

// ProxyWS handles any of the dashboard's WS sub-paths (/api/pty, /api/ws,
// /api/pub, /api/events — see router.go registration) — cookie-gated,
// dials the hermes-tui sidecar, and pumps frames bidirectionally without
// ever parsing the upstream's wire protocol.
func (h *HermesTuiHandler) ProxyWS(c echo.Context) error {
	sessionID, _ := c.Get(sessionIDContextKey).(string)

	ctx, ok := h.registry.Context(sessionID)
	if !ok {
		return httperr.Write(c, httperr.NewUnauthorized("sesi Hermes TUI sudah berakhir"))
	}

	upstreamPath := strings.TrimPrefix(c.Request().URL.Path, hermestui.TUIBasePath)
	target, err := hermestui.WSURL(h.cfg.HermesTuiBaseURL, upstreamPath)
	if err != nil {
		return httperr.Write(c, httperr.NewInternal())
	}

	// Dial upstream BEFORE upgrading the client, and mirror whatever
	// subprotocol the client asked for — the dashboard's several WS
	// endpoints may each negotiate differently (unlike ttyd's single fixed
	// "tty" subprotocol), so this proxy stays generic rather than assuming
	// one. Dialing first also means an unreachable/overloaded hermes-tui
	// can be reported as a normal HTTP error instead of a post-hoc WS close
	// frame, since the client hasn't been upgraded yet.
	bridgeDialer := websocket.Dialer{Subprotocols: websocket.Subprotocols(c.Request())}
	bridgeConn, resp, err := bridgeDialer.Dial(target, nil)
	if err != nil {
		return httperr.Write(c, httperr.NewBadRequest("HERMES_TUI_UNAVAILABLE", "hermes-tui tidak tersedia atau sedang penuh, coba lagi"))
	}
	defer func() { _ = bridgeConn.Close() }()

	upgrader := h.upgrader
	if proto := resp.Header.Get("Sec-WebSocket-Protocol"); proto != "" {
		upgrader.Subprotocols = []string{proto}
	}
	clientConn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return nil // Upgrade already wrote its own error response.
	}
	defer func() { _ = clientConn.Close() }()

	h.registry.IncConn(sessionID)
	defer h.registry.DecConn(sessionID)

	hermestui.Pump(ctx, clientConn, bridgeConn, func() { h.registry.Touch(sessionID) })
	return nil
}

// EndSession handles POST /api/admin/hermes/tui/end — cookie-gated explicit
// close, one of the (only) three ways a session's ended_at gets written (see
// plan §Full connection lifecycle, step 10: a WS pump merely returning does
// NOT end the session, to stay reconnect-safe).
func (h *HermesTuiHandler) EndSession(c echo.Context) error {
	sessionID, _ := c.Get(sessionIDContextKey).(string)

	if h.registry.Close(sessionID) {
		ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
		if err := h.repo.End(ctx, sessionID, time.Now()); err != nil {
			c.Logger().Errorf("hermes tui: gagal menutup audit session %s: %v", sessionID, err)
		}
		cancel()
	}

	c.SetCookie(&http.Cookie{
		Name:     hermestui.SessionCookieName,
		Value:    "",
		Path:     hermestui.TUIBasePath,
		HttpOnly: true,
		Secure:   c.Scheme() == "https",
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})

	return c.JSON(http.StatusOK, map[string]string{"status": "ended"})
}

// RunSweeper reaps idle/hard-cap-expired sessions on a fixed tick until ctx
// is cancelled (see apps/api/main.go — same shutdown pattern as the
// discovery Scheduler). Each reaped session's ended_at is persisted via
// repo.End, matching what an explicit EndSession call would have done.
func (h *HermesTuiHandler) RunSweeper(ctx context.Context) {
	ticker := time.NewTicker(hermesTuiSweepTick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ended := h.registry.SweepExpired(hermesTuiIdleTimeout, hermesTuiHardCap)
			for _, s := range ended {
				endCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				if err := h.repo.End(endCtx, s.ID, time.Now()); err != nil {
					fmt.Printf("hermes tui: gagal menutup audit session %s saat sweep: %v\n", s.ID, err)
				}
				cancel()
			}
		}
	}
}

// prefixTuiLocation memberi prefix TUIBasePath pada Location redirect yang
// root-relative sehingga tetap berada di dalam jalur proxy dashboard. Absolut
// (punya skema/host), fragmen kosong, dan yang sudah berprefix dibiarkan.
// Inti perbaikan "iframe menampilkan SalesFlow": tanpa ini, redirect dashboard
// ke "/auth/login" diresolusi browser terhadap origin SalesFlow.
func prefixTuiLocation(loc, base string) string {
	if loc == "" || !strings.HasPrefix(loc, "/") {
		return loc // kosong, atau absolut (http://…) — jangan diutak-atik
	}
	if strings.HasPrefix(loc, "//") {
		return loc // protocol-relative (//host/…) juga absolut
	}
	if loc == base || strings.HasPrefix(loc, base+"/") {
		return loc // sudah berprefix
	}
	return base + loc
}

// redirectAwayFromBrokenOAuthLogin menambal BUG VENDOR nyata di hermes-agent
// (bukan soal prefix): "/" selalu 302 ke "/auth/login?provider=basic", tapi
// route itu cuma dibuat buat provider OAuth (start_login/redirect_uri) —
// untuk provider basic ini SELALU crash 500 (lihat traceback di
// hermes_cli/dashboard_auth/routes.py:197 — "BasicAuthProvider is
// password-only; there is no OAuth redirect flow. The login page POSTs to
// /auth/password-login instead."). Ini kejadian ke SIAPA PUN yang buka
// hermes.moonlay.com/ dari root, bukan cuma lewat proxy kita — sudah
// dikonfirmasi via curl langsung + docker logs di server Hermes. Selama
// belum ada fix/upgrade upstream, alihkan redirect yang salah itu ke
// "/login" (halaman form password statis yang sudah benar — lihat
// rewriteHermesLoginHTML), pertahankan query "next".
func redirectAwayFromBrokenOAuthLogin(loc string) string {
	u, err := url.Parse(loc)
	if err != nil || u.Path != "/auth/login" || u.Query().Get("provider") != "basic" {
		return loc
	}
	q := url.Values{}
	if next := u.Query().Get("next"); next != "" {
		q.Set("next", next)
	}
	u.Path = "/login"
	u.RawQuery = q.Encode()
	return u.String()
}

// rewriteHermesLoginHTML menambal 3 literal root-relative absolut yang
// diketahui hardcoded di halaman statis GET /login (provider=basic) milik
// hermes-agent — lihat komentar di ModifyResponse untuk detail bug &
// justifikasi. Idempotent: kalau literalnya sudah tidak match (markup
// berubah / fix upstream), fungsi ini no-op dan HTML lewat apa adanya.
func rewriteHermesLoginHTML(body []byte, base string) []byte {
	replacements := [][2]string{
		{`name="next" value="/">`, `name="next" value="` + base + `/">`},
		{`fetch('/auth/password-login'`, `fetch('` + base + `/auth/password-login'`},
		{`(data && data.next) || '/')`, `(data && data.next) || '` + base + `/')`},
	}
	out := body
	for _, r := range replacements {
		out = bytes.ReplaceAll(out, []byte(r[0]), []byte(r[1]))
	}
	return out
}
