// أنواع البيانات الأساسية — يُرجى الرجوع إلى SPEC §1 و§3 و§4 و§6
package model

import (
	"context"
	"time"
)

// --- توكن الدفع وبيانات التاجر ---

// PaymentToken — التوكن المُستقبَل من جهاز الدافع عبر NFC
type PaymentToken struct {
	PublicId  string // المعرّف العام للدافع مثل usr_abc123
	DeviceId  string // معرّف الجهاز مثل dev_456
	Counter   int64  // عداد المعاملات (رقم تصاعدي)
	Timestamp int64  // الطابع الزمني بالثواني (Unix)
	HMAC      string // توقيع HMAC-SHA256 بصيغة base64
}

// MerchantData — بيانات التاجر المُرسلة من SDK التاجر
type MerchantData struct {
	MerchantId       string // معرّف التاجر (رقم الحساب)
	MerchantWalletId string // معرّف محفظة التاجر مثل jawali
	Amount           int64  // المبلغ بالوحدة الصغرى (لا float)
	Currency         string // العملة مثل YER
	AccessToken      string // رمز وصول التاجر للمصادقة
}

// --- طلب واستجابة المعاملة ---

// TransactionRequest — طلب المعاملة المُرسل من SDK التاجر
type TransactionRequest struct {
	PaymentToken  PaymentToken  // توكن الدفع من NFC
	MerchantData  MerchantData  // بيانات التاجر
	Timestamp     int64         // الطابع الزمني للطلب بالثواني (Unix)
}

// TransactionResponse — استجابة المعاملة المُعادة للتاجر
type TransactionResponse struct {
	TransactionId    string // معرّف المعاملة (UUID)
	Status           string // حالة المعاملة: SUCCESS أو FAILED
	ErrorCode        string // رمز الخطأ إن وُجد مثل HMAC_MISMATCH
	ErrorMessage     string // رسالة الخطأ
	LastValidCounter int64  // آخر عداد صالح
	Timestamp        int64  // الطابع الزمني للاستجابة بالثواني (Unix)
}

// --- التسجيل (Enrollment) ---

// EnrollRequest — طلب تسجيل جهاز دافع جديد
type EnrollRequest struct {
	WalletId    string // معرّف المحفظة مثل jawali
	WalletToken string // رمز المصادقة من خادم المحفظة
	UserType    string // نوع المستخدم: P (دافع) أو M (تاجر)
	DeviceId    string // معرّف الجهاز
	PublicKey   string // المفتاح العام بصيغة base64
	Attestation string // شهادة التوثيق بصيغة base64
}

// EnrollResponse — استجابة التسجيل
type EnrollResponse struct {
	PublicId         string // المعرّف العام المُنشأ مثل usr_abc123
	EncryptedSeed    string // البذرة المشفّرة بصيغة base64
	PayerLimit       int64  // حد الدافع الافتراضي بالوحدة الصغرى
	MaxPayerLimit    int64  // الحد الأقصى للدافع بالوحدة الصغرى
	AttestationLevel string // مستوى التوثيق: TEE أو STRONGBOX أو SOFTWARE
	Status           string // الحالة: ACTIVE
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
	PayerWalletId   string // معرّف محفظة الدافع
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
	WalletId      string // معرّف المحفظة
	AccountRef    string // مرجع الحساب (معرّف الدافع)
	Amount        int64  // المبلغ بالوحدة الصغرى
	Currency      string // العملة
	IdempotencyKey string // مفتاح عدم التكرار
}

// CreditParams — معاملات الإيداع في محفظة التاجر
type CreditParams struct {
	WalletId      string // معرّف المحفظة
	AccountRef    string // مرجع الحساب (معرّف التاجر)
	Amount        int64  // المبلغ بالوحدة الصغرى
	Currency      string // العملة
	IdempotencyKey string // مفتاح عدم التكرار
}

// DebitResult — نتيجة الخصم
type DebitResult struct {
	DebitRef string // مرجع عملية الخصم
	Status   string // حالة العملية
}

// CreditResult — نتيجة الإيداع
type CreditResult struct {
	CreditRef string // مرجع عملية الإيداع
	Status    string // حالة العملية
}

// --- محوّل المحافظ — واجهة موحّدة (§6) ---

// ReverseResult — نتيجة عكس الخصم (تعويض)
type ReverseResult struct {
	ReverseRef string // مرجع عملية العكس
	Status     string // حالة العملية
}

// TxStatus — حالة المعاملة في المحفظة
type TxStatus struct {
	Ref    string // مرجع المعاملة
	Status string // حالة المعاملة
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

// Transaction — سجل المعاملة في جدول transactions
type Transaction struct {
	ID               int64     // المعرّف الداخلي (توليد تلقائي)
	TransactionId    string    // معرّف المعاملة (UUID)
	PayerPublicId    string    // المعرّف العام للدافع
	MerchantId       string    // معرّف التاجر
	PayerWalletId    string    // معرّف محفظة الدافع
	MerchantWalletId string    // معرّف محفظة التاجر
	Amount           int64     // المبلغ بالوحدة الصغرى
	Currency         string    // العملة
	Counter          int64     // عداد المعاملة
	Status           string    // الحالة: SUCCESS أو FAILED أو PENDING أو REVERSED
	ErrorCode        string    // رمز الخطأ إن وُجد
	DurationMs       int       // مدة التنفيذ بالملي ثانية
	DebitRef         string    // مرجع الخصم من المحفظة
	CreditRef        string    // مرجع الإيداع في المحفظة
	CreatedAt        time.Time // تاريخ الإنشاء
}

// TransactionFilters — معاملات التصفية لقائمة المعاملات
type TransactionFilters struct {
	PayerPublicId string    // تصفية حسب الدافع
	MerchantId    string    // تصفية حسب التاجر
	Status        string    // تصفية حسب الحالة
	WalletId      string    // تصفية حسب المحفظة
	FromDate      time.Time // من تاريخ
	ToDate        time.Time // إلى تاريخ
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
	RoleSuperAdmin = "SUPER_ADMIN" // مدير أعلى — صلاحية كاملة
	RoleAdmin      = "ADMIN"       // مدير — صلاحية كاملة ما عدا إدارة المديرين
	RoleWalletAdmin = "WALLET_ADMIN" // مدير محفظة — يرى بيانات محفظته فقط
	RoleViewer     = "VIEWER"      // مشاهد — قراءة فقط
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
