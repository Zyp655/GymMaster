import { getTranslations } from 'next-intl/server';
import {
  BuildingStorefrontIcon,
  ClockIcon,
  ExclamationTriangleIcon,
  MapPinIcon,
  PhoneIcon,
} from '@heroicons/react/24/outline';
import { EmptyState } from '@/components/common/EmptyState';
import type { LoadResult, PublicBranch } from '../types';
import { SectionHeading } from './SectionHeading';

export async function BranchesSection({ result }: { result: LoadResult<PublicBranch[]> }) {
  const t = await getTranslations('common.landing');
  const branches = result.ok ? result.data : [];

  return (
    <section id="chi-nhanh" className="scroll-mt-16 bg-surface">
      <div className="mx-auto grid max-w-6xl gap-10 px-4 py-20 sm:px-6">
        <SectionHeading eyebrow={t('branches.eyebrow')} title={t('branches.title')} />
        {!result.ok ? (
          <EmptyState icon={ExclamationTriangleIcon} title={t('loadError')} />
        ) : branches.length === 0 ? (
          <EmptyState
            icon={BuildingStorefrontIcon}
            title={t('branches.emptyTitle')}
            description={t('branches.emptyDescription')}
          />
        ) : (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {branches.map(branch => (
              <article
                key={branch.id}
                className="grid content-start gap-4 rounded-2xl border border-line bg-surface p-5"
              >
                <div className="flex items-center gap-3">
                  <span className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-brand-50 text-brand-600">
                    <BuildingStorefrontIcon className="size-5" />
                  </span>
                  <h3 className="min-w-0 truncate text-h4 font-semibold text-primary">
                    {branch.name}
                  </h3>
                </div>
                <ul className="grid gap-2 text-body text-secondary">
                  <li className="flex items-start gap-2">
                    <MapPinIcon className="mt-0.5 size-4 shrink-0 text-tertiary" />
                    <span>{branch.address || t('branches.noAddress')}</span>
                  </li>
                  <li className="flex items-center gap-2">
                    <ClockIcon className="size-4 shrink-0 text-tertiary" />
                    <span>
                      {t('branches.hours', { open: branch.open_time, close: branch.close_time })}
                    </span>
                  </li>
                  {branch.phone ? (
                    <li className="flex items-center gap-2">
                      <PhoneIcon className="size-4 shrink-0 text-tertiary" />
                      <a
                        href={`tel:${branch.phone.replace(/\s/g, '')}`}
                        className="font-medium text-brand-600 hover:underline"
                      >
                        {t('branches.call', { phone: branch.phone })}
                      </a>
                    </li>
                  ) : null}
                </ul>
              </article>
            ))}
          </div>
        )}
      </div>
    </section>
  );
}
