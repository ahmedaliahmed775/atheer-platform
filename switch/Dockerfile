# ── المرحلة الأولى: بناء الثنائي ──
FROM golang:1.22-alpine AS builder

# تثبيت تبعيات البناء
RUN apk add --no-cache git ca-certificates tzdata

# تعيين مجلد العمل
WORKDIR /app

# نسخ ملفات الوحدات أولاً للاستفادة من طبقات Docker المؤقتة
COPY go.mod go.sum ./
RUN go mod download

# نسخ باقي الكود المصدري
COPY . .

# بناء الثنائي مع تحسينات الإنتاج
# CGO_ENABLED=0 — بناء ثابت بدون libc
# -ldflags="-s -w" — إزالة رموز التصحيح وتقليل الحجم
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /atheer-switch ./cmd/server

# ── المرحلة الثانية: صورة التشغيل ──
FROM alpine:3.19

# تثبيت الشهادات والمنطقة الزمنية
RUN apk add --no-cache ca-certificates tzdata

# إنشاء مستخدم غير جذر للأمان
RUN addgroup -S atheer && adduser -S atheer -G atheer

# تعيين مجلد العمل
WORKDIR /app

# نسخ الثنائي من مرحلة البناء
COPY --from=builder /atheer-switch /app/atheer-switch

# نسخ ملفات الترحيلات
COPY --from=builder /app/migrations /app/migrations

# ملف الإعدادات الافتراضي (يُستبدل بالحجم أو متغيرات البيئة)
COPY config.example.yaml /app/config.yaml

# استخدام المستخدم غير الجذر
USER atheer

# منفذ الخادم
EXPOSE 8080

# فحص الصحة
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

# تشغيل الخادم
ENTRYPOINT ["/app/atheer-switch"]
CMD ["-config", "/app/config.yaml"]
