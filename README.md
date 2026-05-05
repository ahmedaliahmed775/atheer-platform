# 🚀 منصة Atheer

> منصة دفع إلكتروني بتقنية NFC — سويتش مركزي + لوحة تحكم إدارية

---

## البنية التقنية

```
atheer-platform/
├── switch/              ← سويتش Go (API + معالجة المعاملات)
│   ├── cmd/server/      ← نقطة الدخول الرئيسية
│   ├── cmd/seed-admin/  ← أداة إنشاء مدير افتراضي
│   ├── cmd/migrate/     ← أداة تشغيل الترحيلات
│   ├── internal/        ← الكود الداخلي
│   ├── migrations/      ← ملفات SQL لإنشاء الجداول
│   ├── config.example.yaml
│   └── Dockerfile
├── dashboard/           ← لوحة التحكم Next.js 14
│   ├── src/
│   ├── Dockerfile
│   └── .env.example
├── docker-compose.yml
├── .env.example
└── Makefile
```

### السويتش (Go)
- **GATE → VERIFY → EXECUTE** خط معالجة المعاملات
- تشفير مغلّف (Envelope Encryption) للبذور عبر KMS
- HMAC-SHA256 + HKDF للتحقق من صحة التوكن
- Circuit Breaker لمحوّلات المحافظ
- Admin API مع JWT + TOTP
- WebSocket طرفية بعيدة لـ SUPER_ADMIN

### الداشبورد (Next.js 14 + shadcn/ui)
- 🏠 **لوحة القيادة** — بطاقات KPI + رسم بياني + حالة المحوّلات
- 💳 **المعاملات** — جدول مع فلاتر + تصدير Excel
- 📊 **الإحصائيات** — رسوم بيانية متعددة (Recharts)
- 👥 **المستخدمون** — إدارة + تعليق/إلغاء + تعديل الحدود
- 💼 **المحافظ** — CRUD + اختبار الاتصال + API Key/Secret
- 🔄 **التسوية** — تشغيل + تقارير + تصدير Excel
- 🖥️ **الطرفية** — طرفية بعيدة عبر WebSocket (SUPER_ADMIN فقط)
- ⚙️ **الإعدادات** — اتصال السويتش + حسابات + إعدادات النظام + الملف الشخصي

---

## متطلبات التشغيل

| المكون | الإصدار | الوصف |
|--------|---------|-------|
| PostgreSQL | 16+ | قاعدة البيانات |
| Go | 1.22+ | بناء السويتش (أو Docker) |
| Node.js | 20+ | بناء الداشبورد (أو Docker) |
| Docker | 20+ | تشغيل الحاويات (اختياري) |
| Docker Compose | 2+ | تنسيق الحاويات (اختياري) |

---

## الطريقة 1: التشغيل عبر Docker Compose (الأسهل)

### 1. إعداد متغيرات البيئة

```bash
# نسخ ملف المتغيرات
cp .env.example .env

# تعديل القيم الحساسة
# يجب تغيير: POSTGRES_PASSWORD, JWT_SECRET, KMS_MASTER_KEY
```

محتوى `.env`:
```env
POSTGRES_DB=atheer
POSTGRES_USER=atheer
POSTGRES_PASSWORD=كلمة_مرور_قوية_لقاعدة_البيانات

# هذه المتغيرات تُمرَّر للسويتش عبر docker-compose
JWT_SECRET=سر_طويل_عشوائي_للرمز_المميز
KMS_MASTER_KEY=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
```

### 2. إعداد ملف إعدادات السويتش

```bash
cp switch/config.example.yaml switch/config.yaml
```

عدّل `switch/config.yaml` — في وضع Docker يجب أن يكون `host: postgres` (اسم الحاوية):
```yaml
database:
  host: postgres    # اسم حاوية PostgreSQL في Docker
  port: 5432
  name: atheer
  user: atheer
  password: ${DB_PASSWORD}
```

### 3. تشغيل الخدمات

```bash
# بناء وتشغيل كل الخدمات في الخلفية
docker-compose up -d --build

# متابعة السجلات
docker-compose logs -f

# التحقق من حالة الخدمات
docker-compose ps
```

### 4. إنشاء المستخدم الإداري الافتراضي

