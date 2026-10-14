'use client';

import Link from 'next/link';
import dayjs from 'dayjs';
import { useTranslations } from 'next-intl';
import {
  CalendarDaysIcon,
  ChartBarIcon,
  ChevronRightIcon,
  ClockIcon,
  CreditCardIcon,
  ExclamationTriangleIcon,
  QrCodeIcon,
  ScaleIcon,
} from '@heroicons/react/24/outline';
import { EmptyState } from '@/components/common/EmptyState';
import { LoadError } from '@/components/common/LoadError';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardHeader, CardTitle } from '@/components/ui/card';
import { Progress } from '@/components/ui/progress';
import { Skeleton } from '@/components/ui/skeleton';
import { formatDate, formatDateTime } from '@/lib/format';
import { useBookingsQuery, useProfileQuery, usePTSessionsQuery } from '../queries';
import type { Subscription } from '../types';
import { SubscriptionStatusBadge } from './PortalStatusBadge';

const UPCOMING_DAYS = 14;

export function OverviewView() {
  const t = useTranslations('portal.overview');
  const tm = useTranslations('portal.progress.metrics');
  const query = useProfileQuery();

  if (query.isLoading) {
    return (
      <div className="grid gap-4">
        <Skeleton className="h-8 w-56" />
        <Skeleton className="h-24 rounded-2xl" />
        <Skeleton className="h-44 rounded-2xl" />
        <Skeleton className="h-40 rounded-2xl" />
      </div>
    );
  }

  if (query.isError || !query.data) {
    return (
      <Card>
        <LoadError error={query.error} onRetry={() => void query.refetch()} />
      </Card>
    );
  }

  const { member, branch, current_subscription: sub, has_valid_subscription } = query.data;
  const metric = member.latest_metric;

  return (
    <div className="grid gap-4 lg:gap-6">
      <div className="grid gap-1">
        <h1 className="text-h2 font-semibold text-primary">
          {t('greeting', { name: member.full_name })}
        </h1>
        <p className="text-body text-secondary">
          {member.member_code}
          {branch ? ` · ${branch.name}` : ''}
        </p>
      </div>

      {member.booking_blocked_until && dayjs(member.booking_blocked_until).isAfter(dayjs()) ? (
        <div className="flex items-start gap-3 rounded-2xl border border-line bg-warning-50 p-4 text-warning">
          <ExclamationTriangleIcon className="mt-0.5 size-5 shrink-0" />
          <p className="text-body font-medium">
            {t('bookingBlocked', { until: formatDateTime(member.booking_blocked_until) })}
          </p>
        </div>
      ) : null}

      <Link
        href="/me/qr"
        className="flex items-center gap-4 rounded-2xl bg-brand-500 p-5 text-white transition-colors hover:bg-brand-600 focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none"
      >
        <span className="flex size-14 shrink-0 items-center justify-center rounded-2xl bg-white text-brand-600">
          <QrCodeIcon className="size-8" />
        </span>
        <span className="grid min-w-0 flex-1 gap-0.5">
          <span className="text-h3 font-semibold">{t('openQr')}</span>
          <span className="text-body text-brand-50">
            {has_valid_subscription ? t('qrCtaHint') : t('invalidAccess')}
          </span>
        </span>
        <ChevronRightIcon className="size-6 shrink-0" />
      </Link>

      <div className="grid gap-4 lg:grid-cols-2 lg:gap-6">
        <Card>
          <CardHeader className="flex-row items-center justify-between gap-2">
            <CardTitle>{t('currentSubscription')}</CardTitle>
            {sub ? (
              <Badge variant={has_valid_subscription ? 'success' : 'warning'}>
                {has_valid_subscription ? t('validAccess') : t('invalidAccess')}
              </Badge>
            ) : null}
          </CardHeader>
          {sub ? (
            <SubscriptionProgress sub={sub} />
          ) : (
            <EmptyState
              icon={CreditCardIcon}
              title={t('noSubscriptionTitle')}
              description={t('noSubscriptionDesc')}
              className="py-4"
            />
          )}
        </Card>

        <UpcomingCard />

        <Card>
          <CardHeader className="flex-row items-center justify-between gap-2">
            <CardTitle>{t('lastCheckin')}</CardTitle>
            <Button asChild variant="ghost" size="sm">
              <Link href="/me/history">
                {t('viewAll')}
                <ChevronRightIcon />
              </Link>
            </Button>
          </CardHeader>
          <div className="flex items-center gap-3 rounded-xl bg-canvas p-4">
            <ClockIcon className="size-6 shrink-0 text-tertiary" />
            <div className="grid gap-0.5">
              <p className="text-body font-medium text-primary">
                {member.stats.last_checkin_at
                  ? formatDateTime(member.stats.last_checkin_at)
                  : t('neverCheckedIn')}
              </p>
              <p className="text-caption text-secondary">
                {t('totalCheckinsValue', { count: member.stats.total_checkins })}
              </p>
            </div>
          </div>
        </Card>

        <Card>
          <CardHeader className="flex-row items-center justify-between gap-2">
            <CardTitle>{t('latestMetrics')}</CardTitle>
            {metric ? (
              <span className="text-caption text-secondary">
                {t('measuredAt', { date: formatDate(metric.measured_at) })}
              </span>
            ) : null}
          </CardHeader>
          {metric ? (
            <div className="grid gap-3">
              <div className="grid grid-cols-3 gap-2 rounded-xl bg-canvas p-4">
                <div className="min-w-0">
                  <p className="text-caption text-secondary">{tm('weight')}</p>
                  <p className="text-h4 font-semibold text-primary">
                    {tm('weightUnit', { value: metric.weight_kg })}
                  </p>
                </div>
                <div className="min-w-0">
                  <p className="text-caption text-secondary">{tm('bmi')}</p>
                  <p className="text-h4 font-semibold text-primary">{metric.bmi ?? '—'}</p>
                </div>
                <div className="min-w-0">
                  <p className="text-caption text-secondary">{tm('bodyFat')}</p>
                  <p className="text-h4 font-semibold text-primary">
                    {metric.body_fat_pct !== null
                      ? tm('bodyFatUnit', { value: metric.body_fat_pct })
                      : '—'}
                  </p>
                </div>
              </div>
              <Button asChild variant="outline" size="sm" className="w-full">
                <Link href="/me/progress">
                  <ChartBarIcon />
                  {t('viewProgress')}
                </Link>
              </Button>
            </div>
          ) : (
            <EmptyState
              icon={ScaleIcon}
              title={t('noMetricsTitle')}
              description={t('noMetricsDesc')}
              className="py-4"
            />
          )}
        </Card>
      </div>
    </div>
  );
}

