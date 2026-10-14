import { Button, Descriptions, Modal, Result, Space, Typography } from 'antd';
import { PlusOutlined, PrinterOutlined } from '@ant-design/icons';
import { PAYMENT_METHOD_LABEL } from '@/api/invoice';
import type { PosOrderResult } from '@/api/pos';
import { printReceipt80 } from './cart';
import { formatMoney } from '@/utils/format';

export interface SuccessModalProps {
  result: PosOrderResult | null;
  onNewOrder: () => void;
}

/** Thanh toán xong: tiền thừa cỡ lớn, In hóa đơn 80mm, Đơn mới (Enter). */
export default function SuccessModal({ result, onNewOrder }: SuccessModalProps) {
  const inv = result?.invoice;
  return (
    <Modal
      open={!!result}
      footer={null}
      closable={false}
      maskClosable={false}
      keyboard={false}
      width={520}
      centered
      destroyOnHidden
    >
      {result && inv && (
        <Result
          status="success"
          style={{ padding: '16px 0 0' }}
          title={`Đã thanh toán ${inv.invoice_no}`}
          subTitle={inv.member ? `Hội viên ${inv.member.full_name}` : 'Khách lẻ'}
          extra={
            <Space direction="vertical" size={16} style={{ width: '100%' }}>
              <div className="pos-change-box">
                <div>Tiền thừa trả khách</div>
                <div className="pos-change-box__value">{formatMoney(result.change_amount)}</div>
              </div>
              <Descriptions
                size="small"
                column={1}
                bordered
                items={[
                  { label: 'Tổng hóa đơn', children: <b>{formatMoney(inv.total)}</b> },
                  ...(inv.discount > 0
                    ? [{ label: 'Giảm giá', children: `−${formatMoney(inv.discount)}` }]
                    : []),
                  { label: 'Khách đưa', children: formatMoney(result.received_amount) },
                  {
                    label: 'Ghi nhận',
                    children:
                      inv.payments.length === 0
                        ? 'Không thu tiền'
                        : inv.payments
                            .map((p) => `${PAYMENT_METHOD_LABEL[p.method]} ${formatMoney(p.amount)}`)
                            .join(' · '),
                  },
                ]}
              />
              <Space wrap style={{ justifyContent: 'center', width: '100%' }}>
                <Button
                  size="large"
                  icon={<PrinterOutlined />}
                  onClick={() => printReceipt80(inv.id)}
                >
                  In hóa đơn 80mm
                </Button>
                <Button
                  size="large"
                  color="orange"
                  variant="solid"
                  icon={<PlusOutlined />}
                  onClick={onNewOrder}
                  autoFocus
                >
                  Đơn mới
                </Button>
              </Space>
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                Nhấn Enter để bắt đầu đơn mới
              </Typography.Text>
            </Space>
          }
        />
      )}
    </Modal>
  );
}
