// مستودع سجلات الدافعين/التجار — عمليات CRUD على جدول switch_records
// يُرجى الرجوع إلى SPEC §4
package db

import (
	"context"
	"fmt"

	"github.com/atheer/switch/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PayerRepo — واجهة عمليات سجلات الدافعين/التجار
type PayerRepo interface {
	// FindByPublicId — يبحث عن سجل الدافع بمعرّفه العام
	FindByPublicId(ctx context.Context, publicId string) (*model.SwitchRecord, error)

	// Create — ينشئ سجل دافع/تاجر جديد
	Create(ctx context.Context, record *model.SwitchRecord) error

	// UpdateCounter — يحدّث عداد المعاملات
	UpdateCounter(ctx context.Context, publicId string, newCounter int64) error

	// UpdateStatus — يحدّث حالة السجل (ACTIVE أو SUSPENDED)
	UpdateStatus(ctx context.Context, publicId string, status string) error

	// UpdatePayerLimit — يحدّث حد الدافع
	UpdatePayerLimit(ctx context.Context, publicId string, newLimit int64) error

	// Delete — يحذف سجل الدافع (إلغاء التسجيل)
	Delete(ctx context.Context, publicId string) error
}

// payerRepo — تنفيذ مستودع سجلات الدافعين
type payerRepo struct {
	pool *pgxpool.Pool
}

// NewPayerRepo — ينشئ نسخة مستودع سجلات الدافعين
func NewPayerRepo(pool *pgxpool.Pool) PayerRepo {
	return &payerRepo{pool: pool}
}

// FindByPublicId — يبحث عن سجل الدافع بمعرّفه العام
func (r *payerRepo) FindByPublicId(ctx context.Context, publicId string) (*model.SwitchRecord, error) {
	var record model.SwitchRecord
	err := r.pool.QueryRow(ctx, `
		SELECT id, public_id, wallet_id, device_id, seed_encrypted, seed_key_id,
		       counter, payer_limit, status, user_type, created_at, updated_at
		FROM switch_records
		WHERE public_id = $1
	`, publicId).Scan(
		&record.ID, &record.PublicId, &record.WalletId, &record.DeviceId,
		&record.SeedEncrypted, &record.SeedKeyID, &record.Counter,
		&record.PayerLimit, &record.Status, &record.UserType,
		&record.CreatedAt, &record.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // السجل غير موجود — لا خطأ
		}
		return nil, fmt.Errorf("مستودع الدافعين: بحث بمعرّف %s: %w", publicId, err)
	}
	return &record, nil
}

// Create — ينشئ سجل دافع/تاجر جديد
func (r *payerRepo) Create(ctx context.Context, record *model.SwitchRecord) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO switch_records (public_id, wallet_id, device_id, seed_encrypted, seed_key_id,
		                            counter, payer_limit, status, user_type)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, record.PublicId, record.WalletId, record.DeviceId, record.SeedEncrypted,
		record.SeedKeyID, record.Counter, record.PayerLimit, record.Status, record.UserType,
	)
	if err != nil {
		return fmt.Errorf("مستودع الدافعين: إنشاء سجل %s: %w", record.PublicId, err)
	}
	return nil
}

// UpdateCounter — يحدّث عداد المعاملات
func (r *payerRepo) UpdateCounter(ctx context.Context, publicId string, newCounter int64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE switch_records
		SET counter = $1, updated_at = NOW()
		WHERE public_id = $2
	`, newCounter, publicId)
	if err != nil {
		return fmt.Errorf("مستودع الدافعين: تحديث عداد %s: %w", publicId, err)
	}
	return nil
}

// UpdateStatus — يحدّث حالة السجل
func (r *payerRepo) UpdateStatus(ctx context.Context, publicId string, status string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE switch_records
		SET status = $1, updated_at = NOW()
		WHERE public_id = $2
	`, status, publicId)
	if err != nil {
		return fmt.Errorf("مستودع الدافعين: تحديث حالة %s: %w", publicId, err)
	}
	return nil
}

// UpdatePayerLimit — يحدّث حد الدافع
func (r *payerRepo) UpdatePayerLimit(ctx context.Context, publicId string, newLimit int64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE switch_records
		SET payer_limit = $1, updated_at = NOW()
		WHERE public_id = $2
	`, newLimit, publicId)
	if err != nil {
		return fmt.Errorf("مستودع الدافعين: تحديث حد %s: %w", publicId, err)
	}
	return nil
}

// Delete — يحذف سجل الدافع (إلغاء التسجيل)
func (r *payerRepo) Delete(ctx context.Context, publicId string) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM switch_records WHERE public_id = $1
	`, publicId)
	if err != nil {
		return fmt.Errorf("مستودع الدافعين: حذف %s: %w", publicId, err)
	}
	return nil
}
