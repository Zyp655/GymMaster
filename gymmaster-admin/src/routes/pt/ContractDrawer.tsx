import { useState } from 'react';
import {
  Alert,
  Button,
  Descriptions,
  Divider,
  Drawer,
  Empty,
  Progress,
  Result,
  Skeleton,
  Space,
  Table,
  Tooltip,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  CloseCircleOutlined,
  DollarOutlined,
  PlusOutlined,
  PrinterOutlined,
  SwapOutlined,
} from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import dayjs from 'dayjs';
import { Link } from 'react-router-dom';
import type { PtSession } from '@/api/pt';
import MoneyText from '@/components/Money/MoneyText';
import StatusTag from '@/components/StatusTag';
import { PERM } from '@/constants/permissions';
import { INVOICE_STATUS_MAP } from '@/constants/status';
import { useIsMobile } from '@/hooks/useIsMobile';
import { usePermission } from '@/hooks/usePermission';
import { weekdayName } from '@/routes/classes/constants';
import { CollectModal } from '@/routes/invoices/InvoiceActionModals';
import { canCollect } from '@/routes/invoices/utils';
import { openInvoicePrint } from '@/routes/subscriptions/utils';
import { getErrorMessage } from '@/utils/errors';
import { formatDate, formatDateTime, formatPhone } from '@/utils/format';
import {
  PT_CONTRACT_STATUS_MAP,
  PT_SESSION_STATUS_MAP,
  isBookable,
  isLiveContract,
  sessionStatusKey,
} from './constants';
import { CancelContractModal, ChangeTrainerModal } from './ContractModals';
import { invalidatePt, usePtContract } from './queries';
import SessionDrawer, { CountedTag } from './SessionDrawer';
import { BookSessionModal } from './SessionModals';

export interface ContractDrawerProps {
  id: number | null;
  onClose: () => void;
}

type ModalKey = 'book' | 'trainer' | 'cancel' | 'collect' | null;

const DESC_COLUMNS = { xs: 1, sm: 2, md: 2, lg: 2, xl: 2, xxl: 2 };

