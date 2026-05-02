// فحص حدود المعاملات — يومي وشهري
// يُرجى الرجوع إلى SPEC §3 Layer 2 — الخطوة 7
package verify

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/atheer/switch/internal/db"
	"github.com/atheer/switch/internal/model"
)

// LimitsConfig — إعدادات حدود المعاملات
type LimitsConfig struct {
	DailyLimit   int64 // الحد اليومي بالوحدة الصغرى
	MonthlyLimit int64 // الحد الشهري بالوحدة الصغرى
}

// LimitsChecker — فاحص حدود المعاملات اليومية والشهرية
type LimitsChecker struct {
	txRepo  db.TransactionRepo // مستودع المعاملات
	limits  LimitsConfig       // إعدادات الحدود
}

// NewLimitsChecker — ينشئ نسخة فاحص الحدود
func NewLimitsChecker(txRepo db.TransactionRepo, limits LimitsConfig) *LimitsChecker {
	return &LimitsChecker{
		txRepo: txRepo,
		limits: limits,
	}
}

// CheckLimits — يتحقق من أن المبلغ لا يتجاوز الحدود اليومية والشهرية
// يفحص: المبلغ المتراكم اليوم + المبلغ الجديد ≤ الحد اليومي
//        المبلغ المتراكم الشهر + المبلغ الجديد ≤ الحد الشهري
func (lc *LimitsChecker) CheckLimits(ctx context.Context, publicId string, amount int64) error {
	now := time.Now()

	// فحص الحد اليومي
	if lc.limits.DailyLimit > 0 {
		today := now.Format("2006-01-02")
		dailyTotal, err := lc.txRepo.GetDailyTotal(ctx, publicId, today)
		if err != nil {
			return fmt.Errorf("فاحص الحدود: إجمالي يومي: %w", err)
		}

		if dailyTotal+amount > lc.limits.DailyLimit {
			slog.Warn("فاحص الحدود: تجاوز الحد اليومي",
				"publicId", publicId,
				"dailyTotal", dailyTotal,
				"amount", amount,
				"dailyLimit", lc.limits.DailyLimit,
			)
			return model.NewAppError(model.ErrLimitExceeded)
		}
	}

	// فحص الحد الشهري
	if lc.limits.MonthlyLimit > 0 {
		month := now.Format("2006-01")
		monthlyTotal, err := lc.txRepo.GetMonthlyTotal(ctx, publicId, month)
		if err != nil {
			return fmt.Errorf("فاحص الحدود: إجمالي شهري: %w", err)
		}

		if monthlyTotal+amount > lc.limits.MonthlyLimit {
			slog.Warn("فاحص الحدود: تجاوز الحد الشهري",
				"publicId", publicId,
				"monthlyTotal", monthlyTotal,
				"amount", amount,
				"monthlyLimit", lc.limits.MonthlyLimit,
			)
			return model.NewAppError(model.ErrLimitExceeded)
		}
	}

	return nil
}
