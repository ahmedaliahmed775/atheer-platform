// تحديد نوع المعاملة — مباشر أو بين محافظ مختلفة
// يُرجى الرجوع إلى SPEC §3 Layer 3
package execute

// TransactionType — نوع المعاملة
type TransactionType int

const (
	// DIRECT — معاملة مباشرة داخل نفس المحفظة (الدافع والتاجر في نفس المحفظة)
	DIRECT TransactionType = iota

	// CROSS_WALLET — معاملة بين محافظ مختلفة (غير مدعومة في الإصدار الأول)
	CROSS_WALLET
)

// resolveTransactionType — يحدد نوع المعاملة بناءً على محفظة الدافع والتاجر
// إذا كانت المحفظتان متطابقتين → DIRECT
// إذا كانتا مختلفتين → CROSS_WALLET
func resolveTransactionType(payerWalletId, merchantWalletId string) TransactionType {
	if payerWalletId == merchantWalletId {
		return DIRECT
	}
	return CROSS_WALLET
}
