import { useMemo, useState } from 'react';
import {
  Button,
  Card,
  Col,
  DatePicker,
  Row,
  Segmented,
  Select,
  Space,
  Switch,
  Tag,
  Tooltip,
} from 'antd';
import {
  AppstoreOutlined,
  LeftOutlined,
  PlusOutlined,
  ReloadOutlined,
  RightOutlined,
  UnorderedListOutlined,
} from '@ant-design/icons';
import { keepPreviousData, useQuery } from '@tanstack/react-query';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import classApi from '@/api/class';
import type { ClassSchedule, ClassScheduleListParams } from '@/api/class';
import PageTitle from '@/components/PageTitle';
import { PERM } from '@/constants/permissions';
import { useIsMobile } from '@/hooks/useIsMobile';
import { usePermission } from '@/hooks/usePermission';
import { useUrlParams } from '@/hooks/useUrlParams';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { getErrorMessage } from '@/utils/errors';
import { API_DATE_FORMAT, normalizeText } from '@/utils/format';
import CancelScheduleModal from './CancelScheduleModal';
import ScheduleDrawer from './ScheduleDrawer';
import ScheduleFormModal from './ScheduleFormModal';
import { ScheduleAgenda, ScheduleTable } from './ScheduleListView';
import BranchFilterSelect from './BranchFilterSelect';
import WeekCalendar from './WeekCalendar';
import { startOfWeek } from './constants';
import { useBranchHours, useClassTypeOptions, useRoomOptions, useTrainerOptions } from './queries';
import './classes.css';

type View = 'week' | 'list';

const numOrUndef = (v?: string) => (v && Number(v) > 0 ? Number(v) : undefined);
const filterByLabel = (input: string, option?: { label?: unknown }) =>
  normalizeText(String(option?.label ?? '')).includes(normalizeText(input));

