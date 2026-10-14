import Link from 'next/link';
import { getTranslations } from 'next-intl/server';
import { Button } from '@/components/ui/button';

export default async function NotFound() {
  const t = await getTranslations('common.notFound');
  return (
    <div className="flex min-h-dvh flex-col items-center justify-center gap-4 px-6 text-center">
      <p className="text-display font-semibold text-brand-500">404</p>
      <div className="grid max-w-md gap-2">
        <h1 className="text-h2 font-semibold text-primary">{t('title')}</h1>
        <p className="text-body text-secondary">{t('description')}</p>
      </div>
      <Button asChild>
        <Link href="/admin/dashboard">{t('backHome')}</Link>
      </Button>
    </div>
  );
}
