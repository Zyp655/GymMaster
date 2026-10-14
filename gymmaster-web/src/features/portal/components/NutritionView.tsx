'use client';

import { useMemo, useState } from 'react';
import dayjs from 'dayjs';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import {
  CalendarDaysIcon,
  ClipboardDocumentListIcon,
  FireIcon,
  PlusCircleIcon,
  SparklesIcon,
} from '@heroicons/react/24/outline';
import { EmptyState } from '@/components/common/EmptyState';
import { LoadError } from '@/components/common/LoadError';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import { Progress } from '@/components/ui/progress';
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
import { useToastApiError } from '@/hooks/use-api-error';
import { formatDate, formatNumber } from '@/lib/format';
import { looseT } from '@/lib/utils';
import { DEFAULT_VIETNAMESE_FOODS } from '../api';
import {
  useCreateMemberFoodLogMutation,
  useMemberFoodLogsQuery,
  useMemberMealPlansQuery,
} from '../queries';
import { createFoodLogSchema, type FoodLogFormValues } from '../schema';
import type { FoodLogItem, MealType } from '../types';
import { PortalPageHeader } from './PortalPageHeader';

const MEAL_TYPES: MealType[] = ['BREAKFAST', 'LUNCH', 'DINNER', 'SNACK'];

export function NutritionView() {
  const t = useTranslations('portal.nutrition');
  const tl = looseT(t);
  const tv = useTranslations('common.validation');
  const toastError = useToastApiError();

  const [selectedDate, setSelectedDate] = useState(() => dayjs().format('YYYY-MM-DD'));

  const overviewQuery = useMemberMealPlansQuery();
  const logsQuery = useMemberFoodLogsQuery(selectedDate);
  const createLogMutation = useCreateMemberFoodLogMutation();

  const foods = useMemo(() => {
    const list = overviewQuery.data?.foods;
    return list && list.length > 0 ? list : DEFAULT_VIETNAMESE_FOODS;
  }, [overviewQuery.data?.foods]);

  const schema = useMemo(() => createFoodLogSchema(looseT(tv)), [tv]);
  const form = useForm<FoodLogFormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      food_id: foods[0]?.id ?? 1,
      meal_type: 'BREAKFAST',
      qty: foods[0]?.unit === '100g' ? 100 : 1,
    },
  });

  const watchedFoodId = form.watch('food_id');
  const watchedQty = form.watch('qty');
  const selectedFood = useMemo(
    () => foods.find(f => f.id === Number(watchedFoodId)),
    [foods, watchedFoodId],
  );

  const previewMacros = useMemo(() => {
    if (!selectedFood || !watchedQty || watchedQty <= 0) return null;
    const isPer100g = selectedFood.unit.toLowerCase().includes('100');
    const ratio = isPer100g && watchedQty >= 10 ? watchedQty / 100 : watchedQty;
    const p = selectedFood.protein_g ?? selectedFood.protein ?? 0;
    const c = selectedFood.carbs_g ?? selectedFood.carbs ?? 0;
    const f = selectedFood.fat_g ?? selectedFood.fat ?? 0;
    return {
      calories: Math.round(selectedFood.calories * ratio),
      protein: Math.round(p * ratio * 10) / 10,
      carbs: Math.round(c * ratio * 10) / 10,
      fat: Math.round(f * ratio * 10) / 10,
    };
  }, [selectedFood, watchedQty]);

  if (overviewQuery.isLoading && logsQuery.isLoading) {
    return (
      <div className="grid gap-4">
        <Skeleton className="h-10 w-64" />
        <Skeleton className="h-36 rounded-2xl" />
        <Skeleton className="h-56 rounded-2xl" />
        <Skeleton className="h-48 rounded-2xl" />
      </div>
    );
  }

  if (overviewQuery.isError && logsQuery.isError) {
    return (
      <Card>
        <LoadError
          error={overviewQuery.error ?? logsQuery.error}
          onRetry={() => {
            void overviewQuery.refetch();
            void logsQuery.refetch();
          }}
        />
      </Card>
    );
  }

  const mealPlans = overviewQuery.data?.meal_plans ?? overviewQuery.data?.plans ?? [];
  const recommendedCalories =
    overviewQuery.data?.recommended_calories || mealPlans[0]?.target_calories || 2000;

  const rawLogs = logsQuery.data;
  const items: FoodLogItem[] = Array.isArray(rawLogs)
    ? rawLogs
    : (rawLogs?.items ?? rawLogs?.logs ?? []);

  const sumCal = items.reduce((s, i) => s + (i.calories ?? 0), 0);
  const sumP = items.reduce((s, i) => s + (i.protein_g ?? i.protein ?? 0), 0);
  const sumC = items.reduce((s, i) => s + (i.carbs_g ?? i.carbs ?? 0), 0);
  const sumF = items.reduce((s, i) => s + (i.fat_g ?? i.fat ?? 0), 0);

  const totalCalories = Math.round(
    (!Array.isArray(rawLogs) && (rawLogs?.total_calories ?? rawLogs?.totals?.calories)) || sumCal,
  );
  const totalProtein =
    Math.round(
      ((!Array.isArray(rawLogs) &&
        (rawLogs?.total_protein_g ?? rawLogs?.total_protein ?? rawLogs?.totals?.protein)) ||
        sumP) *
        10,
    ) / 10;
  const totalCarbs =
    Math.round(
      ((!Array.isArray(rawLogs) &&
        (rawLogs?.total_carbs_g ?? rawLogs?.total_carbs ?? rawLogs?.totals?.carbs)) ||
        sumC) *
        10,
    ) / 10;
  const totalFat =
    Math.round(
      ((!Array.isArray(rawLogs) &&
        (rawLogs?.total_fat_g ?? rawLogs?.total_fat ?? rawLogs?.totals?.fat)) ||
        sumF) *
        10,
    ) / 10;

  const caloriePercent = Math.min(
    Math.round((totalCalories / Math.max(recommendedCalories, 1)) * 100),
    100,
  );

  const onSubmit = form.handleSubmit(async values => {
    try {
      await createLogMutation.mutateAsync({
        date: selectedDate,
        eaten_at: selectedDate,
        meal: values.meal_type,
        meal_type: values.meal_type,
        food_id: values.food_id,
        qty: values.qty,
      });
      toast.success(t('addFoodSuccess'));
    } catch (error) {
      toastError(error);
    }
  });

  return (
    <div className="grid gap-6">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <PortalPageHeader title={t('title')} subtitle={t('subtitle')} />
        <div className="flex items-center gap-2">
          <CalendarDaysIcon className="size-5 text-secondary" />
          <Input
            type="date"
            value={selectedDate}
            onChange={e => setSelectedDate(e.target.value || dayjs().format('YYYY-MM-DD'))}
            aria-label={t('dateLabel')}
            className="w-44"
          />
        </div>
      </div>

      {/* Tổng quan Calo & Macro trong ngày */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
        <Card className="sm:col-span-2">
          <div className="flex items-center justify-between gap-2">
            <span className="text-caption font-medium text-secondary">
              {t('caloriesConsumed')} / {t('recommendedCalories')}
            </span>
            <FireIcon className="size-5 text-brand-600" />
          </div>
          <div className="my-2 flex items-baseline gap-2">
            <span className="text-h2 font-bold text-primary">
              {t('kcalUnit', { value: formatNumber(totalCalories) })}
            </span>
            <span className="text-body text-secondary">
              / {t('kcalUnit', { value: formatNumber(recommendedCalories) })}
            </span>
          </div>
          <Progress value={caloriePercent} />
        </Card>

        <Card className="flex flex-col justify-between">
          <span className="text-caption text-secondary">{t('proteinLabel')}</span>
          <p className="mt-2 text-h3 font-bold text-brand-600">
            {t('gramUnit', { value: formatNumber(totalProtein) })}
          </p>
        </Card>

        <Card className="flex flex-col justify-between">
          <span className="text-caption text-secondary">{t('carbsLabel')}</span>
          <p className="mt-2 text-h3 font-bold text-info">
            {t('gramUnit', { value: formatNumber(totalCarbs) })}
          </p>
        </Card>

        <Card className="flex flex-col justify-between">
          <span className="text-caption text-secondary">{t('fatLabel')}</span>
          <p className="mt-2 text-h3 font-bold text-warning">
            {t('gramUnit', { value: formatNumber(totalFat) })}
          </p>
        </Card>
      </div>

      {/* Form ghi nhận bữa ăn */}
      <Card>
        <CardHeader>
          <CardTitle>{t('logMealTitle')}</CardTitle>
          <p className="text-caption text-secondary">{t('logMealDesc')}</p>
        </CardHeader>
        <Form {...form}>
          <form onSubmit={onSubmit} className="grid gap-4" noValidate>
            <div className="grid gap-4 sm:grid-cols-3">
              <FormField
                control={form.control}
                name="meal_type"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('mealTypeLabel')}</FormLabel>
                    <Select value={field.value} onValueChange={field.onChange}>
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        {MEAL_TYPES.map(mt => (
                          <SelectItem key={mt} value={mt}>
                            {t(`meals.${mt}`)}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="food_id"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('foodLabel')}</FormLabel>
                    <Select
                      value={field.value ? String(field.value) : undefined}
                      onValueChange={val => {
                        const id = Number(val);
                        field.onChange(id);
                        const found = foods.find(f => f.id === id);
                        if (found) {
                          form.setValue('qty', found.unit.includes('100') ? 100 : 1);
                        }
                      }}
                    >
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue placeholder={t('foodPlaceholder')} />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        {foods.map(food => (
                          <SelectItem key={food.id} value={String(food.id)}>
                            {food.name} ({food.calories} kcal / {food.unit})
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="qty"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t('qtyLabel')}
                      {selectedFood ? ` (${selectedFood.unit})` : ''}
                    </FormLabel>
                    <FormControl>
                      <Input
                        type="number"
                        step="any"
                        min={0.1}
                        value={field.value ?? ''}
                        onChange={e => field.onChange(Number(e.target.value))}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>

            <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl bg-canvas p-3">
              {previewMacros ? (
                <div className="flex flex-wrap items-center gap-2 text-caption text-secondary">
                  <Badge variant="brand">
                    {t('kcalUnit', { value: formatNumber(previewMacros.calories) })}
                  </Badge>
                  <span>P: {t('gramUnit', { value: previewMacros.protein })}</span>
                  <span>· C: {t('gramUnit', { value: previewMacros.carbs })}</span>
                  <span>· F: {t('gramUnit', { value: previewMacros.fat })}</span>
                </div>
              ) : (
                <span />
              )}
              <Button type="submit" size="sm" disabled={createLogMutation.isPending}>
                <PlusCircleIcon />
                {createLogMutation.isPending ? t('addingFood') : t('addFoodBtn')}
              </Button>
            </div>
          </form>
        </Form>
      </Card>

      {/* Nhật ký ăn uống trong ngày */}
      <Card>
        <CardHeader>
          <CardTitle>{t('dailyLogsTitle', { date: formatDate(selectedDate) })}</CardTitle>
        </CardHeader>
        {logsQuery.isLoading ? (
          <div className="grid gap-2">
            <Skeleton className="h-12 rounded-xl" />
            <Skeleton className="h-12 rounded-xl" />
          </div>
        ) : items.length === 0 ? (
          <EmptyState
            icon={ClipboardDocumentListIcon}
            title={t('emptyLogsTitle')}
            description={t('emptyLogsDesc')}
            className="py-6"
          />
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('mealTypeLabel')}</TableHead>
                <TableHead>{t('foodLabel')}</TableHead>
                <TableHead className="text-right">{t('qtyLabel')}</TableHead>
                <TableHead className="text-right">Calo</TableHead>
                <TableHead className="text-right">P / C / F</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map(item => {
                const mealKey = item.meal_type || item.meal || 'BREAKFAST';
                const pVal = item.protein_g ?? item.protein ?? 0;
                const cVal = item.carbs_g ?? item.carbs ?? 0;
                const fVal = item.fat_g ?? item.fat ?? 0;
                return (
                  <TableRow key={item.id}>
                    <TableCell>
                      <Badge variant="info">{tl(`meals.${mealKey}`)}</Badge>
                    </TableCell>
                    <TableCell className="font-medium text-primary">
                      {item.food_name || `#${item.food_id}`}
                    </TableCell>
                    <TableCell className="text-right text-secondary">
                      {formatNumber(item.qty)} {item.unit || ''}
                    </TableCell>
                    <TableCell className="text-right font-semibold text-brand-600">
                      {t('kcalUnit', { value: formatNumber(Math.round(item.calories ?? 0)) })}
                    </TableCell>
                    <TableCell className="text-right text-caption text-secondary">
                      {Math.round(pVal * 10) / 10}g / {Math.round(cVal * 10) / 10}g /{' '}
                      {Math.round(fVal * 10) / 10}g
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        )}
      </Card>

      {/* Thực đơn do HLV giao */}
      <Card>
        <CardHeader className="flex-row items-center justify-between gap-2">
          <CardTitle>{t('mealPlansTitle')}</CardTitle>
          <SparklesIcon className="size-5 text-brand-600" />
        </CardHeader>
        {mealPlans.length === 0 ? (
          <EmptyState
            icon={SparklesIcon}
            title={t('emptyMealPlansTitle')}
            description={t('emptyMealPlansDesc')}
            className="py-6"
          />
        ) : (
          <div className="grid gap-4">
            {mealPlans.map(plan => (
              <div
                key={plan.id}
                className="grid gap-3 rounded-xl border border-line bg-canvas p-4"
              >
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <div className="grid gap-0.5">
                    <p className="text-h4 font-semibold text-primary">{plan.name}</p>
                    {plan.description || plan.goal ? (
                      <p className="text-caption text-secondary">
                        {plan.description || plan.goal}
                      </p>
                    ) : null}
                  </div>
                  <Badge variant="brand">
                    {t('targetCaloriesBadge', {
                      calories: formatNumber(plan.target_calories),
                    })}
                  </Badge>
                </div>
                {plan.items && plan.items.length > 0 ? (
                  <div className="grid gap-2 sm:grid-cols-2">
                    {plan.items.map(entry => {
                      const entryMeal = entry.meal_type || entry.meal || 'BREAKFAST';
                      return (
                        <div
                          key={entry.id}
                          className="flex items-center justify-between gap-2 rounded-lg bg-surface p-3"
                        >
                          <div className="grid gap-0.5">
                            <span className="text-caption font-semibold text-brand-600">
                              {t('dayIndexLabel', { day: entry.day_index })} ·{' '}
                              {tl(`meals.${entryMeal}`)}
                            </span>
                            <span className="text-body font-medium text-primary">
                              {entry.food_name || `#${entry.food_id}`} ({entry.qty})
                            </span>
                          </div>
                          {entry.calories ? (
                            <Badge variant="outline">
                              {t('kcalUnit', { value: formatNumber(entry.calories) })}
                            </Badge>
                          ) : null}
                        </div>
                      );
                    })}
                  </div>
                ) : null}
              </div>
            ))}
          </div>
        )}
      </Card>
    </div>
  );
}
