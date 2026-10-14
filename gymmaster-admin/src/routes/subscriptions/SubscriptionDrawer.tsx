import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  Alert,
  Button,
  Descriptions,
  Drawer,
  Empty,
  Result,
  Skeleton,
  Space,
  Table,
  Timeline,
  Typography,
} from 'antd';
import {
  ArrowUpOutlined,
  CloseCircleOutlined,
  DollarOutlined,
  PauseCircleOutlined,
  PrinterOutlined,
  ReloadOutlined,
  SwapOutlined,
} from '@ant-design/icons';
import { Link } from 'react-router-dom';
import subscriptionApi, {
  ACCESS_SCOPE_LABEL,
  PLAN_TYPE_LABEL,
  SUBSCRIPTION_EVENT_LABEL,
} from '@/api/subscription';
import type { Freeze, SubscriptionDetail, SubscriptionEvent } from '@/api/subscription';
import MoneyText from '@/components/Money/MoneyText';
import StatusTag from '@/components/StatusTag';
import { PERM } from '@/constants/permissions';
import { FREEZE_STATUS_MAP, INVOICE_STATUS_MAP, SUBSCRIPTION_STATUS_MAP } from '@/constants/status';
import { useIsMobile } from '@/hooks/useIsMobile';
import { usePermission } from '@/hooks/usePermission';
import { CollectModal } from '@/routes/invoices/InvoiceActionModals';
import { canCollect } from '@/routes/invoices/utils';
import { getErrorMessage } from '@/utils/errors';
import { formatDate, formatDateTime, formatMoney } from '@/utils/format';
import CancelModal from './CancelModal';
import FreezeActions from './FreezeActions';
import FreezeModal from './FreezeModal';
import RenewModal from './RenewModal';
import TransferModal from './TransferModal';
import UpgradeModal from './UpgradeModal';
import { DESC_COLUMNS, openInvoicePrint, remainingText } from './utils';

type ModalKey = 'renew' | 'upgrade' | 'freeze' | 'transfer' | 'cancel' | 'collect' | null;

const EVENT_COLOR: Partial<Record<SubscriptionEvent['event_type'], string>> = {
  CREATED: 'blue',
  ACTIVATED: 'green',
  RENEWED: 'green',
  UPGRADED: 'purple',
  FROZEN: 'blue',
  UNFROZEN: 'blue',
  FREEZE_REJECTED: 'red',
  TRANSFERRED: 'purple',
  CANCELLED: 'red',
  EXPIRED: 'gray',
};

/** Mô tả ngắn từ `data` (JSON tự do) của sự kiện. */
function eventDetail(e: SubscriptionEvent): string {
  const d = e.data ?? {};
  const s = (k: string) => (typeof d[k] === 'string' ? (d[k] as string) : undefined);
  const n = (k: string) => (typeof d[k] === 'number' ? (d[k] as number) : undefined);
  const parts: string[] = [];
  switch (e.event_type) {
    case 'CREATED':
      if (s('start_date'))
        parts.push(`${formatDate(s('start_date'))} – ${formatDate(s('end_date'))}`);
      if (d.chained) parts.push('nối tiếp gói trước');
      break;
    case 'ACTIVATED':
      if (d.restored) parts.push('Khôi phục do hóa đơn gói mới bị hủy');
      else if (s('invoice_no')) parts.push(`Hóa đơn ${s('invoice_no')} đã thanh toán đủ`);
      break;
    case 'RENEWED':
      if (n('new_subscription_id')) parts.push(`Gói mới #${n('new_subscription_id')}`);
      break;
    case 'UPGRADED':
      if (n('to_subscription_id')) parts.push(`Sang gói #${n('to_subscription_id')}`);
      if (n('remaining_value') !== undefined)
        parts.push(`khấu trừ ${formatMoney(n('credit') ?? n('remaining_value'))}`);
      break;
    case 'FREEZE_REQUESTED':
    case 'FROZEN':
      if (s('start_date'))
        parts.push(`${formatDate(s('start_date'))} – ${formatDate(s('end_date'))}`);
      if (n('days')) parts.push(`${n('days')} ngày`);
      if (s('new_end_date')) parts.push(`hết hạn mới ${formatDate(s('new_end_date'))}`);
      break;
    case 'UNFROZEN':
      if (d.early) parts.push('Kết thúc sớm');
      if (n('returned_days')) parts.push(`trả lại ${n('returned_days')} ngày`);
      if (s('end_date')) parts.push(`hết hạn ${formatDate(s('end_date'))}`);
      break;
    case 'TRANSFERRED':
      if (n('to_subscription_id')) parts.push(`Gói mới #${n('to_subscription_id')}`);
      if (n('transfer_fee') !== undefined) parts.push(`phí ${formatMoney(n('transfer_fee'))}`);
      break;
    case 'CANCELLED':
      if (s('reason')) parts.push(`Lý do: ${s('reason')}`);
      if (n('refund_amount')) parts.push(`hoàn ${formatMoney(n('refund_amount'))}`);
      break;
    case 'EXPIRED':
      if (s('end_date')) parts.push(`Hết hạn ${formatDate(s('end_date'))}`);
      break;
    default:
      if (s('reason')) parts.push(s('reason')!);
  }
  if (!parts.length && s('reason')) parts.push(s('reason')!);
  return parts.join(' · ');
}

