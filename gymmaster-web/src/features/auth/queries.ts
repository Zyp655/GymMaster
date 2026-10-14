'use client';

import { useMutation, useQuery } from '@tanstack/react-query';
import { authApi } from './api';
import type {
  ChangePasswordPayload,
  ForgotPasswordPayload,
  RegisterMemberPayload,
  ResetPasswordPayload,
  VerifyMemberOtpPayload,
} from './types';

export const authKeys = {
  me: ['auth', 'me'] as const,
  publicBranches: ['auth', 'public-branches'] as const,
};

export function useChangePassword() {
  return useMutation({
    mutationFn: (payload: ChangePasswordPayload) => authApi.changePassword(payload),
  });
}

export function useForgotPasswordMutation() {
  return useMutation({
    mutationFn: (payload: ForgotPasswordPayload) => authApi.forgotPassword(payload),
  });
}

export function useResetPasswordMutation() {
  return useMutation({
    mutationFn: (payload: ResetPasswordPayload) => authApi.resetPassword(payload),
  });
}

export function useRegisterMemberMutation() {
  return useMutation({
    mutationFn: (payload: RegisterMemberPayload) => authApi.registerMember(payload),
  });
}

export function useVerifyMemberOtpMutation() {
  return useMutation({
    mutationFn: (payload: VerifyMemberOtpPayload) => authApi.verifyMemberOtp(payload),
  });
}

export function usePublicBranchesQuery() {
  return useQuery({
    queryKey: authKeys.publicBranches,
    queryFn: () => authApi.publicBranches(),
  });
}
