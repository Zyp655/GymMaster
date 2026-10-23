import { useMemo, useState } from 'react';
import {
  Button,
  Card,
  Col,
  DatePicker,
  Empty,
  Result,
  Row,
  Segmented,
  Select,
  Skeleton,
  Space,
  Switch,
  Tag,
  Tooltip,
} from 'antd';
import {
  CalendarOutlined,
  LeftOutlined,
  PlusOutlined,
  ReloadOutlined,
  RightOutlined,
  TeamOutlined,
} from '@ant-design/icons';
import { keepPreviousData, useQuery } from '@tanstack/react-query';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import ptApi from '@/api/pt';
import type {
  PtSession,
  PtSessionListParams,
  PtSessionStatus,
  TrainerScheduleItem,
} from '@/api/pt';
import BranchFilterSelect from '@/routes/classes/BranchFilterSelect';
import PageTitle from '@/components/PageTitle';
import { PERM } from '@/constants/permissions';
import { useIsMobile } from '@/hooks/useIsMobile';
import { usePermission } from '@/hooks/usePermission';
import { useUrlParams } from '@/hooks/useUrlParams';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { startOfWeek, weekdayName } from '@/routes/classes/constants';
import { useBranchHours } from '@/routes/classes/queries';
import { getErrorMessage } from '@/utils/errors';
import { API_DATE_FORMAT, normalizeText } from '@/utils/format';
import ContractDrawer from './ContractDrawer';
import PtWeekCalendar from './PtWeekCalendar';
import type { CalendarEvent } from './PtWeekCalendar';
import SessionDrawer from './SessionDrawer';
import { BookSessionModal } from './SessionModals';
import {
  PT_SESSION_COLOR,
  PT_SESSION_STATUS_MAP,
  PT_SESSION_STATUS_OPTIONS,
  isTrainerOnly,
  legendStatusKeys,
  sessionStatusKey,
} from './constants';
import { usePtTrainerOptions } from './queries';

type Mode = 'pt' | 'trainer';

interface Ev extends CalendarEvent {
  /** id buổi PT (null với lớp nhóm). */
  sessionId: number | null;
  statusKey: string;
}

const numOrUndef = (v?: string) => (v && Number(v) > 0 ? Number(v) : undefined);
const filterByLabel = (input: string, option?: { label?: unknown }) =>
  normalizeText(String(option?.label ?? '')).includes(normalizeText(input));

function sessionTooltip(s: PtSession) {
  const st = dayjs(s.start_at);
  return (
    <div style={{ fontSize: 12, lineHeight: 1.5 }}>
      <b>{s.member_name}</b> · {s.member_code}
      <div>
        {st.format('HH:mm')}–{dayjs(s.end_at).format('HH:mm')} · {weekdayName(st)}{' '}
        {st.format('DD/MM')}
      </div>
      <div>HLV: {s.trainer_name}</div>
      <div>{s.package_name}</div>
      <div>{PT_SESSION_STATUS_MAP[sessionStatusKey(s)]?.label}</div>
    </div>
  );
}

function fromSession(s: PtSession): Ev {
  const statusKey = sessionStatusKey(s);
  return {
    key: `pt-${s.id}`,
    sessionId: s.id,
    statusKey,
    start_at: s.start_at,
    end_at: s.end_at,
    title: s.member_name,
    lines: [`HLV ${s.trainer_name}`, s.package_name],
    color: PT_SESSION_COLOR[statusKey] ?? PT_SESSION_COLOR.SCHEDULED,
    cancelled: s.status === 'CANCELLED',
    tooltip: sessionTooltip(s),
  };
}

