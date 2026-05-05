# سويتش Atheer

> سويتش الدفع المركزي — يستقبل توكنات NFC ويُنفّذ المعاملات عبر محافظ اليمن

## 📋 الوظيفة

السويتش هو الخادم المركزي الذي:
1. يستقبل توكن الدفع الموقّع من التاجر (عبر `POST /api/v1/transaction`)
2. يتحقق من هوية الدافع والتاجر (HMAC + accessToken)
3. يُنفّذ المعاملة عبر محوّل المحفظة (خصم الدافع + إيداع التاجر)
4. يحفظ المعاملة ويُرجع النتيجة

## 🏗️ خط المعالجة (3 طبقات)

```
TransactionRequest ──→ GATE ──→ VERIFY ──→ EXECUTE ──→ TransactionResponse
                        │         │          │
                        ↓         ↓          ↓
                    بحث DB    HMAC+حدود   خصم+إيداع
```

### الطبقة 1: GATE (البوابة)
- استخراج `publicId` من التوكن
- البحث في `switch_records` → استرجاع البذرة المشفرة + العداد + الحدود
- رفض: `UNKNOWN_PAYER`, `ACCOUNT_SUSPENDED`, `DEVICE_MISMATCH`

### الطبقة 2: VERIFY (التحقق)
- التحقق من `accessToken` التاجر عبر محوّل المحفظة
- فحص الطابع الزمني (±60 ثانية)
- فحص العداد (مكافحة إعادة التشغيل + نافذة القبول)
- فك تشفير البذرة → اشتقاق LUK → فحص HMAC
- فحص حد الدافع (`payerLimit`) + حدود المعاملات

### الطبقة 3: EXECUTE (التنفيذ — نمط Saga)
- خصم رصيد الدافع عبر محوّل المحفظة
- إيداع رصيد التاجر عبر محوّل المحفظة
- إذا فشل الإيداع → عكس الخصم (تعويض)
- تحديث العداد + حفظ المعاملة

## 📁 البنية

```
switch/
├── cmd/
│   ├── server/main.go        # نقطة الدخول — تهيئة وتشغيل الخادم
│   ├── seed-admin/main.go    # أداة إنشاء مستخدم إداري افتراضي
│   └── migrate/main.go       # أداة تشغيل الترحيلات
├── internal/
│   ├── model/                # أنواع البيانات (TransactionRequest, GateResult, ...)
│   ├── crypto/               # HKDF-SHA256, HMAC-SHA256, KMS
│   ├── gate/                 # الطبقة 1
│   ├── verify/               # الطبقة 2
│   ├── execute/              # الطبقة 3
│   ├── adapter/              # واجهة محوّل المحفظة + التنفيذات
│   │   ├── adapter.go        # الواجهة الموحدة (WalletAdapter)
│   │   ├── registry.go       # سجل المحوّلات (walletId → Adapter)
│   │   ├── circuit_breaker.go # قاطع الدائرة
│   │   ├── jawali/           # محوّل جوالي
│   │   └── mock/             # محوّل وهمي للاختبارات
│   ├── api/                  # معالجات HTTP العامة
│   │   ├── transaction_handler.go
│   │   ├── enroll_handler.go
│   │   ├── unenroll_handler.go
│   │   ├── sync_handler.go
│   │   ├── health_handler.go
│   │   ├── payer_limit_handler.go
│   │   └── helpers.go
│   ├── api/admin/            # معالجات HTTP الإدارية
│   │   ├── auth_handler.go   # تسجيل الدخول/الخروج/تجديد الرمز
│   │   ├── admin_admins.go   # CRUD حسابات المدراء
│   │   ├── admin_transactions.go
│   │   ├── admin_users.go
│   │   ├── admin_wallets.go
│   │   ├── admin_analytics.go
│   │   ├── admin_health.go
│   │   ├── admin_recon.go
│   │   └── admin_terminal.go # طرفية بعيدة WebSocket
│   ├── db/                   # مستودعات قاعدة البيانات (pgx/v5)
│   ├── config/               # تحميل config.yaml
│   ├── notify/               # إشعارات تيليجرام
│   └── middleware/           # مصادقة JWT، API Key، CORS، تسجيل
├── migrations/               # ملفات SQL لإنشاء الجداول
├── config.example.yaml       # نموذج ملف الإعدادات
├── Dockerfile
├── go.mod
└── go.sum
```

## 🚀 التشغيل

### محلياً

```bash
# تثبيت التبعيات
go mod tidy

# نسخ ملف الإعدادات وتعديله
cp config.example.yaml config.yaml

# تشغيل الاختبارات
go test ./...

# تشغيل الخادم
go run ./cmd/server -config config.yaml

# إنشاء مستخدم إداري افتراضي
go run ./cmd/seed-admin -config config.yaml
```

### عبر Docker

```bash
# بناء الصورة
docker build -t atheer-switch .

# تشغيل الحاوية
docker run -d \
  -p 8080:8080 \
  -e DB_PASSWORD=كلمة_مرور \
  -e JWT_SECRET=سر_طويل \
  -e KMS_MASTER_KEY=مفتاح_سيد_64_حرف \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  atheer-switch
```

## ⚙️ الإعدادات (config.yaml)

