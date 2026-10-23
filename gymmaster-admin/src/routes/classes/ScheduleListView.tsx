import { useMemo } from 'react';
import type { CSSProperties } from 'react';
import { Empty, Progress, Skeleton, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import type { ClassSchedule } from '@/api/class';
import StatusTag from '@/components/StatusTag';
import {
  SCHEDULE_STATUS_MAP,
  scheduleTrainer,
  scheduleTrainerText,
  weekdayName,
} from './constants';

interface ListProps {
  schedules: ClassSchedule[];
  loading?: boolean;
  onSelect: (s: ClassSchedule) => void;
}

/** Ô "đã đặt/sức chứa" + chờ. */
export function SlotText({ s }: { s: ClassSchedule }) {
  if (s.status === 'CANCELLED') return <Typography.Text type="secondary">—</Typography.Text>;
  const full = s.booked_count >= s.capacity;
  return (
    <Space size={4} wrap>
      <Tag color={full ? 'red' : 'blue'} bordered={false} style={{ marginInlineEnd: 0 }}>
        {s.booked_count}/{s.capacity}
      </Tag>
      {s.waitlist_count > 0 && (
        <Tag color="gold" bordered={false}>
          {s.waitlist_count} chờ
        </Tag>
      )}
    </Space>
  );
}

/** Chế độ "Danh sách" trên desktop: bảng các buổi trong tuần. */
export function ScheduleTable({ schedules, loading, onSelect }: ListProps) {
  const columns = useMemo<ColumnsType<ClassSchedule>>(
    () => [
      {
        title: 'Ngày',
        key: 'date',
        className: 'text-nowrap',
        render: (_, s) => {
          const d = dayjs(s.start_at);
          return (
            <>
              <div>{weekdayName(d)}</div>
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                {d.format('DD/MM/YYYY')}
              </Typography.Text>
            </>
          );
        },
      },
      {
        title: 'Giờ',
        key: 'time',
        className: 'text-nowrap',
        render: (_, s) =>
          `${dayjs(s.start_at).format('HH:mm')} – ${dayjs(s.end_at).format('HH:mm')}`,
      },
      {
        title: 'Lớp',
        key: 'class',
        render: (_, s) => (
          <Space size={6}>
            <span
              aria-hidden
              style={{
                width: 10,
                height: 10,
                borderRadius: 3,
                background: s.class_type_color,
                display: 'inline-block',
              }}
            />
            <Typography.Link
              delete={s.status === 'CANCELLED'}
              onClick={() => onSelect(s)}
              style={{ fontWeight: 500 }}
            >
              {s.class_type_name}
            </Typography.Link>
          </Space>
        ),
      },
      {
        title: 'HLV',
        key: 'trainer',
        render: (_, s) => {
          const t = scheduleTrainer(s);
          return t.substituted ? (
            <Space direction="vertical" size={0}>
              <span>
                {t.name}{' '}
                <Tag color="orange" bordered={false} style={{ marginInlineEnd: 0 }}>
                  Dạy thay
                </Tag>
              </span>
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                Thay cho {t.original}
              </Typography.Text>
            </Space>
          ) : (
            t.name
          );
        },
      },
      { title: 'Phòng', dataIndex: 'room_name' },
      { title: 'Chi nhánh', dataIndex: 'branch_name', responsive: ['xl'] },
      {
        title: 'Đã đặt',
        key: 'slots',
        render: (_, s) =>
          s.status === 'CANCELLED' ? (
            <SlotText s={s} />
          ) : (
            <Space direction="vertical" size={0} style={{ minWidth: 90 }}>
              <SlotText s={s} />
              <Progress
                percent={Math.round((s.booked_count / Math.max(s.capacity, 1)) * 100)}
                showInfo={false}
                size="small"
                status={s.booked_count >= s.capacity ? 'exception' : 'normal'}
              />
            </Space>
          ),
      },
      {
        title: 'Trạng thái',
        dataIndex: 'status',
        render: (v: string) => <StatusTag status={v} map={SCHEDULE_STATUS_MAP} />,
      },
    ],
    [onSelect],
  );

  return (
    <Table<ClassSchedule>
      rowKey="id"
      size="small"
      loading={loading}
      dataSource={schedules}
      columns={columns}
      pagination={false}
      scroll={{ x: 'max-content' }}
      onRow={(s) => ({ onClick: () => onSelect(s), style: { cursor: 'pointer' } })}
      locale={{ emptyText: <Empty description="Không có buổi lớp nào trong tuần này" /> }}
    />
  );
}

/** Danh sách theo ngày (điện thoại): mỗi ngày một nhóm thẻ. */
export function ScheduleAgenda({
  schedules,
  weekStart,
  loading,
  onSelect,
}: ListProps & { weekStart: Dayjs }) {
  const today = dayjs();
  const days = Array.from({ length: 7 }, (_, i) => weekStart.add(i, 'day'));
  const groups = days
    .map((d) => ({
      d,
      items: schedules.filter((s) => dayjs(s.start_at).isSame(d, 'day')),
    }))
    .filter((g) => g.items.length > 0);

  if (loading) return <Skeleton active paragraph={{ rows: 6 }} />;
  if (groups.length === 0)
    return <Empty description="Không có buổi lớp nào trong tuần này" style={{ padding: 24 }} />;

  return (
    <div>
      {groups.map(({ d, items }) => (
        <div key={d.valueOf()} className="agenda-day">
          <div className={`agenda-day-title${d.isSame(today, 'day') ? ' is-today' : ''}`}>
            <span>
              {weekdayName(d)}, {d.format('DD/MM')}
            </span>
            <Typography.Text type="secondary" style={{ fontSize: 12, fontWeight: 400 }}>
              {items.length} buổi
            </Typography.Text>
          </div>
          {items.map((s) => {
            const cancelled = s.status === 'CANCELLED';
            const full = !cancelled && s.booked_count >= s.capacity;
            return (
              <div
                key={s.id}
                role="button"
                tabIndex={0}
                className={`agenda-item${cancelled ? ' is-cancelled' : ''}${full ? ' is-full' : ''}`}
                style={{ '--agenda-color': s.class_type_color } as CSSProperties}
                onClick={() => onSelect(s)}
                onKeyDown={(e) => e.key === 'Enter' && onSelect(s)}
              >
                <div className="agenda-time">
                  {dayjs(s.start_at).format('HH:mm')}
                  <small>{dayjs(s.end_at).format('HH:mm')}</small>
                </div>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div className="agenda-name">{s.class_type_name}</div>
                  <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                    {scheduleTrainerText(s)} · {s.room_name}
                  </Typography.Text>
                  {cancelled && (
                    <div style={{ fontSize: 12, color: '#ef4444' }}>
                      Đã hủy{s.cancel_reason ? `: ${s.cancel_reason}` : ''}
                    </div>
                  )}
                </div>
                <div>
                  <SlotText s={s} />
                </div>
              </div>
            );
          })}
        </div>
      ))}
    </div>
  );
}
