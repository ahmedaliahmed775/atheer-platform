// اختبارات حزمة التشفير — يُرجى الرجوع إلى SPEC §5
package crypto

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"testing"
)

// TestDeriveLUK — اختبار اشتقاق LUK من بذرة معروفة (حتمي)
func TestDeriveLUK(t *testing.T) {
	seed := []byte("test-seed-for-derivation-32bytes!")

	t.Run("حتمية الاشتقاق", func(t *testing.T) {
		// اشتقاق LUK مرتين من نفس البذرة يجب أن يعطي نفس النتيجة
		luk1, err := DeriveLUK(seed)
		if err != nil {
			t.Fatalf("فشل الاشتقاق الأول: %v", err)
		}
		defer Zeroize(luk1)

		luk2, err := DeriveLUK(seed)
		if err != nil {
			t.Fatalf("فشل الاشتقاق الثاني: %v", err)
		}
		defer Zeroize(luk2)

		if len(luk1) != 32 {
			t.Errorf("طول LUK = %d، المتوقع 32", len(luk1))
		}

		if !hmac.Equal(luk1, luk2) {
			t.Error("LUK المشتق من نفس البذرة يجب أن يكون متطابقاً")
		}
	})

	t.Run("بذرة فارغة", func(t *testing.T) {
		_, err := DeriveLUK([]byte{})
		if err == nil {
			t.Error("يجب أن يفشل الاشتقاق من بذرة فارغة")
		}
	})

	t.Run("بذور مختلفة تعطي LUK مختلف", func(t *testing.T) {
		seed2 := make([]byte, 32)
		if _, err := rand.Read(seed2); err != nil {
			t.Fatalf("فشل إنشاء بذرة عشوائية: %v", err)
		}

		luk1, err := DeriveLUK(seed)
		if err != nil {
			t.Fatalf("فشل اشتقاق LUK1: %v", err)
		}
		defer Zeroize(luk1)

		luk2, err := DeriveLUK(seed2)
		if err != nil {
			t.Fatalf("فشل اشتقاق LUK2: %v", err)
		}
		defer Zeroize(luk2)

		if hmac.Equal(luk1, luk2) {
			t.Error("بذور مختلفة يجب أن تعطي LUK مختلف")
		}
	})
}

// TestComputeHMAC — اختبار حساب HMAC-SHA256 بمدخلات معروفة
func TestComputeHMAC(t *testing.T) {
	luk := []byte("32-byte-luk-for-hmac-testing!!!")
	defer Zeroize(luk)

	t.Run("حتمية HMAC", func(t *testing.T) {
		hmac1, err := ComputeHMAC(luk, "usr_abc123", "dev_456", 43, 1714340400)
		if err != nil {
			t.Fatalf("فشل حساب HMAC: %v", err)
		}
		defer Zeroize(hmac1)

		hmac2, err := ComputeHMAC(luk, "usr_abc123", "dev_456", 43, 1714340400)
		if err != nil {
			t.Fatalf("فشل حساب HMAC الثاني: %v", err)
		}
		defer Zeroize(hmac2)

		if !hmac.Equal(hmac1, hmac2) {
			t.Error("HMAC من نفس المدخلات يجب أن يكون متطابقاً")
		}
	})

	t.Run("طول HMAC", func(t *testing.T) {
		result, err := ComputeHMAC(luk, "usr_abc123", "dev_456", 43, 1714340400)
		if err != nil {
			t.Fatalf("فشل حساب HMAC: %v", err)
		}
		defer Zeroize(result)

		if len(result) != sha256.Size {
			t.Errorf("طول HMAC = %d، المتوقع %d", len(result), sha256.Size)
		}
	})

	t.Run("مدخلات مختلفة تعطي HMAC مختلف", func(t *testing.T) {
		hmac1, err := ComputeHMAC(luk, "usr_abc123", "dev_456", 43, 1714340400)
		if err != nil {
			t.Fatalf("فشل حساب HMAC1: %v", err)
		}
		defer Zeroize(hmac1)

		hmac2, err := ComputeHMAC(luk, "usr_abc123", "dev_456", 44, 1714340400)
		if err != nil {
			t.Fatalf("فشل حساب HMAC2: %v", err)
		}
		defer Zeroize(hmac2)

		if hmac.Equal(hmac1, hmac2) {
			t.Error("عداد مختلف يجب أن يعطي HMAC مختلف")
		}
	})

	t.Run("LUK فارغ", func(t *testing.T) {
		_, err := ComputeHMAC([]byte{}, "usr_abc123", "dev_456", 43, 1714340400)
		if err == nil {
			t.Error("يجب أن يفشل حساب HMAC بمفتاح فارغ")
		}
	})
}

