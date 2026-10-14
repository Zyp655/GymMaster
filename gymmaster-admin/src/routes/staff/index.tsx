import { useMemo, useState } from 'react';
import { Avatar, Button, Card, Col, Dropdown, Row, Space, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { MoreOutlined } from '@ant-design/icons';
import staffApi from '@/api/staff';
import type { Staff, StaffListParams } from '@/api/staff';
import ActionBar from '@/components/ActionBar';
import DataTable from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import PageTitle from '@/components/PageTitle';
import StatusTag from '@/components/StatusTag';
import { PERM } from '@/constants/permissions';
import { useBranchOptions } from '@/hooks/useBranchOptions';
import { useIsMobile } from '@/hooks/useIsMobile';
import { useTableQuery } from '@/hooks/useTableQuery';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { formatPhone } from '@/utils/format';
import ResetPasswordModal from './ResetPasswordModal';
import StaffForm from './StaffForm';
import StaffMobile from './StaffMobile';
import {
  POSITION_META,
  POSITION_OPTIONS,
  STAFF_STATUS_MAP,
  STAFF_STATUS_OPTIONS,
} from './constants';
import { useStaffActions } from './useStaffActions';

function StaffDesktop() {
  const { branchId } = useWorkingBranch();
  const { branches, isLoading: branchesLoading } = useBranchOptions();
  const [modal, setModal] = useState<{ open: boolean; record: Staff | null }>({
    open: false,
    record: null,
  });
  const [resetFor, setResetFor] = useState<Staff | null>(null);
  const openForm = (record: Staff | null = null) => setModal({ open: true, record });
  const { itemsFor, pendingId } = useStaffActions({
    onEdit: openForm,
    onResetPassword: setResetFor,
  });

  const table = useTableQuery<Staff, StaffListParams>({
    queryKey: [staffApi.key, 'list'],
    queryFn: (params) => staffApi.list(params),
    filterKeys: ['branch_id', 'position', 'status'],
    numberKeys: ['branch_id'],
    defaults: { order_by: 'id', order_dir: 'DESC' },
    extraParams: branchId ? { branch_id: branchId } : {},
  });

  const columns = useMemo<ColumnsType<Staff>>(
    () => [
      {
        title: 'Mã NV',
        key: 'staff_code',
        dataIndex: ['staff_profile', 'staff_code'],
        sorter: true,
        render: (code?: string) =>
          code ? (
            <span className="code-text">{code}</span>
          ) : (
            <Typography.Text type="secondary">—</Typography.Text>
          ),
      },
      {
        title: 'Họ tên',
        dataIndex: 'full_name',
        sorter: true,
        render: (name: string, r) => (
          <Space size={8}>
            <Avatar src={r.avatar_url || undefined} size="small" style={{ background: '#0A3150' }}>
              {name.charAt(0).toUpperCase()}
            </Avatar>
            <Button type="link" size="small" style={{ padding: 0 }} onClick={() => openForm(r)}>
              {name}
            </Button>
          </Space>
        ),
      },
      {
        title: 'Chức vụ',
        key: 'position',
        dataIndex: ['staff_profile', 'position'],
        sorter: true,
        render: (p?: string, r?: Staff) =>
          p ? (
            <Space size={4}>
              <StatusTag status={p} map={POSITION_META} />
              {r?.trainer_profile && (
                <Tag bordered={false} color="gold">
                  {r.trainer_profile.level}
                </Tag>
              )}
            </Space>
          ) : (
            <Typography.Text type="secondary">—</Typography.Text>
          ),
      },
      {
        title: 'Vai trò × chi nhánh',
        key: 'roles',
        render: (_, r) =>
          r.roles.length ? (
            <Space size={[4, 4]} wrap style={{ maxWidth: 320 }}>
              {r.roles.map((x) => (
                <Tag key={`${x.role_id}-${x.branch_id ?? 0}`} bordered={false} color="blue">
                  {x.role_name} · {x.branch_name || 'Toàn hệ thống'}
                </Tag>
              ))}
            </Space>
          ) : (
            <Typography.Text type="secondary">—</Typography.Text>
          ),
      },
      {
        title: 'Số điện thoại',
        dataIndex: 'phone',
        className: 'text-nowrap',
        render: (phone: string) =>
          phone ? <a href={`tel:${phone}`}>{formatPhone(phone)}</a> : '—',
      },
      {
        title: 'Email',
        dataIndex: 'email',
        sorter: true,
        render: (email: string) => (
          <Typography.Text ellipsis={{ tooltip: email }} style={{ maxWidth: 200 }}>
            {email}
          </Typography.Text>
        ),
      },
      {
        title: 'Trạng thái',
        dataIndex: 'status',
        sorter: true,
        render: (s: string) => <StatusTag status={s} map={STAFF_STATUS_MAP} />,
      },
      {
        title: '',
        key: 'actions',
        fixed: 'right',
        width: 56,
        render: (_, r) => (
          <Dropdown
            trigger={['click']}
            menu={{
              items: itemsFor(r).map(({ key, label, icon, danger, disabled }) => ({
                key,
                label,
                icon,
                danger,
                disabled,
              })),
              onClick: ({ key, domEvent }) => {
                domEvent.stopPropagation();
                itemsFor(r)
                  .find((i) => i.key === key)
                  ?.onClick(r);
              },
            }}
          >
            <Button
              type="text"
              size="small"
              icon={<MoreOutlined />}
              loading={pendingId === r.id}
              aria-label="Thao tác"
            />
          </Dropdown>
        ),
      },
    ],
    [itemsFor, pendingId],
  );

  return (
    <Card variant="borderless" className="page-card">
      <Row justify="space-between" align="middle" gutter={[8, 12]} className="page-head">
        <Col>
          <PageTitle
            title="Nhân viên & HLV"
            subtitle="Hồ sơ nhân sự, vai trò và phân quyền theo chi nhánh"
          />
        </Col>
        <Col>
          <ActionBar
            onAdd={() => openForm()}
            addPerm={PERM.STAFF_CREATE}
            addText="Thêm nhân viên"
            showDelete={false}
            onRefresh={() => table.refetch()}
            refreshing={table.isFetching}
          />
        </Col>
      </Row>

      <FilterBar
        searchPlaceholder="Tìm theo tên, email, SĐT, mã NV..."
        filters={[
          ...(branchId
            ? []
            : [
                {
                  name: 'branch_id',
                  placeholder: 'Chi nhánh',
                  options: branches.map((b) => ({ label: b.name, value: b.id })),
                  loading: branchesLoading,
                  col: 5,
                },
              ]),
          { name: 'position', placeholder: 'Chức vụ', options: POSITION_OPTIONS },
          { name: 'status', placeholder: 'Trạng thái', options: STAFF_STATUS_OPTIONS },
        ]}
      />

      <DataTable<Staff> table={table} columns={columns} emptyText="Chưa có nhân viên nào" />

      <StaffForm
        open={modal.open}
        record={modal.record}
        onClose={() => setModal((m) => ({ ...m, open: false }))}
      />
      <ResetPasswordModal staff={resetFor} onClose={() => setResetFor(null)} />
    </Card>
  );
}

export default function StaffPage() {
  return useIsMobile() ? <StaffMobile /> : <StaffDesktop />;
}
