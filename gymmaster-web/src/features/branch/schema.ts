import { z } from 'zod';
import type { Translate } from '@/lib/utils';

export const BRANCH_CODE_REGEX = /^[A-Z0-9_-]{2,20}$/;
export const TIME_REGEX = /^([01]\d|2[0-3]):[0-5]\d$/;
export const PHONE_REGEX = /^(\+?84|0)\d{8,10}$/;

/**
 * Schema form chi nhánh – khớp validate phía backend (docs/api/sprint-01...).
 * Nhận hàm dịch để thông báo lỗi là tiếng Việt từ messages/vi.
 */
export function createBranchSchema(tv: Translate, tb: Translate) {
  return z
    .object({
      code: z
        .string()
        .trim()
        .min(1, tv('required'))
        .transform(v => v.toUpperCase())
        .pipe(z.string().regex(BRANCH_CODE_REGEX, tb('validation.codeFormat'))),
      name: z
        .string()
        .trim()
        .min(1, tv('required'))
        .max(150, tv('maxLength', { max: 150 })),
      address: z
        .string()
        .trim()
        .max(255, tv('maxLength', { max: 255 })),
      phone: z
        .string()
        .trim()
        .refine(v => v === '' || PHONE_REGEX.test(v.replace(/[\s.]/g, '')), tv('phone')),
      email: z
        .string()
        .trim()
        .refine(v => v === '' || z.string().email().safeParse(v).success, tv('email')),
      open_time: z.string().regex(TIME_REGEX, tv('time')),
      close_time: z.string().regex(TIME_REGEX, tv('time')),
      status: z.enum(['ACTIVE', 'INACTIVE']),
    })
    .refine(v => v.close_time > v.open_time, {
      path: ['close_time'],
      message: tb('validation.closeAfterOpen'),
    });
}

export type BranchFormInput = z.input<ReturnType<typeof createBranchSchema>>;
export type BranchFormValues = z.output<ReturnType<typeof createBranchSchema>>;

export const branchFormDefaults: BranchFormInput = {
  code: '',
  name: '',
  address: '',
  phone: '',
  email: '',
  open_time: '05:30',
  close_time: '22:00',
  status: 'ACTIVE',
};
