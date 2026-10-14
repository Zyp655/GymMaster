import { lazy, Suspense, useEffect } from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';
import LoadingScreen from '@/components/LoadingScreen';
import ForgotPassword from '@/routes/auth/ForgotPassword';
import Login from '@/routes/auth/Login';
import ResetPassword from '@/routes/auth/ResetPassword';
import { bootstrapAuth } from '@/store/authSlice';
import { useAppDispatch, useAppSelector } from '@/store';
import AppRoutes from './AppRoutes';

const InvoicePrint = lazy(() => import('@/routes/print/InvoicePrint'));

/** Khôi phục phiên (`/auth/me`) rồi điều hướng: `/login`, `/forgot-password`, `/reset-password`, `/app/*`, `/` → `/app/dashboard`. */
export function MainApp() {
  const dispatch = useAppDispatch();
  const initialized = useAppSelector((s) => s.auth.initialized);

  useEffect(() => {
    if (!initialized) void dispatch(bootstrapAuth());
  }, [dispatch, initialized]);

  if (!initialized) return <LoadingScreen tip="Đang khôi phục phiên làm việc..." />;

  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route path="/forgot-password" element={<ForgotPassword />} />
      <Route path="/reset-password" element={<ResetPassword />} />
      <Route path="/app/*" element={<AppRoutes />} />
      <Route
        path="/print/invoices/:id"
        element={
          <Suspense fallback={<LoadingScreen />}>
            <InvoicePrint />
          </Suspense>
        }
      />
      <Route path="/" element={<Navigate to="/app/dashboard" replace />} />
      <Route path="*" element={<Navigate to="/app/dashboard" replace />} />
    </Routes>
  );
}

export default MainApp;
