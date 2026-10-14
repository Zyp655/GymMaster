import { useCallback, useMemo, useState } from 'react';
import { Button, Card, Col, Progress, Row, Space, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { EyeOutlined } from '@ant-design/icons';
import { Link } from 'react-router-dom';
import ptApi from '@/api/pt';
import type { PtContract, PtContractListParams, PtContractStatus } from '@/api/pt';
import ActionBar from '@/components/ActionBar';
import DataTable from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import MobileList from '@/components/MobileList';
import type { MobileFetchArgs } from '@/components/MobileList';
import MoneyText from '@/components/Money/MoneyText';
import PageTitle from '@/components/PageTitle';
import StatusTag from '@/components/StatusTag';
import { PERM } from '@/constants/permissions';
import { INVOICE_STATUS_MAP } from '@/constants/status';
import { useBranchOptions } from '@/hooks/useBranchOptions';
import { useIsMobile } from '@/hooks/useIsMobile';
import { usePermission } from '@/hooks/usePermission';
import { useTableQuery } from '@/hooks/useTableQuery';
import { useUrlParams } from '@/hooks/useUrlParams';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { formatDate, formatMoney, formatPhone } from '@/utils/format';
import ContractDrawer from './ContractDrawer';
import SellPtModal from './SellPtModal';
import { PT_CONTRACT_STATUS_MAP, PT_CONTRACT_STATUS_OPTIONS, isTrainerOnly } from './constants';
import { usePtPackageOptions, usePtTrainerOptions } from './queries';

const TITLE = 'Hợp đồng PT';
const SUBTITLE = 'Gói huấn luyện cá nhân đã bán: tiến độ buổi, HLV, hóa đơn';

function useFilterOptions() {
  const { user } = usePermission();
  const { branchId } = useWorkingBranch();
  const { branches } = useBranchOptions();
  const packages = usePtPackageOptions();
  const trainerOnly = isTrainerOnly(user);
  const trainers = usePtTrainerOptions(branchId, !trainerOnly);
  return {
    trainerOnly,
    branchOptions: branches.map((b) => ({ label: b.name, value: b.id })),
    packageOptions: (packages.data ?? []).map((p) => ({
      value: p.id,
      label: `${p.code} – ${p.name}`,
    })),
    trainerOptions: (trainers.data ?? []).map((t) => ({ value: t.id, label: t.full_name })),
  };
}

function SessionProgress({ r }: { r: PtContract }) {
  const total = Math.max(r.total_sessions, 1);
  return (
    <div style={{ minWidth: 120 }}>
      <Typography.Text style={{ fontSize: 12 }}>
        {r.used_sessions}/{r.total_sessions} buổi
        {r.scheduled_sessions > 0 && (
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            {' '}
            · đặt {r.scheduled_sessions}
          </Typography.Text>
        )}
      </Typography.Text>
      <Progress
        size="small"
        showInfo={false}
        percent={Math.round(((r.used_sessions + r.scheduled_sessions) / total) * 100)}
        success={{ percent: Math.round((r.used_sessions / total) * 100) }}
      />
    </div>
  );
}

function ContractsDesktop() {
  const url = useUrlParams();
  const { branchId } = useWorkingBranch();
  const { trainerOnly, branchOptions, packageOptions, trainerOptions } = useFilterOptions();
  const [selling, setSelling] = useState(false);
  const viewId = Number(url.get('cid')) || null;
  const urlBranch = url.get('branch_id');

  const table = useTableQuery<PtContract, PtContractListParams>({
    queryKey: [ptApi.key, 'contracts'],
    queryFn: (p) => ptApi.listContracts(p),
    filterKeys: ['status', 'pt_package_id', 'branch_id', 'trainer_id'],
    numberKeys: ['pt_package_id', 'branch_id', 'trainer_id'],
    defaults: { order_by: 'id', order_dir: 'DESC' },
    extraParams: { branch_id: urlBranch ? undefined : (branchId ?? undefined) },
  });

  const open = useCallback((id: number) => url.set({ cid: id }), [url]);

  const columns = useMemo<ColumnsType<PtContract>>(
    () => [
      {
        title: 'Hội viên',
        key: 'member',
        render: (_, r) => (
          <Space direction="vertical" size={0}>
            <Button type="link" size="small" style={{ padding: 0 }} onClick={() => open(r.id)}>
              {r.member_name}
            </Button>
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              {r.member_code} · {formatPhone(r.member_phone)}
            </Typography.Text>
          </Space>
        ),
      },
      {
        title: 'Gói PT',
        key: 'package',
        render: (_, r) => (
          <Space direction="vertical" size={0}>
            <span>{r.package_name}</span>
            <Typography.Text type="secondary" className="code-text" style={{ fontSize: 12 }}>
              {r.package_code}
            </Typography.Text>
          </Space>
        ),
      },
      { title: 'HLV', dataIndex: 'trainer_name' },
      { title: 'Chi nhánh', dataIndex: 'branch_name' },
      {
        title: 'Tiến độ',
        dataIndex: 'used_sessions',
        sorter: true,
        render: (_: number, r) => <SessionProgress r={r} />,
      },
      {
        title: 'Thời hạn',
        dataIndex: 'end_date',
        sorter: true,
        className: 'text-nowrap',
        render: (_: string, r) => (
          <Space direction="vertical" size={0}>
            <span>
              {formatDate(r.start_date)} – {formatDate(r.end_date)}
            </span>
            {r.days_left !== null && (
              <Typography.Text
                type={r.days_left <= 7 ? 'warning' : 'secondary'}
                style={{ fontSize: 12 }}
              >
                Còn {r.days_left} ngày
              </Typography.Text>
            )}
          </Space>
        ),
      },
      {
        title: 'Giá trị',
        dataIndex: 'price',
        sorter: true,
        align: 'right',
        render: (v: number) => <MoneyText value={v} />,
      },
      {
        title: 'Trạng thái',
        dataIndex: 'status',
        sorter: true,
        render: (v: string) => <StatusTag status={v} map={PT_CONTRACT_STATUS_MAP} />,
      },
      {
        title: 'Hóa đơn',
        key: 'invoice',
        render: (_, r) =>
          r.invoice_id ? (
            <Space direction="vertical" size={0}>
              <Link to={`/app/invoices/${r.invoice_id}`} className="code-text">
                {r.invoice_no}
              </Link>
              <StatusTag status={r.invoice_status} map={INVOICE_STATUS_MAP} />
            </Space>
          ) : (
            '—'
          ),
      },
      {
        title: '',
        key: 'actions',
        fixed: 'right',
        width: 56,
        render: (_, r) => (
          <Button
            type="text"
            size="small"
            icon={<EyeOutlined />}
            onClick={() => open(r.id)}
            aria-label="Chi tiết"
          />
        ),
      },
    ],
    [open],
  );

  return (
    <Card variant="borderless" className="page-card">
      <Row justify="space-between" align="middle" gutter={[8, 12]} className="page-head">
        <Col>
          <PageTitle title={TITLE} subtitle={SUBTITLE} />
        </Col>
        <Col>
          <ActionBar
            onAdd={() => setSelling(true)}
            addPerm={PERM.PT_SELL}
            addText="Bán gói PT"
            onRefresh={() => table.refetch()}
            refreshing={table.isFetching}
          >
            <Link to="/app/pt-sessions">
              <Button>Lịch PT</Button>
            </Link>
          </ActionBar>
        </Col>
      </Row>

      <FilterBar
        searchPlaceholder="Tên, SĐT, mã hội viên, HLV, số HĐ..."
        colTextSearch={6}
        filters={[
          {
            name: 'status',
            placeholder: 'Trạng thái',
            options: PT_CONTRACT_STATUS_OPTIONS,
            col: 3,
          },
          { name: 'pt_package_id', placeholder: 'Gói PT', options: packageOptions, col: 4 },
          { name: 'branch_id', placeholder: 'Chi nhánh', options: branchOptions, col: 4 },
          ...(trainerOnly
            ? []
            : [{ name: 'trainer_id', placeholder: 'HLV', options: trainerOptions, col: 3 }]),
        ]}
      />

      <DataTable<PtContract> table={table} columns={columns} emptyText="Chưa có hợp đồng PT nào" />

      <SellPtModal open={selling} onClose={() => setSelling(false)} onOpenContract={open} />
      <ContractDrawer id={viewId} onClose={() => url.set({ cid: undefined })} />
    </Card>
  );
}

function ContractsMobile() {
  const { branchId } = useWorkingBranch();
  const { trainerOnly, branchOptions, packageOptions, trainerOptions } = useFilterOptions();
  const [selling, setSelling] = useState(false);
  const [viewId, setViewId] = useState<number | null>(null);

  const fetchPage = useCallback(
    ({ page, limit, keyword, filters }: MobileFetchArgs) => {
      const f = filters as Record<string, unknown>;
      return ptApi.listContracts({
        page,
        limit,
        keyword: keyword || undefined,
        order_by: 'id',
        order_dir: 'DESC',
        status: (f.status as PtContractStatus) || undefined,
        pt_package_id: (f.pt_package_id as number) || undefined,
        trainer_id: (f.trainer_id as number) || undefined,
        branch_id: (f.branch_id as number | undefined) ?? branchId ?? undefined,
      });
    },
    [branchId],
  );

  return (
    <>
      <MobileList<PtContract>
        title={TITLE}
        queryKey={[ptApi.key, 'contracts', 'mobile', branchId]}
        fetchPage={fetchPage}
        onAdd={() => setSelling(true)}
        addPerm={PERM.PT_SELL}
        onOpen={(r) => setViewId(r.id)}
        searchPlaceholder="Tên, SĐT, mã hội viên, HLV"
        emptyText="Chưa có hợp đồng PT nào"
        filters={[
          { name: 'status', label: 'Trạng thái', options: PT_CONTRACT_STATUS_OPTIONS },
          { name: 'pt_package_id', label: 'Gói PT', options: packageOptions },
          { name: 'branch_id', label: 'Chi nhánh', options: branchOptions },
          ...(trainerOnly ? [] : [{ name: 'trainer_id', label: 'HLV', options: trainerOptions }]),
        ]}
        row={{
          title: (r) => r.member_name,
          subtitle: (r) => `${r.package_name} · HLV ${r.trainer_name}`,
          meta: (r) => [
            `${r.used_sessions}/${r.total_sessions} buổi`,
            `${formatDate(r.start_date)} – ${formatDate(r.end_date)}`,
            formatMoney(r.price),
          ],
          tags: (r) => [
            {
              label: PT_CONTRACT_STATUS_MAP[r.status]?.label ?? r.status,
              color: PT_CONTRACT_STATUS_MAP[r.status]?.color,
            },
          ],
        }}
        actions={(r) => [
          {
            key: 'view',
            label: 'Xem chi tiết',
            icon: <EyeOutlined />,
            onClick: () => setViewId(r.id),
          },
        ]}
        details={(r) => [
          ['Chi nhánh', r.branch_name],
          ['Hóa đơn', r.invoice_no || '—'],
          ['Đang đặt', `${r.scheduled_sessions} buổi`],
        ]}
      />
      <SellPtModal open={selling} onClose={() => setSelling(false)} onOpenContract={setViewId} />
      <ContractDrawer id={viewId} onClose={() => setViewId(null)} />
    </>
  );
}

export default function PtContractsPage() {
  return useIsMobile() ? <ContractsMobile /> : <ContractsDesktop />;
}
