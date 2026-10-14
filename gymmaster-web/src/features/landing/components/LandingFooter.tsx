import Link from 'next/link';
import { getTranslations } from 'next-intl/server';
import { Logo } from '@/components/layout/Logo';
import { Button } from '@/components/ui/button';

export async function LandingFooter() {
  const t = await getTranslations('common.landing');
  return (
    <footer className="bg-ink-950">
      <div className="mx-auto grid max-w-6xl gap-10 px-4 py-16 sm:px-6">
        <div className="flex flex-col gap-6 rounded-3xl bg-brand-500 p-8 sm:flex-row sm:items-center sm:justify-between">
          <div className="grid max-w-xl gap-2">
            <h2 className="text-h2 font-semibold text-white">{t('cta.title')}</h2>
            <p className="text-body-lg text-brand-50">{t('cta.subtitle')}</p>
          </div>
          <Button asChild size="lg" variant="dark">
            <Link href="/login">{t('cta.login')}</Link>
          </Button>
        </div>
        <div className="flex flex-col gap-4 border-t border-ink-800 pt-8 sm:flex-row sm:items-center sm:justify-between">
          <div className="grid gap-1">
            <Logo tone="light" />
            <span className="text-caption text-tertiary">{t('footer.tagline')}</span>
          </div>
          <p className="text-caption text-tertiary">
            {t('footer.copyright', { year: new Date().getFullYear() })}
          </p>
        </div>
      </div>
    </footer>
  );
}
