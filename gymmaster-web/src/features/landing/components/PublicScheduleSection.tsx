'use client';

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import dayjs from 'dayjs';
import { useTranslations } from 'next-intl';
import {
  CalendarDaysIcon,
  MapPinIcon,
  UserIcon,
} from '@heroicons/react/24/outline';
import { EmptyState } from '@/components/common/EmptyState';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { formatDate, formatDateTime } from '@/lib/format';
import { getPublicClassSchedules } from '../client-api';
import type { PublicBranch, PublicClassSchedule } from '../types';
import { SectionHeading } from './SectionHeading';

export function PublicScheduleSection({
  branches,
  initialSchedules = [],
}: {
  branches: PublicBranch[];
  initialSchedules?: PublicClassSchedule[];
}) {
  const t = useTranslations('common.landing.schedules');
  const [branchId, setBranchId] = useState<number | undefined>(undefined);
  const [fromDate] = useState(() => dayjs().format('YYYY-MM-DD'));
  const [toDate] = useState(() => dayjs().add(7, 'day').format('YYYY-MM-DD'));

  const schedulesQuery = useQuery({
    queryKey: ['public', 'class-schedules', branchId ?? 'all', fromDate, toDate],
    queryFn: () =>
      getPublicClassSchedules({
        branch_id: branchId,
        from: fromDate,
        to: toDate,
      }),
    initialData: branchId === undefined && initialSchedules.length > 0 ? initialSchedules : undefined,
  });

  const schedules = schedulesQuery.data ?? [];

  return (
    <section id="lich-lop" className="scroll-mt-16 border-t border-line bg-canvas">
      <div className="mx-auto grid max-w-6xl gap-8 px-4 py-20 sm:px-6">
        <div className="flex flex-wrap items-end justify-between gap-4">
          <SectionHeading
            eyebrow={t('eyebrow')}
            title={t('title')}
            subtitle={t('subtitle')}
          />

          {branches.length > 0 ? (
            <div className="flex flex-wrap items-center gap-2">
              <Button
                size="sm"
                variant={branchId === undefined ? 'primary' : 'outline'}
                onClick={() => setBranchId(undefined)}
              >
                {t('allBranches')}
              </Button>
              {branches.map(branch => (
                <Button
                  key={branch.id}
                  size="sm"
                  variant={branchId === branch.id ? 'primary' : 'outline'}
                  onClick={() => setBranchId(branch.id)}
                >
                  {branch.name}
                </Button>
              ))}
            </div>
          ) : null}
        </div>

        {schedulesQuery.isLoading ? (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            <Skeleton className="h-40 rounded-2xl" />
            <Skeleton className="h-40 rounded-2xl" />
            <Skeleton className="h-40 rounded-2xl" />
          </div>
        ) : schedules.length === 0 ? (
          <Card>
            <EmptyState
              icon={CalendarDaysIcon}
              title={t('emptyTitle')}
              description={t('emptyDescription')}
            />
          </Card>
        ) : (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {schedules.map(item => (
              <Card key={item.id} className="justify-between gap-3">
                <div className="grid gap-2">
                  <div className="flex items-start justify-between gap-2">
                    <div>
                      <h3 className="text-h4 font-semibold text-primary">
                        {item.class_type_name}
                      </h3>
                      <p className="text-body font-medium text-brand-600">
                        {formatDateTime(item.start_at)} – {formatDate(item.end_at, 'HH:mm')}
                      </p>
                    </div>
                    <Badge variant={item.available_slots > 0 ? 'success' : 'warning'}>
                      {item.available_slots > 0
                        ? t('availableSlots', {
                            available: item.available_slots,
                            capacity: item.capacity,
                          })
                        : t('fullSlots', { capacity: item.capacity })}
                    </Badge>
                  </div>

                  <div className="grid gap-1.5 rounded-xl bg-canvas p-3 text-caption text-secondary">
                    <div className="flex items-center gap-1.5">
                      <UserIcon className="size-4 shrink-0 text-tertiary" />
                      <span>{t('trainer', { name: item.trainer_name })}</span>
                    </div>
                    <div className="flex items-center gap-1.5">
                      <MapPinIcon className="size-4 shrink-0 text-tertiary" />
                      <span>
                        {t('room', { name: item.room_name })} · {item.branch_name}
                      </span>
                    </div>
                  </div>
                </div>
              </Card>
            ))}
          </div>
        )}
      </div>
    </section>
  );
}
