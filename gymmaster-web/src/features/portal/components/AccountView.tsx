'use client';

import { useState } from 'react';
import Link from 'next/link';
import { signOut } from 'next-auth/react';
import { useTranslations } from 'next-intl';
import { QRCodeSVG } from 'qrcode.react';
import { toast } from 'sonner';
import {
  ArrowDownTrayIcon,
  ArrowRightStartOnRectangleIcon,
  BellIcon,
  ChartBarIcon,
  ChevronRightIcon,
  ClockIcon,
  ComputerDesktopIcon,
  DocumentCheckIcon,
  DocumentTextIcon,
  FireIcon,
  GiftIcon,
  KeyIcon,
  LockClosedIcon,
  ShieldCheckIcon,
  TrashIcon,
  TrophyIcon,
} from '@heroicons/react/24/outline';
import { ConfirmDialog } from '@/components/common/ConfirmDialog';
import { LoadError } from '@/components/common/LoadError';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardHeader, CardTitle } from '@/components/ui/card';
import { Checkbox } from '@/components/ui/checkbox';
import { Input } from '@/components/ui/input';
import { Skeleton } from '@/components/ui/skeleton';
import { ChangePasswordDialog } from '@/features/auth/components/ChangePasswordDialog';
import { useToastApiError } from '@/hooks/use-api-error';
import { formatDateTime } from '@/lib/format';
import { setClientAccessToken } from '@/lib/session-token';
import {
  useCreateMemberDataRequestMutation,
  useDisableMemberMFAMutation,
  useEnableMemberMFAMutation,
  useMemberConsentsQuery,
  useMemberDataRequestsQuery,
  useMemberSessionsQuery,
  useProfileQuery,
  useRevokeMemberSessionMutation,
  useRevokeOtherMemberSessionsMutation,
  useSetupMemberMFAMutation,
  useUpdateMemberConsentsMutation,
} from '../queries';
import { downloadMemberDataExport } from '../api';
import type { ConsentType, DataRequestType, MFASetupRes, SessionItem } from '../types';
import { describeUserAgent } from '../user-agent';
import { PortalPageHeader } from './PortalPageHeader';
import { DataRequestStatusBadge } from './PortalStatusBadge';
import { ProfileFormCard } from './ProfileFormCard';

/** Tài khoản: hồ sơ, lối tắt, 2FA TOTP, phiên đăng nhập, quyền riêng tư NĐ 13/2023. */
export function AccountView() {
  const t = useTranslations('portal.account');
  const tn = useTranslations('portal.nav');
  const query = useProfileQuery();
  const [passwordOpen, setPasswordOpen] = useState(false);
  const [loggingOut, setLoggingOut] = useState(false);

  const links = [
    { href: '/me/rewards', label: tn('rewards'), icon: GiftIcon },
    { href: '/me/challenges', label: tn('challenges'), icon: TrophyIcon },
    { href: '/me/nutrition', label: tn('nutrition'), icon: FireIcon },
    { href: '/me/contracts', label: tn('contracts'), icon: DocumentCheckIcon },
    { href: '/me/progress', label: tn('progress'), icon: ChartBarIcon },
    { href: '/me/history', label: tn('history'), icon: ClockIcon },
    { href: '/me/invoices', label: tn('invoices'), icon: DocumentTextIcon },
    { href: '/me/notifications', label: tn('notifications'), icon: BellIcon },
  ];

  const logout = async () => {
    setLoggingOut(true);
    setClientAccessToken(null);
    await signOut({ callbackUrl: '/login' });
  };

  return (
    <div className="grid gap-6">
      <PortalPageHeader title={t('title')} subtitle={t('subtitle')} />

      <Card className="gap-0 p-2 lg:hidden">
        <ul className="grid">
          {links.map(link => {
            const Icon = link.icon;
            return (
              <li key={link.href}>
                <Link
                  href={link.href}
                  className="flex items-center gap-3 rounded-xl px-3 py-3 text-body font-medium text-primary transition-colors hover:bg-subtle"
                >
                  <Icon className="size-5 shrink-0 text-tertiary" />
                  <span className="flex-1">{link.label}</span>
                  <ChevronRightIcon className="size-4 text-tertiary" />
                </Link>
              </li>
            );
          })}
        </ul>
      </Card>

      {query.isLoading ? (
        <Skeleton className="h-96 rounded-2xl" />
      ) : query.isError || !query.data ? (
        <Card>
          <LoadError error={query.error} onRetry={() => void query.refetch()} />
        </Card>
      ) : (
        <ProfileFormCard profile={query.data} />
      )}

      <MfaSecurityCard
        key={String(Boolean(query.data?.member.mfa_enabled))}
        initialEnabled={Boolean(query.data?.member.mfa_enabled)}
      />

      <ActiveSessionsCard />

      <PrivacyDataCard />

      <Card className="gap-3">
        <h2 className="text-h4 font-semibold text-primary">{t('securityTitle')}</h2>
        <div className="grid gap-2 sm:flex sm:flex-wrap">
          <Button variant="outline" onClick={() => setPasswordOpen(true)}>
            <KeyIcon />
            {tn('changePassword')}
          </Button>
          <Button variant="danger-ghost" onClick={() => void logout()} disabled={loggingOut}>
            <ArrowRightStartOnRectangleIcon />
            {loggingOut ? t('loggingOut') : tn('logout')}
          </Button>
        </div>
      </Card>

      <ChangePasswordDialog open={passwordOpen} onOpenChange={setPasswordOpen} />
    </div>
  );
}

