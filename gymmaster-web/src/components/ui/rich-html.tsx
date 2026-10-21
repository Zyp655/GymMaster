'use client';

import { useMemo, useSyncExternalStore } from 'react';
import { looksLikeHtml, sanitizeHtml, stripTags } from '@/lib/sanitize-html';
import { cn } from '@/lib/utils';

/**
 * Hiển thị HTML do API dựng sẵn (đã lọc bằng allowlist) với kiểu chữ dạng văn bản.
 * Chuỗi không chứa thẻ HTML được hiển thị như văn bản thường (giữ xuống dòng).
 */
const noopSubscribe = () => () => {};

/** true sau khi hydrate xong (server & lần render hydrate đầu: false) – tránh lệch HTML. */
function useHydrated(): boolean {
  return useSyncExternalStore(
    noopSubscribe,
    () => true,
    () => false,
  );
}

export function RichHtml({ html, className }: { html: string; className?: string }) {
  const isHtml = looksLikeHtml(html);
  const hydrated = useHydrated();
  // Server không có DOMParser → render bản chữ thuần; chỉ thay bằng HTML đã lọc SAU khi hydrate
  // để HTML server/client khớp nhau (trang SSR như /blog/[slug] không bị hydration error).
  const safe = useMemo(
    () => (isHtml && hydrated ? sanitizeHtml(html) : null),
    [html, isHtml, hydrated],
  );

  if (!isHtml || safe === null) {
    return (
      <div className={cn('whitespace-pre-line', className)}>{isHtml ? stripTags(html) : html}</div>
    );
  }

  return (
    <div
      className={cn(
        'break-words',
        '[&_h1]:mb-2 [&_h1]:text-h3 [&_h1]:font-semibold [&_h1]:text-primary',
        '[&_h2]:mt-3 [&_h2]:mb-2 [&_h2]:text-h4 [&_h2]:font-semibold [&_h2]:text-primary',
        '[&_h3]:mt-3 [&_h3]:mb-1.5 [&_h3]:text-body-lg [&_h3]:font-semibold [&_h3]:text-primary',
        '[&_h4]:mt-2 [&_h4]:mb-1 [&_h4]:font-semibold [&_h4]:text-primary',
        '[&_b]:font-semibold [&_p]:mb-2 [&_p:last-child]:mb-0 [&_strong]:font-semibold [&_strong]:text-primary',
        '[&_li]:mb-1 [&_ol]:mb-2 [&_ol]:list-decimal [&_ol]:pl-5 [&_ul]:mb-2 [&_ul]:list-disc [&_ul]:pl-5',
        '[&_a]:text-brand-600 [&_a]:underline [&_hr]:my-3 [&_hr]:border-line',
        '[&_blockquote]:border-l-2 [&_blockquote]:border-line [&_blockquote]:pl-3 [&_blockquote]:italic',
        '[&_table]:my-2 [&_table]:w-full [&_table]:border-collapse [&_td]:border [&_td]:border-line [&_td]:px-2 [&_td]:py-1 [&_th]:border [&_th]:border-line [&_th]:bg-subtle [&_th]:px-2 [&_th]:py-1 [&_th]:text-left',
        '[&_img]:h-auto [&_img]:max-w-full',
        className,
      )}
      dangerouslySetInnerHTML={{ __html: safe }}
    />
  );
}
