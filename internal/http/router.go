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
	"salespilot/internal/http/handlers"
	applog "salespilot/internal/log"
	"salespilot/internal/mcp"
	"salespilot/internal/repository"
	"salespilot/internal/service"
	"salespilot/internal/telemetry"
)

// New creates and configures the Echo instance (middleware + routes). It
// also returns ProfileService and DiscoveryService so apps/api/main.go can
// wire the discovery Scheduler (EP-12 ST-12.5.1) to the exact same instances
// the HTTP handlers use — a second, separately-constructed pair would risk
// drifting out of sync (e.g. a different DiscoveryRunRepository instance).
// AISettingService is returned the same way so main.go can rehydrate the
// bridge's config at boot (EP-18 TK-18.4.4) without re-decoding
// CONFIG_ENC_KEY a second time.
func New(cfg *config.Config, db *gorm.DB, hc hermes.Client) (*echo.Echo, *service.ProfileService, *service.DiscoveryService, *service.AISettingService) {
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
	eventH := handlers.NewEventHandler(eventSvc)

	prospectSvc := service.NewProspectService(prospectRepo, outcomeRepo, learningHook)
	prospectSvc.SetEmitter(emitter)
	prospectH := handlers.NewProspectHandler(prospectSvc)

	profileRepo := repository.NewProfileRepo(db)
	profileExtractor := ai.NewExtractor(hc, hermes.SessionKey(cfg.WorkspaceSessionKey))
	profileSvc := service.NewProfileService(profileRepo, cfg.UploadDir, profileExtractor)
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

	playbookRepo := repository.NewPlaybookRepo(db)
	playbookGen := ai.NewPlaybookGenerator(hc, hermes.SessionKey(cfg.WorkspaceSessionKey))
	playbookSvc := service.NewPlaybookService(playbookGen, playbookRepo, tenderSvc, prospectSvc, profileSvc)
	playbookH := handlers.NewPlaybookHandler(playbookSvc, auditRepo)

	dashboardSvc := service.NewDashboardService(prospectRepo, tenderRepo)
	dashboardH := handlers.NewDashboardHandler(dashboardSvc)

	reportRepo := repository.NewReportRepo(db)
	reportGen := ai.NewReportGenerator(hc, hermes.SessionKey(cfg.WorkspaceSessionKey))
	reportSvc := service.NewReportService(reportGen, reportRepo, dashboardSvc)
	reportSvc.SetEmitter(emitter)
	reportH := handlers.NewReportHandler(reportSvc, auditRepo)

	discoveryRunRepo := repository.NewDiscoveryRunRepo(db)
	discoveryCrawler := ai.NewHermesCrawler(hc, hermes.SessionKey(cfg.WorkspaceSessionKey))
	discoveryOrchestrator := ai.NewDiscoveryOrchestrator(discoveryCrawler, sourceRepo, profileSvc, auditRepo)
	discoverySvc := service.NewDiscoveryService(discoveryOrchestrator, tenderSvc, scoreSvc, discoveryRunRepo)
	discoveryH := handlers.NewDiscoveryHandler(discoverySvc, tenderSvc)

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
	})
	e.Any("/mcp", echo.WrapHandler(mcp.Handler(mcpSrv)), mcp.BearerAuth(cfg.SalesMCPToken))

	api := e.Group("/api")

	// Public auth routes (no JWT required).
	api.POST("/auth/login", authH.Login)
	api.POST("/auth/refresh", authH.Refresh)

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
	tenders.POST("/:id/playbook", playbookH.GenerateTender)
	tenders.GET("/:id/playbooks", playbookH.ListTenderPlaybooks)

	// Events — semua role yang punya CapCRUDData.
	events := authd.Group("/events", auth.RequireCapability(auth.CapCRUDData))
	events.GET("", eventH.List)
	events.POST("", eventH.Create)
	events.GET("/:id", eventH.Get)
	events.PUT("/:id", eventH.Update)
	events.DELETE("/:id", eventH.Delete)
	events.POST("/:id/convert", eventH.Convert)

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
	prospects.POST("/:id/playbook", playbookH.GenerateProspect)
	prospects.GET("/:id/playbooks", playbookH.ListProspectPlaybooks)

	// Playbooks — akses by-id lintas target-type (CapCRUDData, sama seperti tenders/prospects).
	playbooks := authd.Group("/playbooks", auth.RequireCapability(auth.CapCRUDData))
	playbooks.GET("/:id", playbookH.Get)

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

	return e, profileSvc, discoverySvc, aiSettingSvc
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
