/**
 * Đăng nhập web chạy phía server Next (NextAuth authorize) → API chỉ thấy UA "node" và IP ::1.
 * Hàm này lấy UA và IP thật của trình duyệt từ header request gốc để chuyển tiếp sang API
 * (API chỉ tin X-Forwarded-For khi IP Next server nằm trong TRUSTED_PROXIES).
 */
type HeaderBag =
  | Headers
  | Record<string, string | string[] | undefined | null>
  | null
  | undefined;

const MAX_UA_LENGTH = 512;
const IP_PATTERN = /^[0-9a-fA-F:.]{2,45}$/;

function readHeader(headers: HeaderBag, name: string): string | undefined {
  if (!headers) return undefined;
  if (typeof (headers as Headers).get === 'function') {
    return (headers as Headers).get(name) ?? undefined;
  }
  const bag = headers as Record<string, string | string[] | undefined | null>;
  const key = Object.keys(bag).find(k => k.toLowerCase() === name);
  const value = key ? bag[key] : undefined;
  if (Array.isArray(value)) return value[0];
  return value ?? undefined;
}

/** IP client đầu tiên trong X-Forwarded-For (hoặc X-Real-IP); bỏ giá trị không giống IP. */
export function clientIpFrom(headers: HeaderBag): string | undefined {
  const forwarded = readHeader(headers, 'x-forwarded-for')?.split(',')[0]?.trim();
  const candidate = forwarded || readHeader(headers, 'x-real-ip')?.trim();
  return candidate && IP_PATTERN.test(candidate) ? candidate : undefined;
}

/** User-Agent đã gỡ ký tự điều khiển, giới hạn độ dài. */
export function clientUserAgentFrom(headers: HeaderBag): string | undefined {
  const ua = readHeader(headers, 'user-agent')?.replace(/[\u0000-\u001F\u007F]/g, '').trim();
  return ua ? ua.slice(0, MAX_UA_LENGTH) : undefined;
}

/** Header chuyển tiếp cho request đăng nhập gửi tới API. */
export function clientInfoHeaders(headers: HeaderBag): Record<string, string> {
  const out: Record<string, string> = {};
  const ua = clientUserAgentFrom(headers);
  const ip = clientIpFrom(headers);
  if (ua) out['User-Agent'] = ua;
  if (ip) {
    out['X-Forwarded-For'] = ip;
    out['X-Real-IP'] = ip;
  }
  return out;
}
