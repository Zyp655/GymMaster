'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { ArrowTopRightOnSquareIcon, BellIcon, CheckIcon } from '@heroicons/react/24/outline';
import { EmptyState } from '@/components/common/EmptyState';
import { LoadError } from '@/components/common/LoadError';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { useToastApiError } from '@/hooks/use-api-error';
import { formatDateTime } from '@/lib/format';
import { cn } from '@/lib/utils';
import {
  useNotificationsQuery,
  useReadAllNotificationsMutation,
  useReadNotificationMutation,
} from '../queries';
import { mapPortalNotificationLink, NOTIFICATION_FALLBACK_ROUTE } from '../notification-links';
import type { Notification } from '../types';
import { PortalPageHeader } from './PortalPageHeader';

/** Đích điều hướng khi bấm thông báo; link không hợp lệ/không tồn tại → ở lại trang thông báo. */
function openableRoute(link: string): string | null {
  const route = mapPortalNotificationLink(link);
  return route && route !== NOTIFICATION_FALLBACK_ROUTE ? route : null;
}

export function NotificationsView() {
  const t = useTranslations('portal.notifications');
  const router = useRouter();
  const toastError = useToastApiError();
  const [filter, setFilter] = useState<'all' | 'unread'>('all');

  const query = useNotificationsQuery(50);
  const readMutation = useReadNotificationMutation();
  const readAllMutation = useReadAllNotificationsMutation();

  const items = query.data?.items ?? [];
  const unreadItems = items.filter(item => !item.is_read);
  const displayedItems = filter === 'unread' ? unreadItems : items;

  const handleMarkAllRead = async () => {
    try {
      await readAllMutation.mutateAsync();
      toast.success(t('markAllSuccess'));
    } catch (error) {
      toastError(error);
    }
  };

  const handleSelectItem = async (item: Notification) => {
    if (!item.is_read) {
      try {
        await readMutation.mutateAsync(item.id);
      } catch (error) {
        toastError(error);
      }
    }
    const targetRoute = openableRoute(item.link);
    if (targetRoute) {
      router.push(targetRoute);
    }
  };

  return (
    <div className="grid gap-6">
      <PortalPageHeader
        title={t('title')}
        subtitle={t('subtitle')}
        actions={
          unreadItems.length > 0 ? (
            <Button
              variant="outline"
              size="sm"
              disabled={readAllMutation.isPending}
              onClick={() => void handleMarkAllRead()}
            >
              <CheckIcon />
              {t('markAllRead')}
            </Button>
          ) : null
        }
      />

      <div className="flex flex-wrap items-center gap-2 border-b border-line pb-3">
        <Button
          variant={filter === 'all' ? 'primary' : 'ghost'}
          size="sm"
          onClick={() => setFilter('all')}
        >
          {t('filterAll')}
        </Button>
        <Button
          variant={filter === 'unread' ? 'primary' : 'ghost'}
          size="sm"
          onClick={() => setFilter('unread')}
        >
          {t('filterUnread', { count: unreadItems.length })}
        </Button>
      </div>

      {query.isLoading ? (
        <div className="grid gap-3">
          <Skeleton className="h-24 rounded-2xl" />
          <Skeleton className="h-24 rounded-2xl" />
          <Skeleton className="h-24 rounded-2xl" />
        </div>
      ) : query.isError || !query.data ? (
        <Card>
          <LoadError error={query.error} onRetry={() => void query.refetch()} />
        </Card>
      ) : displayedItems.length === 0 ? (
        <Card>
          <EmptyState
            icon={BellIcon}
            title={filter === 'unread' ? t('emptyUnreadTitle') : t('emptyTitle')}
            description={filter === 'unread' ? t('emptyUnreadDesc') : t('emptyDesc')}
          />
        </Card>
      ) : (
        <div className="grid gap-3">
          {displayedItems.map(item => {
            const targetRoute = openableRoute(item.link);
            return (
              <Card
                key={item.id}
                className={cn(
                  'cursor-pointer gap-2 p-4 transition-colors hover:bg-subtle',
                  !item.is_read && 'border-brand-300 bg-brand-25',
                )}
                onClick={() => void handleSelectItem(item)}
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="flex min-w-0 items-center gap-2">
                    {!item.is_read ? (
                      <span className="size-2.5 shrink-0 rounded-full bg-brand-500" />
                    ) : null}
                    <h3 className="truncate text-h4 font-semibold text-primary">{item.title}</h3>
                  </div>
                  <span className="shrink-0 text-caption text-tertiary">
                    {formatDateTime(item.created_at)}
                  </span>
                </div>
                <p className="text-body text-secondary">{item.body}</p>
                {targetRoute ? (
                  <div className="flex items-center gap-1 text-caption font-medium text-brand-600">
                    <span>{t('openLink')}</span>
                    <ArrowTopRightOnSquareIcon className="size-3.5" />
                  </div>
                ) : null}
              </Card>
            );
          })}
        </div>
      )}
    </div>
  );
}
