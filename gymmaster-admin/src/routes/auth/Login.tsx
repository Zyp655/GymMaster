import { useEffect, useState } from 'react';
import { Alert, Button, Form, Input } from 'antd';
import { LockOutlined, MailOutlined, SafetyOutlined } from '@ant-design/icons';
import { Link, Navigate, useLocation, useNavigate } from 'react-router-dom';
import { login, MEMBER_ONLY_CODE, MFA_INVALID_CODE, MFA_REQUIRED_CODE } from '@/store/authSlice';
import { useAppDispatch, useAppSelector } from '@/store';
import { ErrorCode, isApiError } from '@/utils/api';
import { getErrorMessage } from '@/utils/errors';
import { isMfaCodeComplete, mfaInputProps, normalizeMfaCode } from './mfaCode';
import type { MfaCodeMode } from './mfaCode';

interface LoginValues {
  email: string;
  password: string;
  mfaCode?: string;
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
      case MFA_REQUIRED_CODE:
        return 'Tài khoản đã bật xác thực 2 lớp. Nhập mã 6 số từ ứng dụng xác thực (hoặc mã khôi phục).';
      case MFA_INVALID_CODE:
        // Ưu tiên thông điệp server (có thể kèm lý do cụ thể).
        return getErrorMessage(error, 'Mã xác thực 2 lớp hoặc mã khôi phục không đúng.');
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
  /** Bước 2: server yêu cầu mã 2FA (40107) → hiện ô nhập mã, gửi lại kèm email/mật khẩu. */
  const [mfaStep, setMfaStep] = useState(false);
  /** Mã 6 số (bàn phím số) hoặc mã khôi phục chữ + số (bàn phím chữ trên điện thoại). */
  const [mfaMode, setMfaMode] = useState<MfaCodeMode>('totp');
  const [form] = Form.useForm<LoginValues>();

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
      const code = isApiError(err) ? err.code : 0;
      if (code === MFA_REQUIRED_CODE) setMfaStep(true);
      const info = code === MEMBER_ONLY_CODE || code === MFA_REQUIRED_CODE;
      setError({ text: loginErrorMessage(err), type: info ? 'warning' : 'error' });
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
            form={form}
            layout="vertical"
            size="large"
            requiredMark={false}
            onFinish={onFinish}
            onValuesChange={(changed: Partial<LoginValues>) => {
              // Đổi email → bỏ bước 2FA (tài khoản khác có thể chưa bật 2FA).
              if ('email' in changed) {
                setMfaStep(false);
                setMfaMode('totp');
              }
              if (error) setError(null);
            }}
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

            {mfaStep && (
              <Form.Item
                name="mfaCode"
                normalize={(v?: string) => normalizeMfaCode(v ?? '', mfaMode)}
                rules={[
                  { required: true, message: 'Vui lòng nhập mã xác thực 2 lớp' },
                  {
                    validator: (_, v?: string) =>
                      !v || isMfaCodeComplete(v, mfaMode)
                        ? Promise.resolve()
                        : Promise.reject(
                            new Error(
                              mfaMode === 'totp'
                                ? 'Mã xác thực gồm 6 chữ số'
                                : 'Mã khôi phục có dạng XXXX-XXXX (chữ và số)',
                            ),
                          ),
                  },
                ]}
                extra={
                  <Button
                    type="link"
                    size="small"
                    style={{ padding: 0, color: '#fb923c' }}
                    onClick={() => {
                      setMfaMode((m) => (m === 'totp' ? 'recovery' : 'totp'));
                      form.setFieldValue('mfaCode', '');
                      form.setFields([{ name: 'mfaCode', errors: [] }]);
                    }}
                  >
                    {mfaMode === 'totp' ? 'Dùng mã khôi phục' : 'Dùng mã 6 số từ ứng dụng'}
                  </Button>
                }
                style={{ marginTop: 12, marginBottom: 12 }}
              >
                <Input
                  // Đổi key theo chế độ → remount để bàn phím điện thoại đổi theo inputMode.
                  key={mfaMode}
                  prefix={<SafetyOutlined />}
                  {...mfaInputProps(mfaMode)}
                  autoFocus
                />
              </Form.Item>
            )}

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
                {mfaStep ? 'Xác thực & đăng nhập' : 'Đăng nhập'}
              </Button>
            </Form.Item>
          </Form>
        </div>
      </div>

      <div className="gm-login-footer">GymMaster ©2026 – Hệ thống quản lý trung tâm Fitness</div>
    </div>
  );
}
