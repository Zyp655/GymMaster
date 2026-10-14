import { useMemo, useState } from 'react';
import { Button, Card, Col, Row, Space, Tooltip, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { DeleteOutlined, EditOutlined, FileExcelOutlined } from '@ant-design/icons';
import { Link, useNavigate } from 'react-router-dom';
import memberApi, {
  GENDER_LABEL,
  GENDER_OPTIONS,
  MEMBER_STATUS_MAP,
  MEMBER_STATUS_OPTIONS,
  SOURCE_LABEL,
  SOURCE_OPTIONS,
  SUBSCRIPTION_FILTER_OPTIONS,
} from '@/api/member';
import type { Member, MemberItem, MemberListParams } from '@/api/member';
import ActionBar from '@/components/ActionBar';
import ConfirmButton from '@/components/ConfirmButton';
import DataTable from '@/components/DataTable';
import DateText from '@/components/DateText';
import FilterBar from '@/components/FilterBar';
import { MemberAvatar } from '@/components/MemberSelect';
import PageTitle from '@/components/PageTitle';
import PermissionGate from '@/components/PermissionGate';
import StatusTag from '@/components/StatusTag';
import { PERM } from '@/constants/permissions';
import { useBranchOptions } from '@/hooks/useBranchOptions';
import { useIsMobile } from '@/hooks/useIsMobile';
import { usePermission } from '@/hooks/usePermission';
import { useTableQuery } from '@/hooks/useTableQuery';
import { useUrlParams } from '@/hooks/useUrlParams';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { showError } from '@/utils/errors';
import { formatPhone } from '@/utils/format';
import ImportModal from './ImportModal';
import MemberForm from './MemberForm';
import MembersMobile from './MembersMobile';
import SubscriptionCell from './SubscriptionCell';
import { useDeleteMember } from './queries';

const TITLE = 'Hội viên';
const SUBTITLE = 'Hồ sơ hội viên, gói đang dùng và tình trạng hoạt động';

function MembersDesktop() {
  const { can } = usePermission();
  const navigate = useNavigate();
  const url = useUrlParams();
  const { branchId } = useWorkingBranch();
  const { branches } = useBranchOptions();
  const [formOpen, setFormOpen] = useState(false);
  const [importOpen, setImportOpen] = useState(false);
  const [editing, setEditing] = useState<Member | null>(null);
  const remove = useDeleteMember();

  // Bộ lọc chi nhánh trên URL ưu tiên hơn chi nhánh làm việc ở header.
  const urlBranch = url.get('branch_id');
  const table = useTableQuery<MemberItem, MemberListParams>({
    queryKey: [memberApi.key, 'list'],
    queryFn: (params) => memberApi.list(params),
    filterKeys: ['branch_id', 'status', 'gender', 'source', 'subscription_status'],
    numberKeys: ['branch_id'],
    defaults: { order_by: 'id', order_dir: 'DESC' },
    extraParams: { branch_id: urlBranch ? undefined : (branchId ?? undefined) },
  });

  const openEdit = async (id: number) => {
    // Form sửa cần đầy đủ trường (CCCD, địa chỉ...) → lấy hồ sơ chi tiết.
    try {
      setEditing(await memberApi.get(id));
      setFormOpen(true);
    } catch (e) {
      showError(e, 'Không tải được hồ sơ hội viên');
    }
  };

  const columns = useMemo<ColumnsType<MemberItem>>(
    () => [
      {
        title: 'Hội viên',
        dataIndex: 'full_name',
        sorter: true,
        render: (_, r) => (
          <Space size={10}>
            <MemberAvatar src={r.avatar_url} name={r.full_name} size={36} />
            <div style={{ lineHeight: 1.3 }}>
              <Link to={`/app/members/${r.id}`} style={{ fontWeight: 500 }}>
                {r.full_name}
              </Link>
              <br />
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                {GENDER_LABEL[r.gender]}
                {r.email ? ` · ${r.email}` : ''}
              </Typography.Text>
            </div>
          </Space>
        ),
      },
      {
        title: 'Mã',
        dataIndex: 'member_code',
        sorter: true,
        render: (code: string) => <span className="code-text">{code}</span>,
      },
      {
        title: 'Số điện thoại',
        dataIndex: 'phone',
        className: 'text-nowrap',
        render: (phone: string) => <a href={`tel:${phone}`}>{formatPhone(phone)}</a>,
      },
      { title: 'Chi nhánh', dataIndex: 'home_branch_name' },
      {
        title: 'Gói hiện tại',
        dataIndex: 'active_subscription',
        render: (_, r) => <SubscriptionCell sub={r.active_subscription} />,
      },
      {
        title: 'Nguồn',
        dataIndex: 'source',
        render: (s: MemberItem['source']) => SOURCE_LABEL[s] ?? s,
      },
      {
        title: 'Trạng thái',
        dataIndex: 'status',
        sorter: true,
        render: (s: string) => <StatusTag status={s} map={MEMBER_STATUS_MAP} />,
      },
      {
        title: 'Ngày tạo',
        dataIndex: 'created_at',
        sorter: true,
        render: (v: string) => <DateText value={v} />,
      },
      {
        title: '',
        key: 'actions',
        fixed: 'right',
        width: 88,
        render: (_, r) => (
          <Space size={0}>
            {can(PERM.MEMBER_UPDATE) && (
              <Tooltip title="Sửa">
                <Button
                  type="text"
                  size="small"
                  icon={<EditOutlined />}
                  onClick={() => void openEdit(r.id)}
                />
              </Tooltip>
            )}
            <ConfirmButton
              perm={PERM.MEMBER_DELETE}
              type="text"
              size="small"
              danger
              icon={<DeleteOutlined />}
              confirmTitle={`Xóa hội viên ${r.member_code}?`}
              confirmContent={`"${r.full_name}" sẽ bị xóa. Không thể xóa nếu hội viên còn gói đang hiệu lực hoặc bảo lưu.`}
              onConfirm={() => remove.mutateAsync(r.id)}
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
            onAdd={() => {
              setEditing(null);
              setFormOpen(true);
            }}
            addPerm={PERM.MEMBER_CREATE}
            onRefresh={() => table.refetch()}
            refreshing={table.isFetching}
          >
            <PermissionGate perm={PERM.MEMBER_IMPORT}>
              <Button icon={<FileExcelOutlined />} onClick={() => setImportOpen(true)}>
                Nhập Excel
              </Button>
            </PermissionGate>
          </ActionBar>
        </Col>
      </Row>

      <FilterBar
        searchPlaceholder="Tìm theo tên, SĐT, mã GM…, email"
        colTextSearch={5}
        filters={[
          {
            name: 'branch_id',
            placeholder: 'Chi nhánh',
            options: branches.map((b) => ({ label: b.name, value: b.id })),
            col: 4,
          },
          {
            name: 'subscription_status',
            placeholder: 'Tình trạng gói',
            options: SUBSCRIPTION_FILTER_OPTIONS,
            col: 4,
          },
          { name: 'status', placeholder: 'Trạng thái', options: MEMBER_STATUS_OPTIONS, col: 3 },
          { name: 'gender', placeholder: 'Giới tính', options: GENDER_OPTIONS, col: 3 },
          { name: 'source', placeholder: 'Nguồn', options: SOURCE_OPTIONS, col: 3 },
        ]}
      />

      <DataTable<MemberItem>
        table={table}
        columns={columns}
        emptyText="Chưa có hội viên nào"
        onRow={(r) => ({ onDoubleClick: () => navigate(`/app/members/${r.id}`) })}
      />

      <MemberForm
        open={formOpen}
        record={editing}
        onClose={() => setFormOpen(false)}
        onSaved={(m) => !editing && navigate(`/app/members/${m.id}`)}
      />
      <ImportModal open={importOpen} onClose={() => setImportOpen(false)} />
    </Card>
  );
}

export default function MembersPage() {
  const isMobile = useIsMobile();
  return isMobile ? <MembersMobile /> : <MembersDesktop />;
}