// TestVerifyHMAC_Valid — التحقق من HMAC صحيح يعيد true
func TestVerifyHMAC_Valid(t *testing.T) {
	luk := []byte("32-byte-luk-for-hmac-testing!!!")
	defer Zeroize(luk)

	publicId := "usr_abc123"
	deviceId := "dev_456"
	counter := int64(43)
	timestamp := int64(1714340400)

	// حساب HMAC الصحيح
	computed, err := ComputeHMAC(luk, publicId, deviceId, counter, timestamp)
	if err != nil {
		t.Fatalf("فشل حساب HMAC: %v", err)
	}
	defer Zeroize(computed)

	// التحقق يجب أن ينجح
	valid, err := VerifyHMAC(luk, publicId, deviceId, counter, timestamp, computed)
	if err != nil {
		t.Fatalf("فشل التحقق من HMAC: %v", err)
	}

	if !valid {
		t.Error("HMAC المتطابق يجب أن يعيد true")
	}
}

// TestVerifyHMAC_Invalid — التحقق من HMAC خاطئ يعيد false
func TestVerifyHMAC_Invalid(t *testing.T) {
	luk := []byte("32-byte-luk-for-hmac-testing!!!")
	defer Zeroize(luk)

	publicId := "usr_abc123"
	deviceId := "dev_456"
	counter := int64(43)
	timestamp := int64(1714340400)

	t.Run("HMAC خاطئ تماماً", func(t *testing.T) {
		wrongHMAC := make([]byte, 32)
		if _, err := rand.Read(wrongHMAC); err != nil {
			t.Fatalf("فشل إنشاء HMAC عشوائي: %v", err)
		}
		defer Zeroize(wrongHMAC)

		valid, err := VerifyHMAC(luk, publicId, deviceId, counter, timestamp, wrongHMAC)
		if err != nil {
			t.Fatalf("فشل التحقق: %v", err)
		}

		if valid {
			t.Error("HMAC خاطئ يجب أن يعيد false")
		}
	})

	t.Run("عداد مختلف", func(t *testing.T) {
		// حساب HMAC بالعداد الصحيح
		computed, err := ComputeHMAC(luk, publicId, deviceId, counter, timestamp)
		if err != nil {
			t.Fatalf("فشل حساب HMAC: %v", err)
		}
		defer Zeroize(computed)

		// التحقق بالعداد الخاطئ
		valid, err := VerifyHMAC(luk, publicId, deviceId, 999, timestamp, computed)
		if err != nil {
			t.Fatalf("فشل التحقق: %v", err)
		}

		if valid {
			t.Error("HMAC مع عداد مختلف يجب أن يعيد false")
		}
	})

	t.Run("معرّف دافع مختلف", func(t *testing.T) {
		computed, err := ComputeHMAC(luk, publicId, deviceId, counter, timestamp)
		if err != nil {
			t.Fatalf("فشل حساب HMAC: %v", err)
		}
		defer Zeroize(computed)

		valid, err := VerifyHMAC(luk, "usr_WRONG", deviceId, counter, timestamp, computed)
		if err != nil {
			t.Fatalf("فشل التحقق: %v", err)
		}

		if valid {
			t.Error("HMAC مع معرّف دافع مختلف يجب أن يعيد false")
		}
	})
}