/** Thẻ gói hiện tại: ngày/buổi còn lại + thanh tiến độ đã dùng. */
export function SubscriptionProgress({ sub }: { sub: Subscription }) {
  const t = useTranslations('portal.overview');
  const ts = useTranslations('portal.status');
  const today = dayjs().startOf('day');
  const start = dayjs(sub.start_date);
  const end = dayjs(sub.end_date);
  const notStarted = start.isAfter(today);

  let percent: number;
  let progressLabel: string;
  if (sub.plan_type === 'SESSION' && sub.total_sessions) {
    const used = sub.total_sessions - (sub.remaining_sessions ?? 0);
    percent = Math.round((used / sub.total_sessions) * 100);
    progressLabel = t('sessionProgress', { used, total: sub.total_sessions });
  } else {
    const total = Math.max(end.diff(start, 'day') + 1, 1);
    const elapsed = Math.min(Math.max(today.diff(start, 'day'), 0), total);
    percent = Math.round((elapsed / total) * 100);
    progressLabel = t('timeProgress', { percent });
  }
  const width = Math.min(Math.max(percent, 0), 100);

  return (
    <div className="grid gap-3 rounded-xl bg-canvas p-4">
      <div className="flex items-center justify-between gap-2">
        <p className="min-w-0 truncate text-h4 font-semibold text-primary">{sub.plan_name}</p>
        <SubscriptionStatusBadge status={sub.status} />
      </div>
      <p className="text-caption text-secondary">
        {ts(`planType.${sub.plan_type}`)} · {ts(`accessScope.${sub.access_scope}`)}
      </p>
      <div className="flex flex-wrap items-end gap-x-6 gap-y-2">
        {sub.plan_type === 'SESSION' && sub.total_sessions !== null ? (
          <div>
            <p className="text-display font-semibold text-primary">
              {sub.remaining_sessions ?? 0}
            </p>
            <p className="text-caption text-secondary">{t('sessionsLeftLabel')}</p>
          </div>
        ) : null}
        {sub.days_left !== null ? (
          <div>
            <p className="text-display font-semibold text-primary">{sub.days_left}</p>
            <p className="text-caption text-secondary">{t('daysLeftLabel')}</p>
          </div>
        ) : null}
      </div>
      <Progress value={width} />
      <div className="flex flex-wrap justify-between gap-2 text-caption text-secondary">
        <span>{progressLabel}</span>
        <span>
          {notStarted
            ? t('startsFrom', { date: formatDate(sub.start_date) })
            : t('validRange', { start: formatDate(sub.start_date), end: formatDate(sub.end_date) })}
        </span>
      </div>
    </div>
  );
}

