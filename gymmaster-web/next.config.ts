import type { NextConfig } from 'next';
import createNextIntlPlugin from 'next-intl/plugin';
import { buildSecurityHeaders } from './src/lib/security-headers';

const withNextIntl = createNextIntlPlugin('./src/i18n/request.ts');

const nextConfig: NextConfig = {
  output: 'standalone',
  // Không lộ "X-Powered-By: Next.js".
  poweredByHeader: false,
  // Admin app: dữ liệu lấy phía client qua TanStack Query, phiên đăng nhập đọc cookie mỗi request,
  // nên không bật cacheComponents/partialPrefetching (mặc định của create-next-app 16.4).
  turbopack: {
    rules: {
      '*.css': {
        loaders: ['@tailwindcss/turbopack'],
        as: '*.css',
      },
    },
  },
  async headers() {
    return [
      {
        source: '/:path*',
        headers: buildSecurityHeaders({
          apiUrl: process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8000/v1',
          vnpayUrl: process.env.NEXT_PUBLIC_VNPAY_URL,
          isDev: process.env.NODE_ENV !== 'production',
        }),
      },
    ];
  },
};

export default withNextIntl(nextConfig);
