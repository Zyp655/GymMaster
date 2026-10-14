'use client';

import { useTranslations } from 'next-intl';
import { ArrowPathIcon, ExclamationTriangleIcon } from '@heroicons/react/24/outline';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

/** Nội dung cho error.tsx: lỗi render không mong muốn trong một route segment. */
export function ErrorView({ reset, className }: { reset: () => void; className?: string }) {
  const t = useTranslations('common');
  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center gap-4 px-6 py-16 text-center',
        className,
      )}
    >
      <div className="flex size-14 items-center justify-center rounded-2xl bg-danger-50 text-danger">
        <ExclamationTriangleIcon className="size-7" />
      </div>
      <div className="grid max-w-md gap-2">
        <h1 className="text-h3 font-semibold text-primary">{t('errorPage.title')}</h1>
        <p className="text-body text-secondary">{t('errorPage.description')}</p>
      </div>
      <Button variant="outline" onClick={reset}>
        <ArrowPathIcon />
        {t('actions.retry')}
      </Button>
    </div>
  );
}
