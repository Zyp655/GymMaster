'use client';

import { useMutation } from '@tanstack/react-query';
import { authApi } from './api';
import type { ChangePasswordPayload } from './types';

export const authKeys = {
  me: ['auth', 'me'] as const,
};

export function useChangePassword() {
  return useMutation({
    mutationFn: (payload: ChangePasswordPayload) => authApi.changePassword(payload),
  });
}
