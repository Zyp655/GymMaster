import { getTranslations } from 'next-intl/server';
import { StarIcon } from '@heroicons/react/24/solid';
import type { PublicTrainer } from '../types';

/** Huy hiệu điểm đánh giá trung bình của HLV. */
export async function TrainerRating({ trainer }: { trainer: PublicTrainer }) {
  const t = await getTranslations('landing.community');
  return (
    <span className="inline-flex shrink-0 items-center gap-1 rounded-full bg-warning-50 px-2.5 py-1 text-caption font-semibold whitespace-nowrap text-warning">
      <StarIcon className="size-3.5" />
      {trainer.rating_count > 0
        ? `${(trainer.rating_avg || 0).toFixed(1)} (${trainer.rating_count})`
        : t('noRating')}
    </span>
  );
}
