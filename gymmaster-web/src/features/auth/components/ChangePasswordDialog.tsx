'use client';

import { useMemo } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { signOut } from 'next-auth/react';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
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
import { ApiCode } from '@/types/api';
import { useChangePassword } from '../queries';
import { createChangePasswordSchema, type ChangePasswordFormValues } from '../schema';
import { PasswordInput } from './PasswordInput';

export function ChangePasswordDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      {open ? <ChangePasswordContent onDone={() => onOpenChange(false)} /> : null}
    </Dialog>
  );
}

function ChangePasswordContent({ onDone }: { onDone: () => void }) {
  const t = useTranslations('auth.changePassword');
  const ta = useTranslations('auth');
  const tc = useTranslations('common');
  const tv = useTranslations('common.validation');
  const toastError = useToastApiError();
  const mutation = useChangePassword();

  const schema = useMemo(() => createChangePasswordSchema(looseT(tv), looseT(ta)), [tv, ta]);
  const form = useForm<ChangePasswordFormValues>({
    resolver: zodResolver(schema),
    defaultValues: { old_password: '', new_password: '', confirm_password: '' },
    mode: 'onTouched',
  });

  const onSubmit = form.handleSubmit(async values => {
    try {
      await mutation.mutateAsync({
        old_password: values.old_password,
        new_password: values.new_password,
      });
      toast.success(t('success'));
      onDone();
      // Backend đã thu hồi mọi refresh token → đăng nhập lại bằng mật khẩu mới.
      await signOut({ callbackUrl: '/login' });
    } catch (error) {
      if (
        error &&
        typeof error === 'object' &&
        'code' in error &&
        (error.code === ApiCode.INVALID_CREDENTIALS || error.code === ApiCode.BAD_REQUEST)
      ) {
        form.setError('old_password', { message: t('wrongOld') });
        return;
      }
      toastError(error);
    }
  });

  const fields = [
    { name: 'old_password', label: t('oldPassword'), autoComplete: 'current-password' },
    { name: 'new_password', label: t('newPassword'), autoComplete: 'new-password' },
    { name: 'confirm_password', label: t('confirmPassword'), autoComplete: 'new-password' },
  ] as const;

  return (
    <DialogContent className="max-w-md" closeLabel={tc('actions.close')}>
      <DialogHeader>
        <DialogTitle>{t('title')}</DialogTitle>
        <DialogDescription>{t('description')}</DialogDescription>
      </DialogHeader>
      <Form {...form}>
        <form onSubmit={onSubmit} className="grid gap-4" noValidate>
          {fields.map(f => (
            <FormField
              key={f.name}
              control={form.control}
              name={f.name}
              render={({ field }) => (
                <FormItem>
                  <FormLabel required>{f.label}</FormLabel>
                  <FormControl>
                    <PasswordInput {...field} autoComplete={f.autoComplete} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          ))}
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onDone} disabled={mutation.isPending}>
              {tc('actions.cancel')}
            </Button>
            <Button type="submit" disabled={mutation.isPending}>
              {mutation.isPending ? tc('actions.saving') : t('submit')}
            </Button>
          </DialogFooter>
        </form>
      </Form>
    </DialogContent>
  );
}
