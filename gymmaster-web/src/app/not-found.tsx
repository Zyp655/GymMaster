import Link from 'next/link';
import { getTranslations } from 'next-intl/server';
import { Button } from '@/components/ui/button';
import { getCachedSession } from '@/guards/getCachedSession';

export default async function NotFound() {
  const t = await getTranslations('common.notFound');
  const session = await getCachedSession().catch(() => null);
  const isLoggedIn = Boolean(session && !session.error);
  return (
    <div className="flex min-h-dvh flex-col items-center justify-center gap-4 px-6 text-center">
      <p className="text-display font-semibold text-brand-500">404</p>
      <div className="grid max-w-md gap-2">
        <h1 className="text-h2 font-semibold text-primary">{t('title')}</h1>
        <p className="text-body text-secondary">{t('description')}</p>
      </div>
      <div className="flex flex-wrap items-center justify-center gap-3">
        <Button asChild>
          <Link href="/">{t('backHome')}</Link>
        </Button>
        {isLoggedIn ? (
          <Button variant="outline" asChild>
            <Link href="/me">{t('backDashboard')}</Link>
          </Button>
        ) : null}
      </div>
    </div>
  );
}
