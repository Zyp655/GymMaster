import { useEffect } from 'react';
import { App, Button, Col, Form, Input, InputNumber, Modal, Row, Select, Typography } from 'antd';
import { ArrowRightOutlined, DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import BranchSelect from '@/components/BranchSelect';
import { useBranchOptions } from '@/hooks/useBranchOptions';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { setFormError, showError } from '@/utils/errors';
import { formatNumber, normalizeText } from '@/utils/format';
import type { VoucherFormModalProps } from './ImportModal';
import { useProductOptions, useTransferStock } from './queries';

interface Line {
  product_id?: number;
  quantity?: number;
}
interface FormValues {
  from_branch_id?: number;
  to_branch_id?: number;
  note?: string;
  items: Line[];
}

const filterOption = (input: string, option?: { label?: unknown }) =>
  normalizeText(String(option?.label ?? '')).includes(normalizeText(input));

/** Phiếu điều chuyển: từ chi nhánh (trong phạm vi) → chi nhánh nhận; SL ≤ tồn nguồn. */
export default function TransferModal({ open, onClose, onDone }: VoucherFormModalProps) {
  const [form] = Form.useForm<FormValues>();
  const { message } = App.useApp();
  const { branchId } = useWorkingBranch();
  const { branches } = useBranchOptions();
  const save = useTransferStock();
  const from = Form.useWatch('from_branch_id', form);
  const items = (Form.useWatch('items', form) as Line[] | undefined) ?? [];
  const products = useProductOptions(from, open && !!from);
  const byId = new Map((products.data ?? []).map((p) => [p.id, p]));

  useEffect(() => {
    if (!open) return;
    form.resetFields();
    form.setFieldsValue({ from_branch_id: branchId ?? undefined, items: [{}] });
  }, [open, branchId, form]);

  const onFinish = (v: FormValues) =>
    save.mutate(
      {
        from_branch_id: v.from_branch_id!,
        to_branch_id: v.to_branch_id!,
        note: v.note?.trim() ?? '',
        items: v.items.map((l) => ({ product_id: l.product_id!, quantity: l.quantity! })),
      },
      {
        onSuccess: (res) => {
          message.success(`Đã lưu phiếu điều chuyển ${res.voucher_no}`);
          onClose();
          onDone(res);
        },
        onError: (e) => {
          if (
            !setFormError(form, e, {
              to_branch_id: /chi nhánh nhận/i,
              from_branch_id: /chi nhánh xuất/i,
            })
          )
            showError(e, 'Không thể lưu phiếu điều chuyển');
        },
      },
    );

  // Chi nhánh nhận: mọi chi nhánh đang hoạt động (có thể ngoài phạm vi) – danh sách theo user.
  const toOptions = branches
    .filter((b) => b.status !== 'INACTIVE' && b.id !== from)
    .map((b) => ({ label: b.name, value: b.id }));

  return (
    <Modal
      open={open}
      title="Phiếu điều chuyển kho"
      okText="Lưu phiếu điều chuyển"
      cancelText="Hủy"
      onOk={() => form.submit()}
      onCancel={onClose}
      confirmLoading={save.isPending}
      width={820}
      forceRender
    >
      <Form<FormValues> form={form} layout="vertical" onFinish={onFinish} style={{ marginTop: 12 }}>
        <Row gutter={16} align="middle">
          <Col xs={24} sm={11}>
            <Form.Item
              name="from_branch_id"
              label="Từ chi nhánh (xuất)"
              rules={[{ required: true, message: 'Vui lòng chọn chi nhánh xuất' }]}
            >
              <BranchSelect
                activeOnly
                onChange={() => form.setFieldValue('items', [{}])}
              />
            </Form.Item>
          </Col>
          <Col xs={0} sm={2} style={{ textAlign: 'center' }}>
            <ArrowRightOutlined />
          </Col>
          <Col xs={24} sm={11}>
            <Form.Item
              name="to_branch_id"
              label="Đến chi nhánh (nhận)"
              dependencies={['from_branch_id']}
              rules={[
                { required: true, message: 'Vui lòng chọn chi nhánh nhận' },
                ({ getFieldValue }) => ({
                  validator: (_, v?: number) =>
                    v && v === getFieldValue('from_branch_id')
                      ? Promise.reject(new Error('Chi nhánh nhận phải khác chi nhánh xuất'))
                      : Promise.resolve(),
                }),
              ]}
            >
              <Select placeholder="Chọn chi nhánh nhận" options={toOptions} />
            </Form.Item>
          </Col>
          <Col xs={24}>
            <Form.Item
              name="note"
              label="Ghi chú"
              rules={[{ max: 255, message: 'Ghi chú tối đa 255 ký tự' }]}
            >
              <Input maxLength={255} placeholder="VD: Bổ sung hàng cho chi nhánh" />
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
                  : Promise.reject(new Error('Phiếu điều chuyển phải có ít nhất một dòng')),
            },
          ]}
        >
          {(fields, { add, remove }, { errors }) => (
            <>
              {fields.map((field, idx) => {
                const line = items[idx] ?? {};
                const p = line.product_id ? byId.get(line.product_id) : undefined;
                const stock = p?.stock ?? 0;
                return (
                  <Row key={field.key} gutter={8} align="top">
                    <Col xs={24} sm={15}>
                      <Form.Item
                        name={[field.name, 'product_id']}
                        label={idx === 0 ? 'Sản phẩm' : undefined}
                        rules={[{ required: true, message: 'Chọn sản phẩm' }]}
                        extra={p ? `Tồn tại chi nhánh xuất: ${formatNumber(stock)} ${p.unit}` : undefined}
                      >
                        <Select
                          showSearch
                          disabled={!from}
                          placeholder={from ? 'Tìm theo tên, SKU' : 'Chọn chi nhánh xuất trước'}
                          loading={products.isFetching}
                          filterOption={filterOption}
                          options={(products.data ?? []).map((x) => ({
                            label: `${x.sku} – ${x.name} (tồn ${formatNumber(x.stock ?? 0)})`,
                            value: x.id,
                            disabled: (x.stock ?? 0) <= 0,
                          }))}
                        />
                      </Form.Item>
                    </Col>
                    <Col xs={18} sm={7}>
                      <Form.Item
                        name={[field.name, 'quantity']}
                        label={idx === 0 ? 'Số lượng' : undefined}
                        dependencies={[['items', field.name, 'product_id']]}
                        rules={[
                          { required: true, message: 'Nhập SL' },
                          {
                            validator: (_, q?: number) =>
                              q && p && q > stock
                                ? Promise.reject(
                                    new Error(`Vượt tồn nguồn (còn ${formatNumber(stock)})`),
                                  )
                                : Promise.resolve(),
                          },
                        ]}
                      >
                        <InputNumber
                          min={1}
                          max={p ? Math.max(1, stock) : undefined}
                          precision={0}
                          style={{ width: '100%' }}
                          suffix={p?.unit}
                        />
                      </Form.Item>
                    </Col>
                    <Col xs={6} sm={2}>
                      <Form.Item label={idx === 0 ? ' ' : undefined}>
                        {fields.length > 1 && (
                          <Button
                            type="text"
                            danger
                            icon={<DeleteOutlined />}
                            onClick={() => remove(field.name)}
                            aria-label="Bỏ dòng"
                          />
                        )}
                      </Form.Item>
                    </Col>
                  </Row>
                );
              })}
              <Form.ErrorList errors={errors} />
              <Row justify="space-between" align="middle">
                <Col>
                  <Button icon={<PlusOutlined />} onClick={() => add({})} disabled={!from}>
                    Thêm dòng
                  </Button>
                </Col>
                <Col>
                  <Typography.Text>
                    Tổng {formatNumber(items.reduce((s, l) => s + (l?.quantity ?? 0), 0))} sản phẩm
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