function ActionButtons({
  sub,
  onOpen,
}: {
  sub: SubscriptionDetail;
  onOpen: (k: ModalKey) => void;
}) {
  const { can } = usePermission();
  const st = sub.status;
  const live = st === 'ACTIVE' || st === 'FROZEN';
  const buttons = (
    <>
      {can(PERM.SUBSCRIPTION_SELL) && (live || st === 'EXPIRED') && (
        <Button
          color="orange"
          variant="solid"
          icon={<ReloadOutlined />}
          onClick={() => onOpen('renew')}
        >
          Gia hạn
        </Button>
      )}
      {can(PERM.SUBSCRIPTION_SELL) && st === 'ACTIVE' && (
        <Button icon={<ArrowUpOutlined />} onClick={() => onOpen('upgrade')}>
          Nâng cấp
        </Button>
      )}
      {can(PERM.SUBSCRIPTION_FREEZE) && st === 'ACTIVE' && sub.plan.max_freeze_times > 0 && (
        <Button icon={<PauseCircleOutlined />} onClick={() => onOpen('freeze')}>
          Bảo lưu
        </Button>
      )}
      {can(PERM.SUBSCRIPTION_TRANSFER) && st === 'ACTIVE' && (
        <Button icon={<SwapOutlined />} onClick={() => onOpen('transfer')}>
          Chuyển nhượng
        </Button>
      )}
      {can(PERM.INVOICE_COLLECT) && sub.invoice && canCollect(sub.invoice) && (
        <Button type="primary" icon={<DollarOutlined />} onClick={() => onOpen('collect')}>
          Thu tiền
        </Button>
      )}
      {can(PERM.SUBSCRIPTION_CANCEL) && (live || st === 'PENDING') && (
        <Button danger icon={<CloseCircleOutlined />} onClick={() => onOpen('cancel')}>
          Hủy gói
        </Button>
      )}
    </>
  );
  // Không có nút nào (gói đã kết thúc / thiếu quyền) → không chiếm chỗ.
  const hasAny =
    (can(PERM.SUBSCRIPTION_SELL) && (live || st === 'EXPIRED')) ||
    (can(PERM.INVOICE_COLLECT) && !!sub.invoice && canCollect(sub.invoice)) ||
    (can(PERM.SUBSCRIPTION_CANCEL) && (live || st === 'PENDING')) ||
    (st === 'ACTIVE' && can([PERM.SUBSCRIPTION_FREEZE, PERM.SUBSCRIPTION_TRANSFER]));
  return hasAny ? (
    <Space wrap size={8}>
      {buttons}
    </Space>
  ) : null;
}

