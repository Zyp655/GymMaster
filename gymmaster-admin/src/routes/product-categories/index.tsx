import { useEffect, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  App,
  Button,
  Card,
  Col,
  Form,
  Input,
  InputNumber,
  Modal,
  Result,
  Row,
  Space,
  Table,
  Tooltip,
} from 'antd';
import { DeleteOutlined, EditOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { productCategoryApi } from '@/api/product';
import type { ProductCategory, ProductCategoryInput } from '@/api/product';
import ActionBar from '@/components/ActionBar';
import ConfirmButton from '@/components/ConfirmButton';
import DateText from '@/components/DateText';
import FilterBar from '@/components/FilterBar';
import PageTitle from '@/components/PageTitle';
import { PERM } from '@/constants/permissions';
import { usePermission } from '@/hooks/usePermission';
import { useUrlParams } from '@/hooks/useUrlParams';
import { getErrorMessage, setFormError, showError } from '@/utils/errors';
import { formatNumber } from '@/utils/format';
import { useDeleteCategories, useSaveCategory } from '@/routes/products/queries';

function CategoryForm({
  open,
  record,
  onClose,
}: {
  open: boolean;
  record: ProductCategory | null;
  onClose: () => void;
}) {
  const [form] = Form.useForm<ProductCategoryInput>();
  const { message } = App.useApp();
  const save = useSaveCategory();
  const isEdit = !!record;

  useEffect(() => {
    if (!open) return;
    form.resetFields();
    form.setFieldsValue(record ?? { name: '', sort_order: 0 });
  }, [open, record, form]);

  const onFinish = (v: ProductCategoryInput) =>
    save.mutate(
      { id: record?.id, data: { name: v.name.trim(), sort_order: v.sort_order ?? 0 } },
      {
        onSuccess: () => {
          message.success(isEdit ? 'Đã cập nhật danh mục' : 'Đã thêm danh mục');
          onClose();
        },
        onError: (e) => {
          if (!setFormError(form, e, { name: /tên/i, sort_order: /thứ tự/i }))
            showError(e, 'Không thể lưu danh mục');
        },
      },
    );

  return (
    <Modal
      open={open}
      title={isEdit ? `Sửa danh mục ${record?.name}` : 'Thêm danh mục sản phẩm'}
      okText={isEdit ? 'Lưu thay đổi' : 'Thêm danh mục'}
      cancelText="Hủy"
      onOk={() => form.submit()}
      onCancel={onClose}
      confirmLoading={save.isPending}
      forceRender
    >
      <Form form={form} layout="vertical" onFinish={onFinish} style={{ marginTop: 12 }}>
        <Row gutter={16}>
          <Col xs={24} sm={16}>
            <Form.Item
              name="name"
              label="Tên danh mục"
              rules={[
                { required: true, whitespace: true, message: 'Vui lòng nhập tên danh mục' },
                { max: 100, message: 'Tên danh mục tối đa 100 ký tự' },
              ]}
            >
              <Input placeholder="VD: Đồ uống" maxLength={100} />
            </Form.Item>
          </Col>
          <Col xs={24} sm={8}>
            <Form.Item
              name="sort_order"
              label="Thứ tự"
              tooltip="Số nhỏ hiển thị trước (POS, bộ lọc)"
              rules={[{ type: 'number', min: 0, message: 'Thứ tự không được âm' }]}
            >
              <InputNumber min={0} precision={0} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
        </Row>
      </Form>
    </Modal>
  );
}

/** `/app/product-categories` – danh mục sản phẩm (không phân trang). */
export default function ProductCategoriesPage() {
  const navigate = useNavigate();
  const { can } = usePermission();
  const url = useUrlParams();
  const keyword = url.get('keyword') ?? '';
  const remove = useDeleteCategories();
  const [modal, setModal] = useState<{ open: boolean; record: ProductCategory | null }>({
    open: false,
    record: null,
  });
  const canManage = can(PERM.PRODUCT_MANAGE);

  const query = useQuery({
    queryKey: [productCategoryApi.key, 'list', keyword],
    queryFn: () => productCategoryApi.list(keyword),
  });

  return (
    <Card variant="borderless" className="page-card">
      <Row justify="space-between" align="middle" gutter={[8, 12]} className="page-head">
        <Col>
          <PageTitle title="Danh mục sản phẩm" subtitle="Nhóm hàng hiển thị trên POS và bộ lọc" />
        </Col>
        <Col>
          <ActionBar
            onAdd={() => setModal({ open: true, record: null })}
            addPerm={PERM.PRODUCT_MANAGE}
            showDelete={false}
            onRefresh={() => query.refetch()}
            refreshing={query.isFetching}
          />
        </Col>
      </Row>
      <FilterBar searchPlaceholder="Tìm theo tên danh mục" />
      {query.isError ? (
        <Result
          status="warning"
          title={getErrorMessage(query.error, 'Không tải được danh mục')}
          extra={<Button onClick={() => query.refetch()}>Thử lại</Button>}
        />
      ) : (
        <Table<ProductCategory>
          size="small"
          rowKey="id"
          loading={query.isLoading}
          dataSource={query.data ?? []}
          pagination={false}
          scroll={{ x: 'max-content' }}
          locale={{ emptyText: 'Chưa có danh mục nào' }}
          columns={[
            { title: 'Thứ tự', dataIndex: 'sort_order', width: 80, align: 'center' },
            {
              title: 'Tên danh mục',
              dataIndex: 'name',
              render: (v: string, r) =>
                canManage ? (
                  <Button
                    type="link"
                    size="small"
                    style={{ padding: 0 }}
                    onClick={() => setModal({ open: true, record: r })}
                  >
                    {v}
                  </Button>
                ) : (
                  v
                ),
            },
            {
              title: 'Số sản phẩm',
              dataIndex: 'product_count',
              align: 'right',
              render: (v: number, r) => (
                <Button
                  type="link"
                  size="small"
                  style={{ padding: 0 }}
                  onClick={() => navigate(`/app/products?category_id=${r.id}`)}
                >
                  {formatNumber(v)}
                </Button>
              ),
            },
            {
              title: 'Cập nhật lúc',
              dataIndex: 'updated_at',
              render: (v: string) => <DateText value={v} withTime />,
            },
            ...(canManage
              ? [
                  {
                    title: '',
                    key: 'actions',
                    fixed: 'right' as const,
                    width: 88,
                    render: (_: unknown, r: ProductCategory) => (
                      <Space size={0}>
                        <Tooltip title="Sửa">
                          <Button
                            type="text"
                            size="small"
                            icon={<EditOutlined />}
                            onClick={() => setModal({ open: true, record: r })}
                          />
                        </Tooltip>
                        <ConfirmButton
                          type="text"
                          size="small"
                          danger
                          icon={<DeleteOutlined />}
                          confirmTitle={`Xóa danh mục "${r.name}"?`}
                          confirmContent={
                            r.product_count > 0
                              ? `Danh mục đang có ${r.product_count} sản phẩm – cần chuyển hoặc xóa sản phẩm trước.`
                              : undefined
                          }
                          onConfirm={() => remove.mutateAsync([r])}
                          aria-label="Xóa"
                        />
                      </Space>
                    ),
                  },
                ]
              : []),
          ]}
        />
      )}
      <CategoryForm
        open={modal.open}
        record={modal.record}
        onClose={() => setModal((m) => ({ ...m, open: false }))}
      />
    </Card>
  );
}
