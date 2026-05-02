// سجل محوّلات المحافظ — خريطة walletId → WalletAdapter
// يُرجى الرجوع إلى skills/adapter.md — AdapterRegistry
package adapter

import (
	"fmt"
	"sync"
)

// AdapterRegistry — سجل محوّلات المحافظ
// يربط كل معرّف محفظة (مثل jawali) بمحوّلها المناسب
type AdapterRegistry struct {
	mu       sync.RWMutex               // قفل للقراءة/الكتابة المتزامنة
	adapters map[string]WalletAdapter   // خريطة walletId → محوّل
}

// NewAdapterRegistry — ينشئ سجل محوّلات فارغ
func NewAdapterRegistry() *AdapterRegistry {
	return &AdapterRegistry{
		adapters: make(map[string]WalletAdapter),
	}
}

// Register — يسجّل محوّل محفظة في السجل
func (r *AdapterRegistry) Register(walletId string, adapter WalletAdapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[walletId] = adapter
}

// Get — يُرجع محوّل المحفظة المطلوب
// يُرجع خطأ إذا كانت المحفظة غير مسجّلة
func (r *AdapterRegistry) Get(walletId string) (WalletAdapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	a, ok := r.adapters[walletId]
	if !ok {
		return nil, fmt.Errorf("سجل المحوّلات: محفظة %q غير مسجّلة", walletId)
	}
	return a, nil
}

// List — يُرجع قائمة بمعرّفات المحافظ المسجّلة
func (r *AdapterRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.adapters))
	for id := range r.adapters {
		ids = append(ids, id)
	}
	return ids
}

// Unregister — يُزيل محوّل محفظة من السجل
func (r *AdapterRegistry) Unregister(walletId string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.adapters, walletId)
}
