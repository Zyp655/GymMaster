import { useCallback, useMemo, useState } from 'react';
import { Button, Col, Row, Space, Switch, Tag, Tooltip, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { DeleteOutlined, EditOutlined, EyeOutlined } from '@ant-design/icons';
import membershipPlanApi from '@/api/plan';
import type { MembershipPlan, MembershipPlanListParams, PlanType } from '@/api/plan';
import type { ActiveStatus, SortDir } from '@/api/types';
import ActionBar from '@/components/ActionBar';
import ConfirmButton from '@/components/ConfirmButton';
import DataTable from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import MobileList from '@/components/MobileList';
import type { MobileFetchArgs } from '@/components/MobileList';
import { MoneyText } from '@/components/Money';
import StatusTag from '@/components/StatusTag';
import { PERM } from '@/constants/permissions';
import { usePermission } from '@/hooks/usePermission';
import { useTableQuery } from '@/hooks/useTableQuery';
import { formatMoney } from '@/utils/format';
import PlanForm from './PlanForm';
import {
  ACCESS_SCOPE_LABELS,
  accessHoursText,
  PLAN_TYPE_META,
  PLAN_TYPE_OPTIONS,
  SALE_STATUS_MAP,
  SALE_STATUS_OPTIONS,
} from './constants';
import { useDeleteSalePackage, useSetSaleStatus } from './queries';

const durationText = (p: MembershipPlan) =>
  p.type === 'SESSION'
    ? `${p.session_count} buổi / ${p.duration_days} ngày`
    : `${p.duration_days} ngày`;

const freezeText = (p: MembershipPlan) =>
  p.max_freeze_times > 0 ? `${p.max_freeze_days} ngày / ${p.max_freeze_times} lần` : 'Không';

function usePlanCommon() {
  const { can } = usePermission();
  const [modal, setModal] = useState<{ open: boolean; record: MembershipPlan | null }>({
    open: false,
    record: null,
  });
  const setStatus = useSetSaleStatus('plan');
  const remove = useDeleteSalePackage('plan');
  return {
    canManage: can(PERM.PLAN_MANAGE),
    modal,
    openForm: (record: MembershipPlan | null = null) => setModal({ open: true, record }),
    closeForm: () => setModal((m) => ({ ...m, open: false })),
    setStatus,
    remove,
  };
}

/** Tab "Gói tập" – bản desktop. */
export function MembershipPlansDesktop() {
  const { canManage, modal, openForm, closeForm, setStatus, remove } = usePlanCommon();

  const table = useTableQuery<MembershipPlan, MembershipPlanListParams>({
    queryKey: [membershipPlanApi.key, 'list'],
    queryFn: (params) => membershipPlanApi.list(params),
    filterKeys: ['status', 'type'],
    defaults: { order_by: 'sort_order', order_dir: 'ASC' },
  });

  const columns = useMemo<ColumnsType<MembershipPlan>>(
    () => [
      {
        title: 'Mã',
        dataIndex: 'code',
        sorter: true,
        render: (c: string) => <span className="code-text">{c}</span>,
      },
      {
        title: 'Tên gói',
        dataIndex: 'name',
        sorter: true,
        render: (name: string, r) => (
          <Space direction="vertical" size={0}>
            <Button
              type="link"
              size="small"
              style={{ padding: 0, height: 'auto' }}
              onClick={() => openForm(r)}
            >
              {name}
            </Button>
            {/* Phạm vi hiển thị dưới tên để bảng vừa khung 1440 (D8). */}
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              {ACCESS_SCOPE_LABELS[r.access_scope] ?? r.access_scope}
            </Typography.Text>
          </Space>
        ),
      },
      {
        title: 'Loại',
        dataIndex: 'type',
        sorter: true,
        render: (t: string) => <StatusTag status={t} map={PLAN_TYPE_META} />,
      },
      {
        title: 'Thời hạn',
        dataIndex: 'duration_days',
        sorter: true,
        render: (_, r) => (
          <Space direction="vertical" size={0}>
            <span className="text-nowrap">{durationText(r)}</span>
            <Typography.Text type="secondary" style={{ fontSize: 12 }} className="text-nowrap">
              Bảo lưu: {freezeText(r)}
            </Typography.Text>
          </Space>
        ),
      },
      {
        title: 'Giá',
        dataIndex: 'price',
        sorter: true,
        align: 'right',
        render: (v: number) => <MoneyText value={v} strong />,
      },
      {
        // Gộp Lớp nhóm + Vé mời để bảng vừa khung 1440 (D8).
        title: 'Quyền lợi',
        key: 'benefits',
        render: (_, r) =>
          r.includes_classes || r.guest_passes_per_month ? (
            <Space direction="vertical" size={2}>
              {r.includes_classes && (
                <Tag color="green" bordered={false}>
                  Lớp nhóm
                </Tag>
              )}
              {!!r.guest_passes_per_month && (
                <Tag color="blue" bordered={false}>
                  {r.guest_passes_per_month} vé mời/tháng
                </Tag>
              )}
            </Space>
          ) : (
            <Typography.Text type="secondary">—</Typography.Text>
          ),
      },
      {
        title: 'Khung giờ vào tập',
        key: 'access_hours',
        render: (_, r) =>
          r.access_hours?.length ? (
            <Typography.Text
              ellipsis={{ tooltip: accessHoursText(r.access_hours) }}
              style={{ maxWidth: 140 }}
            >
              {accessHoursText(r.access_hours)}
            </Typography.Text>
          ) : (
            <Typography.Text type="secondary">Mọi giờ</Typography.Text>
          ),
      },
      {
        title: 'Thứ tự',
        dataIndex: 'sort_order',
        sorter: true,
        align: 'right',
        responsive: ['xxl'],
      },
      {
        title: 'Đang bán',
        dataIndex: 'status',
        sorter: true,
        fixed: 'right',
        width: 90,
        render: (s: ActiveStatus, r) =>
          canManage ? (
            <Switch
              size="small"
              checked={s === 'ACTIVE'}
              loading={setStatus.isPending && setStatus.variables?.id === r.id}
              onChange={(on) => setStatus.mutate({ id: r.id, status: on ? 'ACTIVE' : 'INACTIVE' })}
            />
          ) : (
            <StatusTag status={s} map={SALE_STATUS_MAP} />
          ),
      },
      {
        title: '',
        key: 'actions',
        fixed: 'right',
        width: 88,
        render: (_, r) => (
          <Space size={0}>
            {canManage && (
              <Tooltip title="Sửa">
                <Button
                  type="text"
                  size="small"
                  icon={<EditOutlined />}
                  onClick={() => openForm(r)}
                />
              </Tooltip>
            )}
            <ConfirmButton
              perm={PERM.PLAN_MANAGE}
              type="text"
              size="small"
              danger
              icon={<DeleteOutlined />}
              confirmTitle={`Xóa gói tập ${r.code}?`}
              confirmContent="Gói đã có hội viên đăng ký không thể xóa – hãy chuyển sang Ngừng bán."
              onConfirm={() => remove.mutateAsync(r)}
              aria-label="Xóa"
            />
          </Space>
        ),
      },
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [canManage, setStatus.isPending, setStatus.variables],
  );

  return (
    <>
      <Row justify="space-between" gutter={[8, 8]} wrap>
        <Col flex="auto">
          <FilterBar
            searchPlaceholder="Tìm theo mã, tên gói..."
            colTextSearch={8}
            filters={[
              { name: 'type', placeholder: 'Loại gói', options: PLAN_TYPE_OPTIONS, col: 5 },
              { name: 'status', placeholder: 'Trạng thái', options: SALE_STATUS_OPTIONS, col: 5 },
            ]}
          />
        </Col>
        <Col>
          <ActionBar
            onAdd={() => openForm()}
            addPerm={PERM.PLAN_MANAGE}
            addText="Thêm gói tập"
            showDelete={false}
            onRefresh={() => table.refetch()}
            refreshing={table.isFetching}
          />
        </Col>
      </Row>
      <DataTable<MembershipPlan> table={table} columns={columns} emptyText="Chưa có gói tập nào" />
      <PlanForm open={modal.open} record={modal.record} onClose={closeForm} />
    </>
  );
}

/** Tab "Gói tập" – bản điện thoại. */
export function MembershipPlansMobile() {
  const { canManage, modal, openForm, closeForm, setStatus, remove } = usePlanCommon();
  const fetchPage = useCallback(({ page, limit, keyword, sort, filters }: MobileFetchArgs) => {
    const [order_by, order_dir] = (sort ?? 'sort_order:ASC').split(':') as [string, SortDir];
    return membershipPlanApi.list({
      page,
      limit,
      keyword: keyword || undefined,
      order_by,
      order_dir,
      type: (filters.type as PlanType | undefined) || undefined,
      status: (filters.status as ActiveStatus | undefined) || undefined,
    });
  }, []);

  return (
    <>
      <MobileList<MembershipPlan>
        title="Gói tập"
        queryKey={[membershipPlanApi.key, 'list']}
        fetchPage={fetchPage}
        searchPlaceholder="Tìm theo mã, tên gói"
        emptyText="Chưa có gói tập nào"
        sorts={[
          { label: 'Thứ tự hiển thị', value: 'sort_order:ASC' },
          { label: 'Giá thấp → cao', value: 'price:ASC' },
          { label: 'Mới nhất', value: 'id:DESC' },
        ]}
        filters={[
          { name: 'type', label: 'Loại gói', options: PLAN_TYPE_OPTIONS },
          { name: 'status', label: 'Trạng thái', options: SALE_STATUS_OPTIONS },
        ]}
        row={{
          title: (r) => r.name,
          subtitle: (r) => `${formatMoney(r.price)} · ${durationText(r)}`,
          meta: (r) => [r.code, ACCESS_SCOPE_LABELS[r.access_scope]],
          tags: (r) => [
            { label: PLAN_TYPE_META[r.type]?.label, color: PLAN_TYPE_META[r.type]?.color },
          ],
          toggle: {
            value: (r) => r.status === 'ACTIVE',
            onChange: canManage
              ? (r, on) => setStatus.mutate({ id: r.id, status: on ? 'ACTIVE' : 'INACTIVE' })
              : undefined,
          },
        }}
        onAdd={() => openForm(null)}
        addPerm={PERM.PLAN_MANAGE}
        details={(r) => [
          ['Mã', r.code],
          ['Giá', formatMoney(r.price)],
          ['Thời hạn', durationText(r)],
          ['Bảo lưu', freezeText(r)],
          ['Lớp nhóm', r.includes_classes ? 'Có' : 'Không'],
          ['Phí chuyển nhượng', formatMoney(r.transfer_fee)],
          ['Vé mời / tháng', String(r.guest_passes_per_month ?? 0)],
          ['Khung giờ vào tập', accessHoursText(r.access_hours)],
        ]}
        actions={(r) => [
          {
            key: 'edit',
            icon: canManage ? <EditOutlined /> : <EyeOutlined />,
            label: canManage ? 'Sửa gói' : 'Xem chi tiết',
            onClick: openForm,
          },
          ...(canManage
            ? [
                {
                  key: 'delete',
                  icon: <DeleteOutlined />,
                  label: 'Xóa gói',
                  danger: true,
                  onClick: () => remove.mutate(r),
                },
              ]
            : []),
        ]}
      />
      <PlanForm open={modal.open} record={modal.record} onClose={closeForm} />
    </>
  );
}
