'use client';

import { useTranslations } from 'next-intl';
import { Badge } from '@/components/ui/badge';

export type ActiveStatus = 'ACTIVE' | 'INACTIVE';

export function StatusBadge({ status }: { status: ActiveStatus }) {
  const t = useTranslations('common.status');
  return (
    <Badge variant={status === 'ACTIVE' ? 'success' : 'neutral'}>
      <span
        className={
          status === 'ACTIVE'
            ? 'size-1.5 rounded-full bg-success'
            : 'size-1.5 rounded-full bg-tertiary'
        }
      />
      {t(status)}
    </Badge>
  );
}
