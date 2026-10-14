import { useMemo, useState } from 'react';
import { Button, Card, Col, Row, Space, Tooltip, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { DeleteOutlined, EditOutlined } from '@ant-design/icons';
import branchApi from '@/api/branch';
import type { Branch, BranchListParams } from '@/api/types';
import ActionBar from '@/components/ActionBar';
import ConfirmButton from '@/components/ConfirmButton';
import DataTable from '@/components/DataTable';
import DateText from '@/components/DateText';
import FilterBar from '@/components/FilterBar';
import PageTitle from '@/components/PageTitle';
import StatusTag from '@/components/StatusTag';
import { ACTIVE_STATUS_OPTIONS } from '@/constants/status';
import { PERM } from '@/constants/permissions';
import { useIsMobile } from '@/hooks/useIsMobile';
import { usePermission } from '@/hooks/usePermission';
import { useRowSelection } from '@/hooks/useRowSelection';
import { useTableQuery } from '@/hooks/useTableQuery';
import { formatPhone } from '@/utils/format';
import BranchForm from './BranchForm';
import BranchesMobile from './BranchesMobile';
import { useDeleteBranches } from './queries';

const TITLE = 'Chi nhánh';
const SUBTITLE = 'Quản lý các cơ sở của trung tâm';

/** Bản desktop – đúng pattern Brands của strongbody_admin: Card → [PageTitle | ActionBar] → FilterBar → Table. */
function BranchesDesktop() {
  const { can } = usePermission();
  const [modal, setModal] = useState<{ open: boolean; record: Branch | null }>({
    open: false,
    record: null,
  });
  const selection = useRowSelection<Branch>();
  const remove = useDeleteBranches();

  const table = useTableQuery<Branch, BranchListParams>({
    queryKey: [branchApi.key, 'list'],
    queryFn: (params) => branchApi.list(params),
    filterKeys: ['status'],
    defaults: { order_by: 'id', order_dir: 'DESC' },
  });

  const openForm = (record: Branch | null = null) => setModal({ open: true, record });

  const columns = useMemo<ColumnsType<Branch>>(
    () => [
      {
        title: 'Mã',
        dataIndex: 'code',
        sorter: true,
        render: (code: string) => <span className="code-text">{code}</span>,
      },
      {
        title: 'Tên chi nhánh',
        dataIndex: 'name',
        sorter: true,
        render: (name: string, record) => (
          <Button type="link" size="small" style={{ padding: 0 }} onClick={() => openForm(record)}>
            {name}
          </Button>
        ),
      },
      {
        title: 'Địa chỉ',
        dataIndex: 'address',
        render: (address: string) =>
          address ? (
            <Typography.Text ellipsis={{ tooltip: address }} style={{ maxWidth: 220 }}>
              {address}
            </Typography.Text>
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
        title: 'Giờ mở cửa',
        key: 'hours',
        className: 'text-nowrap',
        render: (_, r) => `${r.open_time} – ${r.close_time}`,
      },
      {
        title: 'Trạng thái',
        dataIndex: 'status',
        sorter: true,
        render: (status: string) => <StatusTag status={status} />,
      },
      {
        title: 'Cập nhật lúc',
        dataIndex: 'updated_at',
        render: (v: string) => <DateText value={v} withTime />,
      },
      {
        title: '',
        key: 'actions',
        fixed: 'right',
        width: 88,
        render: (_, record) => (
          <Space size={0}>
            {can(PERM.BRANCH_UPDATE) && (
              <Tooltip title="Sửa">
                <Button
                  type="text"
                  size="small"
                  icon={<EditOutlined />}
                  onClick={() => openForm(record)}
                />
              </Tooltip>
            )}
            <ConfirmButton
              perm={PERM.BRANCH_DELETE}
              type="text"
              size="small"
              danger
              icon={<DeleteOutlined />}
              confirmTitle={`Xóa chi nhánh ${record.code}?`}
              confirmContent={`"${record.name}" sẽ bị xóa. Không thể xóa nếu còn nhân viên thuộc chi nhánh.`}
              onConfirm={() => remove.mutateAsync([record])}
              aria-label="Xóa"
            />
          </Space>
        ),
      },
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [can],
  );

  return (
    <Card variant="borderless" className="page-card">
      <Row justify="space-between" align="middle" gutter={[8, 12]} className="page-head">
        <Col>
          <PageTitle title={TITLE} subtitle={SUBTITLE} />
        </Col>
        <Col>
          <ActionBar
            onAdd={() => openForm()}
            addPerm={PERM.BRANCH_CREATE}
            onDelete={async () => {
              await remove.mutateAsync(selection.rows.map(({ id, name }) => ({ id, name })));
              selection.clear();
            }}
            deletePerm={PERM.BRANCH_DELETE}
            selectedCount={selection.keys.length}
            deleteConfirmTitle={`Xóa ${selection.keys.length} chi nhánh đã chọn?`}
            deleteConfirmContent="Chi nhánh còn nhân viên sẽ không xóa được."
            onRefresh={() => table.refetch()}
            refreshing={table.isFetching}
          />
        </Col>
      </Row>

      <FilterBar
        searchPlaceholder="Tìm theo mã, tên, địa chỉ..."
        filters={[{ name: 'status', placeholder: 'Trạng thái', options: ACTIVE_STATUS_OPTIONS }]}
      />

      <DataTable<Branch>
        table={table}
        columns={columns}
        rowSelection={can(PERM.BRANCH_DELETE) ? selection.rowSelection : undefined}
        emptyText="Chưa có chi nhánh nào"
      />

      <BranchForm
        open={modal.open}
        record={modal.record}
        onClose={() => setModal((m) => ({ ...m, open: false }))}
      />
    </Card>
  );
}

/** Điện thoại dùng màn hình thẻ riêng; desktop dùng bảng. */
export default function BranchesPage() {
  const isMobile = useIsMobile();
  return isMobile ? <BranchesMobile /> : <BranchesDesktop />;
}
