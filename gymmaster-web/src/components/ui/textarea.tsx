import * as React from 'react';
import { cn } from '@/lib/utils';

function Textarea({ className, ...props }: React.ComponentProps<'textarea'>) {
  return (
    <textarea
      data-slot="textarea"
      className={cn(
        'flex min-h-20 w-full rounded-xl border border-input bg-surface px-3 py-2 text-body text-primary outline-none placeholder:text-disabled',
        'focus-visible:border-brand-400 focus-visible:ring-2 focus-visible:ring-brand-100',
        'disabled:cursor-not-allowed disabled:bg-subtle aria-invalid:border-danger',
        className,
      )}
      {...props}
    />
  );
}

export { Textarea };
