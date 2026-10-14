import { App, Alert, Form, Input, Modal } from 'antd';
import type { Member } from '@/api/member';
import { setFormError, showError } from '@/utils/errors';
import { useGrantPortal } from '../queries';

interface FormValues {
  email: string;
  password: string;
}

/** Cấp (hoặc mở lại tài khoản đã khóa) tài khoản cổng hội viên. */
export default function PortalModal({
  member,
  open,
  onClose,
}: {
  member: Member;
  open: boolean;
  onClose: () => void;
}) {
  const [form] = Form.useForm<FormValues>();
  const { message } = App.useApp();
  const grant = useGrantPortal(member.id);
  const reopen = member.portal_account?.status === 'LOCKED';

  const onFinish = (v: FormValues) =>
    grant.mutate(
      { email: v.email.trim().toLowerCase(), password: v.password },
      {
        onSuccess: (acc) => {
          message.success(`Đã ${reopen ? 'mở lại' : 'cấp'} tài khoản ${acc.email}`);
          onClose();
        },
        onError: (e) => {
          if (!setFormError(form, e, { email: /email/i, password: /mật khẩu/i }))
            showError(e, 'Không thể cấp tài khoản');
        },
      },
    );

  return (
    <Modal
      open={open}
      title={reopen ? 'Mở lại tài khoản cổng hội viên' : 'Cấp tài khoản cổng hội viên'}
      okText={reopen ? 'Mở lại tài khoản' : 'Cấp tài khoản'}
      cancelText="Hủy"
      onOk={() => form.submit()}
      onCancel={onClose}
      confirmLoading={grant.isPending}
      afterOpenChange={(o) => {
        if (!o) return;
        form.resetFields();
        form.setFieldsValue({ email: member.portal_account?.email || member.email || '' });
      }}
      forceRender
    >
      {reopen && (
        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 12 }}
          message="Tài khoản đang bị khóa. Nhập email và mật khẩu mới để mở lại."
        />
      )}
      <Form<FormValues> form={form} layout="vertical" onFinish={onFinish} autoComplete="off">
        <Form.Item
          name="email"
          label="Email đăng nhập"
          rules={[
            { required: true, message: 'Vui lòng nhập email' },
            { type: 'email', message: 'Email không hợp lệ' },
          ]}
        >
          <Input placeholder="hoivien@example.com" inputMode="email" autoComplete="off" />
        </Form.Item>
        <Form.Item
          name="password"
          label="Mật khẩu"
          extra="Gửi mật khẩu cho hội viên và đề nghị đổi sau lần đăng nhập đầu tiên."
          rules={[
            { required: true, message: 'Vui lòng nhập mật khẩu' },
            { min: 8, max: 72, message: 'Mật khẩu dài 8–72 ký tự' },
          ]}
        >
          <Input.Password autoComplete="new-password" placeholder="Tối thiểu 8 ký tự" />
        </Form.Item>
      </Form>
    </Modal>
  );
}
