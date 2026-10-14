'use client';

import { useMemo, useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import dayjs from 'dayjs';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import {
  CalendarDaysIcon,
  ClockIcon,
  InformationCircleIcon,
  PlusIcon,
  UserGroupIcon,
} from '@heroicons/react/24/outline';
import { EmptyState } from '@/components/common/EmptyState';
import { LoadError } from '@/components/common/LoadError';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Skeleton } from '@/components/ui/skeleton';
import { Textarea } from '@/components/ui/textarea';
import { useToastApiError } from '@/hooks/use-api-error';
import { formatDate, formatDateTime } from '@/lib/format';
import { cn, looseT } from '@/lib/utils';
import {
  useAvailabilityQuery,
  useBookPTSessionMutation,
  useCancelPTSessionMutation,
  useMemberReviewsQuery,
  usePTContractsQuery,
  usePTSessionsQuery,
} from '../queries';
import { createCancelPTSessionSchema, type CancelPTSessionFormValues } from '../schema';
import type { PTContract, PTSession, SlotReason } from '../types';
import { PortalPageHeader } from './PortalPageHeader';
import { PTContractStatusBadge, PTSessionStatusBadge } from './PortalStatusBadge';
import { ReviewActionOrBadge, ReviewDialog, type ReviewTarget } from './ReviewDialog';
import { ScheduleTabs } from './ScheduleTabs';

