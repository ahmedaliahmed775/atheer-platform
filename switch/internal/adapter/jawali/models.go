// أنواع طلبات/ردود محفظة جوالي — أنواع خاصة بواجهة برمجة جوالي
// يُرجى الرجوع إلى Task 08 — Jawali API Mapping
package jawali

// --- رموز استجابة جوالي ---

const (
	// ResponseCodeSuccess — رمز النجاح
	ResponseCodeSuccess = "000"
	// ResponseCodeInsufficientFunds — رصيد غير كافٍ
	ResponseCodeInsufficientFunds = "001"
	// ResponseCodeInvalidAccount — حساب غير صالح
	ResponseCodeInvalidAccount = "002"
	// ResponseCodeTransactionNotFound — معاملة غير موجودة
	ResponseCodeTransactionNotFound = "003"
	// ResponseCodeDuplicateReference — مرجع مكرّر
	ResponseCodeDuplicateReference = "004"
	// ResponseCodeGeneralError — خطأ عام
	ResponseCodeGeneralError = "999"
)

// --- طلبات جوالي ---

// JawaliCashoutRequest — طلب خصم من محفظة جوالي (ECOMMCASHOUT)
type JawaliCashoutRequest struct {
	MerchantId string `json:"merchantId"` // معرّف التاجر
	PayerPhone string `json:"payerPhone"` // رقم هاتف الدافع
	Amount     int64  `json:"amount"`     // المبلغ بالوحدة الصغرى
	Currency   string `json:"currency"`   // العملة مثل YER
	Reference  string `json:"reference"`  // مرجع عدم التكرار (idempotency)
}

// JawaliCashinRequest — طلب إيداع في محفظة جوالي
type JawaliCashinRequest struct {
	MerchantId  string `json:"merchantId"`  // معرّف التاجر المستلم
	AccountRef  string `json:"accountRef"`  // مرجع حساب التاجر
	Amount      int64  `json:"amount"`      // المبلغ بالوحدة الصغرى
	Currency    string `json:"currency"`    // العملة مثل YER
	Reference   string `json:"reference"`   // مرجع عدم التكرار
	DebitRef    string `json:"debitRef"`    // مرجع عملية الخصم المرتبطة
}

// JawaliAuthVerifyRequest — طلب التحقق من رمز وصول التاجر
type JawaliAuthVerifyRequest struct {
	AccessToken string `json:"accessToken"` // رمز وصول التاجر
	WalletId    string `json:"walletId"`    // معرّف المحفظة
}

// JawaliReverseRequest — طلب عكس عملية خصم
type JawaliReverseRequest struct {
	OriginalReference string `json:"originalReference"` // مرجع عملية الخصم الأصلية
	Reason            string `json:"reason"`            // سبب العكس
}

// JawaliInquiryRequest — طلب استعلام عن معاملة (ECOMMERCEINQUIRY)
type JawaliInquiryRequest struct {
	Reference string `json:"reference"` // مرجع المعاملة
}

// --- ردود جوالي ---

// JawaliResponse — هيكل الرد المشترك لجميع عمليات جوالي
type JawaliResponse struct {
	ResponseCode    string `json:"responseCode"`    // رمز الاستجابة مثل 000
	ResponseMessage string `json:"responseMessage"` // رسالة الاستجابة
	Reference       string `json:"reference"`       // مرجع العملية
	Status          string `json:"status"`          // حالة العملية: SUCCESS أو FAILED أو PENDING
}

// JawaliCashoutResponse — رد عملية الخصم
type JawaliCashoutResponse struct {
	JawaliResponse
	TransactionRef string `json:"transactionRef"` // مرجع المعاملة في جوالي
	Balance        int64  `json:"balance"`        // الرصيد المتبقي بالوحدة الصغرى
}

// JawaliCashinResponse — رد عملية الإيداع
type JawaliCashinResponse struct {
	JawaliResponse
	TransactionRef string `json:"transactionRef"` // مرجع المعاملة في جوالي
}

// JawaliAuthVerifyResponse — رد التحقق من الرمز
type JawaliAuthVerifyResponse struct {
	JawaliResponse
	Valid bool `json:"valid"` // هل الرمز صالح
}

// JawaliReverseResponse — رد عملية العكس
type JawaliReverseResponse struct {
	JawaliResponse
	ReverseRef string `json:"reverseRef"` // مرجع عملية العكس
}

// JawaliInquiryResponse — رد الاستعلام
type JawaliInquiryResponse struct {
	JawaliResponse
	TransactionRef string `json:"transactionRef"` // مرجع المعاملة
	Amount         int64  `json:"amount"`         // المبلغ بالوحدة الصغرى
}

// IsSuccess — هل الاستجابة ناجحة
func (r JawaliResponse) IsSuccess() bool {
	return r.ResponseCode == ResponseCodeSuccess
}
