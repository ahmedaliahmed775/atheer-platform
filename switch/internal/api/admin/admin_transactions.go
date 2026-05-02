// معالج معاملات الإدارة — قائمة المعاملات مع تصفية وتصدير
// يُرجى الرجوع إلى SPEC §5 — Admin APIs
package admin

import (
	"encoding/csv"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/atheer/switch/internal/db"
	"github.com/atheer/switch/internal/middleware"
	"github.com/atheer/switch/internal/model"
)

// AdminTransactionsHandler — معالج معاملات الإدارة
type AdminTransactionsHandler struct {
	txRepo db.TransactionRepo
}

// NewAdminTransactionsHandler — ينشئ معالج معاملات الإدارة
func NewAdminTransactionsHandler(txRepo db.TransactionRepo) *AdminTransactionsHandler {
	return &AdminTransactionsHandler{txRepo: txRepo}
}

// TransactionListResponse — استجابة قائمة المعاملات
type TransactionListResponse struct {
	Transactions []model.Transaction `json:"transactions"` // قائمة المعاملات
	TotalCount   int                 `json:"totalCount"`   // العدد الإجمالي
	Page         int                 `json:"page"`         // رقم الصفحة
	PageSize     int                 `json:"pageSize"`     // حجم الصفحة
}

// TransactionDetailResponse — استجابة تفاصيل المعاملة مع الجدول الزمني
type TransactionDetailResponse struct {
	Transaction model.Transaction `json:"transaction"` // بيانات المعاملة
	Timeline    []TimelineEvent   `json:"timeline"`    // الجدول الزمني للأحداث
}

// TimelineEvent — حدث في الجدول الزمني للمعاملة
type TimelineEvent struct {
	Timestamp int64  `json:"timestamp"` // الطابع الزمني
	Event     string `json:"event"`     // اسم الحدث
	Detail    string `json:"detail"`    // تفاصيل الحدث
}

// HandleList — يعالج طلب قائمة المعاملات مع تصفية وصفحات
// GET /admin/v1/transactions?status=SUCCESS&walletId=jawali&page=1&pageSize=20
func (h *AdminTransactionsHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// استخراج معاملات التصفية من الاستعلام
	query := r.URL.Query()

	filters := model.TransactionFilters{
		Status:   query.Get("status"),
		WalletId: query.Get("walletId"),
	}

	// فلترة حسب النطاق — WALLET_ADMIN يرى محفظته فقط
	scopeFilter := middleware.ScopeFilter(ctx)
	if scopeFilter != "" {
		filters.WalletId = scopeFilter // تجاوز فلتر المحفظة بالنطاق
	}

	// معاملات إضافية
	if pid := query.Get("payerPublicId"); pid != "" {
		filters.PayerPublicId = pid
	}
	if mid := query.Get("merchantId"); mid != "" {
		filters.MerchantId = mid
	}
	if from := query.Get("fromDate"); from != "" {
		if t, err := time.Parse("2006-01-02", from); err == nil {
			filters.FromDate = t
		}
	}
	if to := query.Get("toDate"); to != "" {
		if t, err := time.Parse("2006-01-02", to); err == nil {
			filters.ToDate = t
		}
	}

	page := parseIntOrDefault(query.Get("page"), 1)
	pageSize := parseIntOrDefault(query.Get("pageSize"), 20)
	if pageSize > 100 {
		pageSize = 100 // حد أقصى
	}

	// جلب المعاملات
	transactions, totalCount, err := h.txRepo.List(ctx, filters, page, pageSize)
	if err != nil {
		slog.Error("إدارة المعاملات: فشل جلب القائمة", "error", err)
		writeAdminJSON(w, http.StatusInternalServerError, map[string]string{
			"errorCode":    "INTERNAL_ERROR",
			"errorMessage": "خطأ في جلب المعاملات",
		})
		return
	}

	if transactions == nil {
		transactions = []model.Transaction{}
	}

	// التحقق من طلب التصدير
	if query.Get("export") == "csv" {
		h.exportCSV(w, transactions)
		return
	}

	writeAdminJSON(w, http.StatusOK, TransactionListResponse{
		Transactions: transactions,
		TotalCount:   totalCount,
		Page:         page,
		PageSize:     pageSize,
	})
}

// HandleGetByID — يعالج طلب تفاصيل معاملة واحدة مع الجدول الزمني
// GET /admin/v1/transactions/{id}
func (h *AdminTransactionsHandler) HandleGetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// استخراج معرّف المعاملة من المسار
	// في Go 1.22+ نستخدم PathValue
	txID := r.PathValue("id")
	if txID == "" {
		writeAdminJSON(w, http.StatusBadRequest, map[string]string{
			"errorCode":    model.ErrInvalidRequest,
			"errorMessage": "معرّف المعاملة مطلوب",
		})
		return
	}

	// البحث عن المعاملة
	tx, err := h.txRepo.FindByID(ctx, txID)
	if err != nil {
		slog.Error("إدارة المعاملات: فشل البحث", "id", txID, "error", err)
		writeAdminJSON(w, http.StatusInternalServerError, map[string]string{
			"errorCode":    "INTERNAL_ERROR",
			"errorMessage": "خطأ داخلي",
		})
		return
	}

	if tx == nil {
		writeAdminJSON(w, http.StatusNotFound, map[string]string{
			"errorCode":    "TRANSACTION_NOT_FOUND",
			"errorMessage": "المعاملة غير موجودة",
		})
		return
	}

	// فلترة النطاق — WALLET_ADMIN يرى معاملات محفظته فقط
	scopeFilter := middleware.ScopeFilter(ctx)
	if scopeFilter != "" && tx.PayerWalletId != scopeFilter && tx.MerchantWalletId != scopeFilter {
		writeAdminJSON(w, http.StatusForbidden, map[string]string{
			"errorCode":    model.ErrForbiddenRole,
			"errorMessage": "ليس لديك صلاحية لعرض هذه المعاملة",
		})
		return
	}

	// بناء الجدول الزمني
	timeline := h.buildTimeline(tx)

	writeAdminJSON(w, http.StatusOK, TransactionDetailResponse{
		Transaction: *tx,
		Timeline:    timeline,
	})
}

