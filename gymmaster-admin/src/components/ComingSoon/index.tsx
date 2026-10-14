import { Button, Card, Result } from 'antd';
import { ToolOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import PageTitle from '@/components/PageTitle';

/** Trang tạm cho chức năng chưa làm. Agent làm module sẽ thay route bằng trang thật. */
export function ComingSoon({ title = 'Đang phát triển' }: { title?: string }) {
  const navigate = useNavigate();
  return (
    <Card variant="borderless">
      <PageTitle title={title} />
      <Result
        icon={<ToolOutlined style={{ color: '#F97316' }} />}
        title="Chức năng đang được phát triển"
        subTitle="Màn hình này sẽ sớm có mặt trong các phiên bản tiếp theo."
        extra={
          <Button type="primary" onClick={() => navigate('/app/dashboard')}>
            Về trang tổng quan
          </Button>
        }
      />
    </Card>
  );
}

export default ComingSoon;
