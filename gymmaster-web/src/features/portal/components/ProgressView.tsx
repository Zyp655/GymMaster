'use client';

import { useState } from 'react';
import {
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { useTranslations } from 'next-intl';
import { ChartBarIcon, FireIcon } from '@heroicons/react/24/outline';
import { EmptyState } from '@/components/common/EmptyState';
import { LoadError } from '@/components/common/LoadError';
import { Badge } from '@/components/ui/badge';
import { Card, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
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
import { formatDate } from '@/lib/format';
import { extractWorkoutLogs, extractWorkoutProgress, pickDefaultExerciseId } from '../api';
import { useBodyMetricsQuery, useExercisesQuery, useWorkoutLogsQuery } from '../queries';
import { PortalPageHeader } from './PortalPageHeader';

export function ProgressView() {
  const t = useTranslations('portal.progress');
  const tm = useTranslations('portal.progress.metrics');
  const tc = useTranslations('portal.progress.charts');
  const query = useBodyMetricsQuery();

  if (query.isLoading) {
    return (
      <div className="grid gap-6">
        <div className="grid gap-2">
          <Skeleton className="h-8 w-64" />
          <Skeleton className="h-5 w-96 max-w-full" />
        </div>
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <Skeleton className="h-28 rounded-2xl" />
          <Skeleton className="h-28 rounded-2xl" />
          <Skeleton className="h-28 rounded-2xl" />
        </div>
        <div className="grid gap-6 lg:grid-cols-2">
          <Skeleton className="h-80 rounded-2xl" />
          <Skeleton className="h-80 rounded-2xl" />
        </div>
      </div>
    );
  }

  if (query.isError || !query.data) {
    return (
      <Card>
        <LoadError error={query.error} onRetry={() => void query.refetch()} />
      </Card>
    );
  }

  const metrics = query.data;

  if (metrics.length === 0) {
    return (
      <div className="grid gap-6">
        <PortalPageHeader title={t('title')} subtitle={t('subtitle')} />
        <Card>
          <EmptyState
            icon={ChartBarIcon}
            title={t('emptyTitle')}
            description={t('emptyDesc')}
          />
        </Card>
        <WorkoutProgressSection />
      </div>
    );
  }

  const latest = metrics[metrics.length - 1];
  const reversedMetrics = [...metrics].reverse();
  const chartData = metrics.map(m => ({
    date: formatDate(m.measured_at, 'DD/MM'),
    weight_kg: m.weight_kg,
    bmi: m.bmi,
    body_fat_pct: m.body_fat_pct,
    muscle_mass_kg: m.muscle_mass_kg,
  }));

  const summaryCards = [
    {
      label: tm('weight'),
      value: tm('weightUnit', { value: latest.weight_kg }),
    },
    {
      label: tm('height'),
      value: tm('heightUnit', { value: latest.height_cm }),
    },
    {
      label: tm('bmi'),
      value: latest.bmi !== null ? String(latest.bmi) : '—',
    },
    {
      label: tm('bodyFat'),
      value: latest.body_fat_pct !== null ? tm('bodyFatUnit', { value: latest.body_fat_pct }) : '—',
    },
    {
      label: tm('muscleMass'),
      value:
        latest.muscle_mass_kg !== null
          ? tm('muscleMassUnit', { value: latest.muscle_mass_kg })
          : '—',
    },
    {
      label: tm('measurements'),
      value: `${latest.chest_cm ?? '—'} / ${latest.waist_cm ?? '—'} / ${latest.hip_cm ?? '—'} cm`,
    },
  ];

  return (
    <div className="grid gap-6">
      <PortalPageHeader
        title={t('title')}
        subtitle={t('subtitle')}
        actions={
          <Badge variant="brand">
            {t('latestDate', { date: formatDate(latest.measured_at) })}
          </Badge>
        }
      />

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {summaryCards.map(item => (
          <Card key={item.label} className="gap-1 p-4">
            <p className="text-caption text-secondary">{item.label}</p>
            <p className="text-h3 font-semibold text-primary">{item.value}</p>
          </Card>
        ))}
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>{tc('weightBmiTitle')}</CardTitle>
            <CardDescription>{tc('weightBmiDesc')}</CardDescription>
          </CardHeader>
          <div className="h-72 w-full">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={chartData} margin={{ top: 8, right: 12, left: 0, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="var(--color-line)" />
                <XAxis dataKey="date" stroke="var(--color-tertiary)" fontSize={12} />
                <YAxis stroke="var(--color-tertiary)" fontSize={12} />
                <Tooltip />
                <Legend />
                <Line
                  type="monotone"
                  dataKey="weight_kg"
                  name={tm('weight')}
                  stroke="var(--color-brand-500)"
                  strokeWidth={2.5}
                  dot={{ r: 4 }}
                />
                <Line
                  type="monotone"
                  dataKey="bmi"
                  name={tm('bmi')}
                  stroke="var(--color-info)"
                  strokeWidth={2.5}
                  dot={{ r: 4 }}
                  connectNulls
                />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>{tc('fatMuscleTitle')}</CardTitle>
            <CardDescription>{tc('fatMuscleDesc')}</CardDescription>
          </CardHeader>
          <div className="h-72 w-full">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={chartData} margin={{ top: 8, right: 12, left: 0, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="var(--color-line)" />
                <XAxis dataKey="date" stroke="var(--color-tertiary)" fontSize={12} />
                <YAxis stroke="var(--color-tertiary)" fontSize={12} />
                <Tooltip />
                <Legend />
                <Line
                  type="monotone"
                  dataKey="body_fat_pct"
                  name={tm('bodyFat')}
                  stroke="var(--color-warning)"
                  strokeWidth={2.5}
                  dot={{ r: 4 }}
                  connectNulls
                />
                <Line
                  type="monotone"
                  dataKey="muscle_mass_kg"
                  name={tm('muscleMass')}
                  stroke="var(--color-success)"
                  strokeWidth={2.5}
                  dot={{ r: 4 }}
                  connectNulls
                />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </Card>
      </div>

      <WorkoutProgressSection />

      <Card>
        <CardHeader>
          <CardTitle>{t('historyTitle')}</CardTitle>
        </CardHeader>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t('colDate')}</TableHead>
              <TableHead>{t('colWeight')}</TableHead>
              <TableHead>{t('colHeight')}</TableHead>
              <TableHead>{t('colBmi')}</TableHead>
              <TableHead>{t('colBodyFat')}</TableHead>
              <TableHead>{t('colMuscle')}</TableHead>
              <TableHead>{t('colThreeRings')}</TableHead>
              <TableHead>{t('colNote')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {reversedMetrics.map(m => (
              <TableRow key={m.id}>
                <TableCell className="whitespace-nowrap font-medium">
                  {formatDate(m.measured_at)}
                </TableCell>
                <TableCell className="font-semibold">{m.weight_kg}</TableCell>
                <TableCell>{m.height_cm}</TableCell>
                <TableCell>{m.bmi ?? '—'}</TableCell>
                <TableCell>{m.body_fat_pct ?? '—'}</TableCell>
                <TableCell>{m.muscle_mass_kg ?? '—'}</TableCell>
                <TableCell className="whitespace-nowrap text-secondary">
                  {`${m.chest_cm ?? '—'} / ${m.waist_cm ?? '—'} / ${m.hip_cm ?? '—'}`}
                </TableCell>
                <TableCell className="text-secondary">{m.note || '—'}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </Card>
    </div>
  );
}

function WorkoutProgressSection() {
  const t = useTranslations('portal.progress');
  const exercisesQuery = useExercisesQuery();
  const recentLogsQuery = useWorkoutLogsQuery();
  const [selectedExerciseId, setSelectedExerciseId] = useState<number | undefined>(undefined);

  // Mặc định chọn bài tập có nhật ký gần nhất (thay vì bài đầu thư viện – thường chưa có log).
  const defaultExerciseId = pickDefaultExerciseId(
    extractWorkoutLogs(recentLogsQuery.data),
    exercisesQuery.data ?? [],
  );
  const effectiveExerciseId = selectedExerciseId ?? defaultExerciseId;
  const logsQuery = useWorkoutLogsQuery(effectiveExerciseId, Boolean(effectiveExerciseId));
  const progressData = extractWorkoutProgress(logsQuery.data).map(point => ({
    date: formatDate(point.date, 'DD/MM'),
    max_weight: point.max_weight,
    total_volume: point.total_volume,
  }));

  return (
    <Card>
      <CardHeader>
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div className="grid gap-1">
            <CardTitle>{t('workoutProgressTitle')}</CardTitle>
            <CardDescription>{t('workoutProgressDesc')}</CardDescription>
          </div>
          {exercisesQuery.data && exercisesQuery.data.length > 0 ? (
            <div className="w-full sm:w-64">
              <Select
                value={effectiveExerciseId ? String(effectiveExerciseId) : ''}
                onValueChange={v => setSelectedExerciseId(Number(v))}
              >
                <SelectTrigger aria-label={t('workoutExerciseLabel')}>
                  <SelectValue placeholder={t('workoutExercisePlaceholder')} />
                </SelectTrigger>
                <SelectContent>
                  {exercisesQuery.data.map(ex => (
                    <SelectItem key={ex.id} value={String(ex.id)}>
                      {ex.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          ) : null}
        </div>
      </CardHeader>

      {logsQuery.isLoading || exercisesQuery.isLoading || recentLogsQuery.isLoading ? (
        <Skeleton className="h-72 w-full rounded-xl" />
      ) : progressData.length === 0 ? (
        <EmptyState
          icon={FireIcon}
          title={t('workoutEmptyTitle')}
          description={t('workoutEmptyDesc')}
          className="py-8"
        />
      ) : (
        <div className="h-72 w-full">
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={progressData} margin={{ top: 8, right: 12, left: 0, bottom: 0 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--color-line)" />
              <XAxis dataKey="date" stroke="var(--color-tertiary)" fontSize={12} />
              <YAxis yAxisId="left" stroke="var(--color-tertiary)" fontSize={12} />
              <YAxis
                yAxisId="right"
                orientation="right"
                stroke="var(--color-tertiary)"
                fontSize={12}
              />
              <Tooltip />
              <Legend />
              <Line
                yAxisId="left"
                type="monotone"
                dataKey="max_weight"
                name={t('workoutMaxWeight')}
                stroke="var(--color-brand-500)"
                strokeWidth={2.5}
                dot={{ r: 4 }}
              />
              <Line
                yAxisId="right"
                type="monotone"
                dataKey="total_volume"
                name={t('workoutTotalVolume')}
                stroke="var(--color-info)"
                strokeWidth={2.5}
                dot={{ r: 4 }}
              />
            </LineChart>
          </ResponsiveContainer>
        </div>
      )}
    </Card>
  );
}

