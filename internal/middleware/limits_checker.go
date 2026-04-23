// Modified for v3.0 Document Alignment
package middleware

import (
	"log/slog"
	"net/http"

	"github.com/shopspring/decimal"

	"github.com/atheer-payment/atheer-platform/internal/model"
	"github.com/atheer-payment/atheer-platform/internal/repository"
	"github.com/atheer-payment/atheer-platform/pkg/response"
)

type LimitsChecker struct {
	limitsRepo *repository.LimitsMatrixRepository
	txRepo     *repository.TransactionRepository
	recordRepo *repository.SwitchRecordRepository
}

func NewLimitsChecker(limitsRepo *repository.LimitsMatrixRepository, txRepo *repository.TransactionRepository, recordRepo *repository.SwitchRecordRepository) *LimitsChecker {
	return &LimitsChecker{limitsRepo: limitsRepo, txRepo: txRepo, recordRepo: recordRepo}
}

func (lc *LimitsChecker) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		packet := GetPayerPacket(r.Context())
		if packet == nil {
			response.BadRequest(w, response.ErrInternalError, "Request not parsed")
			return
		}

		amount := decimal.NewFromInt(packet.Amount)

		payerRecord, err := lc.recordRepo.GetByPublicID(r.Context(), packet.PublicID)
		if err != nil || payerRecord == nil {
			response.Forbidden(w, "ERR_UNKNOWN_WALLET", "Payer not found")
			return
		}

		// Use a basic P2P transaction type for limits check default
		matrixLimits, err := lc.limitsRepo.GetLimits(r.Context(),
			payerRecord.WalletID, "P2P", "SAR", "basic")
		if err != nil {
			matrixLimits = &model.LimitsMatrix{
				MaxTxAmount: decimal.NewFromInt(999999999),
				MaxDaily:    decimal.NewFromInt(999999999),
			}
		}

		if amount.GreaterThan(matrixLimits.MaxTxAmount) {
			response.BadRequest(w, "ERR_SPEND_LIMIT", "Amount exceeds per-transaction limit")
			return
		}

		dailyTotal, err := lc.txRepo.GetDailyTotalByDevice(r.Context(), packet.PublicID)
		if err != nil {
			slog.Error("Failed to get daily total", "error", err)
		} else {
			if dailyTotal.Add(amount).GreaterThan(matrixLimits.MaxDaily) {
				response.BadRequest(w, "ERR_SPEND_LIMIT", "Amount exceeds daily limit")
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
