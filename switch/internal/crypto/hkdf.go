// اشتقاق مفتاح الاستخدام المحدود (LUK) باستخدام HKDF-SHA256
// يُرجى الرجوع إلى SPEC §5 — LUK = HKDF-SHA256(ikm=seed, salt=nil, info="ATHEER-LUK", len=32)
package crypto

import (
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

// lukInfo — معلومات السياق لاشتقاق LUK (ثابت حسب SPEC)
var lukInfo = []byte("ATHEER-LUK")

// lukLen — طول مفتاح LUK بالبايت (32 بايت = 256 بت)
const lukLen = 32

// DeriveLUK — يشتق مفتاح الاستخدام المحدود من البذرة
// LUK = HKDF-SHA256(ikm=seed, salt=nil, info="ATHEER-LUK", len=32)
// المُستدعي MUST يُصفّر LUK بعد الاستخدام عبر Zeroize()
func DeriveLUK(seed []byte) ([]byte, error) {
	if len(seed) == 0 {
		return nil, fmt.Errorf("اشتقاق LUK: البذرة فارغة")
	}

	// إنشاء قارئ HKDF — salt=nil حسب SPEC
	reader := hkdf.New(sha256.New, seed, nil, lukInfo)

	// قراءة 32 بايت من المشتق
	luk := make([]byte, lukLen)
	if _, err := io.ReadFull(reader, luk); err != nil {
		return nil, fmt.Errorf("اشتقاق LUK: قراءة المشتق: %w", err)
	}

	return luk, nil
}
