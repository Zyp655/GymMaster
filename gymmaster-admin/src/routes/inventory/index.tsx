import { useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  Button,
  Card,
  Col,
  DatePicker,
  Empty,
  Input,
  Row,
  Space,
  Table,
  Tabs,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  ExportOutlined,
  FileSearchOutlined,
  HistoryOutlined,
  ImportOutlined,
  ReconciliationOutlined,
  SwapOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import { useSearchParams } from 'react-router-dom';
import inventoryApi from '@/api/inventory';
import type {
  StockListResult,
  StockListParams,
  StockRow,
  StockTransaction,
  TransactionListParams,
  Voucher,
} from '@/api/inventory';
import { productCategoryApi } from '@/api/product';
import ActionBar from '@/components/ActionBar';
import DataTable from '@/components/DataTable';
import DateText from '@/components/DateText';
import FilterBar from '@/components/FilterBar';
import MoneyText from '@/components/Money/MoneyText';
import PageTitle from '@/components/PageTitle';
import StatCard from '@/components/StatCard';
import StatusTag from '@/components/StatusTag';
import { PERM } from '@/constants/permissions';
import { STOCK_TX_TYPE_MAP, STOCK_TX_TYPE_OPTIONS, VOUCHER_KIND_MAP } from '@/constants/status';
import { useBranchOptions } from '@/hooks/useBranchOptions';
import { usePermission } from '@/hooks/usePermission';
import { useTableQuery } from '@/hooks/useTableQuery';
import { useUrlParams } from '@/hooks/useUrlParams';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import ProductThumb from '@/routes/products/ProductThumb';
import { DATE_FORMAT, formatNumber, toApiDate } from '@/utils/format';
import AdjustmentModal from './AdjustmentModal';
import ImportModal from './ImportModal';
import QtyText from './QtyText';
import TransferModal from './TransferModal';
import VoucherModal from './VoucherModal';
import { useProductOptions } from './queries';
import './inventory.css';

type TabKey = 'stock' | 'transactions' | 'vouchers';
const LOW_OPTIONS = [{ label: 'Chỉ tồn thấp', value: 'true' }];

/** Mã phiếu dạng link mở chi tiết. */
function VoucherLink({ no, onOpen }: { no: string; onOpen: (no: string) => void }) {
  if (!no) return <>—</>;
  return (
    <Button
      type="link"
      size="small"
      className="code-text"
      style={{ padding: 0 }}
      onClick={() => onOpen(no)}
    >
      {no}
    </Button>
  );
}

/* ------------------------------------------------------------------ Tồn kho */

function StockTab({ onHistory }: { onHistory: (productId: number) => void }) {
  const { branchId } = useWorkingBranch();
  const url = useUrlParams();
  const categories = useQuery({
    queryKey: [productCategoryApi.key, 'list'],
    queryFn: () => productCategoryApi.list(),
    staleTime: 60_000,
  });
  const table = useTableQuery<StockRow, StockListParams>({
    queryKey: [inventoryApi.key, 'stock'],
    queryFn: (p) => inventoryApi.stock(p),
    filterKeys: ['category_id', 'low_stock'],
    numberKeys: ['category_id'],
    defaults: { order_by: 'product_id', order_dir: 'ASC' },
    extraParams: { branch_id: branchId ?? undefined },
  });
  const summary = (table.query.data as StockListResult | undefined)?.summary;
  const lowOnly = url.get('low_stock') === 'true';

  const columns: ColumnsType<StockRow> = [
    {
      title: 'Sản phẩm',
      dataIndex: 'name',
      sorter: true,
      render: (_, r) => (
        <Space>
          <ProductThumb src={r.image_url} size={36} />
          <span>
            {r.product_name}
            <br />
            <Typography.Text type="secondary" className="code-text" style={{ fontSize: 12 }}>
              {r.sku}
            </Typography.Text>
          </span>
        </Space>
      ),
    },
    ...(branchId ? [] : [{ title: 'Chi nhánh', dataIndex: 'branch_name', key: 'branch_id', sorter: true }]),
    { title: 'Danh mục', dataIndex: 'category_name', render: (v: string) => v || '—' },
    {
      title: 'Tồn',
      dataIndex: 'quantity',
      sorter: true,
      align: 'right',
      render: (q: number, r) => (
        <Space size={4}>
          <b style={{ color: r.low_stock ? '#ea580c' : undefined }}>
            {formatNumber(q)} {r.unit}
          </b>
          {r.low_stock && (
            <Tag color="orange" bordered={false}>
              Tồn thấp
            </Tag>
          )}
        </Space>
      ),
    },
    {
      title: 'Tồn tối thiểu',
      dataIndex: 'min_stock',
      sorter: true,
      align: 'right',
      render: (v: number) => (v ? formatNumber(v) : '—'),
    },
    {
      title: 'Giá vốn',
      dataIndex: 'cost_price',
      align: 'right',
      render: (v: number) => <MoneyText value={v} />,
    },
    {
      title: 'Giá trị tồn',
      dataIndex: 'stock_value',
      align: 'right',
      render: (v: number) => <MoneyText value={v} strong />,
    },
    {
      title: '',
      key: 'actions',
      fixed: 'right',
      width: 48,
      render: (_, r) => (
        <Tooltip title="Xem thẻ kho">
          <Button
            type="text"
            size="small"
            icon={<HistoryOutlined />}
            onClick={() => onHistory(r.product_id)}
            aria-label="Thẻ kho"
          />
        </Tooltip>
      ),
    },
  ];

  return (
    <>
      <Row gutter={[12, 12]} style={{ marginBottom: 12 }}>
        <Col xs={24} sm={8}>
          <StatCard label="Tổng số lượng tồn" value={summary?.total_quantity ?? null} />
        </Col>
        <Col xs={24} sm={8}>
          <StatCard label="Tổng giá trị tồn (giá vốn)" value={summary?.total_value ?? null} currency />
        </Col>
        <Col xs={24} sm={8}>
          <StatCard
            label="Dòng dưới mức tối thiểu"
            value={summary?.low_stock_count ?? null}
            caption={lowOnly ? 'Đang lọc tồn thấp – bấm để bỏ' : 'Bấm để lọc'}
            onClick={() => url.set({ low_stock: lowOnly ? undefined : 'true' }, { resetPage: true })}
          />
        </Col>
      </Row>
      <FilterBar
        searchPlaceholder="Tên, SKU, mã vạch..."
        filters={[
          {
            name: 'category_id',
            placeholder: 'Danh mục',
            options: (categories.data ?? []).map((c) => ({ label: c.name, value: c.id })),
          },
          { name: 'low_stock', placeholder: 'Tồn kho', options: LOW_OPTIONS, col: 3 },
        ]}
      />
      <DataTable<StockRow>
        table={table}
        columns={columns}
        rowKey={(r) => `${r.product_id}-${r.branch_id}`}
        emptyText={lowOnly ? 'Không có sản phẩm nào dưới mức tối thiểu' : 'Chưa có tồn kho'}
      />
    </>
  );
}

/* ------------------------------------------------------------------ Thẻ kho */

function TransactionsTab({ onOpenVoucher }: { onOpenVoucher: (no: string) => void }) {
  const { branchId } = useWorkingBranch();
  const url = useUrlParams();
  const products = useProductOptions(null);
  const [voucherInput, setVoucherInput] = useState(url.get('voucher_no') ?? '');
  const table = useTableQuery<StockTransaction, TransactionListParams>({
    queryKey: [inventoryApi.key, 'transactions'],
    queryFn: (p) => inventoryApi.transactions(p),
    filterKeys: ['product_id', 'type', 'voucher_no', 'from', 'to'],
    numberKeys: ['product_id'],
    defaults: { order_by: 'id', order_dir: 'DESC' },
    extraParams: { branch_id: branchId ?? undefined },
  });
  const from = url.get('from');
  const to = url.get('to');

  const columns: ColumnsType<StockTransaction> = [
    {
      title: 'Thời gian',
      dataIndex: 'created_at',
      sorter: true,
      render: (v: string) => <DateText value={v} withTime />,
    },
    {
      title: 'Mã phiếu',
      dataIndex: 'voucher_no',
      render: (v: string) => <VoucherLink no={v} onOpen={onOpenVoucher} />,
    },
    {
      title: 'Loại',
      dataIndex: 'type',
      render: (t: string) => <StatusTag status={t} map={STOCK_TX_TYPE_MAP} />,
    },
    {
      title: 'Sản phẩm',
      dataIndex: 'product_name',
      render: (v: string, r) => (
        <span>
          {v}
          <br />
          <Typography.Text type="secondary" className="code-text" style={{ fontSize: 12 }}>
            {r.product_sku}
          </Typography.Text>
        </span>
      ),
    },
    ...(branchId ? [] : [{ title: 'Chi nhánh', dataIndex: 'branch_name' }]),
    {
      title: 'Số lượng',
      dataIndex: 'quantity',
      sorter: true,
      align: 'right',
      render: (q: number, r) => <QtyText value={q} unit={r.unit} />,
    },
    {
      title: 'Tồn sau',
      dataIndex: 'balance_after',
      align: 'right',
      render: (v: number) => <b>{formatNumber(v)}</b>,
    },
    {
      title: 'Đơn giá vốn',
      dataIndex: 'unit_cost',
      align: 'right',
      render: (v: number) => <MoneyText value={v} />,
    },
    {
      title: 'Ghi chú',
      dataIndex: 'note',
      render: (v: string, r) => (
        <Typography.Text ellipsis={{ tooltip: v }} style={{ maxWidth: 240, fontSize: 12 }}>
          {r.supplier_name ? `${r.supplier_name} · ` : ''}
          {v}
        </Typography.Text>
      ),
    },
    { title: 'Người thực hiện', dataIndex: 'created_by_name' },
  ];

  return (
    <>
      <FilterBar
        searchPlaceholder="Sản phẩm, mã phiếu, ghi chú"
        colTextSearch={5}
        filters={[
          {
            name: 'product_id',
            placeholder: 'Sản phẩm',
            options: (products.data ?? []).map((p) => ({ label: `${p.sku} – ${p.name}`, value: p.id })),
            col: 5,
          },
          { name: 'type', placeholder: 'Loại biến động', options: STOCK_TX_TYPE_OPTIONS, col: 3 },
        ]}
      >
        <Col xs={24} sm={12} md={4}>
          <Input.Search
            allowClear
            placeholder="Mã phiếu"
            value={voucherInput}
            onChange={(e) => {
              setVoucherInput(e.target.value);
              if (!e.target.value) url.set({ voucher_no: undefined }, { resetPage: true });
            }}
            onSearch={(v) => url.set({ voucher_no: v.trim().toUpperCase() }, { resetPage: true })}
          />
        </Col>
        <Col xs={24} sm={12} md={6}>
          <DatePicker.RangePicker
            style={{ width: '100%' }}
            format={DATE_FORMAT}
            placeholder={['Từ ngày', 'Đến ngày']}
            value={from || to ? [from ? dayjs(from) : null, to ? dayjs(to) : null] : null}
            allowEmpty={[true, true]}
            presets={[
              { label: 'Hôm nay', value: [dayjs(), dayjs()] },
              { label: '7 ngày qua', value: [dayjs().subtract(6, 'day'), dayjs()] },
              { label: 'Tháng này', value: [dayjs().startOf('month'), dayjs()] },
            ]}
            onChange={(v) =>
              url.set(
                { from: toApiDate(v?.[0]) ?? undefined, to: toApiDate(v?.[1]) ?? undefined },
                { resetPage: true },
              )
            }
          />
        </Col>
      </FilterBar>
      <DataTable<StockTransaction>
        table={table}
        columns={columns}
        emptyText="Chưa có biến động kho"
      />
    </>
  );
}

/* ------------------------------------------------------------------ Phiếu */

interface VoucherSummary {
  voucher_no: string;
  kind: string;
  branch_name: string;
  created_at: string;
  created_by_name: string;
  lines: number;
  quantity: number;
}

const KIND_BY_TYPE: Record<string, string> = {
  IMPORT: 'IMPORT',
  TRANSFER_OUT: 'TRANSFER',
  TRANSFER_IN: 'TRANSFER',
  ADJUST: 'ADJUST',
  SALE_RETURN: 'SALE_RETURN',
  SALE: 'SALE',
};

function VouchersTab({ onOpenVoucher }: { onOpenVoucher: (no: string) => void }) {
  const { branchId } = useWorkingBranch();
  const [lookup, setLookup] = useState('');
  // Không có API danh sách phiếu → gom 100 dòng thẻ kho gần nhất theo mã phiếu (bỏ hóa đơn bán).
  const recent = useQuery({
    queryKey: [inventoryApi.key, 'transactions', 'recent-vouchers', branchId],
    queryFn: () =>
      inventoryApi.transactions({
        branch_id: branchId ?? undefined,
        limit: 100,
        order_by: 'id',
        order_dir: 'DESC',
      }),
  });

  const vouchers = useMemo(() => {
    const map = new Map<string, VoucherSummary>();
    (recent.data?.items ?? []).forEach((t) => {
      if (!t.voucher_no || t.type === 'SALE' || t.voucher_no.startsWith('HD')) return;
      const v = map.get(t.voucher_no);
      if (v) {
        if (t.type !== 'TRANSFER_IN') {
          v.lines += 1;
          v.quantity += Math.abs(t.quantity);
        }
      } else {
        map.set(t.voucher_no, {
          voucher_no: t.voucher_no,
          kind: KIND_BY_TYPE[t.type] ?? t.type,
          branch_name: t.branch_name,
          created_at: t.created_at,
          created_by_name: t.created_by_name,
          lines: t.type === 'TRANSFER_IN' ? 0 : 1,
          quantity: t.type === 'TRANSFER_IN' ? 0 : Math.abs(t.quantity),
        });
      }
    });
    return [...map.values()];
  }, [recent.data]);

  return (
    <>
      <Row gutter={[12, 12]} style={{ marginBottom: 12 }}>
        <Col xs={24} md={12}>
          <Input.Search
            size="large"
            allowClear
            enterButton={
              <span>
                <FileSearchOutlined /> Xem phiếu
              </span>
            }
            placeholder="Nhập mã phiếu (NK…, DC…, KK…, TH…, HD…)"
            value={lookup}
            onChange={(e) => setLookup(e.target.value.toUpperCase())}
            onSearch={(v) => v.trim() && onOpenVoucher(v.trim().toUpperCase())}
          />
        </Col>
      </Row>
      <Typography.Title level={5}>Phiếu gần đây</Typography.Title>
      <Table<VoucherSummary>
        size="small"
        rowKey="voucher_no"
        loading={recent.isLoading}
        dataSource={vouchers}
        pagination={{ pageSize: 10, hideOnSinglePage: true, showSizeChanger: false }}
        scroll={{ x: 'max-content' }}
        locale={{ emptyText: <Empty description="Chưa có phiếu kho" /> }}
        columns={[
          {
            title: 'Mã phiếu',
            dataIndex: 'voucher_no',
            render: (v: string) => <VoucherLink no={v} onOpen={onOpenVoucher} />,
          },
          {
            title: 'Loại',
            dataIndex: 'kind',
            render: (k: string) => <StatusTag status={k} map={VOUCHER_KIND_MAP} />,
          },
          { title: 'Chi nhánh', dataIndex: 'branch_name' },
          {
            title: 'Số dòng',
            dataIndex: 'lines',
            align: 'right',
            render: (v: number) => (v ? formatNumber(v) : '—'),
          },
          {
            title: 'Tổng SL',
            dataIndex: 'quantity',
            align: 'right',
            render: (v: number) => (v ? formatNumber(v) : '—'),
          },
          {
            title: 'Thời gian',
            dataIndex: 'created_at',
            render: (v: string) => <DateText value={v} withTime />,
          },
          { title: 'Người lập', dataIndex: 'created_by_name' },
        ]}
      />
      <Typography.Text type="secondary" style={{ fontSize: 12 }}>
        Danh sách lấy từ 100 biến động kho gần nhất. Tra cứu phiếu cũ hơn bằng ô mã phiếu hoặc tab Thẻ kho.
      </Typography.Text>
    </>
  );
}

/* ------------------------------------------------------------------ Trang */

/** `/app/inventory` – Tồn kho · Thẻ kho · Phiếu (nhập / điều chuyển / kiểm kê). */
export default function InventoryPage() {
  const { can } = usePermission();
  const { branchId } = useWorkingBranch();
  const { branches } = useBranchOptions();
  const [params, setParams] = useSearchParams();
  const tab = (params.get('tab') as TabKey | null) ?? 'stock';
  const [voucherNo, setVoucherNo] = useState<string | null>(null);
  const [form, setForm] = useState<'import' | 'transfer' | 'adjust' | null>(null);
  const branchName = branches.find((b) => b.id === branchId)?.name;

  const switchTab = (key: string, extra: Record<string, string> = {}) =>
    setParams({ tab: key, ...extra }, { replace: true });

  const onDone = (v: Voucher) => setVoucherNo(v.voucher_no);

  return (
    <Card variant="borderless" className="page-card">
      <Row justify="space-between" align="middle" gutter={[8, 12]} className="page-head">
        <Col>
          <PageTitle
            title="Kho hàng"
            subtitle={branchName ? `Tồn kho tại ${branchName}` : 'Tồn kho tất cả chi nhánh'}
          />
        </Col>
        <Col>
          <ActionBar showAdd={false} showDelete={false} showRefresh={false}>
            {can(PERM.INVENTORY_IMPORT) && (
              <Button color="orange" variant="solid" icon={<ImportOutlined />} onClick={() => setForm('import')}>
                Phiếu nhập
              </Button>
            )}
            {can(PERM.INVENTORY_TRANSFER) && (
              <Button icon={<SwapOutlined />} onClick={() => setForm('transfer')}>
                Điều chuyển
              </Button>
            )}
            {can(PERM.INVENTORY_ADJUST) && (
              <Button icon={<ReconciliationOutlined />} onClick={() => setForm('adjust')}>
                Kiểm kê
              </Button>
            )}
          </ActionBar>
        </Col>
      </Row>

      <Tabs
        activeKey={tab}
        onChange={(k) => switchTab(k)}
        destroyOnHidden
        items={[
          {
            key: 'stock',
            label: 'Tồn kho',
            children: (
              <StockTab
                onHistory={(id) => switchTab('transactions', { product_id: String(id) })}
              />
            ),
          },
          {
            key: 'transactions',
            label: 'Thẻ kho',
            children: <TransactionsTab onOpenVoucher={setVoucherNo} />,
          },
          {
            key: 'vouchers',
            label: (
              <span>
                <ExportOutlined /> Phiếu
              </span>
            ),
            children: <VouchersTab onOpenVoucher={setVoucherNo} />,
          },
        ]}
      />

      <ImportModal open={form === 'import'} onClose={() => setForm(null)} onDone={onDone} />
      <TransferModal open={form === 'transfer'} onClose={() => setForm(null)} onDone={onDone} />
      <AdjustmentModal open={form === 'adjust'} onClose={() => setForm(null)} onDone={onDone} />
      <VoucherModal voucherNo={voucherNo} onClose={() => setVoucherNo(null)} />
    </Card>
  );
}
