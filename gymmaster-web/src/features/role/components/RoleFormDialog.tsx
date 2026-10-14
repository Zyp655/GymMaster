'use client';

import { useMemo } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { InformationCircleIcon } from '@heroicons/react/24/outline';
import { LoadError } from '@/components/common/LoadError';
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
import { Skeleton } from '@/components/ui/skeleton';
import { Textarea } from '@/components/ui/textarea';
import { useToastApiError } from '@/hooks/use-api-error';
import { looseT } from '@/lib/utils';
import { ApiCode } from '@/types/api';
import { useCreateRole, usePermissionsQuery, useUpdateRole } from '../queries';
import { createRoleSchema, type RoleFormInput, type RoleFormValues } from '../schema';
import type { Role } from '../types';
import { PermissionChecklist } from './PermissionChecklist';

const OWNER = 'OWNER';

export function RoleFormDialog({
  open,
  onOpenChange,
  role,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  role?: Role | null;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      {open ? (
        <RoleFormContent key={role?.id ?? 'new'} role={role} onDone={() => onOpenChange(false)} />
      ) : null}
    </Dialog>
  );
}

function RoleFormContent({ role, onDone }: { role?: Role | null; onDone: () => void }) {
  const t = useTranslations('role');
  const tc = useTranslations('common');
  const tv = useTranslations('common.validation');
  const toastError = useToastApiError();
  const isEdit = !!role;
  const isOwner = role?.code === OWNER;

  const permissionsQuery = usePermissionsQuery();
  const schema = useMemo(() => createRoleSchema(looseT(tv), looseT(t)), [tv, t]);
  const form = useForm<RoleFormInput, unknown, RoleFormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      code: role?.code ?? '',
      name: role?.name ?? '',
      description: role?.description ?? '',
      permissions: role?.permissions ?? [],
    },
    mode: 'onTouched',
  });

  const createMutation = useCreateRole();
  const updateMutation = useUpdateRole();
  const isPending = createMutation.isPending || updateMutation.isPending;

  const onSubmit = form.handleSubmit(async values => {
    try {
      if (isEdit && role) {
        const updated = await updateMutation.mutateAsync({
          id: role.id,
          payload: {
            name: values.name,
            description: values.description,
            permissions: values.permissions,
          },
        });
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
    <DialogContent className="max-w-3xl" closeLabel={tc('actions.close')}>
      <DialogHeader>
        <DialogTitle>{isEdit ? t('edit') : t('create')}</DialogTitle>
        <DialogDescription>
          {isEdit ? t('editDescription', { code: role.code }) : t('createDescription')}
        </DialogDescription>
      </DialogHeader>

      <Form {...form}>
        <form onSubmit={onSubmit} className="grid gap-5" noValidate>
          {role?.is_system ? (
            <div className="flex items-start gap-2 rounded-xl bg-info-50 p-3 text-body text-info">
              <InformationCircleIcon className="size-5 shrink-0" />
              <span>{isOwner ? t('ownerHint') : t('systemHint')}</span>
            </div>
          ) : null}

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
                      disabled={isEdit}
                      placeholder={t('placeholders.code')}
                      autoComplete="off"
                      onChange={e => field.onChange(e.target.value.toUpperCase())}
                    />
                  </FormControl>
                  {!isEdit ? <FormDescription>{t('codeHint')}</FormDescription> : null}
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
              name="description"
              render={({ field }) => (
                <FormItem className="sm:col-span-2">
                  <FormLabel>{t('fields.description')}</FormLabel>
                  <FormControl>
                    <Textarea {...field} rows={2} placeholder={t('placeholders.description')} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>

          {!isOwner ? (
            <FormField
              control={form.control}
              name="permissions"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('fields.permissions')}</FormLabel>
                  {permissionsQuery.isPending ? (
                    <div className="grid gap-3">
                      <Skeleton className="h-24 w-full rounded-xl" />
                      <Skeleton className="h-24 w-full rounded-xl" />
                    </div>
                  ) : permissionsQuery.error ? (
                    <LoadError
                      error={permissionsQuery.error}
                      onRetry={() => void permissionsQuery.refetch()}
                      className="py-6"
                    />
                  ) : (
                    <PermissionChecklist
                      permissions={permissionsQuery.data ?? []}
                      value={field.value}
                      onChange={field.onChange}
                      disabled={isPending}
                    />
                  )}
                  <FormMessage />
                </FormItem>
              )}
            />
          ) : null}

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
