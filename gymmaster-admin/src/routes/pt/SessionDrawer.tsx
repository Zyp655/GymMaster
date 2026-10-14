import { useState } from 'react';
import {
  Alert,
  Button,
  Descriptions,
  Drawer,
  Result,
  Skeleton,
  Space,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import {
  CheckCircleOutlined,
  ClockCircleOutlined,
  CloseCircleOutlined,
  FileTextOutlined,
  UserDeleteOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import { Link } from 'react-router-dom';
import type { PtSession } from '@/api/pt';
import StatusTag from '@/components/StatusTag';
import { PERM } from '@/constants/permissions';
import { useIsMobile } from '@/hooks/useIsMobile';
import { usePermission } from '@/hooks/usePermission';
import { weekdayName } from '@/routes/classes/constants';
import { getErrorMessage } from '@/utils/errors';
import { formatDateTime, formatPhone } from '@/utils/format';
import {
  PT_SESSION_SOURCE_LABEL,
  PT_SESSION_STATUS_MAP,
  canCancelSession,
  canConfirmSession,
  canReschedule,
  sessionStatusKey,
} from './constants';
import { usePtSession } from './queries';
import {
  CancelSessionModal,
  CompleteSessionModal,
  NoShowModal,
  RescheduleModal,
} from './SessionModals';

export interface SessionDrawerProps {
  sessionId: number | null;
  onClose: () => void;
  /** Mở chi tiết hợp đồng (ẩn link nếu không truyền). */
  onOpenContract?: (contractId: number) => void;
}

type ModalKey = 'reschedule' | 'cancel' | 'complete' | 'noshow' | null;

/** Nhãn "đã tính buổi" cho buổi PT. */
export function CountedTag({ s }: { s: Pick<PtSession, 'counted' | 'status'> }) {
  if (s.counted)
    return (
      <Tag color="purple" bordered={false}>
        Đã trừ buổi
      </Tag>
    );
  if (s.status === 'SCHEDULED') return null;
  return <Tag bordered={false}>Không trừ buổi</Tag>;
}

/** Drawer chi tiết buổi PT: thông tin, đổi giờ, hủy, hoàn thành (giáo án), vắng mặt. */
export default function SessionDrawer({ sessionId, onClose, onOpenContract }: SessionDrawerProps) {
  const isMobile = useIsMobile();
  const { can } = usePermission();
  const query = usePtSession(sessionId);
  const s = query.data;
  const [modal, setModal] = useState<ModalKey>(null);

  const canSchedule = can(PERM.PT_SCHEDULE);
  const canComplete = can(PERM.PT_COMPLETE);
  const start = s ? dayjs(s.start_at) : null;
  const statusKey = s ? sessionStatusKey(s) : '';

  const actions = s && (
    <Space wrap>
      {canSchedule && canReschedule(s) && (
        <Button icon={<ClockCircleOutlined />} onClick={() => setModal('reschedule')}>
          Đổi giờ
        </Button>
      )}
      {canSchedule && canCancelSession(s) && (
        <Button danger icon={<CloseCircleOutlined />} onClick={() => setModal('cancel')}>
          Hủy buổi
        </Button>
      )}
      {canComplete && (s.status === 'SCHEDULED' || statusKey === 'NO_SHOW_PENDING') && (
        <>
          <Tooltip title={canConfirmSession(s) ? undefined : 'Chỉ xác nhận sau giờ bắt đầu'}>
            <Button
              danger
              icon={<UserDeleteOutlined />}
              disabled={!canConfirmSession(s)}
              onClick={() => setModal('noshow')}
            >
              Vắng mặt
            </Button>
          </Tooltip>
          <Tooltip title={canConfirmSession(s) ? undefined : 'Chỉ xác nhận sau giờ bắt đầu'}>
            <Button
              color="orange"
              variant="solid"
              icon={<CheckCircleOutlined />}
              disabled={!canConfirmSession(s)}
              onClick={() => setModal('complete')}
            >
              Hoàn thành
            </Button>
          </Tooltip>
        </>
      )}
    </Space>
  );

  return (
    <Drawer
      open={!!sessionId}
      onClose={onClose}
      width={isMobile ? '100%' : 640}
      destroyOnHidden
      title={
        s ? (
          <Space size={8} wrap>
            <span>Buổi PT – {s.member_name}</span>
            <StatusTag status={statusKey} map={PT_SESSION_STATUS_MAP} />
            <CountedTag s={s} />
          </Space>
        ) : (
          'Buổi PT'
        )
      }
      footer={actions ? <div style={{ textAlign: 'right' }}>{actions}</div> : undefined}
    >
      {query.isLoading ? (
        <Skeleton active paragraph={{ rows: 6 }} />
      ) : query.isError ? (
        <Result
          status="warning"
          title="Không tải được buổi PT"
          subTitle={getErrorMessage(query.error)}
          extra={<Button onClick={() => query.refetch()}>Thử lại</Button>}
        />
      ) : s && start ? (
        <>
          {statusKey === 'NO_SHOW_PENDING' && (
            <Alert
              type="warning"
              showIcon
              style={{ marginBottom: 16 }}
              message="Buổi quá giờ chưa được xác nhận"
              description="Hệ thống đã tự đánh dấu (chưa trừ buổi). HLV đã dạy → bấm Hoàn thành; hội viên thật sự vắng → bấm Vắng mặt để chốt tính buổi; châm chước → để nguyên."
            />
          )}
          {(s.status === 'CANCELLED' || (s.status === 'NO_SHOW' && s.cancel_reason)) && (
            <Alert
              type={s.status === 'CANCELLED' ? 'error' : 'warning'}
              showIcon
              style={{ marginBottom: 16 }}
              message={s.status === 'CANCELLED' ? 'Buổi đã hủy' : 'Vắng mặt'}
              description={s.cancel_reason || undefined}
            />
          )}
          <Descriptions
            size="small"
            column={1}
            styles={{ label: { width: 130, color: '#6b7280' } }}
            items={[
              {
                key: 'time',
                label: 'Thời gian',
                children: (
                  <span>
                    <b>
                      {start.format('HH:mm')} – {dayjs(s.end_at).format('HH:mm')}
                    </b>{' '}
                    · {weekdayName(start)}, {start.format('DD/MM/YYYY')}
                  </span>
                ),
              },
              {
                key: 'member',
                label: 'Hội viên',
                children: (
                  <span>
                    <Link to={`/app/members/${s.member_id}?tab=classes&part=pt`}>
                      {s.member_name}
                    </Link>{' '}
                    <Typography.Text type="secondary">
                      {s.member_code} · {formatPhone(s.member_phone)}
                    </Typography.Text>
                  </span>
                ),
              },
              { key: 'trainer', label: 'HLV', children: s.trainer_name },
              { key: 'branch', label: 'Chi nhánh', children: s.branch_name },
              {
                key: 'contract',
                label: 'Hợp đồng',
                children: onOpenContract ? (
                  <Button
                    type="link"
                    size="small"
                    style={{ padding: 0 }}
                    icon={<FileTextOutlined />}
                    onClick={() => onOpenContract(s.contract_id)}
                  >
                    #{s.contract_id} · {s.package_name}
                  </Button>
                ) : (
                  `#${s.contract_id} · ${s.package_name}`
                ),
              },
              {
                key: 'source',
                label: 'Đặt bởi',
                children: `${s.created_by_name || '—'} (${PT_SESSION_SOURCE_LABEL[s.source] ?? s.source}) · ${formatDateTime(s.created_at)}`,
              },
              ...(s.completed_at
                ? [
                    {
                      key: 'done',
                      label: 'Chốt buổi lúc',
                      children: formatDateTime(s.completed_at),
                    },
                  ]
                : []),
              ...(s.cancelled_at
                ? [{ key: 'cancel', label: 'Hủy lúc', children: formatDateTime(s.cancelled_at) }]
                : []),
              ...(s.workout_note
                ? [
                    {
                      key: 'workout',
                      label: 'Giáo án',
                      children: (
                        <Typography.Paragraph style={{ whiteSpace: 'pre-wrap', margin: 0 }}>
                          {s.workout_note}
                        </Typography.Paragraph>
                      ),
                    },
                  ]
                : []),
              ...(s.trainer_note
                ? [
                    {
                      key: 'trainer_note',
                      label: 'Nhận xét HLV',
                      children: (
                        <Typography.Paragraph style={{ whiteSpace: 'pre-wrap', margin: 0 }}>
                          {s.trainer_note}
                        </Typography.Paragraph>
                      ),
                    },
                  ]
                : []),
            ]}
          />
        </>
      ) : null}

      <RescheduleModal
        session={modal === 'reschedule' ? (s ?? null) : null}
        onClose={() => setModal(null)}
      />
      <CancelSessionModal
        session={modal === 'cancel' ? (s ?? null) : null}
        onClose={() => setModal(null)}
      />
      <CompleteSessionModal
        session={modal === 'complete' ? (s ?? null) : null}
        onClose={() => setModal(null)}
      />
      <NoShowModal
        session={modal === 'noshow' ? (s ?? null) : null}
        onClose={() => setModal(null)}
      />
    </Drawer>
  );
}
