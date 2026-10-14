import type { DefaultSession } from 'next-auth';
import type { AuthUser } from './auth';

declare module 'next-auth' {
  interface Session {
    user: AuthUser & DefaultSession['user'];
    accessToken: string;
    accessTokenExpires: number;
    error?: 'RefreshAccessTokenError';
  }

  /** Object trả về từ authorize() của Credentials provider. */
  interface User {
    id: string;
    authUser: AuthUser;
    accessToken: string;
    refreshToken: string;
    accessTokenExpires: number;
  }
}

declare module 'next-auth/jwt' {
  interface JWT {
    user: AuthUser;
    accessToken: string;
    refreshToken: string;
    accessTokenExpires: number;
    error?: 'RefreshAccessTokenError';
  }
}
