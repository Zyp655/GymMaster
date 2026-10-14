import { useEffect, useMemo, useState } from 'react';
import {
  App,
  Button,
  Card,
  Col,
  DatePicker,
  Empty,
  Input,
  Result,
  Row,
  Space,
  Switch,
  Table,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  CopyOutlined,
  LeftOutlined,
  PlusOutlined,
  ReloadOutlined,
  RightOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import type { Shift } from '@/api/shift';
import type { StaffPosition } from '@/api/staff';
import BranchSelect from '@/components/BranchSelect';
import PageTitle from '@/components/PageTitle';
import StatusTag from '@/components/StatusTag';
import { PERM } from '@/constants/permissions';
import { useBranchOptions } from '@/hooks/useBranchOptions';
import { useIsMobile } from '@/hooks/useIsMobile';
import { usePermission } from '@/hooks/usePermission';
import { useUrlParams } from '@/hooks/useUrlParams';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { getErrorMessage } from '@/utils/errors';
import { API_DATE_FORMAT, normalizeText } from '@/utils/format';
import { POSITION_META } from '../staff/constants';
import ShiftForm from './ShiftForm';
import type { ShiftFormState } from './ShiftForm';
import { useCopyWeek, useStaffOptions, useWeekShifts } from './queries';
import { WEEKDAY_LABELS, shiftDate, shiftHours, shiftLabel, weekStart } from './utils';
import './shifts.css';

interface StaffRow {
  id: number;
  full_name: string;
  staff_code: string;
  position: StaffPosition;
  /** Ca theo ngày `YYYY-MM-DD`. */
  byDay: Record<string, Shift[]>;
  hours: number;
}

const formatHours = (h: number) => `${Number.isInteger(h) ? h : h.toFixed(1)} giờ`;

/** Lịch ca làm theo tuần: hàng = nhân viên, cột = 7 ngày. */
export default function ShiftsPage() {
  const isMobile = useIsMobile();
  const { can } = usePermission();
  const { modal, message } = App.useApp();
  const url = useUrlParams();
  const { branchId: workingBranch } = useWorkingBranch();
  const { branches } = useBranchOptions();
  const canManage = can(PERM.SHIFT_MANAGE);

  // Chi nhánh: URL → chi nhánh làm việc → chi nhánh đầu tiên được phép.
  const urlBranch = Number(url.get('branch_id')) || undefined;
  const branchId = urlBranch ?? workingBranch ?? branches[0]?.id;
  const branchName = branches.find((b) => b.id === branchId)?.name;

  const weekParam = url.get('week');
  const monday = useMemo(
    () => weekStart(weekParam && dayjs(weekParam).isValid() ? dayjs(weekParam) : dayjs()),
    [weekParam],
  );
  const days = useMemo(() => Array.from({ length: 7 }, (_, i) => monday.add(i, 'day')), [monday]);
  const from = monday.format(API_DATE_FORMAT);
  const to = monday.add(6, 'day').format(API_DATE_FORMAT);
  const today = dayjs().format(API_DATE_FORMAT);

  const shifts = useWeekShifts(branchId, from, to);
  const staff = useStaffOptions(branchId);
  const copy = useCopyWeek();

  const [keyword, setKeyword] = useState('');
  const [onlyScheduled, setOnlyScheduled] = useState(false);
  const [form, setForm] = useState<ShiftFormState>({ open: false, record: null });

  // Đổi chi nhánh làm việc ở header → bỏ chi nhánh đang ghim trên URL.
  useEffect(() => {
    if (workingBranch && urlBranch && workingBranch !== urlBranch)
      url.set({ branch_id: undefined });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [workingBranch]);

  const setWeek = (d: Dayjs) => {
    const w = weekStart(d);
    url.set({ week: w.isSame(weekStart(dayjs()), 'day') ? undefined : w.format(API_DATE_FORMAT) });
  };

  const rows = useMemo<StaffRow[]>(() => {
    const map = new Map<number, StaffRow>();
    (staff.data ?? []).forEach((s) => map.set(s.id, { ...s, byDay: {}, hours: 0 }));
    (shifts.data ?? []).forEach((sh) => {
      let row = map.get(sh.staff_id);
      if (!row) {
        row = {
          id: sh.staff_id,
          full_name: sh.staff_name,
          staff_code: sh.staff_code,
          position: sh.position,
          byDay: {},
          hours: 0,
        };
        map.set(sh.staff_id, row);
      }
      const day = shiftDate(sh.start_at);
      (row.byDay[day] ??= []).push(sh);
      row.hours += shiftHours(sh);
    });
    const kw = normalizeText(keyword.trim());
    return [...map.values()]
      .filter((r) => !onlyScheduled || r.hours > 0)
      .filter((r) => !kw || normalizeText(`${r.full_name} ${r.staff_code}`).includes(kw))
      .sort((a, b) => a.full_name.localeCompare(b.full_name, 'vi'));
  }, [staff.data, shifts.data, keyword, onlyScheduled]);

  const totalShifts = shifts.data?.length ?? 0;
  const totalHours = (shifts.data ?? []).reduce((n, s) => n + shiftHours(s), 0);

  const openAdd = (staffId?: number, date?: Dayjs) =>
    canManage && setForm({ open: true, record: null, staffId, date });
  const openEdit = (record: Shift) => setForm({ open: true, record });

  const confirmCopy = () => {
    if (!branchId) return;
    const next = monday.add(7, 'day');
    modal.confirm({
      title: 'Sao chép lịch sang tuần sau?',
      icon: <CopyOutlined />,
      content: (
        <>
          Sao chép toàn bộ {totalShifts} ca của <b>{branchName}</b> tuần {monday.format('DD/MM')}–
          {monday.add(6, 'day').format('DD/MM')} sang tuần {next.format('DD/MM')}–
          {next.add(6, 'day').format('DD/MM/YYYY')}. Ca trùng giờ hoặc nhân viên không còn thuộc chi
          nhánh sẽ được bỏ qua.
        </>
      ),
      okText: 'Sao chép',
      cancelText: 'Hủy',
      onOk: async () => {
        try {
          const res = await copy.mutateAsync({
            branch_id: branchId,
            from_week_start: from,
            to_week_start: next.format(API_DATE_FORMAT),
          });
          modal.success({
            title: 'Đã sao chép lịch tuần',
            content: (
              <>
                Tạo mới <b>{res.created}</b> ca, bỏ qua <b>{res.skipped}</b> ca (trùng giờ hoặc nhân
                viên không còn hợp lệ).
              </>
            ),
            okText: 'Xem tuần sau',
            onOk: () => setWeek(next),
          });
        } catch (e) {
          message.error(getErrorMessage(e, 'Không thể sao chép lịch'));
        }
      },
    });
  };

  const renderCell = (row: StaffRow, day: Dayjs) => {
    const key = day.format(API_DATE_FORMAT);
    const list = row.byDay[key] ?? [];
    return (
      <div
        className={`shift-cell${canManage ? ' shift-cell--clickable' : ''}`}
        onClick={() => openAdd(row.id, day)}
        role={canManage ? 'button' : undefined}
        title={canManage ? 'Bấm để thêm ca' : undefined}
      >
        {list.map((s) => (
          <Tooltip key={s.id} title={s.note || undefined}>
            <Tag
              color="blue"
              className="shift-tag"
              onClick={(e) => {
                e.stopPropagation();
                openEdit(s);
              }}
            >
              {shiftLabel(s)}
            </Tag>
          </Tooltip>
        ))}
        {canManage && list.length === 0 && <PlusOutlined className="shift-cell__plus" />}
      </div>
    );
  };

  const columns: ColumnsType<StaffRow> = [
    {
      title: 'Nhân viên',
      key: 'staff',
      fixed: 'left',
      width: 200,
      render: (_, r) => (
        <div>
          <div style={{ fontWeight: 500 }}>{r.full_name}</div>
          <Space size={4}>
            <Typography.Text type="secondary" className="code-text" style={{ fontSize: 12 }}>
              {r.staff_code}
            </Typography.Text>
            <StatusTag status={r.position} map={POSITION_META} />
          </Space>
        </div>
      ),
    },
    ...days.map((d, i) => {
      const key = d.format(API_DATE_FORMAT);
      return {
        title: (
          <div className={`shift-day${key === today ? ' shift-day--today' : ''}`}>
            <div>{WEEKDAY_LABELS[i]}</div>
            <div className="shift-day__date">{d.format('DD/MM')}</div>
          </div>
        ),
        key,
        width: 120,
        align: 'center' as const,
        className: key === today ? 'shift-col--today' : undefined,
        onCell: () => ({ style: { padding: 4, verticalAlign: 'top' as const } }),
        render: (_: unknown, r: StaffRow) => renderCell(r, d),
      };
    }),
    {
      title: 'Tổng',
      key: 'hours',
      fixed: 'right',
      width: 80,
      align: 'right',
      render: (_, r) => (r.hours ? formatHours(r.hours) : '—'),
    },
  ];

  const weekPicker = (
    <Space wrap size={8}>
      <Button icon={<LeftOutlined />} onClick={() => setWeek(monday.subtract(7, 'day'))}>
        {isMobile ? null : 'Tuần trước'}
      </Button>
      <DatePicker
        picker="week"
        allowClear={false}
        value={monday}
        onChange={(d) => d && setWeek(d)}
        format={(d) =>
          `${weekStart(d).format('DD/MM')} – ${weekStart(d).add(6, 'day').format('DD/MM/YYYY')}`
        }
        style={{ width: 200 }}
      />
      <Button onClick={() => setWeek(monday.add(7, 'day'))}>
        {isMobile ? null : 'Tuần sau'} <RightOutlined />
      </Button>
      <Button onClick={() => setWeek(dayjs())}>Hôm nay</Button>
    </Space>
  );

  const error = shifts.isError || staff.isError;

  return (
    <Card variant="borderless" className="page-card">
      <Row justify="space-between" align="middle" gutter={[8, 12]} className="page-head">
        <Col>
          <PageTitle title="Ca làm việc" subtitle="Lịch ca của nhân viên theo tuần" />
        </Col>
        <Col>
          <Space wrap size={8}>
            {canManage && (
              <Button
                color="orange"
                variant="solid"
                icon={<PlusOutlined />}
                disabled={!branchId}
                onClick={() => openAdd()}
              >
                Thêm ca
              </Button>
            )}
            {canManage && (
              <Button
                icon={<CopyOutlined />}
                disabled={!branchId || totalShifts === 0}
                loading={copy.isPending}
                onClick={confirmCopy}
              >
                Sao chép sang tuần sau
              </Button>
            )}
            <Button
              icon={<ReloadOutlined spin={shifts.isFetching} />}
              onClick={() => {
                void shifts.refetch();
                void staff.refetch();
              }}
            >
              Làm mới
            </Button>
          </Space>
        </Col>
      </Row>

      <Row gutter={[8, 8]} className="filter-bar" align="middle">
        <Col xs={24} sm={12} md={5}>
          <BranchSelect
            value={branchId ?? null}
            onChange={(v) => url.set({ branch_id: (v as number | null) ?? undefined })}
            activeOnly
          />
        </Col>
        <Col xs={24} md="auto">
          {weekPicker}
        </Col>
        <Col xs={24} sm={12} md={4}>
          <Input.Search
            allowClear
            placeholder="Tìm nhân viên"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
          />
        </Col>
        <Col>
          <Space size={6}>
            <Switch size="small" checked={onlyScheduled} onChange={setOnlyScheduled} />
            <span>Chỉ người có ca</span>
          </Space>
        </Col>
        <Col flex="auto" style={{ textAlign: 'right' }}>
          <Typography.Text type="secondary">
            {totalShifts} ca · {formatHours(totalHours)}
          </Typography.Text>
        </Col>
      </Row>

      {!branchId ? (
        <Empty description="Chọn chi nhánh để xem lịch ca" />
      ) : error ? (
        <Result
          status="warning"
          title="Không tải được lịch ca"
          subTitle={getErrorMessage(shifts.error ?? staff.error)}
          extra={
            <Button
              type="primary"
              onClick={() => {
                void shifts.refetch();
                void staff.refetch();
              }}
            >
              Thử lại
            </Button>
          }
        />
      ) : isMobile ? (
        <MobileWeek
          days={days}
          today={today}
          shifts={shifts.data ?? []}
          loading={shifts.isLoading}
          canManage={canManage}
          onAdd={(d) => openAdd(undefined, d)}
          onOpen={openEdit}
        />
      ) : (
        <Table<StaffRow>
          size="small"
          bordered
          rowKey="id"
          columns={columns}
          dataSource={rows}
          loading={shifts.isFetching || staff.isFetching}
          pagination={false}
          scroll={{ x: 'max-content' }}
          className="shift-table"
          locale={{
            emptyText: (
              <Empty
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                description={
                  keyword || onlyScheduled
                    ? 'Không có nhân viên phù hợp'
                    : 'Chi nhánh chưa có nhân viên đang làm việc'
                }
              />
            ),
          }}
        />
      )}

      {branchId && (
        <ShiftForm
          {...form}
          branchId={branchId}
          branchName={branchName}
          staffOptions={staff.data ?? []}
          staffLoading={staff.isLoading}
          onClose={() => setForm((f) => ({ ...f, open: false }))}
        />
      )}
    </Card>
  );
}

/** Bản điện thoại: danh sách ca theo từng ngày. */
function MobileWeek({
  days,
  today,
  shifts,
  loading,
  canManage,
  onAdd,
  onOpen,
}: {
  days: Dayjs[];
  today: string;
  shifts: Shift[];
  loading: boolean;
  canManage: boolean;
  onAdd: (d: Dayjs) => void;
  onOpen: (s: Shift) => void;
}) {
  return (
    <Space direction="vertical" style={{ width: '100%' }} size={8}>
      {days.map((d, i) => {
        const key = d.format(API_DATE_FORMAT);
        const list = shifts.filter((s) => shiftDate(s.start_at) === key);
        return (
          <Card
            key={key}
            size="small"
            loading={loading}
            className={key === today ? 'shift-mobile-day--today' : undefined}
            title={`${WEEKDAY_LABELS[i]} · ${d.format('DD/MM')}`}
            extra={
              canManage && (
                <Button size="small" type="text" icon={<PlusOutlined />} onClick={() => onAdd(d)}>
                  Thêm
                </Button>
              )
            }
          >
            {list.length === 0 ? (
              <Typography.Text type="secondary">Chưa có ca</Typography.Text>
            ) : (
              <Space direction="vertical" size={4} style={{ width: '100%' }}>
                {list.map((s) => (
                  <button
                    key={s.id}
                    type="button"
                    className="shift-mobile-item"
                    onClick={() => onOpen(s)}
                  >
                    <Tag color="blue" style={{ marginRight: 8 }}>
                      {shiftLabel(s)}
                    </Tag>
                    <span>{s.staff_name}</span>
                    {s.note && <Typography.Text type="secondary"> · {s.note}</Typography.Text>}
                  </button>
                ))}
              </Space>
            )}
          </Card>
        );
      })}
    </Space>
  );
}
