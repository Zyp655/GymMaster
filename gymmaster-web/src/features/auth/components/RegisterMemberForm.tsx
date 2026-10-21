'use client';

import { useEffect, useMemo, useState } from 'react';
import { useForm, useWatch } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { signIn } from 'next-auth/react';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { ArrowPathIcon, ShieldCheckIcon, UserPlusIcon } from '@heroicons/react/24/outline';
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import { useApiErrorMessage, useToastApiError } from '@/hooks/use-api-error';
import { looseT } from '@/lib/utils';
import {
  usePublicBranchesQuery,
  useRegisterMemberMutation,
  useVerifyMemberOtpMutation,
} from '../queries';
import {
  buildVerifyMemberOtpPayload,
  classifyVerifyOtpError,
  formatCountdown,
  isSamePhone,
  isThrottleError,
  OTP_RESEND_COOLDOWN_SECONDS,
  parseRetryAfterSeconds,
} from '../register-otp';
import {
  createRegisterMemberSchema,
  createVerifyOtpSchema,
  type RegisterMemberFormValues,
  type VerifyOtpFormValues,
} from '../schema';
import type { RegisterMemberResponse } from '../types';
import { PasswordInput } from './PasswordInput';

type RegisterRequest = Pick<
  RegisterMemberFormValues,
  'full_name' | 'phone' | 'email' | 'password' | 'branch_id'
>;

/**
 * SĐT trùng hồ sơ hội viên có sẵn → backend gửi OTP tới email TRONG HỒ SƠ (`masked_email`).
 * Giữ lại dữ liệu đã gửi để "Gửi lại mã" và gửi mật khẩu ở bước verify (backend không lưu mật khẩu ở bước 1).
 */
interface OtpSession {
  request: RegisterRequest;
  maskedEmail?: string;
  serverMessage?: string;
}