// TestKMSEncryptDecrypt — اختبار دورة التشفير وفك التشفير
func TestKMSEncryptDecrypt(t *testing.T) {
	masterKey := make([]byte, 32)
	if _, err := rand.Read(masterKey); err != nil {
		t.Fatalf("فشل إنشاء المفتاح الرئيسي: %v", err)
	}

	kms, err := NewLocalKMS(masterKey)
	if err != nil {
		t.Fatalf("فشل إنشاء KMS محلي: %v", err)
	}

	ctx := context.Background()

	t.Run("دورة التشفير وفك التشفير", func(t *testing.T) {
		plaintext := []byte("هذه بذرة سرية للاختبار - 32 bytes!!")

		// التشفير
		ciphertext, keyID, err := kms.Encrypt(ctx, plaintext)
		if err != nil {
			t.Fatalf("فشل التشفير: %v", err)
		}

		// معرّف المفتاح يجب أن يكون local-v1
		if keyID != "local-v1" {
			t.Errorf("معرّف المفتاح = %q، المتوقع %q", keyID, "local-v1")
		}

		// النص المشفر يجب أن يختلف عن الأصلي
		if hmac.Equal(ciphertext[12:], plaintext) {
			t.Error("النص المشفر يجب أن يختلف عن الأصلي")
		}

		// فك التشفير
		decrypted, err := kms.Decrypt(ctx, keyID, ciphertext)
		if err != nil {
			t.Fatalf("فشل فك التشفير: %v", err)
		}

		// النص المفكوك يجب أن يطابق الأصلي
		if !hmac.Equal(decrypted, plaintext) {
			t.Error("النص المفكوك لا يطابق الأصلي")
		}
	})

	t.Run("تشفير نفس البيانات مرتين يعطي نصوص مشفرة مختلفة (nonce مختلف)", func(t *testing.T) {
		plaintext := []byte("بيانات اختبار متطابقة")

		ct1, _, err := kms.Encrypt(ctx, plaintext)
		if err != nil {
			t.Fatalf("فشل التشفير الأول: %v", err)
		}

		ct2, _, err := kms.Encrypt(ctx, plaintext)
		if err != nil {
			t.Fatalf("فشل التشفير الثاني: %v", err)
		}

		if hmac.Equal(ct1, ct2) {
			t.Error("تشفير نفس البيانات مرتين يجب أن يعطي نصوصاً مشفرة مختلفة بسبب nonce عشوائي")
		}
	})

	t.Run("معرّف مفتاح خاطئ", func(t *testing.T) {
		plaintext := []byte("بيانات اختبار")
		ciphertext, _, err := kms.Encrypt(ctx, plaintext)
		if err != nil {
			t.Fatalf("فشل التشفير: %v", err)
		}

		_, err = kms.Decrypt(ctx, "wrong-key-id", ciphertext)
		if err == nil {
			t.Error("فك التشفير بمعرّف مفتاح خاطئ يجب أن يفشل")
		}
	})

	t.Run("نص مشفر تالف", func(t *testing.T) {
		plaintext := []byte("بيانات اختبار")
		ciphertext, keyID, err := kms.Encrypt(ctx, plaintext)
		if err != nil {
			t.Fatalf("فشل التشفير: %v", err)
		}

		// تلفيب النص المشفر
		ciphertext[len(ciphertext)-1] ^= 0xFF

		_, err = kms.Decrypt(ctx, keyID, ciphertext)
		if err == nil {
			t.Error("فك تشفير نص تالف يجب أن يفشل")
		}
	})

	t.Run("مفتاح رئيسي بطول خاطئ", func(t *testing.T) {
		_, err := NewLocalKMS([]byte("short-key"))
		if err == nil {
			t.Error("إنشاء KMS بمفتاح قصير يجب أن يفشل")
		}
	})
}

// TestZeroize — اختبار تصفير البيانات الحساسة
func TestZeroize(t *testing.T) {
	t.Run("تصفير شريحة بيانات", func(t *testing.T) {
		data := []byte("بيانات سرية يجب تصفيرها")
		Zeroize(data)

		for i, b := range data {
			if b != 0 {
				t.Errorf("البايت %d = %d، المتوقع 0 بعد التصفير", i, b)
			}
		}
	})

	t.Run("تصفير شريحة فارغة", func(t *testing.T) {
		data := []byte{}
		Zeroize(data) // يجب ألا يُسبب panic
	})

	t.Run("تصفير مفتاح 32 بايت", func(t *testing.T) {
		key := make([]byte, 32)
		for i := range key {
			key[i] = 0xFF
		}
		Zeroize(key)

		allZero := true
		for _, b := range key {
			if b != 0 {
				allZero = false
				break
			}
		}

		if !allZero {
			t.Error("المفتاح يجب أن يكون كله أصفاراً بعد التصفير")
		}
	})
}
