package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// LimitsMatrix defines transaction limits per wallet/operation/currency/tier
type LimitsMatrix struct {
	ID            uuid.UUID       `json:"id"            db:"id"`
	WalletID      string          `json:"walletId"      db:"wallet_id"`
	TransactionType TransactionType `json:"transactionType" db:"transaction_type"`
	Currency      string          `json:"currency"      db:"currency"`
	MaxTxAmount   decimal.Decimal `json:"maxTxAmount"   db:"max_tx_amount"`
	MaxDaily      decimal.Decimal `json:"maxDaily"      db:"max_daily"`
	MaxMonthly    *decimal.Decimal `json:"maxMonthly"   db:"max_monthly"`
	Tier          string          `json:"tier"          db:"tier"`
	IsActive      bool            `json:"isActive"      db:"is_active"`
	UpdatedAt     time.Time       `json:"updatedAt"     db:"updated_at"`
}

// LimitsResult is returned by Adapter.GetLimits()
type LimitsResult struct {
	MaxTxAmount    decimal.Decimal `json:"maxTxAmount"`
	RemainingDaily decimal.Decimal `json:"remainingDaily"`
	MaxDaily       decimal.Decimal `json:"maxDaily"`
}
