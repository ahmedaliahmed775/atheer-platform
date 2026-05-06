# ── المرحلة الأولى: بناء التطبيق ──
FROM node:20-alpine AS builder

WORKDIR /app

# نسخ ملفات التبعيات أولاً للاستفادة من طبقات Docker المؤقتة
COPY package.json package-lock.json ./
RUN npm install --legacy-peer-deps

# نسخ الكود المصدري
COPY . .

# عنوان API الخلفي — يُمرَّر وقت البناء (build-time)
# متغيرات NEXT_PUBLIC_* تُدمج في الكود عند البناء وليس عند التشغيل
ARG NEXT_PUBLIC_API_URL=http://localhost:8080
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL

# بناء التطبيق للإنتاج
ENV NEXT_TELEMETRY_DISABLED=1
ENV NODE_OPTIONS=--max-old-space-size=768
RUN npm run build

# ── المرحلة الثانية: صورة التشغيل (Node.js runtime) ──
# نستخدم next start بدلاً من nginx لأن التطبيق يحتاج:
# - middleware.ts لحماية المسارات (JWT)
# - next.config.mjs rewrites لتوجيه طلبات API
FROM node:20-alpine AS runner

WORKDIR /app

# إنشاء مستخدم غير جذر للأمان
RUN addgroup -S atheer && adduser -S atheer -G atheer

# تعيين متغيرات البيئة
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

# نسخ الملفات اللازمة من مرحلة البناء
COPY --from=builder /app/public ./public
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static

# استخدام المستخدم غير الجذر
USER atheer

# المنفذ
EXPOSE 3000

# فحص الصحة
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:3000/ || exit 1

# تشغيل خادم Next.js
CMD ["node", "server.js"]
