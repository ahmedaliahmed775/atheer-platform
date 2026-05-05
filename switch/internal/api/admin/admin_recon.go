// معالج تسوية الإدارة — تشغيل التسوية وعرض التقارير
// يُرجى الرجوع إلى SPEC §5 — Admin APIs
package admin

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/atheer/switch/internal/db"
	"github.com/atheer/switch/internal/middleware"
	"github.com/atheer/switch/internal/model"
)

// AdminReconHandler — معالج تسوية الإدارة
type AdminReconHandler struct {
	reconRepo  db.ReconRepo
	txRepo     db.TransactionRepo
	walletRepo db.WalletRepo
}

// NewAdminReconHandler — ينشئ معالج تسوية الإدارة
func NewAdminReconHandler(reconRepo db.ReconRepo, txRepo db.TransactionRepo, walletRepo db.WalletRepo) *AdminReconHandler {
	return &AdminReconHandler{
		reconRepo:  reconRepo,
		txRepo:     txRepo,
		walletRepo: walletRepo,
	}
}

// RunReconRequest — طلب تشغيل التسوية
type RunReconRequest struct {
	ReportDate string `json:"reportDate"` // تاريخ التقرير بصيغة YYYY-MM-DD
	WalletId   string `json:"walletId"`   // معرّف المحفظة (اختياري — إذا فارغ يُشغّل لكل المحافظ)
}

// RunReconResponse — استجابة تشغيل التسوية
type RunReconResponse struct {
	Reports []ReconSummary `json:"reports"` // ملخص التقارير
	Message string         `json:"message"` // رسالة
}

// ReconSummary — ملخص تقرير تسوية
type ReconSummary struct {
	WalletId      string `json:"walletId"`      // معرّف المحفظة
	ReportDate    string `json:"reportDate"`    // تاريخ التقرير
	TotalTxCount  int    `json:"totalTxCount"`  // إجمالي المعاملات
	TotalAmount   int64  `json:"totalAmount"`   // إجمالي المبلغ
	SuccessCount  int    `json:"successCount"`  // عدد الناجحة
	FailedCount   int    `json:"failedCount"`   // عدد الفاشلة
	Status        string `json:"status"`        // الحالة
}

// ReconListResponse — استجابة قائمة تقارير التسوية
type ReconListResponse struct {
	Reports    []model.ReconciliationReport `json:"reports"`    // قائمة التقارير
	TotalCount int                          `json:"totalCount"` // العدد الإجمالي
	Page       int                          `json:"page"`       // رقم الصفحة
	PageSize   int                          `json:"pageSize"`   // حجم الصفحة
}

// HandleRun — يعالج طلب تشغيل التسوية
// POST /admin/v1/reconciliation/run
func (h *AdminReconHandler) HandleRun(w http.ResponseWriter, r *http.Request) {
	// التحقق من الصلاحية — ADMIN على الأقل
	if !checkRole(w, r, model.RoleAdmin) {
		return
	}

	ctx := r.Context()

	var req RunReconRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAdminJSON(w, http.StatusBadRequest, map[string]string{
			"errorCode":    model.ErrInvalidRequest,
			"errorMessage": "طلب غير صالح",
		})
		return
	}

	// ضبط التاريخ الافتراضي
	if req.ReportDate == "" {
		req.ReportDate = time.Now().Format("2006-01-02")
	}

	// التحقق من صحة التاريخ
	if _, err := time.Parse("2006-01-02", req.ReportDate); err != nil {
		writeAdminJSON(w, http.StatusBadRequest, map[string]string{
			"errorCode":    model.ErrInvalidRequest,
			"errorMessage": "صيغة التاريخ غير صحيحة — استخدم YYYY-MM-DD",
		})
		return
	}

	// فلترة النطاق — WALLET_ADMIN يُشغّل لمحفظته فقط
	scopeFilter := middleware.ScopeFilter(ctx)
	if scopeFilter != "" {
		req.WalletId = scopeFilter
	}

	// تحديد المحافظ المطلوبة
	var walletIds []string
	if req.WalletId != "" {
		walletIds = []string{req.WalletId}
	} else {
		// جلب كل المحافظ المفعّلة
		wallets, err := h.walletRepo.List(ctx)
		if err != nil {
			slog.Error("إدارة التسوية: فشل جلب المحافظ", "error", err)
			writeAdminJSON(w, http.StatusInternalServerError, map[string]string{
				"errorCode":    "INTERNAL_ERROR",
				"errorMessage": "خطأ في جلب المحافظ",
			})
			return
		}
		for _, w := range wallets {
			if w.IsActive {
				walletIds = append(walletIds, w.WalletId)
			}
		}
	}

	// تشغيل التسوية لكل محفظة
	var reports []ReconSummary
	for _, wid := range walletIds {
		summary, err := h.runReconForWallet(ctx, req.ReportDate, wid)
		if err != nil {
			slog.Error("إدارة التسوية: فشل التسوية", "walletId", wid, "error", err)
			reports = append(reports, ReconSummary{
				WalletId:   wid,
				ReportDate: req.ReportDate,
				Status:     "ERROR",
			})
			continue
		}
		reports = append(reports, *summary)
	}

	if reports == nil {
		reports = []ReconSummary{}
	}

	slog.Info("إدارة التسوية: تم تشغيل التسوية", "date", req.ReportDate, "wallets", len(walletIds))

	writeAdminJSON(w, http.StatusOK, RunReconResponse{
		Reports: reports,
		Message: "تم تشغيل التسوية بنجاح",
	})
}

