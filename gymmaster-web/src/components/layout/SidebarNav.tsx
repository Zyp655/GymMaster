'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { useAuth } from '@/hooks/use-auth';
import { hasAnyPermission } from '@/lib/permissions';
import { cn } from '@/lib/utils';
import { NAV_GROUPS } from './nav-config';

/** Menu điều hướng: chỉ hiện module đã làm và người dùng có quyền xem. */
export function SidebarNav({
  collapsed = false,
  onNavigate,
}: {
  collapsed?: boolean;
  onNavigate?: () => void;
}) {
  const t = useTranslations('layout.nav');
  const pathname = usePathname();
  const { user } = useAuth();

  const groups = NAV_GROUPS.map(group => ({
    ...group,
    items: group.items.filter(
      item => item.enabled && (!item.anyOf || hasAnyPermission(user, item.anyOf)),
    ),
  })).filter(group => group.items.length > 0);

  return (
    <nav className="grid gap-6">
      {groups.map(group => (
        <div key={group.key} className="grid gap-1">
          {collapsed ? (
            <div className="mx-auto mb-1 h-px w-6 bg-ink-700" />
          ) : (
            <p className="px-3 pb-1 text-caption font-medium tracking-wide text-disabled uppercase">
              {t(group.key)}
            </p>
          )}
          {group.items.map(item => {
            const active = pathname === item.href || pathname.startsWith(`${item.href}/`);
            const Icon = item.icon;
            const link = (
              <Link
                href={item.href}
                onClick={onNavigate}
                aria-current={active ? 'page' : undefined}
                className={cn(
                  'flex h-10 items-center gap-3 rounded-xl px-3 text-body font-medium transition-colors',
                  collapsed && 'justify-center px-0',
                  active
                    ? 'bg-brand-500 text-white'
                    : 'text-disabled hover:bg-ink-800 hover:text-white',
                )}
              >
                <Icon className="size-5 shrink-0" />
                {collapsed ? (
                  <span className="sr-only">{t(item.key)}</span>
                ) : (
                  <span className="truncate">{t(item.key)}</span>
                )}
              </Link>
            );
            return collapsed ? (
              <Tooltip key={item.key}>
                <TooltipTrigger asChild>{link}</TooltipTrigger>
                <TooltipContent side="right">{t(item.key)}</TooltipContent>
              </Tooltip>
            ) : (
              <div key={item.key}>{link}</div>
            );
          })}
        </div>
      ))}
    </nav>
  );
}
