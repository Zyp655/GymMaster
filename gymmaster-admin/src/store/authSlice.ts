import { createAsyncThunk, createSlice } from '@reduxjs/toolkit';
import type { PayloadAction } from '@reduxjs/toolkit';
import authApi from '@/api/auth';
import type { AuthUser } from '@/api/types';
import { ROLE } from '@/constants/permissions';
import { ApiError, refreshAccessToken } from '@/utils/api';
import { tokenStore } from '@/utils/token';

const BRANCH_KEY = 'gm_working_branch';

export interface AuthState {
  user: AuthUser | null;
  /** Đã chạy xong bước khôi phục phiên lúc khởi động. */
  initialized: boolean;
  /** Chi nhánh làm việc; `null` = toàn hệ thống (chỉ khi user không bị giới hạn chi nhánh). */
  workingBranchId: number | null;
}

/** Mã lỗi FE: tài khoản chỉ có vai trò hội viên. */
export const MEMBER_ONLY_CODE = 40399;
/** Backend: tài khoản đã bật 2FA, cần gửi kèm `mfa_code`. */
export const MFA_REQUIRED_CODE = 40107;
/** Backend: mã 2FA / mã khôi phục không đúng. */
export const MFA_INVALID_CODE = 40108;

/** Tài khoản có ít nhất một vai trò nhân viên (khác MEMBER). */
export function isStaffUser(user: AuthUser): boolean {
  return (user.roles || []).some((r) => r !== ROLE.MEMBER);
}

function readStoredBranch(): number | null {
  try {
    const raw = localStorage.getItem(BRANCH_KEY);
    const id = raw ? Number(raw) : NaN;
    return Number.isFinite(id) && id > 0 ? id : null;
  } catch {
    return null;
  }
}

function storeBranch(id: number | null) {
  try {
    if (id) localStorage.setItem(BRANCH_KEY, String(id));
    else localStorage.removeItem(BRANCH_KEY);
  } catch {
    /* bỏ qua */
  }
}

/** Chọn chi nhánh làm việc hợp lệ cho user. */
export function resolveWorkingBranch(user: AuthUser, preferred: number | null): number | null {
  const allowed = user.branch_ids || [];
  if (allowed.length === 0) return preferred;
  if (preferred && allowed.includes(preferred)) return preferred;
  return allowed[0];
}

const memberOnlyError = () =>
  new ApiError('Tài khoản hội viên vui lòng dùng cổng hội viên', MEMBER_ONLY_CODE, 403);

/** Khôi phục phiên khi mở app: refresh nếu cần rồi gọi /auth/me. */
let bootstrapStarted = false;

export const bootstrapAuth = createAsyncThunk(
  'auth/bootstrap',
  async () => {
    if (!tokenStore.getRefreshToken() && !tokenStore.getAccessToken()) return null;
    try {
      if (!tokenStore.getAccessToken()) await refreshAccessToken();
      const user = await authApi.me();
      if (!isStaffUser(user)) {
        tokenStore.clear();
        return null;
      }
      return user;
    } catch {
      // Interceptor đã xử lý hết phiên; lỗi mạng thì coi như chưa đăng nhập.
      return null;
    }
  },
  {
    // StrictMode gọi effect 2 lần ở dev → chỉ khôi phục phiên một lần.
    condition: () => {
      if (bootstrapStarted) return false;
      bootstrapStarted = true;
      return true;
    },
  },
);

export const login = createAsyncThunk<
  AuthUser,
  { email: string; password: string; mfaCode?: string }
>(
  'auth/login',
  async ({ email, password, mfaCode }) => {
    const res = await authApi.login(email.trim(), password, mfaCode?.trim() || undefined);
    if (!isStaffUser(res.user)) {
      // Thu hồi refresh token vừa cấp, không lưu phiên.
      await authApi.logout(res.refresh_token).catch(() => undefined);
      throw memberOnlyError();
    }
    tokenStore.setTokens(res.access_token, res.refresh_token);
    return res.user;
  },
  {
    // Giữ nguyên ApiError (createAsyncThunk mặc định serialize lỗi) – component bắt bằng .unwrap()
    serializeError: (err) => err as never,
  },
);

export const logout = createAsyncThunk('auth/logout', async () => {
  const refresh = tokenStore.getRefreshToken();
  // Xóa token cục bộ trước; /auth/logout là public và thu hồi bằng chính refresh token gửi trong body,
  // nên kể cả khi access token đã hết hạn/không có thì phiên vẫn bị thu hồi phía server.
  tokenStore.clear();
  if (refresh) await authApi.logout(refresh).catch(() => undefined);
});

const initialState: AuthState = {
  user: null,
  initialized: false,
  workingBranchId: readStoredBranch(),
};

const authSlice = createSlice({
  name: 'auth',
  initialState,
  reducers: {
    setUser(state, action: PayloadAction<AuthUser>) {
      state.user = action.payload;
      state.workingBranchId = resolveWorkingBranch(action.payload, state.workingBranchId);
      storeBranch(state.workingBranchId);
    },
    clearAuth(state) {
      state.user = null;
      state.initialized = true;
    },
    setWorkingBranch(state, action: PayloadAction<number | null>) {
      state.workingBranchId = state.user
        ? resolveWorkingBranch(state.user, action.payload)
        : action.payload;
      storeBranch(state.workingBranchId);
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(bootstrapAuth.fulfilled, (state, action) => {
        state.initialized = true;
        state.user = action.payload;
        if (action.payload) {
          state.workingBranchId = resolveWorkingBranch(action.payload, state.workingBranchId);
          storeBranch(state.workingBranchId);
        }
      })
      .addCase(bootstrapAuth.rejected, (state) => {
        state.initialized = true;
        state.user = null;
      })
      .addCase(login.fulfilled, (state, action) => {
        state.user = action.payload;
        state.initialized = true;
        state.workingBranchId = resolveWorkingBranch(action.payload, state.workingBranchId);
        storeBranch(state.workingBranchId);
      })
      .addCase(logout.fulfilled, (state) => {
        state.user = null;
      });
  },
});

export const { setUser, clearAuth, setWorkingBranch } = authSlice.actions;
export default authSlice.reducer;
