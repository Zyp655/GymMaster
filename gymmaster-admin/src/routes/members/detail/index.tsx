import { Suspense, useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  App,
  Button,
  Card,
  Col,
  Dropdown,
  Result,
  Row,
  Skeleton,
  Space,
  Spin,
  Tabs,
  Tag,
  Typography,
} from 'antd';
import type { MenuProps } from 'antd';
import {
  DeleteOutlined,
  EditOutlined,
  LockOutlined,
  LoginOutlined,
  MoreOutlined,
  PhoneOutlined,
  QrcodeOutlined,
  SyncOutlined,
  UserAddOutlined,
} from '@ant-design/icons';
import { useNavigate, useParams, useSearchParams } from 'react-router-dom';
import memberApi, { MEMBER_STATUS_MAP } from '@/api/member';
import { MemberAvatar } from '@/components/MemberSelect';
import PageTitle from '@/components/PageTitle';
import StatusTag from '@/components/StatusTag';
import { PERM } from '@/constants/permissions';
import { usePermission } from '@/hooks/usePermission';
import { getErrorMessage } from '@/utils/errors';
import { formatPhone } from '@/utils/format';
import MemberForm from '../MemberForm';
import { useDeleteMember, useLockPortal, useResetQr } from '../queries';
import ManualCheckinModal from './ManualCheckinModal';
import PortalModal from './PortalModal';
import QrModal from './QrModal';
import { getMemberTabs } from './tabs';

type ModalKey = 'edit' | 'portal' | 'qr' | 'checkin' | null;

