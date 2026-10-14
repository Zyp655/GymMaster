'use client';

import { useMemo, useState } from 'react';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import {
  EllipsisHorizontalIcon,
  LockClosedIcon,
  PencilSquareIcon,
  PlusIcon,
  ShieldCheckIcon,
  TrashIcon,
} from '@heroicons/react/24/outline';
import { ConfirmDialog } from '@/components/common/ConfirmDialog';
import { DataTable, type DataTableColumn } from '@/components/common/DataTable';
import { EmptyState } from '@/components/common/EmptyState';
import { PageHeader } from '@/components/common/PageHeader';
import { PermissionGate } from '@/components/common/PermissionGate';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { useToastApiError } from '@/hooks/use-api-error';
import { useAuth } from '@/hooks/use-auth';
import { Perm } from '@/lib/permissions';
import { ApiCode } from '@/types/api';
import { useDeleteRole, useRolesQuery } from '../queries';
import type { Role } from '../types';
import { RoleFormDialog } from './RoleFormDialog';

/** API roles trả mảng (không phân trang) → phân trang tại client cho bảng. */
export function RolesView() {
  const t = useTranslations('role');
  const tc = useTranslations('common');
  const { can } = useAuth();
  const toastError = useToastApiError();
  const query = useRolesQuery();
  const deleteMutation = useDeleteRole();

  const [page, setPage] = useState(1);
  const [limit, setLimit] = useState(10);
  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<Role | null>(null);
  const [deleting, setDeleting] = useState<Role | null>(null);

  const canUpdate = can(Perm.ROLE_UPDATE);
  const canDelete = can(Perm.ROLE_DELETE);

  const roles = query.data;
  const pageItems = useMemo(
    () => roles?.slice((page - 1) * limit, page * limit),
    [roles, page, limit],
  );

  const columns = useMemo<DataTableColumn<Role>[]>(
    () => [
      {
        accessorKey: 'code',
        header: t('fields.code'),
        cell: ({ row }) => (
          <div className="flex items-center gap-2">
            <span className="font-medium text-primary">{row.original.code}</span>
            {row.original.is_system ? (
              <Badge variant="info">
                <LockClosedIcon />
                {t('systemBadge')}
              </Badge>
            ) : null}
          </div>
        ),
      },
      {
        accessorKey: 'name',
        header: t('fields.name'),
        cell: ({ row }) => <span className="font-medium text-primary">{row.original.name}</span>,
      },
      {
        accessorKey: 'description',
        header: t('fields.description'),
        cell: ({ row }) => (
          <span className="line-clamp-2 min-w-48 text-secondary">{row.original.description}</span>
        ),
      },
      {
        id: 'permissions',
        header: t('fields.permissions'),
        cell: ({ row }) =>
          row.original.code === 'OWNER' ? (
            <Badge variant="brand">{t('allPermissions')}</Badge>
          ) : (
            <Badge variant="neutral">
              {t('permissionCount', { count: row.original.permissions.length })}
            </Badge>
          ),
      },
      {
        id: 'actions',
        header: () => <span className="sr-only">{tc('actions.actions')}</span>,
        meta: { className: 'w-14 text-right' },
        cell: ({ row }) => {
          const role = row.original;
          const showDelete = canDelete && !role.is_system;
          if (!canUpdate && !showDelete) return null;
          return (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="icon-sm" aria-label={tc('actions.openMenu')}>
                  <EllipsisHorizontalIcon />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                {canUpdate ? (
                  <DropdownMenuItem
                    onSelect={() => {
                      setEditing(role);
                      setFormOpen(true);
                    }}
                  >
                    <PencilSquareIcon />
                    {tc('actions.edit')}
                  </DropdownMenuItem>
                ) : null}
                {showDelete ? (
                  <DropdownMenuItem variant="danger" onSelect={() => setDeleting(role)}>
                    <TrashIcon />
                    {tc('actions.delete')}
                  </DropdownMenuItem>
                ) : null}
              </DropdownMenuContent>
            </DropdownMenu>
          );
        },
      },
    ],
    [t, tc, canUpdate, canDelete],
  );

  const createButton = (
    <PermissionGate perm={Perm.ROLE_CREATE}>
      <Button
        onClick={() => {
          setEditing(null);
          setFormOpen(true);
        }}
      >
        <PlusIcon />
        {t('create')}
      </Button>
    </PermissionGate>
  );

  const confirmDelete = async () => {
    if (!deleting) return;
    try {
      await deleteMutation.mutateAsync(deleting.id);
      toast.success(t('toast.deleted', { name: deleting.name }));
      setDeleting(null);
      if (pageItems?.length === 1 && page > 1) setPage(page - 1);
    } catch (error) {
      toastError(error, { [ApiCode.CONFLICT]: t('toast.deleteConflict') });
    }
  };

  return (
    <div className="grid gap-6">
      <PageHeader title={t('title')} description={t('description')} actions={createButton} />

      <DataTable
        columns={columns}
        data={pageItems}
        total={roles?.length ?? 0}
        page={page}
        limit={limit}
        onPageChange={setPage}
        onLimitChange={next => {
          setLimit(next);
          setPage(1);
        }}
        isLoading={query.isPending}
        isFetching={query.isFetching}
        error={query.error}
        onRetry={() => void query.refetch()}
        getRowId={row => String(row.id)}
        empty={
          <EmptyState
            icon={ShieldCheckIcon}
            title={t('empty.title')}
            description={t('empty.description')}
            action={createButton}
          />
        }
      />

      <RoleFormDialog open={formOpen} onOpenChange={setFormOpen} role={editing} />

      <ConfirmDialog
        open={!!deleting}
        onOpenChange={open => !open && setDeleting(null)}
        description={tc('confirm.deleteDescription', { name: deleting?.name ?? '' })}
        loading={deleteMutation.isPending}
        onConfirm={confirmDelete}
      />
    </div>
  );
}
