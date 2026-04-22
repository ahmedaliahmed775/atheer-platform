package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// OperationType represents the type of payment operation
type OperationType string

const (
	OpP2PSame  OperationType = "P2P_SAME"
	OpP2MSame  OperationType = "P2M_SAME"
	OpP2MCross OperationType = "P2M_CROSS"
	OpP2PCross OperationType = "P2P_CROSS"
)

// TransactionStatus represents the status of a transaction
type TransactionStatus string

const (
	TxStatusPending   TransactionStatus = "PENDING"
	TxStatusCompleted TransactionStatus = "COMPLETED"
	TxStatusFailed    TransactionStatus = "FAILED"
	TxStatusReversed  TransactionStatus = "REVERSED"
	TxStatusDisputed  TransactionStatus = "DISPUTED"
)

// Channel represents the communication channel used
type Channel string

const (
	ChannelAPN      Channel = "APN"
	ChannelInternet Channel = "INTERNET"
)

// Transaction represents a completed or pending transaction
type Transaction struct {
	ID             uuid.UUID         `json:"id"              db:"id"`
	TxID           string            `json:"txId"            db:"tx_id"`
	Nonce          string            `json:"nonce"           db:"nonce"`
	SideAWalletID  string            `json:"sideAWalletId"   db:"side_a_wallet_id"`
	SideADeviceID  string            `json:"sideADeviceId"   db:"side_a_device_id"`
	SideAAccountID string            `json:"sideAAccountId"  db:"side_a_account_id"`
	SideBWalletID  string            `json:"sideBWalletId"   db:"side_b_wallet_id"`
	SideBDeviceID  string            `json:"sideBDeviceId"   db:"side_b_device_id"`
	SideBAccountID string            `json:"sideBAccountId"  db:"side_b_account_id"`
	MerchantID     *string           `json:"merchantId"      db:"merchant_id"`
	OperationType  OperationType     `json:"operationType"   db:"operation_type"`
	Currency       string            `json:"currency"        db:"currency"`
	Amount         decimal.Decimal   `json:"amount"          db:"amount"`
	ChannelUsed    Channel           `json:"channel"         db:"channel"`
	Status         TransactionStatus `json:"status"          db:"status"`
	ErrorCode      *string           `json:"errorCode"       db:"error_code"`
	ErrorMessage   *string           `json:"errorMessage"    db:"error_message"`
	SideACtr       int64             `json:"sideACtr"        db:"side_a_ctr"`
	CreatedAt      time.Time         `json:"createdAt"       db:"created_at"`
	CompletedAt    *time.Time        `json:"completedAt"     db:"completed_at"`
}

// SideAPayload matches SDK's NfcPayloadBuilder.SideAPayload
type SideAPayload struct {
	WalletID             string   `json:"walletId"             validate:"required"`
	DeviceID             string   `json:"deviceId"             validate:"required"`
	Ctr                  int64    `json:"ctr"                  validate:"gte=0"`
	OperationType        string   `json:"operationType"        validate:"required"`
	Currency             string   `json:"currency"             validate:"required,len=3"`
	Amount               *float64 `json:"amount"`
	Nonce                string   `json:"nonce"                validate:"required,uuid"`
	Timestamp            int64    `json:"timestamp"            validate:"required"`
	Signature            string   `json:"signature"            validate:"required"`
	AttestationSignature string   `json:"attestationSignature" validate:"required"`
}

// SideBPayload matches SDK's NfcPayloadBuilder.SideBPayload
type SideBPayload struct {
	WalletID      string   `json:"walletId"      validate:"required"`
	DeviceID      string   `json:"deviceId"      validate:"required"`
	MerchantID    *string  `json:"merchantId"`
	OperationType string   `json:"operationType" validate:"required"`
	Currency      string   `json:"currency"      validate:"required,len=3"`
	Amount        *float64 `json:"amount"`
	AccountID     string   `json:"accountId"     validate:"required"`
	Timestamp     int64    `json:"timestamp"     validate:"required"`
	Signature     string   `json:"signature"     validate:"required"`
}

// CombinedRequest matches SDK's NfcPayloadBuilder.CombinedRequest
type CombinedRequest struct {
	SideA SideAPayload `json:"sideA" validate:"required"`
	SideB SideBPayload `json:"sideB" validate:"required"`
}