/** Xác thực 2 lớp (2FA TOTP) + Recovery Codes. */
function MfaSecurityCard({ initialEnabled }: { initialEnabled: boolean }) {
  const t = useTranslations('portal.account');
  const toastError = useToastApiError();

  const setupMutation = useSetupMemberMFAMutation();
  const enableMutation = useEnableMemberMFAMutation();
  const disableMutation = useDisableMemberMFAMutation();

  const [mfaEnabled, setMfaEnabled] = useState(initialEnabled);
  const [setupData, setSetupData] = useState<MFASetupRes | null>(null);
  const [otpCode, setOtpCode] = useState('');
  const [recoveryCodes, setRecoveryCodes] = useState<string[]>([]);

  async function handleSetup() {
    try {
      const res = await setupMutation.mutateAsync();
      setSetupData(res);
      setOtpCode('');
    } catch (error) {
      toastError(error);
    }
  }

  async function handleEnable() {
    const cleanCode = otpCode.trim();
    if (!/^[0-9]{6}$/.test(cleanCode)) return;
    try {
      const res = await enableMutation.mutateAsync({
        code: cleanCode,
        secret: setupData?.secret,
      });
      setMfaEnabled(true);
      setSetupData(null);
      setOtpCode('');
      if (res.recovery_codes && res.recovery_codes.length > 0) {
        setRecoveryCodes(res.recovery_codes);
      }
      toast.success(t('mfaEnableSuccess'));
    } catch (error) {
      toastError(error);
    }
  }

  async function handleDisable() {
    try {
      await disableMutation.mutateAsync(otpCode.trim() || undefined);
      setMfaEnabled(false);
      setSetupData(null);
      setRecoveryCodes([]);
      setOtpCode('');
      toast.success(t('mfaDisableSuccess'));
    } catch (error) {
      toastError(error);
    }
  }

  return (
    <Card>
      <CardHeader className="flex-row items-center justify-between gap-2">
        <div className="grid gap-0.5">
          <CardTitle>{t('mfaTitle')}</CardTitle>
          <p className="text-caption text-secondary">{t('mfaDesc')}</p>
        </div>
        <Badge variant={mfaEnabled ? 'success' : 'neutral'}>
          {mfaEnabled ? t('mfaEnabledBadge') : t('mfaDisabledBadge')}
        </Badge>
      </CardHeader>

      {recoveryCodes.length > 0 ? (
        <div className="grid gap-2 rounded-xl border border-warning bg-warning-50 p-4">
          <p className="text-body font-semibold text-primary">{t('mfaRecoveryTitle')}</p>
          <p className="text-caption text-secondary">{t('mfaRecoveryDesc')}</p>
          <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
            {recoveryCodes.map(code => (
              <code
                key={code}
                className="rounded-lg bg-surface px-2.5 py-1.5 text-center font-mono text-caption font-bold text-primary"
              >
                {code}
              </code>
            ))}
          </div>
        </div>
      ) : null}

      {mfaEnabled ? (
        <div className="flex flex-wrap items-end gap-3 rounded-xl bg-canvas p-4">
          <div className="grid flex-1 gap-1">
            <label className="text-caption font-medium text-secondary">
              {t('mfaDisableCodeLabel')}
            </label>
            <Input
              value={otpCode}
              onChange={e => setOtpCode(e.target.value)}
              placeholder={t('mfaDisableCodePlaceholder')}
              maxLength={32}
              // Tắt 2FA chấp nhận mã TOTP 6 số HOẶC mã khôi phục (chữ + số) → bàn phím chữ.
              inputMode="text"
              autoCapitalize="characters"
              autoComplete="one-time-code"
              className="max-w-xs font-mono"
            />
          </div>
          <Button
            type="button"
            variant="danger-ghost"
            disabled={disableMutation.isPending}
            onClick={() => void handleDisable()}
          >
            <LockClosedIcon />
            {disableMutation.isPending ? t('mfaDisabling') : t('mfaDisableBtn')}
          </Button>
        </div>
      ) : setupData ? (
        <div className="grid gap-4 rounded-xl bg-canvas p-4 sm:grid-cols-12 sm:items-center">
          <div className="flex justify-center sm:col-span-4">
            <div className="rounded-xl bg-white p-3 shadow-xs">
              <QRCodeSVG
                value={
                  setupData.otpauth_url ||
                  setupData.qr_url ||
                  `otpauth://totp/GymMaster?secret=${setupData.secret || setupData.totp_secret || ''}`
                }
                size={140}
              />
            </div>
          </div>
          <div className="grid gap-3 sm:col-span-8">
            <p className="text-caption text-secondary">{t('mfaScanHint')}</p>
            <div className="grid gap-0.5">
              <span className="text-caption text-tertiary">{t('mfaSecretLabel')}</span>
              <code className="rounded-lg bg-surface px-3 py-1.5 font-mono text-body font-bold text-brand-600">
                {setupData.secret || setupData.totp_secret}
              </code>
            </div>
            <div className="flex flex-wrap items-end gap-2">
              <div className="grid gap-1">
                <label className="text-caption font-medium text-secondary">
                  {t('mfaCodeLabel')}
                </label>
                <Input
                  value={otpCode}
                  onChange={e => setOtpCode(e.target.value)}
                  placeholder={t('mfaCodePlaceholder')}
                  maxLength={6}
                  inputMode="numeric"
                  className="w-48 font-mono"
                />
              </div>
              <Button
                type="button"
                disabled={!/^[0-9]{6}$/.test(otpCode.trim()) || enableMutation.isPending}
                onClick={() => void handleEnable()}
              >
                <ShieldCheckIcon />
                {enableMutation.isPending ? t('mfaEnabling') : t('mfaEnableBtn')}
              </Button>
            </div>
          </div>
        </div>
      ) : (
        <div>
          <Button
            type="button"
            variant="outline"
            disabled={setupMutation.isPending}
            onClick={() => void handleSetup()}
          >
            <ShieldCheckIcon />
            {setupMutation.isPending ? t('mfaSettingUp') : t('mfaSetupBtn')}
          </Button>
        </div>
      )}
    </Card>
  );
}

