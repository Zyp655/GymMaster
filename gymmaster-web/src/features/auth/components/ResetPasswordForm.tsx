'use client';

import { useMemo } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { ExclamationTriangleIcon, KeyIcon } from '@heroicons/react/24/outline';
import { Button } from '@/components/ui/button';
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form';
import { useToastApiError } from '@/hooks/use-api-error';
import { looseT } from '@/lib/utils';
import { useResetPasswordMutation } from '../queries';
import { createResetPasswordSchema, type ResetPasswordFormValues } from '../schema';
import { PasswordInput } from './PasswordInput';

export function ResetPasswordForm() {
  const t = useTranslations('auth.resetPassword');
  const ta = useTranslations('auth');
  const tv = useTranslations('common.validation');
  const router = useRouter();
  const searchParams = useSearchParams();
  const token = (searchParams.get('token') ?? '').trim();
  const toastError = useToastApiError();
  const mutation = useResetPasswordMutation();

  const schema = useMemo(() => createResetPasswordSchema(looseT(tv), looseT(ta)), [tv, ta]);
  const form = useForm<ResetPasswordFormValues>({
    resolver: zodResolver(schema),
    defaultValues: { new_password: '', confirm_password: '' },
  });

  if (!token) {
    return (
      <div className="grid gap-6">
        <div className="grid gap-3 rounded-2xl border border-line bg-warning-50 p-5 text-warning">
          <div className="flex size-12 items-center justify-center rounded-2xl bg-surface text-warning">
            <ExclamationTriangleIcon className="size-6" />
          </div>
          <div className="grid gap-1.5">
            <h2 className="text-h3 font-semibold text-primary">{t('missingTokenTitle')}</h2>
            <p className="text-body text-secondary">{t('missingTokenDescription')}</p>
          </div>
        </div>
        <div className="grid gap-3">
          <Button asChild size="lg" className="w-full">
            <Link href="/forgot-password">{t('requestAgain')}</Link>
          </Button>
          <Button asChild variant="outline" className="w-full">
            <Link href="/login">{t('backToLogin')}</Link>
          </Button>
        </div>
      </div>
    );
  }

  const onSubmit = form.handleSubmit(async values => {
    try {
      await mutation.mutateAsync({
        token,
        new_password: values.new_password,
      });
      toast.success(t('success'));
      router.replace('/login');
    } catch (error) {
      toastError(error);
    }
  });

  return (
    <Form {...form}>
      <form onSubmit={onSubmit} className="grid gap-5" noValidate>
        <FormField
          control={form.control}
          name="new_password"
          render={({ field }) => (
            <FormItem>
              <FormLabel required>{t('newPassword')}</FormLabel>
              <FormControl>
                <PasswordInput
                  {...field}
                  autoComplete="new-password"
                  autoFocus
                  placeholder={t('newPasswordPlaceholder')}
                  className="h-12"
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
                  className="h-12"
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <Button type="submit" size="lg" className="w-full" disabled={mutation.isPending}>
          <KeyIcon />
          {mutation.isPending ? t('submitting') : t('submit')}
        </Button>
        <p className="text-center text-body text-secondary">
          <Link href="/login" className="font-semibold text-brand-600 hover:underline">
            {t('backToLogin')}
          </Link>
        </p>
      </form>
    </Form>
  );
}