function DrawerBody({ sub, onOpen }: { sub: SubscriptionDetail; onOpen: (k: ModalKey) => void }) {
  const inv = sub.invoice;
  return (
    <Space direction="vertical" size={20} style={{ width: '100%' }}>
      <ActionButtons sub={sub} onOpen={onOpen} />

      <Descriptions
        size="small"
        bordered
        column={DESC_COLUMNS}
        items={[
          {
            key: 'member',
            label: 'Hội viên',
            span: 'filled' as const,
            children: (
              <Link to={`/app/members/${sub.member_id}`}>
                {sub.member_name} · {sub.member_code}
              </Link>
            ),
          },
          {
            key: 'plan',
            label: 'Gói',
            span: 'filled' as const,
            children: (
              <>
                <span className="code-text">{sub.plan_code}</span> {sub.plan_name} ·{' '}
                {PLAN_TYPE_LABEL[sub.plan_type]} · {ACCESS_SCOPE_LABEL[sub.access_scope]}
              </>
            ),
          },
          { key: 'branch', label: 'Chi nhánh bán', children: sub.branch_name },
          {
            key: 'status',
            label: 'Trạng thái',
            children: <StatusTag status={sub.status} map={SUBSCRIPTION_STATUS_MAP} />,
          },
          {
            key: 'time',
            label: 'Thời gian',
            children: `${formatDate(sub.start_date)} – ${formatDate(sub.end_date)}`,
          },
          { key: 'left', label: 'Còn lại', children: remainingText(sub) },
          { key: 'price', label: 'Giá trị gói', children: <MoneyText value={sub.price} /> },
          {
            key: 'freeze',
            label: 'Bảo lưu đã dùng',
            children:
              sub.plan.max_freeze_times > 0
                ? `${sub.freeze_days_used}/${sub.plan.max_freeze_days} ngày · ${sub.freeze_times_used}/${sub.plan.max_freeze_times} lần`
                : 'Không cho phép',
          },
          { key: 'sold', label: 'Người bán', children: sub.sold_by_name || '—' },
          { key: 'created', label: 'Ngày tạo', children: formatDateTime(sub.created_at) },
          ...(sub.previous_id
            ? [
                {
                  key: 'prev',
                  label: 'Gói trước',
                  span: 'filled' as const,
                  children: `#${sub.previous_id}`,
                },
              ]
            : []),
          ...(sub.note
            ? [{ key: 'note', label: 'Ghi chú', span: 'filled' as const, children: sub.note }]
            : []),
        ]}
      />

      <div>
        <Typography.Title level={5}>Hóa đơn liên kết</Typography.Title>
        {inv ? (
          <Descriptions
            size="small"
            bordered
            column={DESC_COLUMNS}
            items={[
              {
                key: 'no',
                label: 'Số hóa đơn',
                children: (
                  <Space>
                    <Link to={`/app/invoices/${inv.id}`} className="code-text">
                      {inv.invoice_no}
                    </Link>
                    <Button
                      size="small"
                      type="text"
                      icon={<PrinterOutlined />}
                      onClick={() => openInvoicePrint(inv.id)}
                      aria-label="In hóa đơn"
                    />
                  </Space>
                ),
              },
              {
                key: 'st',
                label: 'Trạng thái',
                children: <StatusTag status={inv.status} map={INVOICE_STATUS_MAP} />,
              },
              { key: 'total', label: 'Tổng', children: <MoneyText value={inv.total} strong /> },
              { key: 'paid', label: 'Đã thu', children: <MoneyText value={inv.paid_amount} /> },
              {
                key: 'due',
                label: 'Còn nợ',
                children: (
                  <MoneyText
                    value={inv.due}
                    style={inv.due > 0 ? { color: '#cf1322' } : undefined}
                  />
                ),
              },
              {
                key: 'ref',
                label: 'Đã hoàn',
                children: <MoneyText value={inv.refunded_amount} />,
              },
            ]}
          />
        ) : (
          <Typography.Text type="secondary">
            Không có hóa đơn (chuyển nhượng miễn phí)
          </Typography.Text>
        )}
      </div>

      <div>
        <Typography.Title level={5}>Bảo lưu</Typography.Title>
        <Table<Freeze>
          size="small"
          rowKey="id"
          pagination={false}
          dataSource={sub.freezes}
          scroll={{ x: 'max-content' }}
          locale={{ emptyText: 'Chưa có yêu cầu bảo lưu' }}
          columns={[
            {
              title: 'Khoảng',
              key: 'range',
              render: (_, f) => `${formatDate(f.start_date)} – ${formatDate(f.end_date)}`,
            },
            { title: 'Ngày', dataIndex: 'days', align: 'right' },
            { title: 'Lý do', dataIndex: 'reason', render: (v: string) => v || '—' },
            {
              title: 'Trạng thái',
              dataIndex: 'status',
              render: (v: string) => <StatusTag status={v} map={FREEZE_STATUS_MAP} />,
            },
            { title: '', key: 'act', render: (_, f) => <FreezeActions freeze={f} /> },
          ]}
        />
      </div>

      <div>
        <Typography.Title level={5}>Lịch sử</Typography.Title>
        {sub.events.length === 0 ? (
          <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="Chưa có sự kiện" />
        ) : (
          <Timeline
            items={sub.events.map((e) => ({
              color: EVENT_COLOR[e.event_type] ?? 'gray',
              children: (
                <>
                  <Typography.Text strong>
                    {SUBSCRIPTION_EVENT_LABEL[e.event_type] ?? e.event_type}
                  </Typography.Text>{' '}
                  <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                    {formatDateTime(e.created_at)} · {e.actor_id ? e.actor_name : 'Hệ thống'}
                  </Typography.Text>
                  {eventDetail(e) && <div style={{ fontSize: 13 }}>{eventDetail(e)}</div>}
                </>
              ),
            }))}
          />
        )}
      </div>
    </Space>
  );
}

