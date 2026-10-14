import * as React from 'react';
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '@/lib/utils';

const badgeVariants = cva(
  'inline-flex w-fit shrink-0 items-center gap-1 rounded-2xl px-2.5 py-0.5 text-caption font-medium whitespace-nowrap [&_svg]:size-3.5',
  {
    variants: {
      variant: {
        neutral: 'bg-subtle text-secondary',
        brand: 'bg-brand-50 text-brand-700',
        success: 'bg-success-50 text-success',
        warning: 'bg-warning-50 text-warning',
        danger: 'bg-danger-50 text-danger',
        info: 'bg-info-50 text-info',
        outline: 'border border-line text-secondary',
      },
    },
    defaultVariants: { variant: 'neutral' },
  },
);

function Badge({
  className,
  variant,
  ...props
}: React.ComponentProps<'span'> & VariantProps<typeof badgeVariants>) {
  return (
    <span data-slot="badge" className={cn(badgeVariants({ variant }), className)} {...props} />
  );
}

export { Badge, badgeVariants };