```bash
docker-compose exec atheer-switch /app/atheer-switch -config /app/config.yaml -seed-admin
```

أو يدوياً عبر أداة seed-admin:
```bash
docker-compose exec atheer-switch /app/seed-admin -config /app/config.yaml
```

> **بيانات الدخول الافتراضية:** `admin@atheer.ye` / `admin123`
> ⚠️ **غيّر كلمة المرور فوراً بعد أول تسجيل دخول!**

### 5. الوصول للخدمات

| الخدمة | العنوان | الوصف |
|--------|---------|-------|
| الداشبورد | http://localhost:3000 | لوحة التحكم الإدارية |
| السويتش API | http://localhost:8080 | API العام + الإدارة |
| فحص الصحة | http://localhost:8080/health | حالة السويتش |

---

## الطريقة 2: النشر على خادم VPS (استضافة)

### 1. إعداد الخادم

```bash
# تثبيت Docker و Docker Compose
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER

# تثبيت PostgreSQL (أو استخدم حاوية Docker منفصلة)
sudo apt install postgresql-16

# تثبيت Nginx كوكيل عكسي
sudo apt install nginx
```

### 2. إعداد PostgreSQL

```bash
# الدخول لـ PostgreSQL
sudo -u postgres psql

# إنشاء قاعدة البيانات والمستخدم
CREATE DATABASE atheer;
CREATE USER atheer WITH ENCRYPTED PASSWORD 'كلمة_مرور_قوية';
GRANT ALL PRIVILEGES ON DATABASE atheer TO atheer;
\q
```

### 3. بناء السويتش يدوياً (بدون Docker)

```bash
# استنساخ المشروع
git clone <repo-url> /opt/atheer-platform
cd /opt/atheer-platform

# بناء السويتش
cd switch
go build -ldflags="-s -w" -o build/atheer-switch ./cmd/server

# نسخ ملف الإعدادات
cp config.example.yaml config.yaml
```

عدّل `config.yaml`:
```yaml
server:
  port: 8080
  read_timeout: 30s
  write_timeout: 30s

database:
  host: localhost          # أو عنوان خادم PostgreSQL
  port: 5432
  name: atheer
  user: atheer
  password: كلمة_مرور_قوية
  max_conns: 20

security:
  timestamp_tolerance: 60
  look_ahead_window: 10
  default_payer_limit: 5000
  daily_limit: 50000
  monthly_limit: 500000
  jwt_secret: سر_طويل_عشوائي_للرمز_المميز
  jwt_expiry: 8h

kms:
  provider: local
  master_key: مفتاح_سيد_64_حرف_سداسي_عشري

notifications:
  telegram:
    enabled: false
    bot_token: ""
    chat_id: ""
```

### 4. تشغيل السويتش كخدمة نظام (systemd)

أنشئ ملف `/etc/systemd/system/atheer-switch.service`:
```ini
[Unit]
Description=Atheer Payment Switch
After=network.target postgresql.service
Requires=postgresql.service

[Service]
Type=simple
User=atheer
Group=atheer
WorkingDirectory=/opt/atheer-platform/switch
ExecStart=/opt/atheer-platform/switch/build/atheer-switch -config /opt/atheer-platform/switch/config.yaml
Restart=always
RestartSec=5
LimitNOFILE=65536

# متغيرات البيئة (بديل عن config.yaml)
Environment=DB_PASSWORD=كلمة_مرور_قوية
Environment=JWT_SECRET=سر_طويل_عشوائي
Environment=KMS_MASTER_KEY=مفتاح_سيد_64_حرف

# الأمان
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/atheer-platform/switch

[Install]
WantedBy=multi-user.target
```

```bash
# إنشاء مستخدم النظام
sudo useradd -r -s /bin/false atheer
sudo chown -R atheer:atheer /opt/atheer-platform/switch

# تفعيل وتشغيل الخدمة
sudo systemctl daemon-reload
sudo systemctl enable atheer-switch
sudo systemctl start atheer-switch

# التحقق من الحالة
sudo systemctl status atheer-switch
sudo journalctl -u atheer-switch -f
```

### 5. إنشاء المستخدم الإداري الافتراضي