function fromScheduleItem(it: TrainerScheduleItem, viewedTrainerId?: number): Ev {
  if (it.type === 'PT') {
    return {
      key: `pt-${it.id}`,
      sessionId: it.id,
      statusKey: it.status,
      start_at: it.start_at,
      end_at: it.end_at,
      title: it.title,
      lines: [it.branch_name],
      color: PT_SESSION_COLOR[it.status] ?? PT_SESSION_COLOR.SCHEDULED,
      cancelled: it.status === 'CANCELLED',
      tooltip: (
        <div style={{ fontSize: 12 }}>
          <b>{it.title}</b>
          <div>
            {dayjs(it.start_at).format('HH:mm')}–{dayjs(it.end_at).format('HH:mm')} ·{' '}
            {it.branch_name}
          </div>
          <div>{PT_SESSION_STATUS_MAP[it.status]?.label ?? it.status}</div>
        </div>
      ),
    };
  }
  const slots =
    it.booked_count !== null && it.capacity !== null ? `${it.booked_count}/${it.capacity} chỗ` : '';
  // Lớp dạy thay: API đưa vào lịch của người dạy thay, trainer_id vẫn là HLV chính.
  const substitute = !!viewedTrainerId && !!it.trainer_id && it.trainer_id !== viewedTrainerId;
  return {
    key: `class-${it.id}`,
    sessionId: null,
    statusKey: 'CLASS',
    start_at: it.start_at,
    end_at: it.end_at,
    title: substitute ? `Dạy thay: ${it.title}` : `Lớp: ${it.title}`,
    lines: [substitute ? `Thay ${it.trainer_name}` : '', it.room_name, slots].filter(Boolean),
    color: it.color || '#F97316',
    readOnly: true,
    tooltip: (
      <div style={{ fontSize: 12 }}>
        <b>Lớp nhóm – {it.title}</b>
        {substitute && <div>Dạy thay cho HLV {it.trainer_name}</div>}
        <div>
          {dayjs(it.start_at).format('HH:mm')}–{dayjs(it.end_at).format('HH:mm')} · {it.room_name}
        </div>
        <div>{it.branch_name}</div>
        {slots && <div>Đã đặt {slots}</div>}
        <div style={{ opacity: 0.8 }}>Chỉ xem – quản lý ở trang Lịch lớp</div>
      </div>
    ),
  };
}

/** Danh sách theo ngày (điện thoại). */
function Agenda({
  events,
  weekStart,
  onSelect,
}: {
  events: Ev[];
  weekStart: Dayjs;
  onSelect: (e: Ev) => void;
}) {
  const days = Array.from({ length: 7 }, (_, i) => weekStart.add(i, 'day'));
  const today = dayjs();
  const nonEmpty = days.filter((d) => events.some((e) => dayjs(e.start_at).isSame(d, 'day')));
  if (nonEmpty.length === 0)
    return (
      <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="Không có buổi nào trong tuần" />
    );
  return (
    <div>
      {nonEmpty.map((d) => (
        <div key={d.valueOf()} className="agenda-day">
          <div className={`agenda-day-title${d.isSame(today, 'day') ? ' is-today' : ''}`}>
            {weekdayName(d)} <small>{d.format('DD/MM/YYYY')}</small>
          </div>
          {events
            .filter((e) => dayjs(e.start_at).isSame(d, 'day'))
            .sort((a, b) => a.start_at.localeCompare(b.start_at))
            .map((e) => (
              <div
                key={e.key}
                role="button"
                tabIndex={0}
                className={`agenda-item${e.cancelled ? ' is-cancelled' : ''}`}
                style={{
                  ['--agenda-color' as string]: e.color,
                  cursor: e.readOnly ? 'default' : 'pointer',
                }}
                onClick={() => !e.readOnly && onSelect(e)}
              >
                <div className="agenda-time">
                  {dayjs(e.start_at).format('HH:mm')}
                  <small>{dayjs(e.end_at).format('HH:mm')}</small>
                </div>
                <div>
                  <div className="agenda-name">{e.title}</div>
                  <div style={{ fontSize: 12, color: '#6b7280' }}>
                    {(e.lines ?? []).join(' · ')}
                  </div>
                  {e.statusKey !== 'CLASS' && (
                    <Tag
                      bordered={false}
                      color={PT_SESSION_STATUS_MAP[e.statusKey]?.color}
                      style={{ marginTop: 4 }}
                    >
                      {PT_SESSION_STATUS_MAP[e.statusKey]?.label ?? e.statusKey}
                    </Tag>
                  )}
                </div>
              </div>
            ))}
        </div>
      ))}
    </div>
  );
}

