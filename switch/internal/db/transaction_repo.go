// مستودع المعاملات — عمليات CRUD على جدول transactions
// يُرجى الرجوع إلى SPEC §4
package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/atheer/switch/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TransactionRepo — واجهة عمليات المعاملات
type TransactionRepo interface {
	// Save — يحفظ سجل معاملة جديد
	Save(ctx context.Context, tx *model.Transaction) error

	// FindByID — يبحث عن معاملة بمعرّفها
	FindByID(ctx context.Context, transactionId string) (*model.Transaction, error)

	// List — يعرض قائمة المعاملات مع تصفية وصفحات
	// يُرجع المعاملات والعدد الإجمالي
	List(ctx context.Context, filters model.TransactionFilters, page, pageSize int) ([]model.Transaction, int, error)

	// GetDailyTotal — يحسب إجمالي معاملات الدافع في يوم معيّن
	GetDailyTotal(ctx context.Context, publicId string, date string) (int64, error)

	// GetMonthlyTotal — يحسب إجمالي معاملات الدافع في شهر معيّن
	GetMonthlyTotal(ctx context.Context, publicId string, month string) (int64, error)
}

// transactionRepo — تنفيذ مستودع المعاملات
type transactionRepo struct {
	pool *pgxpool.Pool
}

// NewTransactionRepo — ينشئ نسخة مستودع المعاملات
func NewTransactionRepo(pool *pgxpool.Pool) TransactionRepo {
	return &transactionRepo{pool: pool}
}

// Save — يحفظ سجل معاملة جديد
func (r *transactionRepo) Save(ctx context.Context, tx *model.Transaction) error {
	// تحديد مصدر الاتصال — الافتراضي "internet"
	connSource := tx.ConnectionSource
	if connSource == "" {
		connSource = model.SourceInternet
	}

	_, err := r.pool.Exec(ctx, `
		INSERT INTO transactions (
			transaction_id, payer_public_id, merchant_id,
			payer_wallet_id, merchant_wallet_id,
			amount, currency, counter, status,
			error_code, duration_ms, debit_ref, credit_ref,
			connection_source
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`,
		tx.TransactionId, tx.PayerPublicId, tx.MerchantId,
		tx.PayerWalletId, tx.MerchantWalletId,
		tx.Amount, tx.Currency, tx.Counter, tx.Status,
		nullIfEmpty(tx.ErrorCode), tx.DurationMs,
		nullIfEmpty(tx.DebitRef), nullIfEmpty(tx.CreditRef),
		connSource,
	)
	if err != nil {
		return fmt.Errorf("مستودع المعاملات: حفظ المعاملة %s: %w", tx.TransactionId, err)
	}
	return nil
}

// FindByID — يبحث عن معاملة بمعرّفها
func (r *transactionRepo) FindByID(ctx context.Context, transactionId string) (*model.Transaction, error) {
	var tx model.Transaction
	var errorCode, debitRef, creditRef *string

	err := r.pool.QueryRow(ctx, `
		SELECT id, transaction_id, payer_public_id, merchant_id,
		       payer_wallet_id, merchant_wallet_id,
		       amount, currency, counter, status,
		       error_code, duration_ms, debit_ref, credit_ref,
		       connection_source, created_at
		FROM transactions
		WHERE transaction_id = $1
	`, transactionId).Scan(
		&tx.ID, &tx.TransactionId, &tx.PayerPublicId, &tx.MerchantId,
		&tx.PayerWalletId, &tx.MerchantWalletId,
		&tx.Amount, &tx.Currency, &tx.Counter, &tx.Status,
		&errorCode, &tx.DurationMs, &debitRef, &creditRef,
		&tx.ConnectionSource, &tx.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // المعاملة غير موجودة
		}
		return nil, fmt.Errorf("مستودع المعاملات: بحث بمعرّف %s: %w", transactionId, err)
	}

	// تحويل المؤشرات إلى سلاسل
	if errorCode != nil {
		tx.ErrorCode = *errorCode
	}
	if debitRef != nil {
		tx.DebitRef = *debitRef
	}
	if creditRef != nil {
		tx.CreditRef = *creditRef
	}

	return &tx, nil
}

