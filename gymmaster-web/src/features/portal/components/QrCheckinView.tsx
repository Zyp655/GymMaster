'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import Link from 'next/link';
import { QRCodeSVG } from 'qrcode.react';
import { useTranslations } from 'next-intl';
import {
  ArrowPathIcon,
  CheckCircleIcon,
  ClockIcon,
  ExclamationTriangleIcon,
  SunIcon,
} from '@heroicons/react/24/outline';
import { LoadError } from '@/components/common/LoadError';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Progress } from '@/components/ui/progress';
import { Skeleton } from '@/components/ui/skeleton';
import { formatDate, initials } from '@/lib/format';
import { useProfileQuery, useQrQuery } from '../queries';
import type { Profile, QRToken } from '../types';

/** Làm mới mã khi còn ≤ REFRESH_BEFORE giây. */
const REFRESH_BEFORE = 5;
const QR_CACHE_KEY = 'gymmaster:last_qr_token';

function readCachedQr(): QRToken | null {
  if (typeof window === 'undefined') return null;
  try {
    const raw = window.localStorage.getItem(QR_CACHE_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as QRToken;
    return parsed?.token ? parsed : null;
  } catch {
    return null;
  }
}

export function QrCheckinView() {
  const t = useTranslations('portal.qr');
  const qrQuery = useQrQuery();
  const profileQuery = useProfileQuery();
  const { refetch } = qrQuery;
  const refresh = useCallback(() => void refetch(), [refetch]);
  const [cachedQr] = useState<QRToken | null>(() => readCachedQr());

  useEffect(() => {
    if (qrQuery.data) {
      try {
        window.localStorage.setItem(QR_CACHE_KEY, JSON.stringify(qrQuery.data));
      } catch {
        // Ignore storage quota errors
      }
    }
  }, [qrQuery.data]);

  useScreenWakeLock();

  const activeQr = qrQuery.data ?? cachedQr;
  const isUsingCached = !qrQuery.data && Boolean(cachedQr) && qrQuery.isError;

  return (
    <div className="mx-auto grid w-full max-w-md gap-4">
      <div className="grid gap-1 text-center">
        <h1 className="text-h2 font-semibold text-primary">{t('title')}</h1>
        <p className="text-body text-secondary">{t('subtitle')}</p>
      </div>

      {isUsingCached ? (
        <div className="flex items-start gap-3 rounded-2xl border border-line bg-warning-50 p-3.5 text-warning">
          <ExclamationTriangleIcon className="mt-0.5 size-5 shrink-0" />
          <p className="text-caption font-medium">{t('offlineCachedNotice')}</p>
        </div>
      ) : null}

      {qrQuery.isLoading && !activeQr ? (
        <Card className="items-center p-6">
          <Skeleton className="size-64 rounded-2xl" />
          <Skeleton className="h-5 w-48" />
          <Skeleton className="h-2 w-full" />
        </Card>
      ) : !activeQr ? (
        <Card>
          <LoadError error={qrQuery.error} onRetry={refresh} />
        </Card>
      ) : (
        <QrTokenCard data={activeQr} isFetching={qrQuery.isFetching} onRefresh={refresh} />
      )}

      <div className="flex items-start gap-3 rounded-2xl bg-info-50 p-4 text-info">
        <SunIcon className="mt-0.5 size-5 shrink-0" />
        <p className="text-body">{t('brightnessHint')}</p>
      </div>

      {profileQuery.data ? <MemberInfo profile={profileQuery.data} /> : null}
    </div>
  );
}

function QrTokenCard({
  data,
  isFetching,
  onRefresh,
}: {
  data: QRToken;
  isFetching: boolean;
  onRefresh: () => void;
}) {
  const t = useTranslations('portal.qr');
  const { token, expires_at, server_time, ttl, has_valid_subscription } = data;
  // Thời gian sống tính theo giờ máy chủ (bù lệch đồng hồ điện thoại).
  const lifetime = Math.max(
    1,
    Math.round((new Date(expires_at).getTime() - new Date(server_time).getTime()) / 1000) || ttl,
  );
  const [secondsLeft, setSecondsLeft] = useState(lifetime);
  const onRefreshRef = useRef(onRefresh);

  useEffect(() => {
    onRefreshRef.current = onRefresh;
  }, [onRefresh]);

  // Chỉ khởi động lại bộ đếm khi có mã mới.
  useEffect(() => {
    const deadline = Date.now() + lifetime * 1000;
    let requested = false;
    const tick = () => {
      const left = Math.max(0, Math.ceil((deadline - Date.now()) / 1000));
      setSecondsLeft(left);
      if (left <= REFRESH_BEFORE && !requested) {
        requested = true;
        onRefreshRef.current();
      }
    };
    tick();
    const timer = window.setInterval(tick, 1000);
    return () => window.clearInterval(timer);
  }, [token, lifetime]);

  const expired = secondsLeft <= 0;

  return (
    <div className="grid gap-4">
      {!has_valid_subscription ? (
        <div className="flex items-start gap-3 rounded-2xl border border-line bg-warning-50 p-4 text-warning">
          <ExclamationTriangleIcon className="mt-0.5 size-5 shrink-0" />
          <div className="grid gap-0.5">
            <p className="text-body font-semibold">{t('noValidSubTitle')}</p>
            <p className="text-body">{t('noValidSubDesc')}</p>
            <Link href="/me/subscriptions" className="text-body font-medium underline">
              {t('viewSubscriptions')}
            </Link>
          </div>
        </div>
      ) : null}

      <Card className="items-center gap-5 p-6 text-center">
        {has_valid_subscription ? (
          <Badge variant="success">
            <CheckCircleIcon />
            {t('validSubBadge')}
          </Badge>
        ) : null}

        <div className="w-full rounded-2xl bg-surface p-2" aria-live="polite">
          <QRCodeSVG
            value={token}
            size={512}
            level="M"
            marginSize={2}
            className={expired ? 'h-auto w-full opacity-30' : 'h-auto w-full'}
            aria-label={t('qrAlt')}
          />
        </div>

        <div className="grid w-full gap-2">
          <div className="flex items-center justify-center gap-1.5 text-body font-medium text-primary">
            <ClockIcon className="size-4 text-tertiary" />
            {expired ? t('refreshing') : t('refreshIn', { seconds: secondsLeft })}
          </div>
          <Progress value={(secondsLeft / lifetime) * 100} />
          <p className="text-caption text-tertiary">{t('securityNote', { ttl })}</p>
        </div>

        <Button variant="outline" onClick={onRefresh} disabled={isFetching} className="w-full">
          <ArrowPathIcon />
          {isFetching ? t('refreshing') : t('refreshNow')}
        </Button>
      </Card>
    </div>
  );
}

function MemberInfo({ profile }: { profile: Profile }) {
  const t = useTranslations('portal.qr');
  const { member, current_subscription: sub } = profile;
  return (
    <Card className="flex-row items-center gap-3 p-4">
      <Avatar className="size-12">
        {member.avatar_url ? <AvatarImage src={member.avatar_url} alt={member.full_name} /> : null}
        <AvatarFallback>{initials(member.full_name)}</AvatarFallback>
      </Avatar>
      <div className="grid min-w-0 gap-0.5">
        <p className="truncate text-h4 font-semibold text-primary">{member.full_name}</p>
        <p className="text-caption text-secondary">{member.member_code}</p>
        {sub ? (
          <p className="truncate text-caption text-secondary">
            {t('planUntil', { plan: sub.plan_name, date: formatDate(sub.end_date) })}
          </p>
        ) : null}
      </div>
    </Card>
  );
}

/** Giữ màn hình sáng khi đang hiển thị mã (nếu trình duyệt hỗ trợ Wake Lock). */
function useScreenWakeLock() {
  useEffect(() => {
    let lock: WakeLockSentinel | null = null;
    let cancelled = false;
    const request = async () => {
      try {
        if ('wakeLock' in navigator && document.visibilityState === 'visible') {
          const sentinel = await navigator.wakeLock.request('screen');
          if (cancelled) void sentinel.release();
          else lock = sentinel;
        }
      } catch {
        // Không hỗ trợ / bị từ chối: bỏ qua.
      }
    };
    void request();
    const onVisible = () => {
      if (document.visibilityState === 'visible') void request();
    };
    document.addEventListener('visibilitychange', onVisible);
    return () => {
      cancelled = true;
      document.removeEventListener('visibilitychange', onVisible);
      void lock?.release();
    };
  }, []);
}
