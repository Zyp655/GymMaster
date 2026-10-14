'use client';

import { useEffect, useMemo, useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useTranslations } from 'next-intl';
import { z } from 'zod';
import { CheckCircleIcon, SparklesIcon } from '@heroicons/react/24/outline';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
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
import { Textarea } from '@/components/ui/textarea';
import { useToastApiError } from '@/hooks/use-api-error';
import { looseT, type Translate } from '@/lib/utils';
import { createPublicLead } from '../client-api';
import type { PublicBranch, PublicPlan } from '../types';
import { SectionHeading } from './SectionHeading';

const PHONE_REGEX = /^(0|\+84)[0-9]{8,10}$/;

function createTrialLeadSchema(tv: Translate) {
  return z.object({
    full_name: z
      .string()
      .trim()
      .min(1, tv('required'))
      .max(100, tv('maxLength', { max: 100 })),
    phone: z
      .string()
      .trim()
      .min(1, tv('required'))
      .refine(v => PHONE_REGEX.test(v), tv('phone')),
    email: z
      .string()
      .trim()
      .refine(v => !v || z.string().email().safeParse(v).success, tv('email')),
    branch_id: z.number().int().positive(tv('required')),
    interest_plan_id: z.number().int().nonnegative(),
    note: z.string().max(500, tv('maxLength', { max: 500 })),
  });
}

type TrialLeadFormValues = z.infer<ReturnType<typeof createTrialLeadSchema>>;

export function TrialLeadSection({
  branches,
  plans,
}: {
  branches: PublicBranch[];
  plans: PublicPlan[];
}) {
  const t = useTranslations('common.landing.trial');
  const tv = useTranslations('common.validation');
  const toastError = useToastApiError();
  const [submitted, setSubmitted] = useState(false);
  const [referralCode, setReferralCode] = useState<string | null>(null);

  const schema = useMemo(() => createTrialLeadSchema(looseT(tv)), [tv]);
  const form = useForm<TrialLeadFormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      full_name: '',
      phone: '',
      email: '',
      branch_id: branches[0]?.id ?? 0,
      interest_plan_id: 0,
      note: '',
    },
  });

  useEffect(() => {
    if (typeof window === 'undefined') return;
    const params = new URLSearchParams(window.location.search);
    const ref = params.get('ref')?.trim();
    if (ref) {
      setReferralCode(ref);
      const currentNote = form.getValues('note').trim();
      const prefix = t('referralNotePrefix', { code: ref });
      if (!currentNote.includes(ref)) {
        form.setValue('note', currentNote ? `${prefix} ${currentNote}` : prefix);
      }
    }
  }, [form, t]);

  const onSubmit = form.handleSubmit(async values => {
    try {
      const trimmedNote = values.note.trim();
      const refPrefix = referralCode ? t('referralNotePrefix', { code: referralCode }) : '';
      const finalNote =
        referralCode && !trimmedNote.includes(referralCode)
          ? [refPrefix, trimmedNote].filter(Boolean).join(' ')
          : trimmedNote;

      await createPublicLead({
        full_name: values.full_name.trim(),
        phone: values.phone.trim(),
        email: values.email.trim() || undefined,
        branch_id: values.branch_id || undefined,
        interest_plan_id: values.interest_plan_id > 0 ? values.interest_plan_id : undefined,
        note: finalNote || undefined,
      });
      setSubmitted(true);
    } catch (error) {
      toastError(error);
    }
  });

  return (
    <section id="tap-thu" className="scroll-mt-16 border-t border-line bg-canvas">
      <div className="mx-auto grid max-w-6xl gap-10 px-4 py-20 sm:px-6 lg:grid-cols-2 lg:items-center">
        <div className="grid gap-4">
          <SectionHeading
            eyebrow={t('eyebrow')}
            title={t('title')}
            subtitle={t('subtitle')}
          />
          {referralCode ? (
            <div>
              <Badge variant="brand">{t('referralBadge', { code: referralCode })}</Badge>
            </div>
          ) : null}
        </div>

        <Card className="p-6 sm:p-8">
          {submitted ? (
            <div className="grid justify-items-center gap-4 py-6 text-center">
              <div className="flex size-14 items-center justify-center rounded-full bg-success-50 text-success">
                <CheckCircleIcon className="size-8" />
              </div>
              <div className="grid gap-1">
                <h3 className="text-h3 font-semibold text-primary">{t('successTitle')}</h3>
                <p className="text-body text-secondary">{t('successDesc')}</p>
              </div>
              <Button
                variant="outline"
                onClick={() => {
                  form.reset();
                  setSubmitted(false);
                }}
              >
                {t('registerAnother')}
              </Button>
            </div>
          ) : (
            <Form {...form}>
              <form onSubmit={onSubmit} className="grid gap-4" noValidate>
                <div className="grid gap-4 sm:grid-cols-2">
                  <FormField
                    control={form.control}
                    name="full_name"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('fullNameLabel')}</FormLabel>
                        <FormControl>
                          <Input {...field} placeholder={t('fullNamePlaceholder')} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={form.control}
                    name="phone"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('phoneLabel')}</FormLabel>
                        <FormControl>
                          <Input
                            {...field}
                            type="tel"
                            inputMode="tel"
                            placeholder={t('phonePlaceholder')}
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>

                <FormField
                  control={form.control}
                  name="email"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('emailLabel')}</FormLabel>
                      <FormControl>
                        <Input
                          {...field}
                          type="email"
                          inputMode="email"
                          placeholder={t('emailPlaceholder')}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <div className="grid gap-4 sm:grid-cols-2">
                  <FormField
                    control={form.control}
                    name="branch_id"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('branchLabel')}</FormLabel>
                        <Select
                          value={field.value ? String(field.value) : undefined}
                          onValueChange={v => field.onChange(Number(v))}
                        >
                          <FormControl>
                            <SelectTrigger>
                              <SelectValue placeholder={t('branchPlaceholder')} />
                            </SelectTrigger>
                          </FormControl>
                          <SelectContent>
                            {branches.map(branch => (
                              <SelectItem key={branch.id} value={String(branch.id)}>
                                {branch.name}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={form.control}
                    name="interest_plan_id"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('planLabel')}</FormLabel>
                        <Select
                          value={field.value ? String(field.value) : '0'}
                          onValueChange={v => field.onChange(Number(v))}
                        >
                          <FormControl>
                            <SelectTrigger>
                              <SelectValue placeholder={t('planPlaceholder')} />
                            </SelectTrigger>
                          </FormControl>
                          <SelectContent>
                            <SelectItem value="0">{t('planPlaceholder')}</SelectItem>
                            {plans.map(plan => (
                              <SelectItem key={plan.id} value={String(plan.id)}>
                                {plan.name}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>

                <FormField
                  control={form.control}
                  name="note"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('noteLabel')}</FormLabel>
                      <FormControl>
                        <Textarea {...field} placeholder={t('notePlaceholder')} rows={2} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <Button type="submit" disabled={form.formState.isSubmitting}>
                  <SparklesIcon />
                  {form.formState.isSubmitting ? t('submitting') : t('submit')}
                </Button>
              </form>
            </Form>
          )}
        </Card>
      </div>
    </section>
  );
}
