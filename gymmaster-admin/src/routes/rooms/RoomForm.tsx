import { useEffect } from 'react';
import { App, Col, Form, Input, InputNumber, Modal, Row, Segmented } from 'antd';
import type { Room, RoomInput } from '@/api/room';
import BranchSelect from '@/components/BranchSelect';
import { PERM } from '@/constants/permissions';
import { usePermission } from '@/hooks/usePermission';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { setFormError, showError } from '@/utils/errors';
import { useSaveRoom } from './queries';

export interface RoomFormProps {
  open: boolean;
  /** `null` = thêm mới. */
  record: Room | null;
  onClose: () => void;
}

/** Modal thêm/sửa phòng – validate giống backend (api/room/v1/room.go). */
export default function RoomForm({ open, record, onClose }: RoomFormProps) {
  const [form] = Form.useForm<RoomInput>();
  const { message } = App.useApp();
  const { can } = usePermission();
  const { branchId } = useWorkingBranch();
  const save = useSaveRoom();
  const isEdit = !!record;
  const readOnly = !can(PERM.ROOM_MANAGE);

  useEffect(() => {
    if (!open) return;
    form.resetFields();
    form.setFieldsValue(
      record ? { ...record } : { status: 'ACTIVE', capacity: 20, branch_id: branchId ?? undefined },
    );
  }, [open, record, form, branchId]);

  const onFinish = (values: RoomInput) => {
    const data: RoomInput = {
      branch_id: values.branch_id,
      name: values.name.trim(),
      capacity: values.capacity,
      description: values.description?.trim() ?? '',
      status: values.status,
    };
    save.mutate(
      { id: record?.id, data },
      {
        onSuccess: () => {
          message.success(isEdit ? 'Đã cập nhật phòng' : 'Đã thêm phòng');
          onClose();
        },
        onError: (error) => {
          const mapped = setFormError(form, error, {
            name: /tên phòng/i,
            capacity: /sức chứa/i,
            branch_id: /chi nhánh/i,
            description: /mô tả/i,
          });
          if (!mapped) showError(error, 'Không thể lưu phòng');
        },
      },
    );
  };

  return (
    <Modal
      open={open}
      title={isEdit ? `Phòng ${record?.name}` : 'Thêm phòng tập'}
      okText={isEdit ? 'Lưu thay đổi' : 'Thêm phòng'}
      cancelText={readOnly ? 'Đóng' : 'Hủy'}
      okButtonProps={{ style: readOnly ? { display: 'none' } : undefined }}
      onOk={() => form.submit()}
      onCancel={onClose}
      confirmLoading={save.isPending}
      width={600}
      forceRender
    >
      <Form<RoomInput>
        form={form}
        layout="vertical"
        disabled={readOnly}
        onFinish={onFinish}
        style={{ marginTop: 12 }}
      >
        <Row gutter={16}>
          <Col xs={24} sm={12}>
            <Form.Item
              name="branch_id"
              label="Chi nhánh"
              rules={[{ required: true, message: 'Vui lòng chọn chi nhánh' }]}
            >
              <BranchSelect activeOnly />
            </Form.Item>
          </Col>
          <Col xs={24} sm={12}>
            <Form.Item
              name="name"
              label="Tên phòng"
              tooltip="Không trùng trong cùng chi nhánh."
              rules={[
                { required: true, whitespace: true, message: 'Vui lòng nhập tên phòng' },
                { max: 100, message: 'Tên phòng tối đa 100 ký tự' },
              ]}
            >
              <Input placeholder="VD: Phòng Yoga 1" maxLength={100} />
            </Form.Item>
          </Col>
          <Col xs={12} sm={12}>
            <Form.Item
              name="capacity"
              label="Sức chứa (người)"
              rules={[
                { required: true, message: 'Vui lòng nhập sức chứa' },
                {
                  type: 'number',
                  min: 1,
                  max: 1000,
                  message: 'Sức chứa từ 1 đến 1000 người',
                },
              ]}
            >
              <InputNumber min={1} max={1000} precision={0} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
          <Col xs={12} sm={12}>
            <Form.Item name="status" label="Trạng thái" rules={[{ required: true }]}>
              <Segmented
                block
                options={[
                  { label: 'Hoạt động', value: 'ACTIVE' },
                  { label: 'Ngừng', value: 'INACTIVE' },
                ]}
              />
            </Form.Item>
          </Col>
          <Col xs={24}>
            <Form.Item
              name="description"
              label="Mô tả"
              rules={[{ max: 255, message: 'Mô tả tối đa 255 ký tự' }]}
            >
              <Input.TextArea rows={2} maxLength={255} showCount placeholder="VD: Tầng 2" />
            </Form.Item>
          </Col>
        </Row>
      </Form>
    </Modal>
  );
}
