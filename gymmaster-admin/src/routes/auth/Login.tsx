import { useEffect, useState } from 'react';
import { Alert, Button, Form, Input } from 'antd';
import { LockOutlined, MailOutlined } from '@ant-design/icons';
import { Link, Navigate, useLocation, useNavigate } from 'react-router-dom';
import { login, MEMBER_ONLY_CODE } from '@/store/authSlice';
import { useAppDispatch, useAppSelector } from '@/store';
import { ErrorCode, isApiError } from '@/utils/api';
import { getErrorMessage } from '@/utils/errors';

interface LoginValues {
  email: string;
  password: string;
}

function loginErrorMessage(error: unknown): string {
  if (isApiError(error)) {
    switch (error.code) {
      case ErrorCode.INVALID_CREDENTIALS:
        return 'Email hoặc mật khẩu không đúng. Vui lòng kiểm tra lại.';
      case ErrorCode.ACCOUNT_LOCKED:
        return 'Tài khoản đã bị khóa. Vui lòng liên hệ quản trị viên để được hỗ trợ.';
      case MEMBER_ONLY_CODE:
        return 'Tài khoản hội viên vui lòng dùng cổng hội viên.';
      default:
        return getErrorMessage(error, 'Đăng nhập thất bại, vui lòng thử lại');
    }
  }
  return 'Đăng nhập thất bại, vui lòng thử lại';
}

/** Trang đăng nhập (phong cách mẫu strongbody_admin: nền gradient + card kính 420px). */
export default function Login() {
  const dispatch = useAppDispatch();
  const navigate = useNavigate();
  const location = useLocation();
  const user = useAppSelector((s) => s.auth.user);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<{ text: string; type: 'error' | 'warning' } | null>(null);

  useEffect(() => {
    document.title = 'Đăng nhập – GymMaster';
  }, []);

  const from = (location.state as { from?: string } | null)?.from;
  if (user)
    return <Navigate to={from && from.startsWith('/app') ? from : '/app/dashboard'} replace />;

  const onFinish = async (values: LoginValues) => {
    setLoading(true);
    setError(null);
    try {
      await dispatch(login(values)).unwrap();
      navigate(from && from.startsWith('/app') ? from : '/app/dashboard', { replace: true });
    } catch (err) {
      const memberOnly = isApiError(err) && err.code === MEMBER_ONLY_CODE;
      setError({ text: loginErrorMessage(err), type: memberOnly ? 'warning' : 'error' });
      setLoading(false);
    }
  };

  return (
    <div className="gm-login-root">
      <div className="gm-login-topbar">
        <div className="gm-login-brand">
          GymMaster <span>Admin</span>
        </div>
      </div>

      <div className="gm-login-shell">
        <div className="gm-login-card">
          <img src="/logo.svg" alt="GymMaster" className="gm-login-logo" />
          <h1 className="gm-login-title">Chào mừng trở lại</h1>
          <p className="gm-login-sub">Đăng nhập để quản lý trung tâm của bạn</p>

          {error && (
            <Alert
              type={error.type}
              showIcon
              message={error.text}
              style={{ marginBottom: 20, borderRadius: 10 }}
            />
          )}

          <Form<LoginValues>
            layout="vertical"
            size="large"
            requiredMark={false}
            onFinish={onFinish}
            onValuesChange={() => error && setError(null)}
          >
            <Form.Item
              name="email"
              rules={[
                { required: true, message: 'Vui lòng nhập email' },
                { type: 'email', message: 'Email không hợp lệ' },
              ]}
            >
              <Input
                prefix={<MailOutlined />}
                placeholder="Email"
                autoComplete="username"
                autoFocus
              />
            </Form.Item>

            <Form.Item
              name="password"
              rules={[{ required: true, message: 'Vui lòng nhập mật khẩu' }]}
              style={{ marginBottom: 12 }}
            >
              <Input.Password
                prefix={<LockOutlined />}
                placeholder="Mật khẩu"
                autoComplete="current-password"
              />
            </Form.Item>

            <div style={{ textAlign: 'right', marginBottom: 18 }}>
              <Link to="/forgot-password" style={{ color: '#fb923c', fontSize: 13 }}>
                Quên mật khẩu?
              </Link>
            </div>

            <Form.Item style={{ marginBottom: 0 }}>
              <Button
                className="gm-login-btn"
                color="orange"
                variant="solid"
                htmlType="submit"
                block
                loading={loading}
              >
                Đăng nhập
              </Button>
            </Form.Item>
          </Form>
        </div>
      </div>

      <div className="gm-login-footer">GymMaster ©2026 – Hệ thống quản lý trung tâm Fitness</div>
    </div>
  );
}
