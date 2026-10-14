import * as React from 'react';
import { cn } from '@/lib/utils';

function Input({ className, type, ...props }: React.ComponentProps<'input'>) {
  return (
    <input
      type={type}
      data-slot="input"
      className={cn(
        'flex h-10 w-full min-w-0 rounded-xl border border-input bg-surface px-3 text-body text-primary transition-colors outline-none placeholder:text-disabled',
        'focus-visible:border-brand-400 focus-visible:ring-2 focus-visible:ring-brand-100',
        'disabled:cursor-not-allowed disabled:bg-subtle disabled:text-tertiary',
        'aria-invalid:border-danger aria-invalid:ring-danger-50',
        'file:border-0 file:bg-transparent file:text-body file:font-medium',
        className,
      )}
      {...props}
    />
  );
}

export { Input };
