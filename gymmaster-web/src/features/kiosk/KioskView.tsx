'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import Link from 'next/link';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import {
  CheckCircleIcon,
  Cog6ToothIcon,
  KeyIcon,
  QrCodeIcon,
  SignalIcon,
  VideoCameraIcon,
  XCircleIcon,
} from '@heroicons/react/24/outline';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { deviceCheckin, deviceHeartbeat } from '@/features/portal/api';
import type { DeviceCheckinRes } from '@/features/portal/types';
import { ApiError } from '@/types/api';

const DEVICE_KEY_STORAGE = 'gymmaster:kiosk_device_key';
const DEFAULT_DEMO_KEY = 'GMD-demo-kiosk-cg01';
const HEARTBEAT_INTERVAL_MS = 60_000;
const RESULT_RESET_MS = 3_000;

interface KioskResultState {
  allowed: boolean;
  memberName: string;
  memberCode?: string;
  planName?: string;
  lockerCode?: string;
  message: string;
}

export function KioskView() {
  const t = useTranslations('portal.kiosk');
  const tc = useTranslations('common');

  const [deviceKey, setDeviceKey] = useState<string>(DEFAULT_DEMO_KEY);
  const [draftKey, setDraftKey] = useState<string>(DEFAULT_DEMO_KEY);
  const [configOpen, setConfigOpen] = useState(false);
  const [deviceInfo, setDeviceInfo] = useState<{ name?: string; branchName?: string } | null>(null);
  const [heartbeatOk, setHeartbeatOk] = useState<boolean>(false);

  const [codeInput, setCodeInput] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [scanResult, setScanResult] = useState<KioskResultState | null>(null);
  const [cameraActive, setCameraActive] = useState(false);

  const inputRef = useRef<HTMLInputElement | null>(null);
  const resetTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (typeof window === 'undefined') return;
    const saved = window.localStorage.getItem(DEVICE_KEY_STORAGE);
    if (saved && saved.trim()) {
      setDeviceKey(saved.trim());
      setDraftKey(saved.trim());
    } else {
      window.localStorage.setItem(DEVICE_KEY_STORAGE, DEFAULT_DEMO_KEY);
    }
  }, []);

  const focusInput = useCallback(() => {
    if (!configOpen && !cameraActive) {
      inputRef.current?.focus();
    }
  }, [configOpen, cameraActive]);

  useEffect(() => {
    focusInput();
  }, [focusInput, scanResult]);

  // Định kỳ mỗi 60 giây gửi POST /v1/public/device/heartbeat
  useEffect(() => {
    if (!deviceKey) return;

    const sendBeat = async () => {
      try {
        const res = await deviceHeartbeat(deviceKey);
        setHeartbeatOk(true);
        if (res?.device) {
          setDeviceInfo({
            name: res.device.name,
            branchName: res.device.branch_name,
          });
        }
      } catch {
        setHeartbeatOk(false);
      }
    };

    void sendBeat();
    const timer = setInterval(() => {
      void sendBeat();
    }, HEARTBEAT_INTERVAL_MS);

    return () => clearInterval(timer);
  }, [deviceKey]);

  const scheduleAutoReset = useCallback(() => {
    if (resetTimerRef.current) {
      clearTimeout(resetTimerRef.current);
    }
    resetTimerRef.current = setTimeout(() => {
      setScanResult(null);
      setCodeInput('');
      setTimeout(() => {
        inputRef.current?.focus();
      }, 50);
    }, RESULT_RESET_MS);
  }, []);

  useEffect(() => {
    return () => {
      if (resetTimerRef.current) {
        clearTimeout(resetTimerRef.current);
      }
    };
  }, []);

  const handleCheckinToken = useCallback(
    async (rawToken: string) => {
      const token = rawToken.trim();
      if (!token || submitting) return;

      setSubmitting(true);
      try {
        const res: DeviceCheckinRes = await deviceCheckin(token, deviceKey);
        const allowed = Boolean(res.allow ?? res.allowed ?? res.result === 'ALLOWED');
        setScanResult({
          allowed,
          memberName: res.member_name || res.full_name || 'Hội viên / Khách tập',
          memberCode: res.member_code,
          planName: res.plan_name,
          lockerCode: res.locker_code,
          message:
            res.message ||
            res.deny_message ||
            (allowed ? 'Check-in hợp lệ. Chúc bạn buổi tập hiệu quả!' : 'Không đủ điều kiện vào cửa.'),
        });
      } catch (err) {
        const errMsg =
          err instanceof ApiError
            ? err.message
            : 'Mã QR không hợp lệ hoặc thiết bị chưa được xác thực.';
        setScanResult({
          allowed: false,
          memberName: 'Không xác định',
          planName: token.startsWith('TP-')
            ? 'Vé tập thử (Trial Pass)'
            : token.startsWith('DP-')
              ? 'Vé ngày / Khách mời (Day Pass)'
              : undefined,
          message: errMsg,
        });
      } finally {
        setSubmitting(false);
        setCodeInput('');
        scheduleAutoReset();
      }
    },
    [deviceKey, scheduleAutoReset, submitting],
  );

  // Tích hợp quét camera QR nếu bật
  useEffect(() => {
    if (!cameraActive) return;
    let scannerCleanup: (() => Promise<void>) | null = null;
    let mounted = true;

    const startCamera = async () => {
      try {
        const { Html5Qrcode } = await import('html5-qrcode');
        if (!mounted) return;
        const qr = new Html5Qrcode('kiosk-qr-reader');
        scannerCleanup = async () => {
          try {
            if (qr.isScanning) await qr.stop();
            qr.clear();
          } catch {
            // Ignore cleanup errors
          }
        };
        await qr.start(
          { facingMode: 'environment' },
          { fps: 10, qrbox: { width: 240, height: 240 } },
          decodedText => {
            void handleCheckinToken(decodedText);
          },
          () => {
            // Ignore frame errors
          },
        );
      } catch {
        toast.error('Không thể truy cập camera trên thiết bị này. Vui lòng dùng máy quét USB hoặc nhập mã.');
        setCameraActive(false);
      }
    };

    void startCamera();
    return () => {
      mounted = false;
      if (scannerCleanup) {
        void scannerCleanup();
      }
    };
  }, [cameraActive, handleCheckinToken]);

  const handleSaveDeviceKey = () => {
    const clean = draftKey.trim();
    if (!clean) return;
    setDeviceKey(clean);
    if (typeof window !== 'undefined') {
      window.localStorage.setItem(DEVICE_KEY_STORAGE, clean);
    }
    setConfigOpen(false);
    toast.success('Đã lưu cấu hình Device Key cho Kiosk');
  };

  return (
    <div
      className="flex min-h-dvh flex-col justify-between bg-slate-950 p-6 text-white select-none sm:p-10"
      onClick={() => focusInput()}
    >
      {/* Top Bar */}
      <header className="flex flex-wrap items-center justify-between gap-4 border-b border-slate-800 pb-5">
        <div className="flex items-center gap-3">
          <div className="flex size-12 items-center justify-center rounded-2xl bg-emerald-500/20 text-emerald-400">
            <QrCodeIcon className="size-7" />
          </div>
          <div>
            <div className="flex items-center gap-2.5">
              <span className="text-h4 font-bold tracking-tight text-white">GYMMASTER KIOSK</span>
              <span
                className={`inline-flex items-center gap-1.5 rounded-full px-3 py-0.5 text-caption font-semibold ${
                  deviceKey
                    ? heartbeatOk
                      ? 'bg-emerald-500/20 text-emerald-300'
                      : 'bg-sky-500/20 text-sky-300'
                    : 'bg-amber-500/20 text-amber-300'
                }`}
              >
                <SignalIcon className="size-3.5" />
                {deviceKey
                  ? deviceInfo?.name
                    ? `${deviceInfo.name}${deviceInfo.branchName ? ` · ${deviceInfo.branchName}` : ''}`
                    : t('deviceConnected')
                  : t('deviceUnconfigured')}
              </span>
            </div>
            <p className="text-caption text-slate-400">
              Device Key: <code className="text-slate-300">{deviceKey || '—'}</code>
            </p>
          </div>
        </div>

        <div className="flex items-center gap-2" onClick={e => e.stopPropagation()}>
          <Button
            variant="outline"
            size="sm"
            className="border-slate-700 bg-slate-900 text-slate-200 hover:bg-slate-800 hover:text-white"
            onClick={() => setCameraActive(prev => !prev)}
          >
            <VideoCameraIcon className="size-4" />
            {cameraActive ? t('closeCameraBtn') : t('toggleCameraBtn')}
          </Button>
          <Button
            variant="outline"
            size="sm"
            className="border-slate-700 bg-slate-900 text-slate-200 hover:bg-slate-800 hover:text-white"
            onClick={() => {
              setDraftKey(deviceKey);
              setConfigOpen(true);
            }}
          >
            <Cog6ToothIcon className="size-4" />
            {t('deviceConfigBtn')}
          </Button>
        </div>
      </header>

      {/* Main Content: Ready State vs Fullscreen Scan Result */}
      <main className="mx-auto flex w-full max-w-4xl flex-1 flex-col items-center justify-center py-8">
        {scanResult ? (
          <div
            onClick={() => {
              setScanResult(null);
              focusInput();
            }}
            className={`w-full cursor-pointer rounded-3xl border-2 p-8 text-center shadow-2xl transition-all sm:p-14 ${
              scanResult.allowed
                ? 'border-emerald-400 bg-emerald-950/90 text-white'
                : 'border-rose-400 bg-rose-950/90 text-white'
            }`}
          >
            <div className="mx-auto mb-6 flex size-24 items-center justify-center rounded-full bg-white/15">
              {scanResult.allowed ? (
                <CheckCircleIcon className="size-16 text-emerald-300" />
              ) : (
                <XCircleIcon className="size-16 text-rose-300" />
              )}
            </div>

            <h1
              className={`text-4xl font-extrabold tracking-wide uppercase sm:text-5xl ${
                scanResult.allowed ? 'text-emerald-300' : 'text-rose-300'
              }`}
            >
              {scanResult.allowed ? t('allowedTitle') : t('deniedTitle')}
            </h1>

            <p className="mt-4 text-2xl font-bold text-white sm:text-3xl">
              {scanResult.memberName}
              {scanResult.memberCode ? (
                <span className="ml-2 text-lg font-normal text-white/75">
                  ({scanResult.memberCode})
                </span>
              ) : null}
            </p>

            {scanResult.planName ? (
              <div className="mt-4 inline-flex items-center gap-2 rounded-full bg-white/15 px-5 py-2 text-lg font-semibold text-white">
                <span>{t('planLabel')}:</span>
                <span>{scanResult.planName}</span>
              </div>
            ) : null}

            {scanResult.lockerCode ? (
              <div className="mt-3">
                <span className="inline-flex items-center gap-2 rounded-full bg-amber-400/20 px-4 py-1.5 text-base font-semibold text-amber-200">
                  {t('lockerLabel')}: {scanResult.lockerCode}
                </span>
              </div>
            ) : null}

            <p className="mt-6 text-xl font-medium text-white/90">{scanResult.message}</p>

            <p className="mt-8 text-caption text-white/60">{t('autoResetHint')}</p>
          </div>
        ) : (
          <div className="grid w-full max-w-2xl gap-8 text-center">
            <div className="grid gap-3">
              <div className="mx-auto flex size-20 items-center justify-center rounded-3xl border border-slate-800 bg-slate-900 text-emerald-400 shadow-lg">
                <QrCodeIcon className="size-12" />
              </div>
              <h1 className="text-3xl font-extrabold tracking-tight text-white sm:text-4xl">
                {t('title')}
              </h1>
              <p className="text-lg text-slate-300">{t('subtitle')}</p>
            </div>

            {cameraActive ? (
              <div className="mx-auto w-full max-w-sm overflow-hidden rounded-2xl border border-slate-700 bg-slate-900 p-3">
                <div id="kiosk-qr-reader" className="w-full overflow-hidden rounded-xl" />
              </div>
            ) : null}

            <form
              onSubmit={e => {
                e.preventDefault();
                void handleCheckinToken(codeInput);
              }}
              className="grid gap-4 rounded-3xl border border-slate-800 bg-slate-900/90 p-6 shadow-xl sm:p-8"
              onClick={e => e.stopPropagation()}
            >
              <label
                htmlFor="kiosk-code-input"
                className="text-left text-body font-medium text-slate-300"
              >
                {t('inputLabel')}
              </label>
              <div className="flex flex-col gap-3 sm:flex-row">
                <input
                  id="kiosk-code-input"
                  ref={inputRef}
                  type="text"
                  value={codeInput}
                  onChange={e => setCodeInput(e.target.value)}
                  placeholder={t('inputPlaceholder')}
                  autoFocus
                  autoComplete="off"
                  className="h-14 flex-1 rounded-2xl border border-slate-700 bg-slate-950 px-5 font-mono text-lg text-white placeholder:text-slate-500 focus:border-emerald-400 focus:ring-2 focus:ring-emerald-400/30 focus:outline-none"
                />
                <Button
                  type="submit"
                  disabled={submitting || !codeInput.trim()}
                  className="h-14 rounded-2xl bg-emerald-500 px-8 text-base font-bold text-slate-950 hover:bg-emerald-400"
                >
                  {submitting ? t('checkingIn') : t('checkInBtn')}
                </Button>
              </div>
              <div className="flex flex-wrap items-center justify-center gap-2 pt-1 text-caption text-slate-400">
                <span>Hỗ trợ mã nhanh:</span>
                <button
                  type="button"
                  onClick={() => void handleCheckinToken('GM000001')}
                  className="rounded-lg border border-slate-700 bg-slate-800 px-2.5 py-1 font-mono text-slate-200 hover:bg-slate-700"
                >
                  GM000001
                </button>
                <button
                  type="button"
                  onClick={() => void handleCheckinToken('DP-DEMO202601')}
                  className="rounded-lg border border-slate-700 bg-slate-800 px-2.5 py-1 font-mono text-slate-200 hover:bg-slate-700"
                >
                  DP-DEMO202601
                </button>
              </div>
            </form>
          </div>
        )}
      </main>

      {/* Footer */}
      <footer className="flex flex-wrap items-center justify-between gap-4 border-t border-slate-900 pt-4 text-caption text-slate-500">
        <span>GymMaster Smart Access Control · Kiosk Mode</span>
        <Link href="/" className="text-slate-400 hover:text-white">
          Về trang chủ
        </Link>
      </footer>

      {/* Device Key Config Dialog */}
      <Dialog open={configOpen} onOpenChange={setConfigOpen}>
        <DialogContent closeLabel={tc('actions.close')}>
          <DialogHeader>
            <DialogTitle>{t('deviceKeyTitle')}</DialogTitle>
            <DialogDescription>{t('deviceKeyDesc')}</DialogDescription>
          </DialogHeader>
          <div className="grid gap-3 py-2">
            <Input
              value={draftKey}
              onChange={e => setDraftKey(e.target.value)}
              placeholder={t('deviceKeyPlaceholder')}
              className="font-mono"
            />
            <div>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => setDraftKey(DEFAULT_DEMO_KEY)}
              >
                <KeyIcon className="size-4" />
                {t('useDemoKeyBtn')}
              </Button>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setConfigOpen(false)}>
              {tc('actions.cancel')}
            </Button>
            <Button onClick={handleSaveDeviceKey}>{t('saveDeviceKeyBtn')}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
