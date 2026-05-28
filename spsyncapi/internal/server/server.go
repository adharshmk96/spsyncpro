package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"spsyncapi/internal/auth"
	"spsyncapi/internal/backupjob"
	"spsyncapi/internal/backuprun"
	"spsyncapi/internal/bucketstore"
	"spsyncapi/internal/config"
	"spsyncapi/internal/crypto"
	"spsyncapi/internal/handlers"
	"spsyncapi/internal/middleware"
	"spsyncapi/internal/organization"
	"spsyncapi/internal/restorejob"
	"spsyncapi/internal/restorerun"
	"spsyncapi/internal/routes"
	"spsyncapi/internal/storage"
	"spsyncapi/internal/telemetry"
	"spsyncapi/internal/temporal"

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
	orgRepo := storage.NewOrganizationRepository(db)
	bucketStoreRepo := storage.NewBucketStoreRepository(db)
	backupJobRepo := storage.NewBackupJobRepository(db)
	backupRunRepo := storage.NewBackupRunRepository(db)
	restoreJobRepo := storage.NewRestoreJobRepository(db)
	restoreRunRepo := storage.NewRestoreRunRepository(db)

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

	encryptor, err := crypto.NewSecretEncryptor(cfg.Encryption.Secret)
	if err != nil {
		return nil, fmt.Errorf("create secret encryptor: %w", err)
	}

	orgSvc, err := organization.NewService(organization.ServiceConfig{
		Repo:      orgRepo,
		Encryptor: encryptor,
		Logger:    logger,
	})
	if err != nil {
		return nil, fmt.Errorf("create organization service: %w", err)
	}

	bucketStoreSvc, err := bucketstore.NewService(bucketstore.ServiceConfig{
		Repo:      bucketStoreRepo,
		Encryptor: encryptor,
		Logger:    logger,
	})
	if err != nil {
		return nil, fmt.Errorf("create bucket store service: %w", err)
	}

	temporalClient, err := temporal.NewClient(cfg.Temporal)
	if err != nil {
		return nil, fmt.Errorf("temporal client: %w", err)
	}

	scheduleOrchestrator := temporal.NewScheduleOrchestrator(temporalClient, cfg.Temporal, logger)
	runExecutor := temporal.NewRunExecutor(temporalClient, cfg.Temporal)

	backupJobSvc, err := backupjob.NewService(backupjob.ServiceConfig{
		Repo:           backupJobRepo,
		OrgRepo:        orgRepo,
		BucketRepo:     bucketStoreRepo,
		ScheduleSyncer: scheduleOrchestrator,
		Logger:         logger,
	})
	if err != nil {
		return nil, fmt.Errorf("create backup job service: %w", err)
	}

	restoreJobSvc, err := restorejob.NewService(restorejob.ServiceConfig{
		Repo:       restoreJobRepo,
		OrgRepo:    orgRepo,
		BucketRepo: bucketStoreRepo,
		Logger:     logger,
	})
	if err != nil {
		return nil, fmt.Errorf("create restore job service: %w", err)
	}

	backupRunSvc, err := backuprun.NewService(backuprun.ServiceConfig{
		RunRepo:  backupRunRepo,
		JobRepo:  backupJobRepo,
		Executor: runExecutor,
		Logger:   logger,
	})
	if err != nil {
		return nil, fmt.Errorf("create backup run service: %w", err)
	}

	restoreRunSvc, err := restorerun.NewService(restorerun.ServiceConfig{
		RunRepo:  restoreRunRepo,
		JobRepo:  restoreJobRepo,
		Executor: runExecutor,
		Logger:   logger,
	})
	if err != nil {
		return nil, fmt.Errorf("create restore run service: %w", err)
	}

	// --- gin engine --------------------------------------------------------
	gin.SetMode(cfg.GinMode)

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(middleware.Observability(logger, metrics))

	routes.Register(engine, routes.Deps{
		AuthHandler:         handlers.NewAuthHandler(authSvc, logger),
		OrganizationHandler: handlers.NewOrganizationHandler(orgSvc, logger),
		BucketStoreHandler:  handlers.NewBucketStoreHandler(bucketStoreSvc, logger),
		BackupJobHandler:    handlers.NewBackupJobHandler(backupJobSvc, logger),
		BackupRunHandler:    handlers.NewBackupRunHandler(backupRunSvc, logger),
		RestoreJobHandler:   handlers.NewRestoreJobHandler(restoreJobSvc, logger),
		RestoreRunHandler:   handlers.NewRestoreRunHandler(restoreRunSvc, logger),
		AuthService:         authSvc,
		JWTConfig:           jwtCfg,
		Logger:              logger,
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
