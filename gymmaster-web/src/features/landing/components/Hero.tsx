import Link from 'next/link';
import { getTranslations } from 'next-intl/server';
import { ArrowDownIcon, ClockIcon } from '@heroicons/react/24/outline';
import { Button } from '@/components/ui/button';

export async function Hero({
  branchCount,
  planCount,
  ptCount,
  hours,
}: {
  branchCount: number;
  planCount: number;
  ptCount: number;
  hours: { open: string; close: string } | null;
}) {
  const t = await getTranslations('common.landing.hero');
  const stats = [
    { value: branchCount, label: t('statBranches') },
    { value: planCount, label: t('statPlans') },
    { value: ptCount, label: t('statPt') },
  ];

  return (
    <section className="relative overflow-hidden bg-ink-950">
      <div className="pointer-events-none absolute -top-40 -right-32 size-96 rounded-full bg-brand-500/25 blur-3xl" />
      <div className="pointer-events-none absolute -bottom-48 -left-24 size-96 rounded-full bg-brand-700/20 blur-3xl" />
      <div className="relative mx-auto grid max-w-6xl gap-12 px-4 py-20 sm:px-6 lg:grid-cols-5 lg:items-center lg:py-28">
        <div className="grid gap-6 lg:col-span-3">
          <span className="w-fit rounded-full bg-brand-500/15 px-3 py-1 text-caption font-medium text-brand-300">
            {t('eyebrow')}
          </span>
          <h1 className="text-h1 font-semibold text-balance text-white sm:text-display">
            {t('title')}
          </h1>
          <p className="max-w-xl text-body-lg text-disabled">{t('subtitle')}</p>
          <div className="flex flex-wrap gap-3">
            <Button asChild size="lg">
              <a href="#bang-gia">
                {t('ctaPricing')}
                <ArrowDownIcon />
              </a>
            </Button>
            <Button
              asChild
              size="lg"
              variant="outline"
              className="border-ink-700 bg-transparent text-white hover:bg-ink-800"
            >
              <Link href="/login">{t('ctaLogin')}</Link>
            </Button>
          </div>
        </div>

        <div className="grid gap-4 lg:col-span-2">
          <div className="grid grid-cols-3 gap-3">
            {stats.map(stat => (
              <div
                key={stat.label}
                className="grid gap-1 rounded-2xl border border-ink-800 bg-ink-900/60 p-4 text-center"
              >
                <span className="text-h1 font-semibold text-white">{stat.value}</span>
                <span className="text-caption text-tertiary">{stat.label}</span>
              </div>
            ))}
          </div>
          {hours ? (
            <div className="flex items-center gap-3 rounded-2xl border border-ink-800 bg-ink-900/60 p-4">
              <span className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-brand-500/15 text-brand-400">
                <ClockIcon className="size-5" />
              </span>
              <span className="text-body text-disabled">{t('statHours', hours)}</span>
            </div>
          ) : null}
        </div>
      </div>
    </section>
  );
}
