// الصفحة الرئيسية — إعادة توجيه إلى لوحة القيادة
import { redirect } from "next/navigation";

export default function HomePage() {
  redirect("/dashboard");
}
