'use client';

import { useState } from 'react';
import dayjs from 'dayjs';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import {
  CalendarDaysIcon,
  ClockIcon,
  InformationCircleIcon,
  MapPinIcon,
  UserIcon,
} from '@heroicons/react/24/outline';
import { ConfirmDialog } from '@/components/common/ConfirmDialog';
import { EmptyState } from '@/components/common/EmptyState';
import { LoadError } from '@/components/common/LoadError';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import { useToastApiError } from '@/hooks/use-api-error';
import { formatDate, formatDateTime } from '@/lib/format';
import { cn } from '@/lib/utils';
import {
  useBookClassMutation,
  useBookingsQuery,
  useBranchesQuery,
  useCancelBookingMutation,
  useClassesQuery,
  useMemberReviewsQuery,
} from '../queries';
import { PortalPageHeader } from './PortalPageHeader';
import { BookingStatusBadge } from './PortalStatusBadge';
import { ReviewActionOrBadge, ReviewDialog, type ReviewTarget } from './ReviewDialog';
import { ScheduleTabs } from './ScheduleTabs';

interface CancelTarget {
  bookingId: number;
  className: string;
  startAt: string;
}

export function ClassesView({ defaultTab = 'schedule' }: { defaultTab?: 'schedule' | 'bookings' } = {}) {
  const t = useTranslations('portal.classes');
  const toastError = useToastApiError();

  const [activeTab, setActiveTab] = useState<'schedule' | 'bookings'>(defaultTab);
  const [branchId, setBranchId] = useState<number | undefined>(undefined);
  const [fromDate, setFromDate] = useState(() => dayjs().format('YYYY-MM-DD'));
  const [toDate, setToDate] = useState(() => dayjs().add(7, 'day').format('YYYY-MM-DD'));
  const [bookingScheduleId, setBookingScheduleId] = useState<number | null>(null);
  const [cancelTarget, setCancelTarget] = useState<CancelTarget | null>(null);
  const [reviewTarget, setReviewTarget] = useState<ReviewTarget | null>(null);

  const branchesQuery = useBranchesQuery();
  const effectiveBranchId = branchId ?? branchesQuery.data?.[0]?.id;

  const classesQuery = useClassesQuery(
    { branch_id: effectiveBranchId, from: fromDate, to: toDate },
    !branchesQuery.isLoading,
  );
  const bookingsQuery = useBookingsQuery();
  const reviewsQuery = useMemberReviewsQuery();

  const bookMutation = useBookClassMutation();
  const cancelMutation = useCancelBookingMutation();

  const handleBook = async (scheduleId: number) => {
    setBookingScheduleId(scheduleId);
    try {
      const res = await bookMutation.mutateAsync(scheduleId);
      if (res.status === 'WAITLIST') {
        toast.info(t('waitlistSuccess'));
      } else {
        toast.success(t('bookSuccess'));
      }
    } catch (error) {
      toastError(error);
    } finally {
      setBookingScheduleId(null);
    }
  };

  const handleConfirmCancel = async () => {
    if (!cancelTarget) return;
    try {
      await cancelMutation.mutateAsync(cancelTarget.bookingId);
      toast.success(t('cancelSuccess'));
      setCancelTarget(null);
    } catch (error) {
      toastError(error);
    }
  };

  const activeBookingsCount =
    bookingsQuery.data?.filter(b => b.status === 'BOOKED' || b.status === 'WAITLIST').length ?? 0;

  return (
    <div className="grid gap-6">
      <ScheduleTabs />
      <PortalPageHeader title={t('title')} subtitle={t('subtitle')} />

      <div className="flex flex-wrap items-center gap-2 border-b border-line pb-3">
        <Button
          variant={activeTab === 'schedule' ? 'primary' : 'ghost'}
          size="sm"
          onClick={() => setActiveTab('schedule')}
        >
          {t('tabSchedule')}
        </Button>
        <Button
          variant={activeTab === 'bookings' ? 'primary' : 'ghost'}
          size="sm"
          onClick={() => setActiveTab('bookings')}
        >
          {t('tabMyBookings', { count: activeBookingsCount })}
        </Button>
      </div>

      {activeTab === 'schedule' ? (
        <div className="grid gap-6">
          <Card className="p-4">
            <div className="grid gap-4 sm:grid-cols-3">
              <div className="grid gap-1.5">
                <Label>{t('branchLabel')}</Label>
                {branchesQuery.isLoading ? (
                  <Skeleton className="h-10 w-full rounded-xl" />
                ) : (
                  <Select
                    value={effectiveBranchId ? String(effectiveBranchId) : undefined}
                    onValueChange={v => setBranchId(Number(v))}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {(branchesQuery.data ?? []).map(branch => (
                        <SelectItem key={branch.id} value={String(branch.id)}>
                          {branch.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                )}
              </div>
              <div className="grid gap-1.5">
                <Label>{t('fromLabel')}</Label>
                <Input
                  type="date"
                  value={fromDate}
                  onChange={e => setFromDate(e.target.value)}
                />
              </div>
              <div className="grid gap-1.5">
                <Label>{t('toLabel')}</Label>
                <Input
                  type="date"
                  value={toDate}
                  min={fromDate}
                  onChange={e => setToDate(e.target.value)}
                />
              </div>
            </div>

            {classesQuery.data ? (
              <div className="flex items-center gap-2 rounded-xl bg-canvas px-3 py-2 text-caption text-secondary">
                <InformationCircleIcon className="size-4 shrink-0 text-brand-600" />
                <span>
                  {t('policyBanner', {
                    advanceDays: classesQuery.data.advance_days,
                    cancelMinutes: classesQuery.data.cancel_before_minutes,
                  })}
                </span>
              </div>
            ) : null}
          </Card>

          {classesQuery.isLoading || branchesQuery.isLoading ? (
            <div className="grid gap-4 md:grid-cols-2">
              <Skeleton className="h-44 rounded-2xl" />
              <Skeleton className="h-44 rounded-2xl" />
              <Skeleton className="h-44 rounded-2xl" />
              <Skeleton className="h-44 rounded-2xl" />
            </div>
          ) : classesQuery.isError || !classesQuery.data ? (
            <Card>
              <LoadError error={classesQuery.error} onRetry={() => void classesQuery.refetch()} />
            </Card>
          ) : classesQuery.data.items.length === 0 ? (
            <Card>
              <EmptyState
                icon={CalendarDaysIcon}
                title={t('emptyScheduleTitle')}
                description={t('emptyScheduleDesc')}
              />
            </Card>
          ) : (
            <div className="grid gap-4 md:grid-cols-2">
              {classesQuery.data.items.map(item => {
                const myBooking = item.my_booking;
                const canCancel =
                  Boolean(myBooking) &&
                  (myBooking?.status === 'BOOKED' || myBooking?.status === 'WAITLIST') &&
                  dayjs(item.cancel_deadline).isAfter(dayjs());
                const isBookingThis =
                  bookMutation.isPending && bookingScheduleId === item.id;

                return (
                  <Card key={item.id} className="justify-between gap-4">
                    <div className="grid gap-2">
                      <div className="flex flex-wrap items-start justify-between gap-2">
                        <div className="grid gap-0.5">
                          <h3 className="text-h4 font-semibold text-primary">
                            {item.class_type_name}
                          </h3>
                          <p className="text-body font-medium text-brand-600">
                            {formatDateTime(item.start_at)} – {formatDate(item.end_at, 'HH:mm')} (
                            {t('duration', { minutes: item.duration_min })})
                          </p>
                        </div>

                        {myBooking ? (
                          <div className="flex items-center gap-1.5">
                            <BookingStatusBadge status={myBooking.status} />
                            {myBooking.status === 'WAITLIST' && myBooking.waitlist_position ? (
                              <Badge variant="outline">
                                {t('waitlistPosition', { pos: myBooking.waitlist_position })}
                              </Badge>
                            ) : null}
                          </div>
                        ) : (
                          <Badge variant={item.available_slots > 0 ? 'success' : 'warning'}>
                            {item.available_slots > 0
                              ? t('slots', {
                                  available: item.available_slots,
                                  capacity: item.capacity,
                                })
                              : t('fullSlots', { capacity: item.capacity })}
                          </Badge>
                        )}
                      </div>

                      <div className="grid gap-1.5 rounded-xl bg-canvas p-3 text-body text-secondary">
                        <div className="flex flex-wrap items-center gap-4">
                          <span className="inline-flex items-center gap-1.5">
                            <UserIcon className="size-4 text-tertiary" />
                            {t('trainer', { name: item.trainer_name })}
                          </span>
                          <span className="inline-flex items-center gap-1.5">
                            <MapPinIcon className="size-4 text-tertiary" />
                            {t('room', { name: item.room_name })} · {item.branch_name}
                          </span>
                        </div>
                        {item.waitlist_count > 0 ? (
                          <p className="text-caption text-tertiary">
                            {t('waitlistCount', { count: item.waitlist_count })}
                          </p>
                        ) : null}
                        {item.note ? <p className="text-caption text-secondary">{item.note}</p> : null}
                      </div>
                    </div>

                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <span className="text-caption text-tertiary">
                        {t('cancelDeadlineLabel', { time: formatDateTime(item.cancel_deadline) })}
                      </span>

                      {myBooking ? (
                        myBooking.status === 'BOOKED' || myBooking.status === 'WAITLIST' ? (
                          <Button
                            variant="outline"
                            size="sm"
                            disabled={!canCancel}
                            onClick={() =>
                              setCancelTarget({
                                bookingId: myBooking.id,
                                className: item.class_type_name,
                                startAt: item.start_at,
                              })
                            }
                          >
                            {canCancel ? t('cancelBtn') : t('cancelDeadlinePassed')}
                          </Button>
                        ) : myBooking.status === 'ATTENDED' ? (
                          <ReviewActionOrBadge
                            reviews={reviewsQuery.data}
                            targetType="CLASS_SCHEDULE"
                            targetId={item.id}
                            onReview={() =>
                              setReviewTarget({
                                targetType: 'CLASS_SCHEDULE',
                                targetId: item.id,
                                targetName: `${item.class_type_name} · ${item.trainer_name}`,
                              })
                            }
                          />
                        ) : null
                      ) : (
                        <Button
                          variant={item.available_slots > 0 ? 'primary' : 'soft'}
                          size="sm"
                          disabled={!item.bookable || isBookingThis}
                          onClick={() => void handleBook(item.id)}
                        >
                          {!item.bookable
                            ? t('notBookable')
                            : isBookingThis
                              ? t('booking')
                              : item.available_slots > 0
                                ? t('bookBtn')
                                : t('joinWaitlistBtn')}
                        </Button>
                      )}
                    </div>
                  </Card>
                );
              })}
            </div>
          )}
        </div>
      ) : (
        <div className="grid gap-4">
          {bookingsQuery.isLoading ? (
            <div className="grid gap-4">
              <Skeleton className="h-28 rounded-2xl" />
              <Skeleton className="h-28 rounded-2xl" />
              <Skeleton className="h-28 rounded-2xl" />
            </div>
          ) : bookingsQuery.isError || !bookingsQuery.data ? (
            <Card>
              <LoadError
                error={bookingsQuery.error}
                onRetry={() => void bookingsQuery.refetch()}
              />
            </Card>
          ) : bookingsQuery.data.length === 0 ? (
            <Card>
              <EmptyState
                icon={ClockIcon}
                title={t('emptyBookingsTitle')}
                description={t('emptyBookingsDesc')}
              />
            </Card>
          ) : (
            <div className="grid gap-4">
              {bookingsQuery.data.map(booking => (
                <Card
                  key={booking.id}
                  className={cn(
                    'flex-col justify-between gap-4 sm:flex-row sm:items-center',
                    booking.status === 'CANCELLED' && 'opacity-75',
                  )}
                >
                  <div className="grid gap-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <h3 className="text-h4 font-semibold text-primary">
                        {booking.class_type_name}
                      </h3>
                      <BookingStatusBadge status={booking.status} />
                      {booking.status === 'WAITLIST' && booking.waitlist_position ? (
                        <Badge variant="outline">
                          {t('waitlistPosition', { pos: booking.waitlist_position })}
                        </Badge>
                      ) : null}
                    </div>
                    <p className="text-body font-medium text-primary">
                      {formatDateTime(booking.start_at)} – {formatDate(booking.end_at, 'HH:mm')}
                    </p>
                    <p className="text-caption text-secondary">
                      {booking.branch_name} · {t('room', { name: booking.room_name })} ·{' '}
                      {t('trainer', { name: booking.trainer_name })}
                    </p>
                    <p className="text-caption text-tertiary">
                      {t('cancelDeadlineLabel', { time: formatDateTime(booking.cancel_deadline) })}
                    </p>
                  </div>

                  {booking.status === 'BOOKED' || booking.status === 'WAITLIST' ? (
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={!booking.can_cancel}
                      onClick={() =>
                        setCancelTarget({
                          bookingId: booking.id,
                          className: booking.class_type_name,
                          startAt: booking.start_at,
                        })
                      }
                    >
                      {booking.can_cancel ? t('cancelBtn') : t('cancelDeadlinePassed')}
                    </Button>
                  ) : booking.status === 'ATTENDED' ? (
                    <ReviewActionOrBadge
                      reviews={reviewsQuery.data}
                      targetType="CLASS_SCHEDULE"
                      targetId={booking.schedule_id}
                      onReview={() =>
                        setReviewTarget({
                          targetType: 'CLASS_SCHEDULE',
                          targetId: booking.schedule_id,
                          targetName: `${booking.class_type_name} · ${booking.trainer_name}`,
                        })
                      }
                    />
                  ) : null}
                </Card>
              ))}
            </div>
          )}
        </div>
      )}

      <ConfirmDialog
        open={Boolean(cancelTarget)}
        onOpenChange={open => !open && setCancelTarget(null)}
        title={t('cancelConfirmTitle')}
        description={
          cancelTarget
            ? t('cancelConfirmDesc', {
                className: cancelTarget.className,
                time: formatDateTime(cancelTarget.startAt),
              })
            : ''
        }
        confirmLabel={t('cancelConfirmBtn')}
        loading={cancelMutation.isPending}
        onConfirm={handleConfirmCancel}
      />

      <ReviewDialog
        target={reviewTarget}
        onOpenChange={open => !open && setReviewTarget(null)}
      />
    </div>
  );
}
