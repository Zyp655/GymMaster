'use client';

import { useEffect, useMemo } from 'react';
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
import { createLoginSchema, type LoginFormValues } from '../schema';
import { PasswordInput } from './PasswordInput';

const DEFAULT_REDIRECT = '/me';

/** Chỉ cho phép chuyển hướng nội bộ để tránh open-redirect. */
function safeReturnUrl(value: string | null) {
  if (!value || !value.startsWith('/') || value.startsWith('//')) return DEFAULT_REDIRECT;
  return value;
}

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
    defaultValues: { email: '', password: '' },
  });

  const onSubmit = form.handleSubmit(async values => {
    const result = await signIn('credentials', { ...values, redirect: false });
    if (!result || result.error) {
      const code = Number(result?.error);
      const message =
        code === ApiCode.INVALID_CREDENTIALS
          ? t('errors.invalidCredentials')
          : code === ApiCode.ACCOUNT_LOCKED
            ? t('errors.accountLocked')
            : code === ApiCode.TOO_MANY_ATTEMPTS
              ? t('errors.tooManyAttempts')
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
        <Button type="submit" size="lg" className="w-full" disabled={isSubmitting}>
          {isSubmitting ? t('login.submitting') : t('login.submit')}
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
