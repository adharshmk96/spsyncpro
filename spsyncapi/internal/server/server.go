package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"spsyncapi/internal/auth"
	"spsyncapi/internal/config"
	"spsyncapi/internal/handlers"
	"spsyncapi/internal/middleware"
	"spsyncapi/internal/routes"
	"spsyncapi/internal/storage"
	"spsyncapi/internal/telemetry"

	"github.com/gin-gonic/gin"
)

type Server struct {
	cfg    *config.Config
	logger *slog.Logger
	engine *gin.Engine
	http   *http.Server
}

func New(cfg *config.Config, logger *slog.Logger, metrics *telemetry.HTTPMetrics) (*Server, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	if metrics == nil {
		return nil, fmt.Errorf("metrics is required")
	}

	// --- storage -----------------------------------------------------------
	db, err := storage.Open(cfg.DB.SQLitePath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	memberRepo := storage.NewMemberRepository(db)
	sessionRepo := storage.NewSessionRepository(db)
	resetRepo := storage.NewPasswordResetRepository(db)

	// --- JWT config --------------------------------------------------------
	jwtCfg := auth.JWTConfig{
		Secret:    []byte(cfg.Auth.JWTSecret),
		Issuer:    cfg.Auth.JWTIssuer,
		AccessTTL: cfg.Auth.AccessTokenTTL,
	}

	// --- auth service ------------------------------------------------------
	authSvc, err := auth.NewService(auth.ServiceConfig{
		Members:    memberRepo,
		Sessions:   sessionRepo,
		Resets:     resetRepo,
		JWTConfig:  jwtCfg,
		SessionTTL: cfg.Auth.SessionTTL,
		ResetTTL:   cfg.Auth.PasswordResetTTL,
		Logger:     logger,
	})
	if err != nil {
		return nil, fmt.Errorf("create auth service: %w", err)
	}

	// --- gin engine --------------------------------------------------------
	gin.SetMode(cfg.GinMode)

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(middleware.Observability(logger, metrics))

	routes.Register(engine, routes.Deps{
		AuthHandler: handlers.NewAuthHandler(authSvc, logger),
		AuthService: authSvc,
		JWTConfig:   jwtCfg,
		Logger:      logger,
	})

	return &Server{
		cfg:    cfg,
		logger: logger,
		engine: engine,
		http: &http.Server{
			Addr:         cfg.Address(),
			Handler:      engine,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
		},
	}, nil
}

func (s *Server) Start() error {
	s.logger.Info("listening", "address", s.cfg.Address())

	if err := s.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}
