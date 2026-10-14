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

export function createForgotPasswordSchema(tv: Translate) {
  return z.object({
    email: z.string().trim().min(1, tv('required')).email(tv('email')),
  });
}

export type ForgotPasswordFormValues = z.infer<ReturnType<typeof createForgotPasswordSchema>>;

export function createResetPasswordSchema(tv: Translate, ta: Translate) {
  return z
    .object({
      new_password: z
        .string()
        .min(PASSWORD_MIN, tv('minLength', { min: PASSWORD_MIN }))
        .max(72, tv('maxLength', { max: 72 })),
      confirm_password: z.string().min(1, tv('required')),
    })
    .refine(v => v.new_password === v.confirm_password, {
      path: ['confirm_password'],
      message: ta('changePassword.mismatch'),
    });
}

export type ResetPasswordFormValues = z.infer<ReturnType<typeof createResetPasswordSchema>>;

export function createRegisterMemberSchema(tv: Translate, ta: Translate) {
  return z
    .object({
      full_name: z
        .string()
        .trim()
        .min(2, tv('minLength', { min: 2 }))
        .max(150, tv('maxLength', { max: 150 })),
      phone: z
        .string()
        .trim()
        .min(1, tv('required'))
        .refine(v => /^(0|\+84)\d{8,10}$/.test(v.replace(/\s+/g, '')), {
          message: tv('phone'),
        }),
      email: z.string().trim().min(1, tv('required')).email(tv('email')),
      branch_id: z.number().int().positive(tv('required')),
      password: z
        .string()
        .min(PASSWORD_MIN, tv('minLength', { min: PASSWORD_MIN }))
        .max(72, tv('maxLength', { max: 72 })),
      confirm_password: z.string().min(1, tv('required')),
    })
    .refine(v => v.password === v.confirm_password, {
      path: ['confirm_password'],
      message: ta('changePassword.mismatch'),
    });
}

export type RegisterMemberFormValues = z.infer<ReturnType<typeof createRegisterMemberSchema>>;

export function createVerifyOtpSchema(tv: Translate) {
  return z.object({
    otp: z
      .string()
      .trim()
      .regex(/^\d{6}$/, tv('minLength', { min: 6 })),
  });
}

export type VerifyOtpFormValues = z.infer<ReturnType<typeof createVerifyOtpSchema>>;
