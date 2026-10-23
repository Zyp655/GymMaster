/** Tiện ích thuần cho màn hình Kiosk (test được bằng Node, không phụ thuộc React/DOM). */

export const DEVICE_KEY_STORAGE = 'gymmaster:kiosk_device_key';
/**
 * Khóa thiết bị demo (khớp seed) – chỉ có ở bản phát triển; bản build production thay NODE_ENV
 * lúc biên dịch nên hằng là chuỗi rỗng, khóa demo không lọt vào bundle (C10 / FF-03).
 */
export const DEFAULT_DEMO_KEY = process.env.NODE_ENV !== 'production' ? 'GMD-demo-kiosk-cg01' : '';

/** Mã lỗi API khi khóa thiết bị sai / thiết bị bị vô hiệu hóa (rescode.DeviceUnauthorized). */
export const DEVICE_UNAUTHORIZED_CODE = 40106;

export type HeartbeatFailure = 'invalid_key' | 'offline';

/**
 * Phân loại lỗi heartbeat: API trả 40106 / HTTP 401–403 → khóa thiết bị sai hoặc bị vô hiệu hóa;
 * các lỗi khác (mạng, 5xx, timeout) → mất kết nối máy chủ (FF-02).
 */
export function classifyHeartbeatError(err: unknown): HeartbeatFailure {
  if (err && typeof err === 'object') {
    const { code, status } = err as { code?: unknown; status?: unknown };
    if (code === DEVICE_UNAUTHORIZED_CODE) return 'invalid_key';
    if (status === 401 || status === 403) return 'invalid_key';
  }
  return 'offline';
}

/** Mã vé dùng thử nhanh (chỉ hiển thị ở môi trường phát triển – khớp dữ liệu seed demo). */
export const DEV_QUICK_CODES = ['DP-DEMO202601', 'TP-DEMO0001'] as const;

/**
 * Che khóa thiết bị khi hiển thị trên màn hình Kiosk (ai đứng trước máy cũng nhìn thấy):
 * giữ tiền tố "GMD-" (nếu có) và 4 ký tự cuối, phần giữa thay bằng "••••".
 */
export function maskDeviceKey(key: string | null | undefined): string {
  const k = (key ?? '').trim();
  if (!k) return '';
  if (k.length <= 8) return '••••';
  const prefix = k.startsWith('GMD-') ? 'GMD-' : '';
  return `${prefix}••••${k.slice(-4)}`;
}

/** Đọc khóa thiết bị đã lưu (chỉ gọi phía trình duyệt, trong useEffect – tránh lệch hydrate). */
export function readStoredDeviceKey(
  storage: Pick<Storage, 'getItem'> | null | undefined,
  isProd = process.env.NODE_ENV === 'production',
): string {
  const fallback = isProd ? '' : DEFAULT_DEMO_KEY;
  if (!storage) return fallback;
  try {
    const saved = storage.getItem(DEVICE_KEY_STORAGE);
    return saved && saved.trim() ? saved.trim() : fallback;
  } catch {
    return fallback;
  }
}
