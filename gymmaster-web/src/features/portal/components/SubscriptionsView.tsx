'use client';

import { useMemo, useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import dayjs from 'dayjs';
import Link from 'next/link';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { CreditCardIcon, PauseCircleIcon, PlusIcon } from '@heroicons/react/24/outline';
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
import { Skeleton } from '@/components/ui/skeleton';
import { Textarea } from '@/components/ui/textarea';
import { useToastApiError } from '@/hooks/use-api-error';
import { formatCurrency, formatDate } from '@/lib/format';
import { looseT } from '@/lib/utils';
import { useRequestFreezeMutation, useSubscriptionsQuery } from '../queries';
import { createFreezeSchema, type FreezeFormValues } from '../schema';
import type { Subscription } from '../types';
import { PortalPageHeader } from './PortalPageHeader';
import { FreezeStatusBadge, SubscriptionStatusBadge } from './PortalStatusBadge';

export function SubscriptionsView() {
  const t = useTranslations('portal.subscriptions');
  const ts = useTranslations('portal.status');
  const query = useSubscriptionsQuery();
  const [freezeTarget, setFreezeTarget] = useState<Subscription | null>(null);

  if (query.isLoading) {
    return (
      <div className="grid gap-6">
        <div className="grid gap-2">
          <Skeleton className="h-8 w-56" />
          <Skeleton className="h-5 w-96 max-w-full" />
        </div>
        <Skeleton className="h-64 rounded-2xl" />
        <Skeleton className="h-64 rounded-2xl" />
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

  const subscriptions = query.data;

  return (
    <div className="grid gap-6">
      <PortalPageHeader
        title={t('title')}
        subtitle={t('subtitle')}
        actions={
          <Button asChild size="sm">
            <Link href="/me/subscriptions/buy">
              <PlusIcon />
              {t('buyNew')}
            </Link>
          </Button>
        }
      />

      {subscriptions.length === 0 ? (
        <Card>
          <EmptyState
            icon={CreditCardIcon}
            title={t('emptyTitle')}
            description={t('emptyDesc')}
            action={
              <Button asChild size="sm">
                <Link href="/me/subscriptions/buy">
                  <PlusIcon />
                  {t('buyNew')}
                </Link>
              </Button>
            }
          />
        </Card>
      ) : (
        <div className="grid gap-6">
          {subscriptions.map(sub => {
            const timesLeft = Math.max(0, sub.max_freeze_times - sub.freeze_times_used);
            const daysLeft = Math.max(0, sub.max_freeze_days - sub.freeze_days_used);
            const hasPendingFreeze = sub.freezes.some(f => f.status === 'PENDING');
            const canRequestFreeze =
              sub.status === 'ACTIVE' &&
              sub.max_freeze_times > 0 &&
              timesLeft > 0 &&
              daysLeft > 0 &&
              !hasPendingFreeze;

            return (
              <Card key={sub.id} className="gap-5">
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div className="grid gap-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <h2 className="text-h3 font-semibold text-primary">{sub.plan_name}</h2>
                      <SubscriptionStatusBadge status={sub.status} />
                      <Badge variant={sub.includes_classes ? 'brand' : 'outline'}>
                        {sub.includes_classes ? t('includesClasses') : t('noClasses')}
                      </Badge>
                    </div>
                    <p className="text-caption text-secondary">
                      {sub.plan_code} · {ts(`planType.${sub.plan_type}`)} ·{' '}
                      {ts(`accessScope.${sub.access_scope}`)}
                    </p>
                  </div>

                  {canRequestFreeze ? (
                    <Button variant="outline" size="sm" onClick={() => setFreezeTarget(sub)}>
                      <PauseCircleIcon />
                      {t('requestFreeze')}
                    </Button>
                  ) : null}
                </div>

                <div className="grid gap-4 rounded-xl bg-canvas p-4 sm:grid-cols-2 lg:grid-cols-4">
                  <div>
                    <p className="text-caption text-secondary">{t('branch')}</p>
                    <p className="text-body font-medium text-primary">{sub.branch_name}</p>
                  </div>
                  <div>
                    <p className="text-caption text-secondary">{t('period')}</p>
                    <p className="text-body font-medium text-primary">
                      {formatDate(sub.start_date)} – {formatDate(sub.end_date)}
                    </p>
                  </div>
                  <div>
                    <p className="text-caption text-secondary">
                      {sub.plan_type === 'SESSION' ? t('sessions') : t('daysLeft')}
                    </p>
                    <p className="text-body font-semibold text-primary">
                      {sub.plan_type === 'SESSION'
                        ? t('sessionsValue', {
                            remaining: sub.remaining_sessions ?? 0,
                            total: sub.total_sessions ?? 0,
                          })
                        : sub.days_left !== null
                          ? t('daysLeftValue', { days: sub.days_left })
                          : '—'}
                    </p>
                  </div>
                  <div>
                    <p className="text-caption text-secondary">{t('price')}</p>
                    <p className="text-body font-semibold text-primary">
                      {formatCurrency(sub.price)}
                    </p>
                  </div>
                  <div className="sm:col-span-2 lg:col-span-4">
                    <p className="text-caption text-secondary">{t('freezeQuota')}</p>
                    <p className="text-body font-medium text-primary">
                      {sub.max_freeze_times > 0
                        ? t('freezeQuotaValue', {
                            timesLeft,
                            maxTimes: sub.max_freeze_times,
                            daysLeft,
                            maxDays: sub.max_freeze_days,
                          })
                        : t('noFreezeAllowed')}
                    </p>
                  </div>
                </div>

                <div className="grid gap-2.5">
                  <h3 className="text-body font-semibold text-primary">{t('freezeHistory')}</h3>
                  {sub.freezes.length === 0 ? (
                    <p className="text-body text-secondary">{t('noFreezes')}</p>
                  ) : (
                    <div className="grid gap-2">
                      {sub.freezes.map(freeze => (
                        <div
                          key={freeze.id}
                          className="flex flex-wrap items-start justify-between gap-2 rounded-xl bg-canvas p-3"
                        >
                          <div className="grid gap-0.5">
                            <p className="text-body font-medium text-primary">
                              {t('freezeDays', {
                                days: freeze.days,
                                start: formatDate(freeze.start_date),
                                end: formatDate(freeze.end_date),
                              })}
                            </p>
                            {freeze.reason ? (
                              <p className="text-caption text-secondary">
                                {t('freezeReason', { reason: freeze.reason })}
                              </p>
                            ) : null}
                            {freeze.review_note ? (
                              <p className="text-caption text-secondary">
                                {t('reviewNote', { note: freeze.review_note })}
                              </p>
                            ) : null}
                          </div>
                          <FreezeStatusBadge status={freeze.status} />
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </Card>
            );
          })}
        </div>
      )}

      <RequestFreezeDialog
        subscription={freezeTarget}
        onOpenChange={open => !open && setFreezeTarget(null)}
      />
    </div>
  );
}

function RequestFreezeDialog({
  subscription,
  onOpenChange,
}: {
  subscription: Subscription | null;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <Dialog open={Boolean(subscription)} onOpenChange={onOpenChange}>
      {subscription ? (
        <RequestFreezeContent subscription={subscription} onDone={() => onOpenChange(false)} />
      ) : null}
    </Dialog>
  );
}

function RequestFreezeContent({
  subscription,
  onDone,
}: {
  subscription: Subscription;
  onDone: () => void;
}) {
  const t = useTranslations('portal.subscriptions.dialog');
  const tc = useTranslations('common');
  const tv = useTranslations('common.validation');
  const toastError = useToastApiError();
  const mutation = useRequestFreezeMutation();

  const timesLeft = Math.max(0, subscription.max_freeze_times - subscription.freeze_times_used);
  const daysLeft = Math.max(0, subscription.max_freeze_days - subscription.freeze_days_used);
  const today = dayjs().format('YYYY-MM-DD');

  const schema = useMemo(() => createFreezeSchema(looseT(tv), looseT(t)), [tv, t]);
  const form = useForm<FreezeFormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      start_date: today,
      end_date: dayjs().add(Math.min(daysLeft, 7) - 1, 'day').format('YYYY-MM-DD'),
      reason: '',
    },
  });

  const onSubmit = form.handleSubmit(async values => {
    try {
      await mutation.mutateAsync({
        subscriptionId: subscription.id,
        payload: values,
      });
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
          {t('description', { plan: subscription.plan_name, timesLeft, daysLeft })}
        </DialogDescription>
      </DialogHeader>
      <Form {...form}>
        <form onSubmit={onSubmit} className="grid gap-4" noValidate>
          <div className="grid gap-4 sm:grid-cols-2">
            <FormField
              control={form.control}
              name="start_date"
              render={({ field }) => (
                <FormItem>
                  <FormLabel required>{t('startDate')}</FormLabel>
                  <FormControl>
                    <Input {...field} type="date" min={today} max={subscription.end_date} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="end_date"
              render={({ field }) => (
                <FormItem>
                  <FormLabel required>{t('endDate')}</FormLabel>
                  <FormControl>
                    <Input {...field} type="date" min={today} max={subscription.end_date} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
          <FormField
            control={form.control}
            name="reason"
            render={({ field }) => (
              <FormItem>
                <FormLabel required>{t('reason')}</FormLabel>
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
            <Button type="submit" disabled={mutation.isPending}>
              {mutation.isPending ? tc('actions.saving') : t('submit')}
            </Button>
          </DialogFooter>
        </form>
      </Form>
    </DialogContent>
  );
}
