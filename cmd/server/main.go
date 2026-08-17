// Package main is the entrypoint for the design-review platform server.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/cry048/design-review-platform/internal/config"
	"github.com/cry048/design-review-platform/internal/domain/annotation"
	domainaudit "github.com/cry048/design-review-platform/internal/domain/audit"
	"github.com/cry048/design-review-platform/internal/middleware"
	memaudit "github.com/cry048/design-review-platform/internal/platform/audit"
	"github.com/cry048/design-review-platform/internal/platform/notify"
	"github.com/cry048/design-review-platform/internal/platform/storage"
	"github.com/cry048/design-review-platform/internal/repository/memory"
	"github.com/cry048/design-review-platform/internal/seed"
	"github.com/cry048/design-review-platform/internal/service"
	transport "github.com/cry048/design-review-platform/internal/transport/http"
)

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

type uuidGen struct{}

func (uuidGen) New() string { return uuid.NewString() }

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config load failed: %s\n", err)
		os.Exit(1)
	}

	logger, err := buildLogger(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger init failed: %s\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("starting design-review platform", zap.String("config", cfg.String()))

	auditLogger := memaudit.NewMemoryLogger()
	notif := notify.New()

	storageLimits := storage.Limits{
		MaxBytes:     cfg.Storage.MaxBytes,
		AllowedTypes: cfg.AllowedTypesMap(),
		BaseDir:      cfg.Storage.BaseDir,
	}
	store, err := storage.New(storageLimits)
	if err != nil {
		logger.Fatal("storage init failed", zap.Error(err))
	}

	projs := memory.NewProjectRepo()
	annos := memory.NewAnnotationRepo()
	revs := memory.NewReviewRepo()
	runner := memory.NewTxRunner()

	projSvc := &service.ProjectService{
		Projects: projs,
		Audit:    auditAdapter{auditLogger},
		Clock:    realClock{},
		IDs:      uuidGen{},
	}
	annoSvc := &service.AnnotationService{
		Projects:            projs,
		Annotations:         annos,
		Reviews:             revs,
		Audit:               auditAdapter{auditLogger},
		Notifier:            notifierAdapter{notif},
		Storage:             store,
		Clock:               realClock{},
		IDs:                 uuidGen{},
		AttachmentValidator: storeAdapter{store},
		TxRunner:            runner,
	}
	revSvc := &service.ReviewService{
		Projects:    projs,
		Annotations: annos,
		Reviews:     revs,
		Audit:       auditAdapter{auditLogger},
		Clock:       realClock{},
		IDs:         uuidGen{},
		TxRunner:    runner,
	}
	auditSvc := &service.AuditService{Logger: auditLogger}
	notifSvc := &service.NotifierService{N: notif}
	exportSvc := &service.ExportService{
		Annotations: annos,
		Reviews:     revs,
		Projects:    projs,
	}

	if cfg.Runtime.SeedOnStartup {
		seedCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		err := seed.Run(seedCtx, projSvc, annoSvc, revSvc, realClock{}, uuidGen{})
		cancel()
		if err != nil {
			logger.Warn("seed run failed", zap.Error(err))
		} else {
			logger.Info("seed data loaded")
		}
	}

	v := validator.New()
	handlers := &transport.Handlers{
		Projects:    projSvc,
		Annotations: annoSvc,
		Reviews:     revSvc,
		Notifier:    notifSvc,
		Audit:       auditSvc,
		Export:      exportSvc,
		Validator:   v,
	}
	engine := transport.NewRouter(cfg, logger, handlers, auditLogger)

	srv := &http.Server{
		Addr:         cfg.HTTP.Addr,
		Handler:      engine,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
	}

	go func() {
		logger.Info("http listening", zap.String("addr", cfg.HTTP.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("listen failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutdown signal received")
	ctx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", zap.Error(err))
	}
	logger.Info("server stopped")
}

func buildLogger(cfg *config.Config) (*zap.Logger, error) {
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(cfg.Log.Level)); err != nil {
		return nil, err
	}
	zcfg := zap.NewProductionConfig()
	zcfg.Level = zap.NewAtomicLevelAt(level)
	zcfg.Encoding = cfg.Log.Format
	if cfg.Log.Output == "stdout" || cfg.Log.Output == "" {
		zcfg.OutputPaths = []string{"stdout"}
		zcfg.ErrorOutputPaths = []string{"stderr"}
	} else {
		zcfg.OutputPaths = []string{cfg.Log.Output}
		zcfg.ErrorOutputPaths = []string{cfg.Log.Output}
	}
	return zcfg.Build()
}

type auditAdapter struct{ l *memaudit.MemoryLogger }

func (a auditAdapter) Log(ctx context.Context, action, actorID, entityType, entityID, before, after string) error {
	return a.l.Log(ctx, domainaudit.Entry{
		ActorID:    actorID,
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		Before:     before,
		After:      after,
		At:         time.Now(),
		RequestID:  requestIDFromCtx(ctx),
	})
}

type notifierAdapter struct{ n *notify.Notifier }

func (n notifierAdapter) NotifyAnnotationAssigned(ctx context.Context, a *annotation.Annotation) error {
	return n.n.NotifyAnnotationAssigned(ctx, a)
}
func (n notifierAdapter) NotifyAnnotationResolved(ctx context.Context, a *annotation.Annotation) error {
	return n.n.NotifyAnnotationResolved(ctx, a)
}
func (n notifierAdapter) NotifyAnnotationReopened(ctx context.Context, a *annotation.Annotation) error {
	return n.n.NotifyAnnotationReopened(ctx, a)
}
func (n notifierAdapter) NotifyDueApproaching(ctx context.Context, a *annotation.Annotation, when time.Time) error {
	return n.n.NotifyDueApproaching(ctx, a, when)
}

type storeAdapter struct{ s *storage.Store }

func (s storeAdapter) ValidateUpload(mediaType string, size int64) error {
	return s.s.ValidateUpload(mediaType, size)
}

func requestIDFromCtx(ctx context.Context) string {
	if v, ok := ctx.Value(middleware.RequestIDKey).(string); ok {
		return v
	}
	return ""
}
