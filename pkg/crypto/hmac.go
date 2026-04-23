// Modified for v3.0 Document Alignment
// معادلة التشفير حسب القسم 4 — الخطوة 3 (مع إبقاء LUK لقوة أمنية أعلى)
//
// LUK = HMAC-SHA256(Seed, Counter)
// Token = HMAC-SHA256(LUK, Amount || ReceiverID || PayeeType || WalletID || Counter)
package crypto

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/binary"
    "fmt"
)

// DeriveLUK — اشتقاق مفتاح الاستخدام المحدود
// LUK = HMAC-SHA256(Seed, Counter)
func DeriveLUK(seed []byte, counter uint64) []byte {
    ctrBytes := make([]byte, 8)
    binary.BigEndian.PutUint64(ctrBytes, counter)
    mac := hmac.New(sha256.New, seed)
    mac.Write(ctrBytes)
    return mac.Sum(nil)
}

// GenerateTransactionHMAC — إنشاء توقيع المعاملة
// Token = HMAC-SHA256(LUK, Amount || ReceiverID || PayeeType || WalletID || Counter)
func GenerateTransactionHMAC(seed []byte, amount int64, receiverID string,
    payeeType string, walletID string, counter uint64) []byte {

    luk := DeriveLUK(seed, counter)
    data := fmt.Sprintf("%d|%s|%s|%s|%d", amount, receiverID, payeeType, walletID, counter)
    mac := hmac.New(sha256.New, luk)
    mac.Write([]byte(data))
    return mac.Sum(nil)
}

// VerifyTransactionHMAC — التحقق من توقيع المعاملة
// السويتش يستخرج Seed و WalletID من SwitchRecord (لا من الحزمة)
func VerifyTransactionHMAC(seed []byte, amount int64, receiverID string,
    payeeType string, walletID string, counter uint64, hmacBytes []byte) error {

    expected := GenerateTransactionHMAC(seed, amount, receiverID, payeeType, walletID, counter)
    if !hmac.Equal(expected, hmacBytes) {
        return fmt.Errorf("HMAC signature mismatch")
    }
    return nil
}

// TimingSafeEqual — مقارنة آمنة زمنياً
func TimingSafeEqual(a, b []byte) bool {
    return hmac.Equal(a, b)
}
