import { z } from 'zod';
import type { Translate } from '@/lib/utils';

export const ROLE_CODE_REGEX = /^[A-Z0-9_]{2,30}$/;

export function createRoleSchema(tv: Translate, tr: Translate) {
  return z.object({
    code: z
      .string()
      .trim()
      .min(1, tv('required'))
      .transform(v => v.toUpperCase())
      .pipe(z.string().regex(ROLE_CODE_REGEX, tr('validation.codeFormat'))),
    name: z
      .string()
      .trim()
      .min(1, tv('required'))
      .max(100, tv('maxLength', { max: 100 })),
    description: z
      .string()
      .trim()
      .max(255, tv('maxLength', { max: 255 })),
    permissions: z.array(z.string()),
  });
}

export type RoleFormInput = z.input<ReturnType<typeof createRoleSchema>>;
export type RoleFormValues = z.output<ReturnType<typeof createRoleSchema>>;