```bash
cd /opt/atheer-platform/switch
go run ./cmd/seed-admin -config config.yaml
```

> **بيانات الدخول الافتراضية:** `admin@atheer.ye` / `admin123`

### 6. بناء الداشبورد للإنتاج

```bash
cd /opt/atheer-platform/dashboard

# تثبيت التبعيات
npm install --legacy-peer-deps

# بناء للإنتاج
npm run build

# تشغيل بالإنتاج
PORT=3000 node server.js
```

أو أنشئ خدمة systemd `/etc/systemd/system/atheer-dashboard.service`:
```ini
[Unit]
Description=Atheer Dashboard
After=network.target atheer-switch.service

[Service]
Type=simple
User=atheer
Group=atheer
WorkingDirectory=/opt/atheer-platform/dashboard
Environment=NODE_ENV=production
Environment=PORT=3000
Environment=NEXT_PUBLIC_API_URL=http://localhost:8080
ExecStart=/usr/bin/node server.js
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable atheer-dashboard
sudo systemctl start atheer-dashboard
```

### 7. إعداد Nginx كوكيل عكسي

أنشئ ملف `/etc/nginx/sites-available/atheer`:
```nginx
# وكيل الداشبورد
server {
    listen 80;
    server_name dashboard.atheer.ye;

    # توجيه إلى Next.js
    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}

# وكيل سويتش API
server {
    listen 80;
    server_name api.atheer.ye;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # دعم WebSocket للطرفية البعيدة
        proxy_read_timeout 86400s;
        proxy_send_timeout 86400s;
    }
}
```

```bash
# تفعيل الموقع
sudo ln -s /etc/nginx/sites-available/atheer /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx

# تثبيت شهادة SSL (موصى به جداً)
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d dashboard.atheer.ye -d api.atheer.ye
```

---

## ربط الداشبورد بالسويتش

### الطريقة 1: عبر صفحة الإعدادات (الأسهل)

1. افتح الداشبورد: `http://dashboard.atheer.ye`
2. سجّل دخولك بـ `admin@atheer.ye` / `admin123`
3. اذهب إلى **الإعدادات** ← تبويب **اتصال السويتش**
4. أدخل عنوان السويتش: `https://api.atheer.ye`
5. اضغط **اختبار الاتصال** — يجب أن تظهر علامة ✅
6. اضغط **حفظ**

### الطريقة 2: عبر متغير البيئة

في ملف `.env.local` داخل مجلد `dashboard/`:
```env
NEXT_PUBLIC_API_URL=https://api.atheer.ye
```

أو عبر متغير البيئة في Docker Compose:
```yaml
atheer-dashboard:
  environment:
    - NEXT_PUBLIC_API_URL=https://api.atheer.ye
```

### الطريقة 3: عبر Nginx Rewrite

إذا كان الداشبورد والسويتش على نفس النطاق، أضف في إعدادات Nginx:
```nginx
# توجيه طلبات API من الداشبورد
location /api/ {
    proxy_pass http://127.0.0.1:8080/;
}
```

ثم في الداشبورد اضبط عنوان السويتش على `/api`.

---

## الأدوار والصلاحيات

| الدور | المستوى | الصلاحيات |
|-------|---------|-----------|
| **SUPER_ADMIN** | 4 | كل شيء + إدارة المدراء + إضافة/تعديل المحافظ + الطرفية البعيدة |
| **ADMIN** | 3 | إدارة المستخدمين + تشغيل التسوية + اختبار المحافظ |
| **WALLET_ADMIN** | 2 | عرض بيانات محفظته فقط (نطاق محدود) |
| **VIEWER** | 1 | قراءة فقط — المعاملات والإحصائيات وفحص الصحة |

### تفصيل صلاحيات API