```yaml
server:
  port: 8080
  read_timeout: 30s
  write_timeout: 30s

database:
  host: localhost          # postgres في وضع Docker
  port: 5432
  name: atheer
  user: atheer
  password: ${DB_PASSWORD} # يُستبدل من متغير البيئة
  max_conns: 20

security:
  timestamp_tolerance: 60      # ثانية
  look_ahead_window: 10        # نافذة العداد
  default_payer_limit: 5000    # ريال
  daily_limit: 50000
  monthly_limit: 500000
  jwt_secret: ${JWT_SECRET}
  jwt_expiry: 8h

kms:
  provider: local
  master_key: ${KMS_MASTER_KEY}

notifications:
  telegram:
    enabled: false
    bot_token: ${TELEGRAM_BOT_TOKEN}
    chat_id: ${TELEGRAM_CHAT_ID}
```

## 🔐 الأمان

- البذور مشفّرة بـ KMS (envelope encryption) — لا تُخزّن بصيغة مقروءة
- مفاتيح التشفير تُمحى بعد الاستخدام (`clear(lukBytes)`)
- JWT مع أدوار (SUPER_ADMIN / ADMIN / WALLET_ADMIN / VIEWER)
- كل استعلامات DB بمعاملات (`$1`, `$2`) — لا دمج نصوص
- CORS يسمح فقط بالنطاقات المحددة

## 📡 نقاط API

> **العقد الموحد:** جميع نقاط API العامة تتوافق مع ملف العقد الموحد `atheer-api-contract.yaml` (OpenAPI 3.0).
> أسماء الحقول بصيغة **camelCase** في الطلبات والاستجابات.

### API العام (مصادقة API Key)
| المسار | الطريقة | الوصف | حقول الطلب المطلوبة |
|--------|---------|-------|---------------------|
| `/api/v1/enroll` | POST | تسجيل جهاز دافع | `walletId`, `walletToken`, `deviceId`, `userType` |
| `/api/v1/transaction` | POST | تنفيذ معاملة دفع | `paymentToken.*`, `merchantData.*`, `timestamp` |
| `/api/v1/sync` | POST | مزامنة العداد والحدود | `publicId`, `deviceId`, `timestamp` |
| `/api/v1/payer-limit` | POST | تحديث حد الدفع | `publicId`, `deviceId`, `newLimit`, `timestamp` |
| `/api/v1/unenroll` | POST | إلغاء تسجيل جهاز | `publicId`, `deviceId`, `timestamp` |
| `/health` | GET | فحص الصحة | — |

### API الإدارة (مصادقة JWT)
| المسار | الطريقة | الوصف |
|--------|---------|-------|
| `/admin/v1/auth/login` | POST | تسجيل الدخول |
| `/admin/v1/auth/refresh` | POST | تجديد الرمز |
| `/admin/v1/auth/logout` | POST | تسجيل الخروج |
| `/admin/v1/transactions` | GET | قائمة المعاملات |
| `/admin/v1/transactions/:id` | GET | تفاصيل معاملة |
| `/admin/v1/users` | GET | قائمة المستخدمين |
| `/admin/v1/users/:id/status` | PATCH | تعديل حالة مستخدم |
| `/admin/v1/users/:id/limit` | PATCH | تعديل حد الدافع |
| `/admin/v1/wallets` | GET | قائمة المحافظ |
| `/admin/v1/wallets` | POST | إضافة محفظة |
| `/admin/v1/wallets/:id` | PUT | تعديل محفظة |
| `/admin/v1/wallets/:id/test` | POST | اختبار اتصال محفظة |
| `/admin/v1/analytics/summary` | GET | ملخص الأداء |
| `/admin/v1/analytics/volume` | GET | حجم المعاملات |
| `/admin/v1/analytics/errors` | GET | تحليل الأخطاء |
| `/admin/v1/analytics/latency` | GET | زمن الاستجابة |
| `/admin/v1/health/adapters` | GET | حالة المحوّلات |
| `/admin/v1/health/system` | GET | حالة النظام |
| `/admin/v1/reconciliation/run` | POST | تشغيل التسوية |
| `/admin/v1/reconciliation/reports` | GET | تقارير التسوية |
| `/admin/v1/terminal` | GET (WebSocket) | طرفية بعيدة |
| `/admin/v1/admins` | GET | قائمة المدراء |
| `/admin/v1/admins` | POST | إضافة مدير |
| `/admin/v1/admins/:id` | PATCH | تعديل مدير |

### تفصيل استجابات API العام

**Enroll (201):**
```json
{
  "publicId": "usr_abc123def456",
  "encryptedSeed": "base64-encoded-tee-encrypted-seed",
  "payerLimit": 5000,
  "maxPayerLimit": 50000,
  "attestationLevel": "SOFTWARE",
  "status": "ACTIVE"
}
```

**Transaction (200):**
```json
{
  "transactionId": "550e8400-e29b-41d4-a716-446655440000",
  "status": "SUCCESS",
  "errorCode": "",
  "errorMessage": "",
  "lastValidCounter": 43,
  "timestamp": 1714340401
}
```

**Sync (200):**
```json
{
  "lastValidCounter": 42,
  "payerLimit": 5000,
  "maxPayerLimit": 50000,
  "status": "ACTIVE"
}
```

**Payer Limit (200):**
```json
{
  "publicId": "usr_abc123def456",
  "payerLimit": 10000,
  "maxPayerLimit": 50000,
  "status": "ACTIVE"
}
```

**Unenroll (200):**
```json
{
  "publicId": "usr_abc123def456",
  "status": "UNENROLLED",
  "message": "تم إلغاء التسجيل بنجاح"
}
```