export function RegisterMemberForm() {
  const t = useTranslations('auth.register');
  const ta = useTranslations('auth');
  const tv = useTranslations('common.validation');
  const router = useRouter();
  const toastError = useToastApiError();
  const getErrorMessage = useApiErrorMessage();

  const branchesQuery = usePublicBranchesQuery();
  const registerMutation = useRegisterMemberMutation();
  const verifyOtpMutation = useVerifyMemberOtpMutation();

  const [otpSession, setOtpSession] = useState<OtpSession | null>(null);
  /** Đang ở màn nhập OTP (có thể quay lại form mà không mất phiên OTP). */
  const [showOtp, setShowOtp] = useState(false);
  /** Sai OTP quá 5 lần (40105) → mã bị vô hiệu, phải gửi lại mã mới. */
  const [otpLocked, setOtpLocked] = useState(false);
  /** Số giây còn phải chờ trước khi được "Gửi lại mã" (backend chặn 60 giây giữa 2 lần gửi). */
  const [resendSeconds, setResendSeconds] = useState(0);

  useEffect(() => {
    if (resendSeconds <= 0) return;
    const timer = setTimeout(() => setResendSeconds(s => Math.max(0, s - 1)), 1000);
    return () => clearTimeout(timer);
  }, [resendSeconds]);

  const startCooldown = (seconds: number) => setResendSeconds(Math.max(0, Math.ceil(seconds)));

  const registerSchema = useMemo(
    () => createRegisterMemberSchema(looseT(tv), looseT(ta)),
    [tv, ta],
  );
  const otpSchema = useMemo(() => createVerifyOtpSchema(looseT(tv)), [tv]);

  const form = useForm<RegisterMemberFormValues>({
    resolver: zodResolver(registerSchema),
    defaultValues: {
      full_name: '',
      phone: '',
      email: '',
      branch_id: 0,
      password: '',
      confirm_password: '',
    },
  });

  const otpForm = useForm<VerifyOtpFormValues>({
    resolver: zodResolver(otpSchema),
    defaultValues: { otp: '' },
  });

  const watchedPhone = useWatch({ control: form.control, name: 'phone' });
  const canReturnToOtp =
    !showOtp && otpSession !== null && isSamePhone(watchedPhone ?? '', otpSession.request.phone);

  useEffect(() => {
    if (branchesQuery.data && branchesQuery.data.length > 0 && !form.getValues('branch_id')) {
      form.setValue('branch_id', branchesQuery.data[0].id);
    }
  }, [branchesQuery.data, form]);

  const signInOrRedirect = async (email: string, password: string) => {
    try {
      const result = await signIn('credentials', { email, password, redirect: false });
      if (result && !result.error) {
        router.replace('/me');
        router.refresh();
        return;
      }
    } catch {
      // Fallback sang /login
    }
    router.replace('/login');
  };

  /** Kết quả register-member: liên kết hồ sơ → màn OTP; hội viên mới → đăng nhập luôn. */
  const handleRegisterResult = async (
    request: RegisterRequest,
    res: RegisterMemberResponse,
    resent: boolean,
  ) => {
    if (res.requires_otp) {
      setOtpSession({ request, maskedEmail: res.masked_email, serverMessage: res.message });
      setOtpLocked(false);
      setShowOtp(true);
      otpForm.reset({ otp: '' });
      startCooldown(OTP_RESEND_COOLDOWN_SECONDS);
      if (resent) {
        toast.success(
          res.masked_email ? t('otpResent', { email: res.masked_email }) : t('otpResentNoEmail'),
        );
      }
      return;
    }

    toast.success(res.message || t('success'));
    await signInOrRedirect(request.email, request.password);
  };

  const onSubmitRegister = form.handleSubmit(async values => {
    const request: RegisterRequest = {
      full_name: values.full_name,
      phone: values.phone,
      email: values.email,
      password: values.password,
      branch_id: values.branch_id,
    };
    try {
      const res = await registerMutation.mutateAsync(request);
      await handleRegisterResult(request, res, false);
    } catch (error) {
      // OTP cho chính SĐT này vừa được gửi (429 "đợi 60 giây") → quay lại màn OTP với dữ liệu mới nhập.
      if (
        isThrottleError(error) &&
        otpSession &&
        isSamePhone(otpSession.request.phone, values.phone)
      ) {
        setOtpSession({ ...otpSession, request });
        startCooldown(parseRetryAfterSeconds(error instanceof Error ? error.message : undefined));
        setShowOtp(true);
        toast.info(getErrorMessage(error));
        return;
      }
      toastError(error);
    }
  });

  /** Quay lại màn OTP: mật khẩu (có thể vừa sửa) phải hợp lệ vì sẽ được gửi ở bước verify. */
  const onReturnToOtp = async () => {
    if (!otpSession) return;
    const valid = await form.trigger(['password', 'confirm_password']);
    if (!valid) return;
    const { password } = form.getValues();
    setOtpSession({ ...otpSession, request: { ...otpSession.request, password } });
    setShowOtp(true);
  };

  const onResendOtp = async () => {
    if (!otpSession || resendSeconds > 0 || registerMutation.isPending) return;
    try {
      const res = await registerMutation.mutateAsync(otpSession.request);
      await handleRegisterResult(otpSession.request, res, true);
    } catch (error) {
      if (isThrottleError(error)) {
        startCooldown(parseRetryAfterSeconds(error instanceof Error ? error.message : undefined));
      }
      toastError(error);
    }
  };

  const onSubmitOtp = otpForm.handleSubmit(async values => {
    if (!otpSession) return;
    const { password } = otpSession.request;
    try {
      const res = await verifyOtpMutation.mutateAsync(
        buildVerifyMemberOtpPayload(otpSession.request, values.otp),
      );
      toast.success(t('otpSuccess'));
      // Tài khoản dùng email trong hồ sơ (không phải email đã nhập) → đăng nhập bằng user.email trả về.
      await signInOrRedirect(res.user.email, password);
    } catch (error) {
      const kind = classifyVerifyOtpError(error);
      if (kind === 'invalid') {
        otpForm.setError('otp', { message: t('otpInvalid') }, { shouldFocus: true });
        toast.error(t('otpInvalid'));
        return;
      }
      if (kind === 'locked') {
        // Toast: câu của server; dưới ô nhập: hướng dẫn bấm "Gửi lại mã" (nút nổi bật, ô xác nhận bị khóa).
        setOtpLocked(true);
        otpForm.setError('otp', { message: t('otpLocked') });
        toast.error(getErrorMessage(error));
        return;
      }
      toastError(error);
    }
  });

  if (otpSession && showOtp) {
    return (
      <div className="grid gap-6">
        <div className="grid gap-2 rounded-2xl border border-line bg-brand-25 p-4">
          <div className="flex items-center gap-2 text-brand-700">
            <ShieldCheckIcon className="size-5 shrink-0" />
            <h2 className="text-h4 font-semibold">{t('otpTitle')}</h2>
          </div>
          <p className="text-body text-secondary">
            {otpSession.maskedEmail
              ? t('otpSubtitle', {
                  phone: otpSession.request.phone,
                  email: otpSession.maskedEmail,
                })
              : otpSession.serverMessage ||
                t('otpSubtitleNoEmail', { phone: otpSession.request.phone })}
          </p>
        </div>

        <Form {...otpForm}>
          <form onSubmit={onSubmitOtp} className="grid gap-5" noValidate>
            <FormField
              control={otpForm.control}
              name="otp"
              render={({ field }) => (
                <FormItem>
                  <FormLabel required>{t('otpLabel')}</FormLabel>
                  <FormControl>
                    <Input
                      {...field}
                      inputMode="numeric"
                      autoComplete="one-time-code"
                      maxLength={6}
                      autoFocus
                      placeholder={t('otpPlaceholder')}
                      className="h-12 text-center font-mono text-h3 tracking-widest"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <Button
              type="submit"
              size="lg"
              className="w-full"
              disabled={verifyOtpMutation.isPending || otpForm.formState.isSubmitting || otpLocked}
            >
              <ShieldCheckIcon />
              {verifyOtpMutation.isPending ? t('otpSubmitting') : t('otpSubmit')}
            </Button>
            <Button
              type="button"
              variant={otpLocked ? 'primary' : 'outline'}
              className="w-full"
              onClick={onResendOtp}
              disabled={resendSeconds > 0 || registerMutation.isPending}
            >
              <ArrowPathIcon />
              {registerMutation.isPending
                ? t('otpResending')
                : resendSeconds > 0
                  ? t('otpResendIn', { time: formatCountdown(resendSeconds) })
                  : t('otpResend')}
            </Button>
            <Button
              type="button"
              variant="ghost"
              className="w-full"
              onClick={() => setShowOtp(false)}
            >
              {t('otpBack')}
            </Button>
          </form>
        </Form>
      </div>
    );
  }

  return (
    <Form {...form}>
      <form onSubmit={onSubmitRegister} className="grid gap-4" noValidate>
        <FormField
          control={form.control}
          name="full_name"
          render={({ field }) => (
            <FormItem>
              <FormLabel required>{t('fullName')}</FormLabel>
              <FormControl>
                <Input
                  {...field}
                  autoComplete="name"
                  autoFocus
                  placeholder={t('fullNamePlaceholder')}
                  className="h-11"
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />

        <div className="grid gap-4 sm:grid-cols-2">
          <FormField
            control={form.control}
            name="phone"
            render={({ field }) => (
              <FormItem>
                <FormLabel required>{t('phone')}</FormLabel>
                <FormControl>
                  <Input
                    {...field}
                    type="tel"
                    autoComplete="tel"
                    placeholder={t('phonePlaceholder')}
                    className="h-11"
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name="email"
            render={({ field }) => (
              <FormItem>
                <FormLabel required>{t('email')}</FormLabel>
                <FormControl>
                  <Input
                    {...field}
                    type="email"
                    autoComplete="email"
                    placeholder={t('emailPlaceholder')}
                    className="h-11"
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
        </div>

        <FormField
          control={form.control}
          name="branch_id"
          render={({ field }) => (
            <FormItem>
              <FormLabel required>{t('branch')}</FormLabel>
              {branchesQuery.isLoading ? (
                <Skeleton className="h-11 w-full rounded-xl" />
              ) : (
                <Select
                  value={field.value ? String(field.value) : ''}
                  onValueChange={v => field.onChange(Number(v))}
                >
                  <FormControl>
                    <SelectTrigger className="h-11">
                      <SelectValue placeholder={t('branchPlaceholder')} />
                    </SelectTrigger>
                  </FormControl>
                  <SelectContent>
                    {(branchesQuery.data ?? []).map(branch => (
                      <SelectItem key={branch.id} value={String(branch.id)}>
                        {branch.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
              <FormMessage />
            </FormItem>
          )}
        />

        <div className="grid gap-4 sm:grid-cols-2">
          <FormField
            control={form.control}
            name="password"
            render={({ field }) => (
              <FormItem>
                <FormLabel required>{t('password')}</FormLabel>
                <FormControl>
                  <PasswordInput
                    {...field}
                    autoComplete="new-password"
                    placeholder={t('passwordPlaceholder')}
                    className="h-11"
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name="confirm_password"
            render={({ field }) => (
              <FormItem>
                <FormLabel required>{t('confirmPassword')}</FormLabel>
                <FormControl>
                  <PasswordInput
                    {...field}
                    autoComplete="new-password"
                    placeholder={t('confirmPasswordPlaceholder')}
                    className="h-11"
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
        </div>

        <Button
          type="submit"
          size="lg"
          className="mt-1 w-full"
          disabled={registerMutation.isPending || form.formState.isSubmitting}
        >
          <UserPlusIcon />
          {registerMutation.isPending ? t('submitting') : t('submit')}
        </Button>

        {canReturnToOtp ? (
          <Button type="button" variant="soft" className="w-full" onClick={onReturnToOtp}>
            <ShieldCheckIcon />
            {t('otpReturn')}
          </Button>
        ) : null}

        <p className="text-center text-body text-secondary">
          {t('alreadyHaveAccount')}{' '}
          <Link href="/login" className="font-semibold text-brand-600 hover:underline">
            {t('loginLink')}
          </Link>
        </p>
      </form>
    </Form>
  );
}
