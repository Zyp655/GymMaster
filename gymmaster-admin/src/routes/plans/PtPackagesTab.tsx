import { useCallback, useMemo, useState } from 'react';
import { Button, Col, Row, Space, Switch, Tooltip } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { DeleteOutlined, EditOutlined, EyeOutlined } from '@ant-design/icons';
import ptPackageApi from '@/api/ptPackage';
import type { PtPackage, PtPackageListParams } from '@/api/ptPackage';
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
import PtPackageForm from './PtPackageForm';
import { SALE_STATUS_MAP, SALE_STATUS_OPTIONS } from './constants';
import { useDeleteSalePackage, useSetSaleStatus } from './queries';

function usePtCommon() {
  const { can } = usePermission();
  const [modal, setModal] = useState<{ open: boolean; record: PtPackage | null }>({
    open: false,
    record: null,
  });
  return {
    canManage: can(PERM.PLAN_MANAGE),
    modal,
    openForm: (record: PtPackage | null = null) => setModal({ open: true, record }),
    closeForm: () => setModal((m) => ({ ...m, open: false })),
    setStatus: useSetSaleStatus('pt'),
    remove: useDeleteSalePackage('pt'),
  };
}

/** Tab "Gói PT" – bản desktop. */
export function PtPackagesDesktop() {
  const { canManage, modal, openForm, closeForm, setStatus, remove } = usePtCommon();

  const table = useTableQuery<PtPackage, PtPackageListParams>({
    queryKey: [ptPackageApi.key, 'list'],
    queryFn: (params) => ptPackageApi.list(params),
    filterKeys: ['status'],
    defaults: { order_by: 'price', order_dir: 'ASC' },
  });

  const columns = useMemo<ColumnsType<PtPackage>>(
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
          <Button type="link" size="small" style={{ padding: 0 }} onClick={() => openForm(r)}>
            {name}
          </Button>
        ),
      },
      { title: 'Số buổi', dataIndex: 'session_count', sorter: true, align: 'right' },
      {
        title: 'Phút/buổi',
        dataIndex: 'session_minutes',
        align: 'right',
        render: (v: number) => `${v} phút`,
      },
      {
        title: 'Hiệu lực',
        dataIndex: 'validity_days',
        sorter: true,
        align: 'right',
        render: (v: number) => `${v} ngày`,
      },
      {
        title: 'Giá',
        dataIndex: 'price',
        sorter: true,
        align: 'right',
        render: (v: number) => <MoneyText value={v} strong />,
      },
      {
        title: 'Giá/buổi',
        dataIndex: 'price_per_session',
        align: 'right',
        render: (v: number) => <MoneyText value={v} />,
      },
      {
        title: 'Đang bán',
        dataIndex: 'status',
        sorter: true,
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
              confirmTitle={`Xóa gói PT ${r.code}?`}
              confirmContent="Gói đã có hợp đồng PT không thể xóa – hãy chuyển sang Ngừng bán."
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
              { name: 'status', placeholder: 'Trạng thái', options: SALE_STATUS_OPTIONS, col: 5 },
            ]}
          />
        </Col>
        <Col>
          <ActionBar
            onAdd={() => openForm()}
            addPerm={PERM.PLAN_MANAGE}
            addText="Thêm gói PT"
            showDelete={false}
            onRefresh={() => table.refetch()}
            refreshing={table.isFetching}
          />
        </Col>
      </Row>
      <DataTable<PtPackage> table={table} columns={columns} emptyText="Chưa có gói PT nào" />
      <PtPackageForm open={modal.open} record={modal.record} onClose={closeForm} />
    </>
  );
}

/** Tab "Gói PT" – bản điện thoại. */
export function PtPackagesMobile() {
  const { canManage, modal, openForm, closeForm, setStatus, remove } = usePtCommon();
  const fetchPage = useCallback(({ page, limit, keyword, sort, filters }: MobileFetchArgs) => {
    const [order_by, order_dir] = (sort ?? 'price:ASC').split(':') as [string, SortDir];
    return ptPackageApi.list({
      page,
      limit,
      keyword: keyword || undefined,
      order_by,
      order_dir,
      status: (filters.status as ActiveStatus | undefined) || undefined,
    });
  }, []);

  return (
    <>
      <MobileList<PtPackage>
        title="Gói PT"
        queryKey={[ptPackageApi.key, 'list']}
        fetchPage={fetchPage}
        searchPlaceholder="Tìm theo mã, tên gói"
        emptyText="Chưa có gói PT nào"
        sorts={[
          { label: 'Giá thấp → cao', value: 'price:ASC' },
          { label: 'Số buổi', value: 'session_count:ASC' },
          { label: 'Mới nhất', value: 'id:DESC' },
        ]}
        filters={[{ name: 'status', label: 'Trạng thái', options: SALE_STATUS_OPTIONS }]}
        row={{
          title: (r) => r.name,
          subtitle: (r) => `${formatMoney(r.price)} · ${formatMoney(r.price_per_session)}/buổi`,
          meta: (r) => [
            r.code,
            `${r.session_count} buổi × ${r.session_minutes} phút`,
            `${r.validity_days} ngày`,
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
          ['Giá/buổi', formatMoney(r.price_per_session)],
          ['Hiệu lực', `${r.validity_days} ngày`],
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
      <PtPackageForm open={modal.open} record={modal.record} onClose={closeForm} />
    </>
  );
}
