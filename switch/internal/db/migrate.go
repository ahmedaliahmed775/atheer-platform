// تشغيل ترحيلات قاعدة البيانات من ملفات SQL
// يقرأ ملفات الترحيل بالترتيب وينفّذها ضد PostgreSQL
package db

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Migrate — يشغّل ترحيلات قاعدة البيانات من نظام الملفات المدمج
// يبحث عن ملفات .up.sql وينفّذها بالترتيب
func Migrate(ctx context.Context, pool *pgxpool.Pool, migrationsFS embed.FS) error {
	// إنشاء جدول تتبع الترحيلات إن لم يكن موجوداً
	if err := ensureMigrationsTable(ctx, pool); err != nil {
		return fmt.Errorf("الترحيلات: إنشاء جدول التتبع: %w", err)
	}

	// قراءة ملفات الترحيل
	entries, err := fs.ReadDir(migrationsFS, ".")
	if err != nil {
		return fmt.Errorf("الترحيلات: قراءة المجلد: %w", err)
	}

	// فلترة وترتيب ملفات الترحيل الصاعدة
	var upFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".up.sql") {
			upFiles = append(upFiles, entry.Name())
		}
	}
	sort.Strings(upFiles)

	// تنفيذ كل ترحيلة لم تُنفَّذ بعد
	for _, filename := range upFiles {
		version := extractVersion(filename)
		if version == 0 {
			slog.Warn("الترحيلات: تجاهل ملف بدون رقم إصدار", "filename", filename)
			continue
		}

		applied, err := isMigrationApplied(ctx, pool, version)
		if err != nil {
			return fmt.Errorf("الترحيلات: التحقق من الإصدار %d: %w", version, err)
		}
		if applied {
			slog.Debug("الترحيلات: ترحيلة مُطبّقة مسبقاً", "version", version)
			continue
		}

		content, err := fs.ReadFile(migrationsFS, filename)
		if err != nil {
			return fmt.Errorf("الترحيلات: قراءة الملف %s: %w", filename, err)
		}

		slog.Info("الترحيلات: تنفيذ ترحيلة", "version", version, "filename", filename)

		if _, err := pool.Exec(ctx, string(content)); err != nil {
			return fmt.Errorf("الترحيلات: تنفيذ %s: %w", filename, err)
		}

		if err := recordMigration(ctx, pool, version, filename); err != nil {
			return fmt.Errorf("الترحيلات: تسجيل الإصدار %d: %w", version, err)
		}

		slog.Info("الترحيلات: تمت الترحيلة بنجاح", "version", version)
	}

	return nil
}

// ensureMigrationsTable — ينشئ جدول schema_migrations إن لم يكن موجوداً
func ensureMigrationsTable(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    BIGINT PRIMARY KEY,
			filename   VARCHAR(255) NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("إنشاء جدول الترحيلات: %w", err)
	}
	return nil
}

// isMigrationApplied — يتحقق هل ترحيلة معيّنة نُفّذت مسبقاً
func isMigrationApplied(ctx context.Context, pool *pgxpool.Pool, version int) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)",
		version,
	).Scan(&exists)
	return exists, err
}

// recordMigration — يسجّل ترحيلة نُفّذت في جدول التتبع
func recordMigration(ctx context.Context, pool *pgxpool.Pool, version int, filename string) error {
	_, err := pool.Exec(ctx,
		"INSERT INTO schema_migrations (version, filename) VALUES ($1, $2)",
		version, filename,
	)
	return err
}

// extractVersion — يستخرج رقم الإصدار من اسم ملف الترحيلة
// مثال: "001_create_switch_records.up.sql" → 1
func extractVersion(filename string) int {
	parts := strings.SplitN(filename, "_", 2)
	if len(parts) < 2 {
		return 0
	}
	version, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0
	}
	return version
}
