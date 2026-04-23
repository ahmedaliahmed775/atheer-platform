// Modified for v3.0 Document Alignment
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/atheer-payment/atheer-platform/internal/model"
)

// TransactionRepository handles database operations for transactions
type TransactionRepository struct {
	db *pgxpool.Pool
}

// NewTransactionRepository creates a new TransactionRepository
func NewTransactionRepository(db *pgxpool.Pool) *TransactionRepository {
	return &TransactionRepository{db: db}
}

// Create inserts a new transaction record
func (r *TransactionRepository) Create(ctx context.Context, tx *model.Transaction) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO transactions (
			tx_id, payer_public_id, payer_user_id, payer_type,
			payee_id, payee_type, transaction_type,
			amount, currency, wallet_id, counter,
			status, error_code, error_message, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
		tx.TxID, tx.PayerPublicID, tx.PayerUserID, tx.PayerType,
		tx.PayeeID, tx.PayeeType, tx.TransactionType,
		tx.Amount, tx.Currency, tx.WalletID, tx.Counter,
		tx.Status, tx.ErrorCode, tx.ErrorMessage, tx.CreatedAt,
	)
	return err
}

// GetByTxID retrieves a transaction by its ID
func (r *TransactionRepository) GetByTxID(ctx context.Context, txID string) (*model.Transaction, error) {
	tx := &model.Transaction{}
	err := r.db.QueryRow(ctx, `
		SELECT id, tx_id, payer_public_id, payer_user_id, payer_type,
		       payee_id, payee_type, transaction_type,
		       amount, currency, wallet_id, counter,
		       status, error_code, error_message, created_at, completed_at
		FROM transactions WHERE tx_id = $1`, txID,
	).Scan(
		&tx.ID, &tx.TxID, &tx.PayerPublicID, &tx.PayerUserID, &tx.PayerType,
		&tx.PayeeID, &tx.PayeeType, &tx.TransactionType,
		&tx.Amount, &tx.Currency, &tx.WalletID, &tx.Counter,
		&tx.Status, &tx.ErrorCode, &tx.ErrorMessage,
		&tx.CreatedAt, &tx.CompletedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("transaction not found: %s", txID)
	}
	return tx, err
}

// UpdateStatus updates the status of a transaction
func (r *TransactionRepository) UpdateStatus(ctx context.Context, txID string, status model.TransactionStatus, errCode *string, errMsg *string) error {
	now := time.Now()
	_, err := r.db.Exec(ctx, `
		UPDATE transactions SET status = $1, error_code = $2, error_message = $3, completed_at = $4
		WHERE tx_id = $5`,
		status, errCode, errMsg, &now, txID,
	)
	return err
}

// GetDailyTotalByDevice returns the total amount spent today by a payer
func (r *TransactionRepository) GetDailyTotalByDevice(ctx context.Context, payerPublicID string) (decimal.Decimal, error) {
	var total decimal.Decimal
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0)
		FROM transactions
		WHERE payer_public_id = $1
		  AND status = 'COMPLETED'
		  AND created_at >= CURRENT_DATE`,
		payerPublicID,
	).Scan(&total)
	return total, err
}

// ListRecent returns recent transactions (for dashboard)
func (r *TransactionRepository) ListRecent(ctx context.Context, limit int) ([]*model.Transaction, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tx_id, payer_public_id, payer_user_id, payer_type,
		       payee_id, payee_type, transaction_type,
		       amount, currency, wallet_id, counter,
		       status, error_code, error_message, created_at, completed_at
		FROM transactions ORDER BY created_at DESC LIMIT $1`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []*model.Transaction
	for rows.Next() {
		tx := &model.Transaction{}
		if err := rows.Scan(
			&tx.ID, &tx.TxID, &tx.PayerPublicID, &tx.PayerUserID, &tx.PayerType,
			&tx.PayeeID, &tx.PayeeType, &tx.TransactionType,
			&tx.Amount, &tx.Currency, &tx.WalletID, &tx.Counter,
			&tx.Status, &tx.ErrorCode, &tx.ErrorMessage,
			&tx.CreatedAt, &tx.CompletedAt,
		); err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	return txs, nil
}
