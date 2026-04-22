package middleware

import (
	"log/slog"
	"net/http"

	"github.com/shopspring/decimal"

	"github.com/atheer-payment/atheer-platform/internal/model"
	"github.com/atheer-payment/atheer-platform/internal/repository"
	"github.com/atheer-payment/atheer-platform/pkg/response"
)

// LimitsChecker is Layer 8 of the transaction pipeline
// Checks amount against min(LimitsMatrix, AdapterLimits) per SRS FR-LIMIT
type LimitsChecker struct {
	limitsRepo *repository.LimitsMatrixRepository
	txRepo     *repository.TransactionRepository
}

func NewLimitsChecker(limitsRepo *repository.LimitsMatrixRepository, txRepo *repository.TransactionRepository) *LimitsChecker {
	return &LimitsChecker{limitsRepo: limitsRepo, txRepo: txRepo}
}

func (lc *LimitsChecker) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := GetCombinedRequest(r.Context())
		if req == nil {
			response.BadRequest(w, response.ErrInternalError, "Request not parsed")
			return
		}

		sideA := &req.SideA

		// Determine amount
		var amount decimal.Decimal
		if sideA.Amount != nil {
			amount = decimal.NewFromFloat(*sideA.Amount)
		} else {
			// No amount — skip limits check (e.g., inquiry)
			next.ServeHTTP(w, r)
			return
		}

		// 1. Get matrix limits
		matrixLimits, err := lc.limitsRepo.GetLimits(r.Context(),
			sideA.WalletID, sideA.OperationType, sideA.Currency, "basic")
		if err != nil {
			slog.Warn("No limits found, using default",
				"walletId", sideA.WalletID,
				"opType", sideA.OperationType,
				"currency", sideA.Currency,
			)
			// Use very high default if no limits configured
			matrixLimits = &model.LimitsMatrix{
				MaxTxAmount: decimal.NewFromInt(999999999),
				MaxDaily:    decimal.NewFromInt(999999999),
			}
		}

		// 2. Check per-transaction limit
		if amount.GreaterThan(matrixLimits.MaxTxAmount) {
			slog.Warn("Amount exceeds per-tx limit",
				"amount", amount, "limit", matrixLimits.MaxTxAmount)
			response.BadRequest(w, response.ErrAmountExceedsLimit,
				"Amount exceeds per-transaction limit")
			return
		}

		// 3. Check daily limit
		dailyTotal, err := lc.txRepo.GetDailyTotalByDevice(r.Context(), sideA.DeviceID)
		if err != nil {
			slog.Error("Failed to get daily total", "error", err)
		} else {
			if dailyTotal.Add(amount).GreaterThan(matrixLimits.MaxDaily) {
				slog.Warn("Amount exceeds daily limit",
					"dailyTotal", dailyTotal,
					"amount", amount,
					"limit", matrixLimits.MaxDaily)
				response.BadRequest(w, response.ErrAmountExceedsLimit,
					"Amount exceeds daily limit")
				return
			}
		}

		// TODO Phase 4: Also query adapter.GetLimits() and use min(matrix, adapter)

		slog.Debug("Limits check passed",
			"amount", amount,
			"maxTx", matrixLimits.MaxTxAmount,
			"dailyTotal", dailyTotal,
			"maxDaily", matrixLimits.MaxDaily,
		)
		next.ServeHTTP(w, r)
	})
}
