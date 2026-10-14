import { useQuery } from '@tanstack/react-query';
import { Button, Card, Col, Progress, Result, Row, Space, Table } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  CalendarOutlined,
  CheckCircleOutlined,
  PieChartOutlined,
  TeamOutlined,
  UserDeleteOutlined,
} from '@ant-design/icons';
import reportApi from '@/api/report';
import type { OccupancyRow } from '@/api/report';
import StatCard from '@/components/StatCard';
import { getErrorMessage } from '@/utils/errors';
import { formatNumber } from '@/utils/format';

export interface ClassesTabProps {
  branchId?: number;
  from: string;
  to: string;
}

function OccupancyTable({
  title,
  labelTitle,
  rows,
  loading,
}: {
  title: string;
  labelTitle: string;
  rows: OccupancyRow[];
  loading: boolean;
}) {
  const columns: ColumnsType<OccupancyRow> = [
    {
      title: labelTitle,
      dataIndex: 'label',
      render: (v: string, r) => (
        <Space size={6}>
          {r.color && (
            <span
              style={{
                width: 10,
                height: 10,
                borderRadius: '50%',
                background: r.color,
                display: 'inline-block',
              }}
            />
          )}
          <b>{v}</b>
        </Space>
      ),
    },
    {
      title: 'Số buổi',
      dataIndex: 'sessions',
      align: 'right',
      render: (v: number) => formatNumber(v),
    },
    {
      title: 'Sức chứa',
      dataIndex: 'capacity',
      align: 'right',
      responsive: ['md'],
      render: (v: number) => formatNumber(v),
    },
    {
      title: 'Đặt chỗ',
      dataIndex: 'booked',
      align: 'right',
      render: (v: number) => formatNumber(v),
    },
    {
      title: 'Có mặt',
      dataIndex: 'attended',
      align: 'right',
      render: (v: number) => formatNumber(v),
    },
    {
      title: 'Vắng',
      dataIndex: 'no_show',
      align: 'right',
      render: (v: number) => formatNumber(v),
    },
    {
      title: 'Lấp đầy',
      dataIndex: 'fill_rate',
      width: 130,
      render: (pct: number) => <Progress percent={pct} size="small" />,
    },
    {
      title: 'Chuyên cần',
      dataIndex: 'attendance_rate',
      align: 'right',
      render: (pct: number) => `${pct}%`,
    },
  ];

  return (
    <Card size="small" title={title} loading={loading}>
      <Table<OccupancyRow>
        size="small"
        rowKey="key"
        pagination={false}
        dataSource={rows}
        columns={columns}
        scroll={{ x: 'max-content' }}
        locale={{ emptyText: 'Chưa có buổi lớp nào trong kỳ' }}
      />
    </Card>
  );
}

export default function ClassesTab({ branchId, from, to }: ClassesTabProps) {
  const query = useQuery({
    queryKey: [reportApi.key, 'class-occupancy', branchId ?? null, from, to],
    queryFn: () => reportApi.classOccupancy({ branch_id: branchId, from, to }),
  });

  if (query.isError) {
    return (
      <Result
        status="warning"
        title="Không tải được báo cáo lớp nhóm"
        subTitle={getErrorMessage(query.error)}
        extra={<Button onClick={() => void query.refetch()}>Thử lại</Button>}
      />
    );
  }

  const s = query.data?.summary;

  return (
    <div>
      <Row gutter={[12, 12]} style={{ marginBottom: 16 }}>
        <Col xs={12} sm={8} lg={5}>
          <StatCard
            label="Số buổi lớp"
            value={s?.sessions ?? 0}
            caption={`Sức chứa: ${formatNumber(s?.capacity ?? 0)} chỗ`}
            icon={<CalendarOutlined />}
            loading={query.isLoading}
          />
        </Col>
        <Col xs={12} sm={8} lg={5}>
          <StatCard
            label="Lượt đặt chỗ"
            value={s?.booked ?? 0}
            caption={`Có mặt: ${formatNumber(s?.attended ?? 0)} · Chờ: ${formatNumber(s?.waitlist ?? 0)}`}
            icon={<TeamOutlined />}
            loading={query.isLoading}
          />
        </Col>
        <Col xs={12} sm={8} lg={5}>
          <StatCard
            label="Tỷ lệ lấp đầy"
            value={s?.fill_rate ?? 0}
            valueSuffix="%"
            progress={s?.fill_rate ?? 0}
            icon={<PieChartOutlined />}
            loading={query.isLoading}
          />
        </Col>
        <Col xs={12} sm={12} lg={5}>
          <StatCard
            label="Tỷ lệ chuyên cần"
            value={s?.attendance_rate ?? 0}
            valueSuffix="%"
            progress={s?.attendance_rate ?? 0}
            icon={<CheckCircleOutlined />}
            loading={query.isLoading}
          />
        </Col>
        <Col xs={24} sm={12} lg={4}>
          <StatCard
            label="Vắng (No-show)"
            value={s?.no_show ?? 0}
            caption={`Hủy đặt: ${formatNumber(s?.cancelled ?? 0)}`}
            icon={<UserDeleteOutlined />}
            loading={query.isLoading}
          />
        </Col>
      </Row>

      <Row gutter={[12, 12]}>
        <Col xs={24} lg={12}>
          <OccupancyTable
            title="Theo loại lớp"
            labelTitle="Loại lớp"
            rows={query.data?.by_class_type ?? []}
            loading={query.isLoading}
          />
        </Col>
        <Col xs={24} lg={12}>
          <OccupancyTable
            title="Theo huấn luyện viên"
            labelTitle="Huấn luyện viên"
            rows={query.data?.by_trainer ?? []}
            loading={query.isLoading}
          />
        </Col>
        <Col xs={24} lg={12}>
          <OccupancyTable
            title="Theo khung giờ"
            labelTitle="Khung giờ"
            rows={query.data?.by_time_slot ?? []}
            loading={query.isLoading}
          />
        </Col>
        <Col xs={24} lg={12}>
          <OccupancyTable
            title="Theo thứ trong tuần"
            labelTitle="Thứ"
            rows={query.data?.by_weekday ?? []}
            loading={query.isLoading}
          />
        </Col>
      </Row>
    </div>
  );
}
