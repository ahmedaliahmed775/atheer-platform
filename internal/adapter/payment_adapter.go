// Modified for v3.0 Document Alignment
// واجهة المحوّل حسب القسم 5 من الوثيقة
package adapter

import (
    "context"
    "github.com/atheer-payment/atheer-platform/internal/model"
    "github.com/shopspring/decimal"
)

type DebitResult struct {
    Success       bool
    TransactionID string
    NewBalance    decimal.Decimal
}

type CreditResult struct {
    Success       bool
    TransactionID string
    NewBalance    decimal.Decimal
}

type BalanceResult struct {
    Available decimal.Decimal
    Currency  string
}

type StatusResult struct {
    TxID   string
    Status string
}

// WalletAdapter — واجهة المحوّل حسب القسم 5 من الوثيقة
// كل محوّل مسؤول عن: بناء الطلب + تحليل الرد + توحيده إلى AtheerResult
type WalletAdapter interface {
    WalletID() string
    BuildRequest(dto model.TransactionDTO) (*WalletAPIRequest, error)
    ParseResponse(raw []byte) (*model.AtheerResult, error)
}

// SagaExecutor — واجهة داخلية للمحوّلات التي تدعم تنفيذ Saga
// لا تظهر في الوثيقة — طبقة تنفيذ مخفية
type SagaExecutor interface {
    Debit(ctx context.Context, walletID, accountID string, amount decimal.Decimal, txID string) (*DebitResult, error)
    Credit(ctx context.Context, walletID, accountID string, amount decimal.Decimal, txID string) (*CreditResult, error)
    ReverseDebit(ctx context.Context, walletID, accountID string, amount decimal.Decimal, txID string) error
    SendSMS(ctx context.Context, accountID string, message string) error
}

// Registry — سجل المحوّلات
type Registry struct {
    adapters map[string]WalletAdapter
}

func NewRegistry() *Registry {
    return &Registry{adapters: make(map[string]WalletAdapter)}
}

func (r *Registry) Register(adapter WalletAdapter) {
    r.adapters[adapter.WalletID()] = adapter
}

// GetAdapter — يختار المحوّل بناءً على WalletID (من SwitchRecord)
func (r *Registry) GetAdapter(walletID string) (WalletAdapter, bool) {
    a, ok := r.adapters[walletID]
    return a, ok
}

func (r *Registry) ListAdapters() []string {
    ids := make([]string, 0, len(r.adapters))
    for id := range r.adapters {
        ids = append(ids, id)
    }
    return ids
}

type WalletAPIRequest struct {
    URL     string
    Method  string
    Headers map[string]string
    Body    []byte
}