/** Chi tiết hợp đồng PT: thông tin, hóa đơn, danh sách buổi; đặt buổi, đổi HLV, hủy hợp đồng. */
export default function ContractDrawer({ id, onClose }: ContractDrawerProps) {
  const isMobile = useIsMobile();
  const { can } = usePermission();
  const queryClient = useQueryClient();
  const query = usePtContract(id);
  const c = query.data;
  const [modal, setModal] = useState<ModalKey>(null);
  const [sessionId, setSessionId] = useState<number | null>(null);
  const close = () => setModal(null);

  const live = !!c && isLiveContract(c);
  const inv = c?.invoice ?? null;

  const columns: ColumnsType<PtSession> = [
    {
      title: '#',
      key: 'idx',
      width: 44,
      render: (_, __, i) => i + 1,
    },
    {
      title: 'Thời gian',
      key: 'time',
      className: 'text-nowrap',
      render: (_, s) => {
        const st = dayjs(s.start_at);
        return (
          <Button
            type="link"
            size="small"
            style={{ padding: 0 }}
            onClick={() => setSessionId(s.id)}
          >
            {st.format('HH:mm')}–{dayjs(s.end_at).format('HH:mm')} {weekdayName(st)},{' '}
            {st.format('DD/MM/YYYY')}
          </Button>
        );
      },
    },
    { title: 'HLV', dataIndex: 'trainer_name', responsive: ['md'] },
    {
      title: 'Trạng thái',
      key: 'status',
      render: (_, s) => (
        <Space size={4} wrap>
          <StatusTag status={sessionStatusKey(s)} map={PT_SESSION_STATUS_MAP} />
          <CountedTag s={s} />
        </Space>
      ),
    },
    {
      title: 'Giáo án / ghi chú',
      key: 'note',
      ellipsis: true,
      responsive: ['lg'],
      render: (_, s) => s.workout_note || s.cancel_reason || '—',
    },
  ];

  return (
    <Drawer
      open={!!id}
      onClose={onClose}
      width={isMobile ? '100%' : 820}
      destroyOnHidden
      title={
        c ? (
          <Space size={8} wrap>
            <span>Hợp đồng PT #{c.id}</span>
            <StatusTag status={c.status} map={PT_CONTRACT_STATUS_MAP} />
          </Space>
        ) : (
          'Hợp đồng PT'
        )
      }
      extra={
        c && (
          <Space wrap>
            {can(PERM.PT_SCHEDULE) && (
              <Tooltip
                title={
                  isBookable(c)
                    ? undefined
                    : c.status !== 'ACTIVE'
                      ? 'Chỉ hợp đồng đang tập mới đặt buổi được'
                      : 'Đã đặt/dùng hết số buổi'
                }
              >
                <Button
                  type="primary"
                  icon={<PlusOutlined />}
                  disabled={!isBookable(c)}
                  onClick={() => setModal('book')}
                >
                  Đặt buổi
                </Button>
              </Tooltip>
            )}
            {can(PERM.PT_SELL) && live && (
              <Button icon={<SwapOutlined />} onClick={() => setModal('trainer')}>
                {isMobile ? null : 'Đổi HLV'}
              </Button>
            )}
            {can(PERM.PT_CANCEL_CONTRACT) && live && (
              <Button danger icon={<CloseCircleOutlined />} onClick={() => setModal('cancel')}>
                {isMobile ? null : 'Hủy hợp đồng'}
              </Button>
            )}
          </Space>
        )
      }
    >
      {query.isLoading ? (
        <Skeleton active paragraph={{ rows: 8 }} />
      ) : query.isError ? (
        <Result
          status="warning"
          title="Không tải được hợp đồng PT"
          subTitle={getErrorMessage(query.error)}
          extra={<Button onClick={() => query.refetch()}>Thử lại</Button>}
        />
      ) : c ? (
        <>
          {c.status === 'PENDING' && (
            <Alert
              type="warning"
              showIcon
              style={{ marginBottom: 16 }}
              message="Hợp đồng chờ thanh toán"
              description="Thu đủ tiền hóa đơn để kích hoạt hợp đồng và bắt đầu đặt buổi."
            />
          )}
          <Descriptions
            size="small"
            bordered
            column={DESC_COLUMNS}
            items={[
              {
                key: 'member',
                label: 'Hội viên',
                span: 'filled',
                children: (
                  <span>
                    <Link to={`/app/members/${c.member_id}?tab=classes&part=pt`}>
                      {c.member_name}
                    </Link>{' '}
                    <Typography.Text type="secondary">
                      {c.member_code} · {formatPhone(c.member_phone)}
                    </Typography.Text>
                  </span>
                ),
              },
              {
                key: 'package',
                label: 'Gói PT',
                children: (
                  <>
                    <span className="code-text">{c.package_code}</span> {c.package_name}
                  </>
                ),
              },
              { key: 'trainer', label: 'HLV', children: c.trainer_name },
              { key: 'branch', label: 'Chi nhánh', children: c.branch_name },
              { key: 'minutes', label: 'Thời lượng', children: `${c.session_minutes} phút/buổi` },
              {
                key: 'period',
                label: 'Thời hạn',
                children: `${formatDate(c.start_date)} – ${formatDate(c.end_date)}`,
              },
              {
                key: 'left',
                label: 'Còn lại',
                children: c.days_left !== null ? `${c.days_left} ngày` : '—',
              },
              {
                key: 'progress',
                label: 'Tiến độ',
                span: 'filled',
                children: (
                  <div style={{ maxWidth: 420 }}>
                    <Space size={12} wrap>
                      <span>
                        Đã dùng <b>{c.used_sessions}</b>/{c.total_sessions}
                      </span>
                      <span>Đang đặt {c.scheduled_sessions}</span>
                      <span>Còn đặt được {c.bookable_sessions}</span>
                    </Space>
                    <Progress
                      size="small"
                      percent={Math.round(
                        ((c.used_sessions + c.scheduled_sessions) / Math.max(c.total_sessions, 1)) *
                          100,
                      )}
                      success={{
                        percent: Math.round(
                          (c.used_sessions / Math.max(c.total_sessions, 1)) * 100,
                        ),
                      }}
                      showInfo={false}
                    />
                  </div>
                ),
              },
              { key: 'price', label: 'Giá trị', children: <MoneyText value={c.price} strong /> },
              {
                key: 'pps',
                label: 'Giá/buổi',
                children: <MoneyText value={c.price_per_session} />,
              },
              {
                key: 'sold',
                label: 'Người bán',
                children: `${c.sold_by_name || '—'} · ${formatDateTime(c.created_at)}`,
                span: 'filled',
              },
              ...(c.note
                ? [{ key: 'note', label: 'Ghi chú', span: 'filled' as const, children: c.note }]
                : []),
            ]}
          />

          {inv && (
            <>
              <Divider orientation="left">
                <Typography.Text strong>Hóa đơn</Typography.Text>
              </Divider>
              <Space wrap style={{ width: '100%', justifyContent: 'space-between' }}>
                <Space wrap>
                  <Link to={`/app/invoices/${inv.id}`} className="code-text">
                    {inv.invoice_no}
                  </Link>
                  <StatusTag status={inv.status} map={INVOICE_STATUS_MAP} />
                  <span>
                    Tổng <MoneyText value={inv.total} strong />
                  </span>
                  <span>
                    Đã thu <MoneyText value={inv.paid_amount} />
                  </span>
                  {inv.refunded_amount > 0 && (
                    <span>
                      Đã hoàn <MoneyText value={inv.refunded_amount} />
                    </span>
                  )}
                  {inv.due > 0 && (
                    <Typography.Text type="danger">
                      Còn nợ <MoneyText value={inv.due} />
                    </Typography.Text>
                  )}
                </Space>
                <Space>
                  {can(PERM.INVOICE_COLLECT) && canCollect(inv) && (
                    <Button
                      type="primary"
                      icon={<DollarOutlined />}
                      onClick={() => setModal('collect')}
                    >
                      Thu tiền
                    </Button>
                  )}
                  <Button icon={<PrinterOutlined />} onClick={() => openInvoicePrint(inv.id)}>
                    In
                  </Button>
                </Space>
              </Space>
            </>
          )}

          <Divider orientation="left">
            <Typography.Text strong>Buổi tập ({c.sessions.length})</Typography.Text>
          </Divider>
          <Table<PtSession>
            rowKey="id"
            size="small"
            dataSource={c.sessions}
            columns={columns}
            pagination={c.sessions.length > 20 ? { pageSize: 20, showSizeChanger: false } : false}
            scroll={{ x: 'max-content' }}
            locale={{
              emptyText: (
                <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="Chưa đặt buổi nào" />
              ),
            }}
          />
        </>
      ) : null}

      <BookSessionModal open={modal === 'book'} contract={c} onClose={close} />
      <ChangeTrainerModal contract={modal === 'trainer' ? (c ?? null) : null} onClose={close} />
      <CancelContractModal contract={modal === 'cancel' ? (c ?? null) : null} onClose={close} />
      {inv && (
        <CollectModal
          open={modal === 'collect'}
          invoice={inv}
          onClose={() => {
            close();
            void invalidatePt(queryClient);
          }}
        />
      )}
      <SessionDrawer sessionId={sessionId} onClose={() => setSessionId(null)} />
    </Drawer>
  );
}
