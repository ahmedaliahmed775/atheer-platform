// أنواع الأخطاء ورموزها — يُرجى الرجوع إلى SPEC §8
package model

import "fmt"

// AppError — خطأ تطبيقي موحّد يحمل رمز الخطأ ورسالته وحالة HTTP
type AppError struct {
	Code       string // رمز الخطأ مثل HMAC_MISMATCH
	Message    string // رسالة الخطأ بالعربية
	HTTPStatus int    // حالة HTTP المقابلة مثل 401
}

// Error — تلبية واجهة error
func (e *AppError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// --- ثوابت رموز الأخطاء من SPEC §8 ---

const (
	// طبقة البوابة (GATE)
	ErrUnknownPayer     = "UNKNOWN_PAYER"     // الدافع غير مسجّل — 404
	ErrAccountSuspended = "ACCOUNT_SUSPENDED" // الحساب معلّق — 403
	ErrDeviceMismatch   = "DEVICE_MISMATCH"   // الجهاز غير مطابق — 403

	// طبقة التحقق (VERIFY)
	ErrMerchantUnauthorized = "MERCHANT_UNAUTHORIZED" // التاجر غير مُصرَّح — 401
	ErrTimestampExpired      = "TIMESTAMP_EXPIRED"     // الطابع الزمني منتهي — 400
	ErrCounterReplay         = "COUNTER_REPLAY"        // العداد مُعاد تشغيله — 400
	ErrCounterOutOfWindow    = "COUNTER_OUT_OF_WINDOW" // العداد خارج النافذة — 400
	ErrHMACMismatch          = "HMAC_MISMATCH"          // توقيع HMAC غير مطابق — 401
	ErrPayerLimitExceeded    = "PAYER_LIMIT_EXCEEDED"   // حد الدافع متجاوز — 400
	ErrLimitExceeded         = "LIMIT_EXCEEDED"          // حد المعاملات متجاوز — 400

	// طبقة التنفيذ (EXECUTE)
	ErrDebitFailed              = "DEBIT_FAILED"               // فشل الخصم — 502
	ErrCreditFailed             = "CREDIT_FAILED"              // فشل الإيداع — 502
	ErrWalletUnavailable        = "WALLET_UNAVAILABLE"         // المحفظة غير متاحة — 503
	ErrCrossWalletNotSupported    = "CROSS_WALLET_NOT_SUPPORTED"     // المعاملة بين محافظ مختلفة غير مدعومة — 400
	ErrInvalidRequest             = "INVALID_REQUEST"                // طلب غير صالح — حقول مفقودة — 400
	ErrWalletNotFound             = "WALLET_NOT_FOUND"               // المحفظة غير مسجّلة — 404
	ErrWalletInactive             = "WALLET_INACTIVE"                // المحفظة معطّلة — 403
	ErrDeviceAlreadyRegistered    = "DEVICE_ALREADY_REGISTERED"      // الجهاز مسجّل مسبقاً — 409
	ErrWalletAuthFailed           = "WALLET_AUTH_FAILED"             // فشل مصادقة المحفظة — 401
	ErrUnauthorized               = "UNAUTHORIZED"                   // غير مُصرَّح — مفتاح API مفقود — 401

	// أخطاء الإدارة (ADMIN)
	ErrInvalidCredentials  = "INVALID_CREDENTIALS"   // بيانات الدخول غير صحيحة — 401
	ErrTOTPRequired        = "TOTP_REQUIRED"         // رمز TOTP مطلوب — 401
	ErrTokenExpired        = "TOKEN_EXPIRED"          // الرمز منتهي الصلاحية — 401
	ErrTokenRevoked        = "TOKEN_REVOKED"          // الرمز ملغى — 401
	ErrForbiddenRole       = "FORBIDDEN_ROLE"         // الدور لا يملك صلاحية — 403
	ErrAdminNotFound       = "ADMIN_NOT_FOUND"        // المستخدم الإداري غير موجود — 404
	ErrReconInProgress     = "RECON_IN_PROGRESS"      // التسوية قيد التنفيذ — 409
)

// errorRegistry — سجلّ يربط كل رمز خطأ بحالة HTTP والرسالة الافتراضية
var errorRegistry = map[string]struct {
	HTTPStatus int
	Message    string
}{
	ErrUnknownPayer:        {404, "الدافع غير مسجّل في النظام"},
	ErrAccountSuspended:    {403, "الحساب معلّق"},
	ErrDeviceMismatch:      {403, "الجهاز غير مطابق للسجل"},
	ErrMerchantUnauthorized: {401, "رمز الوصول الخاص بالتاجر غير صالح"},
	ErrTimestampExpired:     {400, "الطابع الزمني منتهي الصلاحية"},
	ErrCounterReplay:       {400, "العداد يُشير إلى إعادة تشغيل"},
	ErrCounterOutOfWindow:  {400, "العداد خارج نافذة القبول"},
	ErrHMACMismatch:        {401, "توقيع HMAC غير مطابق"},
	ErrPayerLimitExceeded:  {400, "المبلغ يتجاوز حد الدافع"},
	ErrLimitExceeded:       {400, "المبلغ يتجاوز حدود المعاملات"},
	ErrDebitFailed:              {502, "فشل عملية الخصم من المحفظة"},
	ErrCreditFailed:             {502, "فشل عملية الإيداع في المحفظة"},
	ErrWalletUnavailable:        {503, "محفظة الدفع غير متاحة حالياً"},
	ErrCrossWalletNotSupported:    {400, "المعاملة بين محافظ مختلفة غير مدعومة في الإصدار الحالي"},
	ErrInvalidRequest:             {400, "طلب غير صالح — حقول مفقودة"},
	ErrWalletNotFound:             {404, "المحفظة غير مسجّلة في النظام"},
	ErrWalletInactive:             {403, "المحفظة معطّلة"},
	ErrDeviceAlreadyRegistered:    {409, "الجهاز مسجّل مسبقاً"},
	ErrWalletAuthFailed:           {401, "فشل مصادقة المحفظة — رمز الوصول غير صالح"},
	ErrUnauthorized:               {401, "غير مُصرَّح — مفتاح API مفقود أو غير صالح"},
	ErrInvalidCredentials:         {401, "بيانات الدخول غير صحيحة"},
	ErrTOTPRequired:               {401, "رمز التحقق الثنائي (TOTP) مطلوب"},
	ErrTokenExpired:               {401, "رمز المصادقة منتهي الصلاحية"},
	ErrTokenRevoked:               {401, "رمز المصادقة ملغى"},
	ErrForbiddenRole:              {403, "الدور لا يملك صلاحية الوصول"},
	ErrAdminNotFound:              {404, "المستخدم الإداري غير موجود"},
	ErrReconInProgress:            {409, "التسوية قيد التنفيذ بالفعل"},
}

// NewAppError — إنشاء خطأ تطبيقي من رمز الخطأ مع رسالة افتراضية وحالة HTTP
func NewAppError(code string) *AppError {
	entry, ok := errorRegistry[code]
	if !ok {
		return &AppError{
			Code:       code,
			Message:    "خطأ غير معروف",
			HTTPStatus: 500,
		}
	}
	return &AppError{
		Code:       code,
		Message:    entry.Message,
		HTTPStatus: entry.HTTPStatus,
	}
}

// NewAppErrorWithMessage — إنشاء خطأ تطبيقي مع رسالة مخصصة
func NewAppErrorWithMessage(code string, message string) *AppError {
	entry, ok := errorRegistry[code]
	if !ok {
		return &AppError{
			Code:       code,
			Message:    message,
			HTTPStatus: 500,
		}
	}
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: entry.HTTPStatus,
	}
}
