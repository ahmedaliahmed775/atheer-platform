// دوال مساعدة لمعالجات HTTP — كتابة/قراءة JSON ومعالجة الأخطاء
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/atheer/switch/internal/model"
)

// writeJSON — يكتب استجابة JSON مع حالة HTTP المحددة
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			slog.Error("API: فشل كتابة JSON", "error", err)
		}
	}
}

// readJSON — يقرأ ويحلل JSON من جسم الطلب
func readJSON(r *http.Request, dst interface{}) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields() // رفض الحقول غير المعروفة
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	return nil
}

// writeError — يكتب استجابة خطأ بصيغة JSON من AppError
func writeError(w http.ResponseWriter, appErr *model.AppError) {
	resp := map[string]interface{}{
		"errorCode":    appErr.Code,
		"errorMessage": appErr.Message,
	}
	writeJSON(w, appErr.HTTPStatus, resp)
}

// writeErrorWithCode — يكتب استجابة خطأ من رمز الخطأ
func writeErrorWithCode(w http.ResponseWriter, code string) {
	appErr := model.NewAppError(code)
	writeError(w, appErr)
}

// writeBadRequest — يكتب استجابة خطأ 400 مع رسالة مخصصة
func writeBadRequest(w http.ResponseWriter, message string) {
	appErr := model.NewAppErrorWithMessage(model.ErrInvalidRequest, message)
	writeError(w, appErr)
}
