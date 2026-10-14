import { useEffect, useState } from 'react';
import { Alert, Button, Form, Input, Space } from 'antd';
import { ArrowLeftOutlined, KeyOutlined, LockOutlined } from '@ant-design/icons';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import authApi from '@/api/auth';
import { getErrorMessage } from '@/utils/errors';

interface ResetValues {
  token?: string;
  new_password: string;
  confirm_password: string;
}

/** Trang đặt lại mật khẩu bằng token (/reset-password?token=...). */
export default function ResetPassword() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const tokenFromUrl = searchParams.get('token')?.trim() || '';
  const [loading, setLoading] = useState(false);
  const [done, setDone] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    document.title = 'Đặt lại mật khẩu – GymMaster Admin';
  }, []);

  const onFinish = async (values: ResetValues) => {
    const token = (tokenFromUrl || values.token || '').trim();
    if (!token) {
      setError('Thiếu mã xác thực (token) đặt lại mật khẩu');
      return;
    }
    setLoading(true);
    setError(null);
    try {
      await authApi.resetPassword({
        token,
        new_password: values.new_password,
      });
      setDone(true);
    } catch (err) {
      setError(
        getErrorMessage(
          err,
          'Liên kết đặt lại mật khẩu không hợp lệ hoặc đã hết hạn. Vui lòng yêu cầu lại.',
        ),
      );
    } finally {
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
          <h1 className="gm-login-title">Đặt lại mật khẩu</h1>
          <p className="gm-login-sub">Thiết lập mật khẩu mới cho tài khoản nhân viên</p>

          {error && (
            <Alert
              type="error"
              showIcon
              message={error}
              style={{ marginBottom: 20, borderRadius: 10 }}
            />
          )}

          {done ? (
            <Space direction="vertical" size={16} style={{ width: '100%' }}>
              <Alert
                type="success"
                showIcon
                message="Đặt lại mật khẩu thành công"
                description="Mật khẩu mới đã được cập nhật và các phiên đăng nhập cũ đã được thu hồi. Vui lòng đăng nhập lại."
                style={{ borderRadius: 10 }}
              />
              <Button
                className="gm-login-btn"
                color="orange"
                variant="solid"
                size="large"
                block
                onClick={() => navigate('/login', { replace: true })}
              >
                Đăng nhập ngay
              </Button>
            </Space>
          ) : (
            <Form<ResetValues>
              layout="vertical"
              size="large"
              requiredMark={false}
              onFinish={onFinish}
              onValuesChange={() => error && setError(null)}
            >
              {!tokenFromUrl && (
                <Form.Item
                  name="token"
                  rules={[{ required: true, message: 'Vui lòng nhập mã xác thực (token)' }]}
                >
                  <Input prefix={<KeyOutlined />} placeholder="Mã xác thực (token)" />
                </Form.Item>
              )}

              <Form.Item
                name="new_password"
                rules={[
                  { required: true, message: 'Vui lòng nhập mật khẩu mới' },
                  { min: 8, message: 'Mật khẩu mới phải từ 8 ký tự trở lên' },
                  { max: 72, message: 'Mật khẩu tối đa 72 ký tự' },
                ]}
              >
                <Input.Password
                  prefix={<LockOutlined />}
                  placeholder="Mật khẩu mới (tối thiểu 8 ký tự)"
                  autoComplete="new-password"
                  autoFocus
                />
              </Form.Item>

              <Form.Item
                name="confirm_password"
                dependencies={['new_password']}
                rules={[
                  { required: true, message: 'Vui lòng xác nhận mật khẩu mới' },
                  ({ getFieldValue }) => ({
                    validator(_, value) {
                      if (!value || getFieldValue('new_password') === value) {
                        return Promise.resolve();
                      }
                      return Promise.reject(new Error('Mật khẩu xác nhận không khớp'));
                    },
                  }),
                ]}
              >
                <Input.Password
                  prefix={<LockOutlined />}
                  placeholder="Nhập lại mật khẩu mới"
                  autoComplete="new-password"
                />
              </Form.Item>

              <Form.Item style={{ marginBottom: 12 }}>
                <Button
                  className="gm-login-btn"
                  color="orange"
                  variant="solid"
                  htmlType="submit"
                  block
                  loading={loading}
                >
                  Cập nhật mật khẩu
                </Button>
              </Form.Item>
            </Form>
          )}

          <div style={{ textAlign: 'center', marginTop: 16 }}>
            <Link to="/login" style={{ color: '#94a3b8', fontSize: 14 }}>
              <ArrowLeftOutlined /> Quay lại đăng nhập
            </Link>
          </div>
        </div>
      </div>

      <div className="gm-login-footer">GymMaster ©2026 – Hệ thống quản lý trung tâm Fitness</div>
    </div>
  );
}
