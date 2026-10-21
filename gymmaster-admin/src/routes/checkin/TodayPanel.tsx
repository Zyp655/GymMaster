import { useMemo } from 'react';
import { Column } from '@ant-design/charts';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { App, Button, Card, Col, Empty, Row, Space, Table, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  LogoutOutlined,
  TeamOutlined,
  UserOutlined,
} from '@ant-design/icons';
import { Link } from 'react-router-dom';
import checkinApi from '@/api/checkin';
import type { PresentItem } from '@/api/checkin';
import { MemberAvatar } from '@/components/MemberSelect';
import StatCard from '@/components/StatCard';
import { PERM } from '@/constants/permissions';
import { usePermission } from '@/hooks/usePermission';
import { showError } from '@/utils/errors';
import { formatDateTime } from '@/utils/format';

const minutesText = (m: number) =>
  m >= 60 ? `${Math.floor(m / 60)} giờ ${m % 60} phút` : `${m} phút`;

/** Thống kê hôm nay + biểu đồ theo giờ + danh sách "Đang tập" (có Checkout). */
export default function TodayPanel({ branchId }: { branchId: number | null }) {
  const { can } = usePermission();
  const { message } = App.useApp();
  const queryClient = useQueryClient();

  const stats = useQuery({
    queryKey: [checkinApi.key, 'stats', branchId],
    queryFn: () => checkinApi.stats(branchId),
    refetchInterval: 60_000,
  });
  const present = useQuery({
    queryKey: [checkinApi.key, 'present', branchId],
    queryFn: () => checkinApi.present(branchId),
    refetchInterval: 60_000,
  });

  const checkout = useMutation({
    mutationFn: (id: number) => checkinApi.checkout(id),
    onSuccess: (item) => {
      message.success(`${item.member_name} đã checkout`);
      return queryClient.invalidateQueries({ queryKey: [checkinApi.key] });
    },
    onError: (e) => showError(e, 'Không thể checkout'),
  });

  const chartData = useMemo(() => {
    const rows = stats.data?.by_hour ?? [];
    const active = rows.filter((r) => r.allowed + r.denied > 0).map((r) => r.hour);
    const from = Math.min(5, ...active);
    const to = Math.max(22, ...active);
    return rows
      .filter((r) => r.hour >= from && r.hour <= to)
      .flatMap((r) => [
        { hour: `${String(r.hour).padStart(2, '0')}h`, value: r.allowed, type: 'Cho vào' },
        { hour: `${String(r.hour).padStart(2, '0')}h`, value: r.denied, type: 'Từ chối' },
      ]);
  }, [stats.data]);

  const s = stats.data;
  const columns: ColumnsType<PresentItem> = [
    {
      title: 'Hội viên',
      dataIndex: 'member_name',
      render: (_, r) => (
        <Space size={8}>
          <MemberAvatar src={r.member_avatar_url} name={r.member_name} />
          <div style={{ lineHeight: 1.25 }}>
            {/* Lượt vào bằng vé ngày / vé tập thử không gắn hội viên (member_id = 0). */}
            {r.member_id ? (
              <Link to={`/app/members/${r.member_id}`}>{r.member_name}</Link>
            ) : (
              <Typography.Text>{r.member_name}</Typography.Text>
            )}
            <br />
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              {r.member_code}
              {r.plan_name ? ` · ${r.plan_name}` : ''}
            </Typography.Text>
          </div>
        </Space>
      ),
    },
    ...(branchId ? [] : [{ title: 'Chi nhánh', dataIndex: 'branch_name' }]),
    {
      title: 'Vào lúc',
      dataIndex: 'checked_in_at',
      render: (v: string) => formatDateTime(v).slice(11),
    },
    { title: 'Đã tập', dataIndex: 'minutes', render: minutesText },
    ...(can(PERM.CHECKIN_SCAN)
      ? [
          {
            title: '',
            key: 'actions',
            width: 110,
            render: (_: unknown, r: PresentItem) => (
              <Button
                size="small"
                icon={<LogoutOutlined />}
                loading={checkout.isPending && checkout.variables === r.checkin_id}
                onClick={() => checkout.mutate(r.checkin_id)}
              >
                Checkout
              </Button>
            ),
          },
        ]
      : []),
  ];

  return (
    <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
      <Col xs={12} md={6}>
        <StatCard
          label="Tổng lượt quét hôm nay"
          value={s?.total ?? 0}
          icon={<TeamOutlined />}
          loading={stats.isLoading}
          caption={s ? `${s.unique_members} khách (hội viên + vé)` : undefined}
        />
      </Col>
      <Col xs={12} md={6}>
        <StatCard
          label="Cho vào"
          value={s?.allowed ?? 0}
          icon={<CheckCircleOutlined />}
          loading={stats.isLoading}
        />
      </Col>
      <Col xs={12} md={6}>
        <StatCard
          label="Từ chối"
          value={s?.denied ?? 0}
          icon={<CloseCircleOutlined />}
          loading={stats.isLoading}
        />
      </Col>
      <Col xs={12} md={6}>
        <StatCard
          label="Đang tập"
          value={s?.present ?? 0}
          icon={<UserOutlined />}
          loading={stats.isLoading}
        />
      </Col>
      <Col xs={24} lg={11}>
        <Card size="small" title="Lượt vào theo giờ" loading={stats.isLoading}>
          {s && s.total > 0 ? (
            <Column
              height={260}
              data={chartData}
              xField="hour"
              yField="value"
              colorField="type"
              stack
              scale={{ color: { range: ['#16A34A', '#DC2626'] } }}
              axis={{ x: { title: false }, y: { title: false, tickCount: 4 } }}
              legend={{ color: { position: 'top' } }}
            />
          ) : (
            <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="Hôm nay chưa có lượt quét" />
          )}
          {s && s.deny_reasons.length > 0 && (
            <Typography.Paragraph type="secondary" style={{ margin: '8px 0 0', fontSize: 12 }}>
              Lý do từ chối: {s.deny_reasons.map((d) => `${d.message} (${d.count})`).join(' · ')}
            </Typography.Paragraph>
          )}
        </Card>
      </Col>
      <Col xs={24} lg={13}>
        <Card
          size="small"
          title={`Đang tập (${present.data?.length ?? 0})`}
          extra={
            <Button size="small" type="link" onClick={() => present.refetch()}>
              Làm mới
            </Button>
          }
        >
          <Table<PresentItem>
            size="small"
            rowKey="checkin_id"
            loading={present.isFetching}
            columns={columns}
            dataSource={present.data ?? []}
            pagination={(present.data?.length ?? 0) > 8 ? { pageSize: 8, size: 'small' } : false}
            scroll={{ x: 'max-content' }}
            locale={{ emptyText: 'Chưa có ai trong phòng tập' }}
          />
        </Card>
      </Col>
    </Row>
  );
}
