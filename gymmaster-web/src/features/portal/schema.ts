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

export function createWorkoutLogSchema(tv: Translate) {
  return z.object({
    exercise_id: z.number().int().positive(tv('required')),
    sets: z.number().int().min(1, tv('required')).max(50),
    reps: z.number().int().min(1, tv('required')).max(500),
    weight_kg: z.number().min(0, tv('required')).max(1000),
    note: z.string().trim().max(255, tv('maxLength', { max: 255 })),
  });
}

export type WorkoutLogFormValues = z.infer<ReturnType<typeof createWorkoutLogSchema>>;

export function createMemberReviewSchema(tv: Translate) {
  return z.object({
    rating: z.number().int().min(1, tv('required')).max(5),
    comment: z.string().trim().max(500, tv('maxLength', { max: 500 })),
  });
}

export type MemberReviewFormValues = z.infer<ReturnType<typeof createMemberReviewSchema>>;

export function createSurveyResponseSchema(tv: Translate) {
  return z.object({
    score: z.number().int().min(0, tv('required')).max(10),
    comment: z.string().trim().max(1000, tv('maxLength', { max: 1000 })),
  });
}

export type SurveyResponseFormValues = z.infer<ReturnType<typeof createSurveyResponseSchema>>;

export function createFoodLogSchema(tv: Translate) {
  return z.object({
    food_id: z.number().int().positive(tv('required')),
    qty: z.number().min(0.1, tv('required')).max(5000),
    meal_type: z.enum(['BREAKFAST', 'LUNCH', 'DINNER', 'SNACK']),
  });
}

export type FoodLogFormValues = z.infer<ReturnType<typeof createFoodLogSchema>>;

export function createGuestPassSchema(tv: Translate) {
  return z.object({
    guest_name: z
      .string()
      .trim()
      .min(1, tv('required'))
      .max(150, tv('maxLength', { max: 150 })),
    guest_phone: z
      .string()
      .trim()
      .min(1, tv('required'))
      .refine(v => /^(0|\+84)\d{8,10}$/.test(v.replace(/\s+/g, '')), {
        message: tv('phone'),
      }),
    branch_id: z.number().int().positive(tv('required')),
    valid_date: z.string().min(1, tv('required')),
  });
}

export type GuestPassFormValues = z.infer<ReturnType<typeof createGuestPassSchema>>;

export function createMfaEnableSchema(tv: Translate) {
  return z.object({
    code: z
      .string()
      .trim()
      .regex(/^\d{6}$/, tv('minLength', { min: 6 })),
  });
}

export type MfaEnableFormValues = z.infer<ReturnType<typeof createMfaEnableSchema>>;

