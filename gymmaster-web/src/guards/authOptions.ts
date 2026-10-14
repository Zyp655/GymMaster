import 'server-only';
import type { NextAuthOptions, User } from 'next-auth';
import type { JWT } from 'next-auth/jwt';
import CredentialsProvider from 'next-auth/providers/credentials';
import { getServerEnv } from '@/env';
import { loginRequest, logoutRequest, refreshRequest } from '@/features/auth/server';
import { ApiCode, isApiError } from '@/types/api';
import type { LoginResponse } from '@/types/auth';

/** Refresh sớm 30s trước khi access token hết hạn. */
const EXPIRY_SKEW_MS = 30_000;

interface TokenSet {
  user: JWT['user'];
  accessToken: string;
  refreshToken: string;
  accessTokenExpires: number;
}

function tokenFromLogin(data: LoginResponse): TokenSet {
  return {
    user: data.user,
    accessToken: data.access_token,
    refreshToken: data.refresh_token,
    accessTokenExpires: Date.now() + data.expires_in * 1000,
  };
}

/**
 * Backend xoay vòng refresh token (token cũ bị thu hồi ngay sau khi dùng),
 * nên các request song song trong cùng process phải dùng chung MỘT lần refresh.
 */
const inflightRefresh = new Map<string, Promise<JWT>>();

function refreshAccessToken(token: JWT): Promise<JWT> {
  const key = token.refreshToken;
  const existing = inflightRefresh.get(key);
  if (existing) return existing;

  const promise = (async (): Promise<JWT> => {
    try {
      const data = await refreshRequest(token.refreshToken);
      return { ...token, ...tokenFromLogin(data), error: undefined };
    } catch {
      return { ...token, error: 'RefreshAccessTokenError' };
    } finally {
      setTimeout(() => inflightRefresh.delete(key), 10_000);
    }
  })();
  inflightRefresh.set(key, promise);
  return promise;
}

export const authOptions: NextAuthOptions = {
  secret: getServerEnv().NEXTAUTH_SECRET,
  session: { strategy: 'jwt', maxAge: 7 * 24 * 60 * 60 },
  pages: { signIn: '/login' },
  providers: [
    CredentialsProvider({
      id: 'credentials',
      name: 'credentials',
      credentials: {
        email: { type: 'email' },
        password: { type: 'password' },
      },
      async authorize(credentials): Promise<User> {
        if (!credentials?.email || !credentials.password) {
          throw new Error(String(ApiCode.INVALID_CREDENTIALS));
        }
        try {
          const data = await loginRequest(credentials.email, credentials.password);
          const token = tokenFromLogin(data);
          return {
            id: String(data.user.id),
            authUser: token.user,
            accessToken: token.accessToken,
            refreshToken: token.refreshToken,
            accessTokenExpires: token.accessTokenExpires,
          };
        } catch (error) {
          // Ném mã lỗi để client hiển thị thông báo tiếng Việt tương ứng.
          throw new Error(String(isApiError(error) ? error.code : ApiCode.INTERNAL));
        }
      },
    }),
  ],
  callbacks: {
    async jwt({ token, user, trigger, session }) {
      if (user) {
        return {
          user: user.authUser,
          accessToken: user.accessToken,
          refreshToken: user.refreshToken,
          accessTokenExpires: user.accessTokenExpires,
        };
      }

      if (trigger === 'update') {
        const data = session as { forceRefresh?: boolean; user?: Partial<JWT['user']> } | undefined;
        if (data?.forceRefresh) return refreshAccessToken(token);
        if (data?.user) token.user = { ...token.user, ...data.user };
      }

      if (token.error) return token;
      if (Date.now() < token.accessTokenExpires - EXPIRY_SKEW_MS) return token;
      return refreshAccessToken(token);
    },
    async session({ session, token }) {
      // Không bao giờ đưa refreshToken xuống client.
      session.user = { ...session.user, ...token.user };
      session.accessToken = token.accessToken;
      session.accessTokenExpires = token.accessTokenExpires;
      session.error = token.error;
      return session;
    },
  },
  events: {
    async signOut({ token }) {
      if (!token?.refreshToken) return;
      try {
        await logoutRequest(token.accessToken, token.refreshToken);
      } catch {
        // Bỏ qua: refresh token vẫn tự hết hạn phía backend.
      }
    },
  },
};
