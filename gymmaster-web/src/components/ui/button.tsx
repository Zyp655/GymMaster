import * as React from 'react';
import { Slot } from 'radix-ui';
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '@/lib/utils';

const buttonVariants = cva(
  'inline-flex shrink-0 cursor-pointer items-center justify-center gap-2 rounded-button font-medium whitespace-nowrap transition-colors outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0',
  {
    variants: {
      variant: {
        primary: 'bg-brand-500 text-white hover:bg-brand-600',
        dark: 'bg-ink-900 text-white hover:bg-ink-800',
        outline: 'border border-line-strong bg-surface text-primary hover:bg-subtle',
        soft: 'bg-brand-50 text-brand-700 hover:bg-brand-100',
        ghost: 'text-secondary hover:bg-subtle hover:text-primary',
        danger: 'bg-danger text-white hover:bg-danger/90',
        'danger-ghost': 'text-danger hover:bg-danger-50',
        link: 'h-auto px-0 text-brand-600 underline-offset-4 hover:underline',
      },
      size: {
        sm: 'h-9 px-3 text-body [&_svg]:size-4',
        md: 'h-10 px-4 text-body [&_svg]:size-5',
        lg: 'h-12 px-6 text-body-lg [&_svg]:size-5',
        icon: 'size-9 [&_svg]:size-5',
        'icon-sm': 'size-8 [&_svg]:size-4',
      },
    },
    defaultVariants: { variant: 'primary', size: 'md' },
  },
);

function Button({
  className,
  variant,
  size,
  asChild = false,
  ...props
}: React.ComponentProps<'button'> & VariantProps<typeof buttonVariants> & { asChild?: boolean }) {
  const Comp = asChild ? Slot.Root : 'button';
  return (
    <Comp
      data-slot="button"
      className={cn(buttonVariants({ variant, size, className }))}
      {...props}
    />
  );
}

export { Button, buttonVariants };
