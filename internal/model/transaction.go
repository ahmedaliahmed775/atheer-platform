// Modified for v3.0 Document Alignment
package model

import (
    "time"
    "github.com/google/uuid"
)

// TransactionType — يُحدَّد تلقائياً في السويتش
// حسب القسم 2: TransactionType = PayerType + PayeeType
type TransactionType string

const (
    TxP2P TransactionType = "P2P" // عميل → عميل
    TxP2M TransactionType = "P2M" // عميل → تاجر
    TxM2P TransactionType = "M2P" // تاجر → عميل
    TxM2M TransactionType = "M2M" // تاجر → تاجر
)

// DetermineTransactionType — حسب القسم 2
func DetermineTransactionType(payerType, payeeType UserType) TransactionType {
    switch {
    case payerType == UserTypePersonal && payeeType == UserTypePersonal:
        return TxP2P
    case payerType == UserTypePersonal && payeeType == UserTypeMerchant:
        return TxP2M
    case payerType == UserTypeMerchant && payeeType == UserTypePersonal:
        return TxM2P
    case payerType == UserTypeMerchant && payeeType == UserTypeMerchant:
        return TxM2M
    default:
        return TxP2P
    }
}

// TransactionStatus
type TransactionStatus string

const (
    TxStatusPending   TransactionStatus = "PENDING"
    TxStatusCompleted TransactionStatus = "COMPLETED"
    TxStatusFailed    TransactionStatus = "FAILED"
    TxStatusReversed  TransactionStatus = "REVERSED"
    TxStatusDisputed  TransactionStatus = "DISPUTED"
)

// Transaction — سجل المعاملة
type Transaction struct {
    ID              uuid.UUID         `json:"id"              db:"id"`
    TxID            string            `json:"txId"            db:"tx_id"`
    PayerPublicID   string            `json:"payerPublicId"   db:"payer_public_id"`
    PayerUserID     string            `json:"payerUserId"     db:"payer_user_id"`
    PayerType       UserType          `json:"payerType"       db:"payer_type"`
    PayeeID         string            `json:"payeeId"         db:"payee_id"`
    PayeeType       UserType          `json:"payeeType"       db:"payee_type"`
    TransactionType TransactionType   `json:"transactionType" db:"transaction_type"`
    Amount          int64             `json:"amount"          db:"amount"`
    Currency        string            `json:"currency"        db:"currency"`
    WalletID        string            `json:"walletId"        db:"wallet_id"`
    Counter         uint64            `json:"counter"         db:"counter"`
    Status          TransactionStatus `json:"status"          db:"status"`
    ErrorCode       *string           `json:"errorCode"       db:"error_code"`
    ErrorMessage    *string           `json:"errorMessage"    db:"error_message"`
    CreatedAt       time.Time         `json:"createdAt"       db:"created_at"`
    CompletedAt     *time.Time        `json:"completedAt"     db:"completed_at"`
}

// TransactionDTO — البيانات الداخلية بعد التحقق
// حسب القسم 4 — الخطوة 5
type TransactionDTO struct {
    PayerUserID     string
    PayerType       UserType
    PayeeID         string
    PayeeType       UserType
    TransactionType TransactionType
    Amount          int64
    Currency        string
    WalletID        string
    Counter         uint64
    Timestamp       int64
}

// PayerTlvPacket — الحزمة المستلمة من الطرف A عبر الطرف B
// حسب القسم 4 — الخطوة 4
// PayeeType تم حذفه — السويتش يحدد UserType تلقائياً من SwitchRecord
// Currency أُضيف لحماية هجمات تبديل العملة
type PayerTlvPacket struct {
    PublicID   string `json:"publicId"`
    Amount     int64  `json:"amount"`
    ReceiverID string `json:"receiverId"`
    Currency   string `json:"currency"`
    Counter    uint64 `json:"counter"`
    HMAC       []byte `json:"hmac"`
}

// AtheerResult — النتيجة الموحدة من المحوّل
type AtheerResult struct {
    Success       bool   `json:"success"`
    TransactionID string `json:"transactionId"`
    ErrorCode     string `json:"errorCode,omitempty"`
    ErrorMessage  string `json:"errorMessage,omitempty"`
}

// SwitchErrorCode — كودات الأخطاء حسب القسم 5
type SwitchErrorCode string

const (
    ErrHMACMismatch     SwitchErrorCode = "ERR_HMAC_MISMATCH"
    ErrCounterReplay    SwitchErrorCode = "ERR_COUNTER_REPLAY"
    ErrPayeeTypeMismatch SwitchErrorCode = "ERR_PAYEE_TYPE_MISMATCH"
    ErrSpendLimit       SwitchErrorCode = "ERR_SPEND_LIMIT"
    ErrBalance          SwitchErrorCode = "ERR_BALANCE"
    ErrWalletDown       SwitchErrorCode = "ERR_WALLET_DOWN"
    ErrUnknownWallet    SwitchErrorCode = "ERR_UNKNOWN_WALLET"
)
