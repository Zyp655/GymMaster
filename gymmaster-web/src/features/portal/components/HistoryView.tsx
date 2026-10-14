'use client';

import { useState } from 'react';
import { useTranslations } from 'next-intl';
import { ArrowRightEndOnRectangleIcon, ClockIcon } from '@heroicons/react/24/outline';
import { EmptyState } from '@/components/common/EmptyState';
import { LoadError } from '@/components/common/LoadError';
import { Badge } from '@/components/ui/badge';
import { Card } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { formatDate, formatDateTime } from '@/lib/format';
import { useCheckinsQuery } from '../queries';
import { PaginationControls } from './PaginationControls';
import { PortalPageHeader } from './PortalPageHeader';

const PAGE_LIMIT = 15;

/** Lịch sử ra vào của hội viên (gồm cả lượt bị từ chối). */
export function HistoryView() {
  const t = useTranslations('portal.history');
  const ts = useTranslations('portal.status');
  const [page, setPage] = useState(1);
  const query = useCheckinsQuery(page, PAGE_LIMIT);

  return (
    <div className="grid gap-4">
      <PortalPageHeader title={t('title')} subtitle={t('subtitle')} />
      <Card className="gap-0 p-0">
        {query.isLoading ? (
          <div className="grid gap-2 p-4">
            {Array.from({ length: 5 }, (_, i) => (
              <Skeleton key={i} className="h-16 w-full rounded-xl" />
            ))}
          </div>
        ) : query.isError || !query.data ? (
          <LoadError error={query.error} onRetry={() => void query.refetch()} />
        ) : query.data.items.length === 0 ? (
          <EmptyState icon={ClockIcon} title={t('emptyTitle')} description={t('emptyDesc')} />
        ) : (
          <>
            <ul className="divide-y divide-line-subtle">
              {query.data.items.map(item => (
                <li key={item.id} className="flex items-start gap-3 p-4">
                  <div
                    className={
                      item.result === 'ALLOWED'
                        ? 'flex size-10 shrink-0 items-center justify-center rounded-xl bg-success-50 text-success'
                        : 'flex size-10 shrink-0 items-center justify-center rounded-xl bg-danger-50 text-danger'
                    }
                  >
                    <ArrowRightEndOnRectangleIcon className="size-5" />
                  </div>
                  <div className="grid min-w-0 flex-1 gap-0.5">
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <p className="text-body font-semibold text-primary">
                        {formatDateTime(item.checked_in_at)}
                      </p>
                      <Badge variant={item.result === 'ALLOWED' ? 'success' : 'danger'}>
                        {ts(`checkinResult.${item.result}`)}
                      </Badge>
                    </div>
                    <p className="truncate text-caption text-secondary">
                      {item.branch_name} · {ts(`checkinMethod.${item.method}`)}
                      {item.plan_name ? ` · ${item.plan_name}` : ''}
                    </p>
                    {item.checked_out_at ? (
                      <p className="text-caption text-tertiary">
                        {t('checkedOut', { time: formatDate(item.checked_out_at, 'HH:mm') })}
                      </p>
                    ) : null}
                    {item.result === 'DENIED' && item.deny_message ? (
                      <p className="text-caption text-danger">{item.deny_message}</p>
                    ) : null}
                  </div>
                </li>
              ))}
            </ul>
            <div className="border-t border-line-subtle p-4">
              <PaginationControls
                page={query.data.current_page}
                totalPage={query.data.total_page}
                total={query.data.total}
                onPageChange={setPage}
              />
            </div>
          </>
        )}
      </Card>
    </div>
  );
}
