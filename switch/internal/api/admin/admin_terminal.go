// معالج الطرفية البعيدة — WebSocket للوصول CMD من الداشبورد
// متاح فقط لـ SUPER_ADMIN
package admin

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/atheer/switch/internal/middleware"
	"github.com/atheer/switch/internal/model"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

// TerminalHandler — معالج الطرفية البعيدة
type TerminalHandler struct {
	jwtSecret string
}

// NewTerminalHandler — ينشئ معالج الطرفية البعيدة
func NewTerminalHandler(jwtSecret string) *TerminalHandler {
	return &TerminalHandler{jwtSecret: jwtSecret}
}

// terminalMessage — رسالة WebSocket بين العميل والخادم
type terminalMessage struct {
	Type string `json:"type"` // "input" أو "resize" أو "ping"
	Data string `json:"data"` // البيانات (إدخال نصي، أبعاد، إلخ)
	Cols int    `json:"cols"` // عدد الأعمدة (لـ resize)
	Rows int    `json:"rows"` // عدد الصفوف (لـ resize)
}

// terminalOutput — رسالة إخراج من الخادم
type terminalOutput struct {
	Type     string `json:"type"`     // "output" أو "error" أو "exit" أو "connected"
	Data     string `json:"data"`     // البيانات
	Shell    string `json:"shell"`    // نوع الصدفة
	OS       string `json:"os"`       // نظام التشغيل
	ExitCode int    `json:"exitCode"` // رمز الخروج
}

// wsUpgrader — مُرقّي WebSocket (gorilla/websocket)
var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		// السماح بالاتصال من الداشبورد المحلي
		return origin == "http://localhost:3000" ||
			origin == "http://127.0.0.1:3000" ||
			origin == ""
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// HandleTerminal — يعالج اتصال WebSocket للطرفية البعيدة
// GET /admin/v1/terminal?token=xxx
// هذا المسار لا يمر عبر وسيط JWT العادي لأن المتصفح لا يمكنه إرسال رأس Authorization مع WebSocket
// بدلاً من ذلك، يُحقَّق الرمز من معامل الاستعلام ?token=
func (h *TerminalHandler) HandleTerminal(w http.ResponseWriter, r *http.Request) {
	// التحقق من JWT من معامل الاستعلام
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		writeAdminJSON(w, http.StatusUnauthorized, map[string]string{
			"errorCode":    model.ErrUnauthorized,
			"errorMessage": "رمز المصادقة مطلوب",
		})
		return
	}

	// تحليل والتحقق من الرمز
	claims := &middleware.AdminClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(h.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		slog.Warn("طرفية: رمز غير صالح", "error", err)
		writeAdminJSON(w, http.StatusUnauthorized, map[string]string{
			"errorCode":    model.ErrUnauthorized,
			"errorMessage": "رمز المصادقة غير صالح",
		})
		return
	}

	// التحقق من الدور — SUPER_ADMIN فقط
	if claims.Role != model.RoleSuperAdmin {
		slog.Warn("طرفية: دور غير مصرح", "role", claims.Role, "email", claims.Email)
		writeAdminJSON(w, http.StatusForbidden, map[string]string{
			"errorCode":    model.ErrForbiddenRole,
			"errorMessage": "فقط المدير الأعلى يمكنه استخدام الطرفية",
		})
		return
	}

	slog.Info("طرفية: اتصال جديد", "email", claims.Email)

	// ترقية الاتصال إلى WebSocket
	h.handleWebSocket(w, r, claims.Email)
}

// handleWebSocket — يدير اتصال WebSocket مع صدفة النظام
func (h *TerminalHandler) handleWebSocket(w http.ResponseWriter, r *http.Request, email string) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("طرفية: فشل ترقية WebSocket", "error", err)
		return
	}
	defer conn.Close()

	// تحديد الصدفة المناسبة
	shell, shellArgs := getShell()

	// إنشاء سياق مع إلغاء
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// إنشاء عملية الصدفة
	cmd := exec.CommandContext(ctx, shell, shellArgs...)
	cmd.Env = os.Environ()
	cmd.Dir, _ = os.Getwd()

	// أنابيب الإدخال والإخراج
	stdin, err := cmd.StdinPipe()
	if err != nil {
		slog.Error("طرفية: فشل إنشاء أنبوب الإدخال", "error", err)
		return
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		slog.Error("طرفية: فشل إنشاء أنبوب الإخراج", "error", err)
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		slog.Error("طرفية: فشل إنشاء أنبوب الأخطاء", "error", err)
		return
	}

	// بدء العملية
	if err := cmd.Start(); err != nil {
		slog.Error("طرفية: فشل بدء الصدفة", "error", err)
		return
	}

	// إرسال رسالة اتصال ناجح
	sendTerminalOutput(conn, terminalOutput{
		Type:  "connected",
		Shell: shell,
		OS:    runtime.GOOS,
	})

	var once sync.Once
	cleanup := func() {
		once.Do(func() {
			cancel()
			cmd.Process.Kill()
			cmd.Wait()
		})
	}
	defer cleanup()

	// goroutine: قراءة stdout وإرساله
	go func() {
		buf := make([]byte, 4096)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			n, err := stdout.Read(buf)
			if n > 0 {
				sendTerminalOutput(conn, terminalOutput{
					Type: "output",
					Data: string(buf[:n]),
				})
			}
			if err != nil {
				return
			}
		}
	}()

	// goroutine: قراءة stderr وإرساله
	go func() {
		buf := make([]byte, 4096)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			n, err := stderr.Read(buf)
			if n > 0 {
				sendTerminalOutput(conn, terminalOutput{
					Type: "error",
					Data: string(buf[:n]),
				})
			}
			if err != nil {
				return
			}
		}
	}()

	// goroutine: انتظار انتهاء العملية
	go func() {
		err := cmd.Wait()
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		sendTerminalOutput(conn, terminalOutput{
			Type:     "exit",
			ExitCode: exitCode,
		})
		cleanup()
	}()

	// حلقة قراءة الرسائل من العميل
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break // انقطع الاتصال
		}

		var msg terminalMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case "input":
			if _, err := stdin.Write([]byte(msg.Data)); err != nil {
				slog.Error("طرفية: فشل الكتابة في stdin", "error", err)
				return
			}
		case "resize":
			// لا يمكن تغيير حجم الصدفة بدون PTY على Windows
			// يتم تجاهل هذا على Windows
		case "ping":
			sendTerminalOutput(conn, terminalOutput{Type: "pong"})
		}
	}

	slog.Info("طرفية: انقطع الاتصال", "email", email)
}

// getShell — يُرجع مسار الصدفة المناسبة للنظام
func getShell() (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd.exe", []string{}
	}
	// Linux/macOS — محاولة استخدام bash
	if _, err := os.Stat("/bin/bash"); err == nil {
		return "/bin/bash", []string{"-i"}
	}
	return "/bin/sh", []string{}
}

// sendTerminalOutput — يرسل رسالة إخراج عبر WebSocket
func sendTerminalOutput(conn *websocket.Conn, output terminalOutput) {
	data, err := json.Marshal(output)
	if err != nil {
		return
	}
	// تعيين مهلة الكتابة
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	conn.WriteMessage(websocket.TextMessage, data)
}
