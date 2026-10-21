import { useQuery } from '@tanstack/react-query';
import { Button, Card, Col, Empty, Result, Row, Table, Tooltip } from 'antd';
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  FireOutlined,
  LoginOutlined,
  TeamOutlined,
} from '@ant-design/icons';
import reportApi from '@/api/report';
import StatCard from '@/components/StatCard';
import { getErrorMessage } from '@/utils/errors';
import { formatNumber } from '@/utils/format';

export interface CheckinsTabProps {
  branchId?: number;
  from: string;
  to: string;
}

const HOURS = Array.from({ length: 24 }, (_, i) => i);

export default function CheckinsTab({ branchId, from, to }: CheckinsTabProps) {
  const query = useQuery({
    queryKey: [reportApi.key, 'checkin-heatmap', branchId ?? null, from, to],
    queryFn: () => reportApi.checkinHeatmap({ branch_id: branchId, from, to }),
  });

  if (query.isError) {
    return (
      <Result
        status="warning"
        title="Không tải được báo cáo giờ cao điểm"
        subTitle={getErrorMessage(query.error)}
        extra={<Button onClick={() => void query.refetch()}>Thử lại</Button>}
      />
    );
  }

  const d = query.data;
  const max = d?.max && d.max > 0 ? d.max : 1;

  return (
    <div>
      <Row gutter={[12, 12]} style={{ marginBottom: 16 }}>
        <Col xs={12} sm={8} lg={5}>
          <StatCard
            label="Tổng lượt vào hợp lệ"
            value={d?.total ?? 0}
            icon={<CheckCircleOutlined />}
            loading={query.isLoading}
          />
        </Col>
        <Col xs={12} sm={8} lg={5}>
          <StatCard
            label="Hội viên đi tập"
            value={d?.unique_members ?? 0}
            icon={<TeamOutlined />}
            loading={query.isLoading}
          />
        </Col>
        <Col xs={12} sm={8} lg={5}>
          <StatCard
            label="Trung bình / ngày"
            value={d?.avg_per_day ?? 0}
            icon={<LoginOutlined />}
            loading={query.isLoading}
          />
        </Col>
        <Col xs={12} sm={12} lg={4}>
          <StatCard
            label="Lượt bị từ chối"
            value={d?.denied ?? 0}
            icon={<CloseCircleOutlined />}
            loading={query.isLoading}
          />
        </Col>
        <Col xs={24} sm={12} lg={5}>
          <StatCard
            label="Khung giờ cao điểm"
            value={d?.peak?.label ?? '—'}
            caption={d?.peak ? `${formatNumber(d.peak.count)} lượt` : 'Chưa có lượt vào'}
            icon={<FireOutlined />}
            loading={query.isLoading}
          />
        </Col>
      </Row>

      <Card
        size="small"
        title="Biểu đồ nhiệt (Heatmap) 7 thứ × 24 giờ"
        loading={query.isLoading}
        style={{ marginBottom: 16 }}
      >
        {!d?.matrix?.length ? (
          <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="Chưa có dữ liệu check-in" />
        ) : (
          <div style={{ overflowX: 'auto' }}>
            <table
              style={{
                width: '100%',
                minWidth: 760,
                borderCollapse: 'separate',
                borderSpacing: 3,
                fontSize: 12,
              }}
            >
              <thead>
                <tr>
                  <th style={{ textAlign: 'left', padding: '4px 8px', width: 52 }}>Thứ</th>
                  {HOURS.map((h) => (
                    <th
                      key={h}
                      style={{
                        textAlign: 'center',
                        fontWeight: 500,
                        color: 'var(--gm-text-secondary)',
                      }}
                    >
                      {h}h
                    </th>
                  ))}
                  <th style={{ textAlign: 'right', padding: '4px 6px' }}>Tổng</th>
                  <th style={{ textAlign: 'right', padding: '4px 6px' }}>TB/ngày</th>
                </tr>
              </thead>
              <tbody>
                {d.matrix.map((row) => (
                  <tr key={row.dow}>
                    <td style={{ fontWeight: 600, padding: '4px 8px' }}>{row.label}</td>
                    {HOURS.map((h) => {
                      const count = row.hours[h] ?? 0;
                      const ratio = count > 0 ? Math.max(0.15, count / max) : 0;
                      const bg =
                        count > 0
                          ? `rgba(249, 115, 22, ${ratio.toFixed(2)})`
                          : 'var(--gm-tone-slate-bg)';
                      const fg = ratio > 0.55 ? '#FFFFFF' : 'var(--gm-ink)';
                      return (
                        <Tooltip
                          key={h}
                          title={`${row.label} ${String(h).padStart(2, '0')}:00 – ${formatNumber(count)} lượt`}
                        >
                          <td
                            style={{
                              background: bg,
                              color: fg,
                              textAlign: 'center',
                              borderRadius: 4,
                              padding: '6px 2px',
                              fontVariantNumeric: 'tabular-nums',
                              cursor: 'default',
                            }}
                          >
                            {count > 0 ? count : ''}
                          </td>
                        </Tooltip>
                      );
                    })}
                    <td style={{ textAlign: 'right', fontWeight: 600, padding: '4px 6px' }}>
                      {formatNumber(row.total)}
                    </td>
                    <td
                      style={{
                        textAlign: 'right',
                        color: 'var(--gm-text-secondary)',
                        padding: '4px 6px',
                      }}
                    >
                      {row.avg_per_day}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>

      <Card size="small" title="Lượt check-in theo ngày" loading={query.isLoading}>
        <Table
          size="small"
          rowKey="period"
          dataSource={d?.daily ?? []}
          pagination={(d?.daily.length ?? 0) > 15 ? { pageSize: 15, size: 'small' } : false}
          locale={{ emptyText: 'Chưa có dữ liệu theo ngày' }}
          columns={[
            { title: 'Ngày', dataIndex: 'label' },
            {
              title: 'Số lượt vào hợp lệ',
              dataIndex: 'count',
              align: 'right',
              render: (v: number) => <b>{formatNumber(v)}</b>,
            },
          ]}
        />
      </Card>
    </div>
  );
}
