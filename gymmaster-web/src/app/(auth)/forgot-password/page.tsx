import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { CheckCircleIcon } from '@heroicons/react/24/solid';
import { Logo } from '@/components/layout/Logo';
import { ForgotPasswordForm } from '@/features/auth/components/ForgotPasswordForm';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('auth.forgotPassword');
  return { title: t('title') };
}

export default async function ForgotPasswordPage() {
  const t = await getTranslations('auth');
  const features = [t('brand.feature1'), t('brand.feature2'), t('brand.feature3')];

  return (
    <div className="grid min-h-dvh lg:grid-cols-2">
      <section className="relative hidden flex-col justify-between overflow-hidden bg-ink-950 p-12 lg:flex">
        <div className="pointer-events-none absolute -top-32 -right-32 size-96 rounded-full bg-brand-500/20 blur-3xl" />
        <div className="pointer-events-none absolute -bottom-40 -left-20 size-96 rounded-full bg-brand-700/20 blur-3xl" />
        <Logo tone="light" className="relative" />
        <div className="relative grid max-w-lg gap-8">
          <h2 className="text-display font-semibold text-white">{t('brand.tagline')}</h2>
          <ul className="grid gap-4">
            {features.map(feature => (
              <li key={feature} className="flex items-start gap-3 text-body-lg text-disabled">
                <CheckCircleIcon className="mt-0.5 size-6 shrink-0 text-brand-500" />
                <span>{feature}</span>
              </li>
            ))}
          </ul>
        </div>
        <p className="relative text-caption text-tertiary">
          {t('brand.copyright', { year: new Date().getFullYear() })}
        </p>
      </section>

      <section className="flex items-center justify-center bg-surface px-6 py-12">
        <div className="grid w-full max-w-sm gap-8">
          <Logo className="lg:hidden" />
          <div className="grid gap-2">
            <h1 className="text-h1 font-semibold text-primary">{t('forgotPassword.title')}</h1>
            <p className="text-body text-secondary">{t('forgotPassword.subtitle')}</p>
          </div>
          <ForgotPasswordForm />
        </div>
      </section>
    </div>
  );
}
