import { App, Form, Input, Modal } from 'antd';
import { useMutation } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import authApi from '@/api/auth';
import { logout } from '@/store/authSlice';
import { useAppDispatch } from '@/store';
import { ErrorCode, isApiError } from '@/utils/api';
import { getErrorMessage, showError } from '@/utils/errors';

interface FormValues {
  old_password: string;
  new_password: string;
  confirm_password: string;
}

export interface ChangePasswordModalProps {
  open: boolean;
  onClose: () => void;
}

/**
 * Đổi mật khẩu. Backend thu hồi mọi refresh token sau khi đổi → đăng xuất và yêu cầu đăng nhập lại.
 */
export function ChangePasswordModal({ open, onClose }: ChangePasswordModalProps) {
  const [form] = Form.useForm<FormValues>();
  const { message } = App.useApp();
  const dispatch = useAppDispatch();
  const navigate = useNavigate();

  const mutation = useMutation({
    mutationFn: (v: FormValues) => authApi.changePassword(v.old_password, v.new_password),
    onSuccess: async () => {
      message.success('Đổi mật khẩu thành công, vui lòng đăng nhập lại');
      onClose();
      await dispatch(logout());
      navigate('/login', { replace: true });
    },
    onError: (error) => {
      // Sai mật khẩu cũ / lỗi validate → hiển thị ngay dưới ô nhập.
      if (
        isApiError(error) &&
        [ErrorCode.INVALID_CREDENTIALS, ErrorCode.BAD_REQUEST, ErrorCode.VALIDATION].includes(
          error.code as never,
        )
      ) {
        const field = /mới|new/i.test(error.message) ? 'new_password' : 'old_password';
        form.setFields([{ name: field, errors: [getErrorMessage(error)] }]);
        return;
      }
      showError(error, 'Không thể đổi mật khẩu');
    },
  });

  return (
    <Modal
      open={open}
      title="Đổi mật khẩu"
      okText="Đổi mật khẩu"
      cancelText="Hủy"
      onCancel={onClose}
      onOk={() => form.submit()}
      confirmLoading={mutation.isPending}
      destroyOnHidden
      width={440}
    >
      <Form<FormValues>
        form={form}
        layout="vertical"
        requiredMark={false}
        onFinish={(v) => mutation.mutate(v)}
        preserve={false}
      >
        <Form.Item
          name="old_password"
          label="Mật khẩu hiện tại"
          rules={[{ required: true, message: 'Vui lòng nhập mật khẩu hiện tại' }]}
        >
          <Input.Password autoComplete="current-password" autoFocus />
        </Form.Item>
        <Form.Item
          name="new_password"
          label="Mật khẩu mới"
          dependencies={['old_password']}
          rules={[
            { required: true, message: 'Vui lòng nhập mật khẩu mới' },
            { min: 8, message: 'Mật khẩu mới tối thiểu 8 ký tự' },
            ({ getFieldValue }) => ({
              validator: (_, value) =>
                !value || value !== getFieldValue('old_password')
                  ? Promise.resolve()
                  : Promise.reject(new Error('Mật khẩu mới phải khác mật khẩu hiện tại')),
            }),
          ]}
        >
          <Input.Password autoComplete="new-password" />
        </Form.Item>
        <Form.Item
          name="confirm_password"
          label="Nhập lại mật khẩu mới"
          dependencies={['new_password']}
          rules={[
            { required: true, message: 'Vui lòng nhập lại mật khẩu mới' },
            ({ getFieldValue }) => ({
              validator: (_, value) =>
                !value || value === getFieldValue('new_password')
                  ? Promise.resolve()
                  : Promise.reject(new Error('Mật khẩu nhập lại không khớp')),
            }),
          ]}
        >
          <Input.Password autoComplete="new-password" />
        </Form.Item>
      </Form>
    </Modal>
  );
}

export default ChangePasswordModal;
