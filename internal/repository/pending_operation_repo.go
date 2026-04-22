package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/atheer-payment/atheer-platform/internal/model"
)

// PendingOperationRepository handles Saga pending operations
type PendingOperationRepository struct {
	db *pgxpool.Pool
}

func NewPendingOperationRepository(db *pgxpool.Pool) *PendingOperationRepository {
	return &PendingOperationRepository{db: db}
}

func (r *PendingOperationRepository) Create(ctx context.Context, op *model.PendingOperation) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO pending_operations (id, tx_id, op_type, adapter_id, wallet_id, account_id, amount, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		op.ID, op.TxID, op.OpType, op.AdapterID, op.WalletID, op.AccountID, op.Amount, op.Status,
	)
	return err
}

func (r *PendingOperationRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status model.PendingOpStatus, errMsg string) error {
	var errPtr *string
	if errMsg != "" {
		errPtr = &errMsg
	}
	_, err := r.db.Exec(ctx, `
		UPDATE pending_operations SET status = $1, error = $2, completed_at = NOW()
		WHERE id = $3`,
		status, errPtr, id,
	)
	return err
}

func (r *PendingOperationRepository) GetByTxID(ctx context.Context, txID string) ([]*model.PendingOperation, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tx_id, op_type, adapter_id, wallet_id, account_id, amount, status,
		       retry_count, max_retries, next_retry_at, error, created_at, completed_at
		FROM pending_operations WHERE tx_id = $1 ORDER BY created_at`, txID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ops []*model.PendingOperation
	for rows.Next() {
		op := &model.PendingOperation{}
		if err := rows.Scan(
			&op.ID, &op.TxID, &op.OpType, &op.AdapterID,
			&op.WalletID, &op.AccountID, &op.Amount, &op.Status,
			&op.RetryCount, &op.MaxRetries, &op.NextRetryAt,
			&op.Error, &op.CreatedAt, &op.CompletedAt,
		); err != nil {
			return nil, err
		}
		ops = append(ops, op)
	}
	return ops, nil
}

func (r *PendingOperationRepository) GetPending(ctx context.Context) ([]*model.PendingOperation, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tx_id, op_type, adapter_id, wallet_id, account_id, amount, status,
		       retry_count, max_retries, next_retry_at, error, created_at, completed_at
		FROM pending_operations
		WHERE status = 'PENDING' AND (next_retry_at IS NULL OR next_retry_at <= NOW())
		ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ops []*model.PendingOperation
	for rows.Next() {
		op := &model.PendingOperation{}
		if err := rows.Scan(
			&op.ID, &op.TxID, &op.OpType, &op.AdapterID,
			&op.WalletID, &op.AccountID, &op.Amount, &op.Status,
			&op.RetryCount, &op.MaxRetries, &op.NextRetryAt,
			&op.Error, &op.CreatedAt, &op.CompletedAt,
		); err != nil {
			return nil, err
		}
		ops = append(ops, op)
	}
	return ops, nil
}
