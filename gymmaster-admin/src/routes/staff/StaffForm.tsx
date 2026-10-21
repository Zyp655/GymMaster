import { useEffect, useMemo, useState } from 'react';
import {
  Alert,
  App,
  Button,
  Col,
  DatePicker,
  Form,
  Input,
  InputNumber,
  Modal,
  Row,
  Select,
  Space,
  Tabs,
  Tag,
  Typography,
} from 'antd';
import { DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import roleApi from '@/api/role';
import type {
  Gender,
  RoleAssignment,
  Staff,
  StaffInput,
  StaffPosition,
  TrainerLevel,
} from '@/api/staff';
import { MoneyInput } from '@/components/Money';
import { PERM, ROLE, ROLE_LABELS } from '@/constants/permissions';
import { useBranchOptions } from '@/hooks/useBranchOptions';
import { usePermission } from '@/hooks/usePermission';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { getErrorMessage, isFormError, setFormError, showError } from '@/utils/errors';
import { DATE_FORMAT, toApiDate } from '@/utils/format';
import { Link } from 'react-router-dom';
import { isStaffGuardError } from './staffGuards';
import {
  GENDER_OPTIONS,
  POSITION_OPTIONS,
  SPECIALTY_SUGGESTIONS,
  TRAINER_LEVEL_OPTIONS,
} from './constants';
import { useSaveStaff } from './queries';

/** Giá trị 0 trong ô chi nhánh = toàn hệ thống (backend coi `branch_id = 0` như `null`). */
const GLOBAL = 0;

interface RoleRow {
  role_code?: string;
  branch_id?: number;
}

interface FormValues {
  full_name: string;
  email: string;
  phone?: string;
  password?: string;
  position: StaffPosition;
  gender: Gender;
  dob?: Dayjs | null;
  id_card?: string;
  address?: string;
  hire_date?: Dayjs | null;
  base_salary?: number | null;
  commission_rate?: number | null;
  note?: string;
  roles: RoleRow[];
  trainer_profile?: {
    level: TrainerLevel;
    specialties?: string[];
    years_experience?: number | null;
    certifications?: string;
    bio?: string;
  };
}

type TabKey = 'info' | 'roles' | 'trainer';

export interface StaffFormProps {
  open: boolean;
  /** `null` = thêm mới. */
  record: Staff | null;
  onClose: () => void;
}

const POSITION_TO_ROLE: Partial<Record<StaffPosition, string>> = {
  MANAGER: ROLE.MANAGER,
  RECEPTIONIST: ROLE.RECEPTIONIST,
  TRAINER: ROLE.TRAINER,
};

const roleKey = (r: { role_code?: string; branch_id?: number | null }) =>
  `${(r.role_code ?? '').toUpperCase()}@${r.branch_id || 0}`;

/** Danh sách vai trò có thể gán (không có MEMBER; OWNER chỉ khi người gán là OWNER). */
function useAssignableRoles(isOwner: boolean, enabled: boolean) {
  const { can } = usePermission();
  const canViewRoles = can(PERM.ROLE_VIEW);
  const roles = useQuery({
    queryKey: [roleApi.key, 'list'],
    queryFn: () => roleApi.list(),
    enabled: enabled && canViewRoles,
    staleTime: 5 * 60_000,
  });
  return useMemo(() => {
    const source = canViewRoles
      ? (roles.data ?? []).map((r) => ({ code: r.code, name: r.name }))
      : [ROLE.OWNER, ROLE.MANAGER, ROLE.RECEPTIONIST, ROLE.TRAINER].map((code) => ({
          code,
          name: ROLE_LABELS[code],
        }));
    return {
      loading: roles.isLoading && canViewRoles,
      nameOf: (code: string) =>
        source.find((r) => r.code === code)?.name ?? ROLE_LABELS[code] ?? code,
      options: source
        .filter((r) => r.code !== ROLE.MEMBER && (isOwner || r.code !== ROLE.OWNER))
        .map((r) => ({ value: r.code, label: r.name })),
    };
  }, [roles.data, roles.isLoading, canViewRoles, isOwner]);
}

/** Modal thêm/sửa nhân viên: Thông tin · Vai trò · Hồ sơ HLV (validate giống api/staff/v1/staff.go). */
export default function StaffForm({ open, record, onClose }: StaffFormProps) {
  const [form] = Form.useForm<FormValues>();
  const { message } = App.useApp();
  const { can, isOwner, user } = usePermission();
  const { branchId } = useWorkingBranch();
  const { branches } = useBranchOptions();
  const save = useSaveStaff();
  const [tab, setTab] = useState<TabKey>('info');
  const [rolesError, setRolesError] = useState<string | null>(null);
  /** Lỗi chặn thao tác từ API (403 quyền cao hơn / 400 chính mình) – hiện đầu modal. */
  const [guardError, setGuardError] = useState<string | null>(null);
  const roleOptions = useAssignableRoles(isOwner, open);

  const isEdit = !!record;
  const isSelf = !!record && record.id === user?.id;
  // API từ chối sửa tài khoản của chính mình → chỉ xem (sửa tại trang Hồ sơ cá nhân).
  const readOnly = isEdit ? !can(PERM.STAFF_UPDATE) || isSelf : !can(PERM.STAFF_CREATE);
  const position = Form.useWatch('position', form);
  const isTrainer = position === 'TRAINER';
  const allowedBranches = useMemo(() => user?.branch_ids ?? [], [user]);
  // Người không phải OWNER không được sửa tài khoản OWNER.
  const targetIsOwner = !!record?.roles.some((r) => r.role_code === ROLE.OWNER);
  const blocked = isEdit && !isOwner && targetIsOwner;

  /** Vai trò người gán không được động vào (backend giữ nguyên) – chỉ hiển thị. */
  const lockedRoles = useMemo(() => {
    if (!record || isOwner) return [];
    return record.roles.filter(
      (r) =>
        r.role_code === ROLE.OWNER ||
        r.branch_id === null ||
        (allowedBranches.length > 0 && !allowedBranches.includes(r.branch_id)),
    );
  }, [record, isOwner, allowedBranches]);

  const branchOptions = useMemo(
    () => [
      ...(isOwner ? [{ label: 'Toàn hệ thống', value: GLOBAL }] : []),
      ...branches.map((b) => ({ label: b.name, value: b.id })),
    ],
    [isOwner, branches],
  );

  useEffect(() => {
    if (!open) return;
    setTab('info');
    setRolesError(null);
    setGuardError(null);
    form.resetFields();
    if (record) {
      const p = record.staff_profile;
      const t = record.trainer_profile;
      const locked = new Set(lockedRoles.map(roleKey));
      form.setFieldsValue({
        full_name: record.full_name,
        email: record.email,
        phone: record.phone,
        position: p?.position,
        gender: p?.gender ?? 'OTHER',
        dob: p?.dob ? dayjs(p.dob) : null,
        id_card: p?.id_card,
        address: p?.address,
        hire_date: p?.hire_date ? dayjs(p.hire_date) : null,
        base_salary: p?.base_salary ?? 0,
        commission_rate: p?.commission_rate ?? 0,
        note: p?.note,
        roles: record.roles
          .filter((r) => !locked.has(roleKey(r)))
          .map((r) => ({ role_code: r.role_code, branch_id: r.branch_id ?? GLOBAL })),
        trainer_profile: {
          level: t?.level ?? 'JUNIOR',
          specialties: t?.specialties ?? [],
          years_experience: t?.years_experience ?? 0,
          certifications: t?.certifications ?? '',
          bio: t?.bio ?? '',
        },
      });
    } else {
      const defaultBranch =
        branchId ?? (isOwner ? GLOBAL : branches.length === 1 ? branches[0].id : undefined);
      form.setFieldsValue({
        gender: 'OTHER',
        hire_date: dayjs(),
        base_salary: 0,
        commission_rate: 0,
        roles: [{ role_code: undefined, branch_id: defaultBranch }],
        trainer_profile: { level: 'JUNIOR', specialties: [], years_experience: 0 },
      });
    }
    // Chỉ khởi tạo lại khi mở modal / đổi bản ghi.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, record, form]);

  /** Tạo mới: chọn chức vụ → tự điền vai trò tương ứng cho dòng còn trống. */
  const onValuesChange = (changed: Partial<FormValues>) => {
    if (isEdit || !changed.position) return;
    const role = POSITION_TO_ROLE[changed.position];
    if (!role) return;
    const rows = form.getFieldValue('roles') as RoleRow[];
    if (rows.length > 0 && !rows[0].role_code) {
      form.setFieldValue(['roles', 0, 'role_code'], role);
    }
  };

  const tabOfField = (name: (string | number)[]): TabKey =>
    name[0] === 'roles' ? 'roles' : name[0] === 'trainer_profile' ? 'trainer' : 'info';

  const buildRoles = (rows: RoleRow[]): RoleAssignment[] =>
    rows
      .filter((r) => r.role_code)
      .map((r) => ({
        role_code: r.role_code!,
        branch_id: r.role_code === ROLE.OWNER || !r.branch_id ? null : r.branch_id,
      }));

  const onFinish = (v: FormValues) => {
    setRolesError(null);
    setGuardError(null);
    const t = v.trainer_profile;
    const info: StaffInput = {
      full_name: v.full_name.trim(),
      email: v.email.trim().toLowerCase(),
      phone: v.phone?.trim() ?? '',
      position: v.position,
      gender: v.gender,
      dob: toApiDate(v.dob) ?? '',
      id_card: v.id_card?.trim() ?? '',
      address: v.address?.trim() ?? '',
      hire_date: toApiDate(v.hire_date) ?? undefined,
      base_salary: v.base_salary ?? 0,
      commission_rate: v.commission_rate ?? 0,
      note: v.note?.trim() ?? '',
      trainer_profile:
        v.position === 'TRAINER' && t
          ? {
              level: t.level,
              specialties: (t.specialties ?? []).map((s) => s.trim()).filter(Boolean),
              years_experience: t.years_experience ?? 0,
              certifications: t.certifications?.trim() ?? '',
              bio: t.bio?.trim() ?? '',
            }
          : null,
    };
    const roles = buildRoles(v.roles ?? []);

    const onError = (error: unknown) => {
      const step = (error as { step?: string }).step;
      const msg = getErrorMessage(error);
      // 403/400 "chính mình": không gắn vào ô nhập theo từ khóa – hiện nguyên thông điệp server.
      if (isStaffGuardError(error)) {
        setGuardError(
          step === 'roles' ? `Đã lưu thông tin nhưng chưa gán được vai trò: ${msg}` : msg,
        );
        return;
      }
      if (step === 'roles' || (isFormError(error) && /vai trò|chi nhánh/i.test(msg))) {
        setRolesError(
          step === 'roles' ? `Đã lưu thông tin nhưng chưa gán được vai trò: ${msg}` : msg,
        );
        setTab('roles');
        return;
      }
      const mapped = setFormError(form, error, {
        email: /email/i,
        full_name: /họ tên/i,
        phone: /điện thoại/i,
        password: /mật khẩu/i,
        dob: /ngày sinh/i,
        hire_date: /ngày vào làm/i,
        id_card: /cccd/i,
        address: /địa chỉ/i,
        base_salary: /lương/i,
        commission_rate: /hoa hồng/i,
        note: /ghi chú/i,
      });
      if (mapped) {
        setTab('info');
        return;
      }
      if (isFormError(error) && /hlv|kinh nghiệm|giới thiệu|chứng chỉ/i.test(msg)) {
        setTab('trainer');
      }
      showError(error, 'Không thể lưu nhân viên');
    };

    const onSuccess = () => {
      message.success(isEdit ? 'Đã cập nhật nhân viên' : 'Đã thêm nhân viên');
      onClose();
    };

    if (!record) {
      save.mutate(
        { create: { ...info, password: v.password ?? '', roles } },
        { onSuccess, onError },
      );
      return;
    }
    // Chỉ gửi gán vai trò khi có thay đổi (và không phải chính mình – backend chặn tự đổi vai trò).
    const before = record.roles
      .filter((r) => !lockedRoles.some((l) => roleKey(l) === roleKey(r)))
      .map(roleKey)
      .sort()
      .join(',');
    const after = roles.map(roleKey).sort().join(',');
    const rolesChanged = before !== after;
    save.mutate(
      { id: record.id, update: info, roles: rolesChanged ? roles : undefined },
      { onSuccess, onError },
    );
  };

  const items = [
    {
      key: 'info',
      label: 'Thông tin',
      forceRender: true,
      children: (
        <Row gutter={16}>
          <Col xs={24} sm={12}>
            <Form.Item
              name="full_name"
              label="Họ tên"
              rules={[
                { required: true, whitespace: true, message: 'Vui lòng nhập họ tên' },
                { max: 150, message: 'Họ tên tối đa 150 ký tự' },
              ]}
            >
              <Input maxLength={150} placeholder="VD: Nguyễn Văn An" />
            </Form.Item>
          </Col>
          <Col xs={24} sm={12}>
            <Form.Item
              name="position"
              label="Chức vụ"
              rules={[{ required: true, message: 'Vui lòng chọn chức vụ' }]}
              extra={
                isEdit && record?.trainer_profile && position && position !== 'TRAINER' ? (
                  <Typography.Text type="warning">
                    Đổi khỏi Huấn luyện viên sẽ xóa hồ sơ HLV.
                  </Typography.Text>
                ) : undefined
              }
            >
              <Select options={POSITION_OPTIONS} placeholder="Chọn chức vụ" />
            </Form.Item>
          </Col>
          <Col xs={24} sm={12}>
            <Form.Item
              name="email"
              label="Email đăng nhập"
              rules={[
                { required: true, message: 'Vui lòng nhập email' },
                { type: 'email', message: 'Email không hợp lệ' },
                { max: 150, message: 'Email tối đa 150 ký tự' },
              ]}
            >
              <Input inputMode="email" autoComplete="off" placeholder="VD: an.nv@gymmaster.vn" />
            </Form.Item>
          </Col>
          <Col xs={24} sm={12}>
            <Form.Item
              name="phone"
              label="Số điện thoại"
              rules={[{ pattern: /^[0-9+ ]{8,20}$/, message: 'Số điện thoại không hợp lệ' }]}
            >
              <Input inputMode="tel" maxLength={20} placeholder="VD: 0912345678" />
            </Form.Item>
          </Col>
          {!isEdit && (
            <Col xs={24} sm={12}>
              <Form.Item
                name="password"
                label="Mật khẩu ban đầu"
                tooltip="Nhân viên dùng mật khẩu này để đăng nhập lần đầu."
                rules={[
                  { required: true, message: 'Vui lòng nhập mật khẩu ban đầu' },
                  { min: 8, max: 72, message: 'Mật khẩu từ 8 đến 72 ký tự' },
                ]}
              >
                <Input.Password autoComplete="new-password" maxLength={72} />
              </Form.Item>
            </Col>
          )}
          <Col xs={12} sm={isEdit ? 8 : 6}>
            <Form.Item name="gender" label="Giới tính">
              <Select options={GENDER_OPTIONS} />
            </Form.Item>
          </Col>
          <Col xs={12} sm={isEdit ? 8 : 6}>
            <Form.Item name="dob" label="Ngày sinh">
              <DatePicker
                format={DATE_FORMAT}
                style={{ width: '100%' }}
                disabledDate={(d) => !d.isBefore(dayjs(), 'day')}
                placeholder="Chọn ngày"
              />
            </Form.Item>
          </Col>
          <Col xs={24} sm={isEdit ? 8 : 12}>
            <Form.Item
              name="id_card"
              label="Số CCCD"
              rules={[{ max: 20, message: 'Số CCCD tối đa 20 ký tự' }]}
            >
              <Input maxLength={20} inputMode="numeric" />
            </Form.Item>
          </Col>
          <Col xs={24} sm={isEdit ? 24 : 12}>
            <Form.Item
              name="address"
              label="Địa chỉ"
              rules={[{ max: 255, message: 'Địa chỉ tối đa 255 ký tự' }]}
            >
              <Input maxLength={255} />
            </Form.Item>
          </Col>
          <Col xs={24} sm={8}>
            <Form.Item name="hire_date" label="Ngày vào làm" tooltip="Bỏ trống = hôm nay">
              <DatePicker format={DATE_FORMAT} style={{ width: '100%' }} placeholder="Chọn ngày" />
            </Form.Item>
          </Col>
          <Col xs={14} sm={10}>
            <Form.Item name="base_salary" label="Lương cơ bản">
              <MoneyInput />
            </Form.Item>
          </Col>
          <Col xs={10} sm={6}>
            <Form.Item
              name="commission_rate"
              label="Hoa hồng"
              rules={[{ type: 'number', min: 0, max: 100, message: 'Tỷ lệ hoa hồng từ 0 đến 100' }]}
            >
              <InputNumber min={0} max={100} step={0.5} suffix="%" style={{ width: '100%' }} />
            </Form.Item>
          </Col>
          <Col xs={24}>
            <Form.Item
              name="note"
              label="Ghi chú"
              rules={[{ max: 2000, message: 'Ghi chú tối đa 2000 ký tự' }]}
            >
              <Input.TextArea rows={2} maxLength={2000} />
            </Form.Item>
          </Col>
        </Row>
      ),
    },
    {
      key: 'roles',
      label: 'Vai trò',
      forceRender: true,
      children: (
        <>
          {rolesError && (
            <Alert type="error" showIcon message={rolesError} style={{ marginBottom: 12 }} />
          )}
          {isSelf && (
            <Alert
              type="warning"
              showIcon
              style={{ marginBottom: 12 }}
              message="Bạn không thể tự thay đổi vai trò của chính mình."
            />
          )}
          {lockedRoles.length > 0 && (
            <div style={{ marginBottom: 12 }}>
              <Typography.Text type="secondary">
                Vai trò ngoài phạm vi của bạn (giữ nguyên):{' '}
              </Typography.Text>
              {lockedRoles.map((r) => (
                <Tag key={roleKey(r)} bordered={false}>
                  {r.role_name} · {r.branch_name || 'Toàn hệ thống'}
                </Tag>
              ))}
            </div>
          )}
          <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
            Mỗi dòng là một vai trò tại một chi nhánh. Nhân viên chỉ thao tác được dữ liệu của chi
            nhánh được gán.
            {isOwner && ' Chọn "Toàn hệ thống" để không giới hạn chi nhánh.'}
          </Typography.Paragraph>
          <Form.List
            name="roles"
            rules={[
              {
                validator: async (_, rows?: RoleRow[]) => {
                  if (!rows || rows.length === 0)
                    throw new Error('Vui lòng gán ít nhất một vai trò');
                },
              },
            ]}
          >
            {(fields, { add, remove }, { errors }) => (
              <>
                {fields.map((field) => (
                  <Row gutter={8} key={field.key} align="top" wrap={false}>
                    <Col flex="1 1 45%">
                      <Form.Item
                        name={[field.name, 'role_code']}
                        rules={[{ required: true, message: 'Chọn vai trò' }]}
                      >
                        <Select
                          placeholder="Vai trò"
                          options={roleOptions.options}
                          loading={roleOptions.loading}
                          disabled={isSelf}
                          onChange={(code: string) => {
                            if (code === ROLE.OWNER)
                              form.setFieldValue(['roles', field.name, 'branch_id'], GLOBAL);
                          }}
                        />
                      </Form.Item>
                    </Col>
                    <Col flex="1 1 45%">
                      <Form.Item
                        noStyle
                        shouldUpdate={(a, b) =>
                          a.roles?.[field.name]?.role_code !== b.roles?.[field.name]?.role_code
                        }
                      >
                        {({ getFieldValue }) => {
                          const isOwnerRole =
                            getFieldValue(['roles', field.name, 'role_code']) === ROLE.OWNER;
                          return (
                            <Form.Item
                              name={[field.name, 'branch_id']}
                              rules={[
                                {
                                  validator: async (_, value?: number) => {
                                    if (isOwnerRole) return;
                                    if (value === undefined || value === null)
                                      throw new Error('Chọn chi nhánh');
                                    if (!isOwner && !value)
                                      throw new Error('Vui lòng chọn chi nhánh cụ thể');
                                  },
                                },
                              ]}
                            >
                              <Select
                                placeholder="Chi nhánh"
                                options={
                                  isOwnerRole
                                    ? [{ label: 'Toàn hệ thống', value: GLOBAL }]
                                    : branchOptions
                                }
                                disabled={isOwnerRole || isSelf}
                                showSearch
                                optionFilterProp="label"
                              />
                            </Form.Item>
                          );
                        }}
                      </Form.Item>
                    </Col>
                    <Col flex="none">
                      <Button
                        type="text"
                        danger
                        icon={<DeleteOutlined />}
                        aria-label="Xóa dòng"
                        disabled={fields.length <= 1 || isSelf}
                        onClick={() => remove(field.name)}
                      />
                    </Col>
                  </Row>
                ))}
                <Form.ErrorList errors={errors} />
                <Button
                  type="dashed"
                  block
                  icon={<PlusOutlined />}
                  disabled={isSelf}
                  onClick={() =>
                    add({
                      role_code: undefined,
                      branch_id: branchId ?? (isOwner ? GLOBAL : branches[0]?.id),
                    })
                  }
                >
                  Thêm vai trò
                </Button>
              </>
            )}
          </Form.List>
        </>
      ),
    },
    ...(isTrainer
      ? [
          {
            key: 'trainer',
            label: 'Hồ sơ HLV',
            forceRender: true,
            children: (
              <Row gutter={16}>
                <Col xs={12} sm={8}>
                  <Form.Item name={['trainer_profile', 'level']} label="Cấp độ">
                    <Select options={TRAINER_LEVEL_OPTIONS} />
                  </Form.Item>
                </Col>
                <Col xs={12} sm={8}>
                  <Form.Item
                    name={['trainer_profile', 'years_experience']}
                    label="Số năm kinh nghiệm"
                    rules={[
                      {
                        type: 'number',
                        min: 0,
                        max: 60,
                        message: 'Số năm kinh nghiệm từ 0 đến 60',
                      },
                    ]}
                  >
                    <InputNumber min={0} max={60} precision={0} style={{ width: '100%' }} />
                  </Form.Item>
                </Col>
                <Col xs={24}>
                  <Form.Item
                    name={['trainer_profile', 'specialties']}
                    label="Chuyên môn"
                    tooltip="Gõ rồi nhấn Enter để thêm chuyên môn mới."
                  >
                    <Select
                      mode="tags"
                      placeholder="VD: Yoga, Giảm cân"
                      options={SPECIALTY_SUGGESTIONS.map((s) => ({ label: s, value: s }))}
                      tokenSeparators={[',']}
                    />
                  </Form.Item>
                </Col>
                <Col xs={24}>
                  <Form.Item
                    name={['trainer_profile', 'certifications']}
                    label="Chứng chỉ"
                    rules={[{ max: 2000, message: 'Chứng chỉ tối đa 2000 ký tự' }]}
                  >
                    <Input.TextArea rows={2} maxLength={2000} placeholder="VD: ACE CPT, NASM" />
                  </Form.Item>
                </Col>
                <Col xs={24}>
                  <Form.Item
                    name={['trainer_profile', 'bio']}
                    label="Giới thiệu"
                    rules={[{ max: 2000, message: 'Giới thiệu tối đa 2000 ký tự' }]}
                  >
                    <Input.TextArea rows={3} maxLength={2000} showCount />
                  </Form.Item>
                </Col>
              </Row>
            ),
          },
        ]
      : []),
  ];

  return (
    <Modal
      open={open}
      title={
        isEdit ? (
          <Space size={8}>
            {record?.full_name}
            {record?.staff_profile?.staff_code && (
              <Tag bordered={false} className="code-text">
                {record.staff_profile.staff_code}
              </Tag>
            )}
          </Space>
        ) : (
          'Thêm nhân viên'
        )
      }
      okText={isEdit ? 'Lưu thay đổi' : 'Thêm nhân viên'}
      cancelText={readOnly || blocked ? 'Đóng' : 'Hủy'}
      okButtonProps={{ style: readOnly || blocked ? { display: 'none' } : undefined }}
      onOk={() => form.submit()}
      onCancel={onClose}
      confirmLoading={save.isPending}
      width={760}
      forceRender
    >
      {blocked && (
        <Alert
          type="info"
          showIcon
          style={{ marginTop: 12 }}
          message="Chỉ Chủ trung tâm mới được sửa tài khoản Chủ trung tâm."
        />
      )}
      {isSelf && (
        <Alert
          type="info"
          showIcon
          style={{ marginTop: 12 }}
          message={
            <>
              Đây là tài khoản của bạn. Cập nhật thông tin cá nhân, mật khẩu tại trang{' '}
              <Link to="/app/profile" onClick={onClose}>
                Hồ sơ cá nhân
              </Link>
              .
            </>
          }
        />
      )}
      {guardError && <Alert type="error" showIcon style={{ marginTop: 12 }} message={guardError} />}
      <Form<FormValues>
        form={form}
        layout="vertical"
        disabled={readOnly || blocked}
        onFinish={onFinish}
        onValuesChange={onValuesChange}
        onFinishFailed={({ errorFields }) => {
          if (errorFields[0]) setTab(tabOfField(errorFields[0].name));
        }}
        style={{ marginTop: 4 }}
      >
        <Tabs activeKey={tab} onChange={(k) => setTab(k as TabKey)} items={items} />
      </Form>
    </Modal>
  );
}
