import type { NextConfig } from 'next';
import createNextIntlPlugin from 'next-intl/plugin';

const withNextIntl = createNextIntlPlugin('./src/i18n/request.ts');

const nextConfig: NextConfig = {
  output: 'standalone',
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
};

export default withNextIntl(nextConfig);