export function PTView() {
  const t = useTranslations('portal.pt');
  const contractsQuery = usePTContractsQuery();
  const sessionsQuery = usePTSessionsQuery();
  const reviewsQuery = useMemberReviewsQuery();

  const [bookContract, setBookContract] = useState<PTContract | null>(null);
  const [cancelSession, setCancelSession] = useState<PTSession | null>(null);
  const [reviewTarget, setReviewTarget] = useState<ReviewTarget | null>(null);

  return (
    <div className="grid gap-6">
      <ScheduleTabs />
      <PortalPageHeader title={t('title')} subtitle={t('subtitle')} />

      <section className="grid gap-3">
        <h2 className="text-h3 font-semibold text-primary">{t('contractsTitle')}</h2>

        {contractsQuery.isLoading ? (
          <div className="grid gap-4 md:grid-cols-2">
            <Skeleton className="h-52 rounded-2xl" />
            <Skeleton className="h-52 rounded-2xl" />
          </div>
        ) : contractsQuery.isError || !contractsQuery.data ? (
          <Card>
            <LoadError
              error={contractsQuery.error}
              onRetry={() => void contractsQuery.refetch()}
            />
          </Card>
        ) : contractsQuery.data.length === 0 ? (
          <Card>
            <EmptyState
              icon={UserGroupIcon}
              title={t('emptyContractsTitle')}
              description={t('emptyContractsDesc')}
            />
          </Card>
        ) : (
          <div className="grid gap-4 md:grid-cols-2">
            {contractsQuery.data.map(contract => {
              const canBook =
                contract.status === 'ACTIVE' && contract.bookable_sessions > 0;
              return (
                <Card key={contract.id} className="justify-between gap-4">
                  <div className="grid gap-3">
                    <div className="flex flex-wrap items-start justify-between gap-2">
                      <div>
                        <h3 className="text-h4 font-semibold text-primary">
                          {contract.package_name}
                        </h3>
                        <p className="text-caption text-secondary">
                          {t('sessionDuration', { minutes: contract.session_minutes })}
                        </p>
                      </div>
                      <PTContractStatusBadge status={contract.status} />
                    </div>

                    <div className="grid gap-2 rounded-xl bg-canvas p-3.5 text-body">
                      <div className="flex items-center justify-between gap-2">
                        <span className="text-secondary">{t('trainer')}</span>
                        <span className="font-medium text-primary">{contract.trainer_name}</span>
                      </div>
                      <div className="flex items-center justify-between gap-2">
                        <span className="text-secondary">{t('branch')}</span>
                        <span className="font-medium text-primary">{contract.branch_name}</span>
                      </div>
                      <div className="flex items-center justify-between gap-2">
                        <span className="text-secondary">{t('validity')}</span>
                        <span className="font-medium text-primary">
                          {formatDate(contract.start_date)} – {formatDate(contract.end_date)}
                        </span>
                      </div>
                    </div>

                    <div className="flex flex-wrap items-center gap-2">
                      <Badge variant="neutral">
                        {t('usedSessions', {
                          used: contract.used_sessions,
                          total: contract.total_sessions,
                        })}
                      </Badge>
                      <Badge variant="info">
                        {t('scheduledSessions', { count: contract.scheduled_sessions })}
                      </Badge>
                      <Badge variant={contract.bookable_sessions > 0 ? 'brand' : 'outline'}>
                        {t('bookableSessions', { count: contract.bookable_sessions })}
                      </Badge>
                      {contract.days_left !== null ? (
                        <Badge variant="outline">
                          {t('daysLeft', { days: contract.days_left })}
                        </Badge>
                      ) : null}
                    </div>
                  </div>

                  {canBook ? (
                    <Button size="sm" onClick={() => setBookContract(contract)}>
                      <PlusIcon />
                      {t('bookSessionBtn')}
                    </Button>
                  ) : null}
                </Card>
              );
            })}
          </div>
        )}
      </section>

      <section className="grid gap-3">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h2 className="text-h3 font-semibold text-primary">{t('sessionsTitle')}</h2>
          {sessionsQuery.data ? (
            <span className="inline-flex items-center gap-1.5 text-caption text-secondary">
              <InformationCircleIcon className="size-4 text-brand-600" />
              {t('cancelPolicy', {
                hours: Math.round(sessionsQuery.data.cancel_before_minutes / 60),
              })}
            </span>
          ) : null}
        </div>

        {sessionsQuery.isLoading ? (
          <div className="grid gap-4">
            <Skeleton className="h-32 rounded-2xl" />
            <Skeleton className="h-32 rounded-2xl" />
          </div>
        ) : sessionsQuery.isError || !sessionsQuery.data ? (
          <Card>
            <LoadError error={sessionsQuery.error} onRetry={() => void sessionsQuery.refetch()} />
          </Card>
        ) : sessionsQuery.data.items.length === 0 ? (
          <Card>
            <EmptyState
              icon={CalendarDaysIcon}
              title={t('emptySessionsTitle')}
              description={t('emptySessionsDesc')}
            />
          </Card>
        ) : (
          <div className="grid gap-4">
            {sessionsQuery.data.items.map(session => (
              <Card key={session.id} className="gap-3">
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div className="grid gap-0.5">
                    <div className="flex flex-wrap items-center gap-2">
                      <h3 className="text-h4 font-semibold text-primary">
                        {session.package_name}
                      </h3>
                      <PTSessionStatusBadge status={session.status} />
                    </div>
                    <p className="text-body font-medium text-brand-600">
                      {formatDateTime(session.start_at)} – {formatDate(session.end_at, 'HH:mm')}
                    </p>
                    <p className="text-caption text-secondary">
                      {t('trainer')}: {session.trainer_name} · {session.branch_name}
                    </p>
                  </div>

                  {session.status === 'SCHEDULED' ? (
                    <div className="flex flex-col items-end gap-1">
                      <Button
                        variant="outline"
                        size="sm"
                        disabled={!session.can_cancel}
                        onClick={() => setCancelSession(session)}
                      >
                        {session.can_cancel ? t('cancelSessionBtn') : t('cancelPassed')}
                      </Button>
                      <span className="text-caption text-tertiary">
                        {t('cancelDeadline', { time: formatDateTime(session.cancel_deadline) })}
                      </span>
                    </div>
                  ) : session.status === 'COMPLETED' ? (
                    <ReviewActionOrBadge
                      reviews={reviewsQuery.data}
                      targetType="PT_SESSION"
                      targetId={session.id}
                      onReview={() =>
                        setReviewTarget({
                          targetType: 'PT_SESSION',
                          targetId: session.id,
                          targetName: `${session.package_name} · ${session.trainer_name}`,
                        })
                      }
                    />
                  ) : null}
                </div>

                {session.workout_note || session.trainer_note || session.cancel_reason ? (
                  <div className="grid gap-2 rounded-xl bg-canvas p-3.5 text-body">
                    {session.workout_note ? (
                      <div>
                        <p className="text-caption font-medium text-secondary">
                          {t('workoutNote')}
                        </p>
                        <p className="text-body text-primary">{session.workout_note}</p>
                      </div>
                    ) : null}
                    {session.trainer_note ? (
                      <div>
                        <p className="text-caption font-medium text-secondary">
                          {t('trainerNote')}
                        </p>
                        <p className="text-body text-primary">{session.trainer_note}</p>
                      </div>
                    ) : null}
                    {session.cancel_reason ? (
                      <div>
                        <p className="text-caption font-medium text-secondary">
                          {t('cancelReason')}
                        </p>
                        <p className="text-body text-secondary">{session.cancel_reason}</p>
                      </div>
                    ) : null}
                  </div>
                ) : null}
              </Card>
            ))}
          </div>
        )}
      </section>

      <BookPTSessionDialog
        contract={bookContract}
        onOpenChange={open => !open && setBookContract(null)}
      />

      <CancelPTSessionDialog
        session={cancelSession}
        onOpenChange={open => !open && setCancelSession(null)}
      />

      <ReviewDialog
        target={reviewTarget}
        onOpenChange={open => !open && setReviewTarget(null)}
      />
    </div>
  );
}

