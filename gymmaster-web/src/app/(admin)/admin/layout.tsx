import { redirect } from 'next/navigation';
import { AdminShell } from '@/components/layout/AdminShell';
import { getCachedSession } from '@/guards/getCachedSession';

/** Lớp bảo vệ thứ hai (sau proxy.ts): không có phiên hợp lệ → về trang đăng nhập. */
export default async function AdminLayout({ children }: { children: React.ReactNode }) {
  const session = await getCachedSession();
  if (!session || session.error) redirect('/login');
  return <AdminShell>{children}</AdminShell>;
}
