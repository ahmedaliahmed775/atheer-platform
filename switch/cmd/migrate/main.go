// أمر الترحيل — يشغّل ترحيلات قاعدة البيانات فقط
// يُستخدم: go run ./cmd/migrate -config config.yaml
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/atheer/switch/internal/config"
	"github.com/atheer/switch/internal/db"
	"github.com/atheer/switch/migrations"
)

func main() {
	// تحليل معاملات سطر الأوامر
	configPath := flag.String("config", "config.yaml", "مسار ملف الإعدادات")
	flag.Parse()

	// إعداد المُسجّل
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	slog.Info("بدء تشغيل الترحيلات")

	// تحميل الإعدادات
	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("فشل تحميل الإعدادات", "error", err)
		os.Exit(1)
	}

	// إنشاء سياق مع إشارة إيقاف
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// إنشاء تجمّع اتصالات قاعدة البيانات
	pool, err := db.NewPool(ctx, cfg.Database)
	if err != nil {
		slog.Error("فشل الاتصال بقاعدة البيانات", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	slog.Info("تم الاتصال بقاعدة البيانات", "host", cfg.Database.Host, "db", cfg.Database.Name)

	// تشغيل الترحيلات
	if err := db.Migrate(ctx, pool, migrations.FS); err != nil {
		slog.Error("فشل تشغيل الترحيلات", "error", err)
		os.Exit(1)
	}

	fmt.Println("✅ تم تشغيل جميع الترحيلات بنجاح")
}
