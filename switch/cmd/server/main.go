// نقطة الدخول الرئيسية لسويتش Atheer
// يُوصّل كل المكونات: إعدادات، قاعدة بيانات، مستودعات، KMS، محوّلات، خدمات، معالجات، وسطاء
// يُشغّل خادم HTTP مع إيقاف متأنّي (graceful shutdown)
package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/atheer/switch/internal/adapter"
	"github.com/atheer/switch/internal/adapter/jawali"
	"github.com/atheer/switch/internal/api"
	"github.com/atheer/switch/internal/api/admin"
	"github.com/atheer/switch/internal/config"
	"github.com/atheer/switch/internal/crypto"
	"github.com/atheer/switch/internal/db"
	"github.com/atheer/switch/internal/execute"
	"github.com/atheer/switch/internal/gate"
	"github.com/atheer/switch/internal/middleware"
	"github.com/atheer/switch/internal/model"
	"github.com/atheer/switch/internal/notify"
	"github.com/atheer/switch/internal/verify"
	"github.com/atheer/switch/migrations"
)

func main() {
	// ── تحليل معاملات سطر الأوامر ──
	configPath := flag.String("config", "config.yaml", "مسار ملف الإعدادات")
	flag.Parse()

	// ── إعداد المُسجّل المنظّم ──
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	slog.Info("سويتش Atheer — بدء التشغيل")

	// ── 1. تحميل الإعدادات ──
	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("فشل تحميل الإعدادات", "error", err)
		os.Exit(1)
	}
	slog.Info("تم تحميل الإعدادات", "port", cfg.Server.Port)

	// ── 2. إنشاء سياق مع إشارة إيقاف ──
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// ── 3. إنشاء تجمّع اتصالات قاعدة البيانات ──
	pool, err := db.NewPool(ctx, cfg.Database)
	if err != nil {
		slog.Error("فشل الاتصال بقاعدة البيانات", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	slog.Info("تم الاتصال بقاعدة البيانات", "host", cfg.Database.Host, "db", cfg.Database.Name)

	// ── 4. تشغيل الترحيلات ──
	if err := db.Migrate(ctx, pool, migrations.FS); err != nil {
		slog.Error("فشل تشغيل الترحيلات", "error", err)
		os.Exit(1)
	}
	slog.Info("تم تشغيل الترحيلات بنجاح")

	// ── 5. إنشاء المستودعات ──
	payerRepo := db.NewPayerRepo(pool)
	txRepo := db.NewTransactionRepo(pool)
	walletRepo := db.NewWalletRepo(pool)
	adminRepo := db.NewAdminRepo(pool)
	reconRepo := db.NewReconRepo(pool)
	commissionRepo := db.NewCarrierCommissionRepo(pool)

	// ── 6. إنشاء نظام إدارة المفاتيح (KMS) ──
	masterKeyBytes, err := hex.DecodeString(cfg.KMS.MasterKey)
	if err != nil {
		slog.Error("فشل فك تشفير المفتاح الرئيسي — تأكد من أنه بصيغة hex", "error", err)
		os.Exit(1)
	}
	if len(masterKeyBytes) != 32 {
		slog.Error("المفتاح الرئيسي يجب أن يكون 32 بايت (64 حرف hex)", "len", len(masterKeyBytes))
		os.Exit(1)
	}
	kmsInstance, err := crypto.NewLocalKMS(masterKeyBytes)
	if err != nil {
		slog.Error("فشل إنشاء KMS", "error", err)
		os.Exit(1)
	}
	// تصفير المفتاح الرئيسي من الذاكرة بعد الاستخدام
	defer crypto.Zeroize(masterKeyBytes)
	slog.Info("تم إنشاء KMS", "provider", cfg.KMS.Provider)

	// ── 7. إنشاء سجل المحوّلات وتسجيل محوّل جوالي ──
	registry := adapter.NewAdapterRegistry()

	// تحميل إعدادات المحافظ من قاعدة البيانات وتسجيلها
	walletConfigs, err := walletRepo.List(ctx)
	if err != nil {
		slog.Error("فشل تحميل إعدادات المحافظ", "error", err)
		os.Exit(1)
	}

	for _, wc := range walletConfigs {
		if !wc.IsActive {
			slog.Info("تخطي محفظة غير مفعّلة", "walletId", wc.WalletId)
			continue
		}

		switch wc.WalletId {
		case "jawali":
			jawaliAdapter := jawali.NewJawaliAdapter(jawali.ClientConfig{
				BaseURL:    wc.BaseURL,
				APIKey:     wc.APIKey,
				Secret:     wc.Secret,
				TimeoutMs:  wc.TimeoutMs,
				MaxRetries: wc.MaxRetries,
			})
			registry.Register("jawali", jawaliAdapter)
			slog.Info("تم تسجيل محوّل جوالي", "baseURL", wc.BaseURL)
		default:
			slog.Warn("محفظة غير مدعومة بعد", "walletId", wc.WalletId)
		}
	}

	slog.Info("تم تسجيل المحوّلات", "count", len(registry.List()), "wallets", registry.List())

	// ── 8. إنشاء محوّل التوجيه ──
	// محوّل التوجيه يُنفّذ واجهة WalletAdapter ويُوجّه الاستدعاءات للمحوّل المناسب
	dispatchAdapter := adapter.NewDispatchingAdapter(registry)

	// ── 9. إنشاء مُرسل الإشعارات ──
	notifier := notify.NewTelegramNotifier(
		cfg.Notifications.Telegram.BotToken,
		cfg.Notifications.Telegram.ChatID,
		cfg.Notifications.Telegram.Enabled,
	)
	if cfg.Notifications.Telegram.Enabled {
		slog.Info("إشعارات تيليجرام مفعّلة")
	} else {
		slog.Info("إشعارات تيليجرام معطّلة")
	}

	// ── 10. إنشاء الخدمات ──

	// خدمة البوابة (GATE)
	gateService := gate.NewGateService(payerRepo)

	// فاحص الحدود
	limitsChecker := verify.NewLimitsChecker(txRepo, verify.LimitsConfig{
		DailyLimit:   cfg.Security.DailyLimit,
		MonthlyLimit: cfg.Security.MonthlyLimit,
	})

	// خدمة التحقق (VERIFY)
	// محوّل التوجيه يُنفّذ واجهة MerchantVerifier لأنه يملك VerifyAccessToken
	verifyService := verify.NewVerifyService(
		kmsInstance,
		dispatchAdapter, // يُنفّذ MerchantVerifier
		limitsChecker,
		cfg.Security.TimestampTolerance,
		cfg.Security.LookAheadWindow,
	)

	// خدمة التنفيذ (EXECUTE)
	executeService := execute.NewExecuteService(
		dispatchAdapter, // يُنفّذ WalletAdapter
		payerRepo,
		txRepo,
	)

	// ── 11. إنشاء المعالجات ──
	enrollHandler := api.NewEnrollHandler(payerRepo, walletRepo, kmsInstance)
	transactionHandler := api.NewTransactionHandler(gateService, verifyService, executeService)
	syncHandler := api.NewSyncHandler(payerRepo)
	payerLimitHandler := api.NewPayerLimitHandler(payerRepo, walletRepo)
	unenrollHandler := api.NewUnenrollHandler(payerRepo)
	healthHandler := api.NewHealthHandler(pool, cfg.Carrier.Enabled)

	// ── 11b. إنشاء معالجات الإدارة ──
	// تحليل مدة صلاحية JWT
	jwtExpiry, err := config.ParseDuration(cfg.Security.JWTExpiry)
	if err != nil {
		slog.Error("فشل تحليل مدة صلاحية JWT", "error", err)
		os.Exit(1)
	}

	authHandler := admin.NewAuthHandler(adminRepo, cfg.Security.JWTSecret, jwtExpiry)
	adminTxHandler := admin.NewAdminTransactionsHandler(txRepo)
	adminUsersHandler := admin.NewAdminUsersHandler(payerRepo, walletRepo)
	adminWalletsHandler := admin.NewAdminWalletsHandler(walletRepo, registry)
	adminAnalyticsHandler := admin.NewAdminAnalyticsHandler(txRepo)
	adminHealthHandler := admin.NewAdminHealthHandler(pool, registry)
	adminReconHandler := admin.NewAdminReconHandler(reconRepo, txRepo, walletRepo)
	adminAdminsHandler := admin.NewAdminAdminsHandler(adminRepo)
	adminCommissionHandler := admin.NewAdminCommissionHandler(commissionRepo, cfg.Carrier.CommissionRate)
	terminalHandler := admin.NewTerminalHandler(cfg.Security.JWTSecret)

	// ضبط وقت بدء التشغيل لحساب مدة التشغيل في فحص الصحة
	admin.SetStartTime(time.Now())

	// ── 12. إعداد الوسطاء ──
	apiKeyMiddleware := middleware.APIKeyMiddleware(walletRepo)
	jwtAuthMiddleware := middleware.JWTAuthMiddleware(cfg.Security.JWTSecret)
	corsMiddleware := middleware.CORSMiddleware(middleware.DefaultCORSConfig())
	loggingMiddleware := middleware.LoggingMiddleware
	requestIDMiddleware := middleware.RequestIDMiddleware

	// ── 13. إعداد المُوجّه (Router) ──
	// مسارات API العامة (تحتاج مفتاح API)
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("POST /api/v1/enroll", enrollHandler.Handle)
	apiMux.HandleFunc("POST /api/v1/transaction", transactionHandler.Handle)
	apiMux.HandleFunc("POST /api/v1/sync", syncHandler.Handle)
	apiMux.HandleFunc("POST /api/v1/payer-limit", payerLimitHandler.Handle)
	apiMux.HandleFunc("POST /api/v1/unenroll", unenrollHandler.Handle)

	// مسارات عامة (لا تحتاج مفتاح API)
	publicMux := http.NewServeMux()
	publicMux.HandleFunc("GET /health", healthHandler.Handle)

	// ── 13b. مسارات الإدارة (تحتاج JWT + دور) ──
	adminMux := http.NewServeMux()

	// مسارات المصادقة — لا تحتاج JWT مسبق (تُولّد الرمز)
	adminMux.HandleFunc("POST /admin/v1/auth/login", authHandler.HandleLogin)
	adminMux.HandleFunc("POST /admin/v1/auth/refresh", authHandler.HandleRefresh)
	adminMux.HandleFunc("POST /admin/v1/auth/logout", authHandler.HandleLogout)

	// مسارات المعاملات — تحتاج دور VIEWER على الأقل
	adminMux.HandleFunc("GET /admin/v1/transactions", adminTxHandler.HandleList)
	adminMux.HandleFunc("GET /admin/v1/transactions/{id}", adminTxHandler.HandleGetByID)

	// مسارات المستخدمين — تحتاج دور ADMIN على الأقل
	adminMux.HandleFunc("GET /admin/v1/users", adminUsersHandler.HandleList)
	adminMux.HandleFunc("PATCH /admin/v1/users/{id}/status", adminUsersHandler.HandleUpdateStatus)
	adminMux.HandleFunc("PATCH /admin/v1/users/{id}/limit", adminUsersHandler.HandleUpdatePayerLimit)

	// مسارات المحافظ — تحتاج دور ADMIN على الأقل
	adminMux.HandleFunc("GET /admin/v1/wallets", adminWalletsHandler.HandleList)
	adminMux.HandleFunc("POST /admin/v1/wallets", adminWalletsHandler.HandleCreate)
	adminMux.HandleFunc("PUT /admin/v1/wallets/{id}", adminWalletsHandler.HandleUpdate)
	adminMux.HandleFunc("PATCH /admin/v1/wallets/{id}", adminWalletsHandler.HandlePatch)
	adminMux.HandleFunc("POST /admin/v1/wallets/{id}/test", adminWalletsHandler.HandleTest)

	// مسارات التحليلات — تحتاج دور VIEWER على الأقل
	adminMux.HandleFunc("GET /admin/v1/analytics/summary", adminAnalyticsHandler.HandleSummary)
	adminMux.HandleFunc("GET /admin/v1/analytics/volume", adminAnalyticsHandler.HandleVolume)
	adminMux.HandleFunc("GET /admin/v1/analytics/errors", adminAnalyticsHandler.HandleErrors)
	adminMux.HandleFunc("GET /admin/v1/analytics/latency", adminAnalyticsHandler.HandleLatency)

	// مسارات فحص الصحة — تحتاج دور VIEWER على الأقل
	adminMux.HandleFunc("GET /admin/v1/health/adapters", adminHealthHandler.HandleAdapters)
	adminMux.HandleFunc("GET /admin/v1/health/system", adminHealthHandler.HandleSystem)

	// مسارات حسابات الإدارة — تحتاج دور ADMIN على الأقل (إنشاء/تعديل: SUPER_ADMIN فقط)
	adminMux.HandleFunc("GET /admin/v1/admins", adminAdminsHandler.HandleList)
	adminMux.HandleFunc("POST /admin/v1/admins", adminAdminsHandler.HandleCreate)
	adminMux.HandleFunc("PATCH /admin/v1/admins/{id}", adminAdminsHandler.HandlePatch)

	// مسارات التسوية — تحتاج دور ADMIN على الأقل
	adminMux.HandleFunc("POST /admin/v1/reconciliation/run", adminReconHandler.HandleRun)
	adminMux.HandleFunc("GET /admin/v1/reconciliation/reports", adminReconHandler.HandleListReports)

	// مسارات العمولات — تحتاج دور ADMIN على الأقل
	adminMux.HandleFunc("GET /admin/v1/commission/stats", adminCommissionHandler.HandleStats)

	// تجميع المُوجّهات — المسارات العامة أولاً ثم API ثم الإدارة
	rootMux := http.NewServeMux()
	rootMux.Handle("GET /health", publicMux)
	rootMux.Handle("/api/", apiKeyMiddleware(apiMux))           // مفتاح API لمسارات API فقط
	rootMux.Handle("/admin/", jwtAuthMiddleware(adminMux))       // JWT لمسارات الإدارة
	// مسار الطرفية — WebSocket يتحقق من JWT بنفسه (من معامل الاستعلام ?token=)
	// لا يمر عبر وسيط JWT لأن المتصفح لا يمكنه إرسال رأس Authorization مع WebSocket
	// يجب تسجيله بعد /admin/ لأن Go ServeMux يُفضّل المسارات الأكثر تحديداً
	rootMux.HandleFunc("GET /admin/v1/terminal", terminalHandler.HandleTerminal)

	// تطبيق الوسطاء العامة: CORS ← معرّف الطلب ← التسجيل ← المُوجّه
	var handler http.Handler = rootMux
	handler = corsMiddleware(handler)
	handler = loggingMiddleware(handler)
	handler = requestIDMiddleware(handler)

	// ── 14. إنشاء خادم HTTP العام (إنترنت) ──
	// الوسيط يُعطّي كل الطلبات بمصدر "internet"
	internetHandler := middleware.ConnectionSourceMiddleware(model.SourceInternet)(handler)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      internetHandler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  60 * time.Second,
	}

	// ── 14b. إنشاء خادم HTTP لشبكة الاتصالات (carrier) ──
	// نفس المسارات لكن مع وسيط مصدر الاتصال "carrier"
	var carrierSrv *http.Server
	if cfg.Carrier.Enabled {
		carrierHandler := middleware.ConnectionSourceMiddleware(model.SourceCarrier)(handler)
		carrierSrv = &http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.Carrier.Port),
			Handler:      carrierHandler,
			ReadTimeout:  cfg.Server.ReadTimeout,
			WriteTimeout: cfg.Server.WriteTimeout,
			IdleTimeout:  60 * time.Second,
		}
	}

	// ── 15. تشغيل الخوادم في goroutines ──
	go func() {
		slog.Info("خادم HTTP العام يستمع", "port", cfg.Server.Port, "source", "internet")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("فشل خادم HTTP العام", "error", err)
			os.Exit(1)
		}
	}()

	if carrierSrv != nil {
		go func() {
			slog.Info("خادم HTTP لشبكة الاتصالات يستمع", "port", cfg.Carrier.Port, "source", "carrier")
			if err := carrierSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				slog.Error("فشل خادم HTTP لشبكة الاتصالات", "error", err)
				os.Exit(1)
			}
		}()
	}

	// ── 16. إرسال تنبيه بدء التشغيل ──
	alertMsg := fmt.Sprintf("🚀 سويتش Atheer بدأ التشغيل على المنفذ %d", cfg.Server.Port)
	if cfg.Carrier.Enabled {
		alertMsg += fmt.Sprintf(" (شبكة الاتصالات: المنفذ %d)", cfg.Carrier.Port)
	}
	_ = notifier.SendAlert(context.Background(),
		notify.AlertLevelInfo,
		notify.EventAdapterRecovered,
		alertMsg,
	)

	slog.Info("سويتش Atheer جاهز لاستقبال الطلبات",
		"port", cfg.Server.Port,
		"carrierEnabled", cfg.Carrier.Enabled,
		"carrierPort", cfg.Carrier.Port,
	)

	// ── 17. انتظار إشارة الإيقاف ──
	<-ctx.Done()
	slog.Info("تم استلام إشارة إيقاف — بدء الإيقاف المتأنّي")

	// ── 18. إيقاف متأنّي (graceful shutdown) ──
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("فشل الإيقاف المتأنّي للخادم العام", "error", err)
	}

	if carrierSrv != nil {
		if err := carrierSrv.Shutdown(shutdownCtx); err != nil {
			slog.Error("فشل الإيقاف المتأنّي لخادم الاتصالات", "error", err)
		}
	}

	// إرسال تنبيه إيقاف
	_ = notifier.SendAlert(context.Background(),
		notify.AlertLevelWarning,
		notify.EventAdapterDown,
		"⛔ سويتش Atheer تم إيقافه",
	)

	slog.Info("تم إيقاف سويتش Atheer بنجاح")
}
