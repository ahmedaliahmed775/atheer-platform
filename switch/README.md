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
├── cmd/server/main.go        # نقطة الدخول — تهيئة وتشغيل الخادم
├── internal/
│   ├── model/                # أنواع البيانات (TransactionRequest, GateResult, ...)
│   ├── crypto/               # HKDF-SHA256, HMAC-SHA256, KMS
│   ├── gate/                 # الطبقة 1
│   ├── verify/               # الطبقة 2
│   ├── execute/              # الطبقة 3
│   ├── adapter/              # واجهة محوّل المحفظة + التنفيذات
│   │   ├── adapter.go        # الواجهة الموحدة (WalletAdapter)
│   │   ├── jawali/           # محوّل جوالي
│   │   ├── floosak/          # محوّل فلوسك
│   │   └── mock/             # محوّل وهمي للاختبارات
│   ├── api/                  # معالجات HTTP (enroll, transaction, sync, ...)
│   ├── db/                   # مستودعات قاعدة البيانات (PayerRepo, TransactionRepo, ...)
│   ├── config/               # تحميل config.yaml
│   ├── notify/               # إشعارات تيليجرام
│   └── middleware/           # مصادقة API Key، تسجيل، حد الطلبات
├── migrations/               # ملفات SQL لإنشاء الجداول
├── config.example.yaml       # نموذج ملف الإعدادات
├── Dockerfile
├── go.mod
└── go.sum
```

## 🚀 التشغيل

```bash
# تثبيت التبعيات
go mod tidy

# تشغيل الاختبارات
go test ./...

# تشغيل الخادم
cp config.example.yaml config.yaml
go run cmd/server/main.go
```

## ⚙️ الإعدادات (config.yaml)

```yaml
server:
  port: 8080

database:
  host: localhost
  port: 5432
  name: atheer

security:
  timestamp_tolerance: 60      # ثانية
  look_ahead_window: 10        # نافذة العداد
  default_payer_limit: 5000    # ريال
```

## 🔐 الأمان
- البذور مشفّرة بـ KMS (envelope encryption) — لا تُخزّن بصيغة مقروءة
- LUK يُشتق مؤقتاً في VERIFY ويُمحى بعد الاستخدام
- HMAC يمنع تزوير التوكن — التاجر لا يملك LUK
- العداد يمنع إعادة تشغيل التوكن
- accessToken يُثبت هوية التاجر أمام المحفظة
