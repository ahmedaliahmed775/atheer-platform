package middleware

import (
	"log/slog"
	"math"
	"net/http"
	"time"

	"github.com/atheer-payment/atheer-platform/pkg/response"
)

// CrossValidator is Layer 7 of the transaction pipeline
// Ensures Side A and Side B payloads are consistent (FR-SEC-004)
type CrossValidator struct {
	timeWindowMinutes int
}

func NewCrossValidator(timeWindowMinutes int) *CrossValidator {
	return &CrossValidator{timeWindowMinutes: timeWindowMinutes}
}

func (cv *CrossValidator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := GetCombinedRequest(r.Context())
		if req == nil {
			response.BadRequest(w, response.ErrInternalError, "Request not parsed")
			return
		}

		sideA := &req.SideA
		sideB := &req.SideB
		now := time.Now().UnixMilli()

		// Rule 1: Currency must match
		if sideA.Currency != sideB.Currency {
			slog.Warn("Cross-validation: currency mismatch",
				"sideA", sideA.Currency, "sideB", sideB.Currency)
			response.BadRequest(w, response.ErrCrossValidationFail,
				"Currency mismatch between Side A and Side B")
			return
		}

		// Rule 2: Operation type must match
		if sideA.OperationType != sideB.OperationType {
			slog.Warn("Cross-validation: operation type mismatch",
				"sideA", sideA.OperationType, "sideB", sideB.OperationType)
			response.BadRequest(w, response.ErrCrossValidationFail,
				"Operation type mismatch")
			return
		}

		// Rule 3: Side A timestamp within window
		windowMs := int64(cv.timeWindowMinutes) * 60 * 1000
		if abs64(now-sideA.Timestamp) > windowMs {
			slog.Warn("Cross-validation: Side A timestamp expired",
				"sideA_ts", sideA.Timestamp, "server_ts", now,
				"diff_ms", abs64(now-sideA.Timestamp))
			response.BadRequest(w, response.ErrTimestampExpired,
				"Side A timestamp outside allowed window")
			return
		}

		// Rule 4: Side B timestamp within window
		if abs64(now-sideB.Timestamp) > windowMs {
			slog.Warn("Cross-validation: Side B timestamp expired",
				"sideB_ts", sideB.Timestamp, "server_ts", now)
			response.BadRequest(w, response.ErrTimestampExpired,
				"Side B timestamp outside allowed window")
			return
		}

		// Rule 5: For P2P, amounts must match
		opType := sideA.OperationType
		if opType == "P2P_SAME" || opType == "P2P_CROSS" {
			if !amountsEqual(sideA.Amount, sideB.Amount) {
				slog.Warn("Cross-validation: P2P amount mismatch")
				response.BadRequest(w, response.ErrCrossValidationFail,
					"P2P amount mismatch between Side A and Side B")
				return
			}
		}

		// Rule 6: For P2M, merchantId required on Side B
		if opType == "P2M_SAME" || opType == "P2M_CROSS" {
			if sideB.MerchantID == nil || *sideB.MerchantID == "" {
				slog.Warn("Cross-validation: P2M missing merchantId")
				response.BadRequest(w, response.ErrCrossValidationFail,
					"P2M requires merchantId on Side B")
				return
			}
		}

		// Rule 7: SAME → same wallet; CROSS → different wallets
		if opType == "P2P_SAME" || opType == "P2M_SAME" {
			if sideA.WalletID != sideB.WalletID {
				slog.Warn("Cross-validation: SAME operation but different wallets")
				response.BadRequest(w, response.ErrCrossValidationFail,
					"SAME operation requires same wallet on both sides")
				return
			}
		}
		if opType == "P2P_CROSS" || opType == "P2M_CROSS" {
			if sideA.WalletID == sideB.WalletID {
				slog.Warn("Cross-validation: CROSS operation but same wallet")
				response.BadRequest(w, response.ErrCrossValidationFail,
					"CROSS operation requires different wallets")
				return
			}
		}

		slog.Debug("Cross-validation passed",
			"opType", opType,
			"currency", sideA.Currency,
		)
		next.ServeHTTP(w, r)
	})
}

func abs64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

func amountsEqual(a, b *float64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return math.Abs(*a-*b) < 0.001
}
