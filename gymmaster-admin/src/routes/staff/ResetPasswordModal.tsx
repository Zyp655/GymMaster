import { useEffect } from 'react';
import { Alert, App, Form, Input, Modal } from 'antd';
import type { Staff } from '@/api/staff';
import { getErrorMessage, setFormError, showError } from '@/utils/errors';
import { useResetStaffPassword } from './queries';
import { isStaffGuardError } from './staffGuards';

interface Values {
  password: string;
  confirm: string;
}

/** Đặt lại mật khẩu cho nhân viên (thu hồi mọi phiên đăng nhập). */
export default function ResetPasswordModal({
  staff,
  onClose,
}: {
  staff: Staff | null;
  onClose: () => void;
}) {
  const [form] = Form.useForm<Values>();
  const { message, modal } = App.useApp();
  const reset = useResetStaffPassword();

  useEffect(() => {
    if (staff) form.resetFields();
  }, [staff, form]);

  const onFinish = (v: Values) => {
    if (!staff) return;
    reset.mutate(
      { id: staff.id, password: v.password },
      {
        onSuccess: () => {
          message.success(`Đã đặt lại mật khẩu cho ${staff.full_name}`);
          onClose();
        },
        onError: (e) => {
          // 403/400 "chính mình" không phải lỗi ô mật khẩu → hiện nguyên thông điệp server.
          if (isStaffGuardError(e)) {
            modal.error({ title: 'Không thể đặt lại mật khẩu', content: getErrorMessage(e) });
            return;
          }
          if (!setFormError(form, e, { password: /mật khẩu/i }))
            showError(e, 'Không thể đặt lại mật khẩu');
        },
      },
    );
  };

  return (
    <Modal
      open={!!staff}
      title={`Đặt lại mật khẩu – ${staff?.full_name ?? ''}`}
      okText="Đặt lại mật khẩu"
      cancelText="Hủy"
      onOk={() => form.submit()}
      onCancel={onClose}
      confirmLoading={reset.isPending}
      forceRender
      width={460}
    >
      <Alert
        type="info"
        showIcon
        style={{ margin: '12px 0' }}
        message="Nhân viên sẽ bị đăng xuất khỏi mọi thiết bị và phải đăng nhập bằng mật khẩu mới."
      />
      <Form<Values> form={form} layout="vertical" onFinish={onFinish}>
        <Form.Item
          name="password"
          label="Mật khẩu mới"
          rules={[
            { required: true, message: 'Vui lòng nhập mật khẩu mới' },
            { min: 8, max: 72, message: 'Mật khẩu từ 8 đến 72 ký tự' },
          ]}
        >
          <Input.Password autoComplete="new-password" maxLength={72} />
        </Form.Item>
        <Form.Item
          name="confirm"
          label="Nhập lại mật khẩu"
          dependencies={['password']}
          rules={[
            { required: true, message: 'Vui lòng nhập lại mật khẩu' },
            ({ getFieldValue }) => ({
              validator: (_, value?: string) =>
                !value || value === getFieldValue('password')
                  ? Promise.resolve()
                  : Promise.reject(new Error('Mật khẩu nhập lại không khớp')),
            }),
          ]}
        >
          <Input.Password autoComplete="new-password" maxLength={72} />
        </Form.Item>
      </Form>
    </Modal>
  );
}
