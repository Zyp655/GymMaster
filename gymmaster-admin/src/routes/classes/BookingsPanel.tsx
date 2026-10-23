import { useMemo, useState } from 'react';
import {
  Alert,
  App,
  Badge,
  Button,
  Empty,
  Radio,
  Result,
  Segmented,
  Space,
  Table,
  Tooltip,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  CheckSquareOutlined,
  CloseOutlined,
  StopOutlined,
  UserAddOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import type { ClassBooking, ClassSchedule } from '@/api/class';
import ConfirmButton from '@/components/ConfirmButton';
import DateText from '@/components/DateText';
import { MemberSelect } from '@/components/MemberSelect';
import StatusTag from '@/components/StatusTag';
import { PERM } from '@/constants/permissions';
import { usePermission } from '@/hooks/usePermission';
import { getErrorMessage, showError } from '@/utils/errors';
import { formatPhone } from '@/utils/format';
import { BOOKING_SOURCE_LABEL, BOOKING_STATUS_MAP, attendanceWindow } from './constants';
import { useBookMember, useCancelBooking, useMarkAttendance, useScheduleBookings } from './queries';

type Tab = 'booked' | 'waitlist' | 'cancelled';
type Mark = 'ATTENDED' | 'NO_SHOW';

function MemberCell({ b }: { b: ClassBooking }) {
  return (
    <div style={{ lineHeight: 1.3 }}>
      <div style={{ fontWeight: 500 }}>{b.member_name}</div>
      <Typography.Text type="secondary" style={{ fontSize: 12 }}>
        {b.member_code}
        {b.member_phone ? ` · ${formatPhone(b.member_phone)}` : ''}
      </Typography.Text>
    </div>
  );
}

/** Danh sách đặt chỗ của một buổi lớp: đặt hộ, hủy hộ, waitlist, điểm danh. */
export default function BookingsPanel({ schedule }: { schedule: ClassSchedule }) {
  const { message } = App.useApp();
  const { can } = usePermission();
  const canBook = can(PERM.BOOKING_MANAGE);
  const bookings = useScheduleBookings(schedule.id);
  const book = useBookMember();
  const cancelBooking = useCancelBooking();
  const markAttendance = useMarkAttendance();

  const [tab, setTab] = useState<Tab>('booked');
  const [adding, setAdding] = useState(false);
  const [memberId, setMemberId] = useState<number | undefined>();
  const [bookError, setBookError] = useState<string | null>(null);
  const [attendMode, setAttendMode] = useState(false);
  const [marks, setMarks] = useState<Record<number, Mark>>({});

  const all = useMemo(() => bookings.data ?? [], [bookings.data]);
  const booked = all.filter((b) => ['BOOKED', 'ATTENDED', 'NO_SHOW'].includes(b.status));
  const waitlist = all
    .filter((b) => b.status === 'WAITLIST')
    .sort(
      (a, b) =>
        (a.waitlist_rank ?? a.waitlist_position ?? 0) -
        (b.waitlist_rank ?? b.waitlist_position ?? 0),
    );
  const cancelled = all.filter((b) => b.status === 'CANCELLED');

  const upcoming = schedule.status === 'SCHEDULED' && dayjs(schedule.start_at).isAfter(dayjs());
  const attWin = attendanceWindow(schedule);

  const startAttendance = () => {
    const init: Record<number, Mark> = {};
    booked.forEach((b) => {
      init[b.id] = b.status === 'NO_SHOW' ? 'NO_SHOW' : 'ATTENDED';
    });
    setMarks(init);
    setAttendMode(true);
    setTab('booked');
  };

  const saveAttendance = () => {
    const items = Object.entries(marks).map(([id, status]) => ({ booking_id: Number(id), status }));
    if (items.length === 0) return setAttendMode(false);
    markAttendance.mutate(
      { scheduleId: schedule.id, items },
      {
        onSuccess: (res) => {
          const present = items.filter((i) => i.status === 'ATTENDED').length;
          message.success(
            `Đã điểm danh ${res.updated} hội viên: ${present} có mặt, ${items.length - present} vắng`,
          );
          setAttendMode(false);
        },
        onError: (e) => showError(e, 'Không thể lưu điểm danh'),
      },
    );
  };

  const submitBooking = () => {
    if (!memberId) return;
    setBookError(null);
    book.mutate(
      { scheduleId: schedule.id, memberId },
      {
        onSuccess: (b) => {
          if (b.status === 'WAITLIST') {
            message.warning(
              `Lớp đã đủ chỗ – ${b.member_name} đã vào danh sách chờ (#${b.waitlist_rank ?? waitlist.length + 1})`,
            );
            setTab('waitlist');
          } else {
            message.success(`Đã đặt chỗ cho ${b.member_name}`);
            setTab('booked');
          }
          setMemberId(undefined);
          setAdding(false);
        },
        onError: (e) => setBookError(getErrorMessage(e, 'Không thể đặt chỗ')),
      },
    );
  };

  const cancelAction = (b: ClassBooking) =>
    canBook && (b.status === 'BOOKED' || b.status === 'WAITLIST') ? (
      <ConfirmButton
        type="text"
        size="small"
        danger
        icon={<StopOutlined />}
        okText="Hủy đặt chỗ"
        confirmTitle={`Hủy đặt chỗ của ${b.member_name}?`}
        confirmContent={
          b.status === 'BOOKED' && waitlist.length > 0
            ? `Chỗ trống sẽ tự động chuyển cho người đầu danh sách chờ (${waitlist[0].member_name}).`
            : 'Lượt đặt chỗ sẽ chuyển sang trạng thái đã hủy.'
        }
        onConfirm={() =>
          cancelBooking.mutateAsync(b.id).then(
            () => message.success('Đã hủy đặt chỗ'),
            (e) => {
              showError(e, 'Không thể hủy đặt chỗ');
              throw e;
            },
          )
        }
      >
        Hủy
      </ConfirmButton>
    ) : null;

  const bookedColumns: ColumnsType<ClassBooking> = [
    { title: '#', key: 'idx', width: 40, render: (_, __, i) => i + 1 },
    { title: 'Hội viên', key: 'member', render: (_, b) => <MemberCell b={b} /> },
    {
      title: 'Đặt lúc',
      dataIndex: 'booked_at',
      responsive: ['sm'],
      render: (v: string, b) => (
        <>
          <DateText value={v} withTime />
          <div>
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              {BOOKING_SOURCE_LABEL[b.source] ?? b.source}
            </Typography.Text>
          </div>
        </>
      ),
    },
    {
      title: attendMode ? 'Điểm danh' : 'Trạng thái',
      key: 'status',
      render: (_, b) =>
        attendMode ? (
          <Radio.Group
            size="small"
            optionType="button"
            buttonStyle="solid"
            value={marks[b.id]}
            onChange={(e) => setMarks((m) => ({ ...m, [b.id]: e.target.value as Mark }))}
            options={[
              { label: 'Có mặt', value: 'ATTENDED' },
              { label: 'Vắng', value: 'NO_SHOW' },
            ]}
          />
        ) : (
          <StatusTag status={b.status} map={BOOKING_STATUS_MAP} />
        ),
    },
    ...(attendMode
      ? []
      : [
          {
            title: '',
            key: 'actions',
            width: 70,
            render: (_: unknown, b: ClassBooking) => cancelAction(b),
          },
        ]),
  ];

  const waitlistColumns: ColumnsType<ClassBooking> = [
    {
      title: 'Thứ tự',
      key: 'pos',
      width: 64,
      render: (_, b, i) => <Badge count={b.waitlist_rank ?? i + 1} color="gold" showZero />,
    },
    { title: 'Hội viên', key: 'member', render: (_, b) => <MemberCell b={b} /> },
    {
      title: 'Vào danh sách lúc',
      dataIndex: 'booked_at',
      responsive: ['sm'],
      render: (v: string) => <DateText value={v} withTime />,
    },
    { title: '', key: 'actions', width: 70, render: (_, b) => cancelAction(b) },
  ];

  const cancelledColumns: ColumnsType<ClassBooking> = [
    { title: 'Hội viên', key: 'member', render: (_, b) => <MemberCell b={b} /> },
    {
      title: 'Đặt lúc',
      dataIndex: 'booked_at',
      responsive: ['sm'],
      render: (v: string) => <DateText value={v} withTime />,
    },
    {
      title: 'Hủy lúc',
      dataIndex: 'cancelled_at',
      render: (v: string | null) => <DateText value={v} withTime />,
    },
  ];

  if (bookings.isError)
    return (
      <Result
        status="warning"
        title="Không tải được danh sách đặt chỗ"
        subTitle={getErrorMessage(bookings.error)}
        extra={<Button onClick={() => bookings.refetch()}>Thử lại</Button>}
      />
    );

  const data = tab === 'booked' ? booked : tab === 'waitlist' ? waitlist : cancelled;
  const columns =
    tab === 'booked' ? bookedColumns : tab === 'waitlist' ? waitlistColumns : cancelledColumns;

  return (
    <div>
      <Space wrap style={{ width: '100%', justifyContent: 'space-between', marginBottom: 12 }}>
        <Segmented<Tab>
          value={tab}
          onChange={(v) => !attendMode && setTab(v)}
          disabled={attendMode}
          options={[
            { value: 'booked', label: `Đã đặt (${booked.length})` },
            { value: 'waitlist', label: `Chờ (${waitlist.length})` },
            { value: 'cancelled', label: `Đã hủy (${cancelled.length})` },
          ]}
        />
        {canBook && !attendMode && (
          <Space wrap>
            {upcoming && (
              <Button
                icon={<UserAddOutlined />}
                color="orange"
                variant="solid"
                onClick={() => {
                  setAdding((v) => !v);
                  setBookError(null);
                }}
              >
                Đặt chỗ cho hội viên
              </Button>
            )}
            <Tooltip title={attWin.open ? undefined : attWin.reason}>
              <Button
                icon={<CheckSquareOutlined />}
                disabled={!attWin.open || booked.length === 0}
                onClick={startAttendance}
              >
                Điểm danh
              </Button>
            </Tooltip>
          </Space>
        )}
      </Space>

      {adding && upcoming && (
        <div
          style={{
            border: '1px dashed #f97316',
            borderRadius: 8,
            padding: 12,
            marginBottom: 12,
            background: 'var(--gm-unread)',
          }}
        >
          <Typography.Text strong>Đặt chỗ tại quầy</Typography.Text>
          <Space.Compact style={{ width: '100%', marginTop: 8 }}>
            <MemberSelect
              value={memberId}
              onChange={(id) => {
                setMemberId(id);
                setBookError(null);
              }}
              autoFocus
            />
            <Button
              type="primary"
              loading={book.isPending}
              disabled={!memberId}
              onClick={submitBooking}
            >
              Đặt chỗ
            </Button>
            <Button icon={<CloseOutlined />} onClick={() => setAdding(false)} aria-label="Đóng" />
          </Space.Compact>
          <Typography.Text
            type="secondary"
            style={{ display: 'block', fontSize: 12, marginTop: 6 }}
          >
            {schedule.booked_count >= schedule.capacity
              ? 'Lớp đã đủ chỗ – hội viên sẽ vào danh sách chờ.'
              : `Còn ${schedule.capacity - schedule.booked_count} chỗ trống.`}{' '}
            Hội viên cần có gói tập đang hoạt động bao gồm lớp nhóm.
          </Typography.Text>
          {bookError && (
            <Alert
              type="error"
              showIcon
              style={{ marginTop: 8 }}
              message="Không thể đặt chỗ"
              description={bookError}
              closable
              onClose={() => setBookError(null)}
            />
          )}
        </div>
      )}

      {attendMode && (
        <Alert
          type="warning"
          showIcon
          style={{ marginBottom: 12 }}
          message="Chế độ điểm danh"
          description="Chọn Có mặt / Vắng cho từng hội viên rồi bấm Lưu điểm danh."
          action={
            <Space direction="vertical" size={4}>
              <Button
                size="small"
                onClick={() =>
                  setMarks(Object.fromEntries(booked.map((b) => [b.id, 'ATTENDED' as Mark])))
                }
              >
                Tất cả có mặt
              </Button>
            </Space>
          }
        />
      )}

      <Table<ClassBooking>
        rowKey="id"
        size="small"
        loading={bookings.isLoading}
        dataSource={data}
        columns={columns}
        pagination={false}
        scroll={{ x: 'max-content' }}
        locale={{
          emptyText: (
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description={
                tab === 'booked'
                  ? 'Chưa có hội viên đặt chỗ'
                  : tab === 'waitlist'
                    ? 'Danh sách chờ trống'
                    : 'Không có lượt hủy'
              }
            />
          ),
        }}
      />

      {attendMode && (
        <Space style={{ marginTop: 12, width: '100%', justifyContent: 'flex-end' }}>
          <Button onClick={() => setAttendMode(false)}>Hủy</Button>
          <Button type="primary" loading={markAttendance.isPending} onClick={saveAttendance}>
            Lưu điểm danh
          </Button>
        </Space>
      )}
    </div>
  );
}
