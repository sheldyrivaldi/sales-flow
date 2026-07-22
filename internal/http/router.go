package http

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gorm.io/gorm"

	"salespilot/internal/ai"
	"salespilot/internal/auth"
	"salespilot/internal/config"
	"salespilot/internal/hermes"
	"salespilot/internal/hermestui"
	"salespilot/internal/http/handlers"
	applog "salespilot/internal/log"
	"salespilot/internal/mailer"
	"salespilot/internal/mcp"
	"salespilot/internal/repository"
	"salespilot/internal/service"
	"salespilot/internal/telemetry"
)

// schedulerTickInterval is how often the discovery Scheduler wakes up to
// check whether a crawl is due (EP-12 ST-12.5.1) — not the crawl frequency
// itself (that's per-workspace, company_profile.crawl_frequency). Small
// enough that toggling crawl_enabled or changing crawl_frequency from the
// UI takes effect within a few minutes, not a full day.
const schedulerTickInterval = 5 * time.Minute

// New creates and configures the Echo instance (middleware + routes). It
// also returns ProfileService and DiscoveryService so apps/api/main.go can
// wire the discovery Scheduler (EP-12 ST-12.5.1) to the exact same instances
// the HTTP handlers use — a second, separately-constructed pair would risk
// drifting out of sync (e.g. a different DiscoveryRunRepository instance).
// AISettingService is returned the same way so main.go can rehydrate the
// bridge's config at boot (EP-18 TK-18.4.4) without re-decoding
// CONFIG_ENC_KEY a second time.
func New(cfg *config.Config, db *gorm.DB, hc hermes.Client) (*echo.Echo, *service.ProfileService, *service.DiscoveryService, *service.AISettingService, *handlers.HermesTuiHandler, *ai.Scheduler) {
	e := echo.New()
	e.HideBanner = true

	e.Validator = NewValidator()

	// Structured logging (EP-17 ST-17.2) — replaces Echo's plain-text
	// Logger() with a JSON line per request (request id, method, uri,
	// status, latency, actor if authenticated, error if any).
	slog.SetDefault(applog.New())
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogMethod:    true,
		LogURI:       true,
		LogStatus:    true,
		LogLatency:   true,
		LogRequestID: true,
		LogError:     true,
		HandleError:  true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			attrs := []slog.Attr{
				slog.String("method", v.Method),
				slog.String("uri", v.URI),
				slog.Int("status", v.Status),
				slog.Duration("latency", v.Latency),
				slog.String("request_id", v.RequestID),
			}
			if u, ok := auth.UserFromContext(c); ok {
				attrs = append(attrs, slog.String("user_id", u.ID))
			}
			level := slog.LevelInfo
			if v.Error != nil {
				level = slog.LevelError
				attrs = append(attrs, slog.String("error", v.Error.Error()))
			}
			slog.LogAttrs(c.Request().Context(), level, "request", attrs...)
			return nil
		},
	}))
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(middleware.CORS())

	e.GET("/healthz", healthzHandler(db))

	// Serve uploaded files (chat attachments, etc.) statically so images can
	// render in <img> and PDFs open in a new tab. Filenames are UUIDs, so
	// they're unguessable — acceptable for an internal workspace tool.
	e.Static("/uploads", cfg.UploadDir)

	// Wire dependencies.
	userRepo := repository.NewUserRepo(db)
	authSvc := service.NewAuthService(userRepo, cfg.JWTSecret)
	authH := handlers.NewAuthHandler(authSvc)

	userSvc := service.NewUserService(userRepo)
	userH := handlers.NewUserHandler(userSvc)

	adminH := handlers.NewAdminHandler(hc, cfg)
	healthH := handlers.NewHealthHandler(hc)
	settingsH := handlers.NewSettingsHandler(hc, cfg)

	// AI Provider Config (EP-18 ST-18.4) — feature is simply unavailable
	// (not a boot crash) when CONFIG_ENC_KEY is unset or malformed, per
	// PRD §8's non-blocking principle extended to configuration.
	var aiConfigEncKey []byte
	if cfg.ConfigEncKey != "" {
		if k, err := auth.DecodeEncKey(cfg.ConfigEncKey); err != nil {
			log.Printf("config: CONFIG_ENC_KEY tidak valid, AI Provider Config dinonaktifkan: %v", err)
		} else {
			aiConfigEncKey = k
		}
	}
	aiSettingRepo := repository.NewAISettingRepo(db)
	aiSettingSvc := service.NewAISettingService(aiSettingRepo, hc, aiConfigEncKey)
	aiSettingH := handlers.NewAISettingHandler(aiSettingSvc, hc)

	chatRepo := repository.NewChatRepo(db)
	chatSvc := service.NewChatService(chatRepo, cfg)
	chatH := handlers.NewChatHandler(chatSvc, hc, cfg)

	auditRepo := repository.NewAuditRepo(db)

	// Telemetry (EP-17 ST-17.1) — best-effort metric events. Wired into each
	// handler/service via SetEmitter (optional, nil-safe) rather than their
	// constructors, so existing call sites/tests are unaffected.
	telemetryRepo := repository.NewTelemetryRepo(db)
	emitter := telemetry.NewEmitter(telemetryRepo)
	chatH.SetEmitter(emitter)

	// Learning hook (EP-16 TK-16.2.1) — WON/LOST outcomes and discovery
	// "Tolak" both push a short note to Hermes workspace memory via Chat.
	learningHook := ai.NewLearningHook(hc, hermes.SessionKey(cfg.WorkspaceSessionKey))

	tenderRepo := repository.NewTenderRepo(db)
	outcomeRepo := repository.NewOutcomeRepo(db)
	tenderSvc := service.NewTenderService(tenderRepo, outcomeRepo, learningHook)
	tenderSvc.SetEmitter(emitter)
	tenderH := handlers.NewTenderHandler(tenderSvc, auditRepo)
	tenderH.SetEmitter(emitter)

	prospectRepo := repository.NewProspectRepo(db)
	eventRepo := repository.NewEventRepo(db)
	eventSvc := service.NewEventService(eventRepo, prospectRepo)
	eventH := handlers.NewEventHandler(eventSvc, cfg.UploadDir)

	// Undangan event terjadwal (kirim dari server, bukan mailto). Mailer nil
	// bila SMTP belum dikonfigurasi — penjadwalan tetap jalan, pengiriman
	// ditandai gagal dengan pesan jelas (prinsip non-blocking). Scheduler
	// menyapu undangan jatuh tempo tiap menit; dijalankan dengan
	// context.Background() seperti reaper playbook (umur = proses).
	eventInviteRepo := repository.NewEventInviteRepo(db)
	var inviteMailer mailer.Mailer
	if cfg.SMTPConfigured() {
		inviteMailer = &mailer.SMTPMailer{
			Host:         cfg.SMTPHost,
			Port:         cfg.SMTPPort,
			Username:     cfg.SMTPUsername,
			Password:     cfg.SMTPPassword,
			EnvelopeFrom: cfg.SMTPFrom,
		}
	} else {
		log.Printf("event invite: SMTP belum dikonfigurasi (SMTP_HOST/SMTP_FROM), pengiriman undangan dinonaktifkan")
	}
	inviteLoc, lerr := time.LoadLocation(cfg.InviteTimezone)
	if lerr != nil {
		log.Printf("event invite: INVITE_TIMEZONE=%q tidak valid, pakai waktu lokal server: %v", cfg.InviteTimezone, lerr)
		inviteLoc = time.Local
	}
	eventInviteSvc := service.NewEventInviteService(eventInviteRepo, eventSvc, inviteMailer, inviteLoc)
	eventInviteH := handlers.NewEventInviteHandler(eventInviteSvc, userRepo)
	// Penjadwalan kirim-dari-server DITAHAN untuk sekarang: UI memakai mailto
	// (buka aplikasi email user). Scheduler hanya dijalankan bila SMTP benar
	// dikonfigurasi — jadi saat fitur ditahan, tak ada ticker menganggur.
	// Endpoint /invites tetap terdaftar (dormant) agar mudah dihidupkan lagi.
	if cfg.SMTPConfigured() {
		go eventInviteSvc.RunScheduler(context.Background())
	}

	projectRepo := repository.NewProjectRepo(db)
	projectSvc := service.NewProjectService(projectRepo)
	projectH := handlers.NewProjectHandler(projectSvc)
	// Jembatan tender WON → Proyek Berjalan: tender yang menang otomatis
	// membuat baris project. Diwire di sini karena projectSvc dibangun setelah
	// tenderSvc (pola setter opsional, sama seperti SetEmitter di atas).
	tenderSvc.SetProjectCreator(projectSvc)

	feedbackRepo := repository.NewFeedbackRepo(db)
	feedbackSvc := service.NewFeedbackService(feedbackRepo)
	feedbackH := handlers.NewFeedbackHandler(feedbackSvc)

	// AgentTaskRunner (model titip-tugas fire-and-forget) — dihitung di sini
	// (bukan di dekat playbook di bawah) karena Feedback Client's saran AI
	// asinkron butuh ini lebih dulu.
	var agentRunner hermes.AgentTaskRunner
	if r, ok := hc.(hermes.AgentTaskRunner); ok {
		agentRunner = r
	}

	// Feedback Client dinamis (form builder ala Google Form + bantuan AI) —
	// menggantikan form kaku 0023 sebagai jalur utama; yang lama dibiarkan
	// dorman (routes /feedback di bawah) demi kompatibilitas link lama. Saran
	// AI (susun kuesioner) memakai model titip-tugas ASINKRON yang sama dengan
	// playbook/analisa event — lihat feedback_form_service.go.
	feedbackFormRepo := repository.NewFeedbackFormRepo(db)
	// Job yang masih processing_ai saat server restart tak akan pernah
	// dilapor balik (goroutine dispatch tidak selamat dari restart) — tandai
	// gagal sekali di boot, sama seperti playbookJobRepo.FailInterrupted.
	feedbackFormSvc := service.NewFeedbackFormService(feedbackFormRepo, agentRunner, cfg.InternalAPIBaseURL, cfg.CronTriggerSecret)
	if err := feedbackFormSvc.ReapStaleAISuggest(context.Background(), 0); err != nil {
		log.Printf("feedback ai suggest: gagal menyapu job terputus: %v", err)
	}
	feedbackAISvc := service.NewFeedbackAIService(hc, hermes.SessionKey(cfg.WorkspaceSessionKey))
	feedbackFormH := handlers.NewFeedbackFormHandler(feedbackFormSvc, feedbackAISvc, userRepo)

	prospectSvc := service.NewProspectService(prospectRepo, outcomeRepo, learningHook)
	prospectSvc.SetEmitter(emitter)
	prospectH := handlers.NewProspectHandler(prospectSvc)

	profileRepo := repository.NewProfileRepo(db)
	profileExtractor := ai.NewExtractor(hc, hermes.SessionKey(cfg.WorkspaceSessionKey))
	// Crawl automation upsert (EP-12) — hermestui.NewCronClient returns nil
	// when HERMES_DASHBOARD_SESSION_TOKEN is unset, degrading the feature to
	// "not synced to Hermes" without affecting anything else. Assigned via an
	// explicit nil check, not straight into the CronJobUpserter interface
	// field: a nil *hermestui.CronClient boxed directly into an interface
	// value is NOT itself a nil interface, which would defeat every
	// `s.crawl.Client == nil` guard in crawl_automation.go.
	var cronUpserter service.CronJobUpserter
	if cc := hermestui.NewCronClient(cfg); cc != nil {
		cronUpserter = cc
	}
	crawlAutomation := &service.CrawlAutomation{
		Client:             cronUpserter,
		TriggerSecret:      cfg.CronTriggerSecret,
		InternalAPIBaseURL: cfg.InternalAPIBaseURL,
	}
	profileSvc := service.NewProfileService(profileRepo, cfg.UploadDir, profileExtractor, crawlAutomation)
	profileH := handlers.NewProfileHandler(profileSvc)

	sourceRepo := repository.NewSourceRepo(db)
	sourceSvc := service.NewSourceService(sourceRepo)
	sourceH := handlers.NewSourceHandler(sourceSvc)

	keywordSvc := service.NewKeywordService(hc, hermes.SessionKey(cfg.WorkspaceSessionKey))
	keywordH := handlers.NewKeywordHandler(keywordSvc)

	playbookDraftRepo := repository.NewPlaybookDraftRepo(db)

	scoreRepo := repository.NewProspectScoreRepo(db)
	scorer := ai.NewScorer(hc, hermes.SessionKey(cfg.WorkspaceSessionKey))
	scoreSvc := service.NewScoreService(scorer, scoreRepo, tenderSvc, prospectSvc, profileSvc)
	scoreSvc.SetEmitter(emitter)
	scoreH := handlers.NewScoreHandler(scoreSvc, auditRepo)

	// Playbook jobs (menu Playbooks + generate dari Event) — model TITIP-TUGAS
	// (callback via MCP): app titip instruksi ke Hermes lewat /v1/agent-task
	// (fire-and-forget), Hermes lapor balik lewat MCP tool save_playbook_job.
	// Ini SATU-SATUNYA jalur generate playbook (tender/prospect tidak punya).
	playbookJobRepo := repository.NewPlaybookJobRepo(db)
	// Job yang masih "berjalan" saat server restart tak akan pernah dilapor —
	// tandai failed sekali di boot.
	if err := playbookJobRepo.FailInterrupted(context.Background()); err != nil {
		log.Printf("playbook jobs: gagal menyapu job terputus: %v", err)
	}
	// agentRunner sudah dihitung di atas (dipakai lebih dulu oleh Feedback
	// Client's saran AI asinkron).
	// attachmentReader dibagi ke playbook (lampiran event → dokumen konteks) dan
	// Analisa AI, keduanya membaca berkas event dari UploadDir yang sama.
	attachmentReader := handlers.NewEventAttachmentReader(cfg.UploadDir)
	playbookJobSvc := service.NewPlaybookJobService(playbookJobRepo, agentRunner, profileSvc, eventSvc, attachmentReader, cfg.InternalAPIBaseURL, cfg.CronTriggerSecret)

	// Analisa AI memakai jalur titip-tugas yang SAMA dengan playbook, jadi ia
	// butuh AgentTaskRunner (bukan Client biasa) agar bisa fire-and-forget.
	eventAnalysisSvc := service.NewEventAnalysisService(eventSvc, profileSvc, agentRunner, attachmentReader, cfg.InternalAPIBaseURL, cfg.CronTriggerSecret)
	eventAnalysisH := handlers.NewEventAnalysisHandler(eventAnalysisSvc)
	playbookJobH := handlers.NewPlaybookJobHandler(playbookJobSvc, cfg.UploadDir)

	// Reaper berkala: job yang mandek (Hermes tak pernah lapor balik) ditandai
	// gagal agar UI tidak "Diproses" selamanya. Ambangnya HARUS lebih longgar
	// dari total waktu wajar satu tugas panjang: 3 percobaan x
	// HERMES_API_CALL_STALE_TIMEOUT (default 600 detik) = ~30 menit. Kalau
	// lebih ketat, job yang sebenarnya masih berjalan ikut dibunuh.
	go func() {
		ticker := time.NewTicker(2 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			rctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := playbookJobSvc.ReapStale(rctx, 40*time.Minute); err != nil {
				log.Printf("playbook reaper: %v", err)
			}
			// Analisa event memakai jalur asinkron yang sama, jadi butuh jaring
			// pengaman yang sama: tanpa ini, event yang laporannya tidak pernah
			// datang akan terkunci "running" selamanya.
			if err := eventAnalysisSvc.ReapStale(rctx, 40*time.Minute); err != nil {
				log.Printf("event analysis reaper: %v", err)
			}
			// Saran AI Feedback Client memakai jalur asinkron yang sama —
			// jaring pengaman yang sama juga (lihat ReapStaleAISuggest).
			if err := feedbackFormSvc.ReapStaleAISuggest(rctx, 40*time.Minute); err != nil {
				log.Printf("feedback ai suggest reaper: %v", err)
			}
			cancel()
		}
	}()

	dashboardSvc := service.NewDashboardService(prospectRepo, tenderRepo)
	dashboardH := handlers.NewDashboardHandler(dashboardSvc)

	reportRepo := repository.NewReportRepo(db)
	reportGen := ai.NewReportGenerator(hc, hermes.SessionKey(cfg.WorkspaceSessionKey))
	reportSvc := service.NewReportService(reportGen, reportRepo, dashboardSvc)
	reportSvc.SetEmitter(emitter)
	reportH := handlers.NewReportHandler(reportSvc, auditRepo)

	tenderAssist := ai.NewTenderAssist(hc, hermes.SessionKey(cfg.WorkspaceSessionKey))
	tenderAssistH := handlers.NewTenderAssistHandler(tenderAssist, tenderSvc, profileSvc)

	discoveryRunRepo := repository.NewDiscoveryRunRepo(db)
	discoveryCrawler := ai.NewHermesCrawler(hc, hermes.SessionKey(cfg.WorkspaceSessionKey))
	discoveryOrchestrator := ai.NewDiscoveryOrchestrator(discoveryCrawler, sourceRepo, profileSvc, auditRepo)
	discoverySvc := service.NewDiscoveryService(discoveryOrchestrator, tenderSvc, scoreSvc, discoveryRunRepo)
	discoveryH := handlers.NewDiscoveryHandler(discoverySvc, tenderSvc)

	// Discovery scheduler (EP-12 ST-12.5.1) — constructed here (not in
	// main.go) so InternalHandler below and main.go's own periodic
	// scheduler.Start(...) share this exact instance; a second, separately
	// constructed Scheduler would let their period-bucketed idempotency
	// keys and DiscoveryService instances drift apart.
	scheduler := ai.NewScheduler(profileSvc, discoverySvc, schedulerTickInterval)
	internalH := handlers.NewInternalHandler(scheduler, cfg.CronTriggerSecret, playbookJobSvc, eventAnalysisSvc, feedbackFormSvc)

	// Admin Hermes TUI (native Hermes CLI/TUI over a browser terminal,
	// proxied to the hermes-tui/ttyd sidecar) — feature degrades to simply
	// unavailable (routes unregistered below) rather than a boot crash if
	// HermesTuiBaseURL is somehow malformed, consistent with this codebase's
	// non-blocking configuration principle (see AI Provider Config above).
	hermesTuiSessionRepo := repository.NewHermesTuiSessionRepo(db)
	hermesTuiH, hermesTuiErr := handlers.NewHermesTuiHandler(hermesTuiSessionRepo, cfg)
	if hermesTuiErr != nil {
		log.Printf("hermes tui: HERMES_TUI_BASE_URL tidak valid, fitur dinonaktifkan: %v", hermesTuiErr)
	}

	// MCP server (EP-09) — mounted top-level like /healthz, not under /api,
	// and authenticated with a static Bearer token (SalesMCPToken), not JWT.
	mcpSrv := mcp.NewServer(mcp.Deps{
		Tender:       tenderSvc,
		Event:        eventSvc,
		Prospect:     prospectSvc,
		Profile:      profileSvc,
		ProspectRepo: prospectRepo,
		Audit:        auditRepo,
		Playbook:     playbookDraftRepo,
		PlaybookJob:  playbookJobRepo,
	})
	e.Any("/mcp", echo.WrapHandler(mcp.Handler(mcpSrv)), mcp.BearerAuth(cfg.SalesMCPToken))

	// Internal callback (EP-12 crawl automation via Hermes's own scheduler)
	// — mounted top-level like /mcp, secret-gated instead of JWT since the
	// caller is the Hermes cron job's agent turn, not a logged-in user.
	e.POST("/internal/discovery/trigger", internalH.TriggerDiscovery)
	e.GET("/internal/discovery/trigger", internalH.TriggerDiscovery)
	// Callback bridge saat playbook selesai (fire-and-forget model).
	e.POST("/internal/playbook-jobs/:id/complete", internalH.CompletePlaybookJob)
	e.POST("/internal/events/:id/analysis-complete", internalH.CompleteEventAnalysis)
	e.POST("/internal/feedback-forms/:id/ai-suggest-complete", internalH.CompleteFeedbackAISuggest)

	api := e.Group("/api")

	// Public auth routes (no JWT required).
	api.POST("/auth/login", authH.Login)
	api.POST("/auth/refresh", authH.Refresh)

	// Public feedback pasca-proyek (link dibagikan ke client, tanpa login).
	api.GET("/public/feedback/:token", feedbackH.PublicInfo)
	api.POST("/public/feedback/:token", feedbackH.PublicSubmit)

	// Form feedback dinamis publik (/form/:slug di FE) — tanpa login.
	api.GET("/public/forms/:slug", feedbackFormH.PublicInfo)
	api.POST("/public/forms/:slug", feedbackFormH.PublicSubmit)

	// Protected routes — require valid JWT.
	authd := api.Group("", auth.JWTMiddleware(cfg.JWTSecret))
	authd.GET("/me", authH.Me)

	authd.GET("/dashboard/summary", dashboardH.Summary, auth.RequireCapability(auth.CapViewData))

	// Hermes connectivity status (EP-17 ST-17.3) — distinct from /healthz
	// (DB liveness). Available to any authenticated role since Settings'
	// AI/Hermes tab (EP-18) is visible to more than just admins.
	authd.GET("/health/hermes", healthH.HermesHealth, auth.RequireCapability(auth.CapViewData))

	// User directory: read (CapViewUsers) is available to all roles so
	// owner/teammate names can be resolved for display; mutations
	// (CapManageUsers) remain Admin-only.
	users := authd.Group("/users")
	users.GET("", userH.List, auth.RequireCapability(auth.CapViewUsers))
	usersAdmin := users.Group("", auth.RequireCapability(auth.CapManageUsers))
	usersAdmin.POST("", userH.Create)
	usersAdmin.PATCH("/:id", userH.Update)
	usersAdmin.POST("/:id/reset-password", userH.ResetPassword)

	// Admin — Hermes ops (EP-16 ST-16.3), ADMIN only.
	admin := authd.Group("/admin", auth.RequireCapability(auth.CapManageUsers))
	admin.POST("/hermes/reset-memory", adminH.ResetHermesMemory)

	if hermesTuiH != nil {
		// Ticket issuance is JWT+RBAC gated like every other admin route.
		admin.POST("/hermes/tui/ticket", hermesTuiH.IssueTicket)

		// Everything below is DELIBERATELY registered on api, NOT authd —
		// browsers cannot attach a custom Authorization header to a plain
		// navigation (the iframe's src, /enter) or to the WS handshake
		// ttyd's own frontend initiates (/ws). Auth is the single-use
		// ticket (once, at /enter) then a session cookie (CookieGate, on
		// every request after). Do not "fix" this by moving these under
		// authd — that would silently break the whole feature, since
		// neither a navigation nor ttyd's own JS can supply a Bearer token.
		api.GET("/admin/hermes/tui/enter", hermesTuiH.Enter)

		tuiGated := api.Group("/admin/hermes/tui", hermesTuiH.CookieGate)
		tuiGated.POST("/end", hermesTuiH.EndSession)
		// The dashboard exposes several WS endpoints (unlike ttyd's single
		// /ws) — registered explicitly so echo's router recognizes them as
		// WS upgrades rather than falling through to the HTTP catch-all.
		for _, wsPath := range []string{"/api/pty", "/api/ws", "/api/pub", "/api/events"} {
			tuiGated.GET(wsPath, hermesTuiH.ProxyWS)
		}
		// Catch-all for the dashboard's own HTML/JS/CSS/API — path prefix
		// stripped and X-Forwarded-Prefix set by ProxyHTTP's Director.
		tuiGated.Any("", hermesTuiH.ProxyHTTP)
		tuiGated.Any("/*", hermesTuiH.ProxyHTTP)
	}

	// Settings — AI/Hermes status + test koneksi (EP-18 ST-18.1). CapViewData
	// (semua role) sama seperti /health/hermes — tab AI/Hermes di Settings
	// terbuka untuk semua role, hanya aksi reset/config yang admin-only.
	settings := authd.Group("/settings", auth.RequireCapability(auth.CapViewData))
	settings.GET("/hermes", settingsH.HermesStatus)
	settings.POST("/hermes/test", settingsH.TestHermes)

	// Settings — AI Provider Config (EP-18 ST-18.4), ADMIN only (sensitive:
	// provider/model/API-key). Distinct group from /settings above since
	// RBAC differs (CapManageUsers, not CapViewData).
	settingsAI := authd.Group("/settings/ai", auth.RequireCapability(auth.CapManageUsers))
	settingsAI.GET("", aiSettingH.Get)
	settingsAI.PUT("", aiSettingH.Update)
	settingsAI.POST("/test", aiSettingH.Test)

	// Tenders — semua role yang punya CapCRUDData.
	tenders := authd.Group("/tenders", auth.RequireCapability(auth.CapCRUDData))
	tenders.GET("", tenderH.List)
	tenders.POST("", tenderH.Create)
	tenders.GET("/:id", tenderH.Get)
	tenders.PUT("/:id", tenderH.Update)
	tenders.DELETE("/:id", tenderH.Delete)
	tenders.PATCH("/:id/status", tenderH.UpdateStatus)
	tenders.POST("/:id/outcome", tenderH.RecordOutcome)
	tenders.POST("/:id/promote", tenderH.Promote)
	tenders.POST("/:id/review", tenderH.Review)
	tenders.POST("/:id/score", scoreH.ScoreTender)
	tenders.GET("/:id/score", scoreH.GetTenderScore)
	tenders.POST("/:id/doc-checklist", tenderAssistH.DocChecklist)
	tenders.POST("/:id/proposal-draft", tenderAssistH.ProposalDraft)

	// Events — semua role yang punya CapCRUDData.
	projects := authd.Group("/projects", auth.RequireCapability(auth.CapCRUDData))
	projects.GET("", projectH.List)
	projects.POST("", projectH.Create)
	projects.GET("/summary", projectH.Summary)
	projects.GET("/:id", projectH.Get)
	projects.PUT("/:id", projectH.Update)
	projects.DELETE("/:id", projectH.Delete)
	projects.POST("/:id/activities", projectH.AddActivity)

	feedback := authd.Group("/feedback", auth.RequireCapability(auth.CapCRUDData))
	feedback.GET("", feedbackH.List)
	feedback.POST("", feedbackH.Create)
	feedback.GET("/analytics", feedbackH.Analytics)
	feedback.DELETE("/:id", feedbackH.Delete)

	// Feedback Client dinamis (menu Feedback Client + Analisa Feedback).
	// Rute statis ("/analytics", "/ai/*") didaftarkan sebelum "/:id" agar tidak
	// tertangkap sebagai parameter route.
	feedbackForms := authd.Group("/feedback-forms", auth.RequireCapability(auth.CapCRUDData))
	feedbackForms.GET("", feedbackFormH.List)
	feedbackForms.POST("", feedbackFormH.Create)
	feedbackForms.GET("/analytics", feedbackFormH.AnalyticsGlobal)
	feedbackForms.POST("/ai/suggest", feedbackFormH.AISuggest)
	feedbackForms.POST("/ai/refine", feedbackFormH.AIRefine)
	feedbackForms.POST("/ai/analyze", feedbackFormH.AIAnalyze)
	feedbackForms.GET("/:id", feedbackFormH.Get)
	feedbackForms.PUT("/:id", feedbackFormH.Update)
	feedbackForms.DELETE("/:id", feedbackFormH.Delete)
	feedbackForms.POST("/:id/publish", feedbackFormH.Publish)
	feedbackForms.GET("/:id/submissions", feedbackFormH.Submissions)
	feedbackForms.GET("/:id/analytics", feedbackFormH.Analytics)
	// Saran AI async: jawab klarifikasi (memicu putaran berikutnya) & bersihkan
	// job sementara (setelah dipilih/ditambahkan, atau dibatalkan).
	feedbackForms.POST("/:id/ai/suggest/clarify", feedbackFormH.AISuggestClarify)
	feedbackForms.POST("/:id/ai/suggest/clear", feedbackFormH.AISuggestClear)

	events := authd.Group("/events", auth.RequireCapability(auth.CapCRUDData))
	events.GET("", eventH.List)
	events.POST("", eventH.Create)
	events.GET("/:id", eventH.Get)
	events.PUT("/:id", eventH.Update)
	events.DELETE("/:id", eventH.Delete)
	events.POST("/attachments", eventH.UploadAttachment)
	events.POST("/:id/convert", eventH.Convert)
	events.POST("/:id/analyze", eventAnalysisH.Analyze)
	// Generate playbook terstandarisasi dari konteks event (async job).
	events.POST("/:id/playbook-job", playbookJobH.CreateFromEvent)
	// Undangan event terjadwal (kirim dari server ke daftar peserta).
	events.GET("/:id/invites", eventInviteH.List)
	events.POST("/:id/invites", eventInviteH.Schedule)
	events.DELETE("/:id/invites/:inviteId", eventInviteH.Cancel)

	// Prospects — semua role yang punya CapCRUDData.
	prospects := authd.Group("/prospects", auth.RequireCapability(auth.CapCRUDData))
	prospects.GET("", prospectH.List)
	prospects.POST("", prospectH.Create)
	prospects.GET("/:id", prospectH.Get)
	prospects.PUT("/:id", prospectH.Update)
	prospects.DELETE("/:id", prospectH.Delete)
	prospects.PATCH("/:id/stage", prospectH.UpdateStage)
	prospects.POST("/:id/score", scoreH.ScoreProspect)
	prospects.GET("/:id/score", scoreH.GetProspectScore)

	// Playbook jobs (menu Playbooks) — generate async + riwayat status.
	playbookJobs := authd.Group("/playbook-jobs", auth.RequireCapability(auth.CapCRUDData))
	playbookJobs.GET("", playbookJobH.List)
	playbookJobs.POST("", playbookJobH.Create)
	playbookJobs.GET("/:id", playbookJobH.Get)
	playbookJobs.POST("/:id/refine", playbookJobH.Refine)
	playbookJobs.POST("/:id/retry", playbookJobH.Retry)
	playbookJobs.DELETE("/:id", playbookJobH.Delete)

	// Profile ("Otak Agent") — GET boleh semua role (SALES read-only),
	// PUT (buat versi baru) hanya OPS/MANAGER/ADMIN.
	profile := authd.Group("/profile")
	profile.GET("", profileH.Get, auth.RequireCapability(auth.CapViewData))
	profile.PUT("", profileH.Update, auth.RequireCapability(auth.CapEditProfile))
	profile.POST("/ingest", profileH.Ingest, auth.RequireCapability(auth.CapEditProfile))
	profile.POST("/keywords/generate", keywordH.Generate, auth.RequireCapability(auth.CapEditProfile))

	// Sources — read (CapViewData) semua role; mutasi & aktivasi preset
	// (CapEditProfile) OPS/MANAGER/ADMIN. "/presets" didaftarkan sebelum
	// "/:id" agar tidak tertangkap sebagai parameter route.
	sources := authd.Group("/sources")
	sources.GET("", sourceH.List, auth.RequireCapability(auth.CapViewData))
	sources.GET("/presets", sourceH.Presets, auth.RequireCapability(auth.CapViewData))
	sources.GET("/:id", sourceH.Get, auth.RequireCapability(auth.CapViewData))
	sourcesEdit := sources.Group("", auth.RequireCapability(auth.CapEditProfile))
	sourcesEdit.POST("", sourceH.Create)
	sourcesEdit.POST("/presets", sourceH.ActivatePreset)
	sourcesEdit.PUT("/:id", sourceH.Update)
	sourcesEdit.DELETE("/:id", sourceH.Delete)

	// Discovery — OPS/MANAGER/ADMIN (CapRunDiscovery).
	discovery := authd.Group("/discovery", auth.RequireCapability(auth.CapRunDiscovery))
	discovery.POST("/run", discoveryH.Run)
	discovery.GET("/runs", discoveryH.ListRuns)
	discovery.GET("/inbox", discoveryH.Inbox)

	// Reports — GET (CapViewData) semua role; generate/hapus (CapCRUDData)
	// karena keduanya memicu AI/mengubah data tersimpan, konsisten dengan
	// tenders/prospects/playbooks.
	reports := authd.Group("/reports")
	reports.GET("", reportH.List, auth.RequireCapability(auth.CapViewData))
	reports.GET("/:id", reportH.Get, auth.RequireCapability(auth.CapViewData))
	reportsEdit := reports.Group("", auth.RequireCapability(auth.CapCRUDData))
	reportsEdit.POST("", reportH.Create)
	reportsEdit.DELETE("/:id", reportH.Delete)

	// Chat — semua role yang punya CapUseAI.
	convs := authd.Group("/conversations", auth.RequireCapability(auth.CapUseAI))
	convs.POST("", chatH.Create)
	convs.GET("", chatH.List)
	convs.GET("/:id", chatH.Get)
	convs.POST("/:id/chat", chatH.Chat)
	convs.DELETE("/:id", chatH.Delete)

	return e, profileSvc, discoverySvc, aiSettingSvc, hermesTuiH, scheduler
}

func healthzHandler(db *gorm.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		sqlDB, err := db.DB()
		if err != nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "degraded", "db": "down"})
		}

		ctx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Second)
		defer cancel()

		if err := sqlDB.PingContext(ctx); err != nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "degraded", "db": "down"})
		}

		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}
}
