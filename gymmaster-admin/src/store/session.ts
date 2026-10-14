import { configureSession } from '@/utils/api';
import { feedback } from '@/utils/feedback';
import { queryClient } from '@/queryClient';
import { store } from './index';
import { clearAuth, setUser } from './authSlice';

/** Nối interceptor axios với Redux store (gọi một lần ở main.tsx). */
export function setupSession(): void {
  configureSession({
    getBranchId: () => store.getState().auth.workingBranchId,
    onTokensRefreshed: (data) => {
      store.dispatch(setUser(data.user));
    },
    onSessionExpired: () => {
      const { user } = store.getState().auth;
      if (!user) return; // đã xử lý (nhiều request cùng hết phiên)
      feedback.message.warning({
        content: 'Phiên đăng nhập đã hết hạn, vui lòng đăng nhập lại',
        key: 'session-expired',
      });
      store.dispatch(clearAuth());
      queryClient.clear();
      // ProtectedRoute thấy user = null sẽ tự chuyển về /login.
    },
  });
}
