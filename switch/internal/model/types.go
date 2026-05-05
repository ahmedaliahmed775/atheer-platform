// أنواع البيانات الأساسية — يُرجى الرجوع إلى SPEC §1 و§3 و§4 و§6
// جميع أسماء الحقول بصيغة camelCase عبر json tags لتوافق مع العقد الموحد (OpenAPI)
package model

import (
	"context"
	"time"
)

// --- توكن الدفع وبيانات التاجر ---

// PaymentToken — التوكن المُستقبَل من جهاز الدافع عبر NFC
type PaymentToken struct {
	PublicId  string `json:"publicId"`  // المعرّف العام للدافع مثل usr_abc123
	DeviceId  string `json:"deviceId"`  // معرّف الجهاز مثل dev_456
	Counter   int64  `json:"counter"`   // عداد المعاملات (رقم تصاعدي)
	Timestamp int64  `json:"timestamp"` // الطابع الزمني بالثواني (Unix)
	HMAC      string `json:"hmac"`      // توقيع HMAC-SHA256 بصيغة base64
}

// MerchantData — بيانات التاجر المُرسلة من SDK التاجر
type MerchantData struct {
	MerchantId       string `json:"merchantId"`       // معرّف التاجر (رقم الحساب)
	MerchantWalletId string `json:"merchantWalletId"` // معرّف محفظة التاجر مثل jawali
	Amount           int64  `json:"amount"`           // المبلغ بالوحدة الصغرى (لا float)
	Currency         string `json:"currency"`         // العملة مثل YER
	AccessToken      string `json:"accessToken"`      // رمز وصول التاجر للمصادقة
}

// --- طلب واستجابة المعاملة ---

// TransactionRequest — طلب المعاملة المُرسل من SDK التاجر
type TransactionRequest struct {
	PaymentToken  PaymentToken  `json:"paymentToken"`  // توكن الدفع من NFC
	MerchantData  MerchantData  `json:"merchantData"`  // بيانات التاجر
	Timestamp     int64         `json:"timestamp"`     // الطابع الزمني للطلب بالثواني (Unix)
}

// TransactionResponse — استجابة المعاملة المُعادة للتاجر
type TransactionResponse struct {
	TransactionId    string `json:"transactionId"`    // معرّف المعاملة (UUID)
	Status           string `json:"status"`           // حالة المعاملة: SUCCESS أو FAILED
	ErrorCode        string `json:"errorCode"`        // رمز الخطأ إن وُجد مثل HMAC_MISMATCH
	ErrorMessage     string `json:"errorMessage"`     // رسالة الخطأ
	LastValidCounter int64  `json:"lastValidCounter"` // آخر عداد صالح
	Timestamp        int64  `json:"timestamp"`        // الطابع الزمني للاستجابة بالثواني (Unix)
}

// --- التسجيل (Enrollment) ---

// EnrollRequest — طلب تسجيل جهاز دافع جديد
// الحقول الاختيارية (omitempty) يرسلها الـ SDK عند توفرها فقط
type EnrollRequest struct {
	WalletId             string   `json:"walletId"`                       // معرّف المحفظة مثل jawali
	WalletToken          string   `json:"walletToken"`                    // رمز المصادقة من خادم المحفظة
	DeviceId             string   `json:"deviceId"`                       // معرّف الجهاز (64 حرف hex)
	UserType             string   `json:"userType"`                       // نوع المستخدم: P (دافع) أو M (تاجر)
	PublicKey            string   `json:"publicKey,omitempty"`            // المفتاح العام من TEE بصيغة base64
	AttestationPublicKey string   `json:"attestationPublicKey,omitempty"` // مفتاح شهادة التوثيق ECDSA P-256 بصيغة base64
	AttestationCert      []string `json:"attestationCert,omitempty"`      // سلسلة شهادات التوثيق بصيغة base64
	PlayIntegrityToken   string   `json:"playIntegrityToken,omitempty"`   // رمز Google Play Integrity
}

// EnrollResponse — استجابة التسجيل
type EnrollResponse struct {
	PublicId         string `json:"publicId"`         // المعرّف العام المُنشأ مثل usr_abc123
	EncryptedSeed    string `json:"encryptedSeed"`    // البذرة المشفّرة بصيغة base64
	PayerLimit       int64  `json:"payerLimit"`       // حد الدافع الافتراضي بالوحدة الصغرى
	MaxPayerLimit    int64  `json:"maxPayerLimit"`    // الحد الأقصى للدافع بالوحدة الصغرى
	AttestationLevel string `json:"attestationLevel"` // مستوى التوثيق: TEE أو STRONGBOX أو SOFTWARE
	Status           string `json:"status"`           // الحالة: ACTIVE
}

// --- نتائج طبقات المعالجة ---

// GateResult — نتيجة طبقة البوابة (GATE)
type GateResult struct {
	PayerPublicId string // المعرّف العام للدافع
	PayerWalletId string // معرّف محفظة الدافع
	SeedEncrypted []byte // البذرة المشفّرة (لا تُفكّ هنا)
	SeedKeyID     string // معرّف مفتاح KMS لفك التشفير
	PayerCounter  int64  // آخر عداد مسجّل للدافع
	PayerLimit    int64  // حد الدافع بالوحدة الصغرى
}

