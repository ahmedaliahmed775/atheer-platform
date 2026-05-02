// إشعارات تيليجرام — تنبيهات المراقبة للمشرفين
// يُرجى الرجوع إلى SPEC §7 — Notifications
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// AlertLevel — مستوى التنبيه
type AlertLevel string

const (
	// AlertLevelInfo — تنبيه معلوماتي
	AlertLevelInfo AlertLevel = "INFO"
	// AlertLevelWarning — تنبيه تحذيري
	AlertLevelWarning AlertLevel = "WARNING"
	// AlertLevelCritical — تنبيه حرج
	AlertLevelCritical AlertLevel = "CRITICAL"
)

// AlertEvent — نوع حدث التنبيه
type AlertEvent string

const (
	// EventAdapterDown — محوّل المحفظة متعطّل
	EventAdapterDown AlertEvent = "ADAPTER_DOWN"
	// EventHighErrorRate — ارتفاع معدل الأخطاء
	EventHighErrorRate AlertEvent = "HIGH_ERROR_RATE"
	// EventCircuitOpen — قاطع الدائرة مفتوح
	EventCircuitOpen AlertEvent = "CIRCUIT_OPEN"
	// EventAdapterRecovered — محوّل المحفظة تعافى
	EventAdapterRecovered AlertEvent = "ADAPTER_RECOVERED"
)

// TelegramNotifier — مُرسل إشعارات تيليجرام
type TelegramNotifier struct {
	botToken string // رمز البوت — لا يُسجَّل أبداً
	chatID   string // معرّف المحادثة
	enabled  bool   // هل الإشعارات مفعّلة
	client   *http.Client
}

// NewTelegramNotifier — ينشئ مُرسل إشعارات تيليجرام جديد
func NewTelegramNotifier(botToken, chatID string, enabled bool) *TelegramNotifier {
	return &TelegramNotifier{
		botToken: botToken,
		chatID:   chatID,
		enabled:  enabled,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// telegramSendMessageRequest — طلب إرسال رسالة تيليجرام
type telegramSendMessageRequest struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

// telegramSendMessageResponse — استجابة إرسال رسالة تيليجرام
type telegramSendMessageResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
}

// SendAlert — يُرسل تنبيه عبر تيليجرام
// المستوى: INFO أو WARNING أو CRITICAL
func (t *TelegramNotifier) SendAlert(ctx context.Context, level AlertLevel, event AlertEvent, message string) error {
	if !t.enabled {
		slog.Debug("تيليجرام: الإشعارات معطّلة", "level", level, "event", event)
		return nil
	}

	// تنسيق الرسالة
	emoji := levelEmoji(level)
	text := fmt.Sprintf("%s <b>[%s]</b> <code>%s</code>\n\n%s", emoji, level, event, message)

	// إعداد الطلب
	reqBody := telegramSendMessageRequest{
		ChatID:    t.chatID,
		Text:      text,
		ParseMode: "HTML",
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("تيليجرام: ترميز الطلب: %w", err)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.botToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("تيليجرام: إنشاء الطلب: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// إرسال الطلب
	resp, err := t.client.Do(req)
	if err != nil {
		slog.Error("تيليجرام: فشل الإرسال", "error", err, "event", event)
		return fmt.Errorf("تيليجرام: إرسال الطلب: %w", err)
	}
	defer resp.Body.Close()

	// تحليل الاستجابة
	var tgResp telegramSendMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&tgResp); err != nil {
		return fmt.Errorf("تيليجرام: تحليل الاستجابة: %w", err)
	}

	if !tgResp.OK {
		slog.Error("تيليجرام: استجابة غير ناجحة", "description", tgResp.Description)
		return fmt.Errorf("تيليجرام: استجابة غير ناجحة: %s", tgResp.Description)
	}

	slog.Info("تيليجرام: تم إرسال التنبيه", "level", level, "event", event)
	return nil
}

// NotifyAdapterDown — يُرسل تنبيه تعطّل محوّل محفظة
func (t *TelegramNotifier) NotifyAdapterDown(ctx context.Context, walletId, errMsg string) error {
	msg := fmt.Sprintf("⚠️ محوّل المحفظة <b>%s</b> متعطّل\n\nالخطأ: <code>%s</code>", walletId, errMsg)
	return t.SendAlert(ctx, AlertLevelCritical, EventAdapterDown, msg)
}

// NotifyHighErrorRate — يُرسل تنبيه ارتفاع معدل الأخطاء
func (t *TelegramNotifier) NotifyHighErrorRate(ctx context.Context, walletId string, errorRate float64) error {
	msg := fmt.Sprintf("📈 ارتفاع معدل الأخطاء في محوّل <b>%s</b>\n\nالمعدل: <code>%.1f%%</code>", walletId, errorRate*100)
	return t.SendAlert(ctx, AlertLevelWarning, EventHighErrorRate, msg)
}

// NotifyCircuitOpen — يُرسل تنبيه فتح قاطع الدائرة
func (t *TelegramNotifier) NotifyCircuitOpen(ctx context.Context, walletId string) error {
	msg := fmt.Sprintf("🔴 قاطع الدائرة مفتوح لمحوّل <b>%s</b>\n\nالطلبات مرفوضة حتى التعافي", walletId)
	return t.SendAlert(ctx, AlertLevelCritical, EventCircuitOpen, msg)
}

// NotifyAdapterRecovered — يُرسل تنبيه تعافي محوّل محفظة
func (t *TelegramNotifier) NotifyAdapterRecovered(ctx context.Context, walletId string) error {
	msg := fmt.Sprintf("🟢 محوّل المحفظة <b>%s</b> تعافى\n\nقاطع الدائرة أُغلق والطلبات تعمل", walletId)
	return t.SendAlert(ctx, AlertLevelInfo, EventAdapterRecovered, msg)
}

// levelEmoji — يُرجع رمز تعبيري حسب مستوى التنبيه
func levelEmoji(level AlertLevel) string {
	switch level {
	case AlertLevelInfo:
		return "ℹ️"
	case AlertLevelWarning:
		return "⚠️"
	case AlertLevelCritical:
		return "🔴"
	default:
		return "📢"
	}
}
