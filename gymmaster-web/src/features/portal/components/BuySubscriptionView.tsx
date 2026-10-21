'use client';

import { useState } from 'react';
import Link from 'next/link';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import {
  ArrowLeftIcon,
  CheckCircleIcon,
  CreditCardIcon,
  TagIcon,
  XMarkIcon,
} from '@heroicons/react/24/outline';
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
import { formatCurrency } from '@/lib/format';
import { cn } from '@/lib/utils';
import {
  useCheckPromotionMutation,
  useCheckoutSubscriptionMutation,
  useProfileQuery,
  usePublicBranchesQuery,
  usePublicPlansQuery,
} from '../queries';
import type { CheckPromotionRes } from '../types';
import { PortalPageHeader } from './PortalPageHeader';

export function BuySubscriptionView() {
  const t = useTranslations('portal.buySubscription');
  const ts = useTranslations('portal.status');
  const toastError = useToastApiError();

  const plansQuery = usePublicPlansQuery();
  const branchesQuery = usePublicBranchesQuery();
  const profileQuery = useProfileQuery();

  const checkPromoMutation = useCheckPromotionMutation();
  const checkoutMutation = useCheckoutSubscriptionMutation();

  const [selectedPlanId, setSelectedPlanId] = useState<number | null>(null);
  const [selectedBranchId, setSelectedBranchId] = useState<number | null>(null);
  const [voucherInput, setVoucherInput] = useState('');
  const [appliedPromo, setAppliedPromo] = useState<CheckPromotionRes | null>(null);

  const plans = plansQuery.data?.membership_plans ?? [];
  const branches = branchesQuery.data ?? [];

  const effectivePlanId = selectedPlanId ?? plans[0]?.id ?? null;
  const effectiveBranchId =
    selectedBranchId ?? profileQuery.data?.branch?.id ?? branches[0]?.id ?? null;

  const selectedPlan = plans.find(p => p.id === effectivePlanId) ?? null;

  const handleSelectPlan = (planId: number) => {
    setSelectedPlanId(planId);
    setAppliedPromo(null);
  };

  const handleApplyVoucher = async () => {
    const code = voucherInput.trim().toUpperCase();
    if (!code || !selectedPlan) return;
    try {
      const res = await checkPromoMutation.mutateAsync({
        code,
        plan_id: selectedPlan.id,
        amount: selectedPlan.price,
      });
      setAppliedPromo(res);
      toast.success(t('voucherApplied', { code: res.code, name: res.name }));
    } catch (error) {
      setAppliedPromo(null);
      toastError(error);
    }
  };

  const handleClearVoucher = () => {
    setAppliedPromo(null);
    setVoucherInput('');
  };

  const handleCheckout = async () => {
    if (!selectedPlan) return;
    try {
      const res = await checkoutMutation.mutateAsync({
        plan_id: selectedPlan.id,
        branch_id: effectiveBranchId ?? undefined,
        promotion_code: appliedPromo?.code ?? undefined,
      });
      if (res.payment_url) {
        window.location.assign(res.payment_url);
      }
    } catch (error) {
      toastError(error);
    }
  };

  const subtotal = selectedPlan?.price ?? 0;
  const discount = appliedPromo?.discount ?? 0;
  const finalAmount = appliedPromo ? appliedPromo.final_amount : subtotal;

  return (
    <div className="grid gap-6">
      <PortalPageHeader
        title={t('title')}
        subtitle={t('subtitle')}
        actions={
          <Button variant="outline" size="sm" asChild>
            <Link href="/me/subscriptions">
              <ArrowLeftIcon />
              {t('backToSubscriptions')}
            </Link>
          </Button>
        }
      />

      {plansQuery.isLoading || branchesQuery.isLoading ? (
        <div className="grid gap-6 lg:grid-cols-3">
          <div className="grid gap-4 lg:col-span-2 sm:grid-cols-2">
            <Skeleton className="h-56 rounded-2xl" />
            <Skeleton className="h-56 rounded-2xl" />
            <Skeleton className="h-56 rounded-2xl" />
            <Skeleton className="h-56 rounded-2xl" />
          </div>
          <Skeleton className="h-80 rounded-2xl" />
        </div>
      ) : plansQuery.isError ? (
        <Card>
          <LoadError error={plansQuery.error} onRetry={() => void plansQuery.refetch()} />
        </Card>
      ) : plans.length === 0 ? (
        <Card>
          <EmptyState
            icon={CreditCardIcon}
            title={t('emptyPlansTitle')}
            description={t('emptyPlansDesc')}
          />
        </Card>
      ) : (
        <div className="grid items-start gap-6 lg:grid-cols-3">
          <div className="grid gap-6 lg:col-span-2">
            <div className="grid gap-3">
              <h2 className="text-h3 font-semibold text-primary">{t('stepPlan')}</h2>
              <div className="grid gap-4 sm:grid-cols-2">
                {plans.map(plan => {
                  const isSelected = plan.id === effectivePlanId;
                  return (
                    <button
                      key={plan.id}
                      type="button"
                      onClick={() => handleSelectPlan(plan.id)}
                      className={cn(
                        'flex cursor-pointer flex-col justify-between gap-4 rounded-2xl border p-5 text-left transition-all outline-none focus-visible:ring-2 focus-visible:ring-ring',
                        isSelected
                          ? 'border-brand-500 bg-brand-25 ring-1 ring-brand-500'
                          : 'border-line bg-surface hover:border-brand-300',
                      )}
                    >
                      <div className="grid gap-2">
                        <div className="flex flex-wrap items-center justify-between gap-2">
                          <Badge variant={plan.type === 'TIME' ? 'info' : 'neutral'}>
                            {ts(`planType.${plan.type}`)}
                          </Badge>
                          {isSelected ? (
                            <CheckCircleIcon className="size-5 text-brand-600" />
                          ) : null}
                        </div>
                        <h3 className="text-h3 font-semibold text-primary">{plan.name}</h3>
                        {plan.description ? (
                          <p className="line-clamp-2 text-body text-secondary">
                            {plan.description}
                          </p>
                        ) : null}
                      </div>

                      <div className="grid gap-3 border-t border-line-subtle pt-3">
                        <p className="text-h2 font-semibold text-brand-600">
                          {formatCurrency(plan.price)}
                        </p>
                        <div className="flex flex-wrap gap-1.5">
                          <Badge variant="outline">
                            {t('durationDays', { days: plan.duration_days })}
                          </Badge>
                          <Badge variant="outline">
                            {plan.type === 'SESSION' && plan.session_count
                              ? t('sessionsCount', { count: plan.session_count })
                              : t('unlimited')}
                          </Badge>
                          <Badge variant={plan.includes_classes ? 'brand' : 'outline'}>
                            {plan.includes_classes ? t('includesClasses') : t('noClasses')}
                          </Badge>
                          <Badge variant="outline">
                            {plan.access_scope === 'ALL_BRANCHES'
                              ? t('allBranches')
                              : t('singleBranch')}
                          </Badge>
                          {plan.max_freeze_days > 0 ? (
                            <Badge variant="outline">
                              {t('freezeDays', { days: plan.max_freeze_days })}
                            </Badge>
                          ) : null}
                        </div>
                      </div>
                    </button>
                  );
                })}
              </div>
            </div>

            <Card className="grid gap-5 p-5">
              <div className="grid gap-2">
                <Label className="text-h4 font-semibold text-primary">{t('stepBranch')}</Label>
                <Select
                  value={effectiveBranchId ? String(effectiveBranchId) : ''}
                  onValueChange={v => setSelectedBranchId(Number(v))}
                >
                  <SelectTrigger>
                    <SelectValue placeholder={t('branchPlaceholder')} />
                  </SelectTrigger>
                  <SelectContent>
                    {branches.map(branch => (
                      <SelectItem key={branch.id} value={String(branch.id)}>
                        {branch.name} — {branch.address}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="grid gap-2">
                <Label className="text-h4 font-semibold text-primary">{t('stepVoucher')}</Label>
                <div className="flex flex-wrap items-center gap-2">
                  <div className="relative min-w-52 flex-1">
                    <Input
                      value={voucherInput}
                      onChange={e => setVoucherInput(e.target.value.toUpperCase())}
                      placeholder={t('voucherPlaceholder')}
                      disabled={Boolean(appliedPromo)}
                    />
                  </div>
                  {appliedPromo ? (
                    <Button type="button" variant="outline" onClick={handleClearVoucher}>
                      <XMarkIcon />
                      {t('clearVoucher')}
                    </Button>
                  ) : (
                    <Button
                      type="button"
                      variant="outline"
                      disabled={
                        !voucherInput.trim() || !selectedPlan || checkPromoMutation.isPending
                      }
                      onClick={() => void handleApplyVoucher()}
                    >
                      <TagIcon />
                      {checkPromoMutation.isPending ? t('applyingVoucher') : t('applyVoucher')}
                    </Button>
                  )}
                </div>
                {appliedPromo ? (
                  <p className="text-caption font-medium text-success">
                    {t('voucherApplied', { code: appliedPromo.code, name: appliedPromo.name })}
                  </p>
                ) : null}
              </div>
            </Card>
          </div>

          <Card className="sticky top-24 grid gap-5 p-6">
            <h2 className="text-h3 font-semibold text-primary">{t('stepSummary')}</h2>

            {selectedPlan ? (
              <div className="grid gap-4">
                <div className="rounded-xl bg-canvas p-4">
                  <p className="text-caption text-secondary">{t('selectedPlan')}</p>
                  <p className="text-h4 font-semibold text-primary">{selectedPlan.name}</p>
                  <p className="text-caption text-secondary">
                    {t('durationDays', { days: selectedPlan.duration_days })} ·{' '}
                    {selectedPlan.access_scope === 'ALL_BRANCHES'
                      ? t('allBranches')
                      : t('singleBranch')}
                  </p>
                </div>

                <div className="grid gap-2.5 border-t border-line pt-4 text-body">
                  <div className="flex items-center justify-between">
                    <span className="text-secondary">{t('subtotal')}</span>
                    <span className="font-medium text-primary">{formatCurrency(subtotal)}</span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-secondary">{t('discount')}</span>
                    <span className={discount > 0 ? 'font-medium text-success' : 'text-secondary'}>
                      -{formatCurrency(discount)}
                    </span>
                  </div>
                  <div className="flex items-center justify-between border-t border-line pt-3">
                    <span className="font-semibold text-primary">{t('finalAmount')}</span>
                    <span className="text-h3 font-semibold text-brand-600">
                      {formatCurrency(finalAmount)}
                    </span>
                  </div>
                </div>

                <Button
                  size="lg"
                  className="w-full"
                  disabled={checkoutMutation.isPending}
                  onClick={() => void handleCheckout()}
                >
                  <CreditCardIcon />
                  {checkoutMutation.isPending ? t('processing') : t('payWithVnpay')}
                </Button>
              </div>
            ) : (
              <p className="text-body text-secondary">{t('noPlanSelected')}</p>
            )}
          </Card>
        </div>
      )}
    </div>
  );
}
