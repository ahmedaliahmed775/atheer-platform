// إعداد تجمّع اتصالات قاعدة البيانات باستخدام pgxpool
// يُرجى الرجوع إلى SPEC §4 و skills/go-switch.md
package db

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/atheer/switch/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool — ينشئ تجمّع اتصالات PostgreSQL باستخدام pgxpool
// يضبط الحد الأقصى والأدنى للاتصالات ويتحقق من الاتصال بـ Ping
func NewPool(ctx context.Context, cfg config.DatabaseConfig) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.Name, cfg.User, cfg.Password,
	)

	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("قاعدة البيانات: تحليل DSN: %w", err)
	}

	// ضبط حدود الاتصالات
	poolCfg.MaxConns = int32(cfg.MaxConns)
	poolCfg.MinConns = 2
	poolCfg.HealthCheckPeriod = 30 * time.Second
	poolCfg.MaxConnIdleTime = 5 * time.Minute
	poolCfg.MaxConnLifetime = 1 * time.Hour

	slog.Info("قاعدة البيانات: إنشاء تجمّع الاتصالات",
		"host", cfg.Host,
		"port", cfg.Port,
		"dbname", cfg.Name,
		"max_conns", poolCfg.MaxConns,
		"min_conns", poolCfg.MinConns,
	)

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("قاعدة البيانات: إنشاء التجمع: %w", err)
	}

	// التحقق من الاتصال
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("قاعدة البيانات: فشل الاتصال: %w", err)
	}

	slog.Info("قاعدة البيانات: تم الاتصال بنجاح")
	return pool, nil
}
