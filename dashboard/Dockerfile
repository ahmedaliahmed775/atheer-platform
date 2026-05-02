# ── المرحلة الأولى: بناء التطبيق ──
FROM node:20-alpine AS builder

WORKDIR /app

# نسخ ملفات التبعيات أولاً للاستفادة من طبقات Docker المؤقتة
COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund

# نسخ الكود المصدري
COPY . .

# بناء التطبيق للإنتاج
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build

# ── المرحلة الثانية: صورة التشغيل (nginx) ──
FROM nginx:1.25-alpine

# نسخ إعدادات nginx المُخصّصة
RUN rm /etc/nginx/conf.d/default.conf
COPY <<'EOF' /etc/nginx/conf.d/atheer-dashboard.conf
server {
    listen 3000;
    server_name _;
    root /usr/share/nginx/html;
    index index.html;

    # ضغط gzip
    gzip on;
    gzip_types text/plain text/css application/json application/javascript text/xml;
    gzip_min_length 256;

    # تخزين مؤقت للأصول الثابتة
    location /_next/static/ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }

    # توجيه كل المسارات إلى Next.js
    location / {
        try_files $uri $uri.html $uri/index.html /index.html;
    }

    # فحص الصحة
    location /nginx-health {
        access_log off;
        return 200 "ok";
        add_header Content-Type text/plain;
    }
}
EOF

# نسخ ملفات البناء من المرحلة الأولى
COPY --from=builder /app/out /usr/share/nginx/html

# المنفذ
EXPOSE 3000

# فحص الصحة
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:3000/nginx-health || exit 1

CMD ["nginx", "-g", "daemon off;"]
