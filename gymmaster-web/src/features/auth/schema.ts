import { z } from 'zod';
import type { Translate } from '@/lib/utils';

export function createLoginSchema(tv: Translate) {
  return z.object({
    email: z.string().trim().min(1, tv('required')).email(tv('email')),
    password: z.string().min(1, tv('required')),
  });
}

export type LoginFormValues = z.infer<ReturnType<typeof createLoginSchema>>;

export const PASSWORD_MIN = 8;

export function createChangePasswordSchema(tv: Translate, ta: Translate) {
  return z
    .object({
      old_password: z.string().min(1, tv('required')),
      new_password: z
        .string()
        .min(PASSWORD_MIN, tv('minLength', { min: PASSWORD_MIN }))
        .max(72, tv('maxLength', { max: 72 })),
      confirm_password: z.string().min(1, tv('required')),
    })
    .refine(v => v.new_password === v.confirm_password, {
      path: ['confirm_password'],
      message: ta('changePassword.mismatch'),
    })
    .refine(v => v.new_password !== v.old_password, {
      path: ['new_password'],
      message: ta('changePassword.sameAsOld'),
    });
}

export type ChangePasswordFormValues = z.infer<ReturnType<typeof createChangePasswordSchema>>;
