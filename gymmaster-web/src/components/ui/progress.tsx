import * as React from 'react';
import { cn } from '@/lib/utils';

/** Thanh tiến độ 0–100. Độ rộng động nên dùng style ở primitive (ngoại lệ của DESIGN_SYSTEM). */
function Progress({
  value,
  className,
  indicatorClassName,
  ...props
}: React.ComponentProps<'div'> & { value: number; indicatorClassName?: string }) {
  const pct = Math.min(Math.max(Number.isFinite(value) ? value : 0, 0), 100);
  return (
    <div
      data-slot="progress"
      role="progressbar"
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={Math.round(pct)}
      className={cn('h-2 w-full overflow-hidden rounded-full bg-line', className)}
      {...props}
    >
      <div
        className={cn('h-full rounded-full bg-brand-500 transition-all', indicatorClassName)}
        style={{ width: `${pct}%` }}
      />
    </div>
  );
}

export { Progress };
