// مستودع إعدادات المحافظ — عمليات CRUD على جدول wallet_configs
// يُرجى الرجوع إلى SPEC §4
package db

import (
	"context"
	"fmt"

	"github.com/atheer/switch/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WalletRepo — واجهة عمليات إعدادات المحافظ
type WalletRepo interface {
	// FindByWalletId — يبحث عن إعدادات محفظة بمعرّفها
	FindByWalletId(ctx context.Context, walletId string) (*model.WalletConfig, error)

	// List — يعرض كل إعدادات المحافظ
	List(ctx context.Context) ([]model.WalletConfig, error)

	// Create — ينشئ إعدادات محفظة جديدة
	Create(ctx context.Context, config *model.WalletConfig) error

	// Update — يحدّث إعدادات محفظة
	Update(ctx context.Context, config *model.WalletConfig) error
}

// walletRepo — تنفيذ مستودع إعدادات المحافظ
type walletRepo struct {
	pool *pgxpool.Pool
}

// NewWalletRepo — ينشئ نسخة مستودع إعدادات المحافظ
func NewWalletRepo(pool *pgxpool.Pool) WalletRepo {
	return &walletRepo{pool: pool}
}

// FindByWalletId — يبحث عن إعدادات محفظة بمعرّفها
func (r *walletRepo) FindByWalletId(ctx context.Context, walletId string) (*model.WalletConfig, error) {
	var cfg model.WalletConfig
	var apiKey, secret *string

	err := r.pool.QueryRow(ctx, `
		SELECT id, wallet_id, base_url, api_key, secret,
		       max_payer_limit, timeout_ms, max_retries, is_active,
		       created_at, updated_at
		FROM wallet_configs
		WHERE wallet_id = $1
	`, walletId).Scan(
		&cfg.ID, &cfg.WalletId, &cfg.BaseURL, &apiKey, &secret,
		&cfg.MaxPayerLimit, &cfg.TimeoutMs, &cfg.MaxRetries, &cfg.IsActive,
		&cfg.CreatedAt, &cfg.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // المحفظة غير موجودة
		}
		return nil, fmt.Errorf("مستودع المحافظ: بحث بمعرّف %s: %w", walletId, err)
	}

	if apiKey != nil {
		cfg.APIKey = *apiKey
	}
	if secret != nil {
		cfg.Secret = *secret
	}

	return &cfg, nil
}

// List — يعرض كل إعدادات المحافظ
func (r *walletRepo) List(ctx context.Context) ([]model.WalletConfig, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, wallet_id, base_url, api_key, secret,
		       max_payer_limit, timeout_ms, max_retries, is_active,
		       created_at, updated_at
		FROM wallet_configs
		ORDER BY wallet_id
	`)
	if err != nil {
		return nil, fmt.Errorf("مستودع المحافظ: قائمة المحافظ: %w", err)
	}
	defer rows.Close()

	var configs []model.WalletConfig
	for rows.Next() {
		var cfg model.WalletConfig
		var apiKey, secret *string

		if err := rows.Scan(
			&cfg.ID, &cfg.WalletId, &cfg.BaseURL, &apiKey, &secret,
			&cfg.MaxPayerLimit, &cfg.TimeoutMs, &cfg.MaxRetries, &cfg.IsActive,
			&cfg.CreatedAt, &cfg.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("مستودع المحافظ: قراءة صف: %w", err)
		}

		if apiKey != nil {
			cfg.APIKey = *apiKey
		}
		if secret != nil {
			cfg.Secret = *secret
		}

		configs = append(configs, cfg)
	}

	return configs, nil
}

// Create — ينشئ إعدادات محفظة جديدة
func (r *walletRepo) Create(ctx context.Context, config *model.WalletConfig) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO wallet_configs (wallet_id, base_url, api_key, secret,
		                            max_payer_limit, timeout_ms, max_retries, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, config.WalletId, config.BaseURL,
		nullIfEmpty(config.APIKey), nullIfEmpty(config.Secret),
		config.MaxPayerLimit, config.TimeoutMs, config.MaxRetries, config.IsActive,
	)
	if err != nil {
		return fmt.Errorf("مستودع المحافظ: إنشاء محفظة %s: %w", config.WalletId, err)
	}
	return nil
}

// Update — يحدّث إعدادات محفظة
func (r *walletRepo) Update(ctx context.Context, config *model.WalletConfig) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE wallet_configs
		SET base_url = $1, api_key = $2, secret = $3,
		    max_payer_limit = $4, timeout_ms = $5, max_retries = $6,
		    is_active = $7, updated_at = NOW()
		WHERE wallet_id = $8
	`, config.BaseURL,
		nullIfEmpty(config.APIKey), nullIfEmpty(config.Secret),
		config.MaxPayerLimit, config.TimeoutMs, config.MaxRetries,
		config.IsActive, config.WalletId,
	)
	if err != nil {
		return fmt.Errorf("مستودع المحافظ: تحديث محفظة %s: %w", config.WalletId, err)
	}
	return nil
}
