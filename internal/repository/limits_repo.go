package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/atheer-payment/atheer-platform/internal/model"
)

// LimitsMatrixRepository handles database operations for limits
type LimitsMatrixRepository struct {
	db *pgxpool.Pool
}

func NewLimitsMatrixRepository(db *pgxpool.Pool) *LimitsMatrixRepository {
	return &LimitsMatrixRepository{db: db}
}

// GetLimits retrieves limits for a specific wallet/operation/currency
func (r *LimitsMatrixRepository) GetLimits(ctx context.Context, walletID, opType, currency, tier string) (*model.LimitsMatrix, error) {
	if tier == "" {
		tier = "basic"
	}
	lm := &model.LimitsMatrix{}
	err := r.db.QueryRow(ctx, `
		SELECT id, wallet_id, transaction_type, currency, max_tx_amount,
		       max_daily, max_monthly, tier, is_active, updated_at
		FROM limits_matrix
		WHERE wallet_id = $1 AND transaction_type = $2 AND currency = $3 AND tier = $4 AND is_active = true`,
		walletID, opType, currency, tier,
	).Scan(
		&lm.ID, &lm.WalletID, &lm.TransactionType, &lm.Currency,
		&lm.MaxTxAmount, &lm.MaxDaily, &lm.MaxMonthly,
		&lm.Tier, &lm.IsActive, &lm.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("limits not found: wallet=%s op=%s currency=%s", walletID, opType, currency)
	}
	return lm, err
}

// ListAll returns all active limits (for dashboard)
func (r *LimitsMatrixRepository) ListAll(ctx context.Context) ([]*model.LimitsMatrix, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, wallet_id, transaction_type, currency, max_tx_amount,
		       max_daily, max_monthly, tier, is_active, updated_at
		FROM limits_matrix WHERE is_active = true
		ORDER BY wallet_id, transaction_type`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var limits []*model.LimitsMatrix
	for rows.Next() {
		lm := &model.LimitsMatrix{}
		if err := rows.Scan(
			&lm.ID, &lm.WalletID, &lm.TransactionType, &lm.Currency,
			&lm.MaxTxAmount, &lm.MaxDaily, &lm.MaxMonthly,
			&lm.Tier, &lm.IsActive, &lm.UpdatedAt,
		); err != nil {
			return nil, err
		}
		limits = append(limits, lm)
	}
	return limits, nil
}

// Upsert creates or updates a limits entry
func (r *LimitsMatrixRepository) Upsert(ctx context.Context, lm *model.LimitsMatrix) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO limits_matrix (wallet_id, transaction_type, currency, max_tx_amount, max_daily, max_monthly, tier)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (wallet_id, transaction_type, currency, tier)
		DO UPDATE SET max_tx_amount = $4, max_daily = $5, max_monthly = $6, updated_at = NOW()`,
		lm.WalletID, lm.TransactionType, lm.Currency,
		lm.MaxTxAmount, lm.MaxDaily, lm.MaxMonthly, lm.Tier,
	)
	return err
}

// DisputeRepository handles database operations for disputes
type DisputeRepository struct {
	db *pgxpool.Pool
}

func NewDisputeRepository(db *pgxpool.Pool) *DisputeRepository {
	return &DisputeRepository{db: db}
}

func (r *DisputeRepository) Create(ctx context.Context, d *model.Dispute) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO disputes (tx_id, reason, status, opened_by)
		VALUES ($1, $2, $3, $4)`,
		d.TxID, d.Reason, d.Status, d.OpenedBy,
	)
	return err
}

func (r *DisputeRepository) ListOpen(ctx context.Context) ([]*model.Dispute, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tx_id, reason, status, opened_by, resolved_by, resolution, created_at, resolved_at
		FROM disputes ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var disputes []*model.Dispute
	for rows.Next() {
		d := &model.Dispute{}
		if err := rows.Scan(
			&d.ID, &d.TxID, &d.Reason, &d.Status, &d.OpenedBy,
			&d.ResolvedBy, &d.Resolution, &d.CreatedAt, &d.ResolvedAt,
		); err != nil {
			return nil, err
		}
		disputes = append(disputes, d)
	}
	return disputes, nil
}

// AuditLogRepository handles audit log entries
type AuditLogRepository struct {
	db *pgxpool.Pool
}

func NewAuditLogRepository(db *pgxpool.Pool) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

func (r *AuditLogRepository) Log(ctx context.Context, action, actor, resourceType string, resourceID *string, details map[string]interface{}) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO audit_logs (action, actor, resource_type, resource_id, details)
		VALUES ($1, $2, $3, $4, $5)`,
		action, actor, resourceType, resourceID, details,
	)
	return err
}