/** Số phiên hiển thị trước khi bấm "Xem thêm". */
const SESSIONS_COLLAPSED = 4;

/** Danh sách thiết bị / phiên đăng nhập: phiên hiện tại lên đầu, thu hồi từng phiên hoặc mọi phiên khác. */
function ActiveSessionsCard() {
  const t = useTranslations('portal.account');
  const toastError = useToastApiError();

  const sessionsQuery = useMemberSessionsQuery();
  const revokeOthersMutation = useRevokeOtherMemberSessionsMutation();
  const revokeOneMutation = useRevokeMemberSessionMutation();
  const [expanded, setExpanded] = useState(false);
  const [confirmOthers, setConfirmOthers] = useState(false);
  const [revokeTarget, setRevokeTarget] = useState<SessionItem | null>(null);

  const sessions = [...(sessionsQuery.data ?? [])].sort(
    (a, b) => Number(isCurrentSession(b)) - Number(isCurrentSession(a)),
  );
  const current = sessions.find(isCurrentSession);
  const otherCount = sessions.filter(sess => !isCurrentSession(sess)).length;
  const visible = expanded ? sessions : sessions.slice(0, SESSIONS_COLLAPSED);

  async function handleRevokeOthers() {
    try {
      await revokeOthersMutation.mutateAsync(current?.id);
      toast.success(t('revokeOthersSuccess'));
      setConfirmOthers(false);
    } catch (error) {
      toastError(error);
    }
  }

  async function handleRevokeOne() {
    if (!revokeTarget) return;
    try {
      await revokeOneMutation.mutateAsync(revokeTarget.id);
      toast.success(t('revokeSessionSuccess'));
      setRevokeTarget(null);
    } catch (error) {
      toastError(error);
    }
  }

  return (
    <Card>
      <CardHeader className="flex-row flex-wrap items-center justify-between gap-2">
        <div>
          <CardTitle>{t('sessionsTitle')}</CardTitle>
          <p className="text-caption text-secondary">{t('sessionsDesc')}</p>
        </div>
        {otherCount > 0 ? (
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={revokeOthersMutation.isPending}
            onClick={() => setConfirmOthers(true)}
          >
            <TrashIcon />
            {t('revokeOthersBtn', { count: otherCount })}
          </Button>
        ) : null}
      </CardHeader>

      {sessionsQuery.isLoading ? (
        <Skeleton className="h-20 rounded-xl" />
      ) : sessionsQuery.isError ? (
        <LoadError error={sessionsQuery.error} onRetry={() => void sessionsQuery.refetch()} />
      ) : sessions.length === 0 ? (
        <div className="flex items-center gap-3 rounded-xl bg-canvas p-4 text-secondary">
          <ComputerDesktopIcon className="size-5 shrink-0 text-tertiary" />
          <span className="text-body">{t('emptySessions')}</span>
        </div>
      ) : (
        <div className="grid gap-2.5">
          <ul className="grid gap-2.5">
            {visible.map(session => {
              const isCurrent = isCurrentSession(session);
              const ip = session.ip_address || session.ip;
              const ua = session.user_agent || session.device || session.device_name || '';
              const label = describeUserAgent(ua) || ua || t('sessionFallback', { id: session.id });
              const lastUsed = session.last_used_at || session.created_at;
              return (
                <li
                  key={session.id}
                  className="flex flex-wrap items-center justify-between gap-3 rounded-xl bg-canvas p-3.5"
                >
                  <div className="flex min-w-0 items-center gap-3">
                    <ComputerDesktopIcon className="size-6 shrink-0 text-brand-600" />
                    <div className="grid min-w-0 gap-0.5">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="text-body font-medium break-words text-primary" title={ua}>
                          {label}
                        </span>
                        {isCurrent ? (
                          <Badge variant="success">{t('currentSessionBadge')}</Badge>
                        ) : null}
                      </div>
                      <p className="text-caption text-secondary">
                        {[
                          ip ? t('sessionIp', { ip }) : null,
                          lastUsed ? t('lastUsedAt', { time: formatDateTime(lastUsed) }) : null,
                        ]
                          .filter(Boolean)
                          .join(' · ')}
                      </p>
                    </div>
                  </div>
                  {!isCurrent ? (
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      disabled={revokeOneMutation.isPending}
                      onClick={() => setRevokeTarget(session)}
                    >
                      <ArrowRightStartOnRectangleIcon />
                      {t('revokeSessionBtn')}
                    </Button>
                  ) : null}
                </li>
              );
            })}
          </ul>
          {sessions.length > SESSIONS_COLLAPSED ? (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="justify-self-start"
              onClick={() => setExpanded(v => !v)}
            >
              {expanded
                ? t('sessionsShowLess')
                : t('sessionsShowMore', { count: sessions.length - SESSIONS_COLLAPSED })}
            </Button>
          ) : null}
        </div>
      )}

      <ConfirmDialog
        open={confirmOthers}
        onOpenChange={setConfirmOthers}
        title={t('revokeOthersConfirmTitle')}
        description={t('revokeOthersConfirmDesc', { count: otherCount })}
        confirmLabel={t('revokeOthersConfirmBtn')}
        loading={revokeOthersMutation.isPending}
        onConfirm={handleRevokeOthers}
      />
      <ConfirmDialog
        open={Boolean(revokeTarget)}
        onOpenChange={open => !open && setRevokeTarget(null)}
        title={t('revokeSessionConfirmTitle')}
        description={t('revokeSessionConfirmDesc', {
          device:
            describeUserAgent(revokeTarget?.user_agent) ||
            revokeTarget?.user_agent ||
            t('sessionFallback', { id: revokeTarget?.id ?? 0 }),
        })}
        confirmLabel={t('revokeSessionBtn')}
        loading={revokeOneMutation.isPending}
        onConfirm={handleRevokeOne}
      />
    </Card>
  );
}

