'use client';

import type { ReactNode } from 'react';
import { tableFeatures, useTable, type ColumnDef, type RowData } from '@tanstack/react-table';
import { useTranslations } from 'next-intl';
import {
  ChevronDoubleLeftIcon,
  ChevronDoubleRightIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
} from '@heroicons/react/20/solid';
import { ArrowsUpDownIcon, ArrowDownIcon, ArrowUpIcon } from '@heroicons/react/16/solid';
import { Button } from '@/components/ui/button';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { PAGE_SIZE_OPTIONS } from '@/hooks/use-list-params';
import { cn } from '@/lib/utils';
import { LoadError } from './LoadError';

export interface DataTableColumnMeta {
  /** Class áp cho cả <th> và <td> (căn lề, độ rộng). */
  className?: string;
  /** Tên cột gửi lên API (`order_by`) – có giá trị thì tiêu đề cột bấm được để sắp xếp. */
  sortKey?: string;
}

/**
 * TanStack Table v9: chỉ dùng core (render cột/hàng). Phân trang & sắp xếp do server xử lý,
 * trạng thái nằm trên URL (useListParams) nên không đăng ký rowSorting/rowPagination feature.
 */
export const dataTableFeatures = tableFeatures({ columnMeta: {} as DataTableColumnMeta });

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type DataTableColumn<T extends RowData> = ColumnDef<typeof dataTableFeatures, T, any>;

const EMPTY_ROWS: never[] = [];

export interface DataTableProps<T extends RowData> {
  columns: DataTableColumn<T>[];
  data: T[] | undefined;
  /** Tổng bản ghi từ server. */
  total: number;
  page: number;
  limit: number;
  onPageChange: (page: number) => void;
  onLimitChange: (limit: number) => void;
  sortBy?: string;
  sortDir?: 'ASC' | 'DESC';
  onSortChange?: (sortBy?: string, sortDir?: 'ASC' | 'DESC') => void;
  isLoading?: boolean;
  /** Đang tải trang mới khi đã có dữ liệu cũ (làm mờ bảng thay vì skeleton). */
  isFetching?: boolean;
  error?: unknown;
  onRetry?: () => void;
  /** Thanh công cụ phía trên bảng: ô tìm kiếm, bộ lọc. */
  toolbar?: ReactNode;
  /** Hiển thị khi không có dữ liệu. */
  empty?: ReactNode;
  getRowId?: (row: T) => string;
}

/**
 * Bảng dữ liệu dùng chung: phân trang + sắp xếp phía server.
 * Đủ 4 trạng thái: loading (skeleton), error (thử lại), empty, dữ liệu.
 */
