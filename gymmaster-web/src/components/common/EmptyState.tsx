import type { ComponentType, ReactNode, SVGProps } from 'react';
import { InboxIcon } from '@heroicons/react/24/outline';
import { cn } from '@/lib/utils';

export function EmptyState({
  title,
  description,
  action,
  icon: Icon = InboxIcon,
  className,
}: {
  title: ReactNode;
  description?: ReactNode;
  action?: ReactNode;
  icon?: ComponentType<SVGProps<SVGSVGElement>>;
  className?: string;
}) {
  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center gap-3 px-6 py-12 text-center',
        className,
      )}
    >
      <div className="flex size-12 items-center justify-center rounded-2xl bg-subtle text-tertiary">
        <Icon className="size-6" />
      </div>
      <div className="grid max-w-sm gap-1">
        <p className="text-h4 font-semibold text-primary">{title}</p>
        {description ? <p className="text-body text-secondary">{description}</p> : null}
      </div>
      {action}
    </div>
  );
}
