package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"spsyncapi/internal/config"
	"spsyncapi/internal/middleware"
	"spsyncapi/internal/routes"
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

	gin.SetMode(cfg.GinMode)

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(middleware.Observability(logger, metrics))

	routes.Register(engine)

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
