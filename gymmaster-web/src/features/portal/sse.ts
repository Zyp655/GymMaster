/** Backoff kết nối lại SSE thông báo: 2s, 4s, 8s… tối đa 60s (cộng jitter ≤ 20% để tránh dồn cục). */
export const SSE_RETRY_BASE_MS = 2_000;
export const SSE_RETRY_MAX_MS = 60_000;

export function sseRetryDelay(attempt: number, random: () => number = Math.random): number {
  const n = Math.max(0, Math.floor(attempt));
  const base = Math.min(SSE_RETRY_MAX_MS, SSE_RETRY_BASE_MS * 2 ** Math.min(n, 10));
  return Math.round(base + base * 0.2 * random());
}
