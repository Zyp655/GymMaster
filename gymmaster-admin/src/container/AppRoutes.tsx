import { Navigate, Route, Routes, useLocation } from 'react-router-dom';
import AppLayout from '@/layouts';
import { NotFound } from '@/components/Forbidden';
import { routes } from '@/routes';
import ForgotPassword from '@/routes/auth/ForgotPassword';
import ResetPassword from '@/routes/auth/ResetPassword';
import { useAppSelector } from '@/store';
import ProtectedRoute from './ProtectedRoute';

/** Khu vực `/app/*` và các route xác thực công khai (`/forgot-password`, `/reset-password`). */
export function AppRoutes() {
  const user = useAppSelector((s) => s.auth.user);
  const location = useLocation();

  if (location.pathname === '/forgot-password') return <ForgotPassword />;
  if (location.pathname === '/reset-password') return <ResetPassword />;

  if (!user) {
    return <Navigate to="/login" replace state={{ from: location.pathname + location.search }} />;
  }

  return (
    <Routes>
      <Route path="/forgot-password" element={<ForgotPassword />} />
      <Route path="/reset-password" element={<ResetPassword />} />
      <Route element={<AppLayout />}>
        <Route index element={<Navigate to="dashboard" replace />} />
        {routes.map((r) => (
          <Route
            key={r.path}
            path={r.path}
            element={<ProtectedRoute perm={r.perm}>{r.element}</ProtectedRoute>}
          />
        ))}
        <Route path="*" element={<NotFound />} />
      </Route>
    </Routes>
  );
}

export default AppRoutes;
