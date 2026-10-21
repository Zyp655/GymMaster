import { z } from 'zod';
import type { Translate } from '@/lib/utils';

export function createLoginSchema(tv: Translate) {
  return z.object({
    email: z.string().trim().min(1, tv('required')).email(tv('email')),
    password: z.string().min(1, tv('required')),
    /** Chỉ bắt buộc ở bước 2FA – LoginForm tự kiểm tra định dạng. */
    mfa_code: z.string().trim().optional(),
  });
}

/** Ô nhập 2FA: `totp` = mã 6 số (bàn phím số), `recovery` = mã khôi phục chữ + số (bàn phím chữ). */
export type MfaInputMode = 'totp' | 'recovery';

export const MFA_TOTP_LENGTH = 6;
/** XXXX-XXXX = 9 ký tự (backend sinh mã khôi phục dạng HEX in hoa có gạch giữa). */
export const MFA_RECOVERY_MAX_LENGTH = 9;
const MFA_TOTP_PATTERN = /^\d{6}$/;
const MFA_RECOVERY_PATTERN = /^[A-Z0-9]{4}-[A-Z0-9]{4}$/;

/**
 * Chuẩn hóa mã 2FA trước khi gửi: bỏ khoảng trắng; mã khôi phục in hoa và tự thêm gạch nối
 * khi người dùng gõ liền 8 ký tự ("abcd1234" → "ABCD-1234"). Backend so khớp mã khôi phục dạng in hoa có gạch.
 */
export function normalizeMfaCode(value: string | undefined, mode: MfaInputMode): string {
  const compact = (value ?? '').replace(/\s+/g, '');
  if (mode === 'totp') return compact;
  const upper = compact.toUpperCase();
  return /^[A-Z0-9]{8}$/.test(upper) ? `${upper.slice(0, 4)}-${upper.slice(4)}` : upper;
}

/** Kiểm tra mã 2FA (đã chuẩn hóa) theo chế độ nhập. */
export function isValidMfaCode(code: string, mode: MfaInputMode): boolean {
  return mode === 'totp' ? MFA_TOTP_PATTERN.test(code) : MFA_RECOVERY_PATTERN.test(code);
}

export type LoginFormValues = z.infer<ReturnType<typeof createLoginSchema>>;

export const PASSWORD_MIN = 8;
/** bcrypt chỉ dùng 72 byte đầu – backend từ chối mật khẩu dài hơn. */
export const PASSWORD_MAX_BYTES = 72;

/**
 * Chính sách mật khẩu MỚI – khớp `ValidateNewPassword` của gymmaster-api:
 * ≥ 8 ký tự (đếm theo ký tự Unicode), có ít nhất một chữ cái và một chữ số, tối đa 72 byte UTF-8.
 */
export function isStrongPassword(value: string): boolean {
  return [...value].length >= PASSWORD_MIN && /\p{L}/u.test(value) && /\p{Nd}/u.test(value);
}

export function createNewPasswordSchema(tv: Translate, ta: Translate) {
  return z
    .string()
    .min(1, tv('required'))
    .refine(isStrongPassword, { message: ta('passwordPolicy') })
    .refine(v => new TextEncoder().encode(v).length <= PASSWORD_MAX_BYTES, {
      message: tv('maxLength', { max: PASSWORD_MAX_BYTES }),
    });
}

export function createChangePasswordSchema(tv: Translate, ta: Translate) {
  return z
    .object({
      old_password: z.string().min(1, tv('required')),
      new_password: createNewPasswordSchema(tv, ta),
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
      new_password: createNewPasswordSchema(tv, ta),
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
      password: createNewPasswordSchema(tv, ta),
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