/** Hồ sơ hội viên `/app/members/:id`: header + Tabs cấu hình bằng registry (`./tabs.tsx`). */
export default function MemberDetailPage() {
  const { id: idParam } = useParams();
  const id = Number(idParam);
  const navigate = useNavigate();
  const [search, setSearch] = useSearchParams();
  const { can } = usePermission();
  const { modal } = App.useApp();
  const [open, setOpen] = useState<ModalKey>(null);

  const query = useQuery({
    queryKey: [memberApi.key, 'detail', id],
    queryFn: () => memberApi.get(id),
    enabled: Number.isInteger(id) && id > 0,
  });
  const m = query.data;

  const remove = useDeleteMember();
  const lockPortal = useLockPortal(id);
  const resetQr = useResetQr(id);

  const tabs = useMemo(() => getMemberTabs().filter((t) => can(t.perm)), [can]);
  const activeTab =
    tabs.find((t) => t.key === search.get('tab'))?.key ?? tabs[0]?.key ?? 'overview';

  if (query.isLoading) {
    return (
      <Card variant="borderless" className="page-card">
        <Skeleton avatar active paragraph={{ rows: 6 }} />
      </Card>
    );
  }
  if (query.isError || !m) {
    return (
      <Card variant="borderless" className="page-card">
        <Result
          status="warning"
          title={getErrorMessage(query.error, 'Không tìm thấy hội viên')}
          extra={
            <Space>
              <Button onClick={() => navigate('/app/members')}>Về danh sách</Button>
              <Button type="primary" onClick={() => query.refetch()}>
                Thử lại
              </Button>
            </Space>
          }
        />
      </Card>
    );
  }

  const portal = m.portal_account;
  const portalActive = portal?.status === 'ACTIVE';

  const confirm = (title: string, content: string, onOk: () => Promise<unknown>, danger = false) =>
    modal.confirm({
      title,
      content,
      okText: 'Đồng ý',
      cancelText: 'Hủy',
      okButtonProps: { danger },
      onOk: () => onOk().catch(() => undefined),
    });

  const moreItems: MenuProps['items'] = [
    ...(can(PERM.MEMBER_UPDATE)
      ? [
          portalActive
            ? {
                key: 'lock',
                icon: <LockOutlined />,
                label: 'Khóa tài khoản cổng',
                onClick: () =>
                  confirm(
                    'Khóa tài khoản cổng hội viên?',
                    `Tài khoản ${portal?.email} sẽ bị khóa và đăng xuất khỏi mọi thiết bị.`,
                    () => lockPortal.mutateAsync(),
                    true,
                  ),
              }
            : {
                key: 'portal',
                icon: <UserAddOutlined />,
                label: portal ? 'Mở lại tài khoản cổng' : 'Cấp tài khoản cổng hội viên',
                onClick: () => setOpen('portal'),
              },
          {
            key: 'reset-qr',
            icon: <SyncOutlined />,
            label: 'Đặt lại QR',
            onClick: () =>
              confirm(
                'Đặt lại mã QR?',
                'Mọi mã QR cũ của hội viên (kể cả đang mở trên ứng dụng) sẽ lập tức hết hiệu lực.',
                () => resetQr.mutateAsync(),
              ),
          },
        ]
      : []),
    ...(can(PERM.MEMBER_DELETE)
      ? [
          { type: 'divider' as const },
          {
            key: 'delete',
            icon: <DeleteOutlined />,
            danger: true,
            label: 'Xóa hội viên',
            onClick: () =>
              confirm(
                `Xóa hội viên ${m.member_code}?`,
                `"${m.full_name}" sẽ bị xóa. Không thể xóa nếu còn gói đang hiệu lực hoặc bảo lưu; tài khoản cổng (nếu có) sẽ bị khóa.`,
                () => remove.mutateAsync(m.id).then(() => navigate('/app/members')),
                true,
              ),
          },
        ]
      : []),
  ];

  return (
    <>
      <Card variant="borderless" className="page-card">
        <PageTitle
          title="Hồ sơ hội viên"
          back="/app/members"
          breadcrumb={[{ title: 'Hội viên', path: '/app/members' }, { title: m.member_code }]}
        />
        <Row gutter={[16, 16]} align="middle" style={{ marginTop: 8 }}>
          <Col flex="none">
            <MemberAvatar src={m.avatar_url} name={m.full_name} size={96} />
          </Col>
          <Col flex="auto" style={{ minWidth: 220 }}>
            <Typography.Title level={4} style={{ margin: 0 }}>
              {m.full_name} <StatusTag status={m.status} map={MEMBER_STATUS_MAP} />
            </Typography.Title>
            <Space size={[12, 4]} wrap style={{ marginTop: 4 }}>
              <span className="code-text">{m.member_code}</span>
              <a href={`tel:${m.phone}`}>
                <PhoneOutlined /> {formatPhone(m.phone)}
              </a>
              <Typography.Text type="secondary">{m.home_branch?.name ?? '—'}</Typography.Text>
              {portal && (
                <Tag color={portalActive ? 'green' : 'red'} bordered={false}>
                  Cổng hội viên: {portalActive ? 'hoạt động' : 'đã khóa'}
                </Tag>
              )}
            </Space>
          </Col>
          <Col flex="none">
            <Space wrap>
              {can(PERM.CHECKIN_SCAN) && (
                <>
                  <Button icon={<QrcodeOutlined />} onClick={() => setOpen('qr')}>
                    Hiển thị QR
                  </Button>
                  <Button
                    color="orange"
                    variant="solid"
                    icon={<LoginOutlined />}
                    onClick={() => setOpen('checkin')}
                  >
                    Check-in thủ công
                  </Button>
                </>
              )}
              {can(PERM.MEMBER_UPDATE) && (
                <Button type="primary" icon={<EditOutlined />} onClick={() => setOpen('edit')}>
                  Sửa
                </Button>
              )}
              {moreItems.length > 0 && (
                <Dropdown menu={{ items: moreItems }} trigger={['click']}>
                  <Button icon={<MoreOutlined />} aria-label="Thao tác khác" />
                </Dropdown>
              )}
            </Space>
          </Col>
        </Row>
      </Card>

      <Card variant="borderless" className="page-card" style={{ marginTop: 16 }}>
        <Tabs
          activeKey={activeTab}
          onChange={(key) => setSearch({ tab: key }, { replace: true })}
          destroyOnHidden
          items={tabs.map((t) => ({
            key: t.key,
            label: t.label,
            children: (
              <Suspense fallback={<Spin style={{ display: 'block', margin: '40px auto' }} />}>
                {t.render(m)}
              </Suspense>
            ),
          }))}
        />
      </Card>

      <MemberForm open={open === 'edit'} record={m} onClose={() => setOpen(null)} />
      <PortalModal member={m} open={open === 'portal'} onClose={() => setOpen(null)} />
      <QrModal member={m} open={open === 'qr'} onClose={() => setOpen(null)} />
      <ManualCheckinModal member={m} open={open === 'checkin'} onClose={() => setOpen(null)} />
    </>
  );
}