function BookPTSessionDialog({
  contract,
  onOpenChange,
}: {
  contract: PTContract | null;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <Dialog open={Boolean(contract)} onOpenChange={onOpenChange}>
      {contract ? (
        <BookPTSessionContent contract={contract} onDone={() => onOpenChange(false)} />
      ) : null}
    </Dialog>
  );
}

function BookPTSessionContent({
  contract,
  onDone,
}: {
  contract: PTContract;
  onDone: () => void;
}) {
  const t = useTranslations('portal.pt.bookDialog');
  const tr = useTranslations('portal.pt.slotReasons');
  const tc = useTranslations('common');
  const toastError = useToastApiError();

  const today = dayjs().format('YYYY-MM-DD');
  const minDate = contract.start_date > today ? contract.start_date : today;
  const [date, setDate] = useState(minDate);
  const [selectedStartAt, setSelectedStartAt] = useState<string>('');

  const availabilityQuery = useAvailabilityQuery(contract.id, date);
  const bookMutation = useBookPTSessionMutation();

  const slots = availabilityQuery.data?.slots ?? [];
  const selectedSlot = slots.find(s => s.start_at === selectedStartAt && s.available) ?? null;

  const handleConfirm = async () => {
    if (!selectedSlot) return;
    try {
      await bookMutation.mutateAsync({
        contract_id: contract.id,
        start_at: selectedSlot.start_at,
      });
      toast.success(t('success'));
      onDone();
    } catch (error) {
      toastError(error);
    }
  };

  const reasonLabel = (reason: SlotReason) => {
    if (!reason) return tr('AVAILABLE');
    return tr(reason);
  };

  return (
    <DialogContent className="max-w-2xl" closeLabel={tc('actions.close')}>
      <DialogHeader>
        <DialogTitle>{t('title')}</DialogTitle>
        <DialogDescription>
          {t('description', {
            packageName: contract.package_name,
            trainerName: contract.trainer_name,
            branchName: contract.branch_name,
          })}
        </DialogDescription>
      </DialogHeader>

      <div className="grid gap-4">
        <div className="grid gap-1.5 sm:max-w-xs">
          <Label>{t('dateLabel')}</Label>
          <Input
            type="date"
            value={date}
            min={minDate}
            max={contract.end_date}
            onChange={e => {
              setDate(e.target.value);
              setSelectedStartAt('');
            }}
          />
        </div>

        {availabilityQuery.data ? (
          <p className="text-caption text-secondary">
            {t('branchHours', {
              open: availabilityQuery.data.open_time,
              close: availabilityQuery.data.close_time,
              minutes: availabilityQuery.data.session_minutes,
            })}
          </p>
        ) : null}

        <div className="grid gap-2">
          <Label>{t('selectSlot')}</Label>
          {availabilityQuery.isLoading ? (
            <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 md:grid-cols-4">
              {Array.from({ length: 8 }).map((_, idx) => (
                <Skeleton key={idx} className="h-14 rounded-xl" />
              ))}
            </div>
          ) : availabilityQuery.isError ? (
            <LoadError
              error={availabilityQuery.error}
              onRetry={() => void availabilityQuery.refetch()}
            />
          ) : slots.length === 0 ? (
            <EmptyState
              icon={ClockIcon}
              title={t('noSlotsTitle')}
              description={t('noSlotsDesc')}
              className="py-6"
            />
          ) : (
            <div className="grid max-h-72 grid-cols-2 gap-2 overflow-y-auto pr-1 sm:grid-cols-3 md:grid-cols-4">
              {slots.map(slot => {
                const isSelected = selectedSlot?.start_at === slot.start_at;
                return (
                  <button
                    key={slot.start_at}
                    type="button"
                    disabled={!slot.available}
                    onClick={() => setSelectedStartAt(slot.start_at)}
                    className={cn(
                      'flex flex-col items-start rounded-xl border p-2.5 text-left transition-colors outline-none focus-visible:ring-2 focus-visible:ring-ring',
                      slot.available
                        ? isSelected
                          ? ' cursor-pointer border-brand-500 bg-brand-500 text-white'
                          : 'cursor-pointer border-line bg-surface text-primary hover:border-brand-300 hover:bg-brand-50'
                        : 'cursor-not-allowed border-line-subtle bg-canvas text-disabled',
                    )}
                  >
                    <span className="text-body font-semibold">
                      {formatDate(slot.start_at, 'HH:mm')} – {formatDate(slot.end_at, 'HH:mm')}
                    </span>
                    <span
                      className={cn(
                        'truncate text-caption',
                        isSelected
                          ? 'text-white'
                          : slot.available
                            ? 'text-success'
                            : 'text-tertiary',
                      )}
                    >
                      {reasonLabel(slot.reason)}
                    </span>
                  </button>
                );
              })}
            </div>
          )}
        </div>

        {selectedSlot ? (
          <div className="rounded-xl bg-brand-50 px-3.5 py-2.5 text-body font-medium text-brand-700">
            {t('selectedSlot', {
              start: formatDateTime(selectedSlot.start_at),
              end: formatDate(selectedSlot.end_at, 'HH:mm'),
            })}
          </div>
        ) : null}
      </div>

      <DialogFooter>
        <Button
          type="button"
          variant="outline"
          onClick={onDone}
          disabled={bookMutation.isPending}
        >
          {tc('actions.cancel')}
        </Button>
        <Button
          type="button"
          disabled={!selectedSlot || bookMutation.isPending}
          onClick={() => void handleConfirm()}
        >
          {bookMutation.isPending ? t('submitting') : t('submit')}
        </Button>
      </DialogFooter>
    </DialogContent>
  );
}

