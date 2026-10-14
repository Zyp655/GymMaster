'use client';

import { useEffect } from 'react';
import { signOut, useSession } from 'next-auth/react';
import { setClientAccessToken } from '@/lib/session-token';

/**
 * - Đồng bộ access token mới nhất vào apiClient.
 * - RefreshTokenGuard: refresh thất bại (refresh token hết hạn/bị thu hồi) → đăng xuất về /login.
 */
export function SessionSync() {
  const { data: session } = useSession();

  useEffect(() => {
    setClientAccessToken(session?.accessToken ?? null);
  }, [session?.accessToken]);

  useEffect(() => {
    if (session?.error === 'RefreshAccessTokenError') {
      setClientAccessToken(null);
      const returnUrl = window.location.pathname + window.location.search;
      void signOut({
        callbackUrl: `/login?expired=1&returnUrl=${encodeURIComponent(returnUrl)}`,
      });
    }
  }, [session?.error]);

  return null;
}
