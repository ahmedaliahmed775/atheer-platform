// تصفير البيانات الحساسة من الذاكرة
// يُرجى الرجوع إلى SPEC §5 — البذرة وLUK يجب تصفيرهما بعد الاستخدام
package crypto

// Zeroize — يُصفّر شريحة البايتات في الذاكرة
// يستخدم clear() المدمج (Go 1.21+) لضمان التصفير
// استخدم: defer Zeroize(seed) و defer Zeroize(luk) في كل مكان
func Zeroize(b []byte) {
	clear(b)
}
