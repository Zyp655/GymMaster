import type { ReactNode } from 'react';
import { Button, Result, Space } from 'antd';
import { PrinterOutlined, UserOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import type { SubscriptionDetail } from '@/api/subscription';
import MoneyText from '@/components/Money/MoneyText';
import StatusTag from '@/components/StatusTag';
import { INVOICE_STATUS_MAP, SUBSCRIPTION_STATUS_MAP } from '@/constants/status';
import { formatDate } from '@/utils/format';
import { openInvoicePrint } from './utils';

export interface SaleResultProps {
  title: string;
  sub: SubscriptionDetail;
  onClose: () => void;
  /** Nút phụ (vd. "Bán gói khác"). */
  extra?: ReactNode;
}

/** Màn hình thành công sau bán / gia hạn / nâng cấp / chuyển nhượng. */
export default function SaleResult({ title, sub, onClose, extra }: SaleResultProps) {
  const navigate = useNavigate();
  const inv = sub.invoice;
  return (
    <Result
      status="success"
      title={title}
      subTitle={
        <Space direction="vertical" size={4}>
          <span>
            {sub.member_name} · <b>{sub.plan_name}</b> · {formatDate(sub.start_date)} –{' '}
            {formatDate(sub.end_date)}{' '}
            <StatusTag status={sub.status} map={SUBSCRIPTION_STATUS_MAP} />
          </span>
          {inv && (
            <span>
              Hóa đơn <b className="code-text">{inv.invoice_no}</b>: tổng{' '}
              <MoneyText value={inv.total} strong />, đã thu <MoneyText value={inv.paid_amount} />
              {inv.due > 0 && (
                <>
                  , còn nợ <MoneyText value={inv.due} style={{ color: '#cf1322' }} />
                </>
              )}{' '}
              <StatusTag status={inv.status} map={INVOICE_STATUS_MAP} />
            </span>
          )}
        </Space>
      }
      extra={
        <Space wrap style={{ justifyContent: 'center' }}>
          {inv && (
            <Button
              type="primary"
              icon={<PrinterOutlined />}
              onClick={() => openInvoicePrint(inv.id)}
            >
              In hóa đơn
            </Button>
          )}
          <Button
            icon={<UserOutlined />}
            onClick={() => {
              onClose();
              navigate(`/app/members/${sub.member_id}?tab=subscriptions`);
            }}
          >
            Đến hồ sơ hội viên
          </Button>
          {extra}
          <Button onClick={onClose}>Đóng</Button>
        </Space>
      }
    />
  );
}
