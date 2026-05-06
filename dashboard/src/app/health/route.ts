// نقطة فحص صحة الداشبورد — لا تحتاج مصادقة
// تُستخدم من Docker healthcheck ومن Nginx
export async function GET() {
  return new Response("ok", { status: 200 });
}
