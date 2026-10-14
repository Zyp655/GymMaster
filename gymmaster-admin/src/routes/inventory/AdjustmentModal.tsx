import { useEffect, useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  Alert,
  App,
  Button,
  Checkbox,
  Col,
  Input,
  InputNumber,
  Modal,
  Row,
  Space,
  Table,
  Typography,
} from 'antd';
import { SearchOutlined } from '@ant-design/icons';
import inventoryApi from '@/api/inventory';
import type { StockRow } from '@/api/inventory';
import BranchSelect from '@/components/BranchSelect';
import MoneyText from '@/components/Money/MoneyText';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { getErrorMessage, showError } from '@/utils/errors';
import { formatMoney, formatNumber, normalizeText } from '@/utils/format';
import type { VoucherFormModalProps } from './ImportModal';
import QtyText from './QtyText';
import { useAdjustStock } from './queries';

/**
 * Phiếu kiểm kê: tải tồn hiện tại của chi nhánh → nhập số đếm thực tế → xem chênh lệch → lưu.
 * Chỉ gửi các dòng có chênh lệch (backend bỏ qua dòng khớp).
 */
export default function AdjustmentModal({ open, onClose, onDone }: VoucherFormModalProps) {
  const { message, modal } = App.useApp();
  const { branchId: working } = useWorkingBranch();
  const save = useAdjustStock();
  const [branchId, setBranchId] = useState<number | null>(null);
  const [actual, setActual] = useState<Record<number, number | null>>({});
  const [note, setNote] = useState('');
  const [keyword, setKeyword] = useState('');
  const [onlyDiff, setOnlyDiff] = useState(false);

  useEffect(() => {
    if (!open) return;
    setBranchId(working ?? null);
    setActual({});
    setNote('');
    setKeyword('');
    setOnlyDiff(false);
  }, [open, working]);

  const stock = useQuery({
    queryKey: [inventoryApi.key, 'stock-all', branchId],
    queryFn: () => inventoryApi.stockAll({ branch_id: branchId!, order_by: 'name', order_dir: 'ASC' }),
    enabled: open && !!branchId,
    staleTime: 0,
  });
  const rows = useMemo(() => stock.data ?? [], [stock.data]);

  const diffOf = (r: StockRow) => {
    const a = actual[r.product_id];
    return a === null || a === undefined ? 0 : a - r.quantity;
  };
  const changed = rows.filter((r) => diffOf(r) !== 0);
  const diffValue = changed.reduce((s, r) => s + diffOf(r) * r.cost_price, 0);

  const visible = rows.filter((r) => {
    if (onlyDiff && diffOf(r) === 0) return false;
    const kw = normalizeText(keyword.trim());
    return (
      !kw ||
      normalizeText(r.product_name).includes(kw) ||
      normalizeText(r.sku).includes(kw) ||
      r.barcode.includes(keyword.trim())
    );
  });

  const submit = () => {
    if (!branchId || changed.length === 0) return;
    modal.confirm({
      title: `Lưu phiếu kiểm kê với ${changed.length} dòng chênh lệch?`,
      width: 520,
      content: (
        <div>
          <ul style={{ paddingLeft: 18, maxHeight: 240, overflow: 'auto' }}>
            {changed.map((r) => (
              <li key={r.product_id}>
                {r.product_name}: sổ sách {formatNumber(r.quantity)} → thực tế{' '}
                {formatNumber(actual[r.product_id] ?? 0)} (
                <QtyText value={diffOf(r)} />)
              </li>
            ))}
          </ul>
          Giá trị chênh lệch: <b>{formatMoney(Math.round(diffValue))}</b>. Tồn kho sẽ được điều chỉnh
          theo số thực tế.
        </div>
      ),
      okText: 'Lưu phiếu',
      cancelText: 'Xem lại',
      onOk: () =>
        save
          .mutateAsync({
            branch_id: branchId,
            note: note.trim(),
            items: changed.map((r) => ({
              product_id: r.product_id,
              actual_quantity: actual[r.product_id] ?? 0,
            })),
          })
          .then((res) => {
            message.success(`Đã lưu phiếu kiểm kê ${res.voucher_no}`);
            onClose();
            onDone(res);
          })
          .catch((e: unknown) => showError(e, 'Không thể lưu phiếu kiểm kê')),
    });
  };

  return (
    <Modal
      open={open}
      title="Phiếu kiểm kê"
      onCancel={onClose}
      width={1000}
      footer={[
        <Button key="cancel" onClick={onClose}>
          Hủy
        </Button>,
        <Button
          key="ok"
          type="primary"
          loading={save.isPending}
          disabled={changed.length === 0}
          onClick={submit}
        >
          Lưu phiếu kiểm kê ({changed.length} dòng lệch)
        </Button>,
      ]}
      destroyOnHidden
    >
      <Row gutter={[12, 12]} style={{ margin: '12px 0' }}>
        <Col xs={24} sm={8}>
          <BranchSelect
            activeOnly
            value={branchId}
            placeholder="Chọn chi nhánh kiểm kê"
            onChange={(v) => {
              setBranchId(typeof v === 'number' ? v : null);
              setActual({});
            }}
          />
        </Col>
        <Col xs={24} sm={8}>
          <Input
            allowClear
            prefix={<SearchOutlined />}
            placeholder="Tìm sản phẩm"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
          />
        </Col>
        <Col xs={24} sm={8}>
          <Input
            maxLength={255}
            placeholder="Ghi chú (VD: Kiểm kê cuối tháng)"
            value={note}
            onChange={(e) => setNote(e.target.value)}
          />
        </Col>
      </Row>

      {!branchId ? (
        <Alert type="info" showIcon message="Chọn chi nhánh để tải tồn kho hiện tại" />
      ) : stock.isError ? (
        <Alert
          type="error"
          showIcon
          message={getErrorMessage(stock.error, 'Không tải được tồn kho')}
          action={<Button onClick={() => stock.refetch()}>Thử lại</Button>}
        />
      ) : (
        <>
          <Space wrap style={{ marginBottom: 8 }}>
            <Button
              size="small"
              onClick={() =>
                setActual(Object.fromEntries(rows.map((r) => [r.product_id, r.quantity])))
              }
            >
              Điền thực tế = sổ sách
            </Button>
            <Button size="small" onClick={() => setActual({})}>
              Xóa số đã nhập
            </Button>
            <Checkbox checked={onlyDiff} onChange={(e) => setOnlyDiff(e.target.checked)}>
              Chỉ dòng chênh lệch
            </Checkbox>
            <Typography.Text type="secondary">
              Ô trống = chưa đếm (không điều chỉnh).
            </Typography.Text>
          </Space>
          <Table<StockRow>
            size="small"
            rowKey="product_id"
            loading={stock.isLoading}
            dataSource={visible}
            pagination={false}
            scroll={{ y: 420, x: 'max-content' }}
            locale={{ emptyText: 'Không có sản phẩm' }}
            rowClassName={(r) => (diffOf(r) !== 0 ? 'adjust-row--diff' : '')}
            columns={[
              {
                title: 'Sản phẩm',
                key: 'product',
                render: (_, r) => (
                  <span>
                    {r.product_name}
                    <br />
                    <Typography.Text type="secondary" className="code-text" style={{ fontSize: 12 }}>
                      {r.sku}
                    </Typography.Text>
                  </span>
                ),
              },
              {
                title: 'Sổ sách',
                dataIndex: 'quantity',
                align: 'right',
                width: 100,
                render: (q: number, r) => `${formatNumber(q)} ${r.unit}`,
              },
              {
                title: 'Thực tế',
                key: 'actual',
                width: 130,
                render: (_, r) => (
                  <InputNumber
                    min={0}
                    precision={0}
                    value={actual[r.product_id] ?? null}
                    placeholder={String(r.quantity)}
                    onChange={(v) => setActual((a) => ({ ...a, [r.product_id]: v }))}
                    style={{ width: '100%' }}
                    aria-label={`Thực tế ${r.product_name}`}
                  />
                ),
              },
              {
                title: 'Chênh lệch',
                key: 'diff',
                align: 'right',
                width: 110,
                render: (_, r) => <QtyText value={diffOf(r)} />,
              },
              {
                title: 'Giá trị lệch',
                key: 'value',
                align: 'right',
                width: 130,
                render: (_, r) =>
                  diffOf(r) ? <MoneyText value={Math.round(diffOf(r) * r.cost_price)} colored /> : '—',
              },
            ]}
          />
          <div style={{ textAlign: 'right', marginTop: 8 }}>
            {changed.length} dòng chênh lệch · Giá trị chênh lệch{' '}
            <MoneyText value={Math.round(diffValue)} colored strong />
          </div>
        </>
      )}
    </Modal>
  );
}
