/**
 * Header bảo mật cho mọi response của gymmaster-web (dùng trong next.config.ts → headers()).
 * CSP không dùng nonce (trang render động lẫn tĩnh) nên script/style cần 'unsafe-inline';
 * dev thêm 'unsafe-eval' (React debug) và ws: cho HMR.
 */
export interface SecurityHeaderOptions {
  /** NEXT_PUBLIC_API_URL – origin được phép fetch/SSE/ảnh upload. */
  apiUrl?: string;
  /** Origin cổng VNPay (form-action) – mặc định sandbox. */
  vnpayUrl?: string;
  isDev: boolean;
}

function originOf(url: string | undefined): string | null {
  if (!url) return null;
  try {
    return new URL(url).origin;
  } catch {
    return null;
  }
}

export function buildContentSecurityPolicy({ apiUrl, vnpayUrl, isDev }: SecurityHeaderOptions) {
  const api = originOf(apiUrl);
  const vnpay = originOf(vnpayUrl) ?? 'https://sandbox.vnpayment.vn';
  const connect = ["'self'", api, isDev ? 'ws:' : null, isDev ? 'wss:' : null];
  const directives: Record<string, (string | null)[]> = {
    'default-src': ["'self'"],
    'script-src': ["'self'", "'unsafe-inline'", isDev ? "'unsafe-eval'" : null],
    'style-src': ["'self'", "'unsafe-inline'"],
    // Ảnh đại diện/ảnh bìa có thể nằm ở API (uploads) hoặc CDN https.
    'img-src': ["'self'", 'data:', 'blob:', api, 'https:'],
    'font-src': ["'self'", 'data:'],
    'connect-src': connect,
    'media-src': ["'self'", 'blob:'],
    'object-src': ["'none'"],
    'base-uri': ["'self'"],
    'form-action': ["'self'", vnpay],
    'frame-ancestors': ["'none'"],
    'frame-src': ["'none'"],
    'worker-src': ["'self'", 'blob:'],
    'manifest-src': ["'self'"],
  };
  const parts = Object.entries(directives).map(([name, values]) =>
    [name, ...Array.from(new Set(values.filter((v): v is string => Boolean(v))))].join(' '),
  );
  // Chỉ nâng cấp http→https khi chạy production với API https (dev/localhost dùng http).
  if (!isDev && api?.startsWith('https://')) parts.push('upgrade-insecure-requests');
  return parts.join('; ');
}

export function buildSecurityHeaders(options: SecurityHeaderOptions) {
  return [
    { key: 'Content-Security-Policy', value: buildContentSecurityPolicy(options) },
    { key: 'X-Frame-Options', value: 'DENY' },
    { key: 'X-Content-Type-Options', value: 'nosniff' },
    { key: 'Referrer-Policy', value: 'strict-origin-when-cross-origin' },
    {
      key: 'Permissions-Policy',
      value: 'camera=(self), microphone=(), geolocation=(), payment=(), usb=()',
    },
    { key: 'Cross-Origin-Opener-Policy', value: 'same-origin' },
  ];
}
