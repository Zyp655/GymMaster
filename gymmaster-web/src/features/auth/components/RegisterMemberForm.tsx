'use client';

import { useEffect, useMemo, useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { signIn } from 'next-auth/react';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { ShieldCheckIcon, UserPlusIcon } from '@heroicons/react/24/outline';
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
import { useToastApiError } from '@/hooks/use-api-error';
import { looseT } from '@/lib/utils';
import {
  usePublicBranchesQuery,
  useRegisterMemberMutation,
  useVerifyMemberOtpMutation,
} from '../queries';
import {
  createRegisterMemberSchema,
  createVerifyOtpSchema,
  type RegisterMemberFormValues,
  type VerifyOtpFormValues,
} from '../schema';
import { PasswordInput } from './PasswordInput';

interface PendingOtpState {
  phone: string;
  email: string;
  password: string;
}

export function RegisterMemberForm() {
  const t = useTranslations('auth.register');
  const ta = useTranslations('auth');
  const tv = useTranslations('common.validation');
  const router = useRouter();
  const toastError = useToastApiError();

  const branchesQuery = usePublicBranchesQuery();
  const registerMutation = useRegisterMemberMutation();
  const verifyOtpMutation = useVerifyMemberOtpMutation();

  const [pendingOtp, setPendingOtp] = useState<PendingOtpState | null>(null);

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

  const onSubmitRegister = form.handleSubmit(async values => {
    try {
      const res = await registerMutation.mutateAsync({
        full_name: values.full_name,
        phone: values.phone,
        email: values.email,
        password: values.password,
        branch_id: values.branch_id,
      });

      if (res.requires_otp) {
        if (res.message) toast.info(res.message);
        setPendingOtp({
          phone: values.phone,
          email: values.email,
          password: values.password,
        });
        otpForm.reset({ otp: '' });
        return;
      }

      toast.success(res.message || t('success'));
      await signInOrRedirect(values.email, values.password);
    } catch (error) {
      toastError(error);
    }
  });

  const onSubmitOtp = otpForm.handleSubmit(async values => {
    if (!pendingOtp) return;
    try {
      await verifyOtpMutation.mutateAsync({
        phone: pendingOtp.phone,
        email: pendingOtp.email,
        otp: values.otp,
      });
      toast.success(t('otpSuccess'));
      await signInOrRedirect(pendingOtp.email, pendingOtp.password);
    } catch (error) {
      toastError(error);
    }
  });

  if (pendingOtp) {
    return (
      <div className="grid gap-6">
        <div className="grid gap-2 rounded-2xl border border-line bg-brand-25 p-4">
          <div className="flex items-center gap-2 text-brand-700">
            <ShieldCheckIcon className="size-5 shrink-0" />
            <h2 className="text-h4 font-semibold">{t('otpTitle')}</h2>
          </div>
          <p className="text-body text-secondary">
            {t('otpSubtitle', { phone: pendingOtp.phone, email: pendingOtp.email })}
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
            <Button type="submit" size="lg" className="w-full" disabled={verifyOtpMutation.isPending}>
              <ShieldCheckIcon />
              {verifyOtpMutation.isPending ? t('otpSubmitting') : t('otpSubmit')}
            </Button>
            <Button
              type="button"
              variant="ghost"
              className="w-full"
              onClick={() => setPendingOtp(null)}
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
                  value={field.value ? String(field.value) : undefined}
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
          disabled={registerMutation.isPending}
        >
          <UserPlusIcon />
          {registerMutation.isPending ? t('submitting') : t('submit')}
        </Button>

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
