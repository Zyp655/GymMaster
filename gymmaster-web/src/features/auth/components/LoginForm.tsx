'use client';

import { useEffect, useMemo, useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import { signIn } from 'next-auth/react';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import { looseT } from '@/lib/utils';
import { ApiCode } from '@/types/api';
import {
  createLoginSchema,
  isValidMfaCode,
  MFA_RECOVERY_MAX_LENGTH,
  MFA_TOTP_LENGTH,
  normalizeMfaCode,
  type LoginFormValues,
  type MfaInputMode,
} from '../schema';
import { safeReturnUrl } from '@/lib/safe-return-url';
import { PasswordInput } from './PasswordInput';


export function LoginForm() {
  const t = useTranslations('auth');
  const tv = useTranslations('common.validation');
  const router = useRouter();
  const searchParams = useSearchParams();
  const returnUrl = safeReturnUrl(searchParams.get('returnUrl'));
  const expired = searchParams.get('expired') === '1';

  useEffect(() => {
    if (expired) toast.info(t('login.sessionExpired'), { id: 'session-expired' });
  }, [expired, t]);

  const schema = useMemo(() => createLoginSchema(looseT(tv)), [tv]);
  const form = useForm<LoginFormValues>({
    resolver: zodResolver(schema),
    defaultValues: { email: '', password: '', mfa_code: '' },
  });
  /** Bước 2: API trả 40107 (tài khoản bật 2FA) → hiện ô nhập mã, gửi lại kèm email/mật khẩu. */
  const [mfaStep, setMfaStep] = useState(false);
  /** TOTP 6 số (bàn phím số) hoặc mã khôi phục chữ + số (bàn phím chữ trên di động). */
  const [mfaMode, setMfaMode] = useState<MfaInputMode>('totp');

  const toggleMfaMode = () => {
    setMfaMode(mode => (mode === 'totp' ? 'recovery' : 'totp'));
    form.setValue('mfa_code', '');
    form.clearErrors('mfa_code');
    setTimeout(() => form.setFocus('mfa_code'), 0);
  };

  const onSubmit = form.handleSubmit(async values => {
    const mfaCode = normalizeMfaCode(values.mfa_code, mfaMode);
    if (mfaStep && !isValidMfaCode(mfaCode, mfaMode)) {
      form.setError('mfa_code', {
        message: !mfaCode
          ? tv('required')
          : mfaMode === 'totp'
            ? t('errors.mfaTotpFormat')
            : t('errors.mfaRecoveryFormat'),
      });
      form.setFocus('mfa_code');
      return;
    }
    const result = await signIn('credentials', {
      email: values.email,
      password: values.password,
      ...(mfaStep ? { mfa_code: mfaCode } : {}),
      redirect: false,
    });
    if (!result || result.error) {
      const code = Number(result?.error);
      if (code === ApiCode.MFA_REQUIRED) {
        setMfaStep(true);
        toast.info(t('errors.mfaRequired'));
        setTimeout(() => form.setFocus('mfa_code'), 0);
        return;
      }
      if (code === ApiCode.MFA_INVALID_CODE) {
        setMfaStep(true);
        form.setError('mfa_code', { message: t('errors.mfaInvalid') });
        toast.error(t('errors.mfaInvalid'));
        form.setFocus('mfa_code');
        return;
      }
      const message =
        code === ApiCode.INVALID_CREDENTIALS
          ? t('errors.invalidCredentials')
          : code === ApiCode.ACCOUNT_LOCKED
            ? t('errors.accountLocked')
            : code === ApiCode.TOO_MANY_ATTEMPTS
              ? t('errors.tooManyAttempts')
              : code === ApiCode.TOO_MANY_REQUESTS
                ? t('errors.tooManyRequests')
                : code === ApiCode.NETWORK
                  ? t('errors.network')
                  : t('errors.unknown');
      toast.error(message);
      if (code === ApiCode.INVALID_CREDENTIALS) form.setFocus('password');
      return;
    }
    toast.success(t('login.success'));
    router.replace(returnUrl);
    router.refresh();
  });

  const isSubmitting = form.formState.isSubmitting;

  return (
    <Form {...form}>
      <form onSubmit={onSubmit} className="grid gap-5" noValidate>
        <FormField
          control={form.control}
          name="email"
          render={({ field }) => (
            <FormItem>
              <FormLabel>{t('login.email')}</FormLabel>
              <FormControl>
                <Input
                  {...field}
                  onChange={e => {
                    field.onChange(e);
                    // Đổi email → bỏ bước 2FA (tài khoản khác có thể chưa bật 2FA).
                    if (mfaStep) {
                      setMfaStep(false);
                      setMfaMode('totp');
                      form.setValue('mfa_code', '');
                    }
                  }}
                  type="email"
                  autoComplete="email"
                  autoFocus
                  placeholder={t('login.emailPlaceholder')}
                  className="h-12"
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="password"
          render={({ field }) => (
            <FormItem>
              <div className="flex items-center justify-between gap-2">
                <FormLabel>{t('login.password')}</FormLabel>
                <Link
                  href="/forgot-password"
                  className="text-caption font-medium text-brand-600 hover:underline"
                >
                  {t('login.forgotPasswordLink')}
                </Link>
              </div>
              <FormControl>
                <PasswordInput
                  {...field}
                  autoComplete="current-password"
                  placeholder={t('login.passwordPlaceholder')}
                  className="h-12"
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        {mfaStep ? (
          <FormField
            control={form.control}
            name="mfa_code"
            render={({ field }) => (
              <FormItem>
                <FormLabel required>
                  {mfaMode === 'totp' ? t('login.mfaCode') : t('login.mfaRecoveryCode')}
                </FormLabel>
                <p className="text-caption text-secondary">
                  {mfaMode === 'totp' ? t('login.mfaHint') : t('login.mfaRecoveryHint')}
                </p>
                <FormControl>
                  {mfaMode === 'totp' ? (
                    <Input
                      {...field}
                      key="mfa-totp"
                      value={field.value ?? ''}
                      autoComplete="one-time-code"
                      inputMode="numeric"
                      maxLength={MFA_TOTP_LENGTH}
                      placeholder={t('login.mfaCodePlaceholder')}
                      className="h-12 tracking-widest"
                    />
                  ) : (
                    <Input
                      {...field}
                      key="mfa-recovery"
                      value={field.value ?? ''}
                      autoComplete="off"
                      inputMode="text"
                      autoCapitalize="characters"
                      autoCorrect="off"
                      spellCheck={false}
                      maxLength={MFA_RECOVERY_MAX_LENGTH}
                      placeholder={t('login.mfaRecoveryPlaceholder')}
                      className="h-12 font-mono tracking-widest uppercase"
                    />
                  )}
                </FormControl>
                <FormMessage />
                <Button
                  type="button"
                  variant="link"
                  className="h-auto justify-self-start px-0 text-caption"
                  onClick={toggleMfaMode}
                >
                  {mfaMode === 'totp' ? t('login.mfaUseRecovery') : t('login.mfaUseTotp')}
                </Button>
              </FormItem>
            )}
          />
        ) : null}
        <Button type="submit" size="lg" className="w-full" disabled={isSubmitting}>
          {isSubmitting
            ? t('login.submitting')
            : mfaStep
              ? t('login.mfaSubmit')
              : t('login.submit')}
        </Button>
        <p className="text-center text-body text-secondary">
          {t('login.noAccount')}{' '}
          <Link href="/register" className="font-semibold text-brand-600 hover:underline">
            {t('login.registerLink')}
          </Link>
        </p>
      </form>
    </Form>
  );
}
