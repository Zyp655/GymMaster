import { useMemo, useState } from 'react';
import {
  App,
  Button,
  DatePicker,
  Empty,
  Result,
  Segmented,
  Space,
  Table,
  Tag,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { StopOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import classApi from '@/api/class';
import type { ClassBooking } from '@/api/class';
import ConfirmButton from '@/components/ConfirmButton';
import DateText from '@/components/DateText';
import StatusTag from '@/components/StatusTag';
import { PERM } from '@/constants/permissions';
import { usePermission } from '@/hooks/usePermission';
import { getErrorMessage, showError } from '@/utils/errors';
import { API_DATE_FORMAT, DATE_FORMAT } from '@/utils/format';
import {
  BOOKING_SOURCE_LABEL,
  BOOKING_STATUS_MAP,
  scheduleTrainerText,
  weekdayName,
} from './constants';
import { useCancelBooking } from './queries';

export interface MemberBookingsTabProps {
  memberId: number;
}

type Filter = 'all' | 'upcoming' | 'past';

/**
 * Lịch sử đặt lớp của một hội viên (`GET /admin/class-bookings?member_id=`) –
 * dùng trong tab "Lớp & PT" của hồ sơ hội viên.
 */
export default function MemberBookingsTab({ memberId }: MemberBookingsTabProps) {
  const { message } = App.useApp();
  const { can } = usePermission();
  const canBook = can(PERM.BOOKING_MANAGE);
  const cancelBooking = useCancelBooking();
  const [range, setRange] = useState<[Dayjs, Dayjs]>(() => [
    dayjs().subtract(30, 'day'),
    dayjs().add(30, 'day'),
  ]);
  const [filter, setFilter] = useState<Filter>('all');

  const params = {
    member_id: memberId,
    from: range[0].format(API_DATE_FORMAT),
    to: range[1].format(API_DATE_FORMAT),
  };
  const query = useQuery({
    queryKey: ['class-bookings', 'member', params],
    queryFn: () => classApi.memberBookings(params),
    enabled: memberId > 0,
  });

  const now = dayjs();
  const data = useMemo(() => {
    const list = [...(query.data ?? [])].sort((a, b) => b.start_at.localeCompare(a.start_at));
    if (filter === 'upcoming') return list.filter((b) => dayjs(b.start_at).isAfter(now));
    if (filter === 'past') return list.filter((b) => !dayjs(b.start_at).isAfter(now));
    return list;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [query.data, filter]);

  const summary = useMemo(() => {
    const list = query.data ?? [];
    const count = (st: string) => list.filter((b) => b.status === st).length;
    return {
      booked: count('BOOKED'),
      waitlist: count('WAITLIST'),
      attended: count('ATTENDED'),
      noShow: count('NO_SHOW'),
      cancelled: count('CANCELLED'),
    };
  }, [query.data]);

  const columns: ColumnsType<ClassBooking> = [
    {
      title: 'Buổi lớp',
      key: 'class',
      render: (_, b) => (
        <Space size={6} align="start">
          <span
            aria-hidden
            style={{
              marginTop: 5,
              width: 10,
              height: 10,
              borderRadius: 3,
              background: b.class_type_color,
              display: 'inline-block',
            }}
          />
          <div style={{ lineHeight: 1.3 }}>
            <Typography.Text strong delete={b.schedule_status === 'CANCELLED'}>
              {b.class_type_name}
            </Typography.Text>
            <div>
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                {scheduleTrainerText(b)} · {b.room_name}
              </Typography.Text>
            </div>
          </div>
        </Space>
      ),
    },
    {
      title: 'Thời gian',
      key: 'time',
      className: 'text-nowrap',
      render: (_, b) => {
        const s = dayjs(b.start_at);
        return (
          <>
            <div>
              {s.format('HH:mm')} – {dayjs(b.end_at).format('HH:mm')}
            </div>
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              {weekdayName(s)}, {s.format(DATE_FORMAT)}
            </Typography.Text>
          </>
        );
      },
    },
    { title: 'Chi nhánh', dataIndex: 'branch_name', responsive: ['lg'] },
    {
      title: 'Trạng thái',
      key: 'status',
      render: (_, b) => (
        <Space size={4} wrap>
          <StatusTag status={b.status} map={BOOKING_STATUS_MAP} />
          {b.status === 'WAITLIST' && (b.waitlist_rank ?? b.waitlist_position) && (
            <Tag bordered={false}>#{b.waitlist_rank ?? b.waitlist_position}</Tag>
          )}
          {b.schedule_status === 'CANCELLED' && (
            <Tag color="red" bordered={false}>
              Lớp đã hủy
            </Tag>
          )}
        </Space>
      ),
    },
    {
      title: 'Đặt lúc',
      dataIndex: 'booked_at',
      responsive: ['md'],
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
    ...(canBook
      ? [
          {
            title: '',
            key: 'actions',
            width: 70,
            render: (_: unknown, b: ClassBooking) =>
              (b.status === 'BOOKED' || b.status === 'WAITLIST') &&
              dayjs(b.start_at).isAfter(now) ? (
                <ConfirmButton
                  type="text"
                  size="small"
                  danger
                  icon={<StopOutlined />}
                  okText="Hủy đặt chỗ"
                  confirmTitle={`Hủy đặt chỗ lớp ${b.class_type_name}?`}
                  confirmContent={`Buổi ${dayjs(b.start_at).format('HH:mm DD/MM/YYYY')}. Chỗ trống sẽ chuyển cho người đầu danh sách chờ (nếu có).`}
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
              ) : null,
          },
        ]
      : []),
  ];

  if (query.isError)
    return (
      <Result
        status="warning"
        title="Không tải được lịch sử đặt lớp"
        subTitle={getErrorMessage(query.error)}
        extra={<Button onClick={() => query.refetch()}>Thử lại</Button>}
      />
    );

  return (
    <div>
      <Space wrap style={{ width: '100%', justifyContent: 'space-between', marginBottom: 12 }}>
        <Space wrap>
          <Segmented<Filter>
            value={filter}
            onChange={setFilter}
            options={[
              { value: 'all', label: 'Tất cả' },
              { value: 'upcoming', label: 'Sắp tới' },
              { value: 'past', label: 'Đã qua' },
            ]}
          />
          <DatePicker.RangePicker
            value={range}
            format={DATE_FORMAT}
            allowClear={false}
            onChange={(v) => v?.[0] && v[1] && setRange([v[0], v[1]])}
            disabledDate={(d, { from }) => !!from && Math.abs(d.diff(from, 'day')) > 365}
          />
        </Space>
        <Space wrap size={[4, 4]}>
          <Tag bordered={false} color="blue">
            Đã đặt {summary.booked}
          </Tag>
          <Tag bordered={false} color="gold">
            Chờ {summary.waitlist}
          </Tag>
          <Tag bordered={false} color="green">
            Có mặt {summary.attended}
          </Tag>
          <Tag bordered={false} color="red">
            Vắng {summary.noShow}
          </Tag>
          <Tag bordered={false}>Hủy {summary.cancelled}</Tag>
        </Space>
      </Space>
      <Table<ClassBooking>
        rowKey="id"
        size="small"
        loading={query.isLoading}
        dataSource={data}
        columns={columns}
        pagination={data.length > 20 ? { pageSize: 20, showSizeChanger: false } : false}
        scroll={{ x: 'max-content' }}
        locale={{
          emptyText: (
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description="Hội viên chưa đặt lớp nào trong khoảng thời gian này"
            />
          ),
        }}
      />
    </div>
  );
}
