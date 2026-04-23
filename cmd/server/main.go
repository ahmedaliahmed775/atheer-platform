package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/atheer-payment/atheer-platform/internal/config"
	"github.com/atheer-payment/atheer-platform/internal/handler"
	"github.com/atheer-payment/atheer-platform/internal/middleware"
	"github.com/atheer-payment/atheer-platform/internal/repository"
	"github.com/atheer-payment/atheer-platform/internal/router"
	"github.com/atheer-payment/atheer-platform/internal/service"
)

func main() {
	// === Setup Logger ===
	config.SetupLogger()
	slog.Info("Starting Atheer Switch V3.0")

	// === Load Configuration ===
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// === Connect to PostgreSQL ===
	db, err := config.NewDatabasePool(ctx, cfg.Database)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// === Connect to Redis ===
	rdb, err := config.NewRedisClient(ctx, cfg.Redis)
	if err != nil {
		slog.Error("Failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()

	// === Initialize Repositories ===
	deviceRepo := repository.NewDeviceRepository(db)
	txRepo := repository.NewTransactionRepository(db)
	limitsRepo := repository.NewLimitsMatrixRepository(db)
	disputeRepo := repository.NewDisputeRepository(db)
	_ = repository.NewAuditLogRepository(db)

	// === Initialize Services ===
	enrollService := service.NewEnrollmentService(deviceRepo)
	deviceService := service.NewDeviceService(deviceRepo)
	txService := service.NewTransactionService(txRepo, deviceRepo)
	disputeService := service.NewDisputeService(disputeRepo, txRepo)
	limitsService := service.NewLimitsService(limitsRepo)

	// === Initialize Handlers ===
	healthHandler := handler.NewHealthHandler(db, rdb)
	enrollHandler := handler.NewEnrollHandler(enrollService)
	txHandler := handler.NewTransactionHandler(txService)
	deviceHandler := handler.NewDeviceHandler(deviceService)
	disputeHandler := handler.NewDisputeHandler(disputeService)
	limitsHandler := handler.NewLimitsHandler(limitsService)

	// === Initialize Pipeline Middleware (v3.0) ===
	switchRecordRepo := repository.NewSwitchRecordRepository(db)

	rateLimiter := middleware.NewRateLimiter(rdb,
		cfg.Limits.RateLimitPerDevice,
		cfg.Limits.RateLimitPerWallet,
		cfg.Limits.RateLimitPerIP,
	)
	requestLogger := middleware.NewRequestLogger()
	antiReplay := middleware.NewAntiReplay(rdb)
	limitsChecker := middleware.NewLimitsChecker(limitsRepo, txRepo, switchRecordRepo)
	sigVerifierA := middleware.NewSignatureVerifierA(switchRecordRepo)
	payeeTypeVerifier := middleware.NewPayeeTypeVerifier(switchRecordRepo)
	txTypeResolver := middleware.NewTransactionTypeResolver()

	// === Setup Router with all dependencies ===
	deps := &router.Dependencies{
		HealthHandler:      healthHandler,
		EnrollHandler:      enrollHandler,
		TransactionHandler: txHandler,
		DeviceHandler:      deviceHandler,
		DisputeHandler:     disputeHandler,
		LimitsHandler:      limitsHandler,

		RateLimiter:       rateLimiter,
		RequestLogger:     requestLogger,
		AntiReplay:        antiReplay,
		LimitsChecker:     limitsChecker,
		SigVerifierA:      sigVerifierA,
		PayeeTypeVerifier: payeeTypeVerifier,
		TxTypeResolver:    txTypeResolver,
	}

	r := router.New(deps)

	// === Start HTTP Server ===
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		slog.Info("Atheer Switch listening",
			"port", cfg.Server.Port,
			"version", "3.0.0",
			"pipeline", "10 layers active",
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	slog.Info("Shutting down server", "signal", sig)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server shutdown failed", "error", err)
		os.Exit(1)
	}

	slog.Info("Atheer Switch stopped gracefully")
}
