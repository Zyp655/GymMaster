'use client';

import { useState } from 'react';
import Link from 'next/link';
import { signOut } from 'next-auth/react';
import { useTranslations } from 'next-intl';
import {
  ArrowRightStartOnRectangleIcon,
  BellIcon,
  ChartBarIcon,
  ChevronRightIcon,
  ClockIcon,
  DocumentTextIcon,
  KeyIcon,
} from '@heroicons/react/24/outline';
import { LoadError } from '@/components/common/LoadError';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { ChangePasswordDialog } from '@/features/auth/components/ChangePasswordDialog';
import { setClientAccessToken } from '@/lib/session-token';
import { useProfileQuery } from '../queries';
import { PortalPageHeader } from './PortalPageHeader';
import { ProfileFormCard } from './ProfileFormCard';

/** Tài khoản: hồ sơ (sửa trường được phép), lối tắt, đổi mật khẩu, đăng xuất. */
export function AccountView() {
  const t = useTranslations('portal.account');
  const tn = useTranslations('portal.nav');
  const query = useProfileQuery();
  const [passwordOpen, setPasswordOpen] = useState(false);
  const [loggingOut, setLoggingOut] = useState(false);

  const links = [
    { href: '/me/progress', label: tn('progress'), icon: ChartBarIcon },
    { href: '/me/history', label: tn('history'), icon: ClockIcon },
    { href: '/me/invoices', label: tn('invoices'), icon: DocumentTextIcon },
    { href: '/me/notifications', label: tn('notifications'), icon: BellIcon },
  ];

  const logout = async () => {
    setLoggingOut(true);
    setClientAccessToken(null);
    await signOut({ callbackUrl: '/login' });
  };

  return (
    <div className="grid gap-4">
      <PortalPageHeader title={t('title')} subtitle={t('subtitle')} />

      <Card className="gap-0 p-2 lg:hidden">
        <ul className="grid">
          {links.map(link => {
            const Icon = link.icon;
            return (
              <li key={link.href}>
                <Link
                  href={link.href}
                  className="flex items-center gap-3 rounded-xl px-3 py-3 text-body font-medium text-primary transition-colors hover:bg-subtle"
                >
                  <Icon className="size-5 shrink-0 text-tertiary" />
                  <span className="flex-1">{link.label}</span>
                  <ChevronRightIcon className="size-4 text-tertiary" />
                </Link>
              </li>
            );
          })}
        </ul>
      </Card>

      {query.isLoading ? (
        <Skeleton className="h-96 rounded-2xl" />
      ) : query.isError || !query.data ? (
        <Card>
          <LoadError error={query.error} onRetry={() => void query.refetch()} />
        </Card>
      ) : (
        <ProfileFormCard profile={query.data} />
      )}

      <Card className="gap-3">
        <h2 className="text-h4 font-semibold text-primary">{t('securityTitle')}</h2>
        <div className="grid gap-2 sm:flex sm:flex-wrap">
          <Button variant="outline" onClick={() => setPasswordOpen(true)}>
            <KeyIcon />
            {tn('changePassword')}
          </Button>
          <Button variant="danger-ghost" onClick={() => void logout()} disabled={loggingOut}>
            <ArrowRightStartOnRectangleIcon />
            {loggingOut ? t('loggingOut') : tn('logout')}
          </Button>
        </div>
      </Card>

      <ChangePasswordDialog open={passwordOpen} onOpenChange={setPasswordOpen} />
    </div>
  );
}
