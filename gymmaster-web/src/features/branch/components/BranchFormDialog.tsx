'use client';

import { useMemo } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
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
  FormDescription,
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
import { useToastApiError } from '@/hooks/use-api-error';
import { looseT } from '@/lib/utils';
import { ApiCode } from '@/types/api';
import { useCreateBranch, useUpdateBranch } from '../queries';
import {
  branchFormDefaults,
  createBranchSchema,
  type BranchFormInput,
  type BranchFormValues,
} from '../schema';
import type { Branch } from '../types';

export function BranchFormDialog({
  open,
  onOpenChange,
  branch,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Có giá trị = chế độ sửa. */
  branch?: Branch | null;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      {open ? (
        <BranchFormContent
          key={branch?.id ?? 'new'}
          branch={branch}
          onDone={() => onOpenChange(false)}
        />
      ) : null}
    </Dialog>
  );
}

function BranchFormContent({ branch, onDone }: { branch?: Branch | null; onDone: () => void }) {
  const t = useTranslations('branch');
  const tc = useTranslations('common');
  const tv = useTranslations('common.validation');
  const toastError = useToastApiError();
  const isEdit = !!branch;

  const schema = useMemo(() => createBranchSchema(looseT(tv), looseT(t)), [tv, t]);
  const form = useForm<BranchFormInput, unknown, BranchFormValues>({
    resolver: zodResolver(schema),
    defaultValues: branch
      ? {
          code: branch.code,
          name: branch.name,
          address: branch.address ?? '',
          phone: branch.phone ?? '',
          email: branch.email ?? '',
          open_time: branch.open_time,
          close_time: branch.close_time,
          status: branch.status,
        }
      : branchFormDefaults,
    mode: 'onTouched',
  });

  const createMutation = useCreateBranch();
  const updateMutation = useUpdateBranch();
  const isPending = createMutation.isPending || updateMutation.isPending;

  const onSubmit = form.handleSubmit(async values => {
    try {
      if (isEdit && branch) {
        const updated = await updateMutation.mutateAsync({ id: branch.id, payload: values });
        toast.success(t('toast.updated', { name: updated.name }));
      } else {
        const created = await createMutation.mutateAsync(values);
        toast.success(t('toast.created', { name: created.name }));
      }
      onDone();
    } catch (error) {
      toastError(error, { [ApiCode.CONFLICT]: t('toast.codeConflict') });
    }
  });

  return (
    <DialogContent className="max-w-2xl" closeLabel={tc('actions.close')}>
      <DialogHeader>
        <DialogTitle>{isEdit ? t('edit') : t('create')}</DialogTitle>
        <DialogDescription>
          {isEdit ? t('editDescription', { code: branch.code }) : t('createDescription')}
        </DialogDescription>
      </DialogHeader>

      <Form {...form}>
        <form onSubmit={onSubmit} className="grid gap-5" noValidate>
          <div className="grid gap-4 sm:grid-cols-2">
            <FormField
              control={form.control}
              name="code"
              render={({ field }) => (
                <FormItem>
                  <FormLabel required>{t('fields.code')}</FormLabel>
                  <FormControl>
                    <Input
                      {...field}
                      placeholder={t('placeholders.code')}
                      autoComplete="off"
                      onChange={e => field.onChange(e.target.value.toUpperCase())}
                    />
                  </FormControl>
                  <FormDescription>{t('codeHint')}</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel required>{t('fields.name')}</FormLabel>
                  <FormControl>
                    <Input {...field} placeholder={t('placeholders.name')} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="address"
              render={({ field }) => (
                <FormItem className="sm:col-span-2">
                  <FormLabel>{t('fields.address')}</FormLabel>
                  <FormControl>
                    <Input {...field} placeholder={t('placeholders.address')} />
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
                  <FormLabel>{t('fields.phone')}</FormLabel>
                  <FormControl>
                    <Input {...field} type="tel" placeholder={t('placeholders.phone')} />
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
                  <FormLabel>{t('fields.email')}</FormLabel>
                  <FormControl>
                    <Input {...field} type="email" placeholder={t('placeholders.email')} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="open_time"
              render={({ field }) => (
                <FormItem>
                  <FormLabel required>{t('fields.openTime')}</FormLabel>
                  <FormControl>
                    <Input {...field} type="time" step={300} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="close_time"
              render={({ field }) => (
                <FormItem>
                  <FormLabel required>{t('fields.closeTime')}</FormLabel>
                  <FormControl>
                    <Input {...field} type="time" step={300} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="status"
              render={({ field }) => (
                <FormItem>
                  <FormLabel required>{t('fields.status')}</FormLabel>
                  <Select value={field.value} onValueChange={field.onChange}>
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      <SelectItem value="ACTIVE">{tc('status.ACTIVE')}</SelectItem>
                      <SelectItem value="INACTIVE">{tc('status.INACTIVE')}</SelectItem>
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={onDone} disabled={isPending}>
              {tc('actions.cancel')}
            </Button>
            <Button type="submit" disabled={isPending}>
              {isPending ? tc('actions.saving') : isEdit ? tc('actions.saveChanges') : t('create')}
            </Button>
          </DialogFooter>
        </form>
      </Form>
    </DialogContent>
  );
}
