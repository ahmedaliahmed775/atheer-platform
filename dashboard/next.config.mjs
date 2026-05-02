/** @type {import('next').NextConfig} */
const nextConfig = {
  // بناء مستقل لـ Docker — يُنتج مجلد standalone يحتوي كل التبعيات
  output: 'standalone',

  // عنوان API الخلفي — سويتش Atheer
  async rewrites() {
    return [
      {
        source: '/api/:path*',
        destination: `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/:path*`,
      },
    ];
  },
};

export default nextConfig;