| المسار | SUPER_ADMIN | ADMIN | WALLET_ADMIN | VIEWER |
|--------|:-----------:|:-----:|:------------:|:------:|
| `GET /admin/v1/transactions` | ✅ | ✅ | ✅ (نطاقه) | ✅ |
| `GET /admin/v1/transactions/:id` | ✅ | ✅ | ✅ (نطاقه) | ✅ |
| `GET /admin/v1/users` | ✅ | ✅ | ✅ (نطاقه) | ❌ |
| `PATCH /admin/v1/users/:id/status` | ✅ | ✅ | ❌ | ❌ |
| `PATCH /admin/v1/users/:id/limit` | ✅ | ✅ | ❌ | ❌ |
| `GET /admin/v1/wallets` | ✅ | ✅ | ✅ (نطاقه) | ❌ |
| `POST /admin/v1/wallets` | ✅ | ❌ | ❌ | ❌ |
| `PUT /admin/v1/wallets/:id` | ✅ | ❌ | ❌ | ❌ |
| `POST /admin/v1/wallets/:id/test` | ✅ | ✅ | ❌ | ❌ |
| `GET /admin/v1/analytics/*` | ✅ | ✅ | ✅ | ✅ |
| `GET /admin/v1/health/*` | ✅ | ✅ | ✅ | ✅ |
| `POST /admin/v1/reconciliation/run` | ✅ | ✅ | ❌ | ❌ |
| `GET /admin/v1/reconciliation/reports` | ✅ | ✅ | ❌ | ❌ |
| `GET /admin/v1/terminal` (WebSocket) | ✅ | ❌ | ❌ | ❌ |
| `GET /admin/v1/admins` | ✅ | ❌ | ❌ | ❌ |
| `POST /admin/v1/admins` | ✅ | ❌ | ❌ | ❌ |
| `PATCH /admin/v1/admins/:id` | ✅ | ❌ | ❌ | ❌ |

---

## متغيرات البيئة الكاملة

### ملف `.env` (جذر المشروع — Docker Compose)

| المتغير | الافتراضي | الوصف |
|---------|-----------|-------|
| `POSTGRES_DB` | atheer | اسم قاعدة البيانات |
| `POSTGRES_USER` | atheer | مستخدم PostgreSQL |
| `POSTGRES_PASSWORD` | — | كلمة مرور PostgreSQL (**إلزامي تغييرها**) |
| `JWT_SECRET` | — | سر الرمز المميز JWT (**إلزامي تغييره**) |
| `KMS_MASTER_KEY` | — | مفتاح KMS الرئيسي (64 حرف سداسي عشري) |

### ملف `switch/config.yaml` (السويتش)

| الإعداد | الافتراضي | الوصف |
|---------|-----------|-------|
| `server.port` | 8080 | منفذ الاستماع |
| `server.read_timeout` | 30s | مهلة القراءة |
| `server.write_timeout` | 30s | مهلة الكتابة |
| `database.host` | localhost | عنوان PostgreSQL |
| `database.port` | 5432 | منفذ PostgreSQL |
| `database.name` | atheer | اسم قاعدة البيانات |
| `database.user` | atheer | المستخدم |
| `database.password` | — | كلمة المرور (أو `${DB_PASSWORD}`) |
| `database.max_conns` | 20 | أقصى عدد اتصالات |
| `security.timestamp_tolerance` | 60 | تفاوت الطابع الزمني (ثانية) |
| `security.look_ahead_window` | 10 | نافذة العداد المسموحة |
| `security.default_payer_limit` | 5000 | حد الدافع الافتراضي (ريال) |
| `security.jwt_secret` | — | سر JWT (أو `${JWT_SECRET}`) |
| `security.jwt_expiry` | 8h | مدة صلاحية الرمز |
| `kms.provider` | local | مزود KMS (local فقط حالياً) |
| `kms.master_key` | — | المفتاح الرئيسي (أو `${KMS_MASTER_KEY}`) |
| `notifications.telegram.enabled` | false | تفعيل إشعارات تيليجرام |
| `notifications.telegram.bot_token` | — | رمز بوت تيليجرام |
| `notifications.telegram.chat_id` | — | معرّف محادثة تيليجرام |

### ملف `dashboard/.env.local` (الداشبورد)

| المتغير | الافتراضي | الوصف |
|---------|-----------|-------|
| `NEXT_PUBLIC_API_URL` | http://localhost:8080 | عنوان API السويتش |

---

## المنافذ

| الخدمة | المنفذ | الوصف |
|--------|--------|-------|
| PostgreSQL | 5432 | قاعدة البيانات |
| Switch API | 8080 | API العام + الإدارة + WebSocket |
| Dashboard | 3000 | لوحة التحكم |

