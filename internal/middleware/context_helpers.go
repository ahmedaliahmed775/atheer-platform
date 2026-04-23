// Modified for v3.0 Document Alignment
// Context helpers for the v3.0 pipeline middleware
package middleware

import (
    "context"

    "github.com/atheer-payment/atheer-platform/internal/model"
)

type contextKey string

const (
    payerPacketKey  contextKey = "payerPacket"
    payerRecordKey  contextKey = "payerRecord"
    payeeRecordKey  contextKey = "payeeRecord"
    txTypeKey       contextKey = "transactionType"
)

// GetPayerPacket extracts the parsed PayerTlvPacket from context
func GetPayerPacket(ctx context.Context) *model.PayerTlvPacket {
    if val, ok := ctx.Value(payerPacketKey).(*model.PayerTlvPacket); ok {
        return val
    }
    return nil
}

// SetPayerPacket stores the PayerTlvPacket in context
func SetPayerPacket(ctx context.Context, packet *model.PayerTlvPacket) context.Context {
    return context.WithValue(ctx, payerPacketKey, packet)
}

// GetPayerRecord extracts the payer SwitchRecord from context
func GetPayerRecord(ctx context.Context) *model.SwitchRecord {
    if val, ok := ctx.Value(payerRecordKey).(*model.SwitchRecord); ok {
        return val
    }
    return nil
}

// SetPayerRecord stores the payer SwitchRecord in context
func SetPayerRecord(ctx context.Context, record *model.SwitchRecord) context.Context {
    return context.WithValue(ctx, payerRecordKey, record)
}

// GetPayeeRecord extracts the payee SwitchRecord from context
func GetPayeeRecord(ctx context.Context) *model.SwitchRecord {
    if val, ok := ctx.Value(payeeRecordKey).(*model.SwitchRecord); ok {
        return val
    }
    return nil
}

// SetPayeeRecord stores the payee SwitchRecord in context
func SetPayeeRecord(ctx context.Context, record *model.SwitchRecord) context.Context {
    return context.WithValue(ctx, payeeRecordKey, record)
}

// GetTransactionType extracts the resolved TransactionType from context
func GetTransactionType(ctx context.Context) model.TransactionType {
    if val, ok := ctx.Value(txTypeKey).(model.TransactionType); ok {
        return val
    }
    return ""
}

// SetTransactionType stores the resolved TransactionType in context
func SetTransactionType(ctx context.Context, txType model.TransactionType) context.Context {
    return context.WithValue(ctx, txTypeKey, txType)
}
