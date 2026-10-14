import { Alert, Descriptions, Tag, Typography } from 'antd';
import type { QuoteResult } from '@/api/subscription';
import MoneyText from '@/components/Money/MoneyText';
import { formatDate } from '@/utils/format';
import { DESC_COLUMNS } from './utils';

/** Bảng tóm tắt báo giá (bán gói / gia hạn): thời gian, giá, giảm, tổng. */
export default function QuoteSummary({ quote }: { quote: QuoteResult }) {
  return (
    <>
      {quote.chained && (
        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 12 }}
          message={`Gói mới được nối tiếp sau gói đang chạy – bắt đầu ${formatDate(quote.start_date)}`}
        />
      )}
      <Descriptions
        size="small"
        bordered
        column={DESC_COLUMNS}
        items={[
          {
            key: 'plan',
            label: 'Gói tập',
            span: 'filled' as const,
            children: (
              <>
                <span className="code-text">{quote.plan.code}</span> {quote.plan.name}
              </>
            ),
          },
          { key: 'start', label: 'Ngày bắt đầu', children: formatDate(quote.start_date) },
          { key: 'end', label: 'Ngày kết thúc', children: formatDate(quote.end_date) },
          ...(quote.total_sessions
            ? [
                {
                  key: 'sessions',
                  label: 'Số buổi',
                  span: 'filled' as const,
                  children: `${quote.total_sessions} buổi`,
                },
              ]
            : []),
          { key: 'subtotal', label: 'Giá gói', children: <MoneyText value={quote.subtotal} /> },
          {
            key: 'discount',
            label: 'Giảm giá',
            children: quote.discount ? (
              <>
                <MoneyText value={-quote.discount} colored />
                {quote.promotion && (
                  <Tag color="orange" bordered={false} style={{ marginLeft: 8 }}>
                    {quote.promotion.code}
                  </Tag>
                )}
              </>
            ) : (
              '—'
            ),
          },
          {
            key: 'total',
            label: 'Tổng thanh toán',
            span: 'filled' as const,
            children: (
              <Typography.Text strong style={{ fontSize: 18, color: '#F97316' }}>
                <MoneyText value={quote.total} />
              </Typography.Text>
            ),
          },
        ]}
      />
    </>
  );
}
