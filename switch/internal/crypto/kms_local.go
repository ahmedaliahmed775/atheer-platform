// تنفيذ محلي لنظام إدارة المفاتيح (KMS) باستخدام AES-256-GCM
// للاستخدام في التطوير والاختبار فقط — في الإنتاج استخدم AWS KMS أو GCP KMS أو Vault
package crypto

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

// localKeyID — معرّف المفتاح المحلي الثابت
const localKeyID = "local-v1"

// LocalKMS — تنفيذ محلي لـ KMS باستخدام AES-256-GCM
type LocalKMS struct {
	aead cipher.AEAD // واجهة التشفير المصادق عليه
}

// NewLocalKMS — ينشئ نسخة KMS محلية من المفتاح الرئيسي (32 بايت)
func NewLocalKMS(masterKey []byte) (*LocalKMS, error) {
	if len(masterKey) != 32 {
		return nil, fmt.Errorf("KMS محلي: المفتاح الرئيسي يجب أن يكون 32 بايت، حصلنا على %d", len(masterKey))
	}

	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return nil, fmt.Errorf("KMS محلي: إنشاء شيفرة AES: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("KMS محلي: إنشاء GCM: %w", err)
	}

	return &LocalKMS{aead: aead}, nil
}

// Encrypt — يشفر النص الصريح باستخدام AES-256-GCM
// الصيغة: nonce(12) || ciphertext || tag(16)
// معرّف المفتاح = "local-v1"
func (k *LocalKMS) Encrypt(_ context.Context, plaintext []byte) ([]byte, string, error) {
	// إنشاء nonce عشوائي (12 بايت لـ GCM)
	nonce := make([]byte, k.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, "", fmt.Errorf("KMS محلي: إنشاء nonce: %w", err)
	}

	// التشفير مع إلحاق الشهادة بالنص المشفر
	// النتيجة: nonce || ciphertext || tag
	ciphertext := k.aead.Seal(nonce, nonce, plaintext, nil)

	return ciphertext, localKeyID, nil
}

// Decrypt — يفك تشفير النص المشفر باستخدام AES-256-GCM
// يتوقع الصيغة: nonce(12) || ciphertext || tag(16)
func (k *LocalKMS) Decrypt(_ context.Context, keyID string, ciphertext []byte) ([]byte, error) {
	if keyID != localKeyID {
		return nil, fmt.Errorf("KMS محلي: معرّف مفتاح غير معروف %q", keyID)
	}

	// استخراج nonce من بداية النص المشفر
	nonceSize := k.aead.NonceSize()
	if len(ciphertext) < nonceSize+k.aead.Overhead() {
		return nil, fmt.Errorf("KMS محلي: النص المشفر قصير جداً")
	}

	nonce, ciphertextBody := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// فك التشفير والتحقق من الشهادة
	plaintext, err := k.aead.Open(nil, nonce, ciphertextBody, nil)
	if err != nil {
		return nil, fmt.Errorf("KMS محلي: فك التشفير: %w", err)
	}

	return plaintext, nil
}
