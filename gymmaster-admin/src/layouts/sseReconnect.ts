/**
 * Kết nối SSE thông báo có tự nối lại (A3):
 * - Server gửi `auth_expired` / `auth_revoked` rồi đóng luồng → refresh token rồi mở lại.
 * - `onerror` (mất mạng, server khởi động lại, token hết hạn lúc nối) → đóng và mở lại theo
 *   backoff 1s, 2s, 4s… tối đa 60s; lỗi liên tiếp từ lần 2 thì refresh token trước khi nối.
 * - Refresh thất bại (refresh token hết hạn/thu hồi) → dừng hẳn; request API kế tiếp sẽ đưa về đăng nhập.
 * Thuần (không phụ thuộc React) để kiểm thử bằng Node.
 */

export const SSE_BACKOFF_MAX_MS = 60_000;

/** Thời gian chờ trước lần nối lại thứ `attempt` (0-based). */
export function sseBackoffDelay(attempt: number, baseMs = 1000): number {
  return Math.min(baseMs * 2 ** Math.max(0, attempt), SSE_BACKOFF_MAX_MS);
}

export interface SseLike {
  addEventListener(type: string, listener: (event: MessageEvent) => void): void;
  onmessage: ((event: MessageEvent) => void) | null;
  onerror: ((event: Event) => void) | null;
  onopen: ((event: Event) => void) | null;
  close(): void;
}

export interface SseConnectOptions {
  /** URL kèm access token hiện tại; `null` = chưa có token → không nối. */
  buildUrl: () => string | null;
  /** Refresh access token (single-flight dùng chung với interceptor). */
  refresh: () => Promise<unknown>;
  onEvent: (event: MessageEvent) => void;
  createSource?: (url: string) => SseLike;
  setTimer?: (fn: () => void, ms: number) => unknown;
  clearTimer?: (t: unknown) => void;
}

/** Mở kết nối; trả hàm dừng (gọi khi unmount/đăng xuất). */
export function connectSse(opts: SseConnectOptions): () => void {
  const create = opts.createSource ?? ((url: string) => new EventSource(url) as unknown as SseLike);
  const setTimer = opts.setTimer ?? ((fn, ms) => setTimeout(fn, ms));
  const clearTimer = opts.clearTimer ?? ((t) => clearTimeout(t as ReturnType<typeof setTimeout>));

  let es: SseLike | null = null;
  let timer: unknown = null;
  let stopped = false;
  let failures = 0;

  const closeCurrent = () => {
    if (!es) return;
    es.onerror = null;
    es.onmessage = null;
    es.onopen = null;
    es.close();
    es = null;
  };

  const open = () => {
    if (stopped) return;
    const url = opts.buildUrl();
    if (!url) return;
    try {
      es = create(url);
    } catch {
      schedule(false);
      return;
    }
    es.onopen = () => {
      failures = 0;
    };
    es.addEventListener('notification', opts.onEvent);
    es.onmessage = opts.onEvent;
    // Token hết hạn / bị thu hồi: refresh ngay rồi nối lại.
    es.addEventListener('auth_expired', () => schedule(true, 0));
    es.addEventListener('auth_revoked', () => schedule(true, 0));
    es.onerror = () => {
      failures += 1;
      schedule(failures >= 2);
    };
  };

  /** Đóng kết nối hiện tại và hẹn nối lại (một lịch duy nhất). */
  const schedule = (needRefresh: boolean, delay?: number) => {
    closeCurrent();
    if (stopped || timer !== null) return;
    const wait = delay ?? sseBackoffDelay(Math.max(0, failures - 1));
    timer = setTimer(() => {
      timer = null;
      if (stopped) return;
      if (!needRefresh) {
        open();
        return;
      }
      opts.refresh().then(
        () => open(),
        () => {
          // Refresh token không còn hiệu lực → dừng; phiên sẽ kết thúc ở request API kế tiếp.
          stopped = true;
        },
      );
    }, wait);
  };

  open();
  return () => {
    stopped = true;
    if (timer !== null) clearTimer(timer);
    timer = null;
    closeCurrent();
  };
}
