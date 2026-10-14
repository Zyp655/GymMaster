import { useEffect, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  App,
  Button,
  Card,
  Col,
  DatePicker,
  Empty,
  Form,
  InputNumber,
  Modal,
  Progress,
  Row,
  Select,
  Space,
  Spin,
  Table,
  Tabs,
  Tag,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  AimOutlined,
  DeleteOutlined,
  DashboardOutlined,
  PlusOutlined,
  ReloadOutlined,
  TableOutlined,
  TeamOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import {
  KPI_METRIC_LABELS,
  KPI_METRIC_OPTIONS,
  kpiAnalyticsApi,
  OCCUPANCY_LEVEL_META,
} from '@/api/operations-ext';
import type {
  BranchOccupancyItem,
  CohortRow,
  KPIMetric,
  KPIReportItem,
  OccupancyLevel,
  SalesTargetInput,
} from '@/api/operations-ext';
import staffApi from '@/api/staff';
import BranchSelect from '@/components/BranchSelect';
import ConfirmButton from '@/components/ConfirmButton';
import PageTitle from '@/components/PageTitle';
import { PERM } from '@/constants/permissions';
import { useBranchOptions } from '@/hooks/useBranchOptions';
import { usePermission } from '@/hooks/usePermission';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { showError } from '@/utils/errors';
import { formatMoney, formatNumber } from '@/utils/format';

/* --------------------------------------------------------------------------
 * TAB 1: Mật độ phòng tập Realtime (/public/branches/occupancy)
 * -------------------------------------------------------------------------- */
function OccupancyRealtimeTab() {
  const { branchId } = useWorkingBranch();
  const occQuery = useQuery({
    queryKey: [kpiAnalyticsApi.occupancyKey, branchId],
    queryFn: () => kpiAnalyticsApi.occupancy(branchId),
    refetchInterval: 60_000,
  });

  const items: BranchOccupancyItem[] = occQuery.data ?? [];

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Row justify="space-between" align="middle">
        <Col>
          <Typography.Text type="secondary">
            Dữ liệu cập nhật tự động mỗi 60 giây dựa trên lượt hội viên đang tập (Check-in trong 2 giờ gần nhất chưa Check-out).
          </Typography.Text>
        </Col>
        <Col>
          <Button
            icon={<ReloadOutlined spin={occQuery.isFetching} />}
            onClick={() => occQuery.refetch()}
          >
            Làm mới ngay
          </Button>
        </Col>
      </Row>

      {occQuery.isLoading ? (
        <div style={{ textAlign: 'center', padding: 48 }}>
          <Spin size="large" />
        </div>
      ) : items.length === 0 ? (
        <Card>
          <Empty description="Chưa có dữ liệu mật độ phòng tập" />
        </Card>
      ) : (
        <Row gutter={[16, 16]}>
          {items.map((b) => {
            const present = Number(b.present ?? b.current_count ?? 0);
            const capacity = Number(b.capacity || 120);
            const pct = Math.min(
              100,
              Math.round(Number(b.pct ?? b.occupancy_pct ?? (capacity > 0 ? (present / capacity) * 100 : 0))),
            );
            const level: OccupancyLevel =
              b.level ||
              (pct >= 90 ? 'FULL' : pct >= 70 ? 'BUSY' : pct >= 35 ? 'NORMAL' : 'QUIET');
            const meta = OCCUPANCY_LEVEL_META[level] ?? OCCUPANCY_LEVEL_META.QUIET;

            return (
              <Col xs={24} md={12} lg={8} key={b.branch_id}>
                <Card
                  size="small"
                  title={
                    <Space>
                      <TeamOutlined />
                      <Typography.Text strong style={{ fontSize: 15 }}>
                        {b.branch_name || `Chi nhánh #${b.branch_id}`}
                      </Typography.Text>
                    </Space>
                  }
                  extra={<Tag color={meta.color}>{meta.label}</Tag>}
                >
                  <Space direction="vertical" size={10} style={{ width: '100%' }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline' }}>
                      <div>
                        <span style={{ fontSize: 28, fontWeight: 700, color: '#0A3150' }}>
                          {present}
                        </span>
                        <span style={{ color: '#64748b', fontSize: 14 }}>
                          {' '}
                          / {capacity} hội viên
                        </span>
                      </div>
                      <Typography.Text strong style={{ color: meta.strokeColor }}>
                        {pct}% sức chứa
                      </Typography.Text>
                    </div>

                    <Progress
                      percent={pct}
                      strokeColor={meta.strokeColor}
                      showInfo={false}
                      size={['100%', 10]}
                    />

                    {b.peak_hours && b.peak_hours.length > 0 && (
                      <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                        Khung giờ cao điểm thường gặp:{' '}
                        {b.peak_hours.map((h) => `${h}:00`).join(', ')}
                      </Typography.Text>
                    )}
                  </Space>
                </Card>
              </Col>
            );
          })}
        </Row>
      )}
    </Space>
  );
}

/* --------------------------------------------------------------------------
 * TAB 2: Chỉ tiêu doanh số KPI (/admin/sales-targets, /admin/reports/kpi)
 * -------------------------------------------------------------------------- */
function KpiTargetsTab({ canManage }: { canManage: boolean }) {
  const { message } = App.useApp();
  const qc = useQueryClient();
  const { branchId } = useWorkingBranch();
  const { branches } = useBranchOptions();

  const [periodDayjs, setPeriodDayjs] = useState<Dayjs>(dayjs());
  const [selectedBranch, setSelectedBranch] = useState<number | undefined>(branchId ?? undefined);
  const [modalOpen, setModalOpen] = useState(false);
  const [form] = Form.useForm<{
    branch_id: number;
    staff_id?: number;
    period: Dayjs;
    metric: KPIMetric;
    target_value: number;
  }>();

  useEffect(() => {
    if (branchId) setSelectedBranch(branchId);
  }, [branchId]);

  const periodStr = periodDayjs.format('YYYY-MM');

  const staffQuery = useQuery({
    queryKey: [staffApi.key, 'options', 'kpi', selectedBranch ?? branchId],
    queryFn: () => staffApi.options(selectedBranch ?? branchId),
    enabled: modalOpen,
  });

  const kpiQuery = useQuery({
    queryKey: [kpiAnalyticsApi.kpiReportKey, selectedBranch ?? branchId, periodStr],
    queryFn: () =>
      kpiAnalyticsApi.kpiReport({
        branch_id: selectedBranch ?? branchId ?? undefined,
        period: periodStr,
      }),
  });

  const targetsQuery = useQuery({
    queryKey: [kpiAnalyticsApi.targetKey, selectedBranch ?? branchId, periodStr],
    queryFn: () =>
      kpiAnalyticsApi.listTargets({
        branch_id: selectedBranch ?? branchId ?? undefined,
        period: periodStr,
        limit: 100,
      }),
  });

  const createTargetMut = useMutation({
    mutationFn: (v: {
      branch_id: number;
      staff_id?: number;
      period: Dayjs;
      metric: KPIMetric;
      target_value: number;
    }) => {
      const payload: SalesTargetInput = {
        branch_id: v.branch_id,
        staff_id: v.staff_id ?? null,
        period: v.period.format('YYYY-MM'),
        metric: v.metric,
        target_value: Number(v.target_value) || 0,
      };
      return kpiAnalyticsApi.createTarget(payload);
    },
    onSuccess: () => {
      message.success('Đã thiết lập chỉ tiêu KPI mới!');
      void qc.invalidateQueries({ queryKey: [kpiAnalyticsApi.kpiReportKey] });
      void qc.invalidateQueries({ queryKey: [kpiAnalyticsApi.targetKey] });
      setModalOpen(false);
    },
    onError: (err) => showError(err, 'Không thể lưu chỉ tiêu KPI'),
  });

  const deleteTargetMut = useMutation({
    mutationFn: (id: number) => kpiAnalyticsApi.deleteTarget(id),
    onSuccess: () => {
      message.success('Đã xóa chỉ tiêu KPI');
      void qc.invalidateQueries({ queryKey: [kpiAnalyticsApi.kpiReportKey] });
      void qc.invalidateQueries({ queryKey: [kpiAnalyticsApi.targetKey] });
    },
    onError: (err) => showError(err, 'Không thể xóa chỉ tiêu KPI'),
  });

  const rows: KPIReportItem[] = useMemo(() => {
    const reportItems = kpiQuery.data?.items ?? [];
    if (reportItems.length > 0) return reportItems;
    return (targetsQuery.data?.items ?? []).map((t) => ({
      id: t.id,
      target_id: t.id,
      branch_id: t.branch_id,
      branch_name: t.branch_name,
      staff_id: t.staff_id,
      staff_name: t.staff_name,
      period: t.period,
      metric: t.metric,
      target_value: t.target_value,
      actual_value: t.actual_value ?? 0,
      pct: t.pct ?? 0,
    }));
  }, [kpiQuery.data, targetsQuery.data]);

  const formatMetricVal = (metric: KPIMetric, val: number) => {
    if (metric === 'REVENUE' || metric === 'PT_SALES') {
      return formatMoney(val);
    }
    return `${formatNumber(val)} lượt`;
  };

  const columns = useMemo<ColumnsType<KPIReportItem>>(
    () => [
      {
        title: 'Kỳ KPI',
        dataIndex: 'period',
        width: 100,
        render: (v: string) => <Tag className="code-text">{v || periodStr}</Tag>,
      },
      {
        title: 'Chi nhánh / Nhân sự phụ trách',
        key: 'scope',
        render: (_, r) => (
          <Space direction="vertical" size={0}>
            <Typography.Text strong>
              {r.branch_name || `Chi nhánh #${r.branch_id}`}
            </Typography.Text>
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              {r.staff_name
                ? `Cá nhân: ${r.staff_name}`
                : r.staff_id
                  ? `Nhân sự #${r.staff_id}`
                  : 'Chỉ tiêu chung toàn chi nhánh'}
            </Typography.Text>
          </Space>
        ),
      },
      {
        title: 'Chỉ số đo lường (Metric)',
        dataIndex: 'metric',
        render: (m: KPIMetric) => (
          <Tag color="geekblue">{KPI_METRIC_LABELS[m] ?? m}</Tag>
        ),
      },
      {
        title: 'Mục tiêu (Target)',
        dataIndex: 'target_value',
        align: 'right',
        render: (v: number, r) => (
          <Typography.Text strong>{formatMetricVal(r.metric, v)}</Typography.Text>
        ),
      },
      {
        title: 'Thực đạt (Actual)',
        dataIndex: 'actual_value',
        align: 'right',
        render: (v: number, r) => (
          <Typography.Text strong style={{ color: '#0A3150' }}>
            {formatMetricVal(r.metric, v || 0)}
          </Typography.Text>
        ),
      },
      {
        title: 'Tiến độ hoàn thành KPI',
        key: 'progress',
        width: 240,
        render: (_, r) => {
          const target = Number(r.target_value) || 0;
          const actual = Number(r.actual_value) || 0;
          const pct = Number(
            r.achievement_pct ?? r.pct ?? (target > 0 ? Math.round((actual / target) * 1000) / 10 : 0),
          );
          return (
            <Progress
              percent={Math.min(100, Math.round(pct))}
              format={() => `${pct}%`}
              status={pct >= 100 ? 'success' : 'active'}
              strokeColor={pct >= 100 ? '#16a34a' : pct >= 70 ? '#2563eb' : '#ea580c'}
            />
          );
        },
      },
      {
        title: '',
        key: 'actions',
        width: 70,
        render: (_, r) => {
          const tid = r.target_id ?? r.id;
          return canManage && tid ? (
            <ConfirmButton
              perm={PERM.KPI_MANAGE}
              type="text"
              size="small"
              danger
              icon={<DeleteOutlined />}
              confirmTitle="Xóa chỉ tiêu KPI này?"
              onConfirm={() => deleteTargetMut.mutateAsync(tid)}
              aria-label="Xóa"
            />
          ) : null;
        },
      },
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [canManage, periodStr],
  );

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Card size="small">
        <Row justify="space-between" align="middle" gutter={[12, 12]}>
          <Col>
            <Space wrap>
              {!branchId && (
                <Select
                  allowClear
                  placeholder="Tất cả chi nhánh"
                  style={{ width: 200 }}
                  value={selectedBranch}
                  onChange={(v) => setSelectedBranch(v)}
                  options={branches.map((b) => ({ value: b.id, label: b.name }))}
                />
              )}
              <DatePicker
                picker="month"
                format="YYYY-MM"
                allowClear={false}
                value={periodDayjs}
                onChange={(d) => d && setPeriodDayjs(d)}
              />
            </Space>
          </Col>
          <Col>
            <Space>
              {canManage && (
                <Button
                  color="orange"
                  variant="solid"
                  icon={<PlusOutlined />}
                  onClick={() => {
                    form.resetFields();
                    form.setFieldsValue({
                      branch_id: selectedBranch ?? branchId ?? branches[0]?.id ?? 1,
                      period: periodDayjs,
                      metric: 'REVENUE',
                      target_value: 150000000,
                    });
                    setModalOpen(true);
                  }}
                >
                  Thiết lập chỉ tiêu KPI
                </Button>
              )}
              <Button
                icon={<ReloadOutlined spin={kpiQuery.isFetching || targetsQuery.isFetching} />}
                onClick={() => {
                  void kpiQuery.refetch();
                  void targetsQuery.refetch();
                }}
              >
                Làm mới
              </Button>
            </Space>
          </Col>
        </Row>
      </Card>

      <Card size="small" title={`Bảng theo dõi tiến độ KPI thực tế vs mục tiêu – Kỳ ${periodStr}`}>
        <Table<KPIReportItem>
          size="small"
          rowKey={(r) => `${r.id ?? r.target_id ?? ''}_${r.branch_id}_${r.staff_id ?? 0}_${r.metric}`}
          loading={kpiQuery.isLoading || targetsQuery.isLoading}
          columns={columns}
          dataSource={rows}
          pagination={false}
          locale={{
            emptyText: (
              <Empty description="Chưa có chỉ tiêu KPI nào trong kỳ này. Hãy bấm 'Thiết lập chỉ tiêu KPI'." />
            ),
          }}
        />
      </Card>

      <Modal
        open={modalOpen}
        title="Thiết lập Chỉ tiêu Doanh số KPI (/admin/sales-targets)"
        okText="Lưu chỉ tiêu KPI"
        cancelText="Hủy"
        onOk={() => form.submit()}
        onCancel={() => setModalOpen(false)}
        confirmLoading={createTargetMut.isPending}
        width={520}
        forceRender
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={(v) => createTargetMut.mutate(v)}
          style={{ marginTop: 12 }}
        >
          <Row gutter={12}>
            <Col xs={12}>
              <Form.Item
                name="branch_id"
                label="Chi nhánh"
                rules={[{ required: true, message: 'Chọn chi nhánh' }]}
              >
                <BranchSelect activeOnly />
              </Form.Item>
            </Col>
            <Col xs={12}>
              <Form.Item
                name="period"
                label="Kỳ KPI (YYYY-MM)"
                rules={[{ required: true, message: 'Chọn kỳ' }]}
              >
                <DatePicker picker="month" format="YYYY-MM" style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col xs={24}>
              <Form.Item name="staff_id" label="Giao cho nhân sự cụ thể (Để trống = Chỉ tiêu cả chi nhánh)">
                <Select
                  allowClear
                  showSearch
                  optionFilterProp="label"
                  placeholder="Chỉ tiêu chung toàn chi nhánh"
                  options={(staffQuery.data ?? []).map((s) => ({
                    value: s.id,
                    label: `${s.full_name} (${s.staff_code})`,
                  }))}
                />
              </Form.Item>
            </Col>
            <Col xs={12}>
              <Form.Item name="metric" label="Loại chỉ tiêu KPI" rules={[{ required: true }]}>
                <Select options={KPI_METRIC_OPTIONS} />
              </Form.Item>
            </Col>
            <Col xs={12}>
              <Form.Item
                name="target_value"
                label="Giá trị mục tiêu"
                rules={[{ required: true, message: 'Nhập mục tiêu' }]}
              >
                <InputNumber min={1} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Modal>
    </Space>
  );
}

/* --------------------------------------------------------------------------
 * TAB 3: Ma trận Cohort Retention (/admin/reports/cohort)
 * -------------------------------------------------------------------------- */
function CohortRetentionTab() {
  const { branchId } = useWorkingBranch();
  const { branches } = useBranchOptions();
  const [selectedBranch, setSelectedBranch] = useState<number | undefined>(branchId ?? undefined);

  useEffect(() => {
    if (branchId) setSelectedBranch(branchId);
  }, [branchId]);

  const cohortQuery = useQuery({
    queryKey: [kpiAnalyticsApi.cohortKey, selectedBranch ?? branchId],
    queryFn: () =>
      kpiAnalyticsApi.cohortReport({
        branch_id: selectedBranch ?? branchId ?? undefined,
      }),
  });

  const cohorts = cohortQuery.data?.cohorts ?? [];
  const maxMonths = cohortQuery.data?.max_months ?? 6;

  const cellBgColor = (pct: number | undefined) => {
    if (pct === undefined || pct === null) return 'transparent';
    if (pct >= 80) return '#1e3a8a';
    if (pct >= 60) return '#2563eb';
    if (pct >= 40) return '#60a5fa';
    if (pct >= 20) return '#bfdbfe';
    if (pct > 0) return '#eff6ff';
    return '#f8fafc';
  };

  const cellTextColor = (pct: number | undefined) => {
    if (pct === undefined || pct === null) return '#94a3b8';
    return pct >= 40 ? '#ffffff' : '#0f172a';
  };

  const columns = useMemo<ColumnsType<CohortRow>>(() => {
    const monthCols: ColumnsType<CohortRow> = Array.from({ length: maxMonths }, (_, idx) => ({
      title: `Tháng +${idx}`,
      key: `m_${idx}`,
      align: 'center',
      width: 105,
      render: (_, r) => {
        const cellObj = r.months?.find((m) => m.month_index === idx) ?? r.months?.[idx];
        const pct =
          cellObj?.pct ??
          r.retention_pct?.[idx] ??
          (idx === 0 && (r.cohort_size || r.initial_members) ? 100 : undefined);

        if (pct === undefined) return <span style={{ color: '#cbd5e1' }}>—</span>;

        return (
          <div
            style={{
              background: cellBgColor(pct),
              color: cellTextColor(pct),
              borderRadius: 6,
              padding: '4px 8px',
              fontWeight: 600,
              fontSize: 12,
            }}
          >
            {pct}%
            {cellObj?.active_count !== undefined ? ` (${cellObj.active_count})` : ''}
          </div>
        );
      },
    }));

    return [
      {
        title: 'Tháng gia nhập (Cohort)',
        dataIndex: 'cohort_month',
        fixed: 'left',
        width: 160,
        render: (v: string, r) => (
          <Typography.Text strong className="code-text">
            {r.label || v}
          </Typography.Text>
        ),
      },
      {
        title: 'Quy mô nhóm (HV)',
        key: 'size',
        width: 140,
        align: 'right',
        render: (_, r) => (
          <Tag color="blue">{formatNumber(r.cohort_size ?? r.initial_members ?? 0)} HV</Tag>
        ),
      },
      ...monthCols,
    ];
  }, [maxMonths]);

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Card size="small">
        <Row justify="space-between" align="middle">
          <Col>
            <Space>
              {!branchId && (
                <Select
                  allowClear
                  placeholder="Toàn hệ thống"
                  style={{ width: 200 }}
                  value={selectedBranch}
                  onChange={(v) => setSelectedBranch(v)}
                  options={branches.map((b) => ({ value: b.id, label: b.name }))}
                />
              )}
              <Typography.Text type="secondary">
                Tỷ lệ % hội viên còn duy trì gói tập ACTIVE ở các tháng tiếp theo kể từ tháng đăng ký đầu tiên.
              </Typography.Text>
            </Space>
          </Col>
          <Col>
            <Button
              icon={<ReloadOutlined spin={cohortQuery.isFetching} />}
              onClick={() => cohortQuery.refetch()}
            >
              Làm mới
            </Button>
          </Col>
        </Row>
      </Card>

      <Card size="small" title="Ma trận giữ chân hội viên theo nhóm tháng gia nhập (Cohort Retention Matrix)">
        <Table<CohortRow>
          size="small"
          rowKey="cohort_month"
          scroll={{ x: 'max-content' }}
          loading={cohortQuery.isLoading}
          columns={columns}
          dataSource={cohorts}
          pagination={false}
          locale={{ emptyText: <Empty description="Chưa có dữ liệu phân tích Cohort Retention" /> }}
        />
      </Card>
    </Space>
  );
}

export default function KpiAnalyticsPage() {
  const { can } = usePermission();
  const canManage = can(PERM.KPI_MANAGE);

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <PageTitle
        title="Mật độ phòng tập, Chỉ tiêu KPI & Cohort Retention"
        subtitle="Giám sát công suất phòng tập thời gian thực, theo dõi tiến độ doanh số KPI và ma trận giữ chân hội viên theo tháng"
      />

      <Tabs
        defaultActiveKey="occupancy"
        items={[
          {
            key: 'occupancy',
            label: (
              <Space size={6}>
                <DashboardOutlined />
                <span>Mật độ phòng tập Realtime</span>
              </Space>
            ),
            children: <OccupancyRealtimeTab />,
          },
          {
            key: 'kpi',
            label: (
              <Space size={6}>
                <AimOutlined />
                <span>Chỉ tiêu doanh số KPI</span>
              </Space>
            ),
            children: <KpiTargetsTab canManage={canManage} />,
          },
          {
            key: 'cohort',
            label: (
              <Space size={6}>
                <TableOutlined />
                <span>Ma trận Cohort Retention</span>
              </Space>
            ),
            children: <CohortRetentionTab />,
          },
        ]}
      />
    </Space>
  );
}