/** Buổi lớp / PT sắp tới (14 ngày). */
function UpcomingCard() {
  const t = useTranslations('portal.overview');
  const range = {
    from: dayjs().format('YYYY-MM-DD'),
    to: dayjs().add(UPCOMING_DAYS, 'day').format('YYYY-MM-DD'),
  };
  const bookings = useBookingsQuery(range);
  const sessions = usePTSessionsQuery(range);
  const now = dayjs();

  const items = [
    ...(bookings.data ?? [])
      .filter(b => (b.status === 'BOOKED' || b.status === 'WAITLIST') && dayjs(b.start_at).isAfter(now))
      .map(b => ({
        key: `c${b.id}`,
        kind: t('upcomingClass'),
        title: b.class_type_name,
        detail: t('withTrainer', { name: b.trainer_name }),
        startAt: b.start_at,
        href: '/me/bookings',
      })),
    ...(sessions.data?.items ?? [])
      .filter(s => s.status === 'SCHEDULED' && dayjs(s.start_at).isAfter(now))
      .map(s => ({
        key: `p${s.id}`,
        kind: t('upcomingPT'),
        title: s.package_name,
        detail: t('withTrainer', { name: s.trainer_name }),
        startAt: s.start_at,
        href: '/me/pt',
      })),
  ]
    .sort((a, b) => dayjs(a.startAt).valueOf() - dayjs(b.startAt).valueOf())
    .slice(0, 3);

  return (
    <Card>
      <CardHeader className="flex-row items-center justify-between gap-2">
        <CardTitle>{t('upcomingTitle')}</CardTitle>
        <Button asChild variant="ghost" size="sm">
          <Link href="/me/classes">
            {t('viewSchedule')}
            <ChevronRightIcon />
          </Link>
        </Button>
      </CardHeader>
      {bookings.isLoading || sessions.isLoading ? (
        <div className="grid gap-2">
          <Skeleton className="h-16 rounded-xl" />
          <Skeleton className="h-16 rounded-xl" />
        </div>
      ) : bookings.isError || sessions.isError ? (
        <LoadError
          error={bookings.error ?? sessions.error}
          onRetry={() => {
            void bookings.refetch();
            void sessions.refetch();
          }}
          className="py-4"
        />
      ) : items.length === 0 ? (
        <EmptyState
          icon={CalendarDaysIcon}
          title={t('upcomingEmpty')}
          description={t('upcomingEmptyDesc')}
          className="py-4"
        />
      ) : (
        <ul className="grid gap-2">
          {items.map(item => (
            <li key={item.key}>
              <Link
                href={item.href}
                className="flex items-center gap-3 rounded-xl bg-canvas p-3 transition-colors hover:bg-subtle"
              >
                <div className="flex w-14 shrink-0 flex-col items-center rounded-xl bg-surface py-1.5">
                  <span className="text-caption text-secondary">{formatDate(item.startAt, 'DD/MM')}</span>
                  <span className="text-body font-semibold text-primary">
                    {formatDate(item.startAt, 'HH:mm')}
                  </span>
                </div>
                <div className="grid min-w-0 flex-1 gap-0.5">
                  <span className="text-caption font-medium text-brand-600">{item.kind}</span>
                  <span className="truncate text-body font-medium text-primary">{item.title}</span>
                  <span className="truncate text-caption text-secondary">{item.detail}</span>
                </div>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </Card>
  );
}
