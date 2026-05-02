// قاطع الدائرة — نمط حماية من الفشل المتكرر
// يُرجى الرجوع إلى skills/adapter.md — Circuit Breaker
// الحالات: CLOSED (طبيعي) → OPEN (مرفوض) → HALF_OPEN (تجربة واحدة)
package adapter

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// State — حالة قاطع الدائرة
type State int

const (
	// Closed — الحالة الطبيعية: الطلبات تمرّ
	Closed State = iota

	// Open — الحالة المفتوحة: الطلبات مرفوضة فوراً
	Open

	// HalfOpen — حالة التجربة: يُسمح بطلب واحد للاختبار
	HalfOpen
)

// String — تمثيل نصي للحالة
func (s State) String() string {
	switch s {
	case Closed:
		return "CLOSED"
	case Open:
		return "OPEN"
	case HalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreaker — قاطع دائرة بسيط لحماية محوّلات المحافظ
// بعد N فشل متتالي → ينتقل إلى OPEN (يرفض الطلبات)
// بعد مهلة زمنية → ينتقل إلى HALF_OPEN (يسمح بطلب واحد)
// عند النجاح → يعود إلى CLOSED
type CircuitBreaker struct {
	mu             sync.Mutex
	name           string        // اسم المحفظة
	state          State         // الحالة الحالية
	failureCount   int           // عدد الفشل المتتالي
	failureThreshold int         // حد الفشل قبل الانتقال إلى OPEN
	timeout        time.Duration // مدة الانتظار قبل HALF_OPEN
	lastFailure    time.Time     // وقت آخر فشل
	successCount   int           // عدد النجاح المتتالي (للتسجيل)
}

// NewCircuitBreaker — ينشئ قاطع دائرة جديد
// failureThreshold: عدد الفشل المتتالي قبل الانتقال إلى OPEN (افتراضي 5)
// timeout: مدة الانتظار قبل الانتقال إلى HALF_OPEN (افتراضي 30 ثانية)
func NewCircuitBreaker(name string, failureThreshold int, timeout time.Duration) *CircuitBreaker {
	if failureThreshold <= 0 {
		failureThreshold = 5
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &CircuitBreaker{
		name:             name,
		state:            Closed,
		failureThreshold: failureThreshold,
		timeout:          timeout,
	}
}

// Allow — يتحقق هل يُسمح بتمرير الطلب
// يُرجع خطأ إذا كانت الحالة OPEN
func (cb *CircuitBreaker) Allow() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case Closed:
		return nil

	case Open:
		// التحقق هل انتهت المهلة
		if time.Since(cb.lastFailure) >= cb.timeout {
			slog.Info("قاطع الدائرة: انتقال إلى HALF_OPEN",
				"name", cb.name,
				"failures", cb.failureCount,
			)
			cb.state = HalfOpen
			return nil // السماح بطلب واحد
		}
		return fmt.Errorf("قاطع الدائرة: المحفظة %q في حالة OPEN — الطلبات مرفوضة", cb.name)

	case HalfOpen:
		// السماح بطلب واحد فقط
		return nil

	default:
		return nil
	}
}

// RecordSuccess — يسجّل نجاح الطلب
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount = 0
	cb.successCount++

	if cb.state == HalfOpen {
		slog.Info("قاطع الدائرة: انتقال إلى CLOSED بعد نجاح التجربة",
			"name", cb.name,
		)
		cb.state = Closed
	}
}

// RecordFailure — يسجّل فشل الطلب
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailure = time.Now()
	cb.successCount = 0

	if cb.state == HalfOpen {
		// فشل في حالة التجربة → العودة إلى OPEN
		slog.Warn("قاطع الدائرة: انتقال إلى OPEN بعد فشل التجربة",
			"name", cb.name,
		)
		cb.state = Open
		return
	}

	if cb.failureCount >= cb.failureThreshold {
		slog.Warn("قاطع الدائرة: انتقال إلى OPEN بعد تجاوز حد الفشل",
			"name", cb.name,
			"failures", cb.failureCount,
			"threshold", cb.failureThreshold,
		)
		cb.state = Open
	}
}

// State — يُرجع الحالة الحالية
func (cb *CircuitBreaker) State() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// FailureCount — يُرجع عدد الفشل المتتالي الحالي
func (cb *CircuitBreaker) FailureCount() int {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.failureCount
}

// Reset — يُعيد تعيين قاطع الدائرة إلى الحالة CLOSED
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = Closed
	cb.failureCount = 0
	cb.successCount = 0
}
