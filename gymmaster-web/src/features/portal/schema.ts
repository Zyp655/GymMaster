import { z } from 'zod';
import type { Translate } from '@/lib/utils';

export function createProfileSchema(tv: Translate) {
  return z.object({
    email: z
      .string()
      .trim()
      .refine(v => v === '' || z.string().email().safeParse(v).success, {
        message: tv('email'),
      }),
    address: z
      .string()
      .trim()
      .max(255, tv('maxLength', { max: 255 })),
    emergency_contact_name: z
      .string()
      .trim()
      .max(100, tv('maxLength', { max: 100 })),
    emergency_contact_phone: z
      .string()
      .trim()
      .refine(v => v === '' || /^(0|\+84)\d{8,10}$/.test(v.replace(/\s+/g, '')), {
        message: tv('phone'),
      }),
    avatar_url: z.string().trim(),
  });
}

export type ProfileFormValues = z.infer<ReturnType<typeof createProfileSchema>>;

export function createFreezeSchema(tv: Translate, tp: Translate) {
  return z
    .object({
      start_date: z.string().min(1, tv('required')),
      end_date: z.string().min(1, tv('required')),
      reason: z
        .string()
        .trim()
        .min(1, tv('required'))
        .max(500, tv('maxLength', { max: 500 })),
    })
    .refine(v => !v.start_date || !v.end_date || v.end_date >= v.start_date, {
      path: ['end_date'],
      message: tp('invalidEndDate'),
    });
}

export type FreezeFormValues = z.infer<ReturnType<typeof createFreezeSchema>>;

export function createBookPTSessionSchema(tv: Translate) {
  return z.object({
    contract_id: z.number().int().positive(),
    start_at: z.string().min(1, tv('required')),
  });
}

export type BookPTSessionFormValues = z.infer<ReturnType<typeof createBookPTSessionSchema>>;

export function createCancelPTSessionSchema(tv: Translate) {
  return z.object({
    reason: z
      .string()
      .trim()
      .max(500, tv('maxLength', { max: 500 })),
  });
}

export type CancelPTSessionFormValues = z.infer<ReturnType<typeof createCancelPTSessionSchema>>;
