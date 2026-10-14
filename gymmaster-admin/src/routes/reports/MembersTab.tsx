import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Button, Card, Col, Progress, Result, Row, Segmented, Space, Table, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { RetweetOutlined, TeamOutlined, UserAddOutlined, UserDeleteOutlined } from '@ant-design/icons';
import reportApi from '@/api/report';
import type { CountShare, MemberPopulation, ReportGroupBy, RetentionRow } from '@/api/report';
import StatCard, { SectionHeader } from '@/components/StatCard';
import { getErrorMessage } from '@/utils/errors';
import { formatNumber } from '@/utils/format';

export interface MembersTabProps {
  branchId?: number;
  from: string;
  to: string;
  groupBy: ReportGroupBy;
}

function CountShareTable({
  title,
  rows,
  loading,
}: {
  title: string;
  rows: CountShare[];
  loading: boolean;
}) {
  return (
    <Card size="small" title={title} loading={loading}>
      <Table<CountShare>
        size="small"
        rowKey="key"
        pagination={false}
        dataSource={rows}
        locale={{ emptyText: 'Chưa có dữ liệu' }}
        columns={[
          { title: 'Nhóm', dataIndex: 'label' },
          {
            title: 'Số lượng',
            dataIndex: 'count',
            align: 'right',
            width: 90,
            render: (v: number) => <b>{formatNumber(v)}</b>,
          },
          {
            title: 'Tỷ lệ',
            dataIndex: 'pct',
            width: 150,
            render: (pct: number) => <Progress percent={pct} size="small" />,
          },
        ]}
      />
    </Card>
  );
}

export default function MembersTab({ branchId, from, to, groupBy }: MembersTabProps) {
  const [population, setPopulation] = useState<MemberPopulation>('new');

  const membersQuery = useQuery({
    queryKey: [reportApi.key, 'members', branchId ?? null, from, to, groupBy, population],
    queryFn: () =>
      reportApi.members({
        branch_id: branchId,
        from,
        to,
        group_by: groupBy,
        population,
      }),
  });

  const retentionQuery = useQuery({
    queryKey: [reportApi.key, 'retention', branchId ?? null, from, to],
    queryFn: () => reportApi.retention({ branch_id: branchId, from, to }),
  });

  if (membersQuery.isError || retentionQuery.isError) {
    return (
      <Result
        status="warning"
        title="Không tải được báo cáo hội viên"
        subTitle={getErrorMessage(membersQuery.error ?? retentionQuery.error)}
        extra={
          <Button
            onClick={() => {
              void membersQuery.refetch();
              void retentionQuery.refetch();
            }}
          >
            Thử lại
          </Button>
        }
      />
    );
  }

  const m = membersQuery.data;
  const r = retentionQuery.data;
  const rs = r?.summary;

  const retentionColumns: ColumnsType<RetentionRow> = [
    { title: 'Tháng hết hạn', dataIndex: 'label' },
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
      render: (v: number) => formatNumber(v),
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
      align: 'right',
      render: (v: number) => <b>{v}%</b>,
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
            label="Tổng hội viên hiện tại"
            value={m?.total_members ?? 0}
            icon={<TeamOutlined />}
            loading={membersQuery.isLoading}
          />
        </Col>
        <Col xs={12} lg={6}>
          <StatCard
            label="Hội viên mới trong kỳ"
            value={m?.new_total ?? 0}
            icon={<UserAddOutlined />}
            loading={membersQuery.isLoading}
          />
        </Col>
        <Col xs={12} lg={6}>
          <StatCard
            label="Tỷ lệ gia hạn"
            value={rs?.renewal_rate ?? 0}
            valueSuffix="%"
            progress={rs?.renewal_rate ?? 0}
            caption={
              rs
                ? `${formatNumber(rs.renewed)} / ${formatNumber(rs.expired)} hội viên hết hạn`
                : undefined
            }
            icon={<RetweetOutlined />}
            loading={retentionQuery.isLoading}
          />
        </Col>
        <Col xs={12} lg={6}>
          <StatCard
            label="Tỷ lệ rời bỏ (Churn)"
            value={rs?.churn_rate ?? 0}
            valueSuffix="%"
            progress={rs?.churn_rate ?? 0}
            caption={
              rs
                ? `${formatNumber(rs.churned)} rời bỏ · ${formatNumber(rs.pending)} chờ gia hạn`
                : undefined
            }
            icon={<UserDeleteOutlined />}
            loading={retentionQuery.isLoading}
          />
        </Col>
      </Row>

      <SectionHeader
        title="Phân bố hội viên"
        extra={
          <Space>
            <Typography.Text type="secondary">Tập dữ liệu:</Typography.Text>
            <Segmented<MemberPopulation>
              value={population}
              onChange={setPopulation}
              options={[
                { label: 'Hội viên mới trong kỳ', value: 'new' },
                { label: 'Toàn bộ hội viên', value: 'all' },
              ]}
            />
          </Space>
        }
      />

      <Row gutter={[12, 12]} style={{ marginBottom: 16 }}>
        <Col xs={24} md={12} lg={8}>
          <CountShareTable
            title="Trạng thái hội viên (hiện tại)"
            rows={m?.status ?? []}
            loading={membersQuery.isLoading}
          />
        </Col>
        <Col xs={24} md={12} lg={8}>
          <CountShareTable
            title="Theo nguồn khách hàng"
            rows={m?.by_source ?? []}
            loading={membersQuery.isLoading}
          />
        </Col>
        <Col xs={24} md={12} lg={8}>
          <CountShareTable
            title="Theo giới tính"
            rows={m?.by_gender ?? []}
            loading={membersQuery.isLoading}
          />
        </Col>
        <Col xs={24} md={12} lg={8}>
          <CountShareTable
            title="Theo độ tuổi"
            rows={m?.by_age ?? []}
            loading={membersQuery.isLoading}
          />
        </Col>
        <Col xs={24} md={12} lg={8}>
          <CountShareTable
            title="Theo chi nhánh"
            rows={m?.by_branch ?? []}
            loading={membersQuery.isLoading}
          />
        </Col>
        <Col xs={24} md={12} lg={8}>
          <Card size="small" title="Hội viên mới theo thời gian" loading={membersQuery.isLoading}>
            <Table
              size="small"
              rowKey="period"
              pagination={(m?.new_series.length ?? 0) > 8 ? { pageSize: 8, size: 'small' } : false}
              dataSource={m?.new_series ?? []}
              locale={{ emptyText: 'Chưa có dữ liệu' }}
              columns={[
                { title: 'Kỳ', dataIndex: 'label' },
                {
                  title: 'Số hội viên mới',
                  dataIndex: 'count',
                  align: 'right',
                  render: (v: number) => <b>{formatNumber(v)}</b>,
                },
              ]}
            />
          </Card>
        </Col>
      </Row>

      <Card
        size="small"
        title={`Tỷ lệ gia hạn & rời bỏ theo tháng (cửa sổ gia hạn ${r?.window_days ?? 30} ngày)`}
        loading={retentionQuery.isLoading}
      >
        <Table<RetentionRow>
          size="small"
          rowKey="month"
          pagination={false}
          dataSource={r?.months ?? []}
          columns={retentionColumns}
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
                <Table.Summary.Cell index={5} align="right">
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
