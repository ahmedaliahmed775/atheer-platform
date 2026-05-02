// واجهة نظام إدارة المفاتيح (KMS)
// يُرجى الرجوع إلى SPEC §5 — تشفير مغلّف (envelope encryption) للبذور في قاعدة البيانات
package crypto

import "context"

// KMS — واجهة نظام إدارة المفاتيح للتشفير وفك التشفير
// في الإنتاج: استبدل بـ AWS KMS أو GCP KMS أو Vault
type KMS interface {
	// Encrypt — يشفر النص الصريح ويعيد النص المشفر ومعرّف المفتاح
	Encrypt(ctx context.Context, plaintext []byte) (ciphertext []byte, keyID string, err error)

	// Decrypt — يفك تشفير النص المشفر باستخدام معرّف المفتاح
	Decrypt(ctx context.Context, keyID string, ciphertext []byte) (plaintext []byte, err error)
}
