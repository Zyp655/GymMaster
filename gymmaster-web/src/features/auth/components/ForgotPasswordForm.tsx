'use client';

import { useMemo, useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import Link from 'next/link';
import { useTranslations } from 'next-intl';
import { CheckCircleIcon, EnvelopeIcon } from '@heroicons/react/24/outline';
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
import { useToastApiError } from '@/hooks/use-api-error';
import { looseT } from '@/lib/utils';
import { useForgotPasswordMutation } from '../queries';
import { createForgotPasswordSchema, type ForgotPasswordFormValues } from '../schema';

export function ForgotPasswordForm() {
  const t = useTranslations('auth.forgotPassword');
  const tv = useTranslations('common.validation');
  const toastError = useToastApiError();
  const mutation = useForgotPasswordMutation();
  const [sentEmail, setSentEmail] = useState<string | null>(null);

  const schema = useMemo(() => createForgotPasswordSchema(looseT(tv)), [tv]);
  const form = useForm<ForgotPasswordFormValues>({
    resolver: zodResolver(schema),
    defaultValues: { email: '' },
  });

  const onSubmit = form.handleSubmit(async values => {
    try {
      await mutation.mutateAsync({ email: values.email, portal: 'WEB' });
      setSentEmail(values.email);
    } catch (error) {
      toastError(error);
    }
  });

  if (sentEmail) {
    return (
      <div className="grid gap-6">
        <div className="grid gap-3 rounded-2xl border border-line bg-brand-25 p-5">
          <div className="flex size-12 items-center justify-center rounded-2xl bg-brand-50 text-brand-600">
            <CheckCircleIcon className="size-6" />
          </div>
          <div className="grid gap-1.5">
            <h2 className="text-h3 font-semibold text-primary">{t('sentTitle')}</h2>
            <p className="text-body text-secondary">{t('sentDescription', { email: sentEmail })}</p>
          </div>
        </div>
        <div className="grid gap-3">
          <Button asChild size="lg" className="w-full">
            <Link href="/login">{t('backToLogin')}</Link>
          </Button>
          <Button
            type="button"
            variant="ghost"
            className="w-full"
            onClick={() => {
              setSentEmail(null);
              form.reset();
            }}
          >
            {t('resend')}
          </Button>
        </div>
      </div>
    );
  }

  return (
    <Form {...form}>
      <form onSubmit={onSubmit} className="grid gap-5" noValidate>
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
                  autoFocus
                  placeholder={t('emailPlaceholder')}
                  className="h-12"
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <Button type="submit" size="lg" className="w-full" disabled={mutation.isPending}>
          <EnvelopeIcon />
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
