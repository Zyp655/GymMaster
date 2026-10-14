import { getTranslations } from 'next-intl/server';
import {
  CheckIcon,
  ExclamationTriangleIcon,
  SparklesIcon,
  XMarkIcon,
} from '@heroicons/react/20/solid';
import { EmptyState } from '@/components/common/EmptyState';
import { Badge } from '@/components/ui/badge';
import { formatCurrency } from '@/lib/format';
import { cn } from '@/lib/utils';
import type { LoadResult, PublicPlan } from '../types';
import { bestValuePlanId, splitDuration } from '../utils';
import { SectionHeading } from './SectionHeading';

export async function PricingSection({ result }: { result: LoadResult<PublicPlan[]> }) {
  const t = await getTranslations('common.landing');
  const plans = result.ok ? result.data : [];
  const bestId = bestValuePlanId(plans);

  const duration = (days: number) => {
    const { unit, count } = splitDuration(days);
    return t(`duration.${unit}`, { count });
  };

  return (
    <section id="bang-gia" className="scroll-mt-16 bg-surface">
      <div className="mx-auto grid max-w-6xl gap-10 px-4 py-20 sm:px-6">
        <SectionHeading
          eyebrow={t('pricing.eyebrow')}
          title={t('pricing.title')}
          subtitle={t('pricing.subtitle')}
        />
        {!result.ok ? (
          <EmptyState icon={ExclamationTriangleIcon} title={t('loadError')} />
        ) : plans.length === 0 ? (
          <EmptyState title={t('pricing.emptyTitle')} description={t('pricing.emptyDescription')} />
        ) : (
          <div className="grid gap-5 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
            {plans.map(plan => {
              const isBest = plan.id === bestId;
              const span = splitDuration(plan.duration_days);
              const months =
                span.unit === 'years'
                  ? span.count * 12
                  : span.unit === 'months'
                    ? span.count
                    : plan.duration_days / 30;
              const features: { ok: boolean; text: string }[] = [
                plan.type === 'TIME'
                  ? { ok: true, text: t('pricing.unlimited') }
                  : { ok: true, text: t('pricing.sessions', { count: plan.session_count ?? 0 }) },
                {
                  ok: true,
                  text: t('pricing.validFor', { duration: duration(plan.duration_days) }),
                },
                plan.max_freeze_days > 0
                  ? { ok: true, text: t('pricing.freeze', { days: plan.max_freeze_days }) }
                  : { ok: false, text: t('pricing.noFreeze') },
                plan.includes_classes
                  ? { ok: true, text: t('pricing.classes') }
                  : { ok: false, text: t('pricing.noClasses') },
                {
                  ok: true,
                  text:
                    plan.access_scope === 'ALL_BRANCHES'
                      ? t('pricing.allBranches')
                      : t('pricing.singleBranch'),
                },
              ];
              const unitPrice =
                plan.type === 'SESSION' && plan.session_count
                  ? t('pricing.perSession', {
                      price: formatCurrency(Math.round(plan.price / plan.session_count)),
                    })
                  : months >= 2
                    ? t('pricing.perMonth', {
                        price: formatCurrency(Math.round(plan.price / months)),
                      })
                    : null;

              return (
                <article
                  key={plan.id}
                  className={cn(
                    'relative flex flex-col gap-5 rounded-2xl border p-6',
                    isBest
                      ? 'border-brand-500 bg-brand-25 ring-1 ring-brand-500'
                      : 'border-line bg-surface',
                  )}
                >
                  <div className="flex flex-wrap items-center gap-2">
                    <Badge variant={plan.type === 'TIME' ? 'info' : 'neutral'}>
                      {t(`pricing.type${plan.type}`)}
                    </Badge>
                    {isBest ? (
                      <Badge variant="brand">
                        <SparklesIcon />
                        {t('pricing.bestValue')}
                      </Badge>
                    ) : null}
                  </div>
                  <div className="grid gap-1">
                    <h3 className="text-h3 font-semibold text-primary">{plan.name}</h3>
                    {plan.description ? (
                      <p className="line-clamp-2 text-body text-secondary">{plan.description}</p>
                    ) : null}
                  </div>
                  <div className="grid gap-0.5">
                    <span className="text-h1 font-semibold text-primary">
                      {formatCurrency(plan.price)}
                    </span>
                    {unitPrice ? (
                      <span className="text-caption text-tertiary">{unitPrice}</span>
                    ) : null}
                  </div>
                  <ul className="grid gap-2.5 border-t border-line-subtle pt-5">
                    {features.map(f => (
                      <li key={f.text} className="flex items-start gap-2 text-body">
                        {f.ok ? (
                          <CheckIcon className="mt-0.5 size-4 shrink-0 text-success" />
                        ) : (
                          <XMarkIcon className="mt-0.5 size-4 shrink-0 text-disabled" />
                        )}
                        <span className={f.ok ? 'text-primary' : 'text-tertiary'}>{f.text}</span>
                      </li>
                    ))}
                  </ul>
                </article>
              );
            })}
          </div>
        )}
      </div>
    </section>
  );
}