---

## أوامر الإدارة

### السويتش

```bash
# بناء الثنائي
make build

# تشغيل الاختبارات
make test

# تشغيل محلياً
make run

# تشغيل الترحيلات
make migrate

# إنشاء مستخدم إداري افتراضي
cd switch && go run ./cmd/seed-admin -config config.yaml
```

### الداشبورد

```bash
# تثبيت التبعيات
cd dashboard && npm install --legacy-peer-deps

# تشغيل في وضع التطوير
npm run dev

# بناء للإنتاج
npm run build

# تشغيل بالإنتاج
npm start
```

### Docker

```bash
# بناء وتشغيل
docker-compose up -d --build

# إيقاف
docker-compose down

# عرض السجلات
docker-compose logs -f atheer-switch
docker-compose logs -f atheer-dashboard

# إعادة بناء بعد تعديل الكود
docker-compose up -d --build --force-recreate

# الدخول لحاوية السويتش
docker-compose exec atheer-switch sh

# نسخة احتياطية لقاعدة البيانات
docker-compose exec postgres pg_dump -U atheer atheer > backup.sql

# استعادة النسخة الاحتياطية
cat backup.sql | docker-compose exec -T postgres psql -U atheer atheer
```

---

## البنية الأمنية

- **JWT + TOTP** للمصادقة الإدارية — الرمز المميز صالح 8 ساعات
- **HMAC-SHA256** للتحقق من توكن الدفع NFC
- **Envelope Encryption** لتشفير البذور — البذور مشفّرة بـ KMS ولا تُخزّن بصيغة مقروءة
- **Role-Based Access** — SUPER_ADMIN / ADMIN / WALLET_ADMIN / VIEWER
- **Scope Filtering** — WALLET_ADMIN يرى بيانات محفظته فقط
- **CORS** — السماح بطلبات المتصفح من نطاقات محددة
- **SQL Parameterized** — كل استعلامات DB بمعاملات (`$1`, `$2`) — لا دمج نصوص
- **Zeroize** — مفاتيح التشفير تُمحى بعد الاستخدام: `clear(lukBytes)`

### نصائح أمنية للإنتاج

1. **غيّر كل كلمات المرور والأسرار الافتراضية** قبل النشر
2. **استخدم HTTPS** عبر شهادة SSL (Let's Encrypt مجاني)
3. **قيّد الوصول** لقاعدة البيانات — لا تعرّض المنفذ 5432 للإنترنت
4. **فعّل جدار الحماية** — اسمح فقط بالمنافذ 80 و 443
5. **غيّر كلمة مرور المدير الافتراضي** فور أول تسجيل دخول
6. **فعّل TOTP** لكل حساب إداري من صفحة الملف الشخصي
7. **راقب السجلات** — `journalctl -u atheer-switch -f`
8. **خذ نسخاً احتياطية** منتظمة لقاعدة البيانات

---

## حل المشاكل الشائعة

### السويتش لا يبدأ

```bash
# التحقق من السجلات
sudo journalctl -u atheer-switch -n 50

# التحقق من اتصال قاعدة البيانات
psql -h localhost -U atheer -d atheer

# التحقق من المنفذ
ss -tlnp | grep 8080
```

### الداشبورد لا يتصل بالسويتش

1. تحقق من عنوان السويتش في **الإعدادات ← اتصال السويتش**
2. تأكد أن السويتش يعمل: `curl http://localhost:8080/health`
3. إذا كنت خلف وكيل عكسي، تأكد من إعدادات CORS
4. تحقق من متغير `NEXT_PUBLIC_API_URL`

### خطأ في تسوية التقارير

إذا ظهر خطأ `cannot scan date (OID 1082)` — تأكد أنك تستخدم أحدث إصدار من السويتش.

### الطرفية البعيدة لا تعمل

1. الطرفية متاحة فقط لـ **SUPER_ADMIN**
2. تأكد أن WebSocket مدعوم عبر الوكيل العكسي (إعدادات Nginx أعلاه)
3. تحقق من إعدادات `proxy_read_timeout` و `proxy_send_timeout`

---

## الترخيص

ملكية خاصة — Atheer Platform © 2026
