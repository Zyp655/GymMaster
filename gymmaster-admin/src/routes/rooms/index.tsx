import { useMemo, useState } from 'react';
import { Button, Card, Col, Row, Space, Tooltip, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { DeleteOutlined, EditOutlined } from '@ant-design/icons';
import roomApi from '@/api/room';
import type { Room, RoomListParams } from '@/api/room';
import ActionBar from '@/components/ActionBar';
import ConfirmButton from '@/components/ConfirmButton';
import DataTable from '@/components/DataTable';
import DateText from '@/components/DateText';
import FilterBar from '@/components/FilterBar';
import PageTitle from '@/components/PageTitle';
import StatusTag from '@/components/StatusTag';
import { ACTIVE_STATUS_OPTIONS } from '@/constants/status';
import { PERM } from '@/constants/permissions';
import { useBranchOptions } from '@/hooks/useBranchOptions';
import { useIsMobile } from '@/hooks/useIsMobile';
import { usePermission } from '@/hooks/usePermission';
import { useRowSelection } from '@/hooks/useRowSelection';
import { useTableQuery } from '@/hooks/useTableQuery';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import RoomForm from './RoomForm';
import RoomsMobile from './RoomsMobile';
import { useDeleteRooms } from './queries';

function RoomsDesktop() {
  const { can } = usePermission();
  const { branchId } = useWorkingBranch();
  const { branches, isLoading: branchesLoading } = useBranchOptions();
  const [modal, setModal] = useState<{ open: boolean; record: Room | null }>({
    open: false,
    record: null,
  });
  const selection = useRowSelection<Room>();
  const remove = useDeleteRooms();
  const canManage = can(PERM.ROOM_MANAGE);

  const table = useTableQuery<Room, RoomListParams>({
    queryKey: [roomApi.key, 'list'],
    queryFn: (params) => roomApi.list(params),
    filterKeys: ['status', 'branch_id'],
    numberKeys: ['branch_id'],
    defaults: { order_by: 'id', order_dir: 'DESC' },
    // Lọc chi nhánh trên trang ưu tiên hơn chi nhánh làm việc ở header.
    extraParams: branchId ? { branch_id: branchId } : {},
  });

  const openForm = (record: Room | null = null) => setModal({ open: true, record });

  const columns = useMemo<ColumnsType<Room>>(
    () => [
      {
        title: 'Tên phòng',
        dataIndex: 'name',
        sorter: true,
        render: (name: string, record) => (
          <Button type="link" size="small" style={{ padding: 0 }} onClick={() => openForm(record)}>
            {name}
          </Button>
        ),
      },
      { title: 'Chi nhánh', dataIndex: 'branch_name', key: 'branch_id', sorter: true },
      {
        title: 'Sức chứa',
        dataIndex: 'capacity',
        sorter: true,
        align: 'right',
        render: (v: number) => `${v} người`,
      },
      {
        title: 'Mô tả',
        dataIndex: 'description',
        render: (d: string) =>
          d ? (
            <Typography.Text ellipsis={{ tooltip: d }} style={{ maxWidth: 260 }}>
              {d}
            </Typography.Text>
          ) : (
            <Typography.Text type="secondary">—</Typography.Text>
          ),
      },
      {
        title: 'Trạng thái',
        dataIndex: 'status',
        sorter: true,
        render: (s: string) => <StatusTag status={s} />,
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
            {canManage && (
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
              perm={PERM.ROOM_MANAGE}
              type="text"
              size="small"
              danger
              icon={<DeleteOutlined />}
              confirmTitle={`Xóa phòng ${record.name}?`}
              confirmContent="Không thể xóa nếu phòng còn lịch lớp sắp diễn ra."
              onConfirm={() => remove.mutateAsync([record])}
              aria-label="Xóa"
            />
          </Space>
        ),
      },
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [canManage],
  );

  return (
    <Card variant="borderless" className="page-card">
      <Row justify="space-between" align="middle" gutter={[8, 12]} className="page-head">
        <Col>
          <PageTitle title="Phòng tập" subtitle="Phòng tập nhóm của từng chi nhánh" />
        </Col>
        <Col>
          <ActionBar
            onAdd={() => openForm()}
            addPerm={PERM.ROOM_MANAGE}
            onDelete={async () => {
              await remove.mutateAsync(selection.rows.map(({ id, name }) => ({ id, name })));
              selection.clear();
            }}
            deletePerm={PERM.ROOM_MANAGE}
            selectedCount={selection.keys.length}
            deleteConfirmTitle={`Xóa ${selection.keys.length} phòng đã chọn?`}
            deleteConfirmContent="Phòng còn lịch lớp sắp diễn ra sẽ không xóa được."
            onRefresh={() => table.refetch()}
            refreshing={table.isFetching}
          />
        </Col>
      </Row>

      <FilterBar
        searchPlaceholder="Tìm theo tên, mô tả..."
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
          { name: 'status', placeholder: 'Trạng thái', options: ACTIVE_STATUS_OPTIONS },
        ]}
      />

      <DataTable<Room>
        table={table}
        columns={columns}
        rowSelection={canManage ? selection.rowSelection : undefined}
        emptyText="Chưa có phòng nào"
      />

      <RoomForm
        open={modal.open}
        record={modal.record}
        onClose={() => setModal((m) => ({ ...m, open: false }))}
      />
    </Card>
  );
}

export default function RoomsPage() {
  return useIsMobile() ? <RoomsMobile /> : <RoomsDesktop />;
}
