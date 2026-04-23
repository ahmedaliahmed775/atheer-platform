// Modified for v3.0 Document Alignment
// حسب القسم 2: TransactionType = PayerType + PayeeType
package middleware

import (
    "net/http"

    "github.com/atheer-payment/atheer-platform/internal/model"
    "github.com/atheer-payment/atheer-platform/pkg/response"
)

type TransactionTypeResolver struct{}

func NewTransactionTypeResolver() *TransactionTypeResolver {
    return &TransactionTypeResolver{}
}

func (tr *TransactionTypeResolver) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, hr *http.Request) {
        payerRecord := GetPayerRecord(hr.Context())
        payeeRecord := GetPayeeRecord(hr.Context())
        if payerRecord == nil || payeeRecord == nil {
            response.BadRequest(w, response.ErrInternalError, "Records not resolved")
            return
        }

        txType := model.DetermineTransactionType(payerRecord.UserType, payeeRecord.UserType)
        ctx := SetTransactionType(hr.Context(), txType)
        next.ServeHTTP(w, hr.WithContext(ctx))
    })
}
