'use client';

import { useEffect, useMemo, useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { ClipboardDocumentCheckIcon, FireIcon, PlusIcon } from '@heroicons/react/24/outline';
import { EmptyState } from '@/components/common/EmptyState';
import { LoadError } from '@/components/common/LoadError';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
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
import { Skeleton } from '@/components/ui/skeleton';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Textarea } from '@/components/ui/textarea';
import { useToastApiError } from '@/hooks/use-api-error';
import { formatDateTime } from '@/lib/format';
import { looseT } from '@/lib/utils';
import { extractWorkoutLogs } from '../api';
import {
  useCreateWorkoutLogMutation,
  useExercisesQuery,
  useWorkoutLogsQuery,
  useWorkoutPlansQuery,
} from '../queries';
import { createWorkoutLogSchema, type WorkoutLogFormValues } from '../schema';
import type { ExerciseItem, WorkoutPlan, WorkoutPlanItem } from '../types';
import { PortalPageHeader } from './PortalPageHeader';

export function WorkoutView() {
  const t = useTranslations('portal.workout');
  const plansQuery = useWorkoutPlansQuery();
  const exercisesQuery = useExercisesQuery();
  const logsQuery = useWorkoutLogsQuery();

  const [logPreset, setLogPreset] = useState<Partial<WorkoutLogFormValues> | null>(null);

  const openFreeLog = () => {
    const firstExerciseId = exercisesQuery.data?.[0]?.id ?? 0;
    setLogPreset({
      exercise_id: firstExerciseId,
      sets: 3,
      reps: 10,
      weight_kg: 20,
      note: '',
    });
  };

  const openItemLog = (item: WorkoutPlanItem) => {
    const parsedReps = Number.parseInt(String(item.reps), 10);
    setLogPreset({
      exercise_id: item.exercise_id,
      sets: item.sets || 3,
      reps: Number.isFinite(parsedReps) && parsedReps > 0 ? parsedReps : 10,
      weight_kg: item.weight_kg ?? 0,
      note: '',
    });
  };

  const logs = extractWorkoutLogs(logsQuery.data);

  return (
    <div className="grid gap-6">
      <PortalPageHeader
        title={t('title')}
        subtitle={t('subtitle')}
        actions={
          <Button size="sm" onClick={openFreeLog}>
            <PlusIcon />
            {t('logBtn')}
          </Button>
        }
      />

      <section className="grid gap-3">
        <h2 className="text-h3 font-semibold text-primary">{t('plansTitle')}</h2>

        {plansQuery.isLoading ? (
          <div className="grid gap-4">
            <Skeleton className="h-64 rounded-2xl" />
          </div>
        ) : plansQuery.isError || !plansQuery.data ? (
          <Card>
            <LoadError error={plansQuery.error} onRetry={() => void plansQuery.refetch()} />
          </Card>
        ) : plansQuery.data.length === 0 ? (
          <Card>
            <EmptyState
              icon={FireIcon}
              title={t('emptyPlansTitle')}
              description={t('emptyPlansDesc')}
            />
          </Card>
        ) : (
          <div className="grid gap-4">
            {plansQuery.data.map(plan => (
              <WorkoutPlanCard key={plan.id} plan={plan} onLogItem={openItemLog} />
            ))}
          </div>
        )}
      </section>

      <section className="grid gap-3">
        <h2 className="text-h3 font-semibold text-primary">{t('recentLogsTitle')}</h2>

        {logsQuery.isLoading ? (
          <Card className="p-4">
            <Skeleton className="h-44 rounded-xl" />
          </Card>
        ) : logsQuery.isError ? (
          <Card>
            <LoadError error={logsQuery.error} onRetry={() => void logsQuery.refetch()} />
          </Card>
        ) : logs.length === 0 ? (
          <Card>
            <EmptyState
              icon={ClipboardDocumentCheckIcon}
              title={t('emptyLogsTitle')}
              description={t('emptyLogsDesc')}
            />
          </Card>
        ) : (
          <Card>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('colDate')}</TableHead>
                  <TableHead>{t('colExercise')}</TableHead>
                  <TableHead>{t('colSets')}</TableHead>
                  <TableHead>{t('colReps')}</TableHead>
                  <TableHead>{t('colWeight')}</TableHead>
                  <TableHead>{t('colNote')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {logs.map(log => (
                  <TableRow key={log.id}>
                    <TableCell className="whitespace-nowrap text-secondary">
                      {formatDateTime(log.logged_at)}
                    </TableCell>
                    <TableCell className="font-semibold text-primary">
                      {log.exercise_name}
                    </TableCell>
                    <TableCell>{log.sets}</TableCell>
                    <TableCell>{log.reps}</TableCell>
                    <TableCell className="font-semibold text-brand-600">
                      {log.weight_kg} kg
                    </TableCell>
                    <TableCell className="text-secondary">{log.note || '—'}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </Card>
        )}
      </section>

      <WorkoutLogDialog
        preset={logPreset}
        exercises={exercisesQuery.data ?? []}
        onOpenChange={open => !open && setLogPreset(null)}
      />
    </div>
  );
}

function WorkoutPlanCard({
  plan,
  onLogItem,
}: {
  plan: WorkoutPlan;
  onLogItem: (item: WorkoutPlanItem) => void;
}) {
  const t = useTranslations('portal.workout');
  const sortedDays = useMemo(
    () => [...(plan.days ?? [])].sort((a, b) => a.day_index - b.day_index),
    [plan.days],
  );
  const [activeDayIndex, setActiveDayIndex] = useState<number>(
    sortedDays[0]?.day_index ?? 1,
  );

  const currentDay =
    sortedDays.find(d => d.day_index === activeDayIndex) ?? sortedDays[0] ?? null;

  return (
    <Card className="gap-4">
      <CardHeader>
        <div className="flex flex-wrap items-start justify-between gap-2">
          <div className="grid gap-1">
            <CardTitle>{plan.title}</CardTitle>
            {plan.description ? <CardDescription>{plan.description}</CardDescription> : null}
          </div>
          <div className="flex flex-wrap items-center gap-2">
            {plan.goal ? (
              <Badge variant="brand">{t('goal', { goal: plan.goal })}</Badge>
            ) : null}
            {plan.trainer_name ? (
              <Badge variant="neutral">{t('trainer', { name: plan.trainer_name })}</Badge>
            ) : null}
          </div>
        </div>
      </CardHeader>

      {sortedDays.length > 0 ? (
        <div className="grid gap-4">
          <div className="flex flex-wrap items-center gap-2 border-b border-line pb-3">
            {sortedDays.map(day => (
              <Button
                key={day.id || day.day_index}
                size="sm"
                variant={currentDay?.day_index === day.day_index ? 'primary' : 'ghost'}
                onClick={() => setActiveDayIndex(day.day_index)}
              >
                {day.title
                  ? t('dayLabel', { index: day.day_index, title: day.title })
                  : `Ngày ${day.day_index}`}
              </Button>
            ))}
          </div>

          {currentDay && currentDay.items.length > 0 ? (
            <div className="grid gap-3 sm:grid-cols-2">
              {currentDay.items.map(item => (
                <div
                  key={item.id}
                  className="flex flex-col justify-between gap-3 rounded-xl border border-line bg-canvas p-4"
                >
                  <div className="grid gap-2">
                    <div className="flex items-start justify-between gap-2">
                      <h4 className="text-body font-semibold text-primary">
                        {item.exercise_name}
                      </h4>
                      {item.muscle_group ? (
                        <Badge variant="outline">{item.muscle_group}</Badge>
                      ) : null}
                    </div>

                    <div className="flex flex-wrap items-center gap-2">
                      <Badge variant="brand">
                        {t('setsReps', { sets: item.sets, reps: item.reps })}
                      </Badge>
                      {item.weight_kg != null && item.weight_kg > 0 ? (
                        <Badge variant="info">{t('weightKg', { weight: item.weight_kg })}</Badge>
                      ) : null}
                      {item.rest_sec != null && item.rest_sec > 0 ? (
                        <Badge variant="neutral">
                          {t('restSec', { seconds: item.rest_sec })}
                        </Badge>
                      ) : null}
                    </div>

                    {item.note ? (
                      <p className="text-caption text-secondary">{item.note}</p>
                    ) : null}
                  </div>

                  <div className="flex justify-end">
                    <Button size="sm" variant="outline" onClick={() => onLogItem(item)}>
                      <PlusIcon />
                      {t('logFromPlanBtn')}
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          ) : null}
        </div>
      ) : null}
    </Card>
  );
}

function WorkoutLogDialog({
  preset,
  exercises,
  onOpenChange,
}: {
  preset: Partial<WorkoutLogFormValues> | null;
  exercises: ExerciseItem[];
  onOpenChange: (open: boolean) => void;
}) {
  const t = useTranslations('portal.workout.logDialog');
  const tc = useTranslations('common');
  const tv = useTranslations('common.validation');
  const toastError = useToastApiError();
  const mutation = useCreateWorkoutLogMutation();

  const schema = useMemo(() => createWorkoutLogSchema(looseT(tv)), [tv]);
  const form = useForm<WorkoutLogFormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      exercise_id: preset?.exercise_id ?? exercises[0]?.id ?? 0,
      sets: preset?.sets ?? 3,
      reps: preset?.reps ?? 10,
      weight_kg: preset?.weight_kg ?? 20,
      note: preset?.note ?? '',
    },
  });

  useEffect(() => {
    if (preset) {
      form.reset({
        exercise_id: preset.exercise_id || exercises[0]?.id || 0,
        sets: preset.sets ?? 3,
        reps: preset.reps ?? 10,
        weight_kg: preset.weight_kg ?? 20,
        note: preset.note ?? '',
      });
    }
  }, [preset, exercises, form]);

  const onSubmit = form.handleSubmit(async values => {
    try {
      await mutation.mutateAsync({
        exercise_id: values.exercise_id,
        sets: values.sets,
        reps: values.reps,
        weight_kg: values.weight_kg,
        note: values.note?.trim() || undefined,
      });
      toast.success(t('success'));
      onOpenChange(false);
    } catch (error) {
      toastError(error);
    }
  });

  return (
    <Dialog open={Boolean(preset)} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md" closeLabel={tc('actions.close')}>
        <DialogHeader>
          <DialogTitle>{t('title')}</DialogTitle>
          <DialogDescription>{t('description')}</DialogDescription>
        </DialogHeader>

        <Form {...form}>
          <form onSubmit={onSubmit} className="grid gap-4" noValidate>
            <FormField
              control={form.control}
              name="exercise_id"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('exerciseLabel')}</FormLabel>
                  <Select
                    value={field.value ? String(field.value) : undefined}
                    onValueChange={v => field.onChange(Number(v))}
                  >
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue placeholder={t('exercisePlaceholder')} />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      {exercises.map(ex => (
                        <SelectItem key={ex.id} value={String(ex.id)}>
                          {ex.name} {ex.muscle_group ? `(${ex.muscle_group})` : ''}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />

            <div className="grid grid-cols-3 gap-3">
              <FormField
                control={form.control}
                name="sets"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('setsLabel')}</FormLabel>
                    <FormControl>
                      <Input
                        type="number"
                        min={1}
                        value={field.value}
                        onChange={e => field.onChange(Number(e.target.value))}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="reps"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('repsLabel')}</FormLabel>
                    <FormControl>
                      <Input
                        type="number"
                        min={1}
                        value={field.value}
                        onChange={e => field.onChange(Number(e.target.value))}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="weight_kg"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('weightLabel')}</FormLabel>
                    <FormControl>
                      <Input
                        type="number"
                        step="0.5"
                        min={0}
                        value={field.value}
                        onChange={e => field.onChange(Number(e.target.value))}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>

            <FormField
              control={form.control}
              name="note"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('noteLabel')}</FormLabel>
                  <FormControl>
                    <Textarea {...field} placeholder={t('notePlaceholder')} rows={2} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => onOpenChange(false)}
                disabled={mutation.isPending}
              >
                {tc('actions.cancel')}
              </Button>
              <Button type="submit" disabled={mutation.isPending}>
                {mutation.isPending ? t('submitting') : t('submit')}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