export interface SubscriptionDrawerProps {
  id: number | null;
  onClose: () => void;
}

/** Drawer chi tiết gói: thông tin, hóa đơn, bảo lưu, timeline + các thao tác vòng đời. */
export default function SubscriptionDrawer({ id, onClose }: SubscriptionDrawerProps) {
  const isMobile = useIsMobile();
  const [modal, setModal] = useState<ModalKey>(null);
  const query = useQuery({
    queryKey: [subscriptionApi.key, 'detail', id],
    queryFn: () => subscriptionApi.get(id!),
    enabled: !!id,
  });
  const sub = query.data;
  const close = () => setModal(null);

  return (
    <Drawer
      open={!!id}
      onClose={onClose}
      width={isMobile ? '100%' : 760}
      title={
        sub ? (
          <Space wrap>
            <span>
              Gói #{sub.id} · {sub.plan_name}
            </span>
            <StatusTag status={sub.status} map={SUBSCRIPTION_STATUS_MAP} />
          </Space>
        ) : (
          'Chi tiết gói tập'
        )
      }
      destroyOnHidden
    >
      {query.isLoading ? (
        <Skeleton active paragraph={{ rows: 10 }} />
      ) : query.isError || !sub ? (
        <Result
          status="warning"
          title={getErrorMessage(query.error, 'Không tải được gói tập')}
          extra={<Button onClick={() => query.refetch()}>Thử lại</Button>}
        />
      ) : (
        <>
          {sub.status === 'PENDING' && (
            <Alert
              type="warning"
              showIcon
              style={{ marginBottom: 16 }}
              message="Gói đang chờ thanh toán – hội viên chưa thể check-in cho đến khi thu đủ tiền."
            />
          )}
          <DrawerBody sub={sub} onOpen={setModal} />
          <RenewModal open={modal === 'renew'} sub={sub} onClose={close} />
          <UpgradeModal open={modal === 'upgrade'} sub={sub} onClose={close} />
          <FreezeModal open={modal === 'freeze'} sub={sub} onClose={close} />
          <TransferModal open={modal === 'transfer'} sub={sub} onClose={close} />
          <CancelModal open={modal === 'cancel'} sub={sub} onClose={close} />
          <CollectModal open={modal === 'collect'} invoice={sub.invoice} onClose={close} />
        </>
      )}
    </Drawer>
  );
}
