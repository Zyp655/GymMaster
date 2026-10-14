/**
 * Bộ nhớ đệm access token phía client, được đồng bộ từ useSession() bởi <SessionSync />.
 * Giúp apiClient không phải gọi /api/auth/session cho mỗi request.
 */
let accessToken: string | null = null;

export function setClientAccessToken(token: string | null) {
  accessToken = token;
}

export function getClientAccessToken() {
  return accessToken;
}
