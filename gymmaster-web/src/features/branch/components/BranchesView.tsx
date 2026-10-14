'use client';

import { useEffect, useMemo, useState } from 'react';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import {
  BuildingStorefrontIcon,
  EllipsisHorizontalIcon,
  PencilSquareIcon,
  PlusIcon,
  TrashIcon,
} from '@heroicons/react/24/outline';
import { ConfirmDialog } from '@/components/common/ConfirmDialog';
import { DataTable, type DataTableColumn } from '@/components/common/DataTable';
import { EmptyState } from '@/components/common/EmptyState';
import { PageHeader } from '@/components/common/PageHeader';
import { PermissionGate } from '@/components/common/PermissionGate';
import { SearchInput } from '@/components/common/SearchInput';
import { StatusBadge } from '@/components/common/StatusBadge';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { useToastApiError } from '@/hooks/use-api-error';
import { useAuth } from '@/hooks/use-auth';
import { useListParams } from '@/hooks/use-list-params';
import { Perm } from '@/lib/permissions';
import { ApiCode } from '@/types/api';
import { useBranchesQuery, useDeleteBranch } from '../queries';
import type { Branch } from '../types';
import { BranchFormDialog } from './BranchFormDialog';

const FILTER_KEYS = ['status'] as const;
const ALL = 'ALL';

export function BranchesView() {
  const t = useTranslations('branch');
  const tc = useTranslations('common');
  const { can } = useAuth();
  const toastError = useToastApiError();
  const list = useListParams(FILTER_KEYS);
  const query = useBranchesQuery(list.apiParams);

  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<Branch | null>(null);
  const [deleting, setDeleting] = useState<Branch | null>(null);
  const deleteMutation = useDeleteBranch();

  // Xóa bản ghi cuối cùng của trang cuối → lùi về trang trước.
  const { data } = query;
  const { page } = list.state;
  const { setPage } = list;
  useEffect(() => {
    if (data && data.items.length === 0 && data.total > 0 && page > 1) {
      setPage(Math.max(1, data.total_page));
    }
  }, [data, page, setPage]);

  const canUpdate = can(Perm.BRANCH_UPDATE);
  const canDelete = can(Perm.BRANCH_DELETE);

  const columns = useMemo<DataTableColumn<Branch>[]>(
    () => [
      {
        accessorKey: 'code',
        header: t('fields.code'),
        meta: { sortKey: 'code' },
        cell: ({ row }) => <span className="font-medium text-primary">{row.original.code}</span>,
      },
      {
        accessorKey: 'name',
        header: t('fields.name'),
        meta: { sortKey: 'name' },
        cell: ({ row }) => (
          <div className="grid min-w-48 gap-0.5">
            <span className="font-medium text-primary">{row.original.name}</span>
            {row.original.email ? (
              <span className="text-caption text-tertiary">{row.original.email}</span>
            ) : null}
          </div>
        ),
      },
      {
        accessorKey: 'address',
        header: t('fields.address'),
        cell: ({ row }) => (
          <span className="line-clamp-2 min-w-48 text-secondary">{row.original.address}</span>
        ),
      },
      {
        accessorKey: 'phone',
        header: t('fields.phone'),
        cell: ({ row }) => (
          <span className="whitespace-nowrap text-secondary">{row.original.phone}</span>
        ),
      },
      {
        id: 'hours',
        header: t('fields.hours'),
        cell: ({ row }) => (
          <span className="whitespace-nowrap text-secondary">
            {row.original.open_time} – {row.original.close_time}
          </span>
        ),
      },
      {
        accessorKey: 'status',
        header: t('fields.status'),
        meta: { sortKey: 'status' },
        cell: ({ row }) => <StatusBadge status={row.original.status} />,
      },
      {
        id: 'actions',
        header: () => <span className="sr-only">{tc('actions.actions')}</span>,
        meta: { className: 'w-14 text-right' },
        cell: ({ row }) =>
          canUpdate || canDelete ? (
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
                      setEditing(row.original);
                      setFormOpen(true);
                    }}
                  >
                    <PencilSquareIcon />
                    {tc('actions.edit')}
                  </DropdownMenuItem>
                ) : null}
                {canDelete ? (
                  <DropdownMenuItem variant="danger" onSelect={() => setDeleting(row.original)}>
                    <TrashIcon />
                    {tc('actions.delete')}
                  </DropdownMenuItem>
                ) : null}
              </DropdownMenuContent>
            </DropdownMenu>
          ) : null,
      },
    ],
    [t, tc, canUpdate, canDelete],
  );

  const openCreate = () => {
    setEditing(null);
    setFormOpen(true);
  };

  const createButton = (
    <PermissionGate perm={Perm.BRANCH_CREATE}>
      <Button onClick={openCreate}>
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
    } catch (error) {
      toastError(error, { [ApiCode.CONFLICT]: t('toast.deleteConflict') });
    }
  };

  return (
    <div className="grid gap-6">
      <PageHeader title={t('title')} description={t('description')} actions={createButton} />

      <DataTable
        columns={columns}
        data={query.data?.items}
        total={query.data?.total ?? 0}
        page={list.state.page}
        limit={list.state.limit}
        onPageChange={list.setPage}
        onLimitChange={list.setLimit}
        sortBy={list.state.order_by}
        sortDir={list.state.order_dir}
        onSortChange={list.setSort}
        isLoading={query.isPending}
        isFetching={query.isFetching}
        error={query.error}
        onRetry={() => void query.refetch()}
        getRowId={row => String(row.id)}
        toolbar={
          <>
            <SearchInput
              value={list.state.keyword}
              onSearch={list.setKeyword}
              placeholder={t('searchPlaceholder')}
            />
            <Select
              value={list.state.filters.status ?? ALL}
              onValueChange={value => list.setFilter('status', value === ALL ? undefined : value)}
            >
              <SelectTrigger className="w-full sm:w-52" aria-label={t('fields.status')}>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={ALL}>{t('allStatus')}</SelectItem>
                <SelectItem value="ACTIVE">{tc('status.ACTIVE')}</SelectItem>
                <SelectItem value="INACTIVE">{tc('status.INACTIVE')}</SelectItem>
              </SelectContent>
            </Select>
            {list.hasActiveFilters ? (
              <Button variant="ghost" size="sm" onClick={list.clearFilters}>
                {tc('actions.clearFilters')}
              </Button>
            ) : null}
          </>
        }
        empty={
          list.hasActiveFilters ? (
            <EmptyState
              title={tc('states.noResultTitle')}
              description={tc('states.noResultDescription')}
            />
          ) : (
            <EmptyState
              icon={BuildingStorefrontIcon}
              title={t('empty.title')}
              description={t('empty.description')}
              action={createButton}
            />
          )
        }
      />

      <BranchFormDialog open={formOpen} onOpenChange={setFormOpen} branch={editing} />

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
