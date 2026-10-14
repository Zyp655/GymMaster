import { getTranslations } from 'next-intl/server';
import {
  CalendarDaysIcon,
  ClockIcon,
  ExclamationTriangleIcon,
  UserIcon,
} from '@heroicons/react/24/outline';
import { EmptyState } from '@/components/common/EmptyState';
import { formatCurrency } from '@/lib/format';
import type { LoadResult, PublicPTPackage } from '../types';
import { splitDuration } from '../utils';
import { SectionHeading } from './SectionHeading';

export async function PTSection({ result }: { result: LoadResult<PublicPTPackage[]> }) {
  const t = await getTranslations('common.landing');
  const packages = result.ok ? result.data : [];

  return (
    <section id="huan-luyen-ca-nhan" className="scroll-mt-16">
      <div className="mx-auto grid max-w-6xl gap-10 px-4 py-20 sm:px-6">
        <SectionHeading
          eyebrow={t('pt.eyebrow')}
          title={t('pt.title')}
          subtitle={t('pt.subtitle')}
        />
        {!result.ok ? (
          <EmptyState icon={ExclamationTriangleIcon} title={t('loadError')} />
        ) : packages.length === 0 ? (
          <EmptyState title={t('pt.emptyTitle')} description={t('pt.emptyDescription')} />
        ) : (
          <div className="grid gap-5 md:grid-cols-2 lg:grid-cols-3">
            {packages.map(pkg => {
              const { unit, count } = splitDuration(pkg.validity_days);
              const rows = [
                { icon: UserIcon, text: t('pt.sessions', { count: pkg.session_count }) },
                { icon: ClockIcon, text: t('pt.minutes', { minutes: pkg.session_minutes }) },
                {
                  icon: CalendarDaysIcon,
                  text: t('pt.validity', { duration: t(`duration.${unit}`, { count }) }),
                },
              ];
              return (
                <article
                  key={pkg.id}
                  className="flex flex-col gap-5 rounded-2xl border border-line bg-surface p-6"
                >
                  <div className="grid gap-1">
                    <h3 className="text-h3 font-semibold text-primary">{pkg.name}</h3>
                    {pkg.description ? (
                      <p className="line-clamp-2 text-body text-secondary">{pkg.description}</p>
                    ) : null}
                  </div>
                  <ul className="grid gap-3 rounded-xl bg-canvas p-4">
                    {rows.map(({ icon: Icon, text }) => (
                      <li key={text} className="flex items-center gap-2.5 text-body text-primary">
                        <Icon className="size-5 shrink-0 text-brand-500" />
                        {text}
                      </li>
                    ))}
                  </ul>
                  <div className="mt-auto flex flex-wrap items-end justify-between gap-2">
                    <span className="text-h2 font-semibold text-primary">
                      {formatCurrency(pkg.price)}
                    </span>
                    <span className="text-caption text-tertiary">
                      {t('pt.perSession', {
                        price: formatCurrency(Math.round(pkg.price / pkg.session_count)),
                      })}
                    </span>
                  </div>
                </article>
              );
            })}
          </div>
        )}
      </div>
    </section>
  );
}
