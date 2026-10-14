import { useMemo, useState } from 'react';
import {
  App,
  Button,
  Card,
  Col,
  DatePicker,
  Empty,
  Result,
  Row,
  Segmented,
  Space,
  Table,
  Tooltip,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { DollarOutlined, ReloadOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import dayjs from 'dayjs';
import ptApi from '@/api/pt';
import type { Commission, CommissionStatus, CommissionSummaryItem } from '@/api/pt';
import BranchSelect from '@/components/BranchSelect';
import ConfirmButton from '@/components/ConfirmButton';
import MoneyText from '@/components/Money/MoneyText';
import PageTitle from '@/components/PageTitle';
import StatCard from '@/components/StatCard';
import StatusTag from '@/components/StatusTag';
import { PERM } from '@/constants/permissions';
import { useIsMobile } from '@/hooks/useIsMobile';
import { useUrlParams } from '@/hooks/useUrlParams';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { getErrorMessage, showError } from '@/utils/errors';
import { formatDateTime, formatMoney } from '@/utils/format';
import { COMMISSION_SOURCE_LABEL, COMMISSION_STATUS_MAP } from './constants';
import { usePayCommission } from './queries';

const PERIOD_RE = /^\d{4}-\d{2}$/;

/** Chi tiết hoa hồng của một HLV trong kỳ (mở rộng theo hàng). */
function CommissionDetail({
  trainerId,
  period,
  branchId,
}: {
  trainerId: number;
  period: string;
  branchId?: number;
}) {
  const [status, setStatus] = useState<CommissionStatus | 'ALL'>('ALL');
  const [page, setPage] = useState(1);
  const params = {
    trainer_id: trainerId,
    period,
    branch_id: branchId,
    status: status === 'ALL' ? undefined : status,
    page,
    limit: 20,
    order_by: 'id',
    order_dir: 'DESC' as const,
  };
  const q = useQuery({
    queryKey: [ptApi.commissionKey, 'list', params],
    queryFn: () => ptApi.listCommissions(params),
  });

  const columns: ColumnsType<Commission> = [
    {
      title: 'Nguồn',
      key: 'source',
      render: (_, c) => (
        <Space direction="vertical" size={0}>
          <span>{COMMISSION_SOURCE_LABEL[c.source_type] ?? c.source_type}</span>
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            {c.member_name || `#${c.source_id}`}
            {c.contract_id ? ` · HĐ #${c.contract_id}` : ''}
          </Typography.Text>
        </Space>
      ),
    },
    {
      title: 'Buổi lúc',
      dataIndex: 'session_start_at',
      className: 'text-nowrap',
      render: (v: string | null) => (v ? formatDateTime(v) : '—'),
    },
    { title: 'Chi nhánh', dataIndex: 'branch_name', responsive: ['lg'] },
    {
      title: 'Giá/buổi',
      dataIndex: 'base_amount',
      align: 'right',
      render: (v: number) => <MoneyText value={v} />,
    },
    { title: 'Tỷ lệ', dataIndex: 'rate', align: 'right', render: (v: number) => `${v}%` },
    {
      title: 'Hoa hồng',
      dataIndex: 'amount',
      align: 'right',
      render: (v: number) => <MoneyText value={v} strong />,
    },
    {
      title: 'Trạng thái',
      key: 'status',
      render: (_, c) => (
        <Space direction="vertical" size={0}>
          <StatusTag status={c.status} map={COMMISSION_STATUS_MAP} />
          {c.paid_at && (
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              {formatDateTime(c.paid_at)} · {c.paid_by_name}
            </Typography.Text>
          )}
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: '4px 0' }}>
      <Segmented
        size="small"
        style={{ marginBottom: 8 }}
        value={status}
        onChange={(v) => {
          setStatus(v as CommissionStatus | 'ALL');
          setPage(1);
        }}
        options={[
          { value: 'ALL', label: 'Tất cả' },
          { value: 'PENDING', label: 'Chờ chi' },
          { value: 'PAID', label: 'Đã chi' },
        ]}
      />
      {q.isError ? (
        <Typography.Text type="danger">{getErrorMessage(q.error)}</Typography.Text>
      ) : (
        <Table<Commission>
          rowKey="id"
          size="small"
          loading={q.isLoading}
          dataSource={q.data?.items ?? []}
          columns={columns}
          scroll={{ x: 'max-content' }}
          pagination={
            (q.data?.total ?? 0) > 20
              ? {
                  current: page,
                  pageSize: 20,
                  total: q.data?.total,
                  onChange: setPage,
                  showSizeChanger: false,
                }
              : false
          }
          locale={{ emptyText: 'Không có dòng hoa hồng nào' }}
        />
      )}
    </div>
  );
}

/** Hoa hồng HLV: tổng hợp theo kỳ + chi nhánh, chi trả, chi tiết theo hàng. */
export default function CommissionsPage() {
  const { message } = App.useApp();
  const isMobile = useIsMobile();
  const { get, set } = useUrlParams();
  const { branchId: workingBranchId } = useWorkingBranch();
  const periodParam = get('period');
  const period =
    periodParam && PERIOD_RE.test(periodParam) ? periodParam : dayjs().format('YYYY-MM');
  const branchParam = get('branch_id');
  const branchId =
    branchParam === 'all'
      ? undefined
      : branchParam && Number(branchParam) > 0
        ? Number(branchParam)
        : (workingBranchId ?? undefined);

  const summary = useQuery({
    queryKey: [ptApi.commissionKey, 'summary', period, branchId ?? null],
    queryFn: () => ptApi.commissionSummary(period, branchId),
  });
  const pay = usePayCommission();
  const data = summary.data;

  const periodLabel = dayjs(`${period}-01`).format('MM/YYYY');

  const columns = useMemo<ColumnsType<CommissionSummaryItem>>(
    () => [
      {
        title: 'Huấn luyện viên',
        key: 'trainer',
        render: (_, r) => (
          <Space direction="vertical" size={0}>
            <b>{r.trainer_name}</b>
            {r.staff_code && (
              <Typography.Text type="secondary" className="code-text" style={{ fontSize: 12 }}>
                {r.staff_code}
              </Typography.Text>
            )}
          </Space>
        ),
      },
      { title: 'Số buổi', dataIndex: 'sessions', align: 'right' },
      {
        title: 'Doanh thu buổi',
        dataIndex: 'base_total',
        align: 'right',
        responsive: ['md'],
        render: (v: number) => <MoneyText value={v} />,
      },
      {
        title: 'Tổng hoa hồng',
        dataIndex: 'total_amount',
        align: 'right',
        render: (v: number) => <MoneyText value={v} strong />,
      },
      {
        title: 'Đã chi',
        dataIndex: 'paid_amount',
        align: 'right',
        render: (v: number) => <MoneyText value={v} />,
      },
      {
        title: 'Chưa chi',
        dataIndex: 'pending_amount',
        align: 'right',
        render: (v: number) =>
          v > 0 ? (
            <Typography.Text type="warning" strong>
              <MoneyText value={v} />
            </Typography.Text>
          ) : (
            <MoneyText value={v} />
          ),
      },
      {
        title: '',
        key: 'actions',
        fixed: 'right',
        width: 110,
        render: (_, r) => (
          <ConfirmButton
            perm={PERM.COMMISSION_PAY}
            size="small"
            type="primary"
            icon={<DollarOutlined />}
            disabled={r.pending_amount <= 0}
            okText="Chi trả"
            confirmTitle={`Chi trả hoa hồng kỳ ${periodLabel} cho ${r.trainer_name}?`}
            confirmContent={`Số tiền: ${formatMoney(r.pending_amount)}${branchId ? ' (chỉ hoa hồng phát sinh tại chi nhánh đang chọn)' : ''}. Các dòng "Chờ chi" sẽ chuyển sang "Đã chi".`}
            onConfirm={() =>
              pay.mutateAsync({ trainer_id: r.trainer_id, period, branch_id: branchId }).then(
                (res) =>
                  message.success(
                    `Đã chi ${formatMoney(res.amount)} (${res.count} dòng) cho ${res.trainer_name}`,
                  ),
                (e) => {
                  showError(e, 'Không thể chi trả hoa hồng');
                  throw e;
                },
              )
            }
          >
            Chi trả
          </ConfirmButton>
        ),
      },
    ],
    [branchId, message, pay, period, periodLabel],
  );

  return (
    <Card variant="borderless" className="page-card">
      <Row justify="space-between" align="middle" gutter={[8, 12]} className="page-head">
        <Col>
          <PageTitle title="Hoa hồng HLV" subtitle={`Kỳ ${periodLabel}`} />
        </Col>
        <Col>
          <Tooltip title="Tải lại">
            <Button
              icon={<ReloadOutlined spin={summary.isFetching} />}
              onClick={() => summary.refetch()}
            />
          </Tooltip>
        </Col>
      </Row>

      <Row gutter={[8, 8]} style={{ marginBottom: 16 }}>
        <Col xs={12} sm={8} lg={4}>
          <DatePicker
            picker="month"
            allowClear={false}
            format="MM/YYYY"
            value={dayjs(`${period}-01`)}
            onChange={(d) => d && set({ period: d.format('YYYY-MM') })}
            style={{ width: '100%' }}
          />
        </Col>
        <Col xs={12} sm={8} lg={5}>
          <BranchSelect
            allowAll
            value={branchId ?? null}
            onChange={(v) => set({ branch_id: v === null ? 'all' : (v as number) })}
          />
        </Col>
      </Row>

      <Row gutter={[12, 12]} style={{ marginBottom: 16 }}>
        <Col xs={12} lg={6}>
          <StatCard label="Số buổi tính hoa hồng" value={data?.sessions ?? 0} />
        </Col>
        <Col xs={12} lg={6}>
          <StatCard label="Tổng hoa hồng" value={data?.total_amount ?? 0} currency />
        </Col>
        <Col xs={12} lg={6}>
          <StatCard label="Đã chi" value={data?.paid_amount ?? 0} currency />
        </Col>
        <Col xs={12} lg={6}>
          <StatCard label="Chưa chi" value={data?.pending_amount ?? 0} currency />
        </Col>
      </Row>

      {summary.isError ? (
        <Result
          status="warning"
          title="Không tải được hoa hồng"
          subTitle={getErrorMessage(summary.error)}
          extra={<Button onClick={() => summary.refetch()}>Thử lại</Button>}
        />
      ) : (
        <Table<CommissionSummaryItem>
          rowKey="trainer_id"
          size="small"
          loading={summary.isLoading}
          dataSource={data?.items ?? []}
          columns={columns}
          pagination={false}
          scroll={{ x: 'max-content' }}
          expandable={{
            expandedRowRender: (r) => (
              <CommissionDetail trainerId={r.trainer_id} period={period} branchId={branchId} />
            ),
            expandRowByClick: isMobile,
          }}
          locale={{
            emptyText: (
              <Empty
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                description={`Chưa có hoa hồng nào trong kỳ ${periodLabel}`}
              />
            ),
          }}
          summary={(rows) =>
            rows.length > 1 && data ? (
              <Table.Summary.Row>
                <Table.Summary.Cell index={0} />
                <Table.Summary.Cell index={1}>
                  <b>Tổng</b>
                </Table.Summary.Cell>
                <Table.Summary.Cell index={2} align="right">
                  {data.sessions}
                </Table.Summary.Cell>
                {!isMobile && <Table.Summary.Cell index={3} />}
                <Table.Summary.Cell index={4} align="right">
                  <MoneyText value={data.total_amount} strong />
                </Table.Summary.Cell>
                <Table.Summary.Cell index={5} align="right">
                  <MoneyText value={data.paid_amount} />
                </Table.Summary.Cell>
                <Table.Summary.Cell index={6} align="right">
                  <MoneyText value={data.pending_amount} />
                </Table.Summary.Cell>
                <Table.Summary.Cell index={7} />
              </Table.Summary.Row>
            ) : null
          }
        />
      )}
    </Card>
  );
}
