import { NextResponse, type NextRequest } from 'next/server';
import { getToken } from 'next-auth/jwt';

const LOGIN_PATH = '/login';
const HOME_PATH = '/admin/dashboard';

/**
 * Chặn route theo phiên đăng nhập (chạy trước khi render):
 * - /admin/* chưa đăng nhập → /login?returnUrl=...
 * - đã đăng nhập mà vào /login → /admin/dashboard
 * - "/" là trang công khai (landing) – ai cũng xem được, không chặn ở đây.
 * Backend vẫn là nơi kiểm tra quyền cuối cùng.
 */
export async function proxy(request: NextRequest) {
  const { pathname, search } = request.nextUrl;
  const token = await getToken({ req: request, secret: process.env.NEXTAUTH_SECRET });
  const isAuthenticated = !!token && !token.error;

  if (pathname.startsWith('/admin') && !isAuthenticated) {
    const url = new URL(LOGIN_PATH, request.url);
    url.searchParams.set('returnUrl', pathname + search);
    return NextResponse.redirect(url);
  }

  if (pathname === LOGIN_PATH && isAuthenticated) {
    return NextResponse.redirect(new URL(HOME_PATH, request.url));
  }

  return NextResponse.next();
}

export const config = {
  matcher: ['/login', '/admin/:path*'],
};
