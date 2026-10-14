import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Button, Space, Table, Tooltip, Typography } from 'antd';
import { EyeOutlined, PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import { Link } from 'react-router-dom';
import type { Member } from '@/api/member';
import subscriptionApi from '@/api/subscription';
import type { Subscription } from '@/api/subscription';
import MoneyText from '@/components/Money/MoneyText';
import StatusTag from '@/components/StatusTag';
import { PERM } from '@/constants/permissions';
import { INVOICE_STATUS_MAP, SUBSCRIPTION_STATUS_MAP } from '@/constants/status';
import { usePermission } from '@/hooks/usePermission';
import { getErrorMessage } from '@/utils/errors';
import { formatDate } from '@/utils/format';
import RenewModal from './RenewModal';
import SellSubscriptionModal from './SellSubscriptionModal';
import SubscriptionDrawer from './SubscriptionDrawer';
import { remainingText } from './utils';

/** Tab "Gói tập" của hồ sơ hội viên: lịch sử gói + Bán gói / Gia hạn / xem chi tiết. */
export default function MemberSubscriptionsTab({ member }: { member: Member }) {
  const { can } = usePermission();
  const [selling, setSelling] = useState(false);
  const [renewing, setRenewing] = useState<Subscription | null>(null);
  const [viewId, setViewId] = useState<number | null>(null);

  const query = useQuery({
    queryKey: [subscriptionApi.key, 'member', member.id],
    queryFn: () => subscriptionApi.byMember(member.id),
    enabled: can(PERM.SUBSCRIPTION_VIEW),
  });

  const canSell = can(PERM.SUBSCRIPTION_SELL) && member.status === 'ACTIVE';

  return (
    <>
      <Space style={{ marginBottom: 12, justifyContent: 'space-between', width: '100%' }} wrap>
        <Typography.Text type="secondary">
          {member.active_subscription
            ? `Gói hiệu lực: ${member.active_subscription.plan_name} đến ${formatDate(member.active_subscription.end_date)}`
            : 'Hội viên chưa có gói hiệu lực'}
        </Typography.Text>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => query.refetch()}>
            Làm mới
          </Button>
          {canSell && (
            <Button
              color="orange"
              variant="solid"
              icon={<PlusOutlined />}
              onClick={() => setSelling(true)}
            >
              Bán gói
            </Button>
          )}
        </Space>
      </Space>
      <Table<Subscription>
        size="small"
        rowKey="id"
        loading={query.isFetching}
        dataSource={query.data ?? []}
        pagination={false}
        scroll={{ x: 'max-content' }}
        locale={{
          emptyText: query.isError
            ? getErrorMessage(query.error, 'Không tải được gói tập')
            : 'Hội viên chưa mua gói nào',
        }}
        columns={[
          {
            title: 'Gói',
            key: 'plan',
            render: (_, r) => (
              <Button
                type="link"
                size="small"
                style={{ padding: 0 }}
                onClick={() => setViewId(r.id)}
              >
                {r.plan_name}
              </Button>
            ),
          },
          { title: 'Chi nhánh', dataIndex: 'branch_name' },
          {
            title: 'Thời gian',
            key: 'time',
            className: 'text-nowrap',
            render: (_, r) => `${formatDate(r.start_date)} – ${formatDate(r.end_date)}`,
          },
          { title: 'Còn lại', key: 'left', render: (_, r) => remainingText(r) },
          {
            title: 'Giá trị',
            dataIndex: 'price',
            align: 'right',
            render: (v: number) => <MoneyText value={v} />,
          },
          {
            title: 'Trạng thái',
            dataIndex: 'status',
            render: (v: string) => <StatusTag status={v} map={SUBSCRIPTION_STATUS_MAP} />,
          },
          {
            title: 'Hóa đơn',
            key: 'inv',
            render: (_, r) =>
              r.invoice_id ? (
                <Space size={4}>
                  <Link to={`/app/invoices/${r.invoice_id}`} className="code-text">
                    {r.invoice_no}
                  </Link>
                  <StatusTag status={r.invoice_status} map={INVOICE_STATUS_MAP} />
                </Space>
              ) : (
                '—'
              ),
          },
          {
            title: '',
            key: 'actions',
            fixed: 'right',
            render: (_, r) => (
              <Space size={0}>
                {can(PERM.SUBSCRIPTION_SELL) &&
                  ['ACTIVE', 'FROZEN', 'EXPIRED'].includes(r.status) && (
                    <Button size="small" type="link" onClick={() => setRenewing(r)}>
                      Gia hạn
                    </Button>
                  )}
                <Tooltip title="Chi tiết">
                  <Button
                    size="small"
                    type="text"
                    icon={<EyeOutlined />}
                    onClick={() => setViewId(r.id)}
                    aria-label="Chi tiết"
                  />
                </Tooltip>
              </Space>
            ),
          },
        ]}
      />
      <SellSubscriptionModal
        open={selling}
        onClose={() => setSelling(false)}
        member={{
          id: member.id,
          member_code: member.member_code,
          full_name: member.full_name,
          phone: member.phone,
          avatar_url: member.avatar_url,
          status: member.status,
        }}
      />
      <RenewModal open={!!renewing} sub={renewing} onClose={() => setRenewing(null)} />
      <SubscriptionDrawer id={viewId} onClose={() => setViewId(null)} />
    </>
  );
}
