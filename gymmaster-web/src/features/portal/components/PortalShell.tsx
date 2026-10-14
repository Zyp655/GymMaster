'use client';

import { useState, type ReactNode } from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { signOut } from 'next-auth/react';
import { useTranslations } from 'next-intl';
import {
  ArrowRightStartOnRectangleIcon,
  Bars3Icon,
  BellIcon,
  CalendarDaysIcon,
  ChartBarIcon,
  ClockIcon,
  CreditCardIcon,
  DocumentTextIcon,
  HomeIcon,
  KeyIcon,
  QrCodeIcon,
  ShieldExclamationIcon,
  UserCircleIcon,
  UserGroupIcon,
} from '@heroicons/react/24/outline';
import { Logo } from '@/components/layout/Logo';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetTitle,
  SheetTrigger,
} from '@/components/ui/sheet';
import { ChangePasswordDialog } from '@/features/auth/components/ChangePasswordDialog';
import { initials } from '@/lib/format';
import { cn } from '@/lib/utils';
import type { AuthUser } from '@/types/auth';
import { useUnreadCountQuery } from '../queries';

export function PortalShell({ user, children }: { user: AuthUser; children: ReactNode }) {
  const t = useTranslations('portal');
  const tc = useTranslations('common');
  const pathname = usePathname();
  const isMember = Boolean(user.roles?.includes('MEMBER'));
  const [passwordOpen, setPasswordOpen] = useState(false);
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  const unreadQuery = useUnreadCountQuery(isMember);
  const unreadCount = unreadQuery.data?.count ?? 0;

  const navItems = [
    { href: '/me', label: t('nav.overview'), icon: HomeIcon, exact: true },
    { href: '/me/qr', label: t('nav.qr'), icon: QrCodeIcon },
    { href: '/me/subscriptions', label: t('nav.subscriptions'), icon: CreditCardIcon },
    { href: '/me/classes', label: t('nav.classes'), icon: CalendarDaysIcon },
    { href: '/me/pt', label: t('nav.pt'), icon: UserGroupIcon },
    { href: '/me/progress', label: t('nav.progress'), icon: ChartBarIcon },
    { href: '/me/invoices', label: t('nav.invoices'), icon: DocumentTextIcon },
  ];

  const bottomNavItems = navItems.slice(0, 5);

  const isActive = (href: string, exact?: boolean) =>
    exact ? pathname === href : pathname === href || pathname.startsWith(`${href}/`);

  return (
    <div className="flex min-h-dvh flex-col bg-canvas pb-16 lg:pb-0">
      <header className="sticky top-0 z-40 border-b border-line bg-surface/90 backdrop-blur">
        <div className="mx-auto flex h-16 max-w-6xl items-center justify-between gap-3 px-4 sm:px-6">
          <div className="flex items-center gap-3">
            {isMember ? (
              <Sheet open={mobileMenuOpen} onOpenChange={setMobileMenuOpen}>
                <SheetTrigger asChild>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="lg:hidden"
                    aria-label={t('nav.menu')}
                  >
                    <Bars3Icon />
                  </Button>
                </SheetTrigger>
                <SheetContent side="left" closeLabel={tc('actions.close')} className="p-5">
                  <div className="mb-6 flex items-center gap-2">
                    <Logo />
                  </div>
                  <SheetTitle className="sr-only">{t('nav.menu')}</SheetTitle>
                  <SheetDescription className="sr-only">{t('metaTitle')}</SheetDescription>
                  <nav className="grid gap-1">
                    {navItems.map(item => {
                      const Icon = item.icon;
                      const active = isActive(item.href, item.exact);
                      return (
                        <SheetClose asChild key={item.href}>
                          <Link
                            href={item.href}
                            className={cn(
                              'flex items-center gap-3 rounded-xl px-3 py-2.5 text-body font-medium transition-colors',
                              active
                                ? 'bg-brand-50 text-brand-700'
                                : 'text-secondary hover:bg-subtle hover:text-primary',
                            )}
                          >
                            <Icon className="size-5 shrink-0" />
                            <span>{item.label}</span>
                          </Link>
                        </SheetClose>
                      );
                    })}
                    <SheetClose asChild>
                      <Link
                        href="/me/notifications"
                        className={cn(
                          'flex items-center justify-between rounded-xl px-3 py-2.5 text-body font-medium transition-colors',
                          isActive('/me/notifications')
                            ? 'bg-brand-50 text-brand-700'
                            : 'text-secondary hover:bg-subtle hover:text-primary',
                        )}
                      >
                        <span className="flex items-center gap-3">
                          <BellIcon className="size-5 shrink-0" />
                          <span>{t('nav.notifications')}</span>
                        </span>
                        {unreadCount > 0 ? (
                          <span className="rounded-full bg-brand-500 px-2 py-0.5 text-caption font-semibold text-white">
                            {unreadCount}
                          </span>
                        ) : null}
                      </Link>
                    </SheetClose>
                  </nav>
                </SheetContent>
              </Sheet>
            ) : null}

            <Link href="/me" aria-label={tc('appName')}>
              <Logo />
            </Link>
          </div>

          {isMember ? (
            <nav aria-label={t('nav.menu')} className="hidden items-center gap-1 lg:flex">
              {navItems.map(item => {
                const Icon = item.icon;
                const active = isActive(item.href, item.exact);
                return (
                  <Link
                    key={item.href}
                    href={item.href}
                    className={cn(
                      'inline-flex items-center gap-1.5 rounded-button px-3 py-2 text-body font-medium transition-colors',
                      active
                        ? 'bg-brand-50 text-brand-700'
                        : 'text-secondary hover:bg-subtle hover:text-primary',
                    )}
                  >
                    <Icon className="size-4 shrink-0" />
                    <span>{item.label}</span>
                  </Link>
                );
              })}
            </nav>
          ) : null}

          <div className="flex items-center gap-2">
            {isMember ? (
              <Button
                variant="ghost"
                size="icon"
                asChild
                className="relative"
                aria-label={
                  unreadCount > 0
                    ? t('nav.unreadCount', { count: unreadCount })
                    : t('nav.notifications')
                }
              >
                <Link href="/me/notifications">
                  <BellIcon />
                  {unreadCount > 0 ? (
                    <span className="absolute top-1 right-1 flex h-4 min-w-4 items-center justify-center rounded-full bg-brand-500 px-1 text-caption font-semibold text-white">
                      {unreadCount > 99 ? '99+' : unreadCount}
                    </span>
                  ) : null}
                </Link>
              </Button>
            ) : null}

            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <button
                  type="button"
                  className="flex cursor-pointer items-center gap-2 rounded-full p-1 transition-colors hover:bg-subtle focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none"
                >
                  <Avatar className="size-9">
                    {user.avatar_url ? (
                      <AvatarImage src={user.avatar_url} alt={user.full_name} />
                    ) : null}
                    <AvatarFallback>{initials(user.full_name)}</AvatarFallback>
                  </Avatar>
                  <span className="hidden max-w-36 truncate pr-2 text-body font-medium text-primary sm:inline">
                    {user.full_name}
                  </span>
                </button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-60">
                <DropdownMenuLabel className="grid gap-0.5">
                  <span className="truncate font-semibold text-primary">{user.full_name}</span>
                  <span className="truncate text-caption font-normal text-secondary">
                    {user.email}
                  </span>
                </DropdownMenuLabel>
                <DropdownMenuSeparator />
                {isMember ? (
                  <>
                    <DropdownMenuItem asChild>
                      <Link href="/me/account">
                        <UserCircleIcon />
                        <span>{t('nav.account')}</span>
                      </Link>
                    </DropdownMenuItem>
                    <DropdownMenuItem asChild>
                      <Link href="/me/history">
                        <ClockIcon />
                        <span>{t('nav.history')}</span>
                      </Link>
                    </DropdownMenuItem>
                  </>
                ) : null}
                <DropdownMenuItem asChild>
                  <Link href="/">
                    <HomeIcon />
                    <span>{t('nav.home')}</span>
                  </Link>
                </DropdownMenuItem>
                <DropdownMenuItem onSelect={() => setPasswordOpen(true)}>
                  <KeyIcon />
                  <span>{t('nav.changePassword')}</span>
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  variant="danger"
                  onSelect={() => {
                    void signOut({ callbackUrl: '/login' });
                  }}
                >
                  <ArrowRightStartOnRectangleIcon />
                  <span>{t('nav.logout')}</span>
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
      </header>

      <main className="mx-auto w-full max-w-6xl flex-1 px-4 py-6 sm:px-6 sm:py-8">
        {isMember ? (
          children
        ) : (
          <Card className="mx-auto max-w-lg items-center p-8 text-center">
            <div className="flex size-12 items-center justify-center rounded-2xl bg-warning-50 text-warning">
              <ShieldExclamationIcon className="size-6" />
            </div>
            <div className="grid gap-1.5">
              <h1 className="text-h3 font-semibold text-primary">{t('staffNotice.title')}</h1>
              <p className="text-body text-secondary">
                {t('staffNotice.description', { email: user.email })}
              </p>
            </div>
            <div className="flex flex-wrap items-center justify-center gap-3 pt-2">
              <Button variant="outline" asChild>
                <Link href="/">{t('staffNotice.backLanding')}</Link>
              </Button>
              <Button
                onClick={() => {
                  void signOut({ callbackUrl: '/login' });
                }}
              >
                <ArrowRightStartOnRectangleIcon />
                {t('staffNotice.logout')}
              </Button>
            </div>
          </Card>
        )}
      </main>

      {isMember ? (
        <nav
          aria-label={t('nav.menu')}
          className="fixed inset-x-0 bottom-0 z-30 border-t border-line bg-surface lg:hidden"
        >
          <div className="grid grid-cols-5">
            {bottomNavItems.map(item => {
              const Icon = item.icon;
              const active = isActive(item.href, item.exact);
              return (
                <Link
                  key={item.href}
                  href={item.href}
                  className={cn(
                    'flex flex-col items-center justify-center gap-1 py-2 text-caption font-medium transition-colors',
                    active ? 'text-brand-600' : 'text-tertiary hover:text-primary',
                  )}
                >
                  <Icon className="size-5 shrink-0" />
                  <span className="max-w-full truncate px-1">{item.label}</span>
                </Link>
              );
            })}
          </div>
        </nav>
      ) : null}

      <ChangePasswordDialog open={passwordOpen} onOpenChange={setPasswordOpen} />
    </div>
  );
}
