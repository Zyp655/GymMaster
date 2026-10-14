import { useEffect, useState } from 'react';
import { Alert, Button, Form, Input, Space } from 'antd';
import { ArrowLeftOutlined, MailOutlined } from '@ant-design/icons';
import { Link } from 'react-router-dom';
import authApi from '@/api/auth';
import { getErrorMessage } from '@/utils/errors';

interface ForgotValues {
  email: string;
}

/** Trang quên mật khẩu cho nhân viên (/forgot-password). */
export default function ForgotPassword() {
  const [loading, setLoading] = useState(false);
  const [sentEmail, setSentEmail] = useState<string | null>(null);
  const [devToken, setDevToken] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    document.title = 'Quên mật khẩu – GymMaster Admin';
  }, []);

  const onFinish = async (values: ForgotValues) => {
    setLoading(true);
    setError(null);
    try {
      const res = await authApi.forgotPassword({
        email: values.email.trim(),
        portal: 'ADMIN',
      });
      setSentEmail(values.email.trim());
      if (res && typeof res === 'object' && 'reset_token' in res && res.reset_token) {
        setDevToken(res.reset_token);
      }
    } catch (err) {
      setError(getErrorMessage(err, 'Không thể gửi yêu cầu đặt lại mật khẩu, vui lòng thử lại'));
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
          <h1 className="gm-login-title">Quên mật khẩu</h1>
          <p className="gm-login-sub">
            Nhập email nhân viên để nhận liên kết đặt lại mật khẩu
          </p>

          {error && (
            <Alert
              type="error"
              showIcon
              message={error}
              style={{ marginBottom: 20, borderRadius: 10 }}
            />
          )}

          {sentEmail ? (
            <Space direction="vertical" size={16} style={{ width: '100%' }}>
              <Alert
                type="success"
                showIcon
                message="Yêu cầu đã được ghi nhận"
                description={`Nếu tài khoản ${sentEmail} tồn tại trong hệ thống, liên kết đặt lại mật khẩu (hiệu lực 30 phút) đã được gửi tới hộp thư của bạn.`}
                style={{ borderRadius: 10 }}
              />
              {devToken && (
                <Alert
                  type="info"
                  showIcon
                  message="Liên kết đặt lại mật khẩu (Dev)"
                  description={
                    <Link to={`/reset-password?token=${encodeURIComponent(devToken)}`}>
                      Mở trang đặt lại mật khẩu ngay
                    </Link>
                  }
                  style={{ borderRadius: 10 }}
                />
              )}
              <Button block size="large" onClick={() => setSentEmail(null)}>
                Gửi lại với email khác
              </Button>
            </Space>
          ) : (
            <Form<ForgotValues>
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
                  placeholder="Email công việc"
                  autoComplete="email"
                  autoFocus
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
                  Gửi liên kết đặt lại mật khẩu
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
