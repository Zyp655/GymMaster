import { useEffect } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  App,
  Col,
  Form,
  Input,
  InputNumber,
  Modal,
  Row,
  Segmented,
  Select,
  Table,
  Typography,
} from 'antd';
import productApi, { productCategoryApi } from '@/api/product';
import type { Product, ProductInput } from '@/api/product';
import ImageUpload from '@/components/ImageUpload';
import MoneyInput from '@/components/Money/MoneyInput';
import MoneyText from '@/components/Money/MoneyText';
import { PERM } from '@/constants/permissions';
import { usePermission } from '@/hooks/usePermission';
import { setFormError, showError } from '@/utils/errors';
import { formatNumber } from '@/utils/format';
import { useSaveProduct } from './queries';

interface FormValues extends Omit<ProductInput, 'member_price'> {
  member_price: number | null | undefined;
}

export interface ProductFormProps {
  open: boolean;
  record: Product | null;
  onClose: () => void;
}

const DEFAULTS: Partial<FormValues> = {
  unit: 'cái',
  cost_price: 0,
  min_stock: 0,
  status: 'ACTIVE',
  image_url: '',
  barcode: '',
  member_price: null,
  category_id: null,
};

/** Modal thêm/sửa sản phẩm – validate giống `api/product/v1/product.go`. */
export default function ProductForm({ open, record, onClose }: ProductFormProps) {
  const [form] = Form.useForm<FormValues>();
  const { message } = App.useApp();
  const { can } = usePermission();
  const save = useSaveProduct();
  const isEdit = !!record;
  const readOnly = !can(PERM.PRODUCT_MANAGE);

  const categories = useQuery({
    queryKey: [productCategoryApi.key, 'list'],
    queryFn: () => productCategoryApi.list(),
    enabled: open,
  });

  // Tồn theo chi nhánh (chỉ khi sửa).
  const detail = useQuery({
    queryKey: [productApi.key, 'detail', record?.id],
    queryFn: () => productApi.get(record!.id),
    enabled: open && !!record,
  });

  useEffect(() => {
    if (!open) return;
    form.resetFields();
    form.setFieldsValue(record ? { ...record } : DEFAULTS);
  }, [open, record, form]);

  const onFinish = (v: FormValues) => {
    const data: ProductInput = {
      sku: v.sku.trim().toUpperCase(),
      barcode: v.barcode?.trim() ?? '',
      name: v.name.trim(),
      category_id: v.category_id ?? null,
      unit: v.unit?.trim() || 'cái',
      // Giá vốn do phiếu nhập tính (bình quân gia quyền) – khi sửa backend BỎ QUA trường này (không ghi đè).
      cost_price: isEdit ? (detail.data?.cost_price ?? record.cost_price) : (v.cost_price ?? 0),
      sell_price: v.sell_price,
      member_price: v.member_price === undefined || v.member_price === null ? null : v.member_price,
      image_url: v.image_url ?? '',
      min_stock: v.min_stock ?? 0,
      status: v.status,
    };
    save.mutate(
      { id: record?.id, data },
      {
        onSuccess: () => {
          message.success(isEdit ? 'Đã cập nhật sản phẩm' : 'Đã thêm sản phẩm');
          onClose();
        },
        onError: (error) => {
          const mapped = setFormError(form, error, {
            sku: /sku/i,
            barcode: /mã vạch/i,
            name: /tên/i,
            member_price: /giá hội viên/i,
            sell_price: /giá bán/i,
            category_id: /danh mục/i,
            unit: /đơn vị/i,
            min_stock: /tồn tối thiểu/i,
          });
          if (!mapped) showError(error, 'Không thể lưu sản phẩm');
        },
      },
    );
  };

  return (
    <Modal
      open={open}
      title={isEdit ? `Sản phẩm ${record?.sku}` : 'Thêm sản phẩm'}
      okText={isEdit ? 'Lưu thay đổi' : 'Thêm sản phẩm'}
      cancelText={readOnly ? 'Đóng' : 'Hủy'}
      okButtonProps={{ style: readOnly ? { display: 'none' } : undefined }}
      onOk={() => form.submit()}
      onCancel={onClose}
      confirmLoading={save.isPending}
      width={760}
      forceRender
    >
      <Form<FormValues>
        form={form}
        layout="vertical"
        disabled={readOnly}
        onFinish={onFinish}
        style={{ marginTop: 12 }}
      >
        <Row gutter={16}>
          <Col xs={24} sm={6}>
            <Form.Item name="image_url" label="Ảnh">
              <ImageUpload />
            </Form.Item>
          </Col>
          <Col xs={24} sm={18}>
            <Row gutter={16}>
              <Col xs={24} sm={10}>
                <Form.Item
                  name="sku"
                  label="Mã SKU"
                  tooltip="2–30 ký tự chữ, số, '.', '-', '_'. Lưu IN HOA, không trùng."
                  normalize={(v: string) => v?.toUpperCase()}
                  rules={[
                    { required: true, message: 'Vui lòng nhập mã SKU' },
                    {
                      pattern: /^[A-Za-z0-9_.-]{2,30}$/,
                      message: "SKU gồm 2-30 ký tự chữ, số, '.', '-', '_'",
                    },
                  ]}
                >
                  <Input placeholder="VD: AQUA-500" maxLength={30} className="code-text" />
                </Form.Item>
              </Col>
              <Col xs={24} sm={14}>
                <Form.Item
                  name="barcode"
                  label="Mã vạch"
                  rules={[
                    { max: 50, message: 'Mã vạch tối đa 50 ký tự' },
                    { pattern: /^[0-9A-Za-z-]*$/, message: "Mã vạch chỉ gồm chữ, số, '-'" },
                  ]}
                >
                  <Input placeholder="Quét hoặc nhập mã vạch" maxLength={50} allowClear />
                </Form.Item>
              </Col>
              <Col xs={24}>
                <Form.Item
                  name="name"
                  label="Tên sản phẩm"
                  rules={[
                    { required: true, whitespace: true, message: 'Vui lòng nhập tên sản phẩm' },
                    { max: 150, message: 'Tên tối đa 150 ký tự' },
                  ]}
                >
                  <Input placeholder="VD: Nước suối Aquafina 500ml" maxLength={150} />
                </Form.Item>
              </Col>
            </Row>
          </Col>

          <Col xs={24} sm={12}>
            <Form.Item name="category_id" label="Danh mục">
              <Select
                allowClear
                placeholder="Chưa phân loại"
                loading={categories.isLoading}
                options={(categories.data ?? []).map((c) => ({ label: c.name, value: c.id }))}
              />
            </Form.Item>
          </Col>
          <Col xs={12} sm={6}>
            <Form.Item
              name="unit"
              label="Đơn vị tính"
              rules={[{ max: 20, message: 'Đơn vị tính tối đa 20 ký tự' }]}
            >
              <Input placeholder="cái, chai, hộp..." maxLength={20} />
            </Form.Item>
          </Col>
          <Col xs={12} sm={6}>
            <Form.Item
              name="min_stock"
              label="Tồn tối thiểu"
              tooltip="Dưới mức này sẽ cảnh báo tồn thấp (0 = không cảnh báo)"
              rules={[{ type: 'number', min: 0, message: 'Tồn tối thiểu không được âm' }]}
            >
              <InputNumber min={0} precision={0} style={{ width: '100%' }} />
            </Form.Item>
          </Col>

          <Col xs={24} sm={8}>
            {isEdit ? (
              <Form.Item
                label="Giá vốn"
                tooltip="Giá vốn bình quân gia quyền do hệ thống tự tính khi nhập hàng"
              >
                <Input
                  readOnly
                  disabled
                  value={formatNumber(detail.data?.cost_price ?? record?.cost_price)}
                  suffix="₫"
                />
              </Form.Item>
            ) : (
              <Form.Item
                name="cost_price"
                label="Giá vốn"
                tooltip="Thường để 0 – phiếu nhập sẽ tính giá vốn bình quân"
              >
                <MoneyInput step={500} />
              </Form.Item>
            )}
          </Col>
          <Col xs={24} sm={8}>
            <Form.Item
              name="sell_price"
              label="Giá bán"
              rules={[{ required: true, message: 'Vui lòng nhập giá bán' }]}
            >
              <MoneyInput />
            </Form.Item>
          </Col>
          <Col xs={24} sm={8}>
            <Form.Item
              name="member_price"
              label="Giá hội viên"
              tooltip="Để trống = hội viên mua theo giá bán"
              dependencies={['sell_price']}
              rules={[
                ({ getFieldValue }) => ({
                  validator: (_, v: number | null | undefined) => {
                    const sell = getFieldValue('sell_price') as number | undefined;
                    if (v === null || v === undefined || sell === undefined)
                      return Promise.resolve();
                    return v <= sell
                      ? Promise.resolve()
                      : Promise.reject(new Error('Giá hội viên không được lớn hơn giá bán'));
                  },
                }),
              ]}
            >
              <MoneyInput placeholder="Theo giá bán" />
            </Form.Item>
          </Col>

          <Col xs={24} sm={12}>
            <Form.Item
              name="status"
              label="Trạng thái"
              rules={[{ required: true, message: 'Vui lòng chọn trạng thái' }]}
            >
              <Segmented
                block
                options={[
                  { label: 'Đang kinh doanh', value: 'ACTIVE' },
                  { label: 'Ngừng kinh doanh', value: 'INACTIVE' },
                ]}
              />
            </Form.Item>
          </Col>
        </Row>
      </Form>

      {isEdit && (
        <>
          <Typography.Title level={5} style={{ marginTop: 0 }}>
            Tồn theo chi nhánh
          </Typography.Title>
          <Table
            size="small"
            rowKey="branch_id"
            pagination={false}
            loading={detail.isLoading}
            dataSource={detail.data?.stocks ?? []}
            locale={{ emptyText: 'Chưa phát sinh tồn kho' }}
            columns={[
              { title: 'Chi nhánh', dataIndex: 'branch_name' },
              {
                title: 'Tồn',
                dataIndex: 'quantity',
                align: 'right',
                render: (v: number, r) => (
                  <span style={{ color: r.low_stock ? '#ea580c' : undefined }}>
                    {formatNumber(v)} {record?.unit}
                    {r.low_stock && ' (tồn thấp)'}
                  </span>
                ),
              },
              {
                title: 'Giá trị tồn',
                key: 'value',
                align: 'right',
                render: (_, r) => (
                  <MoneyText
                    value={Math.round(
                      r.quantity * (detail.data?.cost_price ?? record?.cost_price ?? 0),
                    )}
                  />
                ),
              },
            ]}
          />
        </>
      )}
    </Modal>
  );
}
