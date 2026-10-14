'use client';

import type { ReactNode } from 'react';
import type { Session } from 'next-auth';
import { SessionProvider } from 'next-auth/react';
import { Toaster } from '@/components/ui/sonner';
import { TooltipProvider } from '@/components/ui/tooltip';
import { QueryProvider } from './QueryProvider';
import { SessionSync } from './SessionSync';

export function AppProviders({
  session,
  children,
}: {
  session: Session | null;
  children: ReactNode;
}) {
  return (
    // refetchInterval: kiểm tra phiên mỗi 4 phút để refresh access token (15 phút) trước khi hết hạn.
    <SessionProvider session={session} refetchInterval={4 * 60} refetchOnWindowFocus>
      <SessionSync />
      <QueryProvider>
        <TooltipProvider>{children}</TooltipProvider>
      </QueryProvider>
      <Toaster />
    </SessionProvider>
  );
}
