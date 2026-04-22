package adapter

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/atheer-payment/atheer-platform/internal/model"
)

// PaymentAdapter defines the interface for wallet payment adapters
// Each wallet (JEEP, WENET, WASEL) implements this interface
type PaymentAdapter interface {
	// ID returns the adapter identifier (e.g., "JEEP", "WENET", "WASEL")
	ID() string

	// Debit deducts amount from the payer's wallet
	Debit(ctx context.Context, walletID, accountID string, amount decimal.Decimal, txID string) (*DebitResult, error)

	// Credit adds amount to the receiver's wallet
	Credit(ctx context.Context, walletID, accountID string, amount decimal.Decimal, txID string) (*CreditResult, error)

	// ReverseDebit reverses a previous debit (Saga compensation)
	ReverseDebit(ctx context.Context, walletID, accountID string, amount decimal.Decimal, originalTxID string) error

	// CheckBalance checks the balance of a wallet account
	CheckBalance(ctx context.Context, walletID, accountID string) (*BalanceResult, error)

	// GetTransactionStatus gets the status of a transaction on the wallet server
	GetTransactionStatus(ctx context.Context, txID string) (*StatusResult, error)

	// SendSMS sends a notification SMS to the user
	SendSMS(ctx context.Context, phone, message string) error

	// GetLimits queries the wallet server for real-time limits
	GetLimits(ctx context.Context, walletID, accountID string, opType model.OperationType) (*model.LimitsResult, error)
}

// DebitResult contains the result of a debit operation
type DebitResult struct {
	Success       bool            `json:"success"`
	TransactionID string          `json:"transactionId"`
	NewBalance    decimal.Decimal `json:"newBalance"`
}

// CreditResult contains the result of a credit operation
type CreditResult struct {
	Success       bool            `json:"success"`
	TransactionID string          `json:"transactionId"`
	NewBalance    decimal.Decimal `json:"newBalance"`
}

// BalanceResult contains the balance of a wallet account
type BalanceResult struct {
	Available decimal.Decimal `json:"available"`
	Currency  string          `json:"currency"`
}

// StatusResult contains the status of a transaction on the wallet server
type StatusResult struct {
	TxID   string `json:"txId"`
	Status string `json:"status"`
}

// Registry manages all registered payment adapters
type Registry struct {
	adapters map[string]PaymentAdapter
}

// NewRegistry creates a new adapter registry
func NewRegistry() *Registry {
	return &Registry{
		adapters: make(map[string]PaymentAdapter),
	}
}

// Register adds an adapter to the registry
func (r *Registry) Register(adapter PaymentAdapter) {
	r.adapters[adapter.ID()] = adapter
}

// GetAdapter returns the adapter for a given wallet ID
func (r *Registry) GetAdapter(walletID string) (PaymentAdapter, bool) {
	adapter, ok := r.adapters[walletID]
	return adapter, ok
}

// ListAdapters returns all registered adapter IDs
func (r *Registry) ListAdapters() []string {
	ids := make([]string, 0, len(r.adapters))
	for id := range r.adapters {
		ids = append(ids, id)
	}
	return ids
}
