import {
  Alert,
  Button,
  Descriptions,
  Divider,
  Drawer,
  Progress,
  Space,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import { CloseCircleOutlined, EditOutlined, RetweetOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import type { ClassSchedule } from '@/api/class';
import StatusTag from '@/components/StatusTag';
import { PERM } from '@/constants/permissions';
import { useIsMobile } from '@/hooks/useIsMobile';
import { usePermission } from '@/hooks/usePermission';
import BookingsPanel from './BookingsPanel';
import { SCHEDULE_STATUS_MAP, isUpcoming, scheduleTrainer, weekdayName } from './constants';
import { useSchedule } from './queries';

export interface ScheduleDrawerProps {
  /** Buổi lớp đang xem (bản từ lịch tuần – được làm mới bằng GET chi tiết). */
  schedule: ClassSchedule | null;
  onClose: () => void;
  onEdit: (s: ClassSchedule) => void;
  onCancel: (s: ClassSchedule) => void;
}

/** Drawer chi tiết buổi lớp: thông tin, sửa/hủy, danh sách đặt chỗ + điểm danh. */
export default function ScheduleDrawer({
  schedule,
  onClose,
  onEdit,
  onCancel,
}: ScheduleDrawerProps) {
  const isMobile = useIsMobile();
  const { can } = usePermission();
  const detail = useSchedule(schedule?.id);
  const s = detail.data ?? schedule;

  const canManage = can(PERM.CLASS_MANAGE);
  const editable = !!s && isUpcoming(s);
  const start = s ? dayjs(s.start_at) : null;
  const trainer = s ? scheduleTrainer(s) : null;
  const end = s ? dayjs(s.end_at) : null;
  const full = !!s && s.status !== 'CANCELLED' && s.booked_count >= s.capacity;

  return (
    <Drawer
      open={!!schedule}
      onClose={onClose}
      width={isMobile ? '100%' : 760}
      destroyOnHidden
      title={
        s && (
          <Space size={8} wrap>
            <span
              aria-hidden
              style={{
                display: 'inline-block',
                width: 12,
                height: 12,
                borderRadius: 3,
                background: s.class_type_color,
              }}
            />
            <span style={{ textDecoration: s.status === 'CANCELLED' ? 'line-through' : undefined }}>
              {s.class_type_name}
            </span>
            <StatusTag status={s.status} map={SCHEDULE_STATUS_MAP} />
            {full && (
              <Tag color="red" bordered={false}>
                Đã đủ chỗ
              </Tag>
            )}
          </Space>
        )
      }
      extra={
        s &&
        canManage && (
          <Space>
            <Tooltip title={editable ? undefined : 'Chỉ sửa được buổi chưa bắt đầu'}>
              <Button icon={<EditOutlined />} disabled={!editable} onClick={() => onEdit(s)}>
                {isMobile ? null : 'Sửa'}
              </Button>
            </Tooltip>
            <Tooltip title={editable ? undefined : 'Chỉ hủy được buổi chưa bắt đầu'}>
              <Button
                danger
                icon={<CloseCircleOutlined />}
                disabled={!editable}
                onClick={() => onCancel(s)}
              >
                {isMobile ? null : 'Hủy buổi'}
              </Button>
            </Tooltip>
          </Space>
        )
      }
    >
      {s && start && end && (
        <>
          {s.status === 'CANCELLED' && (
            <Alert
              type="error"
              showIcon
              style={{ marginBottom: 16 }}
              message="Buổi lớp đã bị hủy"
              description={s.cancel_reason ? `Lý do: ${s.cancel_reason}` : undefined}
            />
          )}

          <Descriptions
            size="small"
            column={{ xs: 1, sm: 2, md: 2, lg: 2, xl: 2, xxl: 2 }}
            styles={{ label: { width: 96, color: '#6b7280' } }}
            items={[
              {
                key: 'time',
                label: 'Thời gian',
                span: 'filled',
                children: (
                  <span>
                    <b>
                      {start.format('HH:mm')} – {end.format('HH:mm')}
                    </b>{' '}
                    · {weekdayName(start)}, {start.format('DD/MM/YYYY')} ({s.duration_min} phút)
                  </span>
                ),
              },
              { key: 'branch', label: 'Chi nhánh', children: s.branch_name },
              { key: 'room', label: 'Phòng', children: s.room_name },
              {
                key: 'trainer',
                label: 'HLV',
                children: trainer?.substituted ? (
                  <span>
                    {trainer.name}{' '}
                    <Tag color="orange" bordered={false}>
                      Dạy thay
                    </Tag>
                    <Typography.Text type="secondary">
                      (HLV gốc: {trainer.original})
                    </Typography.Text>
                  </span>
                ) : (
                  s.trainer_name
                ),
              },
              {
                key: 'series',
                label: 'Lịch lặp',
                children: s.series_id ? (
                  <Tag icon={<RetweetOutlined />} color="purple" bordered={false}>
                    Thuộc chuỗi lặp hàng tuần
                  </Tag>
                ) : (
                  'Buổi lẻ'
                ),
              },
              {
                key: 'slots',
                label: 'Đã đặt',
                span: 'filled',
                children:
                  s.status === 'CANCELLED' ? (
                    '—'
                  ) : (
                    <div style={{ maxWidth: 360 }}>
                      <Space size={12} wrap>
                        <span>
                          <b>{s.booked_count}</b>/{s.capacity} chỗ
                        </span>
                        <span>Còn trống: {s.available_slots}</span>
                        <span>Chờ: {s.waitlist_count}</span>
                        <span>Có mặt: {s.attended_count}</span>
                      </Space>
                      <Progress
                        percent={Math.round((s.booked_count / Math.max(s.capacity, 1)) * 100)}
                        status={full ? 'exception' : 'normal'}
                        size="small"
                        showInfo={false}
                      />
                    </div>
                  ),
              },
              ...(s.note
                ? [{ key: 'note', label: 'Ghi chú', span: 'filled' as const, children: s.note }]
                : []),
            ]}
          />

          <Divider orientation="left" style={{ marginTop: 24 }}>
            <Typography.Text strong>Danh sách đặt chỗ</Typography.Text>
          </Divider>
          <BookingsPanel schedule={s} />
        </>
      )}
    </Drawer>
  );
}
