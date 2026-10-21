import { useQuery } from '@tanstack/react-query';
import { Button, Card, Col, Progress, Result, Row, Table } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  ClockCircleOutlined,
  RetweetOutlined,
  TeamOutlined,
  UserDeleteOutlined,
} from '@ant-design/icons';
import reportApi from '@/api/report';
import type { RetentionRow } from '@/api/report';
import StatCard from '@/components/StatCard';
import { getErrorMessage } from '@/utils/errors';
import { formatNumber } from '@/utils/format';
import { rangeParams } from './types';
import type { ReportFilters } from './types';

export default function RetentionTab(f: ReportFilters) {
  const query = useQuery({
    queryKey: [reportApi.key, 'retention', f.branchId ?? null, f.from, f.to],
    queryFn: () => reportApi.retention(rangeParams(f)),
  });

  if (query.isError) {
    return (
      <Result
        status="warning"
        title="Không tải được báo cáo giữ chân hội viên"
        subTitle={getErrorMessage(query.error)}
        extra={<Button onClick={() => void query.refetch()}>Thử lại</Button>}
      />
    );
  }

  const r = query.data;
  const rs = r?.summary;

  const columns: ColumnsType<RetentionRow> = [
    { title: 'Tháng hết hạn', dataIndex: 'label', render: (v: string) => <b>{v}</b> },
    {
      title: 'Hết hạn',
      dataIndex: 'expired',
      align: 'right',
      render: (v: number) => formatNumber(v),
    },
    {
      title: 'Đã gia hạn',
      dataIndex: 'renewed',
      align: 'right',
      render: (v: number) => <b>{formatNumber(v)}</b>,
    },
    {
      title: 'Đang chờ (trong hạn)',
      dataIndex: 'pending',
      align: 'right',
      render: (v: number) => formatNumber(v),
    },
    {
      title: 'Rời bỏ (Churn)',
      dataIndex: 'churned',
      align: 'right',
      render: (v: number) => formatNumber(v),
    },
    {
      title: 'Tỷ lệ gia hạn',
      dataIndex: 'renewal_rate',
      width: 150,
      render: (pct: number) => <Progress percent={pct} size="small" />,
    },
    {
      title: 'Tỷ lệ rời bỏ',
      dataIndex: 'churn_rate',
      align: 'right',
      render: (v: number) => `${v}%`,
    },
  ];

  return (
    <div>
      <Row gutter={[12, 12]} style={{ marginBottom: 16 }}>
        <Col xs={12} lg={6}>
          <StatCard
            label="Tổng gói hết hạn"
            value={rs?.expired ?? 0}
            icon={<TeamOutlined />}
            loading={query.isLoading}
          />
        </Col>
        <Col xs={12} lg={6}>
          <StatCard
            label="Tỷ lệ gia hạn"
            value={rs?.renewal_rate ?? 0}
            valueSuffix="%"
            progress={rs?.renewal_rate ?? 0}
            caption={rs ? `${formatNumber(rs.renewed)} hội viên đã gia hạn` : undefined}
            icon={<RetweetOutlined />}
            loading={query.isLoading}
          />
        </Col>
        <Col xs={12} lg={6}>
          <StatCard
            label="Đang trong cửa sổ gia hạn"
            value={rs?.pending ?? 0}
            caption={`Cửa sổ theo dõi ${r?.window_days ?? 30} ngày`}
            icon={<ClockCircleOutlined />}
            loading={query.isLoading}
          />
        </Col>
        <Col xs={12} lg={6}>
          <StatCard
            label="Tỷ lệ rời bỏ (Churn)"
            value={rs?.churn_rate ?? 0}
            valueSuffix="%"
            progress={rs?.churn_rate ?? 0}
            caption={rs ? `${formatNumber(rs.churned)} hội viên chưa quay lại` : undefined}
            icon={<UserDeleteOutlined />}
            loading={query.isLoading}
          />
        </Col>
      </Row>

      <Card
        size="small"
        title={`Tỷ lệ gia hạn & rời bỏ theo tháng (cửa sổ gia hạn ${r?.window_days ?? 30} ngày)`}
        loading={query.isLoading}
      >
        <Table<RetentionRow>
          size="small"
          rowKey="month"
          pagination={false}
          dataSource={r?.months ?? []}
          columns={columns}
          scroll={{ x: 'max-content' }}
          locale={{ emptyText: 'Chưa có dữ liệu gia hạn trong kỳ' }}
          summary={() =>
            rs ? (
              <Table.Summary.Row>
                <Table.Summary.Cell index={0}>
                  <b>{rs.label || 'Tổng'}</b>
                </Table.Summary.Cell>
                <Table.Summary.Cell index={1} align="right">
                  <b>{formatNumber(rs.expired)}</b>
                </Table.Summary.Cell>
                <Table.Summary.Cell index={2} align="right">
                  <b>{formatNumber(rs.renewed)}</b>
                </Table.Summary.Cell>
                <Table.Summary.Cell index={3} align="right">
                  <b>{formatNumber(rs.pending)}</b>
                </Table.Summary.Cell>
                <Table.Summary.Cell index={4} align="right">
                  <b>{formatNumber(rs.churned)}</b>
                </Table.Summary.Cell>
                <Table.Summary.Cell index={5}>
                  <b>{rs.renewal_rate}%</b>
                </Table.Summary.Cell>
                <Table.Summary.Cell index={6} align="right">
                  <b>{rs.churn_rate}%</b>
                </Table.Summary.Cell>
              </Table.Summary.Row>
            ) : null
          }
        />
      </Card>
    </div>
  );
}
