import { Navigate, Route, Routes, useLocation } from 'react-router-dom';
import AppLayout from '@/layouts';
import { NotFound } from '@/components/Forbidden';
import { routes } from '@/routes';
import { useAppSelector } from '@/store';
import ProtectedRoute from './ProtectedRoute';

/** Khu vực `/app/*`: yêu cầu đăng nhập, bọc AppLayout, kiểm tra quyền từng route. */
export function AppRoutes() {
  const user = useAppSelector((s) => s.auth.user);
  const location = useLocation();

  if (!user) {
    return <Navigate to="/login" replace state={{ from: location.pathname + location.search }} />;
  }

  return (
    <Routes>
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