export function DataTable<T extends RowData>({
  columns,
  data,
  total,
  page,
  limit,
  onPageChange,
  onLimitChange,
  sortBy,
  sortDir,
  onSortChange,
  isLoading,
  isFetching,
  error,
  onRetry,
  toolbar,
  empty,
  getRowId,
}: DataTableProps<T>) {
  const t = useTranslations('common.table');
  const totalPage = Math.max(1, Math.ceil(total / limit));

  const table = useTable({
    features: dataTableFeatures,
    columns,
    data: data ?? (EMPTY_ROWS as T[]),
    getRowId: getRowId ? (row: T) => getRowId(row) : undefined,
  });

  const toggleSort = (key: string) => {
    if (!onSortChange) return;
    // Chu kỳ: chưa sắp xếp → tăng dần → giảm dần → bỏ sắp xếp.
    if (sortBy !== key) onSortChange(key, 'ASC');
    else if (sortDir === 'ASC') onSortChange(key, 'DESC');
    else onSortChange(undefined, undefined);
  };

  const colCount = columns.length;
  const rows = table.getRowModel().rows;
  const showError = !!error && !data;

  return (
    <div className="overflow-hidden rounded-2xl border border-line bg-surface">
      {toolbar ? (
        <div className="flex flex-wrap items-center gap-3 border-b border-line p-4">{toolbar}</div>
      ) : null}

      {showError ? (
        <LoadError error={error} onRetry={onRetry} />
      ) : (
        <Table className={cn(isFetching && !isLoading && 'opacity-60 transition-opacity')}>
          <TableHeader>
            {table.getHeaderGroups().map(headerGroup => (
              <TableRow key={headerGroup.id} className="hover:bg-transparent">
                {headerGroup.headers.map(header => {
                  const meta = header.column.columnDef.meta;
                  const sortKey = onSortChange ? meta?.sortKey : undefined;
                  const sorted = sortKey && sortBy === sortKey ? sortDir : undefined;
                  const content = header.isPlaceholder ? null : (
                    <table.FlexRender header={header} />
                  );
                  return (
                    <TableHead
                      key={header.id}
                      className={meta?.className}
                      aria-sort={
                        sorted === 'ASC'
                          ? 'ascending'
                          : sorted === 'DESC'
                            ? 'descending'
                            : undefined
                      }
                    >
                      {sortKey ? (
                        <button
                          type="button"
                          onClick={() => toggleSort(sortKey)}
                          className="inline-flex cursor-pointer items-center gap-1 uppercase hover:text-primary"
                          aria-label={sorted === 'ASC' ? t('sortDesc') : t('sortAsc')}
                        >
                          {content}
                          {sorted === 'ASC' ? (
                            <ArrowUpIcon className="size-3.5 text-brand-600" />
                          ) : sorted === 'DESC' ? (
                            <ArrowDownIcon className="size-3.5 text-brand-600" />
                          ) : (
                            <ArrowsUpDownIcon className="size-3.5" />
                          )}
                        </button>
                      ) : (
                        content
                      )}
                    </TableHead>
                  );
                })}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {isLoading ? (
              Array.from({ length: Math.min(limit, 8) }).map((_, i) => (
                <TableRow key={`skeleton-${i}`} className="hover:bg-transparent">
                  {Array.from({ length: colCount }).map((__, j) => (
                    <TableCell key={j}>
                      <Skeleton className="h-5 w-full max-w-40" />
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : rows.length === 0 ? (
              <TableRow className="hover:bg-transparent">
                <TableCell colSpan={colCount} className="p-0">
                  {empty}
                </TableCell>
              </TableRow>
            ) : (
              rows.map(row => (
                <TableRow key={row.id}>
                  {row.getAllCells().map(cell => (
                    <TableCell key={cell.id} className={cell.column.columnDef.meta?.className}>
                      <table.FlexRender cell={cell} />
                    </TableCell>
                  ))}
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      )}

      {!showError && total > 0 ? (
        <div className="flex flex-wrap items-center justify-between gap-3 border-t border-line px-4 py-3">
          <div className="flex items-center gap-3 text-body text-secondary">
            <span>{t('totalRows', { total })}</span>
            <div className="flex items-center gap-2">
              <span className="hidden sm:inline">{t('rowsPerPage')}</span>
              <Select value={String(limit)} onValueChange={value => onLimitChange(Number(value))}>
                <SelectTrigger className="h-9 w-20" aria-label={t('rowsPerPage')}>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {PAGE_SIZE_OPTIONS.map(size => (
                    <SelectItem key={size} value={String(size)}>
                      {size}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
          <div className="flex items-center gap-1">
            <span className="mr-2 text-body text-secondary">
              {t('pageInfo', { page, totalPage })}
            </span>
            <Button
              variant="outline"
              size="icon-sm"
              onClick={() => onPageChange(1)}
              disabled={page <= 1}
              aria-label={t('firstPage')}
            >
              <ChevronDoubleLeftIcon />
            </Button>
            <Button
              variant="outline"
              size="icon-sm"
              onClick={() => onPageChange(page - 1)}
              disabled={page <= 1}
              aria-label={t('prevPage')}
            >
              <ChevronLeftIcon />
            </Button>
            <Button
              variant="outline"
              size="icon-sm"
              onClick={() => onPageChange(page + 1)}
              disabled={page >= totalPage}
              aria-label={t('nextPage')}
            >
              <ChevronRightIcon />
            </Button>
            <Button
              variant="outline"
              size="icon-sm"
              onClick={() => onPageChange(totalPage)}
              disabled={page >= totalPage}
              aria-label={t('lastPage')}
            >
              <ChevronDoubleRightIcon />
            </Button>
          </div>
        </div>
      ) : null}
    </div>
  );
}
