'use client';

import { useState } from 'react';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import {
  CheckBadgeIcon,
  ClipboardDocumentCheckIcon,
  ClipboardDocumentIcon,
  ClockIcon,
  GiftIcon,
  ShareIcon,
  SparklesIcon,
  TicketIcon,
} from '@heroicons/react/24/outline';
import { EmptyState } from '@/components/common/EmptyState';
import { LoadError } from '@/components/common/LoadError';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { useToastApiError } from '@/hooks/use-api-error';
import { formatDate, formatDateTime, formatNumber } from '@/lib/format';
import { cn, looseT } from '@/lib/utils';
import {
  useMemberBadgesQuery,
  useMemberLoyaltyQuery,
  useProfileQuery,
  useRedeemLoyaltyRewardMutation,
} from '../queries';
import type { LoyaltyRewardItem } from '../types';
import { PortalPageHeader } from './PortalPageHeader';

export function RewardsView() {
  const t = useTranslations('portal.rewards');
  const tl = looseT(t);
  const toastError = useToastApiError();

  const loyaltyQuery = useMemberLoyaltyQuery();
  const badgesQuery = useMemberBadgesQuery();
  const profileQuery = useProfileQuery();
  const redeemMutation = useRedeemLoyaltyRewardMutation();

  const [redeemingId, setRedeemingId] = useState<number | null>(null);
  const [lastVoucherCode, setLastVoucherCode] = useState<string | null>(null);
  const [copiedKey, setCopiedKey] = useState<string | null>(null);

  if (loyaltyQuery.isLoading) {
    return (
      <div className="grid gap-4">
        <Skeleton className="h-10 w-64" />
        <Skeleton className="h-36 rounded-2xl" />
        <Skeleton className="h-56 rounded-2xl" />
        <Skeleton className="h-48 rounded-2xl" />
      </div>
    );
  }

  if (loyaltyQuery.isError || !loyaltyQuery.data) {
    return (
      <Card>
        <LoadError error={loyaltyQuery.error} onRetry={() => void loyaltyQuery.refetch()} />
      </Card>
    );
  }

  const data = loyaltyQuery.data;
  const balance = data.balance ?? data.points ?? data.loyalty_points ?? 0;
  const rewards = data.rewards ?? data.loyalty_rewards ?? [];
  const ledgers = data.ledgers ?? data.loyalty_ledgers ?? [];
  const effectiveReferralCode =
    data.referral_code ||
    profileQuery.data?.member.referral_code ||
    profileQuery.data?.member.member_code ||
    '';

  const origin = typeof window !== 'undefined' ? window.location.origin : '';
  const referralUrl = effectiveReferralCode
    ? `${origin}/?ref=${encodeURIComponent(effectiveReferralCode)}#tap-thu`
    : '';

  async function copyText(text: string, key: string, successMsg: string) {
    try {
      await navigator.clipboard.writeText(text);
      setCopiedKey(key);
      toast.success(successMsg);
      setTimeout(() => setCopiedKey(prev => (prev === key ? null : prev)), 2000);
    } catch {
      toast.success(successMsg);
    }
  }

  async function handleRedeem(reward: LoyaltyRewardItem) {
    setRedeemingId(reward.id);
    try {
      const res = await redeemMutation.mutateAsync(reward.id);
      setLastVoucherCode(res.voucher_code ?? res.promotion_code ?? res.code ?? null);
      toast.success(res.message || t('redeemSuccessTitle'));
    } catch (error) {
      toastError(error);
    } finally {
      setRedeemingId(null);
    }
  }

  function translateReason(reason: string): string {
    const known = ['CHECKIN', 'RENEW', 'REFERRAL', 'REVIEW', 'REDEEM', 'CHALLENGE', 'ADJUST'];
    const upper = reason.toUpperCase();
    if (known.includes(upper)) {
      return tl(`reasons.${upper}`);
    }
    return reason;
  }

  const badges = badgesQuery.data ?? [];

  return (
    <div className="grid gap-6">
      <PortalPageHeader title={t('title')} subtitle={t('subtitle')} />

      {/* Điểm thưởng & Giới thiệu bạn bè */}
      <div className="grid gap-4 lg:grid-cols-3 lg:gap-6">
        <Card className="flex flex-col justify-between bg-brand-500 text-white">
          <div className="flex items-center justify-between gap-2">
            <span className="text-body font-medium text-brand-50">{t('pointsBalance')}</span>
            <SparklesIcon className="size-6 text-brand-100" />
          </div>
          <div className="my-4">
            <p className="text-display font-bold tracking-tight">
              {t('pointsUnit', { points: formatNumber(balance) })}
            </p>
          </div>
          <p className="text-caption text-brand-100">{t('subtitle')}</p>
        </Card>

        <Card className="lg:col-span-2">
          <CardHeader className="flex-row items-center justify-between gap-2">
            <CardTitle>{t('referralTitle')}</CardTitle>
            <ShareIcon className="size-5 text-brand-600" />
          </CardHeader>
          <div className="grid gap-3">
            <p className="text-body text-secondary">{t('referralDesc')}</p>
            {effectiveReferralCode ? (
              <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl bg-canvas p-4">
                <div className="grid min-w-0 gap-0.5">
                  <span className="text-caption text-secondary">{t('referralCodeLabel')}</span>
                  <span className="font-mono text-h4 font-bold text-brand-600">
                    {effectiveReferralCode}
                  </span>
                  {referralUrl ? (
                    <span className="truncate text-caption text-tertiary">{referralUrl}</span>
                  ) : null}
                </div>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() =>
                    void copyText(
                      referralUrl || effectiveReferralCode,
                      'referral',
                      t('copiedLink'),
                    )
                  }
                >
                  {copiedKey === 'referral' ? (
                    <ClipboardDocumentCheckIcon />
                  ) : (
                    <ClipboardDocumentIcon />
                  )}
                  {t('copyLinkBtn')}
                </Button>
              </div>
            ) : null}
          </div>
        </Card>
      </div>

      {/* Hiển thị mã voucher vừa đổi thành công */}
      {lastVoucherCode ? (
        <Card className="border-success bg-success-50">
          <div className="flex flex-wrap items-center justify-between gap-4">
            <div className="flex items-start gap-3">
              <TicketIcon className="mt-0.5 size-6 shrink-0 text-success" />
              <div className="grid gap-1">
                <p className="text-h4 font-semibold text-primary">{t('redeemSuccessTitle')}</p>
                <p className="text-body text-secondary">{t('redeemVoucherHint')}</p>
                <p className="font-mono text-h3 font-bold text-brand-600">{lastVoucherCode}</p>
              </div>
            </div>
            <Button
              type="button"
              size="sm"
              onClick={() => void copyText(lastVoucherCode, 'voucher', t('copiedVoucher'))}
            >
              {copiedKey === 'voucher' ? (
                <ClipboardDocumentCheckIcon />
              ) : (
                <ClipboardDocumentIcon />
              )}
              {t('copyVoucherBtn')}
            </Button>
          </div>
        </Card>
      ) : null}

      {/* Danh mục Quà & Voucher đổi điểm */}
      <Card>
        <CardHeader>
          <CardTitle>{t('rewardsCatalogTitle')}</CardTitle>
          <p className="text-caption text-secondary">{t('rewardsCatalogSubtitle')}</p>
        </CardHeader>
        {rewards.length === 0 ? (
          <EmptyState
            icon={GiftIcon}
            title={t('emptyRewardsTitle')}
            description={t('emptyRewardsDesc')}
            className="py-6"
          />
        ) : (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {rewards.map(reward => {
              const cost = reward.points_cost ?? reward.cost_points ?? 0;
              const outOfStock =
                reward.stock !== null && reward.stock !== undefined && reward.stock <= 0;
              const notEnoughPoints = balance < cost;
              const isRedeeming = redeemingId === reward.id && redeemMutation.isPending;

              return (
                <div
                  key={reward.id}
                  className="flex flex-col justify-between gap-4 rounded-xl border border-line bg-canvas p-4"
                >
                  <div className="grid gap-2">
                    <div className="flex items-start justify-between gap-2">
                      <p className="text-h4 font-semibold text-primary">{reward.name}</p>
                      <Badge variant="brand">
                        {t('costPoints', { points: formatNumber(cost) })}
                      </Badge>
                    </div>
                    {reward.description ? (
                      <p className="text-caption text-secondary">{reward.description}</p>
                    ) : null}
                    {reward.stock !== null && reward.stock !== undefined ? (
                      <p className="text-caption text-tertiary">
                        {t('stockLeft', { stock: reward.stock })}
                      </p>
                    ) : null}
                  </div>
                  <Button
                    type="button"
                    size="sm"
                    disabled={outOfStock || notEnoughPoints || isRedeeming}
                    onClick={() => void handleRedeem(reward)}
                  >
                    <GiftIcon />
                    {isRedeeming
                      ? t('redeeming')
                      : outOfStock
                        ? t('outOfStock')
                        : notEnoughPoints
                          ? t('insufficientPoints')
                          : t('redeemBtn')}
                  </Button>
                </div>
              );
            })}
          </div>
        )}
      </Card>

      {/* Huy hiệu thành tích */}
      <Card>
        <CardHeader>
          <CardTitle>{t('badgesTitle')}</CardTitle>
          <p className="text-caption text-secondary">{t('badgesSubtitle')}</p>
        </CardHeader>
        {badgesQuery.isLoading ? (
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            <Skeleton className="h-28 rounded-xl" />
            <Skeleton className="h-28 rounded-xl" />
            <Skeleton className="h-28 rounded-xl" />
          </div>
        ) : badges.length === 0 ? (
          <EmptyState
            icon={CheckBadgeIcon}
            title={t('badgesTitle')}
            description={t('badgesSubtitle')}
            className="py-4"
          />
        ) : (
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {badges.map(badge => (
              <div
                key={badge.code}
                className={cn(
                  'flex items-start gap-3 rounded-xl border p-4 transition-colors',
                  badge.earned
                    ? 'border-brand-500/40 bg-brand-50/40'
                    : 'border-line bg-canvas opacity-70',
                )}
              >
                <span
                  className={cn(
                    'flex size-11 shrink-0 items-center justify-center rounded-xl',
                    badge.earned ? 'bg-brand-500 text-white' : 'bg-subtle text-tertiary',
                  )}
                >
                  <CheckBadgeIcon className="size-6" />
                </span>
                <div className="grid min-w-0 flex-1 gap-1">
                  <div className="flex flex-wrap items-center justify-between gap-1.5">
                    <p className="text-body font-semibold text-primary">{badge.name}</p>
                    <Badge variant={badge.earned ? 'success' : 'neutral'}>
                      {badge.earned
                        ? badge.earned_at
                          ? t('earnedBadge', { date: formatDate(badge.earned_at) })
                          : t('earnedBadgeSimple')
                        : t('lockedBadge')}
                    </Badge>
                  </div>
                  <p className="text-caption text-secondary">{badge.description}</p>
                </div>
              </div>
            ))}
          </div>
        )}
      </Card>

      {/* Lịch sử cộng / trừ điểm */}
      <Card>
        <CardHeader>
          <CardTitle>{t('ledgersTitle')}</CardTitle>
        </CardHeader>
        {ledgers.length === 0 ? (
          <EmptyState
            icon={ClockIcon}
            title={t('emptyLedgersTitle')}
            description={t('emptyLedgersDesc')}
            className="py-6"
          />
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('colTime')}</TableHead>
                <TableHead>{t('colReason')}</TableHead>
                <TableHead>{t('colNote')}</TableHead>
                <TableHead className="text-right">{t('colPoints')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {ledgers.map(entry => {
                const positive = entry.points >= 0;
                return (
                  <TableRow key={entry.id}>
                    <TableCell className="whitespace-nowrap text-secondary">
                      {formatDateTime(entry.created_at)}
                    </TableCell>
                    <TableCell className="font-medium text-primary">
                      {translateReason(entry.reason)}
                    </TableCell>
                    <TableCell className="text-secondary">{entry.note || '—'}</TableCell>
                    <TableCell
                      className={cn(
                        'text-right font-semibold',
                        positive ? 'text-success' : 'text-danger',
                      )}
                    >
                      {positive ? `+${formatNumber(entry.points)}` : formatNumber(entry.points)}
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        )}
      </Card>
    </div>
  );
}
