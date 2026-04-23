// Modified for v3.0 Document Alignment
package model

// UserType represents user classification — determined exclusively by Switch
type UserType string

const (
    UserTypePersonal UserType = "P" // عميل
    UserTypeMerchant UserType = "M" // تاجر
)

// SwitchRecord — السجل الداخلي لكل مستخدم في السويتش
// حسب القسم 4 — المرحلة الأولى من الوثيقة المرجعية
type SwitchRecord struct {
    PublicID  string   `json:"publicId"  db:"public_id"`   // معرّف عام غير مرتبط بهوية
    Seed      []byte   `json:"-"         db:"seed"`         // البذرة التشفيرية (HSM)
    UserID    string   `json:"userId"    db:"user_id"`      // معرّف المستخدم في المحفظة
    UserType  UserType `json:"userType"  db:"user_type"`    // P | M
    WalletID  string   `json:"walletId"  db:"wallet_id"`    // معرّف المحفظة
    Counter   uint64   `json:"counter"   db:"counter"`      // العداد التصاعدي
    Status    string   `json:"status"    db:"status"`       // ACTIVE | SUSPENDED
}
