import { useMemo, useState } from 'react';
import { Button, Card, Col, Row, Space, Tooltip, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { DeleteOutlined, EditOutlined } from '@ant-design/icons';
import classApi from '@/api/class';
import type { ClassType, ClassTypeListParams } from '@/api/class';
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
import { LEVEL_META, LEVEL_OPTIONS } from '../classes/constants';
import ClassTypeForm from './ClassTypeForm';
import ClassTypesMobile from './ClassTypesMobile';
import { useDeleteClassTypes } from './queries';

const TITLE = 'Loại lớp';
const SUBTITLE = 'Danh mục lớp nhóm dùng chung toàn hệ thống';

/** Ô màu nhỏ của loại lớp. */
export function ColorDot({ color, size = 14 }: { color: string; size?: number }) {
  return (
    <span
      aria-hidden
      style={{
        display: 'inline-block',
        width: size,
        height: size,
        borderRadius: 4,
        background: color,
        boxShadow: 'inset 0 0 0 1px rgba(0,0,0,.08)',
        verticalAlign: 'middle',
      }}
    />
  );
}

function ClassTypesDesktop() {
  const { can } = usePermission();
  const [modal, setModal] = useState<{ open: boolean; record: ClassType | null }>({
    open: false,
    record: null,
  });
  const selection = useRowSelection<ClassType>();
  const remove = useDeleteClassTypes();

  const table = useTableQuery<ClassType, ClassTypeListParams>({
    queryKey: [classApi.typeKey, 'list'],
    queryFn: (params) => classApi.listTypes(params),
    filterKeys: ['status', 'level'],
    defaults: { order_by: 'id', order_dir: 'DESC' },
  });

  const openForm = (record: ClassType | null = null) => setModal({ open: true, record });

  const columns = useMemo<ColumnsType<ClassType>>(
    () => [
      {
        title: 'Màu',
        dataIndex: 'color',
        width: 60,
        align: 'center',
        render: (color: string) => (
          // ColorDot không forwardRef → bọc <span> để Tooltip không phải dùng findDOMNode.
          <Tooltip title={color}>
            <span style={{ display: 'inline-flex' }}>
              <ColorDot color={color} size={18} />
            </span>
          </Tooltip>
        ),
      },
      {
        title: 'Mã',
        dataIndex: 'code',
        sorter: true,
        render: (code: string) => <span className="code-text">{code}</span>,
      },
      {
        title: 'Tên loại lớp',
        dataIndex: 'name',
        sorter: true,
        render: (name: string, record) => (
          <div>
            <Button
              type="link"
              size="small"
              style={{ padding: 0 }}
              onClick={() => openForm(record)}
            >
              {name}
            </Button>
            {record.description && (
              <Typography.Text
                type="secondary"
                ellipsis={{ tooltip: record.description }}
                style={{ display: 'block', maxWidth: 260, fontSize: 12 }}
              >
                {record.description}
              </Typography.Text>
            )}
          </div>
        ),
      },
      {
        title: 'Thời lượng',
        dataIndex: 'default_duration_min',
        align: 'right',
        className: 'text-nowrap',
        render: (v: number) => `${v} phút`,
      },
      {
        title: 'Sức chứa',
        dataIndex: 'default_capacity',
        align: 'right',
        className: 'text-nowrap',
        render: (v: number) => `${v} người`,
      },
      {
        title: 'Cấp độ',
        dataIndex: 'level',
        sorter: true,
        render: (level: string) => <StatusTag status={level} map={LEVEL_META} />,
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
            {can(PERM.CLASS_MANAGE) && (
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
              perm={PERM.CLASS_MANAGE}
              type="text"
              size="small"
              danger
              icon={<DeleteOutlined />}
              confirmTitle={`Xóa loại lớp ${record.code}?`}
              confirmContent={`"${record.name}" sẽ bị xóa. Không thể xóa nếu còn lịch lớp sắp diễn ra.`}
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
            addPerm={PERM.CLASS_MANAGE}
            onDelete={async () => {
              await remove.mutateAsync(selection.rows.map(({ id, name }) => ({ id, name })));
              selection.clear();
            }}
            deletePerm={PERM.CLASS_MANAGE}
            selectedCount={selection.keys.length}
            deleteConfirmTitle={`Xóa ${selection.keys.length} loại lớp đã chọn?`}
            deleteConfirmContent="Loại lớp còn lịch sắp diễn ra sẽ không xóa được."
            onRefresh={() => table.refetch()}
            refreshing={table.isFetching}
          />
        </Col>
      </Row>

      <FilterBar
        searchPlaceholder="Tìm theo mã, tên..."
        filters={[
          { name: 'level', placeholder: 'Cấp độ', options: LEVEL_OPTIONS },
          { name: 'status', placeholder: 'Trạng thái', options: ACTIVE_STATUS_OPTIONS },
        ]}
      />

      <DataTable<ClassType>
        table={table}
        columns={columns}
        rowSelection={can(PERM.CLASS_MANAGE) ? selection.rowSelection : undefined}
        emptyText="Chưa có loại lớp nào"
      />

      <ClassTypeForm
        open={modal.open}
        record={modal.record}
        onClose={() => setModal((m) => ({ ...m, open: false }))}
      />
    </Card>
  );
}

export default function ClassTypesPage() {
  const isMobile = useIsMobile();
  return isMobile ? <ClassTypesMobile /> : <ClassTypesDesktop />;
}
