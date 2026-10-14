import type { Metadata } from 'next';
import Link from 'next/link';
import { getTranslations } from 'next-intl/server';
import {
  ArrowRightIcon,
  BanknotesIcon,
  BuildingStorefrontIcon,
  ClockIcon,
  QrCodeIcon,
  ShieldCheckIcon,
  UserGroupIcon,
} from '@heroicons/react/24/outline';
import { PageHeader } from '@/components/common/PageHeader';
import { Card } from '@/components/ui/card';
import { getCachedSession } from '@/guards/getCachedSession';
import { hasPermission, Perm } from '@/lib/permissions';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('dashboard');
  return { title: t('title') };
}

export default async function DashboardPage() {
  const [t, session] = await Promise.all([getTranslations('dashboard'), getCachedSession()]);
  const user = session?.user ?? null;

  const stats = [
    { key: 'activeMembers', icon: UserGroupIcon, tone: 'bg-brand-50 text-brand-600' },
    { key: 'todayCheckins', icon: QrCodeIcon, tone: 'bg-info-50 text-info' },
    { key: 'monthRevenue', icon: BanknotesIcon, tone: 'bg-success-50 text-success' },
    { key: 'expiringSoon', icon: ClockIcon, tone: 'bg-warning-50 text-warning' },
  ] as const;

  const quickLinks = [
    {
      href: '/admin/branches',
      icon: BuildingStorefrontIcon,
      title: t('quickLinks.branches'),
      description: t('quickLinks.branchesDescription'),
      visible: hasPermission(user, Perm.BRANCH_VIEW),
    },
    {
      href: '/admin/roles',
      icon: ShieldCheckIcon,
      title: t('quickLinks.roles'),
      description: t('quickLinks.rolesDescription'),
      visible: hasPermission(user, Perm.ROLE_VIEW),
    },
  ].filter(link => link.visible);

  return (
    <div className="grid gap-6">
      <PageHeader
        title={t('welcome', { name: user?.full_name ?? '' })}
        description={t('subtitle')}
      />

      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {stats.map(stat => {
          const Icon = stat.icon;
          return (
            <Card key={stat.key} className="gap-3">
              <div className="flex items-center justify-between gap-3">
                <span className="text-body text-secondary">{t(`stats.${stat.key}`)}</span>
                <span
                  className={`flex size-10 items-center justify-center rounded-xl ${stat.tone}`}
                >
                  <Icon className="size-5" />
                </span>
              </div>
              <span className="text-h1 font-semibold text-disabled">—</span>
            </Card>
          );
        })}
      </div>

      <p className="rounded-xl bg-subtle p-4 text-body text-secondary">{t('placeholderNote')}</p>

      {quickLinks.length > 0 ? (
        <section className="grid gap-3">
          <h2 className="text-h4 font-semibold text-primary">{t('quickLinks.title')}</h2>
          <div className="grid gap-4 md:grid-cols-2">
            {quickLinks.map(link => {
              const Icon = link.icon;
              return (
                <Link
                  key={link.href}
                  href={link.href}
                  className="group flex items-center gap-4 rounded-2xl border border-line bg-surface p-5 transition-colors hover:border-brand-300"
                >
                  <span className="flex size-12 shrink-0 items-center justify-center rounded-xl bg-brand-50 text-brand-600">
                    <Icon className="size-6" />
                  </span>
                  <span className="grid min-w-0 flex-1 gap-0.5">
                    <span className="text-body-lg font-semibold text-primary">{link.title}</span>
                    <span className="text-body text-secondary">{link.description}</span>
                  </span>
                  <ArrowRightIcon className="size-5 text-tertiary transition-transform group-hover:translate-x-1 group-hover:text-brand-600" />
                </Link>
              );
            })}
          </div>
        </section>
      ) : null}
    </div>
  );
}
