/**
 * Lưu token phía trình duyệt.
 *
 * - access token: giữ trong bộ nhớ (đọc nhanh, mỗi request) + localStorage (để F5 không phải refresh ngay).
 * - refresh token: localStorage.
 *
 * Trade-off: localStorage đọc được bằng JS nên kém an toàn hơn cookie httpOnly nếu bị XSS.
 * Backend hiện trả refresh token trong body (SPA thuần) nên chấp nhận cách này; xem README.
 */
export interface TokenStore {
  getAccessToken(): string | null;
  getRefreshToken(): string | null;
  setTokens(accessToken: string, refreshToken: string): void;
  clear(): void;
}

const ACCESS_KEY = 'gm_access_token';
const REFRESH_KEY = 'gm_refresh_token';

let memoryAccess: string | null = null;

function safeGet(key: string): string | null {
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
}

function safeSet(key: string, value: string | null): void {
  try {
    if (value === null) localStorage.removeItem(key);
    else localStorage.setItem(key, value);
  } catch {
    /* private mode – chỉ giữ trong bộ nhớ */
  }
}

export const tokenStore: TokenStore = {
  getAccessToken() {
    if (memoryAccess === null) memoryAccess = safeGet(ACCESS_KEY);
    return memoryAccess;
  },
  getRefreshToken() {
    return safeGet(REFRESH_KEY);
  },
  setTokens(accessToken, refreshToken) {
    memoryAccess = accessToken;
    safeSet(ACCESS_KEY, accessToken);
    safeSet(REFRESH_KEY, refreshToken);
  },
  clear() {
    memoryAccess = null;
    safeSet(ACCESS_KEY, null);
    safeSet(REFRESH_KEY, null);
  },
};
