'use client';

import { useTranslations } from 'next-intl';
import { ArrowPathIcon, ExclamationTriangleIcon } from '@heroicons/react/24/outline';
import { Button } from '@/components/ui/button';
import { useApiErrorMessage } from '@/hooks/use-api-error';
import { cn } from '@/lib/utils';

export function LoadError({
  error,
  onRetry,
  className,
}: {
  error?: unknown;
  onRetry?: () => void;
  className?: string;
}) {
  const t = useTranslations('common');
  const getMessage = useApiErrorMessage();
  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center gap-3 px-6 py-12 text-center',
        className,
      )}
    >
      <div className="flex size-12 items-center justify-center rounded-2xl bg-danger-50 text-danger">
        <ExclamationTriangleIcon className="size-6" />
      </div>
      <div className="grid max-w-sm gap-1">
        <p className="text-h4 font-semibold text-primary">{t('states.errorTitle')}</p>
        <p className="text-body text-secondary">
          {error ? getMessage(error) : t('states.errorDescription')}
        </p>
      </div>
      {onRetry ? (
        <Button variant="outline" size="sm" onClick={onRetry}>
          <ArrowPathIcon />
          {t('actions.retry')}
        </Button>
      ) : null}
    </div>
  );
}
