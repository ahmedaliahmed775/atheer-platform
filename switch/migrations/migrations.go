// تضمين ملفات ترحيل قاعدة البيانات — يُستخدم من قبل cmd/server و cmd/migrate
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