function isCurrentSession(session: SessionItem): boolean {
  return Boolean(session.is_current ?? session.current);
}

/** Quyền riêng tư & Dữ liệu cá nhân (Nghị định 13/2023/NĐ-CP). */
function PrivacyDataCard() {
  const t = useTranslations('portal.account');
  const toastError = useToastApiError();

  const consentsQuery = useMemberConsentsQuery();
  const updateConsentMutation = useUpdateMemberConsentsMutation();
  const dataRequestsQuery = useMemberDataRequestsQuery();
  const createDataRequestMutation = useCreateMemberDataRequestMutation();

  const [reason, setReason] = useState('');
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [downloadingId, setDownloadingId] = useState<number | null>(null);

  const consents = consentsQuery.data ?? [];
  const dataRequests = dataRequestsQuery.data ?? [];

  function isGranted(type: ConsentType, defaultVal = false): boolean {
    const found = consents.find(c => (c.consent_type || c.type) === type);
    return found ? found.granted : defaultVal;
  }

  async function handleToggleConsent(type: ConsentType, granted: boolean) {
    try {
      await updateConsentMutation.mutateAsync({
        type,
        consent_type: type,
        granted,
      });
      toast.success(t('consentUpdateSuccess'));
    } catch (error) {
      toastError(error);
    }
  }

  async function handleDownload(id: number) {
    setDownloadingId(id);
    try {
      const { blob, filename } = await downloadMemberDataExport(id);
      saveBlob(blob, filename || `gymmaster-du-lieu-ca-nhan-${id}.json`);
    } catch (error) {
      toastError(error);
    } finally {
      setDownloadingId(null);
    }
  }

  async function handleRequestData(type: DataRequestType) {
    try {
      await createDataRequestMutation.mutateAsync({
        type,
        reason: reason.trim() || undefined,
      });
      setReason('');
      setConfirmDelete(false);
      toast.success(type === 'EXPORT' ? t('requestExportSuccess') : t('requestDeleteSuccess'));
    } catch (error) {
      toastError(error);
    }
  }

  const consentItems: Array<{
    type: ConsentType;
    title: string;
    desc: string;
    defaultGranted: boolean;
  }> = [
    {
      type: 'TERMS',
      title: t('consentTermsTitle'),
      desc: t('consentTermsDesc'),
      defaultGranted: true,
    },
    {
      type: 'MARKETING',
      title: t('consentMarketingTitle'),
      desc: t('consentMarketingDesc'),
      defaultGranted: false,
    },
    {
      type: 'PHOTO',
      title: t('consentPhotoTitle'),
      desc: t('consentPhotoDesc'),
      defaultGranted: false,
    },
  ];

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('privacyTitle')}</CardTitle>
        <p className="text-caption text-secondary">{t('privacyDesc')}</p>
      </CardHeader>

      <div className="grid gap-5">
        <div className="grid gap-3">
          {consentItems.map(item => {
            const checked = isGranted(item.type, item.defaultGranted);
            return (
              <label
                key={item.type}
                className="flex cursor-pointer items-start gap-3 rounded-xl bg-canvas p-3.5"
              >
                <Checkbox
                  checked={checked}
                  disabled={updateConsentMutation.isPending}
                  onCheckedChange={val => void handleToggleConsent(item.type, Boolean(val))}
                  className="mt-0.5"
                />
                <span className="grid gap-0.5">
                  <span className="text-body font-medium text-primary">{item.title}</span>
                  <span className="text-caption text-secondary">{item.desc}</span>
                </span>
              </label>
            );
          })}
        </div>

        <div className="grid gap-3 rounded-xl border border-line bg-canvas p-4">
          <p className="text-body font-semibold text-primary">{t('dataRequestsTitle')}</p>
          <Input
            value={reason}
            onChange={e => setReason(e.target.value)}
            placeholder={t('requestReasonPlaceholder')}
          />
          <div className="flex flex-wrap gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={createDataRequestMutation.isPending}
              onClick={() => void handleRequestData('EXPORT')}
            >
              <ArrowDownTrayIcon />
              {t('requestExportBtn')}
            </Button>
            <Button
              type="button"
              variant="danger-ghost"
              size="sm"
              disabled={createDataRequestMutation.isPending}
              onClick={() => setConfirmDelete(true)}
            >
              <TrashIcon />
              {t('requestDeleteBtn')}
            </Button>
          </div>

          {dataRequests.length === 0 ? (
            <p className="text-caption text-tertiary">{t('emptyDataRequests')}</p>
          ) : (
            <ul className="grid gap-2 pt-2">
              {dataRequests.map(req => (
                <li
                  key={req.id}
                  className="flex flex-wrap items-center justify-between gap-2 rounded-lg bg-surface p-3"
                >
                  <div className="flex flex-wrap items-center gap-2">
                    <Badge variant="outline">
                      {req.type === 'EXPORT'
                        ? t('dataRequestTypes.EXPORT')
                        : req.type === 'DELETE'
                          ? t('dataRequestTypes.DELETE')
                          : req.type}
                    </Badge>
                    <DataRequestStatusBadge status={req.status} />
                    <span className="text-caption text-secondary">
                      {formatDateTime(req.created_at)}
                    </span>
                  </div>
                  {req.type === 'EXPORT' && req.status === 'DONE' ? (
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      disabled={downloadingId === req.id}
                      onClick={() => void handleDownload(req.id)}
                    >
                      <ArrowDownTrayIcon />
                      {downloadingId === req.id ? t('downloadingExport') : t('downloadExportBtn')}
                    </Button>
                  ) : null}
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>

      <ConfirmDialog
        open={confirmDelete}
        onOpenChange={setConfirmDelete}
        title={t('requestDeleteConfirmTitle')}
        description={t('requestDeleteConfirmDesc')}
        confirmLabel={t('requestDeleteConfirmBtn')}
        loading={createDataRequestMutation.isPending}
        onConfirm={() => handleRequestData('DELETE')}
      />
    </Card>
  );
}

/** Lưu Blob thành file tải về trên trình duyệt. */
function saveBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  a.remove();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}
