package http

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gorm.io/gorm"

	"salespilot/internal/auth"
	"salespilot/internal/config"
	"salespilot/internal/hermes"
	"salespilot/internal/http/handlers"
	"salespilot/internal/repository"
	"salespilot/internal/service"
)

// New creates and configures the Echo instance (middleware + routes).
func New(cfg *config.Config, db *gorm.DB, hc hermes.Client) *echo.Echo {
	e := echo.New()
	e.HideBanner = true

	e.Validator = NewValidator()

	e.Use(middleware.Logger()) //nolint:staticcheck // EP-17 will replace with structured RequestLogger
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

	chatRepo := repository.NewChatRepo(db)
	chatSvc := service.NewChatService(chatRepo, cfg)
	chatH := handlers.NewChatHandler(chatSvc, hc, cfg)

	tenderRepo := repository.NewTenderRepo(db)
	outcomeRepo := repository.NewOutcomeRepo(db)
	tenderSvc := service.NewTenderService(tenderRepo, outcomeRepo, service.NoopLearningHook())
	tenderH := handlers.NewTenderHandler(tenderSvc)

	prospectRepo := repository.NewProspectRepo(db)
	eventRepo := repository.NewEventRepo(db)
	eventSvc := service.NewEventService(eventRepo, prospectRepo)
	eventH := handlers.NewEventHandler(eventSvc)

	prospectSvc := service.NewProspectService(prospectRepo, outcomeRepo, service.NoopLearningHook())
	prospectH := handlers.NewProspectHandler(prospectSvc)

	api := e.Group("/api")

	// Public auth routes (no JWT required).
	api.POST("/auth/login", authH.Login)
	api.POST("/auth/refresh", authH.Refresh)

	// Protected routes — require valid JWT.
	authd := api.Group("", auth.JWTMiddleware(cfg.JWTSecret))
	authd.GET("/me", authH.Me)

	// User directory: read (CapViewUsers) is available to all roles so
	// owner/teammate names can be resolved for display; mutations
	// (CapManageUsers) remain Admin-only.
	users := authd.Group("/users")
	users.GET("", userH.List, auth.RequireCapability(auth.CapViewUsers))
	usersAdmin := users.Group("", auth.RequireCapability(auth.CapManageUsers))
	usersAdmin.POST("", userH.Create)
	usersAdmin.PATCH("/:id", userH.Update)
	usersAdmin.POST("/:id/reset-password", userH.ResetPassword)

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

	// Chat — semua role yang punya CapUseAI.
	convs := authd.Group("/conversations", auth.RequireCapability(auth.CapUseAI))
	convs.POST("", chatH.Create)
	convs.GET("", chatH.List)
	convs.GET("/:id", chatH.Get)
	convs.POST("/:id/chat", chatH.Chat)

	return e
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
