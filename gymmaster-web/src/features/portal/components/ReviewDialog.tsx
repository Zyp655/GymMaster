'use client';

import { useEffect, useMemo, useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { StarIcon as StarOutlineIcon } from '@heroicons/react/24/outline';
import { StarIcon as StarSolidIcon } from '@heroicons/react/24/solid';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
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
import { Textarea } from '@/components/ui/textarea';
import { useToastApiError } from '@/hooks/use-api-error';
import { cn, looseT } from '@/lib/utils';
import { useCreateMemberReviewMutation } from '../queries';
import { createMemberReviewSchema, type MemberReviewFormValues } from '../schema';
import type { MemberReviewItem, ReviewTargetType } from '../types';

export interface ReviewTarget {
  targetType: ReviewTargetType;
  targetId: number;
  targetName: string;
}

export function ReviewActionOrBadge({
  reviews,
  targetType,
  targetId,
  onReview,
}: {
  reviews: MemberReviewItem[] | undefined;
  targetType: ReviewTargetType;
  targetId: number;
  onReview: () => void;
}) {
  const t = useTranslations('portal.reviews');
  const existing = reviews?.find(
    r => r.target_type === targetType && r.target_id === targetId,
  );

  if (existing) {
    return (
      <Badge variant="warning" className="gap-1">
        <StarSolidIcon className="size-3.5 text-warning" />
        <span>{t('ratedBadge', { rating: existing.rating })}</span>
      </Badge>
    );
  }

  return (
    <Button variant="outline" size="sm" onClick={onReview}>
      <StarOutlineIcon className="size-4 text-warning" />
      {t('rateBtn')}
    </Button>
  );
}

export function ReviewDialog({
  target,
  onOpenChange,
}: {
  target: ReviewTarget | null;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useTranslations('portal.reviews.dialog');
  const tc = useTranslations('common');
  const tv = useTranslations('common.validation');
  const toastError = useToastApiError();
  const mutation = useCreateMemberReviewMutation();
  const [hoverRating, setHoverRating] = useState<number>(0);

  const schema = useMemo(() => createMemberReviewSchema(looseT(tv)), [tv]);
  const form = useForm<MemberReviewFormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      rating: 5,
      comment: '',
    },
  });

  useEffect(() => {
    if (target) {
      form.reset({ rating: 5, comment: '' });
      setHoverRating(0);
    }
  }, [target, form]);

  const currentRating = form.watch('rating');
  const activeStars = hoverRating || currentRating;

  const onSubmit = form.handleSubmit(async values => {
    if (!target) return;
    try {
      await mutation.mutateAsync({
        target_type: target.targetType,
        target_id: target.targetId,
        rating: values.rating,
        comment: values.comment?.trim() || undefined,
      });
      toast.success(t('success'));
      onOpenChange(false);
    } catch (error) {
      toastError(error);
    }
  });

  return (
    <Dialog open={Boolean(target)} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md" closeLabel={tc('actions.close')}>
        <DialogHeader>
          <DialogTitle>{t('title')}</DialogTitle>
          <DialogDescription>
            {target ? `${target.targetName} — ${t('description')}` : t('description')}
          </DialogDescription>
        </DialogHeader>

        <Form {...form}>
          <form onSubmit={onSubmit} className="grid gap-4" noValidate>
            <FormField
              control={form.control}
              name="rating"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('ratingLabel')}</FormLabel>
                  <FormControl>
                    <div className="flex items-center gap-2 py-1">
                      {[1, 2, 3, 4, 5].map(star => {
                        const filled = star <= activeStars;
                        const StarComp = filled ? StarSolidIcon : StarOutlineIcon;
                        return (
                          <button
                            key={star}
                            type="button"
                            onClick={() => field.onChange(star)}
                            onMouseEnter={() => setHoverRating(star)}
                            onMouseLeave={() => setHoverRating(0)}
                            aria-label={`${star} sao`}
                            className="cursor-pointer rounded-lg p-1 transition-transform hover:scale-110 focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none"
                          >
                            <StarComp
                              className={cn(
                                'size-8',
                                filled ? 'text-warning' : 'text-tertiary',
                              )}
                            />
                          </button>
                        );
                      })}
                      <span className="ml-2 text-body font-semibold text-primary">
                        {currentRating}/5
                      </span>
                    </div>
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="comment"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('commentLabel')}</FormLabel>
                  <FormControl>
                    <Textarea {...field} placeholder={t('commentPlaceholder')} rows={3} />
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
