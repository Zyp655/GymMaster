import { useMemo, useState } from 'react';
import { App, Button, Card, Col, Row, Space, Table, Tag, Tooltip, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { DeleteOutlined, EditOutlined, LockOutlined } from '@ant-design/icons';
import type { Role } from '@/api/types';
import ActionBar from '@/components/ActionBar';
import ConfirmButton from '@/components/ConfirmButton';
import FilterBar from '@/components/FilterBar';
import PageTitle from '@/components/PageTitle';
import { PERM, ROLE } from '@/constants/permissions';
import { usePermission } from '@/hooks/usePermission';
import { useUrlParams } from '@/hooks/useUrlParams';
import { getErrorMessage, showError } from '@/utils/errors';
import { normalizeText } from '@/utils/format';
import RoleForm from './RoleForm';
import { useDeleteRole, useRoles } from './queries';

const TYPE_OPTIONS = [
  { label: 'Vai trò hệ thống', value: 'system' },
  { label: 'Vai trò tùy chỉnh', value: 'custom' },
];

/**
 * Vai trò & quyền. API trả mảng (không phân trang) → lọc phía client,
 * nhưng vẫn dùng FilterBar (giá trị trên URL) để thống nhất trải nghiệm.
 */
export default function RolesPage() {
  const { can } = usePermission();
  const { message } = App.useApp();
  const url = useUrlParams();
  const roles = useRoles();
  const remove = useDeleteRole();
  const [modal, setModal] = useState<{ open: boolean; record: Role | null }>({
    open: false,
    record: null,
  });

  const openForm = (record: Role | null = null) => setModal({ open: true, record });

  const keyword = normalizeText(url.get('keyword') ?? '');
  const type = url.get('type');
  const data = useMemo(
    () =>
      (roles.data ?? []).filter(
        (r) =>
          (!keyword || normalizeText(`${r.code} ${r.name} ${r.description}`).includes(keyword)) &&
          (!type || (type === 'system' ? r.is_system : !r.is_system)),
      ),
    [roles.data, keyword, type],
  );

  const columns: ColumnsType<Role> = [
    {
      title: 'Mã',
      dataIndex: 'code',
      width: 150,
      sorter: (a, b) => a.code.localeCompare(b.code),
      render: (code: string) => <span className="code-text">{code}</span>,
    },
    {
      title: 'Tên vai trò',
      dataIndex: 'name',
      sorter: (a, b) => a.name.localeCompare(b.name, 'vi'),
      render: (name: string, r) => (
        <Button type="link" size="small" style={{ padding: 0 }} onClick={() => openForm(r)}>
          {name}
        </Button>
      ),
    },
    {
      title: 'Mô tả',
      dataIndex: 'description',
      ellipsis: true,
      render: (d: string) => d || <Typography.Text type="secondary">—</Typography.Text>,
    },
    {
      title: 'Quyền',
      key: 'permissions',
      sorter: (a, b) => (a.permissions?.length ?? 0) - (b.permissions?.length ?? 0),
      render: (_, r) =>
        r.code === ROLE.OWNER ? (
          <Tag color="gold" bordered={false}>
            Toàn quyền
          </Tag>
        ) : (
          <span>{r.permissions?.length ?? 0} quyền</span>
        ),
    },
    {
      title: 'Loại',
      dataIndex: 'is_system',
      render: (sys: boolean) =>
        sys ? (
          <Tag icon={<LockOutlined />} color="blue" bordered={false}>
            Hệ thống
          </Tag>
        ) : (
          <Tag bordered={false}>Tùy chỉnh</Tag>
        ),
    },
    {
      title: '',
      key: 'actions',
      fixed: 'right',
      width: 88,
      render: (_, r) => (
        <Space size={0}>
          {can(PERM.ROLE_UPDATE) && (
            <Tooltip title="Sửa">
              <Button
                type="text"
                size="small"
                icon={<EditOutlined />}
                onClick={() => openForm(r)}
              />
            </Tooltip>
          )}
          {!r.is_system && (
            <ConfirmButton
              perm={PERM.ROLE_DELETE}
              type="text"
              size="small"
              danger
              icon={<DeleteOutlined />}
              confirmTitle={`Xóa vai trò ${r.name}?`}
              confirmContent="Không thể xóa nếu vẫn còn tài khoản giữ vai trò này."
              onConfirm={() =>
                remove.mutateAsync(r.id, {
                  onSuccess: () => message.success('Đã xóa vai trò'),
                  onError: (e) => showError(e, 'Không thể xóa vai trò'),
                })
              }
              aria-label="Xóa"
            />
          )}
        </Space>
      ),
    },
  ];

  return (
    <Card variant="borderless" className="page-card">
      <Row justify="space-between" align="middle" gutter={[8, 12]} className="page-head">
        <Col>
          <PageTitle
            title="Vai trò & quyền"
            subtitle="Phân quyền truy cập cho nhân viên theo vai trò"
          />
        </Col>
        <Col>
          <ActionBar
            onAdd={() => openForm()}
            addPerm={PERM.ROLE_CREATE}
            showDelete={false}
            onRefresh={() => roles.refetch()}
            refreshing={roles.isFetching}
          />
        </Col>
      </Row>

      <FilterBar
        searchPlaceholder="Tìm theo mã, tên vai trò..."
        filters={[{ name: 'type', placeholder: 'Loại vai trò', options: TYPE_OPTIONS, col: 5 }]}
      />

      <Table<Role>
        size="small"
        rowKey="id"
        columns={columns}
        dataSource={data}
        loading={roles.isFetching}
        scroll={{ x: 'max-content' }}
        pagination={false}
        locale={{
          emptyText: roles.isError ? (
            <Space direction="vertical">
              <Typography.Text type="danger">
                {getErrorMessage(roles.error, 'Không tải được dữ liệu')}
              </Typography.Text>
              <Button onClick={() => roles.refetch()}>Thử lại</Button>
            </Space>
          ) : (
            'Không có vai trò phù hợp'
          ),
        }}
      />

      <RoleForm
        open={modal.open}
        record={modal.record}
        onClose={() => setModal((m) => ({ ...m, open: false }))}
      />
    </Card>
  );
}
