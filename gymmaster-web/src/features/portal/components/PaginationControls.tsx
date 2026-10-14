'use client';

import { useTranslations } from 'next-intl';
import { ChevronLeftIcon, ChevronRightIcon } from '@heroicons/react/20/solid';
import { Button } from '@/components/ui/button';

export function PaginationControls({
  page,
  totalPage,
  total,
  onPageChange,
}: {
  page: number;
  totalPage: number;
  total: number;
  onPageChange: (nextPage: number) => void;
}) {
  const t = useTranslations('common.table');
  const safeTotalPage = Math.max(1, totalPage);

  return (
    <div className="flex flex-wrap items-center justify-between gap-3 border-t border-line pt-4">
      <p className="text-caption text-secondary">{t('totalRows', { total })}</p>
      <div className="flex items-center gap-2">
        <span className="text-caption font-medium text-secondary">
          {t('pageInfo', { page, totalPage: safeTotalPage })}
        </span>
        <Button
          variant="outline"
          size="icon-sm"
          disabled={page <= 1}
          onClick={() => onPageChange(page - 1)}
          aria-label={t('prevPage')}
        >
          <ChevronLeftIcon />
        </Button>
        <Button
          variant="outline"
          size="icon-sm"
          disabled={page >= safeTotalPage}
          onClick={() => onPageChange(page + 1)}
          aria-label={t('nextPage')}
        >
          <ChevronRightIcon />
        </Button>
      </div>
    </div>
  );
}
