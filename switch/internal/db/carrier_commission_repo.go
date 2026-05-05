// مستودع إحصائيات العمولات — استعلامات تجميعية لعمولات شركة الاتصالات
package db

import (
	"context"
	"fmt"

	"github.com/atheer/switch/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CarrierCommissionRepo — واجهة مستودع إحصائيات العمولات
type CarrierCommissionRepo interface {
	// GetCommissionStats — يُرجع إحصائيات العمولات لكل محفظة في فترة معيّنة
	GetCommissionStats(ctx context.Context, commissionRate int64, fromDate, toDate string) (*model.CarrierCommissionSummary, error)
}

// carrierCommissionRepo — تنفيذ مستودع إحصائيات العمولات
type carrierCommissionRepo struct {
	pool *pgxpool.Pool
}

// NewCarrierCommissionRepo — ينشئ نسخة مستودع إحصائيات العمولات
func NewCarrierCommissionRepo(pool *pgxpool.Pool) CarrierCommissionRepo {
	return &carrierCommissionRepo{pool: pool}
}

// GetCommissionStats — يُرجع إحصائيات العمولات لكل محفظة في فترة معيّنة
// fromDate و toDate بصيغة "2024-01-15" أو "2024-01"
func (r *carrierCommissionRepo) GetCommissionStats(ctx context.Context, commissionRate int64, fromDate, toDate string) (*model.CarrierCommissionSummary, error) {
	// بناء شروط التاريخ
	var dateCondition string
	var args []interface{}
	argIdx := 1

	if fromDate != "" && toDate != "" {
		dateCondition = fmt.Sprintf("AND created_at >= $%d::date AND created_at < ($%d::date + interval '1 day')", argIdx, argIdx+1)
		args = append(args, fromDate, toDate)
		argIdx += 2
	} else if fromDate != "" {
		dateCondition = fmt.Sprintf("AND created_at >= $%d::date", argIdx)
		args = append(args, fromDate)
		argIdx++
	} else if toDate != "" {
		dateCondition = fmt.Sprintf("AND created_at < ($%d::date + interval '1 day')", argIdx)
		args = append(args, toDate)
		argIdx++
	}

	// استعلام تجميعي لكل محفظة — فقط معاملات شبكة الاتصالات
	query := fmt.Sprintf(`
		SELECT
			payer_wallet_id,
			COUNT(*) AS total_tx_count,
			COALESCE(SUM(CASE WHEN status = 'SUCCESS' THEN amount ELSE 0 END), 0) AS total_amount,
			COUNT(*) FILTER (WHERE status = 'SUCCESS') AS success_count,
			COUNT(*) FILTER (WHERE status = 'FAILED') AS failed_count
		FROM transactions
		WHERE connection_source = 'carrier'
		%s
		GROUP BY payer_wallet_id
		ORDER BY total_amount DESC
	`, dateCondition)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("مستودع العمولات: استعلام الإحصائيات: %w", err)
	}
	defer rows.Close()

	var wallets []model.CarrierCommissionStats
	var totalTxCount int
	var totalAmount int64
	var totalDue int64

	for rows.Next() {
		var stat model.CarrierCommissionStats
		if err := rows.Scan(
			&stat.WalletId,
			&stat.TotalTxCount,
			&stat.TotalAmount,
			&stat.SuccessCount,
			&stat.FailedCount,
		); err != nil {
			return nil, fmt.Errorf("مستودع العمولات: قراءة صف: %w", err)
		}

		// حساب العمولة: (المبلغ × نسبة العمولة) / 1000
		stat.CommissionRate = commissionRate
		stat.CommissionDue = (stat.TotalAmount * commissionRate) / 1000

		totalTxCount += stat.TotalTxCount
		totalAmount += stat.TotalAmount
		totalDue += stat.CommissionDue

		wallets = append(wallets, stat)
	}

	if wallets == nil {
		wallets = []model.CarrierCommissionStats{}
	}

	return &model.CarrierCommissionSummary{
		Wallets:      wallets,
		TotalTxCount: totalTxCount,
		TotalAmount:  totalAmount,
		TotalDue:     totalDue,
	}, nil
}
