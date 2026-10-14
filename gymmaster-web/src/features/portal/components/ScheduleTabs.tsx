'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { cn } from '@/lib/utils';

/** Thanh chuyển nhanh trong nhóm "Lịch tập": lớp nhóm · lớp đã đặt · PT. */
export function ScheduleTabs() {
  const t = useTranslations('portal.nav');
  const pathname = usePathname();
  const tabs = [
    { href: '/me/classes', label: t('classes') },
    { href: '/me/bookings', label: t('bookings') },
    { href: '/me/pt', label: t('pt') },
  ];
  return (
    <nav className="grid grid-cols-3 gap-1 rounded-xl bg-subtle p-1" aria-label={t('schedule')}>
      {tabs.map(tab => {
        const active = pathname === tab.href;
        return (
          <Link
            key={tab.href}
            href={tab.href}
            aria-current={active ? 'page' : undefined}
            className={cn(
              'truncate rounded-lg px-2 py-2 text-center text-body font-medium transition-colors',
              active ? 'bg-surface text-primary' : 'text-secondary hover:text-primary',
            )}
          >
            {tab.label}
          </Link>
        );
      })}
    </nav>
  );
}
