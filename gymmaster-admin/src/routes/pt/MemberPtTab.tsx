import { useMemo, useState } from 'react';
import { Button, Divider, Empty, Result, Space, Table, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { PlusOutlined, ShoppingCartOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import dayjs from 'dayjs';
import type { Member, MemberOption } from '@/api/member';
import ptApi from '@/api/pt';
import type { PtContract, PtSession } from '@/api/pt';
import MoneyText from '@/components/Money/MoneyText';
import StatusTag from '@/components/StatusTag';
import { PERM } from '@/constants/permissions';
import { usePermission } from '@/hooks/usePermission';
import { weekdayName } from '@/routes/classes/constants';
import { getErrorMessage } from '@/utils/errors';
import { API_DATE_FORMAT, formatDate } from '@/utils/format';
import ContractDrawer from './ContractDrawer';
import SellPtModal from './SellPtModal';
import SessionDrawer from './SessionDrawer';
import { BookSessionModal } from './SessionModals';
import {
  PT_CONTRACT_STATUS_MAP,
  PT_SESSION_STATUS_MAP,
  isBookable,
  sessionStatusKey,
} from './constants';
import { useMemberPtContracts } from './queries';

/** Phần PT trong hồ sơ hội viên: hợp đồng PT + buổi sắp tới. */
export default function MemberPtTab({ member }: { member: Member }) {
  const { can } = usePermission();
  const [selling, setSelling] = useState(false);
  const [booking, setBooking] = useState(false);
  const [contractId, setContractId] = useState<number | null>(null);
  const [sessionId, setSessionId] = useState<number | null>(null);

  const memberOption: MemberOption = useMemo(
    () => ({
      id: member.id,
      member_code: member.member_code,
      full_name: member.full_name,
      phone: member.phone,
      avatar_url: member.avatar_url,
      status: member.status,
    }),
    [member],
  );

  const contracts = useMemberPtContracts(member.id);
  const params = {
    member_id: member.id,
    from: dayjs().format(API_DATE_FORMAT),
    to: dayjs().add(60, 'day').format(API_DATE_FORMAT),
    status: 'SCHEDULED' as const,
  };
  const upcoming = useQuery({
    queryKey: [ptApi.key, 'sessions', params],
    queryFn: () => ptApi.listSessions(params),
  });

  const hasBookable = (contracts.data ?? []).some(isBookable);

  const contractColumns: ColumnsType<PtContract> = [
    {
      title: 'Gói PT',
      key: 'pkg',
      render: (_, c) => (
        <Button type="link" size="small" style={{ padding: 0 }} onClick={() => setContractId(c.id)}>
          {c.package_name}
        </Button>
      ),
    },
    { title: 'HLV', dataIndex: 'trainer_name' },
    {
      title: 'Buổi',
      key: 'sessions',
      className: 'text-nowrap',
      render: (_, c) => (
        <span>
          {c.used_sessions}/{c.total_sessions}
          {c.scheduled_sessions > 0 && (
            <Typography.Text type="secondary"> · đặt {c.scheduled_sessions}</Typography.Text>
          )}
        </span>
      ),
    },
    {
      title: 'Thời hạn',
      key: 'period',
      className: 'text-nowrap',
      render: (_, c) => `${formatDate(c.start_date)} – ${formatDate(c.end_date)}`,
    },
    { title: 'Chi nhánh', dataIndex: 'branch_name', responsive: ['lg'] },
    {
      title: 'Giá trị',
      dataIndex: 'price',
      align: 'right',
      responsive: ['md'],
      render: (v: number) => <MoneyText value={v} />,
    },
    {
      title: 'Trạng thái',
      dataIndex: 'status',
      render: (v: string) => <StatusTag status={v} map={PT_CONTRACT_STATUS_MAP} />,
    },
  ];

  const sessionColumns: ColumnsType<PtSession> = [
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
    { title: 'HLV', dataIndex: 'trainer_name' },
    { title: 'Gói', dataIndex: 'package_name', responsive: ['md'] },
    { title: 'Chi nhánh', dataIndex: 'branch_name', responsive: ['lg'] },
    {
      title: 'Trạng thái',
      key: 'status',
      render: (_, s) => <StatusTag status={sessionStatusKey(s)} map={PT_SESSION_STATUS_MAP} />,
    },
  ];

  if (contracts.isError)
    return (
      <Result
        status="warning"
        title="Không tải được hợp đồng PT"
        subTitle={getErrorMessage(contracts.error)}
        extra={<Button onClick={() => contracts.refetch()}>Thử lại</Button>}
      />
    );

  return (
    <div>
      <Space wrap style={{ width: '100%', justifyContent: 'space-between', marginBottom: 12 }}>
        <Typography.Text strong>Hợp đồng PT</Typography.Text>
        <Space wrap>
          {can(PERM.PT_SCHEDULE) && (
            <Button
              icon={<PlusOutlined />}
              disabled={!hasBookable}
              onClick={() => setBooking(true)}
            >
              Đặt buổi
            </Button>
          )}
          {can(PERM.PT_SELL) && (
            <Button
              color="orange"
              variant="solid"
              icon={<ShoppingCartOutlined />}
              onClick={() => setSelling(true)}
            >
              Bán gói PT
            </Button>
          )}
        </Space>
      </Space>
      <Table<PtContract>
        rowKey="id"
        size="small"
        loading={contracts.isLoading}
        dataSource={contracts.data ?? []}
        columns={contractColumns}
        pagination={false}
        scroll={{ x: 'max-content' }}
        locale={{
          emptyText: (
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description="Hội viên chưa có hợp đồng PT"
            />
          ),
        }}
      />

      <Divider orientation="left">
        <Typography.Text strong>Buổi PT sắp tới (60 ngày)</Typography.Text>
      </Divider>
      {upcoming.isError ? (
        <Typography.Text type="danger">{getErrorMessage(upcoming.error)}</Typography.Text>
      ) : (
        <Table<PtSession>
          rowKey="id"
          size="small"
          loading={upcoming.isLoading}
          dataSource={upcoming.data ?? []}
          columns={sessionColumns}
          pagination={false}
          scroll={{ x: 'max-content' }}
          locale={{
            emptyText: (
              <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="Không có buổi PT sắp tới" />
            ),
          }}
        />
      )}

      <SellPtModal
        open={selling}
        onClose={() => setSelling(false)}
        member={memberOption}
        onOpenContract={setContractId}
      />
      <BookSessionModal open={booking} member={memberOption} onClose={() => setBooking(false)} />
      <ContractDrawer id={contractId} onClose={() => setContractId(null)} />
      <SessionDrawer
        sessionId={sessionId}
        onClose={() => setSessionId(null)}
        onOpenContract={setContractId}
      />
    </div>
  );
}
