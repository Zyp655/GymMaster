import { useEffect } from 'react';
import type { ReactNode } from 'react';
import { Breadcrumb, Button, Space, Typography } from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import { Link, useNavigate } from 'react-router-dom';

export interface BreadcrumbItem {
  title: ReactNode;
  /** Bỏ trống ở mục cuối (trang hiện tại). */
  path?: string;
}

export interface PageTitleProps {
  title: ReactNode;
  subtitle?: ReactNode;
  /** `true` = quay lại trang trước; chuỗi = đường dẫn quay về. */
  back?: boolean | string;
  onBack?: () => void;
  breadcrumb?: BreadcrumbItem[];
  /** Nội dung bên phải tiêu đề (hiếm dùng – thường đặt ActionBar ở cột bên phải). */
  extra?: ReactNode;
}

/** Tiêu đề trang (+ nút quay lại, breadcrumb). Cũng đặt `document.title`. */
export function PageTitle({ title, subtitle, back, onBack, breadcrumb, extra }: PageTitleProps) {
  const navigate = useNavigate();

  useEffect(() => {
    if (typeof title === 'string') document.title = `${title} – GymMaster`;
  }, [title]);

  const handleBack = () => {
    if (onBack) onBack();
    else if (typeof back === 'string') navigate(back);
    else navigate(-1);
  };

  return (
    <div className="page-title">
      {breadcrumb && breadcrumb.length > 0 && (
        <Breadcrumb
          className="page-title__breadcrumb"
          items={breadcrumb.map((item) => ({
            title: item.path ? <Link to={item.path}>{item.title}</Link> : item.title,
          }))}
        />
      )}
      <Space align="center" size={8} wrap>
        {(back || onBack) && (
          <Button
            type="text"
            icon={<ArrowLeftOutlined />}
            onClick={handleBack}
            aria-label="Quay lại"
          />
        )}
        <Typography.Title level={4} className="page-title__text">
          {title}
        </Typography.Title>
        {extra}
      </Space>
      {subtitle && (
        <Typography.Text type="secondary" className="page-title__subtitle">
          {subtitle}
        </Typography.Text>
      )}
    </div>
  );
}

export default PageTitle;
