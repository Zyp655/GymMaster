'use client';

import { useState } from 'react';
import Link from 'next/link';
import dayjs from 'dayjs';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import {
  CalendarDaysIcon,
  ChartBarIcon,
  ChatBubbleBottomCenterTextIcon,
  ChevronRightIcon,
  ClockIcon,
  CreditCardIcon,
  ExclamationTriangleIcon,
  KeyIcon,
  MegaphoneIcon,
  QrCodeIcon,
  ScaleIcon,
  UserGroupIcon,
} from '@heroicons/react/24/outline';
import { EmptyState } from '@/components/common/EmptyState';
import { LoadError } from '@/components/common/LoadError';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardHeader, CardTitle } from '@/components/ui/card';
import { Progress } from '@/components/ui/progress';
import { Skeleton } from '@/components/ui/skeleton';
import { Textarea } from '@/components/ui/textarea';
import { useToastApiError } from '@/hooks/use-api-error';
import { formatDate, formatDateTime } from '@/lib/format';
import { cn } from '@/lib/utils';
import {
  useBookingsQuery,
  useBranchOccupancyQuery,
  useMemberAnnouncementsQuery,
  useMemberLockerQuery,
  useMemberSurveysQuery,
  useProfileQuery,
  usePTSessionsQuery,
  useSubmitSurveyResponseMutation,
} from '../queries';
import type { Subscription } from '../types';
import {
  LockerTypeBadge,
  OccupancyStatusBadge,
  SubscriptionStatusBadge,
} from './PortalStatusBadge';

const UPCOMING_DAYS = 14;
const NPS_SCORES = [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10] as const;

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
  const activeBranchId = branch?.id ?? member.home_branch_id;

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

        <BranchOccupancyCard branchId={activeBranchId} branchName={branch?.name} />

        <ActiveLockerCard />

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

        <AnnouncementsCard />

        <QuickSurveyCard />
      </div>
    </div>
  );
}

/** Thẻ mật độ phòng tập hiện tại (QUIET / NORMAL / BUSY / FULL). */
function BranchOccupancyCard({
  branchId,
  branchName,
}: {
  branchId: number | null | undefined;
  branchName?: string;
}) {
  const t = useTranslations('portal.overview');
  const query = useBranchOccupancyQuery();
  const list = query.data ?? [];
  const occ = (branchId ? list.find(b => b.branch_id === branchId) : undefined) ?? list[0];

  const present = occ?.present_count ?? 0;
  const capacity = occ ? Math.max(occ.capacity || 100, 1) : 100;
  const percent = occ
    ? Math.min(
        Math.max(
          Math.round(
            occ.percent ?? occ.occupancy_pct ?? occ.percentage ?? (present / capacity) * 100,
          ),
          0,
        ),
        100,
      )
    : 0;
  const status = (occ?.status ?? occ?.level ?? 'NORMAL') as 'QUIET' | 'NORMAL' | 'BUSY' | 'FULL';

  return (
    <Card>
      <CardHeader className="flex-row items-center justify-between gap-2">
        <CardTitle>{t('occupancyTitle')}</CardTitle>
        {occ ? <OccupancyStatusBadge status={status} /> : null}
      </CardHeader>
      {query.isLoading ? (
        <Skeleton className="h-24 rounded-xl" />
      ) : occ ? (
        <div className="grid gap-3 rounded-xl bg-canvas p-4">
          <div className="flex items-center justify-between gap-2">
            <div className="flex items-center gap-2 text-body font-medium text-primary">
              <UserGroupIcon className="size-5 text-brand-600" />
              <span>{occ.branch_name || branchName || '—'}</span>
            </div>
            <span className="text-caption font-semibold text-primary">{percent}%</span>
          </div>
          <Progress value={percent} />
          <p className="text-caption text-secondary">
            {t('occupancyPeople', {
              present,
              capacity,
              percent,
            })}
          </p>
        </div>
      ) : (
        <div className="flex items-center gap-3 rounded-xl bg-canvas p-4 text-secondary">
          <UserGroupIcon className="size-6 shrink-0 text-tertiary" />
          <p className="text-body">{t('occupancyEmpty')}</p>
        </div>
      )}
    </Card>
  );
}

/** Thẻ tủ đồ đang sử dụng (theo lượt hoặc thuê tháng). */
function ActiveLockerCard() {
  const t = useTranslations('portal.overview');
  const query = useMemberLockerQuery();
  const locker = query.data?.[0];
  const lockerType = ((locker?.type ?? locker?.rental_type ?? 'DAILY') as 'DAILY' | 'MONTHLY');
  const lockerCode = locker?.code || locker?.locker_code || '—';
  const lockerZone = locker?.zone || '—';
  const lockerEnd = locker?.end_date || locker?.end_at || locker?.assigned_until;

  return (
    <Card>
      <CardHeader className="flex-row items-center justify-between gap-2">
        <CardTitle>{t('lockerTitle')}</CardTitle>
        {locker ? <LockerTypeBadge type={lockerType} /> : null}
      </CardHeader>
      {query.isLoading ? (
        <Skeleton className="h-24 rounded-xl" />
      ) : locker ? (
        <div className="flex items-center gap-4 rounded-xl bg-canvas p-4">
          <span className="flex size-12 shrink-0 items-center justify-center rounded-xl bg-brand-50 text-brand-600">
            <KeyIcon className="size-6" />
          </span>
          <div className="grid min-w-0 flex-1 gap-0.5">
            <p className="text-h4 font-semibold text-primary">
              {t('lockerCode', { code: lockerCode })}
            </p>
            <p className="text-caption text-secondary">
              {t('lockerZone', { zone: lockerZone })}
              {locker.branch_name ? ` · ${locker.branch_name}` : ''}
            </p>
            {lockerEnd ? (
              <p className="text-caption text-secondary">
                {t('lockerUntil', {
                  time: locker.end_date ? formatDate(lockerEnd) : formatDateTime(lockerEnd),
                })}
              </p>
            ) : null}
          </div>
        </div>
      ) : (
        <EmptyState
          icon={KeyIcon}
          title={t('noLockerTitle')}
          description={t('noLockerDesc')}
          className="py-4"
        />
      )}
    </Card>
  );
}