// VerifyResult — نتيجة طبقة التحقق (VERIFY)
type VerifyResult struct {
	PayerWalletId    string // معرّف محفظة الدافع
	MerchantWalletId string // معرّف محفظة التاجر
	Amount          int64  // المبلغ بالوحدة الصغرى
	Currency        string // العملة
	NewCounter      int64  // العداد الجديد بعد التحقق
}

// --- سجلات قاعدة البيانات (§4) ---

// SwitchRecord — سجل الدافع/التاجر في جدول switch_records
type SwitchRecord struct {
	ID            int64     // المعرّف الداخلي (توليد تلقائي)
	PublicId      string    // المعرّف العام الفريد
	WalletId      string    // معرّف المحفظة
	DeviceId      string    // معرّف الجهاز
	SeedEncrypted []byte    // البذرة المشفّرة
	SeedKeyID     string    // معرّف مفتاح KMS
	Counter       int64     // عداد المعاملات
	PayerLimit    int64     // حد الدافع بالوحدة الصغرى
	Status        string    // الحالة: ACTIVE أو SUSPENDED
	UserType      string    // نوع المستخدم: P أو M
	CreatedAt     time.Time // تاريخ الإنشاء
	UpdatedAt     time.Time // تاريخ التحديث
}

// WalletConfig — إعدادات المحفظة في جدول wallet_configs
type WalletConfig struct {
	ID            int64     // المعرّف الداخلي (توليد تلقائي)
	WalletId      string    // معرّف المحفظة الفريد
	BaseURL       string    // عنوان API الأساسي للمحفظة
	APIKey        string    // مفتاح API
	Secret        string    // السر المشترك
	MaxPayerLimit int64     // الحد الأقصى للدافع بالوحدة الصغرى
	TimeoutMs     int       // مهلة الطلب بالملي ثانية
	MaxRetries    int       // عدد إعادة المحاولات
	IsActive      bool      // هل المحفظة مفعّلة
	CreatedAt     time.Time // تاريخ الإنشاء
	UpdatedAt     time.Time // تاريخ التحديث
}

// --- محوّل المحافظ (§6) ---

// DebitParams — معاملات الخصم من محفظة الدافع
type DebitParams struct {
	WalletId       string `json:"walletId"`
	AccountRef     string `json:"accountRef"`
	Amount         int64  `json:"amount"`
	Currency       string `json:"currency"`
	IdempotencyKey string `json:"idempotencyKey"`
}

// CreditParams — معاملات الإيداع في محفظة التاجر
type CreditParams struct {
	WalletId       string `json:"walletId"`
	AccountRef     string `json:"accountRef"`
	Amount         int64  `json:"amount"`
	Currency       string `json:"currency"`
	IdempotencyKey string `json:"idempotencyKey"`
}

// DebitResult — نتيجة الخصم
type DebitResult struct {
	DebitRef string `json:"debitRef"`
	Status   string `json:"status"`
}

// CreditResult — نتيجة الإيداع
type CreditResult struct {
	CreditRef string `json:"creditRef"`
	Status    string `json:"status"`
}

// --- محوّل المحافظ — واجهة موحّدة (§6) ---

// ReverseResult — نتيجة عكس الخصم (تعويض)
type ReverseResult struct {
	ReverseRef string `json:"reverseRef"`
	Status     string `json:"status"`
}

// TxStatus — حالة المعاملة في المحفظة
type TxStatus struct {
	Ref    string `json:"ref"`
	Status string `json:"status"`
}

// WalletAdapter — واجهة محوّل المحفظة (§6)
// كل محفظة (جوالي، فلوسك) تُنفّذ هذه الواجهة
type WalletAdapter interface {
	// VerifyAccessToken — يتحقق من صحة رمز وصول التاجر
	VerifyAccessToken(ctx context.Context, walletId, accessToken string) (bool, error)
	// Debit — يخصم مبلغاً من حساب الدافع
	Debit(ctx context.Context, params DebitParams) (*DebitResult, error)
	// Credit — يودع مبلغاً في حساب التاجر
	Credit(ctx context.Context, params CreditParams) (*CreditResult, error)
	// ReverseDebit — يعكس عملية خصم سابقة (تعويض في Saga)
	ReverseDebit(ctx context.Context, debitRef string) (*ReverseResult, error)
	// QueryTransaction — يستعلم عن حالة معاملة في المحفظة
	QueryTransaction(ctx context.Context, txRef string) (*TxStatus, error)
}

// --- سجل المعاملات (§4) ---

// --- ثوابت مصدر الاتصال ---

const (
	SourceInternet = "internet" // الإنترنت العام
	SourceCarrier  = "carrier"  // شبكة شركة الاتصالات (بدون رسوم بيانات)
)

// ConnectionSourceCtxKey — مفتاح سياق مصدر الاتصال
type ConnectionSourceCtxKey struct{}

