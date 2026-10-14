'use client';

import { create } from 'zustand';
import { createJSONStorage, persist } from 'zustand/middleware';

interface UiState {
  sidebarCollapsed: boolean;
  toggleSidebar: () => void;
}

/** Trạng thái giao diện thuần client (không phải dữ liệu server → không dùng TanStack Query). */
export const useUiStore = create<UiState>()(
  persist(
    set => ({
      sidebarCollapsed: false,
      toggleSidebar: () => set(s => ({ sidebarCollapsed: !s.sidebarCollapsed })),
    }),
    {
      name: 'gm-ui',
      // Tự rehydrate sau khi mount (AdminShell) để HTML server và client khớp nhau.
      skipHydration: true,
      storage: createJSONStorage(() => {
        try {
          return window.localStorage;
        } catch {
          return {
            getItem: () => null,
            setItem: () => undefined,
            removeItem: () => undefined,
          };
        }
      }),
    },
  ),
);
