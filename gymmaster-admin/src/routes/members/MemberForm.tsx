import { useEffect } from 'react';
import { App, Col, DatePicker, Divider, Form, Input, Modal, Row, Select } from 'antd';
import type { Rule } from 'antd/es/form';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import { GENDER_OPTIONS, MEMBER_STATUS_OPTIONS, SOURCE_OPTIONS } from '@/api/member';
import type { Member, MemberInput } from '@/api/member';
import BranchSelect from '@/components/BranchSelect';
import ImageUpload from '@/components/ImageUpload';
import { PERM } from '@/constants/permissions';
import { usePermission } from '@/hooks/usePermission';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { setFormError, showError } from '@/utils/errors';
import { DATE_FORMAT, toApiDate } from '@/utils/format';
import { useSaveMember } from './queries';
import { normalizeVNPhone } from './utils';

interface FormValues extends Omit<MemberInput, 'dob'> {
  dob?: Dayjs | null;
}

export interface MemberFormProps {
  open: boolean;
  /** `null` = thêm mới. */
  record: Member | null;
  onClose: () => void;
  onSaved?: (member: Member) => void;
}

const phoneRule = (required: boolean): Rule => ({
  validator: (_, value?: string) => {
    if (!value?.trim()) {
      return required
        ? Promise.reject(new Error('Vui lòng nhập số điện thoại'))
        : Promise.resolve();
    }
    return normalizeVNPhone(value)
      ? Promise.resolve()
      : Promise.reject(new Error('Số điện thoại không hợp lệ (SĐT Việt Nam 10 số)'));
  },
});

const trim = (v?: string | null) => v?.trim() ?? '';