// List — يعرض قائمة المعاملات مع تصفية وصفحات
func (r *transactionRepo) List(ctx context.Context, filters model.TransactionFilters, page, pageSize int) ([]model.Transaction, int, error) {
	// بناء شروط التصفية
	var conditions []string
	var args []interface{}
	argIdx := 1

	if filters.PayerPublicId != "" {
		conditions = append(conditions, fmt.Sprintf("payer_public_id = $%d", argIdx))
		args = append(args, filters.PayerPublicId)
		argIdx++
	}
	if filters.MerchantId != "" {
		conditions = append(conditions, fmt.Sprintf("merchant_id = $%d", argIdx))
		args = append(args, filters.MerchantId)
		argIdx++
	}
	if filters.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, filters.Status)
		argIdx++
	}
	if filters.WalletId != "" {
		conditions = append(conditions, fmt.Sprintf("(payer_wallet_id = $%d OR merchant_wallet_id = $%d)", argIdx, argIdx))
		args = append(args, filters.WalletId)
		argIdx++
	}
	if filters.ConnectionSource != "" {
		conditions = append(conditions, fmt.Sprintf("connection_source = $%d", argIdx))
		args = append(args, filters.ConnectionSource)
		argIdx++
	}
	if !filters.FromDate.IsZero() {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, filters.FromDate)
		argIdx++
	}
	if !filters.ToDate.IsZero() {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIdx))
		args = append(args, filters.ToDate)
		argIdx++
	}

	// بناء جملة WHERE
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// حساب العدد الإجمالي
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM transactions %s", whereClause)
	var totalCount int
	if err := r.pool.QueryRow(ctx, countSQL, args...).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("مستودع المعاملات: عدّ المعاملات: %w", err)
	}

	// حساب الإزاحة
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	// جلب الصفحة المطلوبة
	querySQL := fmt.Sprintf(`
		SELECT id, transaction_id, payer_public_id, merchant_id,
		       payer_wallet_id, merchant_wallet_id,
		       amount, currency, counter, status,
		       error_code, duration_ms, debit_ref, credit_ref,
		       connection_source, created_at
		FROM transactions %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	args = append(args, pageSize, offset)

	rows, err := r.pool.Query(ctx, querySQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("مستودع المعاملات: قائمة المعاملات: %w", err)
	}
	defer rows.Close()

	var transactions []model.Transaction
	for rows.Next() {
		var tx model.Transaction
		var errorCode, debitRef, creditRef *string

		if err := rows.Scan(
			&tx.ID, &tx.TransactionId, &tx.PayerPublicId, &tx.MerchantId,
			&tx.PayerWalletId, &tx.MerchantWalletId,
			&tx.Amount, &tx.Currency, &tx.Counter, &tx.Status,
			&errorCode, &tx.DurationMs, &debitRef, &creditRef,
			&tx.ConnectionSource, &tx.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("مستودع المعاملات: قراءة صف: %w", err)
		}

		if errorCode != nil {
			tx.ErrorCode = *errorCode
		}
		if debitRef != nil {
			tx.DebitRef = *debitRef
		}
		if creditRef != nil {
			tx.CreditRef = *creditRef
		}

		transactions = append(transactions, tx)
	}

	return transactions, totalCount, nil
}

// GetDailyTotal — يحسب إجمالي معاملات الدافع الناجحة في يوم معيّن
// المعامل date بصيغة "2024-01-15"
func (r *transactionRepo) GetDailyTotal(ctx context.Context, publicId string, date string) (int64, error) {
	var total int64
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0)
		FROM transactions
		WHERE payer_public_id = $1
		  AND status = 'SUCCESS'
		  AND DATE(created_at) = $2::date
	`, publicId, date).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("مستودع المعاملات: إجمالي يومي %s: %w", publicId, err)
	}
	return total, nil
}

// GetMonthlyTotal — يحسب إجمالي معاملات الدافع الناجحة في شهر معيّن
// المعامل month بصيغة "2024-01"
func (r *transactionRepo) GetMonthlyTotal(ctx context.Context, publicId string, month string) (int64, error) {
	var total int64
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0)
		FROM transactions
		WHERE payer_public_id = $1
		  AND status = 'SUCCESS'
		  AND TO_CHAR(created_at, 'YYYY-MM') = $2
	`, publicId, month).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("مستودع المعاملات: إجمالي شهري %s: %w", publicId, err)
	}
	return total, nil
}

// nullIfEmpty — يحوّل السلسلة الفارغة إلى nil لقاعدة البيانات
func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
