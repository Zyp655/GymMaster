import Link from 'next/link';
import { getTranslations } from 'next-intl/server';
import { ArrowRightEndOnRectangleIcon } from '@heroicons/react/24/outline';
import { Logo } from '@/components/layout/Logo';
import { Button } from '@/components/ui/button';
import { LandingMobileNav } from './LandingMobileNav';

export async function LandingHeader({ isAuthenticated }: { isAuthenticated: boolean }) {
  const [t, tl] = await Promise.all([
    getTranslations('common.landing.nav'),
    getTranslations('landing.nav'),
  ]);
  const links = [
    { href: '/#bang-gia', label: t('pricing') },
    { href: '/#huan-luyen-ca-nhan', label: t('pt') },
    { href: '/#lich-lop', label: t('schedules') },
    { href: '/trainers', label: tl('trainers') },
    { href: '/blog', label: tl('blog') },
    { href: '/#chi-nhanh', label: t('branches') },
    { href: '/#tap-thu', label: t('trial') },
  ];

  return (
    <header className="sticky top-0 z-40 border-b border-line bg-surface/90 backdrop-blur">
      <div className="mx-auto flex h-16 max-w-6xl items-center justify-between gap-4 px-4 sm:px-6">
        <div className="flex items-center gap-1">
          <LandingMobileNav
            links={links}
            labels={{ open: tl('openMenu'), close: tl('closeMenu'), title: tl('menuTitle') }}
          />
          <Link href="/" aria-label="GymMaster">
            <Logo />
          </Link>
        </div>
        <nav aria-label={t('menu')} className="hidden items-center gap-1 lg:flex">
          {links.map(link => (
            <a
              key={link.href}
              href={link.href}
              className="rounded-button px-3 py-2 text-body font-medium text-secondary transition-colors hover:bg-subtle hover:text-primary"
            >
              {link.label}
            </a>
          ))}
        </nav>
        <Button asChild>
          {/* Đã đăng nhập: /login được proxy chuyển thẳng vào hệ thống. */}
          <Link href="/login">
            <ArrowRightEndOnRectangleIcon />
            {isAuthenticated ? t('enter') : t('login')}
          </Link>
        </Button>
      </div>
    </header>
  );
}