/** Modal thêm/sửa hội viên – validate giống `api/member/v1` (members.md mục 2). */
export default function MemberForm({ open, record, onClose, onSaved }: MemberFormProps) {
  const [form] = Form.useForm<FormValues>();
  const { message } = App.useApp();
  const { can } = usePermission();
  const { branchId } = useWorkingBranch();
  const save = useSaveMember();
  const isEdit = !!record;
  const readOnly = isEdit ? !can(PERM.MEMBER_UPDATE) : !can(PERM.MEMBER_CREATE);

  useEffect(() => {
    if (!open) return;
    form.resetFields();
    form.setFieldsValue(
      record
        ? { ...record, dob: record.dob ? dayjs(record.dob) : null }
        : {
            gender: 'OTHER',
            source: 'WALK_IN',
            status: 'ACTIVE',
            home_branch_id: branchId ?? undefined,
          },
    );
  }, [open, record, form, branchId]);

  const onFinish = (v: FormValues) => {
    const data: MemberInput = {
      full_name: trim(v.full_name),
      phone: trim(v.phone),
      gender: v.gender ?? 'OTHER',
      dob: toApiDate(v.dob),
      email: trim(v.email),
      id_card: trim(v.id_card),
      address: trim(v.address),
      avatar_url: v.avatar_url ?? '',
      emergency_contact_name: trim(v.emergency_contact_name),
      emergency_contact_phone: trim(v.emergency_contact_phone),
      home_branch_id: v.home_branch_id,
      source: v.source ?? 'WALK_IN',
      status: v.status ?? 'ACTIVE',
      note: trim(v.note),
    };
    save.mutate(
      { id: record?.id, data },
      {
        onSuccess: (member) => {
          message.success(
            isEdit ? 'Đã cập nhật hội viên' : `Đã thêm hội viên ${member.member_code}`,
          );
          onSaved?.(member);
          onClose();
        },
        onError: (error) => {
          const mapped = setFormError(form, error, {
            phone: /số điện thoại|sđt/i,
            dob: /ngày sinh/i,
            email: /email/i,
            home_branch_id: /chi nhánh/i,
            id_card: /cccd|căn cước/i,
            full_name: /họ tên/i,
          });
          if (!mapped) showError(error, 'Không thể lưu hội viên');
        },
      },
    );
  };

  return (
    <Modal
      open={open}
      title={isEdit ? `Sửa hội viên ${record?.member_code}` : 'Thêm hội viên'}
      okText={isEdit ? 'Lưu thay đổi' : 'Thêm hội viên'}
      cancelText={readOnly ? 'Đóng' : 'Hủy'}
      okButtonProps={{ style: readOnly ? { display: 'none' } : undefined }}
      onOk={() => form.submit()}
      onCancel={onClose}
      confirmLoading={save.isPending}
      width={820}
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
            <Form.Item name="avatar_url" label="Ảnh đại diện">
              <ImageUpload buttonText="Tải ảnh" />
            </Form.Item>
          </Col>
          <Col xs={24} sm={18}>
            <Row gutter={16}>
              <Col xs={24} sm={14}>
                <Form.Item
                  name="full_name"
                  label="Họ tên"
                  rules={[
                    { required: true, whitespace: true, message: 'Vui lòng nhập họ tên' },
                    { max: 150, message: 'Họ tên tối đa 150 ký tự' },
                  ]}
                >
                  <Input placeholder="VD: Nguyễn Văn An" maxLength={150} />
                </Form.Item>
              </Col>
              <Col xs={24} sm={10}>
                <Form.Item name="phone" label="Số điện thoại" required rules={[phoneRule(true)]}>
                  <Input placeholder="VD: 0912 345 678" maxLength={20} inputMode="tel" />
                </Form.Item>
              </Col>
              <Col xs={24} sm={14}>
                <Form.Item
                  name="email"
                  label="Email"
                  rules={[{ type: 'email', message: 'Email không hợp lệ' }]}
                >
                  <Input placeholder="VD: an@example.com" inputMode="email" />
                </Form.Item>
              </Col>
              <Col xs={12} sm={10}>
                <Form.Item name="gender" label="Giới tính">
                  <Select options={GENDER_OPTIONS} />
                </Form.Item>
              </Col>
            </Row>
          </Col>

          <Col xs={12} sm={8}>
            <Form.Item
              name="dob"
              label="Ngày sinh"
              rules={[
                {
                  validator: (_, v?: Dayjs | null) =>
                    !v || !v.isAfter(dayjs(), 'day')
                      ? Promise.resolve()
                      : Promise.reject(new Error('Ngày sinh không được ở tương lai')),
                },
              ]}
            >
              <DatePicker
                format={DATE_FORMAT}
                placeholder="dd/mm/yyyy"
                style={{ width: '100%' }}
                disabledDate={(d) => d.isAfter(dayjs(), 'day') || d.year() < 1900}
              />
            </Form.Item>
          </Col>
          <Col xs={12} sm={8}>
            <Form.Item
              name="id_card"
              label="CCCD"
              rules={[{ pattern: /^\d{9,12}$/, message: 'CCCD gồm 9–12 chữ số' }]}
            >
              <Input placeholder="012345678901" maxLength={12} inputMode="numeric" />
            </Form.Item>
          </Col>
          <Col xs={24} sm={8}>
            <Form.Item
              name="home_branch_id"
              label="Chi nhánh chính"
              rules={[{ required: true, message: 'Vui lòng chọn chi nhánh' }]}
            >
              <BranchSelect activeOnly={!isEdit} />
            </Form.Item>
          </Col>
          <Col xs={24}>
            <Form.Item name="address" label="Địa chỉ">
              <Input placeholder="Số nhà, đường, quận/huyện, tỉnh/thành" maxLength={255} />
            </Form.Item>
          </Col>

          <Col xs={24}>
            <Divider orientation="left" plain style={{ margin: '0 0 12px' }}>
              Người liên hệ khẩn cấp
            </Divider>
          </Col>
          <Col xs={24} sm={12}>
            <Form.Item name="emergency_contact_name" label="Họ tên người liên hệ">
              <Input placeholder="VD: Nguyễn Thị B" maxLength={150} />
            </Form.Item>
          </Col>
          <Col xs={24} sm={12}>
            <Form.Item
              name="emergency_contact_phone"
              label="SĐT người liên hệ"
              rules={[phoneRule(false)]}
            >
              <Input placeholder="VD: 0988 111 222" maxLength={20} inputMode="tel" />
            </Form.Item>
          </Col>

          <Col xs={12} sm={8}>
            <Form.Item name="source" label="Nguồn">
              <Select options={SOURCE_OPTIONS} />
            </Form.Item>
          </Col>
          <Col xs={12} sm={8}>
            <Form.Item name="status" label="Trạng thái">
              <Select options={MEMBER_STATUS_OPTIONS} />
            </Form.Item>
          </Col>
          <Col xs={24}>
            <Form.Item
              name="note"
              label="Ghi chú"
              rules={[{ max: 2000, message: 'Ghi chú tối đa 2000 ký tự' }]}
            >
              <Input.TextArea rows={2} maxLength={2000} placeholder="VD: Khách VIP, dị ứng..." />
            </Form.Item>
          </Col>
        </Row>
      </Form>
    </Modal>
  );
}
