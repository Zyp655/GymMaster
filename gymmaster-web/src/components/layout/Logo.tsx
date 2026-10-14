import { cn } from '@/lib/utils';

/** Logo GymMaster: biểu tượng tạ đơn + chữ. */
export function Logo({
  className,
  showText = true,
  tone = 'dark',
}: {
  className?: string;
  showText?: boolean;
  /** dark = chữ đen trên nền sáng; light = chữ trắng trên nền tối. */
  tone?: 'dark' | 'light';
}) {
  return (
    <span className={cn('inline-flex items-center gap-2.5', className)}>
      <span className="flex size-9 shrink-0 items-center justify-center rounded-xl bg-brand-500 text-white">
        <svg viewBox="0 0 24 24" fill="none" className="size-5" aria-hidden="true">
          <path
            d="M6.5 6.5v11M17.5 6.5v11M3.5 9v6M20.5 9v6M6.5 12h11"
            stroke="currentColor"
            strokeWidth="2.2"
            strokeLinecap="round"
          />
        </svg>
      </span>
      {showText ? (
        <span
          className={cn(
            'text-h4 font-semibold tracking-tight',
            tone === 'light' ? 'text-white' : 'text-primary',
          )}
        >
          Gym<span className="text-brand-500">Master</span>
        </span>
      ) : null}
    </span>
  );
}