/** Lịch PT: lịch tuần buổi PT theo chi nhánh/HLV hoặc lịch dạy của một HLV (PT + lớp nhóm). */
export default function PtSessionsPage() {
  const isMobile = useIsMobile();
  const { can, user } = usePermission();
  const canSchedule = can(PERM.PT_SCHEDULE);
  const trainerOnly = isTrainerOnly(user);
  const { branchId: workingBranchId } = useWorkingBranch();
  const { get, set } = useUrlParams();

  const weekParam = get('week');
  const weekStart = useMemo(
    () => startOfWeek(weekParam && dayjs(weekParam).isValid() ? dayjs(weekParam) : dayjs()),
    [weekParam],
  );
  const modeParam = get('mode');
  const mode: Mode =
    modeParam === 'pt' || modeParam === 'trainer' ? modeParam : trainerOnly ? 'trainer' : 'pt';
  const branchParam = get('branch_id');
  const branchId =
    branchParam === 'all' ? null : (numOrUndef(branchParam) ?? workingBranchId ?? null);
  const trainerId = trainerOnly ? undefined : numOrUndef(get('trainer_id'));
  const status = (get('status') as PtSessionStatus | undefined) || undefined;
  const showCancelled = get('cancelled') === '1';

  const [sessionId, setSessionId] = useState<number | null>(null);
  const [contractId, setContractId] = useState<number | null>(null);
  const [bookAt, setBookAt] = useState<{ open: boolean; at: Dayjs | null }>({
    open: false,
    at: null,
  });

  const from = weekStart.format(API_DATE_FORMAT);
  const to = weekStart.add(6, 'day').format(API_DATE_FORMAT);

  const ptParams: PtSessionListParams = {
    from,
    to,
    branch_id: branchId ?? undefined,
    trainer_id: trainerId,
    status,
  };
  const sessions = useQuery({
    queryKey: [ptApi.key, 'sessions', ptParams],
    queryFn: () => ptApi.listSessions(ptParams),
    placeholderData: keepPreviousData,
    enabled: mode === 'pt',
  });

  const trainerReady = mode === 'trainer' && (trainerOnly || !!trainerId);
  const schedParams = { from, to, trainer_id: trainerId };
  const schedule = useQuery({
    queryKey: [ptApi.key, 'trainer-schedule', schedParams],
    queryFn: () => ptApi.trainerSchedule(schedParams),
    placeholderData: keepPreviousData,
    enabled: trainerReady,
  });

  const trainers = usePtTrainerOptions(branchId, !trainerOnly);
  const hours = useBranchHours();
  const active = mode === 'pt' ? sessions : schedule;

  const events = useMemo<Ev[]>(() => {
    if (mode === 'pt')
      return (sessions.data ?? [])
        .filter((s) => showCancelled || status === 'CANCELLED' || s.status !== 'CANCELLED')
        .map(fromSession);
    const viewed = trainerId ?? user?.id;
    return (schedule.data ?? []).map((it) => fromScheduleItem(it, viewed));
  }, [mode, sessions.data, schedule.data, showCancelled, status, trainerId, user?.id]);

  const stats = useMemo(() => {
    const map = new Map<string, number>();
    events.forEach((e) => map.set(e.statusKey, (map.get(e.statusKey) ?? 0) + 1));
    return map;
  }, [events]);

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

  const goWeek = (d: Dayjs) => {
    const w = startOfWeek(d);
    set({ week: w.isSame(startOfWeek(dayjs()), 'day') ? undefined : w.format(API_DATE_FORMAT) });
  };
  const isCurrentWeek = weekStart.isSame(startOfWeek(dayjs()), 'day');
  const weekLabel = `${weekStart.format('DD/MM')} – ${weekStart.add(6, 'day').format('DD/MM/YYYY')}`;

  const onSelect = (e: Ev) => e.sessionId && setSessionId(e.sessionId);
  const onCreateAt = canSchedule ? (at: Dayjs) => setBookAt({ open: true, at }) : undefined;

  const trainerSelect = !trainerOnly && (
    <Select
      allowClear={mode === 'pt'}
      showSearch
      placeholder={mode === 'pt' ? 'Tất cả HLV' : 'Chọn HLV'}
      value={trainerId}
      loading={trainers.isLoading}
      options={(trainers.data ?? []).map((t) => ({ value: t.id, label: t.full_name }))}
      filterOption={filterByLabel}
      onChange={(v?: number) => set({ trainer_id: v })}
      style={{ width: '100%' }}
      status={mode === 'trainer' && !trainerId ? 'warning' : undefined}
    />
  );

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
          <Segmented<Mode>
            value={mode}
            onChange={(v) => set({ mode: v })}
            options={[
              { value: 'pt', label: 'Lịch PT', icon: <CalendarOutlined /> },
              {
                value: 'trainer',
                label: trainerOnly ? 'Lịch của tôi' : 'Lịch của HLV',
                icon: <TeamOutlined />,
              },
            ]}
          />
        </Space>
      </Col>
      {mode === 'pt' && (
        <Col xs={24} sm={12} lg={4}>
          <BranchFilterSelect
            value={branchId}
            onChange={(v) => set({ branch_id: v === null ? 'all' : v, trainer_id: undefined })}
          />
        </Col>
      )}
      {trainerSelect && (
        <Col xs={24} sm={12} lg={4}>
          {trainerSelect}
        </Col>
      )}
      {mode === 'pt' && (
        <>
          <Col xs={12} sm={12} lg={3}>
            <Select
              allowClear
              placeholder="Trạng thái"
              value={status}
              options={PT_SESSION_STATUS_OPTIONS}
              onChange={(v?: string) => set({ status: v })}
              style={{ width: '100%' }}
            />
          </Col>
          <Col xs={12} sm={12} lg="auto">
            <Space size={6}>
              <Switch
                size="small"
                checked={showCancelled}
                onChange={(v) => set({ cancelled: v ? '1' : undefined })}
              />
              <span style={{ whiteSpace: 'nowrap' }}>Hiện buổi đã hủy</span>
            </Space>
          </Col>
        </>
      )}
    </Row>
  );

  const legend = (
    <Space wrap size={[6, 6]} style={{ marginBottom: 12 }}>
      <Tag bordered={false}>{events.length} mục</Tag>
      {/* Chú thích luôn đủ màu mọi trạng thái (kể cả Vắng mặt) để đọc được lịch dù tuần chưa có buổi đó.
          Lịch HLV không phân biệt NO_SHOW đã chốt / chưa chốt → bỏ mục "Quá giờ chưa xác nhận". */}
      {legendStatusKeys(mode).map((k) => (
        <Tag key={k} bordered={false} style={stats.get(k) ? undefined : { opacity: 0.6 }}>
          <span className="pt-legend-dot" style={{ background: PT_SESSION_COLOR[k] }} />
          {PT_SESSION_STATUS_MAP[k].label}: {stats.get(k) ?? 0}
        </Tag>
      ))}
      {mode === 'trainer' && (
        <Tag bordered={false}>
          <span className="pt-legend-dot is-dashed" />
          Lớp nhóm (chỉ xem): {stats.get('CLASS') ?? 0}
        </Tag>
      )}
    </Space>
  );

  let content;
  if (mode === 'trainer' && !trainerReady) {
    content = (
      <Empty
        style={{ padding: 48 }}
        description="Chọn huấn luyện viên để xem lịch dạy (buổi PT + lớp nhóm)"
      />
    );
  } else if (active.isError) {
    content = (
      <Result
        status="warning"
        title="Không tải được lịch"
        subTitle={getErrorMessage(active.error)}
        extra={<Button onClick={() => active.refetch()}>Thử lại</Button>}
      />
    );
  } else if (active.isLoading) {
    content = <Skeleton active paragraph={{ rows: 8 }} />;
  } else if (isMobile) {
    content = <Agenda events={events} weekStart={weekStart} onSelect={onSelect} />;
  } else {
    content = (
      <div style={{ opacity: active.isFetching ? 0.7 : 1 }}>
        <PtWeekCalendar<Ev>
          weekStart={weekStart}
          events={events}
          openTime={openTime}
          closeTime={closeTime}
          onSelect={onSelect}
          onCreateAt={onCreateAt}
          countLabel="mục"
        />
      </div>
    );
  }

  return (
    <Card variant="borderless" className="page-card">
      <Row justify="space-between" align="middle" gutter={[8, 12]} className="page-head">
        <Col>
          <PageTitle title="Lịch PT" subtitle={`Tuần ${weekLabel}`} />
        </Col>
        <Col>
          <Space>
            <Tooltip title="Tải lại">
              <Button
                icon={<ReloadOutlined spin={active.isFetching} />}
                onClick={() => active.refetch()}
              />
            </Tooltip>
            {canSchedule && (
              <Button
                color="orange"
                variant="solid"
                icon={<PlusOutlined />}
                onClick={() => setBookAt({ open: true, at: null })}
              >
                Đặt buổi
              </Button>
            )}
          </Space>
        </Col>
      </Row>

      {toolbar}
      {legend}
      {content}

      <SessionDrawer
        sessionId={sessionId}
        onClose={() => setSessionId(null)}
        onOpenContract={(id) => setContractId(id)}
      />
      <ContractDrawer id={contractId} onClose={() => setContractId(null)} />
      <BookSessionModal
        open={bookAt.open}
        initialAt={bookAt.at}
        trainerId={trainerId}
        onClose={() => setBookAt({ open: false, at: null })}
      />
    </Card>
  );
}