// buildTimeline — يبني الجدول الزمني لأحداث المعاملة
func (h *AdminTransactionsHandler) buildTimeline(tx *model.Transaction) []TimelineEvent {
	var timeline []TimelineEvent

	// حدث الإنشاء
	timeline = append(timeline, TimelineEvent{
		Timestamp: tx.CreatedAt.Unix(),
		Event:     "CREATED",
		Detail:    fmt.Sprintf("تم إنشاء المعاملة — المبلغ: %d %s", tx.Amount, tx.Currency),
	})

	// حدث الخصم
	if tx.DebitRef != "" {
		timeline = append(timeline, TimelineEvent{
			Timestamp: tx.CreatedAt.Unix(),
			Event:     "DEBIT_COMPLETED",
			Detail:    fmt.Sprintf("تم الخصم من الدافع — المرجع: %s", tx.DebitRef),
		})
	}

	// حدث الإيداع
	if tx.CreditRef != "" {
		timeline = append(timeline, TimelineEvent{
			Timestamp: tx.CreatedAt.Unix(),
			Event:     "CREDIT_COMPLETED",
			Detail:    fmt.Sprintf("تم الإيداع للتاجر — المرجع: %s", tx.CreditRef),
		})
	}

	// حدث الفشل
	if tx.Status == "FAILED" && tx.ErrorCode != "" {
		timeline = append(timeline, TimelineEvent{
			Timestamp: tx.CreatedAt.Unix(),
			Event:     "FAILED",
			Detail:    fmt.Sprintf("فشلت المعاملة — السبب: %s", tx.ErrorCode),
		})
	}

	return timeline
}

// exportCSV — يُصدّر المعاملات بصيغة CSV
func (h *AdminTransactionsHandler) exportCSV(w http.ResponseWriter, transactions []model.Transaction) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=transactions_"+strconv.FormatInt(time.Now().Unix(), 10)+".csv")

	// كتابة BOM لدعم العربية في Excel
	w.Write([]byte{0xEF, 0xBB, 0xBF})

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// كتابة رؤوس الأعمدة
	headers := []string{
		"معرّف المعاملة", "معرّف الدافع", "معرّف التاجر",
		"محفظة الدافع", "محفظة التاجر", "المبلغ", "العملة",
		"العداد", "الحالة", "رمز الخطأ", "المدة (ms)", "التاريخ",
	}
	writer.Write(headers)

	// كتابة البيانات
	for _, tx := range transactions {
		row := []string{
			tx.TransactionId, tx.PayerPublicId, tx.MerchantId,
			tx.PayerWalletId, tx.MerchantWalletId,
			strconv.FormatInt(tx.Amount, 10), tx.Currency,
			strconv.FormatInt(tx.Counter, 10), tx.Status,
			tx.ErrorCode, strconv.Itoa(tx.DurationMs),
			tx.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		writer.Write(row)
	}
}

// HandleExport — يُصدّر المعاملات بصيغة JSON
// GET /admin/v1/transactions/export?format=json
func (h *AdminTransactionsHandler) HandleExport(w http.ResponseWriter, r *http.Request) {
	// إعادة توجيه إلى HandleList مع معامل export
	r.URL.RawQuery += "&export=csv"
	h.HandleList(w, r)
}

// parseTransactionFilters — يحلل معاملات التصفية من الاستعلام (مُصدَّر للاستخدام العام)
func parseTransactionFilters(query map[string][]string) model.TransactionFilters {
	filters := model.TransactionFilters{
		Status:       firstOrEmpty(query["status"]),
		WalletId:     firstOrEmpty(query["walletId"]),
		PayerPublicId: firstOrEmpty(query["payerPublicId"]),
		MerchantId:   firstOrEmpty(query["merchantId"]),
	}

	if from := firstOrEmpty(query["fromDate"]); from != "" {
		if t, err := time.Parse("2006-01-02", from); err == nil {
			filters.FromDate = t
		}
	}
	if to := firstOrEmpty(query["toDate"]); to != "" {
		if t, err := time.Parse("2006-01-02", to); err == nil {
			filters.ToDate = t
		}
	}

	return filters
}

// firstOrEmpty — يُرجع أول قيمة أو سلسلة فارغة
func firstOrEmpty(vals []string) string {
	if len(vals) > 0 && vals[0] != "" {
		return vals[0]
	}
	return ""
}