function CancelPTSessionDialog({
  session,
  onOpenChange,
}: {
  session: PTSession | null;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <Dialog open={Boolean(session)} onOpenChange={onOpenChange}>
      {session ? (
        <CancelPTSessionContent session={session} onDone={() => onOpenChange(false)} />
      ) : null}
    </Dialog>
  );
}

function CancelPTSessionContent({
  session,
  onDone,
}: {
  session: PTSession;
  onDone: () => void;
}) {
  const t = useTranslations('portal.pt.cancelDialog');
  const tc = useTranslations('common');
  const tv = useTranslations('common.validation');
  const toastError = useToastApiError();
  const mutation = useCancelPTSessionMutation();

  const schema = useMemo(() => createCancelPTSessionSchema(looseT(tv)), [tv]);
  const form = useForm<CancelPTSessionFormValues>({
    resolver: zodResolver(schema),
    defaultValues: { reason: '' },
  });

  const onSubmit = form.handleSubmit(async values => {
    try {
      await mutation.mutateAsync({ id: session.id, reason: values.reason });
      toast.success(t('success'));
      onDone();
    } catch (error) {
      toastError(error);
    }
  });

  return (
    <DialogContent className="max-w-md" closeLabel={tc('actions.close')}>
      <DialogHeader>
        <DialogTitle>{t('title')}</DialogTitle>
        <DialogDescription>
          {t('description', {
            trainer: session.trainer_name,
            time: formatDateTime(session.start_at),
          })}
        </DialogDescription>
      </DialogHeader>
      <Form {...form}>
        <form onSubmit={onSubmit} className="grid gap-4" noValidate>
          <FormField
            control={form.control}
            name="reason"
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('reasonLabel')}</FormLabel>
                <FormControl>
                  <Textarea {...field} placeholder={t('reasonPlaceholder')} rows={3} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onDone} disabled={mutation.isPending}>
              {tc('actions.cancel')}
            </Button>
            <Button type="submit" variant="danger" disabled={mutation.isPending}>
              {mutation.isPending ? t('submitting') : t('submit')}
            </Button>
          </DialogFooter>
        </form>
      </Form>
    </DialogContent>
  );
}
