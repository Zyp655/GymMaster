import { useQuery } from '@tanstack/react-query';
import { Button, Card, Col, Result, Row, Table } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import reportApi from '@/api/report';
import type { TrainerPerformanceRow } from '@/api/report';
import MoneyText from '@/components/Money/MoneyText';
import StatusTag from '@/components/StatusTag';
import { TRAINER_LEVEL_META } from '@/routes/staff/constants';
import { getErrorMessage } from '@/utils/errors';
import { formatNumber } from '@/utils/format';
import { rangeParams } from './types';
import type { ReportFilters } from './types';
import { StatCard } from '@/components/StatCard';
import { DollarOutlined, TeamOutlined, TrophyOutlined } from '@ant-design/icons';

export default function TrainersTab(f: ReportFilters) {
  const query = useQuery({
    queryKey: [reportApi.key, 'trainer-performance', f.branchId ?? null, f.from, f.to],
    queryFn: () => reportApi.trainerPerformance(rangeParams(f)),
  });

  if (query.isError) {
    return (
      <Result
        status="warning"
        title="Không tải được báo cáo hiệu suất HLV"
        subTitle={getErrorMessage(query.error)}
        extra={<Button onClick={() => void query.refetch()}>Thử lại</Button>}
      />
    );
  }

  const items = query.data?.items ?? [];
  const t = query.data?.totals;

  const columns: ColumnsType<TrainerPerformanceRow> = [
    {
      title: 'Huấn luyện viên',
      dataIndex: 'trainer_name',
      fixed: 'left',
      render: (v: string) => <b>{v}</b>,
    },
    {
      title: 'Cấp bậc',
      dataIndex: 'level',
      render: (v: string) => (v ? <StatusTag status={v} map={TRAINER_LEVEL_META} /> : '—'),
    },
    {
      title: 'Đánh giá TB',
      key: 'rating',
      align: 'right',
      render: (_, r) =>
        r.rating_count && r.rating_count > 0 ? (
          <span>
            <b>{Number(r.rating_avg ?? 0).toFixed(1)} ⭐</b> ({formatNumber(r.rating_count)} lượt)
          </span>
        ) : r.rating_avg ? (
          <b>{Number(r.rating_avg).toFixed(1)} ⭐</b>
        ) : (
          '—'
        ),
    },
    {
      title: 'PT hoàn thành',
      dataIndex: 'pt_completed',
      align: 'right',
      render: (v: number) => <b>{formatNumber(v)}</b>,
    },
    {
      title: 'PT đã lịch',
      dataIndex: 'pt_scheduled',
      align: 'right',
      render: (v: number) => formatNumber(v),
    },
    {
      title: 'PT vắng',
      dataIndex: 'pt_no_show',
      align: 'right',
      render: (v: number) => formatNumber(v),
    },
    {
      title: 'Lớp đã dạy',
      dataIndex: 'classes_taught',
      align: 'right',
      render: (v: number) => formatNumber(v),
    },
    {
      title: 'Lượt học viên lớp',
      dataIndex: 'class_attendees',
      align: 'right',
      render: (v: number) => formatNumber(v),
    },
    {
      title: 'HV PT đang dạy',
      dataIndex: 'active_pt_members',
      align: 'right',
      render: (v: number) => formatNumber(v),
    },
    {
      title: 'HĐ PT bán',
      dataIndex: 'pt_contracts_sold',
      align: 'right',
      render: (v: number) => formatNumber(v),
    },
    {
      title: 'Doanh số PT',
      dataIndex: 'pt_revenue',
      align: 'right',
      render: (v: number) => <MoneyText value={v} />,
    },
    {
      title: 'Hoa hồng',
      dataIndex: 'commission',
      align: 'right',
      render: (v: number) => <MoneyText value={v} strong />,
    },
    {
      title: 'Hoa hồng đã chi',
      dataIndex: 'commission_paid',
      align: 'right',
      render: (v: number) => <MoneyText value={v} />,
    },
  ];

  return (
    <div>
      <Row gutter={[12, 12]} style={{ marginBottom: 16 }}>
        <Col xs={12} md={8}>
          <StatCard
            label="Tổng buổi PT hoàn thành"
            value={t?.pt_completed ?? 0}
            caption={`Đã lên lịch: ${formatNumber(t?.pt_scheduled ?? 0)} buổi`}
            icon={<TrophyOutlined />}
            loading={query.isLoading}
          />
        </Col>
        <Col xs={12} md={8}>
          <StatCard
            label="Doanh số PT"
            value={t?.pt_revenue ?? 0}
            currency
            caption={`${formatNumber(t?.pt_contracts_sold ?? 0)} hợp đồng PT bán mới`}
            icon={<DollarOutlined />}
            loading={query.isLoading}
          />
        </Col>
        <Col xs={24} md={8}>
          <StatCard
            label="Tổng hoa hồng HLV"
            value={t?.commission ?? 0}
            currency
            caption={`${formatNumber(t?.classes_taught ?? 0)} buổi lớp đã dạy`}
            icon={<TeamOutlined />}
            loading={query.isLoading}
          />
        </Col>
      </Row>

      <Card size="small" title="Hiệu suất & doanh số huấn luyện viên" loading={query.isLoading}>
        <Table<TrainerPerformanceRow>
          size="small"
          rowKey="trainer_id"
          pagination={false}
          dataSource={items}
          columns={columns}
          scroll={{ x: 'max-content' }}
          locale={{ emptyText: 'Chưa có dữ liệu huấn luyện viên trong kỳ' }}
          summary={() =>
            items.length > 0 && t ? (
              <Table.Summary.Row>
                <Table.Summary.Cell index={0}>
                  <b>{t.trainer_name || 'Tổng'}</b>
                </Table.Summary.Cell>
                <Table.Summary.Cell index={1}>—</Table.Summary.Cell>
                <Table.Summary.Cell index={2} align="right">
                  {t.rating_count && t.rating_count > 0 ? (
                    <b>
                      {Number(t.rating_avg ?? 0).toFixed(1)} ⭐ ({formatNumber(t.rating_count)}{' '}
                      lượt)
                    </b>
                  ) : (
                    '—'
                  )}
                </Table.Summary.Cell>
                <Table.Summary.Cell index={3} align="right">
                  <b>{formatNumber(t.pt_completed)}</b>
                </Table.Summary.Cell>
                <Table.Summary.Cell index={4} align="right">
                  <b>{formatNumber(t.pt_scheduled)}</b>
                </Table.Summary.Cell>
                <Table.Summary.Cell index={5} align="right">
                  <b>{formatNumber(t.pt_no_show)}</b>
                </Table.Summary.Cell>
                <Table.Summary.Cell index={6} align="right">
                  <b>{formatNumber(t.classes_taught)}</b>
                </Table.Summary.Cell>
                <Table.Summary.Cell index={7} align="right">
                  <b>{formatNumber(t.class_attendees)}</b>
                </Table.Summary.Cell>
                <Table.Summary.Cell index={8} align="right">
                  <b>{formatNumber(t.active_pt_members)}</b>
                </Table.Summary.Cell>
                <Table.Summary.Cell index={9} align="right">
                  <b>{formatNumber(t.pt_contracts_sold)}</b>
                </Table.Summary.Cell>
                <Table.Summary.Cell index={10} align="right">
                  <MoneyText value={t.pt_revenue} strong />
                </Table.Summary.Cell>
                <Table.Summary.Cell index={11} align="right">
                  <MoneyText value={t.commission} strong />
                </Table.Summary.Cell>
                <Table.Summary.Cell index={12} align="right">
                  <MoneyText value={t.commission_paid} strong />
                </Table.Summary.Cell>
              </Table.Summary.Row>
            ) : null
          }
        />
      </Card>
    </div>
  );
}
