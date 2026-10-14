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
  useRevokeOtherMemberSessionsMutation,
  useSetupMemberMFAMutation,
  useUpdateMemberConsentsMutation,
} from '../queries';
import type { ConsentType, DataRequestType, MFASetupRes } from '../types';
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

      <MfaSecurityCard initialEnabled={Boolean(query.data?.member.mfa_enabled)} />

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
              {t('mfaCodeLabel')}
            </label>
            <Input
              value={otpCode}
              onChange={e => setOtpCode(e.target.value)}
              placeholder={t('mfaCodePlaceholder')}
              maxLength={6}
              inputMode="numeric"
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

/** Danh sách thiết bị / phiên đăng nhập & thu hồi các phiên khác. */
function ActiveSessionsCard() {
  const t = useTranslations('portal.account');
  const toastError = useToastApiError();

  const sessionsQuery = useMemberSessionsQuery();
  const revokeMutation = useRevokeOtherMemberSessionsMutation();

  const sessions = sessionsQuery.data ?? [];

  async function handleRevokeOthers() {
    try {
      await revokeMutation.mutateAsync();
      toast.success(t('revokeOthersSuccess'));
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
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={revokeMutation.isPending}
          onClick={() => void handleRevokeOthers()}
        >
          <TrashIcon />
          {revokeMutation.isPending ? t('revokingOthers') : t('revokeOthersBtn')}
        </Button>
      </CardHeader>

      {sessionsQuery.isLoading ? (
        <Skeleton className="h-20 rounded-xl" />
      ) : sessions.length === 0 ? (
        <div className="flex items-center gap-3 rounded-xl bg-canvas p-4 text-secondary">
          <ComputerDesktopIcon className="size-5 shrink-0 text-tertiary" />
          <span className="text-body">{t('emptySessions')}</span>
        </div>
      ) : (
        <ul className="grid gap-2.5">
          {sessions.map(session => {
            const isCurrent = Boolean(session.is_current ?? session.current);
            const ip = session.ip_address || session.ip;
            return (
              <li
                key={session.id}
                className="flex flex-wrap items-center justify-between gap-3 rounded-xl bg-canvas p-3.5"
              >
                <div className="flex items-center gap-3">
                  <ComputerDesktopIcon className="size-6 shrink-0 text-brand-600" />
                  <div className="grid gap-0.5">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="text-body font-medium text-primary">
                        {session.device ||
                          session.device_name ||
                          session.user_agent ||
                          `Session #${session.id}`}
                      </span>
                      {isCurrent ? (
                        <Badge variant="success">{t('currentSessionBadge')}</Badge>
                      ) : null}
                    </div>
                    <p className="text-caption text-secondary">
                      {ip ? `IP: ${ip} · ` : ''}
                      {session.last_used_at || session.created_at
                        ? t('lastUsedAt', {
                            time: formatDateTime(session.last_used_at || session.created_at!),
                          })
                        : ''}
                    </p>
                  </div>
                </div>
              </li>
            );
          })}
        </ul>
      )}
    </Card>
  );
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

  async function handleRequestData(type: DataRequestType) {
    try {
      await createDataRequestMutation.mutateAsync({
        type,
        reason: reason.trim() || undefined,
      });
      setReason('');
      toast.success(
        type === 'EXPORT' ? t('requestExportSuccess') : t('requestDeleteSuccess'),
      );
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
                  onCheckedChange={val =>
                    void handleToggleConsent(item.type, Boolean(val))
                  }
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
              onClick={() => void handleRequestData('DELETE')}
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
                    <Badge variant="outline">{req.type}</Badge>
                    <DataRequestStatusBadge status={req.status} />
                    <span className="text-caption text-secondary">
                      {formatDateTime(req.created_at)}
                    </span>
                  </div>
                  {req.export_url ? (
                    <Button asChild variant="outline" size="sm">
                      <a href={req.export_url} target="_blank" rel="noreferrer">
                        <ArrowDownTrayIcon />
                        {t('downloadExportBtn')}
                      </a>
                    </Button>
                  ) : null}
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </Card>
  );
}
