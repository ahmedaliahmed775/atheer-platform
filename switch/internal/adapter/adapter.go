// واجهة محوّل المحافظ — تعريف موحّد لجميع محافظ الدفع
// يُرجى الرجوع إلى SPEC §6 و skills/adapter.md
// الواجهة معرّفة في model/types.go — هذا الملف يُعيد تصديرها للراحة
package adapter

import (
	"github.com/atheer/switch/internal/model"
)

// WalletAdapter — واجهة محوّل المحفظة
// كل محفظة (جوالي، فلوسك، إلخ) تُنفّذ هذه الواجهة
// الواجهة الأصلية معرّفة في model.WalletAdapter
type WalletAdapter = model.WalletAdapter
