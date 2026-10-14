import { useEffect } from 'react';
import { useQuery } from '@tanstack/react-query';
import { App, Button, Col, Form, Input, InputNumber, Modal, Row, Select, Typography } from 'antd';
import { DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import type { Voucher } from '@/api/inventory';
import { supplierApi } from '@/api/product';
import BranchSelect from '@/components/BranchSelect';
import MoneyInput from '@/components/Money/MoneyInput';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { setFormError, showError } from '@/utils/errors';
import { formatMoney, formatNumber, normalizeText } from '@/utils/format';
import { useImportStock, useProductOptions } from './queries';

interface Line {
  product_id?: number;
  quantity?: number;
  unit_cost?: number;
}
interface FormValues {
  branch_id?: number;
  supplier_id?: number | null;
  note?: string;
  items: Line[];
}

export interface VoucherFormModalProps {
  open: boolean;
  onClose: () => void;
  /** Lưu thành công → mở chi tiết phiếu. */
  onDone: (v: Voucher) => void;
}

const filterOption = (input: string, option?: { label?: unknown }) =>
  normalizeText(String(option?.label ?? '')).includes(normalizeText(input));

/** Phiếu nhập kho: NCC + các dòng sản phẩm × SL × đơn giá nhập (giá vốn bình quân tự tính). */
export default function ImportModal({ open, onClose, onDone }: VoucherFormModalProps) {
  const [form] = Form.useForm<FormValues>();
  const { message } = App.useApp();
  const { branchId } = useWorkingBranch();
  const save = useImportStock();
  const branch = Form.useWatch('branch_id', form);
  const items = (Form.useWatch('items', form) as Line[] | undefined) ?? [];

  const products = useProductOptions(branch, open);
  const suppliers = useQuery({
    queryKey: [supplierApi.key, 'options'],
    queryFn: () => supplierApi.options(),
    enabled: open,
  });

  useEffect(() => {
    if (!open) return;
    form.resetFields();
    form.setFieldsValue({ branch_id: branchId ?? undefined, items: [{}] });
  }, [open, branchId, form]);

  const byId = new Map((products.data ?? []).map((p) => [p.id, p]));
  const total = items.reduce((s, l) => s + (l?.quantity ?? 0) * (l?.unit_cost ?? 0), 0);
  const totalQty = items.reduce((s, l) => s + (l?.quantity ?? 0), 0);

  const onFinish = (v: FormValues) =>
    save.mutate(
      {
        branch_id: v.branch_id!,
        supplier_id: v.supplier_id ?? null,
        note: v.note?.trim() ?? '',
        items: v.items.map((l) => ({
          product_id: l.product_id!,
          quantity: l.quantity!,
          unit_cost: l.unit_cost ?? 0,
        })),
      },
      {
        onSuccess: (res) => {
          message.success(`Đã lưu phiếu nhập ${res.voucher_no}`);
          onClose();
          onDone(res);
        },
        onError: (e) => {
          if (!setFormError(form, e, { branch_id: /chi nhánh/i, supplier_id: /nhà cung cấp/i }))
            showError(e, 'Không thể lưu phiếu nhập');
        },
      },
    );

  return (
    <Modal
      open={open}
      title="Phiếu nhập kho"
      okText="Lưu phiếu nhập"
      cancelText="Hủy"
      onOk={() => form.submit()}
      onCancel={onClose}
      confirmLoading={save.isPending}
      width={980}
      forceRender
    >
      <Form<FormValues> form={form} layout="vertical" onFinish={onFinish} style={{ marginTop: 12 }}>
        <Row gutter={16}>
          <Col xs={24} sm={8}>
            <Form.Item
              name="branch_id"
              label="Chi nhánh nhập"
              rules={[{ required: true, message: 'Vui lòng chọn chi nhánh nhập' }]}
            >
              <BranchSelect activeOnly />
            </Form.Item>
          </Col>
          <Col xs={24} sm={8}>
            <Form.Item name="supplier_id" label="Nhà cung cấp">
              <Select
                allowClear
                showSearch
                placeholder="Không chọn"
                loading={suppliers.isLoading}
                filterOption={filterOption}
                options={(suppliers.data ?? []).map((s) => ({ label: s.name, value: s.id }))}
              />
            </Form.Item>
          </Col>
          <Col xs={24} sm={8}>
            <Form.Item
              name="note"
              label="Ghi chú"
              rules={[{ max: 255, message: 'Ghi chú tối đa 255 ký tự' }]}
            >
              <Input maxLength={255} placeholder="VD: Nhập hàng tháng 10" />
            </Form.Item>
          </Col>
        </Row>

        <Form.List
          name="items"
          rules={[
            {
              validator: (_, v: Line[] | undefined) =>
                v && v.length > 0
                  ? Promise.resolve()
                  : Promise.reject(new Error('Phiếu nhập phải có ít nhất một dòng')),
            },
          ]}
        >
          {(fields, { add, remove }, { errors }) => (
            <>
              <div className="voucher-lines">
                {fields.map((field, idx) => {
                  const line = items[idx] ?? {};
                  const p = line.product_id ? byId.get(line.product_id) : undefined;
                  return (
                    <Row key={field.key} gutter={8} align="top" className="voucher-line">
                      <Col xs={24} md={10}>
                        <Form.Item
                          name={[field.name, 'product_id']}
                          label={idx === 0 ? 'Sản phẩm' : undefined}
                          rules={[{ required: true, message: 'Chọn sản phẩm' }]}
                          extra={
                            p
                              ? `Giá vốn hiện tại ${formatMoney(p.cost_price)} · tổng tồn ${formatNumber(p.total_stock)} ${p.unit}`
                              : undefined
                          }
                        >
                          <Select
                            showSearch
                            placeholder="Tìm theo tên, SKU"
                            loading={products.isLoading}
                            filterOption={filterOption}
                            options={(products.data ?? []).map((x) => ({
                              label: `${x.sku} – ${x.name}`,
                              value: x.id,
                            }))}
                            onChange={(id: number) => {
                              const prod = byId.get(id);
                              if (prod && prod.cost_price > 0)
                                form.setFieldValue(
                                  ['items', field.name, 'unit_cost'],
                                  Math.round(prod.cost_price),
                                );
                            }}
                          />
                        </Form.Item>
                      </Col>
                      <Col xs={8} md={4}>
                        <Form.Item
                          name={[field.name, 'quantity']}
                          label={idx === 0 ? 'Số lượng' : undefined}
                          rules={[{ required: true, message: 'Nhập SL' }]}
                        >
                          <InputNumber
                            min={1}
                            precision={0}
                            style={{ width: '100%' }}
                            suffix={p?.unit}
                          />
                        </Form.Item>
                      </Col>
                      <Col xs={10} md={5}>
                        <Form.Item
                          name={[field.name, 'unit_cost']}
                          label={idx === 0 ? 'Đơn giá nhập' : undefined}
                          rules={[{ required: true, message: 'Nhập đơn giá' }]}
                        >
                          <MoneyInput step={500} />
                        </Form.Item>
                      </Col>
                      <Col xs={6} md={5}>
                        <Form.Item label={idx === 0 ? 'Thành tiền' : undefined}>
                          <div className="voucher-line__amount">
                            <span>
                              {formatMoney((line.quantity ?? 0) * (line.unit_cost ?? 0))}
                            </span>
                            {fields.length > 1 && (
                              <Button
                                type="text"
                                danger
                                size="small"
                                icon={<DeleteOutlined />}
                                onClick={() => remove(field.name)}
                                aria-label="Bỏ dòng"
                              />
                            )}
                          </div>
                        </Form.Item>
                      </Col>
                    </Row>
                  );
                })}
              </div>
              <Form.ErrorList errors={errors} />
              <Row justify="space-between" align="middle" gutter={[8, 8]}>
                <Col>
                  <Button icon={<PlusOutlined />} onClick={() => add({})}>
                    Thêm dòng
                  </Button>
                </Col>
                <Col>
                  <Typography.Text>
                    {fields.length} dòng · {formatNumber(totalQty)} sản phẩm · Tổng tiền{' '}
                    <b style={{ fontSize: 16 }}>{formatMoney(total)}</b>
                  </Typography.Text>
                </Col>
              </Row>
            </>
          )}
        </Form.List>
      </Form>
    </Modal>
  );
}
