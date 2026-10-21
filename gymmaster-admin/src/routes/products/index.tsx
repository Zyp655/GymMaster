import { useCallback, useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { App, Button, Card, Col, Row, Space, Tag, Tooltip, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { DeleteOutlined, EditOutlined, EyeOutlined } from '@ant-design/icons';
import productApi, { productCategoryApi } from '@/api/product';
import type { Product, ProductListParams } from '@/api/product';
import type { ActiveStatus, SortDir } from '@/api/types';
import ActionBar from '@/components/ActionBar';
import ConfirmButton from '@/components/ConfirmButton';
import DataTable from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import MobileList from '@/components/MobileList';
import type { MobileFetchArgs } from '@/components/MobileList';
import MoneyText from '@/components/Money/MoneyText';
import PageTitle from '@/components/PageTitle';
import StatusTag from '@/components/StatusTag';
import { PERM } from '@/constants/permissions';
import { PRODUCT_STATUS_MAP, PRODUCT_STATUS_OPTIONS } from '@/constants/status';
import { useBranchOptions } from '@/hooks/useBranchOptions';
import { useIsMobile } from '@/hooks/useIsMobile';
import { usePermission } from '@/hooks/usePermission';
import { useRowSelection } from '@/hooks/useRowSelection';
import { useTableQuery } from '@/hooks/useTableQuery';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { formatMoney, formatNumber } from '@/utils/format';
import ProductForm from './ProductForm';
import ProductThumb from './ProductThumb';
import { useDeleteProducts } from './queries';

const TITLE = 'Sản phẩm';
const SUBTITLE = 'Hàng hóa bán tại quầy – giá bán, giá hội viên, tồn kho';
const LOW_OPTIONS = [{ label: 'Chỉ tồn thấp', value: 'true' }];

function useCategoryOptions() {
  const q = useQuery({
    queryKey: [productCategoryApi.key, 'list'],
    queryFn: () => productCategoryApi.list(),
    staleTime: 60_000,
  });
  return (q.data ?? []).map((c) => ({ label: c.name, value: c.id }));
}

function ProductsDesktop() {
  const { can } = usePermission();
  const { branchId } = useWorkingBranch();
  const { branches } = useBranchOptions();
  const branchName = branches.find((b) => b.id === branchId)?.name;
  const [modal, setModal] = useState<{ open: boolean; record: Product | null }>({
    open: false,
    record: null,
  });
  const selection = useRowSelection<Product>();
  const remove = useDeleteProducts();
  const categoryOptions = useCategoryOptions();

  const table = useTableQuery<Product, ProductListParams>({
    queryKey: [productApi.key, 'list'],
    queryFn: (p) => productApi.list(p),
    filterKeys: ['status', 'category_id', 'low_stock'],
    numberKeys: ['category_id'],
    defaults: { order_by: 'id', order_dir: 'DESC' },
    extraParams: { branch_id: branchId ?? undefined },
  });

  const openForm = (record: Product | null = null) => setModal({ open: true, record });

  const columns = useMemo<ColumnsType<Product>>(
    () => [
      {
        title: '',
        dataIndex: 'image_url',
        width: 56,
        render: (v: string) => <ProductThumb src={v} />,
      },
      {
        title: 'SKU',
        dataIndex: 'sku',
        sorter: true,
        width: 130,
        render: (v: string, r) => (
          <Space direction="vertical" size={0}>
            <span className="code-text">{v}</span>
            {r.barcode && (
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                {r.barcode}
              </Typography.Text>
            )}
          </Space>
        ),
      },
      {
        // Độ rộng cố định + xuống dòng: tên dài không đẩy cột giá/tồn/trạng thái ra sau cột thao tác cố định ở 1440px.
        title: 'Tên sản phẩm',
        dataIndex: 'name',
        sorter: true,
        width: 200,
        render: (name: string, r) => (
          <Button
            type="link"
            size="small"
            style={{ padding: 0, whiteSpace: 'normal', textAlign: 'left', height: 'auto' }}
            onClick={() => openForm(r)}
          >
            {name}
          </Button>
        ),
      },
      {
        title: 'Danh mục',
        dataIndex: 'category_name',
        width: 120,
        ellipsis: true,
        render: (v: string) =>
          v || <Typography.Text type="secondary">Chưa phân loại</Typography.Text>,
      },
      {
        title: (
          <Tooltip title="Giá vốn bình quân gia quyền – tự tính lại mỗi lần nhập kho, không sửa tay">
            Giá vốn
          </Tooltip>
        ),
        dataIndex: 'cost_price',
        sorter: true,
        align: 'right',
        width: 105,
        render: (v: number) => <MoneyText value={v} />,
      },
      {
        // Giá hội viên hiển thị ngay dưới giá bán (gộp cột để bảng vừa màn hình 1440px).
        title: 'Giá bán / HV',
        dataIndex: 'sell_price',
        sorter: true,
        align: 'right',
        width: 115,
        render: (v: number, r) => (
          <Space direction="vertical" size={0} align="end">
            <MoneyText value={v} strong />
            {r.member_price !== null && (
              <Tooltip title="Giá hội viên">
                <Typography.Text type="secondary" style={{ fontSize: 12, whiteSpace: 'nowrap' }}>
                  HV {formatMoney(r.member_price)}
                </Typography.Text>
              </Tooltip>
            )}
          </Space>
        ),
      },
      {
        title: branchName ? `Tồn tại ${branchName}` : 'Tổng tồn',
        key: 'stock',
        align: 'right',
        width: 125,
        render: (_, r) => (
          <Space size={4} wrap style={{ justifyContent: 'flex-end' }}>
            <span>
              {formatNumber(branchId ? (r.stock ?? 0) : r.total_stock)} {r.unit}
            </span>
            {r.low_stock && (
              <Tag color="orange" bordered={false}>
                Tồn thấp
              </Tag>
            )}
          </Space>
        ),
      },
      {
        title: 'Trạng thái',
        dataIndex: 'status',
        sorter: true,
        width: 115,
        render: (v: string) => <StatusTag status={v} map={PRODUCT_STATUS_MAP} />,
      },
      {
        title: '',
        key: 'actions',
        fixed: 'right',
        width: 80,
        render: (_, r) => (
          <Space size={0}>
            <Tooltip title={can(PERM.PRODUCT_MANAGE) ? 'Sửa' : 'Xem'}>
              <Button
                type="text"
                size="small"
                icon={can(PERM.PRODUCT_MANAGE) ? <EditOutlined /> : <EyeOutlined />}
                onClick={() => openForm(r)}
              />
            </Tooltip>
            <ConfirmButton
              perm={PERM.PRODUCT_MANAGE}
              type="text"
              size="small"
              danger
              icon={<DeleteOutlined />}
              confirmTitle={`Xóa sản phẩm ${r.sku}?`}
              confirmContent={`"${r.name}" sẽ bị xóa. Chỉ xóa được khi tổng tồn mọi chi nhánh bằng 0.`}
              onConfirm={() => remove.mutateAsync([r])}
              aria-label="Xóa"
            />
          </Space>
        ),
      },
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [can, branchId, branchName],
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
            addPerm={PERM.PRODUCT_MANAGE}
            onDelete={async () => {
              await remove.mutateAsync(selection.rows.map(({ id, name }) => ({ id, name })));
              selection.clear();
            }}
            deletePerm={PERM.PRODUCT_MANAGE}
            selectedCount={selection.keys.length}
            deleteConfirmTitle={`Xóa ${selection.keys.length} sản phẩm đã chọn?`}
            deleteConfirmContent="Sản phẩm còn tồn kho sẽ không xóa được."
            onRefresh={() => table.refetch()}
            refreshing={table.isFetching}
          />
        </Col>
      </Row>

      <FilterBar
        searchPlaceholder="Tìm theo tên, SKU, mã vạch..."
        filters={[
          { name: 'category_id', placeholder: 'Danh mục', options: categoryOptions },
          { name: 'status', placeholder: 'Trạng thái', options: PRODUCT_STATUS_OPTIONS },
          ...(branchId
            ? [{ name: 'low_stock', placeholder: 'Tồn kho', options: LOW_OPTIONS, col: 3 }]
            : []),
        ]}
      />

      <DataTable<Product>
        table={table}
        columns={columns}
        rowSelection={can(PERM.PRODUCT_MANAGE) ? selection.rowSelection : undefined}
        emptyText="Chưa có sản phẩm nào"
        // Bố cục cố định theo độ rộng cột: ở 1440px mọi cột (giá/tồn/trạng thái) nằm trước cột thao tác cố định.
        tableLayout="fixed"
        scroll={{ x: 1080 }}
      />

      <ProductForm
        open={modal.open}
        record={modal.record}
        onClose={() => setModal((m) => ({ ...m, open: false }))}
      />
    </Card>
  );
}

function ProductsMobile() {
  const { can } = usePermission();
  const { modal } = App.useApp();
  const { branchId } = useWorkingBranch();
  const remove = useDeleteProducts();
  const categoryOptions = useCategoryOptions();
  const [form, setForm] = useState<{ open: boolean; record: Product | null }>({
    open: false,
    record: null,
  });

  const fetchPage = useCallback(
    ({ page, limit, keyword, sort, filters }: MobileFetchArgs) => {
      const [order_by, order_dir] = (sort ?? 'id:DESC').split(':') as [string, SortDir];
      return productApi.list({
        page,
        limit,
        keyword: keyword || undefined,
        order_by,
        order_dir,
        status: (filters.status as ActiveStatus | undefined) || undefined,
        category_id: (filters.category_id as number | undefined) || undefined,
        branch_id: branchId ?? undefined,
        low_stock: branchId && filters.low_stock ? true : undefined,
      });
    },
    [branchId],
  );

  const openForm = (record: Product | null) => setForm({ open: true, record });

  return (
    <>
      <MobileList<Product>
        title={TITLE}
        queryKey={[productApi.key, 'list', branchId]}
        fetchPage={fetchPage}
        searchPlaceholder="Tên, SKU, mã vạch"
        emptyText="Chưa có sản phẩm nào"
        sorts={[
          { label: 'Mới nhất', value: 'id:DESC' },
          { label: 'Tên A → Z', value: 'name:ASC' },
          { label: 'Giá bán tăng dần', value: 'sell_price:ASC' },
        ]}
        filters={[
          { name: 'category_id', label: 'Danh mục', options: categoryOptions },
          { name: 'status', label: 'Trạng thái', options: PRODUCT_STATUS_OPTIONS },
          ...(branchId ? [{ name: 'low_stock', label: 'Tồn kho', options: LOW_OPTIONS }] : []),
        ]}
        row={{
          thumb: (r) => r.image_url || undefined,
          title: (r) => r.name,
          subtitle: (r) => `${r.sku} · ${formatMoney(r.sell_price)}`,
          meta: (r) => [
            r.category_name,
            `Tồn ${formatNumber(branchId ? (r.stock ?? 0) : r.total_stock)} ${r.unit}`,
          ],
          tags: (r) => [
            ...(r.low_stock ? [{ label: 'Tồn thấp', color: 'orange' }] : []),
            {
              label: PRODUCT_STATUS_MAP[r.status]?.label,
              color: PRODUCT_STATUS_MAP[r.status]?.color,
            },
          ],
        }}
        onAdd={() => openForm(null)}
        addPerm={PERM.PRODUCT_MANAGE}
        details={(r) => [
          ['Giá vốn', formatMoney(r.cost_price)],
          ['Giá hội viên', r.member_price === null ? '—' : formatMoney(r.member_price)],
          ['Mã vạch', r.barcode],
          ['Tồn tối thiểu', formatNumber(r.min_stock)],
        ]}
        actions={(r) => [
          {
            key: 'edit',
            icon: can(PERM.PRODUCT_MANAGE) ? <EditOutlined /> : <EyeOutlined />,
            label: can(PERM.PRODUCT_MANAGE) ? 'Sửa sản phẩm' : 'Xem chi tiết',
            onClick: openForm,
          },
          ...(can(PERM.PRODUCT_MANAGE)
            ? [
                {
                  key: 'delete',
                  icon: <DeleteOutlined />,
                  label: 'Xóa sản phẩm',
                  danger: true,
                  onClick: () =>
                    modal.confirm({
                      title: `Xóa sản phẩm ${r.sku}?`,
                      content: 'Chỉ xóa được khi tổng tồn mọi chi nhánh bằng 0.',
                      okText: 'Xóa',
                      okButtonProps: { danger: true },
                      cancelText: 'Hủy',
                      onOk: () => remove.mutateAsync([r]).catch(() => undefined),
                    }),
                },
              ]
            : []),
        ]}
      />
      <ProductForm
        open={form.open}
        record={form.record}
        onClose={() => setForm((f) => ({ ...f, open: false }))}
      />
    </>
  );
}

export default function ProductsPage() {
  return useIsMobile() ? <ProductsMobile /> : <ProductsDesktop />;
}
