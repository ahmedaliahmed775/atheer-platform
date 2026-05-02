// محوّل التوجيه — يوجّه الاستدعاءات إلى المحوّل المناسب حسب walletId
// يُنفّذ واجهة WalletAdapter ويفحص معاملات الخصم/الإيداع لتحديد المحوّل
package adapter

import (
	"context"
	"fmt"

	"github.com/atheer/switch/internal/model"
)

// DispatchingAdapter — محوّل يوجّه الاستدعاءات إلى المحوّل المناسب من السجل
// يُنفّذ واجهة WalletAdapter ليُمرَّر إلى ExecuteService
type DispatchingAdapter struct {
	registry *AdapterRegistry // سجل المحوّلات
}

// NewDispatchingAdapter — ينشئ محوّل توجيه جديد
func NewDispatchingAdapter(registry *AdapterRegistry) *DispatchingAdapter {
	return &DispatchingAdapter{registry: registry}
}

// VerifyAccessToken — يوجّه التحقق من الرمز إلى المحوّل المناسب
func (d *DispatchingAdapter) VerifyAccessToken(ctx context.Context, walletId, accessToken string) (bool, error) {
	a, err := d.registry.Get(walletId)
	if err != nil {
		return false, fmt.Errorf("محوّل التوجيه: التحقق من الرمز: %w", err)
	}
	return a.VerifyAccessToken(ctx, walletId, accessToken)
}

// Debit — يوجّه الخصم إلى المحوّل المناسب حسب WalletId في المعاملات
func (d *DispatchingAdapter) Debit(ctx context.Context, params model.DebitParams) (*model.DebitResult, error) {
	a, err := d.registry.Get(params.WalletId)
	if err != nil {
		return nil, fmt.Errorf("محوّل التوجيه: الخصم: %w", err)
	}
	return a.Debit(ctx, params)
}

// Credit — يوجّه الإيداع إلى المحوّل المناسب حسب WalletId في المعاملات
func (d *DispatchingAdapter) Credit(ctx context.Context, params model.CreditParams) (*model.CreditResult, error) {
	a, err := d.registry.Get(params.WalletId)
	if err != nil {
		return nil, fmt.Errorf("محوّل التوجيه: الإيداع: %w", err)
	}
	return a.Credit(ctx, params)
}

// ReverseDebit — يوجّه عكس الخصم إلى المحوّل الأول المتاح
// ملاحظة: يحتاج معرّف المحفظة لتحديد المحوّل المناسب
// في حالات عكس الخصم، يكون debitRef مرتبطاً بمحوّل محدد
func (d *DispatchingAdapter) ReverseDebit(ctx context.Context, debitRef string) (*model.ReverseResult, error) {
	// البحث عن المحوّل المناسب — نجرّب كل المحوّلات المسجّلة
	// في الإصدار الأول، نفترض محوّل واحد (جوالي)
	ids := d.registry.List()
	if len(ids) == 0 {
		return nil, fmt.Errorf("محوّل التوجيه: عكس الخصم: لا توجد محوّلات مسجّلة")
	}

	// نستخدم أول محوّل متاح — في الإصدار الأول محفظة واحدة فقط
	a, err := d.registry.Get(ids[0])
	if err != nil {
		return nil, fmt.Errorf("محوّل التوجيه: عكس الخصم: %w", err)
	}
	return a.ReverseDebit(ctx, debitRef)
}

// QueryTransaction — يوجّه الاستعلام إلى المحوّل الأول المتاح
func (d *DispatchingAdapter) QueryTransaction(ctx context.Context, txRef string) (*model.TxStatus, error) {
	ids := d.registry.List()
	if len(ids) == 0 {
		return nil, fmt.Errorf("محوّل التوجيه: استعلام المعاملة: لا توجد محوّلات مسجّلة")
	}

	a, err := d.registry.Get(ids[0])
	if err != nil {
		return nil, fmt.Errorf("محوّل التوجيه: استعلام المعاملة: %w", err)
	}
	return a.QueryTransaction(ctx, txRef)
}
