/**
 * Chống open-redirect: chỉ chấp nhận đường dẫn nội bộ tương đối bắt đầu bằng MỘT dấu "/".
 * Từ chối "//host", "/\host" (trình duyệt coi "\" như "/"), "javascript:", URL tuyệt đối,
 * và các ký tự điều khiển (tab/xuống dòng bị trình duyệt bỏ qua: "/\t/host" → "//host").
 */
const PLACEHOLDER_ORIGIN = 'http://gymmaster.invalid';

export function isSafeRelativePath(value: string | null | undefined): value is string {
  if (typeof value !== 'string' || value.length === 0 || value.length > 2048) return false;
  if (/[\u0000-\u001F\u007F\\]/.test(value)) return false;
  if (!value.startsWith('/') || value.startsWith('//')) return false;
  try {
    // Đối chiếu bằng trình phân tích URL chuẩn: phải giữ nguyên origin giả lập.
    return new URL(value, PLACEHOLDER_ORIGIN).origin === PLACEHOLDER_ORIGIN;
  } catch {
    return false;
  }
}

/** Trả `value` nếu là đường dẫn nội bộ an toàn, ngược lại trả `fallback` (mặc định "/me"). */
export function safeReturnUrl(value: string | null | undefined, fallback = '/me'): string {
  return isSafeRelativePath(value) ? value : fallback;
}

/**
 * Dùng cho callback `redirect` của NextAuth: chấp nhận đường dẫn tương đối an toàn hoặc URL tuyệt đối
 * CÙNG origin với `baseUrl` (NextAuth client hay gửi `window.location.href`); còn lại về `baseUrl`.
 */
export function resolveAuthRedirect(url: string, baseUrl: string): string {
  const base = baseUrl.replace(/\/+$/, '');
  if (isSafeRelativePath(url)) return `${base}${url}`;
  try {
    const target = new URL(url);
    const origin = new URL(base).origin;
    if (target.origin === origin) {
      const path = `${target.pathname}${target.search}${target.hash}`;
      if (isSafeRelativePath(path)) return `${base}${path}`;
    }
  } catch {
    // URL không hợp lệ → về trang gốc.
  }
  return base;
}
