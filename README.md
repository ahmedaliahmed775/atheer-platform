# 🚀 Atheer Platform

> منصة دفع إلكتروني بتقنية NFC — سويتش مركزي + لوحة تحكم إدارية

---

## البنية التقنية

```
atheer-platform/
├── switch/         ← سويتش Go (API + معالجة المعاملات)
├── dashboard/      ← لوحة التحكم Next.js 14
└── docker-compose.yml
```

### السويتش (Go 1.22)
- **GATE → VERIFY → EXECUTE** pipeline لمعالجة المعاملات
- تشفير مغلّف (Envelope Encryption) للبذور عبر KMS
- HMAC-SHA256 + HKDF للتحقق من صحة التوكن
- Circuit Breaker لمحوّلات المحافظ
- Admin API مع JWT + TOTP

### الداشبورد (Next.js 14 + shadcn/ui)
- 🏠 **لوحة القيادة** — 4 بطاقات KPI + رسم بياني + حالة المحوّلات
- 💳 **المعاملات** — TanStack Table + فلاتر + تصدير Excel
- 📊 **الإحصائيات** — 5 رسوم بيانية (Recharts)
- 👥 **المستخدمون** — إدارة + تعليق/إلغاء + تعديل الحدود
- 💼 **المحافظ** — CRUD + اختبار الاتصال
- 🔄 **التسوية** — تشغيل + تقارير + تصدير
- ⚙️ **الإعدادات** — حسابات + إعدادات النظام + TOTP

---

## التشغيل السريع

### متطلبات
- Docker + Docker Compose
- أو: Go 1.22 + Node.js 20 + PostgreSQL 16

### عبر Docker Compose
```bash
# تشغيل كل الخدمات
docker-compose up -d

# السويتش: http://localhost:8080
# الداشبورد: http://localhost:3000
```

### تشغيل محلي
```bash
# السويتش
cd switch
go run ./cmd/server -config config.yaml

# الداشبورد
cd dashboard
npm install
npm run dev
```

---

## الاختبارات

```bash
# اختبارات السويتش (Go)
cd switch
go test ./... -v

# بناء الداشبورد
cd dashboard
npm run build
```

---

## المنافذ

| الخدمة | المنفذ | الوصف |
|--------|--------|-------|
| PostgreSQL | 5432 | قاعدة البيانات |
| Switch API | 8080 | API العام + الإدارة |
| Dashboard | 3000 | لوحة التحكم |

---

## البنية الأمنية

- **JWT + TOTP** للمصادقة الإدارية
- **HMAC-SHA256** للتحقق من توكن الدفع
- **Envelope Encryption** لتشفير البذور
- **Role-Based Access** — SUPER_ADMIN / ADMIN / WALLET_ADMIN / VIEWER
- **Scope Filtering** — WALLET_ADMIN يرى محفظته فقط

---

## المتغيرات البيئية

| المتغير | الافتراضي | الوصف |
|---------|-----------|-------|
| `POSTGRES_DB` | atheer | اسم قاعدة البيانات |
| `POSTGRES_USER` | atheer | مستخدم PostgreSQL |
| `POSTGRES_PASSWORD` | atheer_secret | كلمة المرور |
| `NEXT_PUBLIC_API_URL` | http://localhost:8080 | عنوان API للداشبورد |

---

## الترخيص

ملكية خاصة — Atheer Platform © 2026
