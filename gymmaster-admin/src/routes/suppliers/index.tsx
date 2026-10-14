import { useCallback, useEffect, useMemo, useState } from 'react';
import { App, Button, Card, Col, Form, Input, Modal, Row, Space, Tooltip, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { DeleteOutlined, EditOutlined, EyeOutlined, PhoneOutlined } from '@ant-design/icons';
import { supplierApi } from '@/api/product';
import type { Supplier, SupplierInput, SupplierListParams } from '@/api/product';
import type { SortDir } from '@/api/types';
import ActionBar from '@/components/ActionBar';
import ConfirmButton from '@/components/ConfirmButton';
import DataTable from '@/components/DataTable';
import DateText from '@/components/DateText';
import FilterBar from '@/components/FilterBar';
import MobileList from '@/components/MobileList';
import type { MobileFetchArgs } from '@/components/MobileList';
import PageTitle from '@/components/PageTitle';
import { PERM } from '@/constants/permissions';
import { useIsMobile } from '@/hooks/useIsMobile';
import { usePermission } from '@/hooks/usePermission';
import { useRowSelection } from '@/hooks/useRowSelection';
import { useTableQuery } from '@/hooks/useTableQuery';
import { useDeleteSuppliers, useSaveSupplier } from '@/routes/products/queries';
import { setFormError, showError } from '@/utils/errors';
import { formatDateTime, formatPhone } from '@/utils/format';

const TITLE = 'Nhà cung cấp';
const SUBTITLE = 'Đối tác cung cấp hàng hóa cho phiếu nhập kho';

function SupplierForm({
  open,
  record,
  onClose,
}: {
  open: boolean;
  record: Supplier | null;
  onClose: () => void;
}) {
  const [form] = Form.useForm<SupplierInput>();
  const { message } = App.useApp();
  const { can } = usePermission();
  const save = useSaveSupplier();
  const isEdit = !!record;
  const readOnly = !can(PERM.PRODUCT_MANAGE);

  useEffect(() => {
    if (!open) return;
    form.resetFields();
    if (record) form.setFieldsValue(record);
  }, [open, record, form]);

  const onFinish = (v: SupplierInput) => {
    const data: SupplierInput = {
      name: v.name.trim(),
      phone: v.phone?.trim() ?? '',
      email: v.email?.trim() ?? '',
      address: v.address?.trim() ?? '',
      note: v.note?.trim() ?? '',
    };
    save.mutate(
      { id: record?.id, data },
      {
        onSuccess: () => {
          message.success(isEdit ? 'Đã cập nhật nhà cung cấp' : 'Đã thêm nhà cung cấp');
          onClose();
        },
        onError: (e) => {
          const mapped = setFormError(form, e, {
            name: /tên/i,
            phone: /điện thoại/i,
            email: /email/i,
            address: /địa chỉ/i,
            note: /ghi chú/i,
          });
          if (!mapped) showError(e, 'Không thể lưu nhà cung cấp');
        },
      },
    );
  };

  return (
    <Modal
      open={open}
      title={isEdit ? record?.name : 'Thêm nhà cung cấp'}
      okText={isEdit ? 'Lưu thay đổi' : 'Thêm nhà cung cấp'}
      cancelText={readOnly ? 'Đóng' : 'Hủy'}
      okButtonProps={{ style: readOnly ? { display: 'none' } : undefined }}
      onOk={() => form.submit()}
      onCancel={onClose}
      confirmLoading={save.isPending}
      width={680}
      forceRender
    >
      <Form<SupplierInput>
        form={form}
        layout="vertical"
        disabled={readOnly}
        onFinish={onFinish}
        style={{ marginTop: 12 }}
      >
        <Row gutter={16}>
          <Col xs={24}>
            <Form.Item
              name="name"
              label="Tên nhà cung cấp"
              rules={[
                { required: true, whitespace: true, message: 'Vui lòng nhập tên nhà cung cấp' },
                { max: 150, message: 'Tên tối đa 150 ký tự' },
              ]}
            >
              <Input maxLength={150} placeholder="VD: Công ty TNHH Dinh Dưỡng Thể Hình" />
            </Form.Item>
          </Col>
          <Col xs={24} sm={12}>
            <Form.Item
              name="phone"
              label="Số điện thoại"
              rules={[{ pattern: /^[0-9+ ]{8,20}$/, message: 'Số điện thoại không hợp lệ' }]}
            >
              <Input maxLength={20} inputMode="tel" />
            </Form.Item>
          </Col>
          <Col xs={24} sm={12}>
            <Form.Item
              name="email"
              label="Email"
              rules={[
                { type: 'email', message: 'Email không hợp lệ' },
                { max: 150, message: 'Email tối đa 150 ký tự' },
              ]}
            >
              <Input maxLength={150} inputMode="email" />
            </Form.Item>
          </Col>
          <Col xs={24}>
            <Form.Item
              name="address"
              label="Địa chỉ"
              rules={[{ max: 255, message: 'Địa chỉ tối đa 255 ký tự' }]}
            >
              <Input maxLength={255} />
            </Form.Item>
          </Col>
          <Col xs={24}>
            <Form.Item
              name="note"
              label="Ghi chú"
              rules={[{ max: 255, message: 'Ghi chú tối đa 255 ký tự' }]}
            >
              <Input.TextArea maxLength={255} autoSize={{ minRows: 2, maxRows: 4 }} />
            </Form.Item>
          </Col>
        </Row>
      </Form>
    </Modal>
  );
}

function SuppliersDesktop() {
  const { can } = usePermission();
  const [modal, setModal] = useState<{ open: boolean; record: Supplier | null }>({
    open: false,
    record: null,
  });
  const selection = useRowSelection<Supplier>();
  const remove = useDeleteSuppliers();
  const table = useTableQuery<Supplier, SupplierListParams>({
    queryKey: [supplierApi.key, 'list'],
    queryFn: (p) => supplierApi.list(p),
    defaults: { order_by: 'id', order_dir: 'DESC' },
  });
  const openForm = (record: Supplier | null = null) => setModal({ open: true, record });

  const columns = useMemo<ColumnsType<Supplier>>(
    () => [
      {
        title: 'Tên nhà cung cấp',
        dataIndex: 'name',
        sorter: true,
        render: (v: string, r) => (
          <Button
            type="link"
            size="small"
            style={{ padding: 0, whiteSpace: 'normal', textAlign: 'left', height: 'auto' }}
            onClick={() => openForm(r)}
          >
            {v}
          </Button>
        ),
      },
      {
        title: 'Điện thoại',
        dataIndex: 'phone',
        className: 'text-nowrap',
        render: (v: string) => (v ? <a href={`tel:${v}`}>{formatPhone(v)}</a> : '—'),
      },
      { title: 'Email', dataIndex: 'email', render: (v: string) => v || '—' },
      {
        title: 'Địa chỉ',
        dataIndex: 'address',
        render: (v: string) =>
          v ? (
            <Typography.Text ellipsis={{ tooltip: v }} style={{ maxWidth: 240 }}>
              {v}
            </Typography.Text>
          ) : (
            '—'
          ),
      },
      {
        title: 'Ghi chú',
        dataIndex: 'note',
        render: (v: string) =>
          v ? (
            <Typography.Text ellipsis={{ tooltip: v }} style={{ maxWidth: 200 }}>
              {v}
            </Typography.Text>
          ) : (
            '—'
          ),
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
              confirmTitle={`Xóa nhà cung cấp "${r.name}"?`}
              confirmContent="Thẻ kho cũ vẫn giữ tên nhà cung cấp."
              onConfirm={() => remove.mutateAsync([r])}
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
            onAdd={() => openForm()}
            addPerm={PERM.PRODUCT_MANAGE}
            onDelete={async () => {
              await remove.mutateAsync(selection.rows.map(({ id, name }) => ({ id, name })));
              selection.clear();
            }}
            deletePerm={PERM.PRODUCT_MANAGE}
            selectedCount={selection.keys.length}
            deleteConfirmTitle={`Xóa ${selection.keys.length} nhà cung cấp đã chọn?`}
            onRefresh={() => table.refetch()}
            refreshing={table.isFetching}
          />
        </Col>
      </Row>
      <FilterBar searchPlaceholder="Tìm theo tên, SĐT, email..." />
      <DataTable<Supplier>
        table={table}
        columns={columns}
        rowSelection={can(PERM.PRODUCT_MANAGE) ? selection.rowSelection : undefined}
        emptyText="Chưa có nhà cung cấp nào"
      />
      <SupplierForm
        open={modal.open}
        record={modal.record}
        onClose={() => setModal((m) => ({ ...m, open: false }))}
      />
    </Card>
  );
}

function SuppliersMobile() {
  const { can } = usePermission();
  const { modal } = App.useApp();
  const remove = useDeleteSuppliers();
  const [form, setForm] = useState<{ open: boolean; record: Supplier | null }>({
    open: false,
    record: null,
  });
  const fetchPage = useCallback(({ page, limit, keyword, sort }: MobileFetchArgs) => {
    const [order_by, order_dir] = (sort ?? 'id:DESC').split(':') as [string, SortDir];
    return supplierApi.list({ page, limit, keyword: keyword || undefined, order_by, order_dir });
  }, []);
  const openForm = (record: Supplier | null) => setForm({ open: true, record });

  return (
    <>
      <MobileList<Supplier>
        title={TITLE}
        queryKey={[supplierApi.key, 'list']}
        fetchPage={fetchPage}
        searchPlaceholder="Tên, SĐT, email"
        emptyText="Chưa có nhà cung cấp nào"
        sorts={[
          { label: 'Mới nhất', value: 'id:DESC' },
          { label: 'Tên A → Z', value: 'name:ASC' },
        ]}
        row={{
          title: (r) => r.name,
          subtitle: (r) => r.address || 'Chưa có địa chỉ',
          meta: (r) => [r.phone && formatPhone(r.phone), r.email],
        }}
        onAdd={() => openForm(null)}
        addPerm={PERM.PRODUCT_MANAGE}
        details={(r) => [
          ['Ghi chú', r.note],
          ['Ngày tạo', formatDateTime(r.created_at)],
        ]}
        actions={(r) => [
          {
            key: 'edit',
            icon: can(PERM.PRODUCT_MANAGE) ? <EditOutlined /> : <EyeOutlined />,
            label: can(PERM.PRODUCT_MANAGE) ? 'Sửa' : 'Xem chi tiết',
            onClick: openForm,
          },
          ...(r.phone
            ? [
                {
                  key: 'call',
                  icon: <PhoneOutlined />,
                  label: `Gọi ${formatPhone(r.phone)}`,
                  onClick: () => window.open(`tel:${r.phone}`),
                },
              ]
            : []),
          ...(can(PERM.PRODUCT_MANAGE)
            ? [
                {
                  key: 'delete',
                  icon: <DeleteOutlined />,
                  label: 'Xóa',
                  danger: true,
                  onClick: () =>
                    modal.confirm({
                      title: `Xóa nhà cung cấp "${r.name}"?`,
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
      <SupplierForm
        open={form.open}
        record={form.record}
        onClose={() => setForm((f) => ({ ...f, open: false }))}
      />
    </>
  );
}

/** `/app/suppliers` – nhà cung cấp. */
export default function SuppliersPage() {
  return useIsMobile() ? <SuppliersMobile /> : <SuppliersDesktop />;
}
