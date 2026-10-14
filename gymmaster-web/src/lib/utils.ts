import { clsx, type ClassValue } from 'clsx';
import { extendTailwindMerge } from 'tailwind-merge';

// Đăng ký các token tùy chỉnh để twMerge không gộp nhầm text-h1 (cỡ chữ) với text-primary (màu chữ).
const twMerge = extendTailwindMerge({
  extend: {
    classGroups: {
      'font-size': [{ text: ['display', 'h1', 'h2', 'h3', 'h4', 'body-lg', 'body', 'caption'] }],
      rounded: [{ rounded: ['button'] }],
    },
  },
});

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export type Translate = (key: string, values?: Record<string, string | number>) => string;

/**
 * next-intl sinh kiểu chặt cho key; các schema zod dùng chung nhận hàm dịch dạng string.
 * Helper này nới kiểu để truyền `useTranslations(...)` vào factory schema.
 */
export function looseT(t: unknown): Translate {
  return t as Translate;
}
