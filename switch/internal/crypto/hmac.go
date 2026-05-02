// حساب والتحقق من HMAC-SHA256 لتوكن الدفع
// يُرجى الرجوع إلى SPEC §5 — HMAC-SHA256(LUK, "publicId|deviceId|counter|timestamp")
package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
)

// ComputeHMAC — يحسب توقيع HMAC-SHA256 لتوكن الدفع
// البيانات بصيغة: "publicId|deviceId|counter|timestamp" (مفصولة بأنبوب)
// المُستدعي MUST يُصفّر النتيجة بعد الاستخدام عبر Zeroize()
func ComputeHMAC(luk []byte, publicId, deviceId string, counter, timestamp int64) ([]byte, error) {
	if len(luk) == 0 {
		return nil, fmt.Errorf("حساب HMAC: مفتاح LUK فارغ")
	}

	// بناء البيانات بصيغة الأنابيب حسب SPEC
	data := fmt.Sprintf("%s|%s|%d|%d", publicId, deviceId, counter, timestamp)

	// حساب HMAC-SHA256
	mac := hmac.New(sha256.New, luk)
	mac.Write([]byte(data))
	return mac.Sum(nil), nil
}

// VerifyHMAC — يتحقق من صحة توقيع HMAC المُقدَّم
// يستخدم hmac.Equal للمقارنة بوقت ثابت لمنع هجمات التوقيت
func VerifyHMAC(luk []byte, publicId, deviceId string, counter, timestamp int64, providedHMAC []byte) (bool, error) {
	// حساب التوقيع المتوقع
	expectedHMAC, err := ComputeHMAC(luk, publicId, deviceId, counter, timestamp)
	if err != nil {
		return false, fmt.Errorf("التحقق من HMAC: %w", err)
	}
	defer Zeroize(expectedHMAC)

	// مقارنة بوقت ثابت لمنع هجمات التوقيت
	return hmac.Equal(expectedHMAC, providedHMAC), nil
}
