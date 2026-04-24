// Modified for v3.0 Security Hardening
package main

import (
        "context"
        "crypto/tls"
        "crypto/x509"
        "fmt"
        "log/slog"
        "net/http"
        "os"
        "os/signal"
        "syscall"
        "time"

        "github.com/atheer-payment/atheer-platform/internal/adapter"
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
        slog.Info("Starting Atheer Switch V3.0 — Security Hardened")

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

        // === Initialize KMS for Seed encryption ===
        kms, err := service.NewLocalKMS()
        if err != nil {
                slog.Error("Failed to initialize KMS", "error", err)
                os.Exit(1)
        }
        slog.Info("KMS initialized for Seed encryption")

        // === Initialize Attestation Verifier ===
        strictAttestation := !config.IsDevelopment()
        attestation := service.NewAttestationVerifier(strictAttestation)
        slog.Info("Attestation verifier initialized", "strict", strictAttestation)

        // === Initialize Repositories ===
        deviceRepo := repository.NewDeviceRepository(db)
        txRepo := repository.NewTransactionRepository(db)
        limitsRepo := repository.NewLimitsMatrixRepository(db)
        disputeRepo := repository.NewDisputeRepository(db)
        pendingOpRepo := repository.NewPendingOperationRepository(db)
        switchRecordRepo := repository.NewSwitchRecordRepository(db)
        auditLogRepo := repository.NewAuditLogRepository(db)

        _ = auditLogRepo // Available for future audit middleware

        // === Initialize Adapter Registry ===
        adapterRegistry := adapter.NewRegistry()

        // Register wallet adapters based on configuration
        if cfg.Adapters.JEEP.BaseURL != "" {
                jeepAdapter := adapter.NewJEEPAdapter(adapter.JEEPConfig{
                        BaseURL:    cfg.Adapters.JEEP.BaseURL,
                        APIKey:     cfg.Adapters.JEEP.APIKey,
                        TimeoutSec: cfg.Adapters.JEEP.TimeoutSec,
                })
                adapterRegistry.Register(jeepAdapter)
                slog.Info("JEEP adapter registered")
        }
        if cfg.Adapters.WENET.BaseURL != "" {
                wenetAdapter := adapter.NewWENETAdapter(adapter.WENETConfig{
                        BaseURL:    cfg.Adapters.WENET.BaseURL,
                        APIKey:     cfg.Adapters.WENET.APIKey,
                        TimeoutSec: cfg.Adapters.WENET.TimeoutSec,
                })
                adapterRegistry.Register(wenetAdapter)
                slog.Info("WENET adapter registered")
        }
        if cfg.Adapters.WASEL.BaseURL != "" {
                waselAdapter := adapter.NewWASELAdapter(adapter.WASELConfig{
                        BaseURL:    cfg.Adapters.WASEL.BaseURL,
                        APIKey:     cfg.Adapters.WASEL.APIKey,
                        TimeoutSec: cfg.Adapters.WASEL.TimeoutSec,
                })
                adapterRegistry.Register(waselAdapter)
                slog.Info("WASEL adapter registered")
        }

        slog.Info("Adapter registry initialized", "adapters", adapterRegistry.ListAdapters())

        // === Initialize Services (with KMS + Attestation + Adapters) ===
        enrollService := service.NewEnrollmentService(deviceRepo, kms, attestation)
        deviceService := service.NewDeviceService(deviceRepo)
        txService := service.NewTransactionService(txRepo, deviceRepo)
        disputeService := service.NewDisputeService(disputeRepo, txRepo)
        limitsService := service.NewLimitsService(limitsRepo)

        // === Initialize SagaService ===
        sagaService := service.NewSagaService(
                adapterRegistry,
                txRepo,
                pendingOpRepo,
                deviceRepo,
        )
        slog.Info("SagaService initialized and wired")

        // === Initialize Handlers ===
        healthHandler := handler.NewHealthHandler(db, rdb)
        enrollHandler := handler.NewEnrollHandler(enrollService)
        txHandler := handler.NewTransactionHandler(txService, sagaService, switchRecordRepo)
        deviceHandler := handler.NewDeviceHandler(deviceService)
        disputeHandler := handler.NewDisputeHandler(disputeService)
        limitsHandler := handler.NewLimitsHandler(limitsService)

        // === Initialize Pipeline Middleware (v3.0: 7 layers) ===
        rateLimiter := middleware.NewRateLimiter(rdb,
                cfg.Limits.RateLimitPerDevice,
                cfg.Limits.RateLimitPerWallet,
                cfg.Limits.RateLimitPerIP,
        )
        requestLogger := middleware.NewRequestLogger()
        antiReplay := middleware.NewAntiReplay(rdb)
        limitsChecker := middleware.NewLimitsChecker(limitsRepo, txRepo, switchRecordRepo)
        sigVerifierA := middleware.NewSignatureVerifierA(switchRecordRepo, kms)
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

                AllowDevCORS: config.IsDevelopment(),
        }

        r := router.New(deps)

        // === Start Server (TLS/mTLS in production, HTTP in dev) ===
        srv := &http.Server{
                Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
                Handler:      r,
                ReadTimeout:  cfg.Server.ReadTimeout,
                WriteTimeout: cfg.Server.WriteTimeout,
        }

        // Configure mTLS if certificates are provided
        if cfg.Server.TLSCertPath != "" && cfg.Server.TLSKeyPath != "" {
                tlsConfig := &tls.Config{
                        MinVersion: tls.VersionTLS13,
                }

                // Load client CA for mTLS if provided
                mtlsCAPath := os.Getenv("MTLS_CLIENT_CA_PATH")
                if mtlsCAPath != "" {
                        caCert, err := os.ReadFile(mtlsCAPath)
                        if err != nil {
                                slog.Error("Failed to read mTLS CA certificate", "error", err)
                                os.Exit(1)
                        }
                        caCertPool := x509.NewCertPool()
                        caCertPool.AppendCertsFromPEM(caCert)
                        tlsConfig.ClientCAs = caCertPool
                        tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
                        slog.Info("mTLS enabled — client certificates required")
                }

                srv.TLSConfig = tlsConfig

                go func() {
                        slog.Info("Atheer Switch listening (TLS/mTLS)",
                                "port", cfg.Server.Port,
                                "version", "3.0.0",
                                "pipeline", "7 layers active",
                                "mtls", mtlsCAPath != "",
                        )
                        if err := srv.ListenAndServeTLS(cfg.Server.TLSCertPath, cfg.Server.TLSKeyPath); err != nil && err != http.ErrServerClosed {
                                slog.Error("Server failed", "error", err)
                                os.Exit(1)
                        }
                }()
        } else {
                if !config.IsDevelopment() {
                        slog.Warn("Running WITHOUT TLS — this is NOT safe for production!")
                }
                go func() {
                        slog.Info("Atheer Switch listening (HTTP — dev only)",
                                "port", cfg.Server.Port,
                                "version", "3.0.0",
                                "pipeline", "7 layers active",
                        )
                        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
                                slog.Error("Server failed", "error", err)
                                os.Exit(1)
                        }
                }()
        }

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
