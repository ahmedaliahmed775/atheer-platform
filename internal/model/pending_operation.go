package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PendingOpType represents the type of pending operation in a Saga
type PendingOpType string

const (
	PendingOpDebit    PendingOpType = "DEBIT"
	PendingOpCredit   PendingOpType = "CREDIT"
	PendingOpReversal PendingOpType = "REVERSAL"
)

// PendingOpStatus represents the status of a pending operation
type PendingOpStatus string

const (
	PendingOpStatusPending PendingOpStatus = "PENDING"
	PendingOpStatusDone    PendingOpStatus = "DONE"
	PendingOpStatusFailed  PendingOpStatus = "FAILED"
)

// PendingOperation represents a step in a Saga transaction
type PendingOperation struct {
	ID          uuid.UUID       `json:"id"          db:"id"`
	TxID        string          `json:"txId"        db:"tx_id"`
	OpType      PendingOpType   `json:"opType"      db:"op_type"`
	AdapterID   string          `json:"adapterId"   db:"adapter_id"`
	WalletID    string          `json:"walletId"    db:"wallet_id"`
	AccountID   string          `json:"accountId"   db:"account_id"`
	Amount      decimal.Decimal `json:"amount"      db:"amount"`
	Status      PendingOpStatus `json:"status"      db:"status"`
	RetryCount  int             `json:"retryCount"  db:"retry_count"`
	MaxRetries  int             `json:"maxRetries"  db:"max_retries"`
	NextRetryAt *time.Time      `json:"nextRetryAt" db:"next_retry_at"`
	Error       *string         `json:"error"       db:"error"`
	CreatedAt   time.Time       `json:"createdAt"   db:"created_at"`
	CompletedAt *time.Time      `json:"completedAt" db:"completed_at"`
}
