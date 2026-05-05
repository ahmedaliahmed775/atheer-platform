// مستودع تقارير التسوية — عمليات CRUD على جدول reconciliation_reports
package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/atheer/switch/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ReconRepo — واجهة عمليات تقارير التسوية
type ReconRepo interface {
	// Save — يحفظ تقرير تسوية جديد
	Save(ctx context.Context, report *model.ReconciliationReport) error

	// FindByDateAndWallet — يبحث عن تقرير بتاريخه ومعرّف المحفظة
	FindByDateAndWallet(ctx context.Context, reportDate, walletId string) (*model.ReconciliationReport, error)

	// List — يعرض قائمة تقارير التسوية مع تصفية
	List(ctx context.Context, walletId string, page, pageSize int) ([]model.ReconciliationReport, int, error)

	// Update — يحدّث تقرير تسوية
	Update(ctx context.Context, report *model.ReconciliationReport) error
}

// reconRepo — تنفيذ مستودع تقارير التسوية
type reconRepo struct {
	pool *pgxpool.Pool
}

// NewReconRepo — ينشئ نسخة مستودع تقارير التسوية
func NewReconRepo(pool *pgxpool.Pool) ReconRepo {
	return &reconRepo{pool: pool}
}

// Save — يحفظ تقرير تسوية جديد
func (r *reconRepo) Save(ctx context.Context, report *model.ReconciliationReport) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO reconciliation_reports (
			report_date, wallet_id, total_tx_count, total_amount,
			success_count, failed_count, disputed_count, status, notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`,
		report.ReportDate, report.WalletId,
		report.TotalTxCount, report.TotalAmount,
		report.SuccessCount, report.FailedCount, report.DisputedCount,
		report.Status, nullIfEmpty(report.Notes),
	)
	if err != nil {
		return fmt.Errorf("مستودع التسوية: حفظ التقرير: %w", err)
	}
	return nil
}

// FindByDateAndWallet — يبحث عن تقرير بتاريخه ومعرّف المحفظة
func (r *reconRepo) FindByDateAndWallet(ctx context.Context, reportDate, walletId string) (*model.ReconciliationReport, error) {
	var report model.ReconciliationReport
	var notes *string
	var reportDateDB time.Time // pgx يتطلب time.Time لمسح نوع date

	err := r.pool.QueryRow(ctx, `
		SELECT id, report_date, wallet_id, total_tx_count, total_amount,
		       success_count, failed_count, disputed_count, status, notes,
		       created_at, updated_at
		FROM reconciliation_reports
		WHERE report_date = $1 AND wallet_id = $2
	`, reportDate, walletId).Scan(
		&report.ID, &reportDateDB, &report.WalletId,
		&report.TotalTxCount, &report.TotalAmount,
		&report.SuccessCount, &report.FailedCount, &report.DisputedCount,
		&report.Status, &notes, &report.CreatedAt, &report.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("مستودع التسوية: بحث بتاريخ %s ومحفظة %s: %w", reportDate, walletId, err)
	}

	report.ReportDate = reportDateDB.Format("2006-01-02")
	if notes != nil {
		report.Notes = *notes
	}
	return &report, nil
}

// List — يعرض قائمة تقارير التسوية مع تصفية
func (r *reconRepo) List(ctx context.Context, walletId string, page, pageSize int) ([]model.ReconciliationReport, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if walletId != "" {
		conditions = append(conditions, fmt.Sprintf("wallet_id = $%d", argIdx))
		args = append(args, walletId)
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// حساب العدد الإجمالي
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM reconciliation_reports %s", whereClause)
	var totalCount int
	if err := r.pool.QueryRow(ctx, countSQL, args...).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("مستودع التسوية: عدّ التقارير: %w", err)
	}

	// حساب الإزاحة
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	querySQL := fmt.Sprintf(`
		SELECT id, report_date, wallet_id, total_tx_count, total_amount,
		       success_count, failed_count, disputed_count, status, notes,
		       created_at, updated_at
		FROM reconciliation_reports %s
		ORDER BY report_date DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	args = append(args, pageSize, offset)

	rows, err := r.pool.Query(ctx, querySQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("مستودع التسوية: قائمة التقارير: %w", err)
	}
	defer rows.Close()

	var reports []model.ReconciliationReport
	for rows.Next() {
		var report model.ReconciliationReport
		var notes *string
		var reportDateDB time.Time // pgx يتطلب time.Time لمسح نوع date

		if err := rows.Scan(
			&report.ID, &reportDateDB, &report.WalletId,
			&report.TotalTxCount, &report.TotalAmount,
			&report.SuccessCount, &report.FailedCount, &report.DisputedCount,
			&report.Status, &notes, &report.CreatedAt, &report.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("مستودع التسوية: قراءة صف: %w", err)
		}

		report.ReportDate = reportDateDB.Format("2006-01-02")
		if notes != nil {
			report.Notes = *notes
		}
		reports = append(reports, report)
	}

	return reports, totalCount, nil
}

// Update — يحدّث تقرير تسوية
func (r *reconRepo) Update(ctx context.Context, report *model.ReconciliationReport) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE reconciliation_reports
		SET total_tx_count = $1, total_amount = $2,
		    success_count = $3, failed_count = $4, disputed_count = $5,
		    status = $6, notes = $7, updated_at = NOW()
		WHERE id = $8
	`,
		report.TotalTxCount, report.TotalAmount,
		report.SuccessCount, report.FailedCount, report.DisputedCount,
		report.Status, nullIfEmpty(report.Notes), report.ID,
	)
	if err != nil {
		return fmt.Errorf("مستودع التسوية: تحديث التقرير %d: %w", report.ID, err)
	}
	return nil
}
