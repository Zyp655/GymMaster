import { useEffect } from 'react';
import { Alert, App, Col, Form, Input, Modal, Row, Skeleton, Tag } from 'antd';
import type { Role } from '@/api/types';
import { PERM, ROLE } from '@/constants/permissions';
import { usePermission } from '@/hooks/usePermission';
import { getErrorMessage, setFormError, showError } from '@/utils/errors';
import PermissionPicker from './PermissionPicker';
import { usePermissions, useSaveRole } from './queries';

interface FormValues {
  code: string;
  name: string;
  description?: string;
  permissions: string[];
}

export interface RoleFormProps {
  open: boolean;
  record: Role | null;
  onClose: () => void;
}

/** Modal tạo/sửa vai trò + phân quyền. Vai trò hệ thống khóa mã; OWNER luôn toàn quyền. */
export default function RoleForm({ open, record, onClose }: RoleFormProps) {
  const [form] = Form.useForm<FormValues>();
  const { message } = App.useApp();
  const { can } = usePermission();
  const perms = usePermissions(open);
  const save = useSaveRole();

  const isEdit = !!record;
  const isOwner = record?.code === ROLE.OWNER;
  const readOnly = isEdit ? !can(PERM.ROLE_UPDATE) : !can(PERM.ROLE_CREATE);

  useEffect(() => {
    if (!open) return;
    form.resetFields();
    form.setFieldsValue(
      record
        ? {
            code: record.code,
            name: record.name,
            description: record.description,
            permissions: record.permissions ?? [],
          }
        : { permissions: [] },
    );
  }, [open, record, form]);

  const onFinish = (v: FormValues) => {
    const base = {
      name: v.name.trim(),
      description: v.description?.trim() ?? '',
      permissions: v.permissions ?? [],
    };
    save.mutate(
      record
        ? { id: record.id, update: base }
        : { create: { ...base, code: v.code.trim().toUpperCase() } },
      {
        onSuccess: () => {
          message.success(isEdit ? 'Đã cập nhật vai trò' : 'Đã thêm vai trò');
          onClose();
        },
        onError: (error) => {
          if (!setFormError(form, error, { code: /mã/i, name: /tên/i, description: /mô tả/i })) {
            showError(error, 'Không thể lưu vai trò');
          }
        },
      },
    );
  };

  const allCodes = perms.data?.map((p) => p.code) ?? [];

  return (
    <Modal
      open={open}
      title={
        isEdit ? (
          <>
            Vai trò {record?.name}{' '}
            {record?.is_system && (
              <Tag color="blue" bordered={false}>
                Hệ thống
              </Tag>
            )}
          </>
        ) : (
          'Thêm vai trò'
        )
      }
      okText={isEdit ? 'Lưu thay đổi' : 'Thêm vai trò'}
      cancelText={readOnly ? 'Đóng' : 'Hủy'}
      okButtonProps={{ style: readOnly ? { display: 'none' } : undefined }}
      onOk={() => form.submit()}
      onCancel={onClose}
      confirmLoading={save.isPending}
      width={900}
      forceRender
      styles={{ body: { maxHeight: '70vh', overflowY: 'auto', paddingRight: 4 } }}
    >
      <Form<FormValues>
        form={form}
        layout="vertical"
        disabled={readOnly}
        onFinish={onFinish}
        style={{ marginTop: 12 }}
      >
        <Row gutter={16}>
          <Col xs={24} sm={8}>
            <Form.Item
              name="code"
              label="Mã vai trò"
              tooltip={
                isEdit ? 'Không thể đổi mã vai trò' : 'Chữ in hoa, số, "_" và bắt đầu bằng chữ'
              }
              normalize={(v: string) => v?.toUpperCase()}
              rules={[
                { required: true, message: 'Vui lòng nhập mã vai trò' },
                {
                  pattern: /^[A-Z][A-Z0-9_]{1,49}$/,
                  message: "Mã gồm chữ in hoa, số, '_' và bắt đầu bằng chữ",
                },
              ]}
            >
              <Input
                disabled={isEdit}
                placeholder="VD: ACCOUNTANT"
                className="code-text"
                maxLength={50}
              />
            </Form.Item>
          </Col>
          <Col xs={24} sm={16}>
            <Form.Item
              name="name"
              label="Tên vai trò"
              rules={[
                { required: true, whitespace: true, message: 'Vui lòng nhập tên vai trò' },
                { max: 100, message: 'Tên tối đa 100 ký tự' },
              ]}
            >
              <Input placeholder="VD: Kế toán" maxLength={100} />
            </Form.Item>
          </Col>
          <Col xs={24}>
            <Form.Item
              name="description"
              label="Mô tả"
              rules={[{ max: 255, message: 'Mô tả tối đa 255 ký tự' }]}
            >
              <Input.TextArea
                rows={2}
                maxLength={255}
                showCount
                placeholder="Mô tả ngắn về vai trò"
              />
            </Form.Item>
          </Col>
        </Row>

        <Form.Item label="Phân quyền" required={false} style={{ marginBottom: 0 }}>
          {isOwner && (
            <Alert
              type="info"
              showIcon
              style={{ marginBottom: 12 }}
              message="Chủ trung tâm luôn có toàn quyền"
              description="Danh sách quyền của vai trò OWNER được hệ thống tự cấp đầy đủ và không thể thu hẹp."
            />
          )}
          {perms.isLoading ? (
            <Skeleton active paragraph={{ rows: 6 }} />
          ) : perms.isError ? (
            <Alert
              type="error"
              showIcon
              message={getErrorMessage(perms.error, 'Không tải được danh sách quyền')}
            />
          ) : isOwner ? (
            <PermissionPicker permissions={perms.data ?? []} value={allCodes} disabled />
          ) : (
            <Form.Item name="permissions" noStyle>
              <PermissionPicker permissions={perms.data ?? []} disabled={readOnly} />
            </Form.Item>
          )}
        </Form.Item>
      </Form>
    </Modal>
  );
}