/** Lịch lớp nhóm: lưới tuần / danh sách, thêm lịch, chi tiết buổi + đặt chỗ + điểm danh. */
export default function ClassSchedulesPage() {
  const isMobile = useIsMobile();
  const { can } = usePermission();
  const canManage = can(PERM.CLASS_MANAGE);
  const { branchId: workingBranchId } = useWorkingBranch();
  const { get, set } = useUrlParams();

  // ---- trạng thái trên URL (F5 giữ nguyên)
  const weekParam = get('week');
  const weekStart = useMemo(
    () => startOfWeek(weekParam && dayjs(weekParam).isValid() ? dayjs(weekParam) : dayjs()),
    [weekParam],
  );
  const view: View = get('view') === 'list' ? 'list' : 'week';
  const branchParam = get('branch_id');
  const branchId =
    branchParam === 'all' ? null : (numOrUndef(branchParam) ?? workingBranchId ?? null);
  const trainerId = numOrUndef(get('trainer_id'));
  const classTypeId = numOrUndef(get('class_type_id'));
  const roomId = numOrUndef(get('room_id'));
  const showCancelled = get('cancelled') !== '0';

  // ---- modal/drawer
  const [selected, setSelected] = useState<ClassSchedule | null>(null);
  const [form, setForm] = useState<{
    open: boolean;
    record: ClassSchedule | null;
    at: Dayjs | null;
  }>({ open: false, record: null, at: null });
  const [cancelTarget, setCancelTarget] = useState<ClassSchedule | null>(null);

  // ---- dữ liệu
  const params: ClassScheduleListParams = {
    from: weekStart.format(API_DATE_FORMAT),
    to: weekStart.add(6, 'day').format(API_DATE_FORMAT),
    branch_id: branchId ?? undefined,
    trainer_id: trainerId,
    class_type_id: classTypeId,
    room_id: roomId,
  };
  const schedules = useQuery({
    queryKey: [classApi.key, 'list', params],
    queryFn: () => classApi.listSchedules(params),
    placeholderData: keepPreviousData,
  });
  const types = useClassTypeOptions();
  const trainers = useTrainerOptions(branchId);
  const rooms = useRoomOptions(branchId);
  const hours = useBranchHours();

  const items = useMemo(
    () => (schedules.data ?? []).filter((s) => showCancelled || s.status !== 'CANCELLED'),
    [schedules.data, showCancelled],
  );

  const [openTime, closeTime] = useMemo(() => {
    const list = (hours.data ?? []).filter((b) => !branchId || b.id === branchId);
    if (list.length === 0) return ['06:00', '22:00'];
    return [
      list.map((b) => b.open_time).sort()[0],
      list
        .map((b) => b.close_time)
        .sort()
        .reverse()[0],
    ];
  }, [hours.data, branchId]);

  const stats = useMemo(() => {
    const active = (schedules.data ?? []).filter((s) => s.status !== 'CANCELLED');
    return {
      total: active.length,
      booked: active.reduce((n, s) => n + s.booked_count, 0),
      capacity: active.reduce((n, s) => n + s.capacity, 0),
      full: active.filter((s) => s.booked_count >= s.capacity).length,
      waitlist: active.reduce((n, s) => n + s.waitlist_count, 0),
      cancelled: (schedules.data ?? []).length - active.length,
    };
  }, [schedules.data]);

  const legend = useMemo(() => {
    const map = new Map<number, { name: string; color: string }>();
    (schedules.data ?? []).forEach((s) =>
      map.set(s.class_type_id, { name: s.class_type_name, color: s.class_type_color }),
    );
    return [...map.values()];
  }, [schedules.data]);

  const goWeek = (d: Dayjs) => {
    const w = startOfWeek(d);
    set({ week: w.isSame(startOfWeek(dayjs()), 'day') ? undefined : w.format(API_DATE_FORMAT) });
  };

  const isCurrentWeek = weekStart.isSame(startOfWeek(dayjs()), 'day');
  const weekLabel = `${weekStart.format('DD/MM')} – ${weekStart.add(6, 'day').format('DD/MM/YYYY')}`;

  const openCreate = (at: Dayjs | null = null) => setForm({ open: true, record: null, at });

  const toolbar = (
    <Row gutter={[8, 8]} align="middle" className="class-toolbar" style={{ marginBottom: 12 }}>
      <Col xs={24} lg="auto">
        <Space wrap>
          <Button onClick={() => goWeek(dayjs())} disabled={isCurrentWeek}>
            Hôm nay
          </Button>
          <Space.Compact>
            <Tooltip title="Tuần trước">
              <Button
                icon={<LeftOutlined />}
                onClick={() => goWeek(weekStart.subtract(1, 'week'))}
              />
            </Tooltip>
            <DatePicker
              picker="week"
              value={weekStart}
              allowClear={false}
              format={() => weekLabel}
              onChange={(d) => d && goWeek(d)}
              style={{ width: 190 }}
            />
            <Tooltip title="Tuần sau">
              <Button icon={<RightOutlined />} onClick={() => goWeek(weekStart.add(1, 'week'))} />
            </Tooltip>
          </Space.Compact>
          {!isMobile && (
            <Segmented<View>
              value={view}
              onChange={(v) => set({ view: v === 'week' ? undefined : v })}
              options={[
                { value: 'week', label: 'Tuần', icon: <AppstoreOutlined /> },
                { value: 'list', label: 'Danh sách', icon: <UnorderedListOutlined /> },
              ]}
            />
          )}
        </Space>
      </Col>
      <Col xs={24} sm={12} lg={4}>
        <BranchFilterSelect
          value={branchId}
          onChange={(v) =>
            set({
              branch_id: v === null ? 'all' : v,
              trainer_id: undefined,
              room_id: undefined,
            })
          }
        />
      </Col>
      <Col xs={12} sm={12} lg={4}>
        <Select
          allowClear
          showSearch
          placeholder="Tất cả HLV"
          value={trainerId}
          loading={trainers.isLoading}
          options={(trainers.data ?? []).map((t) => ({ value: t.id, label: t.full_name }))}
          filterOption={filterByLabel}
          onChange={(v?: number) => set({ trainer_id: v })}
          style={{ width: '100%' }}
        />
      </Col>
      <Col xs={12} sm={12} lg={4}>
        <Select
          allowClear
          showSearch
          placeholder="Tất cả loại lớp"
          value={classTypeId}
          loading={types.isLoading}
          options={(types.data ?? []).map((t) => ({ value: t.id, label: t.name }))}
          filterOption={filterByLabel}
          onChange={(v?: number) => set({ class_type_id: v })}
          style={{ width: '100%' }}
        />
      </Col>
      <Col xs={12} sm={12} lg={3}>
        <Select
          allowClear
          showSearch
          placeholder="Tất cả phòng"
          value={roomId}
          loading={rooms.isLoading}
          options={(rooms.data ?? []).map((r) => ({ value: r.id, label: r.name }))}
          filterOption={filterByLabel}
          onChange={(v?: number) => set({ room_id: v })}
          style={{ width: '100%' }}
        />
      </Col>
      <Col xs={12} sm={12} lg="auto">
        <Space size={6}>
          <Switch
            size="small"
            checked={showCancelled}
            onChange={(v) => set({ cancelled: v ? undefined : '0' })}
          />
          <span style={{ whiteSpace: 'nowrap' }}>Hiện buổi đã hủy</span>
        </Space>
      </Col>
    </Row>
  );

  return (
    <Card variant="borderless" className="page-card">
      <Row justify="space-between" align="middle" gutter={[8, 12]} className="page-head">
        <Col>
          <PageTitle title="Lịch lớp" subtitle={`Tuần ${weekLabel}`} />
        </Col>
        <Col>
          <Space>
            <Tooltip title="Tải lại">
              <Button
                icon={<ReloadOutlined spin={schedules.isFetching} />}
                onClick={() => schedules.refetch()}
              />
            </Tooltip>
            {canManage && (
              <Button
                color="orange"
                variant="solid"
                icon={<PlusOutlined />}
                onClick={() => openCreate()}
              >
                Thêm lịch
              </Button>
            )}
          </Space>
        </Col>
      </Row>

      {toolbar}

      <Space wrap size={[6, 6]} style={{ marginBottom: 12 }}>
        <Tag bordered={false}>{stats.total} buổi</Tag>
        <Tag bordered={false} color="blue">
          {stats.booked}/{stats.capacity} chỗ đã đặt
        </Tag>
        {stats.full > 0 && (
          <Tag bordered={false} color="red">
            {stats.full} lớp đã đầy
          </Tag>
        )}
        {stats.waitlist > 0 && (
          <Tag bordered={false} color="gold">
            {stats.waitlist} người chờ
          </Tag>
        )}
        {stats.cancelled > 0 && <Tag bordered={false}>{stats.cancelled} buổi đã hủy</Tag>}
        {!isMobile &&
          legend.map((l) => (
            <Tag key={l.name} color={l.color} style={{ marginInlineEnd: 0 }}>
              {l.name}
            </Tag>
          ))}
      </Space>

      {schedules.isError ? (
        <div style={{ padding: 24, textAlign: 'center' }}>
          <p>{getErrorMessage(schedules.error, 'Không tải được lịch lớp')}</p>
          <Button onClick={() => schedules.refetch()}>Thử lại</Button>
        </div>
      ) : isMobile ? (
        <ScheduleAgenda
          schedules={items}
          weekStart={weekStart}
          loading={schedules.isLoading}
          onSelect={setSelected}
        />
      ) : view === 'list' ? (
        <ScheduleTable schedules={items} loading={schedules.isLoading} onSelect={setSelected} />
      ) : (
        <div style={{ opacity: schedules.isFetching && !schedules.isLoading ? 0.7 : 1 }}>
          <WeekCalendar
            weekStart={weekStart}
            schedules={items}
            openTime={openTime}
            closeTime={closeTime}
            onSelect={setSelected}
            onCreateAt={canManage ? (at) => openCreate(at) : undefined}
          />
        </div>
      )}

      <ScheduleDrawer
        schedule={selected}
        onClose={() => setSelected(null)}
        onEdit={(s) => setForm({ open: true, record: s, at: null })}
        onCancel={setCancelTarget}
      />
      <ScheduleFormModal
        open={form.open}
        record={form.record}
        branchId={branchId}
        initialAt={form.at}
        onClose={() => setForm((f) => ({ ...f, open: false }))}
      />
      <CancelScheduleModal
        open={!!cancelTarget}
        schedule={cancelTarget}
        onClose={() => setCancelTarget(null)}
      />
    </Card>
  );
}