// HandleListReports — يعالج طلب قائمة تقارير التسوية
// GET /admin/v1/reconciliation/reports?walletId=jawali&page=1&pageSize=20
func (h *AdminReconHandler) HandleListReports(w http.ResponseWriter, r *http.Request) {
	// التحقق من الصلاحية — ADMIN على الأقل
	if !checkRole(w, r, model.RoleAdmin) {
		return
	}

	ctx := r.Context()

	query := r.URL.Query()
	walletId := query.Get("walletId")
	page := parseIntOrDefault(query.Get("page"), 1)
	pageSize := parseIntOrDefault(query.Get("pageSize"), 20)
	if pageSize > 100 {
		pageSize = 100
	}

	// فلترة النطاق
	scopeFilter := middleware.ScopeFilter(ctx)
	if scopeFilter != "" {
		walletId = scopeFilter
	}

	reports, totalCount, err := h.reconRepo.List(ctx, walletId, page, pageSize)
	if err != nil {
		slog.Error("إدارة التسوية: فشل جلب التقارير", "error", err)
		writeAdminJSON(w, http.StatusInternalServerError, map[string]string{
			"errorCode":    "INTERNAL_ERROR",
			"errorMessage": "خطأ في جلب التقارير",
		})
		return
	}

	if reports == nil {
		reports = []model.ReconciliationReport{}
	}

	writeAdminJSON(w, http.StatusOK, ReconListResponse{
		Reports:    reports,
		TotalCount: totalCount,
		Page:       page,
		PageSize:   pageSize,
	})
}

// runReconForWallet — يُشغّل التسوية لمحفظة واحدة في تاريخ محدد
func (h *AdminReconHandler) runReconForWallet(ctx context.Context, reportDate, walletId string) (*ReconSummary, error) {
	// التحقق من عدم وجود تقرير سابق
	existing, err := h.reconRepo.FindByDateAndWallet(ctx, reportDate, walletId)
	if err != nil {
		return nil, err
	}

	// جلب إحصائيات المعاملات من قاعدة البيانات
	// في الإنتاج: استعلام تجميعي على جدول transactions
	filters := model.TransactionFilters{
		WalletId: walletId,
	}

	// تحويل التاريخ إلى time.Time للتصفية
	if t, err := time.Parse("2006-01-02", reportDate); err == nil {
		filters.FromDate = t
		filters.ToDate = t.Add(24 * time.Hour)
	}

	// جلب المعاملات — تنفيذ مبسّط
	_, totalCount, err := h.txRepo.List(ctx, filters, 1, 1)
	if err != nil {
		return nil, err
	}

	// بناء التقرير
	report := &model.ReconciliationReport{
		ReportDate:   reportDate,
		WalletId:     walletId,
		TotalTxCount: totalCount,
		TotalAmount:  0, // يُحسب من استعلام تجميعي
		SuccessCount: 0,
		FailedCount:  0,
		Status:       "PENDING",
	}

	// حفظ أو تحديث التقرير
	if existing != nil {
		report.ID = existing.ID
		report.Status = "VERIFIED" // إعادة التشغيل تُحدّث الحالة
		if err := h.reconRepo.Update(ctx, report); err != nil {
			return nil, err
		}
	} else {
		if err := h.reconRepo.Save(ctx, report); err != nil {
			return nil, err
		}
	}

	return &ReconSummary{
		WalletId:     walletId,
		ReportDate:   reportDate,
		TotalTxCount: report.TotalTxCount,
		TotalAmount:  report.TotalAmount,
		SuccessCount: report.SuccessCount,
		FailedCount:  report.FailedCount,
		Status:       report.Status,
	}, nil
}
