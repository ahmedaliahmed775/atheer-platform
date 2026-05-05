// معالج إحصائيات العمولات — عرض عمولات شركة الاتصالات لكل محفظة
package admin

import (
	"log/slog"
	"net/http"

	"github.com/atheer/switch/internal/db"
	"github.com/atheer/switch/internal/model"
)

// AdminCommissionHandler — معالج إحصائيات العمولات
type AdminCommissionHandler struct {
	commissionRepo  db.CarrierCommissionRepo
	commissionRate  int64 // نسبة العمولة بالألف
}

// NewAdminCommissionHandler — ينشئ معالج إحصائيات العمولات
func NewAdminCommissionHandler(commissionRepo db.CarrierCommissionRepo, commissionRate int64) *AdminCommissionHandler {
	return &AdminCommissionHandler{
		commissionRepo: commissionRepo,
		commissionRate: commissionRate,
	}
}

// CommissionStatsResponse — استجابة إحصائيات العمولات
type CommissionStatsResponse struct {
	Wallets      []CommissionWalletStats `json:"wallets"`      // تفاصيل كل محفظة
	TotalTxCount int                     `json:"totalTxCount"` // إجمالي المعاملات عبر الاتصالات
	TotalAmount  int64                   `json:"totalAmount"`  // إجمالي المبلغ
	TotalDue     int64                   `json:"totalDue"`     // إجمالي العمولات المستحقة
	Rate         int64                   `json:"rate"`         // نسبة العمولة بالألف
}

// CommissionWalletStats — إحصائيات عمولات محفظة واحدة
type CommissionWalletStats struct {
	WalletId       string `json:"walletId"`       // معرّف المحفظة
	TotalTxCount   int    `json:"totalTxCount"`   // إجمالي عدد المعاملات
	TotalAmount    int64  `json:"totalAmount"`    // إجمالي المبلغ بالوحدة الصغرى
	SuccessCount   int    `json:"successCount"`   // عدد المعاملات الناجحة
	FailedCount    int    `json:"failedCount"`    // عدد المعاملات الفاشلة
	CommissionRate int64  `json:"commissionRate"` // نسبة العمولة بالألف
	CommissionDue  int64  `json:"commissionDue"`  // العمولة المستحقة بالوحدة الصغرى
}

// HandleStats — يعالج طلب إحصائيات العمولات
// GET /admin/v1/commission/stats?fromDate=2024-01-01&toDate=2024-01-31
func (h *AdminCommissionHandler) HandleStats(w http.ResponseWriter, r *http.Request) {
	// التحقق من الصلاحية — ADMIN على الأقل
	if !checkRole(w, r, model.RoleAdmin) {
		return
	}

	// استخراج معاملات التصفية
	query := r.URL.Query()
	fromDate := query.Get("fromDate")
	toDate := query.Get("toDate")

	// جلب الإحصائيات
	summary, err := h.commissionRepo.GetCommissionStats(r.Context(), h.commissionRate, fromDate, toDate)
	if err != nil {
		slog.Error("العمولات: فشل جلب الإحصائيات", "error", err)
		writeAdminJSON(w, http.StatusInternalServerError, map[string]string{"error": "فشل جلب إحصائيات العمولات"})
		return
	}

	// تحويل النموذج إلى استجابة API
	walletStats := make([]CommissionWalletStats, 0, len(summary.Wallets))
	for _, w := range summary.Wallets {
		walletStats = append(walletStats, CommissionWalletStats{
			WalletId:       w.WalletId,
			TotalTxCount:   w.TotalTxCount,
			TotalAmount:    w.TotalAmount,
			SuccessCount:   w.SuccessCount,
			FailedCount:    w.FailedCount,
			CommissionRate: w.CommissionRate,
			CommissionDue:  w.CommissionDue,
		})
	}

	resp := CommissionStatsResponse{
		Wallets:      walletStats,
		TotalTxCount: summary.TotalTxCount,
		TotalAmount:  summary.TotalAmount,
		TotalDue:     summary.TotalDue,
		Rate:         h.commissionRate,
	}

	writeAdminJSON(w, http.StatusOK, resp)
}