/** Bảng tin phòng tập (ghim & mới nhất). */
function AnnouncementsCard() {
  const t = useTranslations('portal.overview');
  const query = useMemberAnnouncementsQuery();
  const items = (query.data ?? []).slice(0, 4);

  return (
    <Card>
      <CardHeader className="flex-row items-center justify-between gap-2">
        <CardTitle>{t('announcementsTitle')}</CardTitle>
        <MegaphoneIcon className="size-5 text-brand-600" />
      </CardHeader>
      {query.isLoading ? (
        <div className="grid gap-2">
          <Skeleton className="h-16 rounded-xl" />
          <Skeleton className="h-16 rounded-xl" />
        </div>
      ) : items.length === 0 ? (
        <div className="flex items-center gap-3 rounded-xl bg-canvas p-4 text-secondary">
          <MegaphoneIcon className="size-6 shrink-0 text-tertiary" />
          <p className="text-body">{t('noAnnouncements')}</p>
        </div>
      ) : (
        <ul className="grid gap-2.5">
          {items.map(item => {
            const isPinned = Boolean(item.is_pinned ?? item.pinned);
            const pubAt = item.published_at || item.publish_at;
            return (
              <li key={item.id} className="grid gap-1 rounded-xl bg-canvas p-3.5">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <p className="text-body font-semibold text-primary">{item.title}</p>
                  <div className="flex items-center gap-1.5">
                    {isPinned ? <Badge variant="brand">{t('pinnedBadge')}</Badge> : null}
                    {pubAt ? (
                      <span className="text-caption text-tertiary">{formatDate(pubAt)}</span>
                    ) : null}
                  </div>
                </div>
                <p className="line-clamp-3 text-caption text-secondary">
                  {item.content || item.body}
                </p>
              </li>
            );
          })}
        </ul>
      )}
    </Card>
  );
}

/** Khảo sát nhanh NPS (0-10) + nhận xét. */
function QuickSurveyCard() {
  const t = useTranslations('portal.overview');
  const toastError = useToastApiError();
  const surveysQuery = useMemberSurveysQuery();
  const submitMutation = useSubmitSurveyResponseMutation();

  const activeSurvey = (surveysQuery.data ?? []).find(s => !s.responded) ?? surveysQuery.data?.[0];
  const [selectedScore, setSelectedScore] = useState<number | null>(null);
  const [comment, setComment] = useState('');
  const [submittedLocal, setSubmittedLocal] = useState(false);

  const isCompleted = Boolean(activeSurvey?.responded || submittedLocal);

  async function handleSubmit() {
    if (selectedScore === null) return;
    const surveyId = activeSurvey?.id ?? 1;
    try {
      await submitMutation.mutateAsync({
        id: surveyId,
        payload: {
          score: selectedScore,
          comment: comment.trim() || undefined,
        },
      });
      setSubmittedLocal(true);
      toast.success(t('surveySuccess'));
    } catch (error) {
      toastError(error);
    }
  }

  return (
    <Card>
      <CardHeader className="flex-row items-center justify-between gap-2">
        <CardTitle>{activeSurvey?.title || t('surveyTitle')}</CardTitle>
        <ChatBubbleBottomCenterTextIcon className="size-5 text-brand-600" />
      </CardHeader>
      {isCompleted ? (
        <div className="rounded-xl bg-success-50 p-4 text-body font-medium text-success">
          {t('surveySuccess')}
        </div>
      ) : (
        <div className="grid gap-3 rounded-xl bg-canvas p-4">
          <p className="text-caption text-secondary">
            {activeSurvey?.description || t('surveySubtitle')}
          </p>
          <div className="grid grid-cols-11 gap-1">
            {NPS_SCORES.map(score => {
              const selected = selectedScore === score;
              return (
                <button
                  key={score}
                  type="button"
                  onClick={() => setSelectedScore(score)}
                  className={cn(
                    'flex h-9 items-center justify-center rounded-lg border text-caption font-semibold transition-colors',
                    selected
                      ? 'border-brand-600 bg-brand-500 text-white'
                      : 'border-line bg-surface text-primary hover:border-brand-500/60',
                  )}
                >
                  {score}
                </button>
              );
            })}
          </div>
          <div className="flex justify-between text-caption text-tertiary">
            <span>{t('surveyLowLabel')}</span>
            <span>{t('surveyHighLabel')}</span>
          </div>
          <Textarea
            value={comment}
            onChange={e => setComment(e.target.value)}
            placeholder={t('surveyCommentPlaceholder')}
            rows={2}
          />
          <Button
            type="button"
            size="sm"
            disabled={selectedScore === null || submitMutation.isPending}
            onClick={() => void handleSubmit()}
          >
            {submitMutation.isPending ? t('surveySubmitting') : t('surveySubmit')}
          </Button>
        </div>
      )}
    </Card>
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
