import { Button, Result } from 'antd';
import { useNavigate } from 'react-router-dom';

/** Trang 403 – đã đăng nhập nhưng thiếu quyền. */
export function Forbidden() {
  const navigate = useNavigate();
  return (
    <Result
      status="403"
      title="Không có quyền truy cập"
      subTitle="Tài khoản của bạn chưa được cấp quyền xem trang này. Vui lòng liên hệ quản lý để được hỗ trợ."
      extra={[
        <Button key="back" onClick={() => navigate(-1)}>
          Quay lại
        </Button>,
        <Button key="home" type="primary" onClick={() => navigate('/app/dashboard')}>
          Về trang tổng quan
        </Button>,
      ]}
    />
  );
}

/** Trang 404. */
export function NotFound() {
  const navigate = useNavigate();
  return (
    <Result
      status="404"
      title="Không tìm thấy trang"
      subTitle="Đường dẫn không tồn tại hoặc đã bị thay đổi."
      extra={
        <Button type="primary" onClick={() => navigate('/app/dashboard')}>
          Về trang tổng quan
        </Button>
      }
    />
  );
}

export default Forbidden;