// Transaction — سجل المعاملة في جدول transactions
type Transaction struct {
	ID               int64     `json:"id"`
	TransactionId    string    `json:"transactionId"`
	PayerPublicId    string    `json:"payerPublicId"`
	MerchantId       string    `json:"merchantId"`
	PayerWalletId    string    `json:"payerWalletId"`
	MerchantWalletId string    `json:"merchantWalletId"`
	Amount           int64     `json:"amount"`
	Currency         string    `json:"currency"`
	Counter          int64     `json:"counter"`
	Status           string    `json:"status"`
	ErrorCode        string    `json:"errorCode"`
	DurationMs       int       `json:"durationMs"`
	DebitRef         string    `json:"debitRef"`
	CreditRef        string    `json:"creditRef"`
	ConnectionSource string    `json:"connectionSource"`
	CreatedAt        time.Time `json:"createdAt"`
}

// TransactionFilters — معاملات التصفية لقائمة المعاملات
type TransactionFilters struct {
	PayerPublicId    string    // تصفية حسب الدافع
	MerchantId       string    // تصفية حسب التاجر
	Status           string    // تصفية حسب الحالة
	WalletId         string    // تصفية حسب المحفظة
	ConnectionSource string    // تصفية حسب مصدر الاتصال: internet أو carrier
	FromDate         time.Time // من تاريخ
	ToDate           time.Time // إلى تاريخ
}

// --- إحصائيات العمولات ---

// CarrierCommissionStats — إحصائيات عمولات شركة الاتصالات لكل محفظة
type CarrierCommissionStats struct {
	WalletId       string `json:"walletId"`
	TotalTxCount   int    `json:"totalTxCount"`
	TotalAmount    int64  `json:"totalAmount"`
	SuccessCount   int    `json:"successCount"`
	FailedCount    int    `json:"failedCount"`
	CommissionRate int64  `json:"commissionRate"`
	CommissionDue  int64  `json:"commissionDue"`
}

// CarrierCommissionSummary — ملخص إحصائيات العمولات
type CarrierCommissionSummary struct {
	Wallets      []CarrierCommissionStats `json:"wallets"`
	TotalTxCount int                      `json:"totalTxCount"`
	TotalAmount  int64                    `json:"totalAmount"`
	TotalDue     int64                    `json:"totalDue"`
}

// --- المستخدمون الإداريون (§4) ---

// AdminUser — مستخدم إداري في جدول admin_users
type AdminUser struct {
	ID           int64      // المعرّف الداخلي (توليد تلقائي)
	Email        string     // البريد الإلكتروني
	PasswordHash string     // تجزئة كلمة المرور
	TOTPSecret   string     // سر المصادقة الثنائية
	Role         string     // الدور: SUPER_ADMIN أو ADMIN أو WALLET_ADMIN أو VIEWER
	Scope        string     // نطاق الصلاحيات
	IsActive     bool       // هل الحساب مفعّل
	LastLoginAt  *time.Time // آخر تسجيل دخول
	CreatedAt    time.Time  // تاريخ الإنشاء
	UpdatedAt    time.Time  // تاريخ التحديث
}

// --- أدوار الإدارة وصلاحياتها ---

const (
	RoleSuperAdmin  = "SUPER_ADMIN"   // مدير أعلى — صلاحية كاملة
	RoleAdmin       = "ADMIN"         // مدير — صلاحية كاملة ما عدا إدارة المديرين
	RoleWalletAdmin = "WALLET_ADMIN" // مدير محفظة — يرى بيانات محفظته فقط
	RoleViewer      = "VIEWER"       // مشاهد — قراءة فقط
)

// CanAccess — يتحقق من أن الدور يملك مستوى الصلاحية المطلوب
func CanAccess(userRole, requiredRole string) bool {
	roleLevel := map[string]int{
		RoleSuperAdmin:  4,
		RoleAdmin:       3,
		RoleWalletAdmin: 2,
		RoleViewer:      1,
	}
	return roleLevel[userRole] >= roleLevel[requiredRole]
}

// --- تقارير التسوية (§4) ---

// ReconciliationReport — تقرير تسوية يومي بين السويتش والمحافظ
type ReconciliationReport struct {
	ID            int64     // المعرّف الداخلي (توليد تلقائي)
	ReportDate    string    // تاريخ التقرير بصيغة YYYY-MM-DD
	WalletId      string    // معرّف المحفظة
	TotalTxCount  int       // إجمالي عدد المعاملات
	TotalAmount   int64     // إجمالي المبلغ بالوحدة الصغرى
	SuccessCount  int       // عدد المعاملات الناجحة
	FailedCount   int       // عدد المعاملات الفاشلة
	DisputedCount int       // عدد المعاملات المتنازع عليها
	Status        string    // الحالة: PENDING أو VERIFIED أو DISPUTED أو RESOLVED
	Notes         string    // ملاحظات
	CreatedAt     time.Time // تاريخ الإنشاء
	UpdatedAt     time.Time // تاريخ التحديث
}
