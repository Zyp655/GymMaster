import { useEffect, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Alert,
  App,
  Button,
  Card,
  Col,
  DatePicker,
  Descriptions,
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
import type { ColumnsType } from 'antd/es/table';
import {
  CopyOutlined,
  DeleteOutlined,
  EditOutlined,
  EyeOutlined,
  FileProtectOutlined,
  FileTextOutlined,
  QrcodeOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import {
  contractsPassesApi,
  DAY_PASS_TYPE_LABELS,
  DAY_PASS_TYPE_OPTIONS,
} from '@/api/operations-ext';
import type {
  ContractStatus,
  ContractTemplate,
  ContractTemplateInput,
  DayPass,
  DayPassInput,
  DayPassType,
  MemberContract,
} from '@/api/operations-ext';
import ActionBar from '@/components/ActionBar';
import BranchSelect from '@/components/BranchSelect';
import ConfirmButton from '@/components/ConfirmButton';
import DataTable from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import MemberSelect from '@/components/MemberSelect';
import MoneyInput from '@/components/Money/MoneyInput';
import MoneyText from '@/components/Money/MoneyText';
import PageTitle from '@/components/PageTitle';
import StatusTag from '@/components/StatusTag';
import { PERM } from '@/constants/permissions';
import {
  ACTIVE_STATUS_OPTIONS,
  CONTRACT_STATUS_MAP,
  CONTRACT_STATUS_OPTIONS,
  COUNTER_PAYMENT_METHOD_OPTIONS,
} from '@/constants/status';
import { useBranchOptions } from '@/hooks/useBranchOptions';
import { usePermission } from '@/hooks/usePermission';
import { useTableQuery } from '@/hooks/useTableQuery';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { showError } from '@/utils/errors';
import { isSafeSignatureSrc, safeHtml } from './sanitizeHtml';
import { DATE_FORMAT, formatDate, formatDateTime, formatPhone, toApiDate } from '@/utils/format';

/* --------------------------------------------------------------------------
 * TAB 1: Mẫu hợp đồng điện tử (/admin/contract-templates)
 * -------------------------------------------------------------------------- */
function ContractTemplatesTab({ canManage }: { canManage: boolean }) {
  const { message } = App.useApp();
  const qc = useQueryClient();
  const [modal, setModal] = useState<{ open: boolean; record: ContractTemplate | null }>({
    open: false,
    record: null,
  });
  const [previewItem, setPreviewItem] = useState<ContractTemplate | null>(null);
  const [form] = Form.useForm<ContractTemplateInput>();

  const table = useTableQuery<ContractTemplate, { status?: string; page?: number; limit?: number }>(
    {
      queryKey: [contractsPassesApi.templateKey, 'list'],
      queryFn: (p) => contractsPassesApi.listTemplates(p),
      filterKeys: ['status'],
      defaults: { order_by: 'id', order_dir: 'DESC' },
    },
  );

  useEffect(() => {
    if (!modal.open) return;
    form.resetFields();
    if (modal.record) {
      form.setFieldsValue(modal.record);
    } else {
      form.setFieldsValue({
        version: 1,
        status: 'ACTIVE',
        body_html:
          '<h3>HỢP ĐỒNG HỘI VIÊN GYMMASTER</h3><p>Họ và tên hội viên: <strong>{{member_name}}</strong> ({{member_code}})</p><p>Số điện thoại: <strong>{{member_phone}}</strong></p><p>Gói tập đăng ký: <strong>{{plan_name}}</strong></p><p>Tôi xác nhận đủ điều kiện sức khỏe tham gia tập luyện và đồng ý tuân thủ nội quy phòng tập GymMaster.</p>',
      });
    }
  }, [modal, form]);

  const saveMut = useMutation({
    mutationFn: (v: ContractTemplateInput) => {
      const payload: ContractTemplateInput = {
        code: v.code.trim().toUpperCase(),
        name: v.name.trim(),
        body_html: v.body_html,
        version: Number(v.version) || 1,
        status: v.status ?? 'ACTIVE',
      };
      return modal.record
        ? contractsPassesApi.updateTemplate(modal.record.id, payload)
        : contractsPassesApi.createTemplate(payload);
    },
    onSuccess: () => {
      message.success(
        modal.record ? 'Đã cập nhật mẫu hợp đồng' : 'Đã tạo mẫu hợp đồng điện tử mới',
      );
      void qc.invalidateQueries({ queryKey: [contractsPassesApi.templateKey] });
      setModal({ open: false, record: null });
    },
    onError: (err) => showError(err, 'Không thể lưu mẫu hợp đồng'),
  });

  const deleteMut = useMutation({
    mutationFn: (id: number) => contractsPassesApi.deleteTemplate(id),
    onSuccess: () => {
      message.success('Đã xóa mẫu hợp đồng');
      void qc.invalidateQueries({ queryKey: [contractsPassesApi.templateKey] });
    },
    onError: (err) => showError(err, 'Không thể xóa mẫu hợp đồng'),
  });

  const columns = useMemo<ColumnsType<ContractTemplate>>(
    () => [
      {
        title: 'Mã mẫu',
        dataIndex: 'code',
        width: 170,
        render: (c: string) => <Tag className="code-text">{c}</Tag>,
      },
      {
        title: 'Tên mẫu hợp đồng / Cam kết',
        dataIndex: 'name',
        render: (v: string) => <Typography.Text strong>{v}</Typography.Text>,
      },
      {
        title: 'Phiên bản',
        dataIndex: 'version',
        width: 110,
        render: (v: number) => <Tag color="blue">v{v || 1}</Tag>,
      },
      {
        title: 'Trạng thái',
        dataIndex: 'status',
        width: 150,
        render: (s: string) => <StatusTag status={s} />,
      },
      {
        title: '',
        key: 'actions',
        width: 150,
        render: (_, r) => (
          <Space size={4}>
            <Button
              size="small"
              type="text"
              icon={<EyeOutlined />}
              onClick={() => setPreviewItem(r)}
            >
              Xem mẫu
            </Button>
            {canManage && (
              <>
                <Button
                  type="text"
                  size="small"
                  icon={<EditOutlined />}
                  onClick={() => setModal({ open: true, record: r })}
                />
                <ConfirmButton
                  perm={PERM.CONTRACT_MANAGE}
                  type="text"
                  size="small"
                  danger
                  icon={<DeleteOutlined />}
                  confirmTitle={`Xóa mẫu hợp đồng "${r.name}"?`}
                  onConfirm={() => deleteMut.mutateAsync(r.id)}
                  aria-label="Xóa"
                />
              </>
            )}
          </Space>
        ),
      },
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [canManage],
  );

  return (
    <>
      <Row justify="space-between" align="middle" style={{ marginBottom: 12 }}>
        <Col />
        <Col>
          <ActionBar
            onAdd={() => setModal({ open: true, record: null })}
            addPerm={PERM.CONTRACT_MANAGE}
            addText="Thêm mẫu hợp đồng"
            onRefresh={() => table.refetch()}
            refreshing={table.isFetching}
          />
        </Col>
      </Row>

      <FilterBar
        searchPlaceholder="Tìm theo mã, tên mẫu hợp đồng..."
        colTextSearch={6}
        filters={[
          {
            name: 'status',
            placeholder: 'Trạng thái',
            options: ACTIVE_STATUS_OPTIONS,
            col: 4,
          },
        ]}
      />

      <DataTable<ContractTemplate>
        table={table}
        columns={columns}
        emptyText="Chưa có mẫu hợp đồng điện tử nào"
      />

      <Modal
        open={modal.open}
        title={
          modal.record ? `Sửa mẫu hợp đồng: ${modal.record.code}` : 'Tạo mẫu hợp đồng điện tử mới'
        }
        okText="Lưu mẫu hợp đồng"
        cancelText="Hủy"
        onOk={() => form.submit()}
        onCancel={() => setModal({ open: false, record: null })}
        confirmLoading={saveMut.isPending}
        width={680}
        forceRender
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={(v) => saveMut.mutate(v)}
          style={{ marginTop: 12 }}
        >
          <Alert
            type="info"
            showIcon
            style={{ marginBottom: 12 }}
            message="Biến tự động trộn (Template Variables)"
            description="Hỗ trợ các biến: {{member_name}}, {{member_code}}, {{member_phone}}, {{plan_name}} trong nội dung HTML."
          />
          <Row gutter={12}>
            <Col xs={24} sm={9}>
              <Form.Item
                name="code"
                label="Mã mẫu hợp đồng"
                rules={[{ required: true, whitespace: true, message: 'Nhập mã mẫu' }]}
              >
                <Input placeholder="VD: MEMBER_TERMS_V1" disabled={!!modal.record} />
              </Form.Item>
            </Col>
            <Col xs={24} sm={15}>
              <Form.Item
                name="name"
                label="Tên mẫu hợp đồng"
                rules={[{ required: true, whitespace: true, message: 'Nhập tên mẫu' }]}
              >
                <Input placeholder="VD: Hợp đồng Hội viên & Cam kết An toàn Tập luyện" />
              </Form.Item>
            </Col>
            <Col xs={12}>
              <Form.Item name="version" label="Phiên bản">
                <InputNumber min={1} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col xs={12}>
              <Form.Item name="status" label="Trạng thái">
                <Select options={ACTIVE_STATUS_OPTIONS} />
              </Form.Item>
            </Col>
            <Col xs={24}>
              <Form.Item
                name="body_html"
                label="Nội dung điều khoản hợp đồng (HTML)"
                rules={[{ required: true, whitespace: true, message: 'Nhập nội dung hợp đồng' }]}
              >
                <Input.TextArea rows={7} className="code-text" />
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Modal>

      <Modal
        open={!!previewItem}
        title={`Xem trước mẫu hợp đồng: ${previewItem?.name}`}
        footer={<Button onClick={() => setPreviewItem(null)}>Đóng</Button>}
        onCancel={() => setPreviewItem(null)}
        width={680}
      >
        {previewItem && (
          <Card size="small" style={{ background: '#f8fafc', marginTop: 12 }}>
            <div dangerouslySetInnerHTML={{ __html: safeHtml(previewItem.body_html) }} />
          </Card>
        )}
      </Modal>
    </>
  );
}

/* --------------------------------------------------------------------------
 * TAB 2: Hợp đồng hội viên đã ký (/admin/member-contracts)
 * -------------------------------------------------------------------------- */
function MemberContractsTab() {
  const { message } = App.useApp();
  const qc = useQueryClient();
  const [createOpen, setCreateOpen] = useState(false);
  const [viewContract, setViewContract] = useState<MemberContract | null>(null);
  const [form] = Form.useForm<{ member_id: number; template_id: number }>();

  const templatesQuery = useQuery({
    queryKey: [contractsPassesApi.templateKey, 'options'],
    queryFn: () => contractsPassesApi.listTemplates({ status: 'ACTIVE', limit: 100 }),
    enabled: createOpen,
  });

  const table = useTableQuery<
    MemberContract,
    { status?: ContractStatus; page?: number; limit?: number }
  >({
    queryKey: [contractsPassesApi.memberContractKey, 'list'],
    queryFn: (p) => contractsPassesApi.listMemberContracts(p),
    filterKeys: ['status'],
    defaults: { order_by: 'id', order_dir: 'DESC' },
  });

  const createMut = useMutation({
    mutationFn: (v: { member_id: number; template_id: number }) =>
      contractsPassesApi.createMemberContract(v),
    onSuccess: () => {
      message.success('Đã phát hành hợp đồng điện tử chờ hội viên ký!');
      void qc.invalidateQueries({ queryKey: [contractsPassesApi.memberContractKey] });
      setCreateOpen(false);
    },
    onError: (err) => showError(err, 'Không thể tạo hợp đồng hội viên'),
  });

  const columns = useMemo<ColumnsType<MemberContract>>(
    () => [
      {
        title: 'Mã HĐ',
        dataIndex: 'id',
        width: 90,
        render: (id: number) => <span className="code-text">CT-#{id}</span>,
      },
      {
        title: 'Hội viên',
        key: 'member',
        render: (_, r) => (
          <Space direction="vertical" size={0}>
            <Typography.Text strong>{r.member_name || `Hội viên #${r.member_id}`}</Typography.Text>
            {r.member_code && (
              <Typography.Text type="secondary" className="code-text" style={{ fontSize: 12 }}>
                {r.member_code}
              </Typography.Text>
            )}
          </Space>
        ),
      },
      {
        title: 'Mẫu hợp đồng',
        key: 'template',
        render: (_, r) => (
          <Space>
            <span>{r.template_name || r.template_code || `Mẫu #${r.template_id}`}</span>
            <Tag>v{r.template_version || 1}</Tag>
          </Space>
        ),
      },
      {
        title: 'Trạng thái ký',
        dataIndex: 'status',
        render: (s: ContractStatus) => <StatusTag status={s} map={CONTRACT_STATUS_MAP} />,
      },
      {
        title: 'Thời điểm ký',
        dataIndex: 'signed_at',
        render: (v: string | null | undefined) => (v ? formatDateTime(v) : 'Chưa ký'),
      },
      {
        title: 'IP ký số',
        dataIndex: 'ip',
        render: (v: string) => v || '—',
      },
      {
        title: '',
        key: 'actions',
        width: 120,
        render: (_, r) => (
          <Button
            size="small"
            type="link"
            icon={<EyeOutlined />}
            onClick={() => setViewContract(r)}
          >
            Xem bản ký
          </Button>
        ),
      },
    ],
    [],
  );

  return (
    <>
      <Row justify="space-between" align="middle" style={{ marginBottom: 12 }}>
        <Col />
        <Col>
          <ActionBar
            onAdd={() => {
              form.resetFields();
              setCreateOpen(true);
            }}
            addPerm={PERM.CONTRACT_MANAGE}
            addText="Phát hành hợp đồng cho HV"
            onRefresh={() => table.refetch()}
            refreshing={table.isFetching}
          />
        </Col>
      </Row>

      <FilterBar
        searchPlaceholder="Tìm theo tên hoặc mã hội viên..."
        colTextSearch={6}
        filters={[
          {
            name: 'status',
            placeholder: 'Trạng thái ký',
            options: CONTRACT_STATUS_OPTIONS,
            col: 5,
          },
        ]}
      />

      <DataTable<MemberContract>
        table={table}
        columns={columns}
        emptyText="Chưa có hợp đồng hội viên nào"
      />

      <Modal
        open={createOpen}
        title="Phát hành Hợp đồng điện tử cho Hội viên"
        okText="Phát hành hợp đồng"
        cancelText="Hủy"
        onOk={() => form.submit()}
        onCancel={() => setCreateOpen(false)}
        confirmLoading={createMut.isPending}
        width={500}
        forceRender
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={(v) => createMut.mutate(v)}
          style={{ marginTop: 12 }}
        >
          <Form.Item
            name="member_id"
            label="Chọn Hội viên"
            rules={[{ required: true, message: 'Vui lòng chọn hội viên' }]}
          >
            <MemberSelect />
          </Form.Item>
          <Form.Item
            name="template_id"
            label="Chọn Mẫu hợp đồng điện tử"
            rules={[{ required: true, message: 'Vui lòng chọn mẫu hợp đồng' }]}
          >
            <Select
              placeholder="Chọn mẫu hợp đồng"
              options={(templatesQuery.data?.items ?? []).map((t) => ({
                value: t.id,
                label: `${t.code} – ${t.name} (v${t.version})`,
              }))}
            />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        open={!!viewContract}
        title={`Chi tiết Hợp đồng điện tử CT-#${viewContract?.id}`}
        footer={<Button onClick={() => setViewContract(null)}>Đóng</Button>}
        onCancel={() => setViewContract(null)}
        width={680}
      >
        {viewContract && (
          <Space direction="vertical" size={12} style={{ width: '100%', marginTop: 12 }}>
            <Descriptions bordered size="small" column={2}>
              <Descriptions.Item label="Hội viên">
                {viewContract.member_name || `Hội viên #${viewContract.member_id}`}
              </Descriptions.Item>
              <Descriptions.Item label="Trạng thái">
                <StatusTag status={viewContract.status} map={CONTRACT_STATUS_MAP} />
              </Descriptions.Item>
              <Descriptions.Item label="Ký lúc">
                {viewContract.signed_at ? formatDateTime(viewContract.signed_at) : 'Chưa ký'}
              </Descriptions.Item>
              <Descriptions.Item label="Địa chỉ IP">{viewContract.ip || '—'}</Descriptions.Item>
            </Descriptions>

            <Card
              size="small"
              title="Nội dung hợp đồng đã kết xuất"
              style={{ background: '#f8fafc' }}
            >
              <div
                dangerouslySetInnerHTML={{
                  __html: viewContract.rendered_html
                    ? safeHtml(viewContract.rendered_html)
                    : '<p>Chưa có nội dung kết xuất</p>',
                }}
              />
            </Card>

            {isSafeSignatureSrc(viewContract.signature_url) && (
              <Card size="small" title="Chữ ký điện tử của hội viên">
                <img
                  src={viewContract.signature_url}
                  alt="Chữ ký hội viên"
                  style={{ maxHeight: 120, border: '1px dashed #cbd5e1', background: '#fff' }}
                />
              </Card>
            )}
          </Space>
        )}
      </Modal>
    </>
  );
}

/* --------------------------------------------------------------------------
 * TAB 3: Bán vé ngày / Vé khách QR (/admin/day-passes)
 * -------------------------------------------------------------------------- */
type DaySaleValues = {
  branch_id: number;
  buyer_name: string;
  phone: string;
  valid_date: Dayjs;
  price: number;
  payment_method: string;
};

function DayPassesTab({ canManage }: { canManage: boolean }) {
  const { message } = App.useApp();
  const qc = useQueryClient();
  const { branchId } = useWorkingBranch();
  const { branches } = useBranchOptions();

  const [sellOpen, setSellOpen] = useState(false);
  const [issuedPass, setIssuedPass] = useState<DayPass | null>(null);
  // Nhân viên chỉ bán vé ngày (DAY) thu tiền tại quầy. Vé khách mời (GUEST) do hội viên tự tạo trên cổng hội viên
  // theo hạn mức gói tập (PLAN §Sprint 24) – API bán vé tại quầy luôn tạo vé DAY.
  const [form] = Form.useForm<DaySaleValues>();

  const table = useTableQuery<
    DayPass,
    { branch_id?: number; type?: DayPassType; page?: number; limit?: number }
  >({
    queryKey: [contractsPassesApi.dayPassKey, 'list'],
    queryFn: (p) => contractsPassesApi.listDayPasses(p),
    filterKeys: ['branch_id', 'type'],
    numberKeys: ['branch_id'],
    defaults: { order_by: 'id', order_dir: 'DESC' },
    extraParams: branchId ? { branch_id: branchId } : {},
  });

  const createPassMut = useMutation({
    mutationFn: (v: DaySaleValues) => {
      const payload: DayPassInput = {
        branch_id: v.branch_id,
        buyer_name: v.buyer_name.trim(),
        phone: v.phone.trim(),
        type: 'DAY',
        valid_date: toApiDate(v.valid_date) ?? dayjs().format('YYYY-MM-DD'),
        price: Number(v.price) || 0,
        payment_method: v.payment_method,
      };
      return contractsPassesApi.createDayPass(payload);
    },
    onSuccess: (res) => {
      message.success('Đã phát hành vé ngày QR thành công!');
      void qc.invalidateQueries({ queryKey: [contractsPassesApi.dayPassKey] });
      setSellOpen(false);
      setIssuedPass(res);
    },
    onError: (err) => showError(err, 'Không thể phát hành vé ngày'),
  });

  const copyToken = async (token: string) => {
    try {
      await navigator.clipboard.writeText(token);
      message.success(`Đã sao chép mã QR: ${token}`);
    } catch {
      message.info(token);
    }
  };

  const columns = useMemo<ColumnsType<DayPass>>(
    () => [
      {
        title: 'Mã QR Vé ngày',
        dataIndex: 'qr_token',
        width: 190,
        render: (token: string, r) => (
          <Space size={4}>
            <Tag color="geekblue" className="code-text" style={{ margin: 0 }}>
              {token}
            </Tag>
            <Button
              type="text"
              size="small"
              icon={<CopyOutlined />}
              onClick={() => void copyToken(token)}
            />
            <Button
              type="text"
              size="small"
              icon={<QrcodeOutlined />}
              onClick={() => setIssuedPass(r)}
            />
          </Space>
        ),
      },
      {
        title: 'Khách tập',
        dataIndex: 'buyer_name',
        render: (name: string, r) => (
          <Space direction="vertical" size={0}>
            <Typography.Text strong>{name}</Typography.Text>
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              {formatPhone(r.phone)}
            </Typography.Text>
          </Space>
        ),
      },
      {
        title: 'Loại vé',
        dataIndex: 'type',
        render: (t: DayPassType) => (
          <Tag color={t === 'DAY' ? 'blue' : 'purple'}>{DAY_PASS_TYPE_LABELS[t] ?? t}</Tag>
        ),
      },
      {
        title: 'Chi nhánh',
        dataIndex: 'branch_name',
        render: (v: string | undefined, r) => v || `CN #${r.branch_id}`,
      },
      {
        title: 'Ngày hiệu lực',
        dataIndex: 'valid_date',
        render: (v: string) => formatDate(v),
      },
      {
        title: 'Giá vé',
        dataIndex: 'price',
        align: 'right',
        render: (v: number) => <MoneyText value={v} />,
      },
      {
        title: 'Trạng thái sử dụng',
        dataIndex: 'used_at',
        render: (usedAt: string | null | undefined) =>
          usedAt ? (
            <Tag color="default">Đã quét lúc {formatDateTime(usedAt)}</Tag>
          ) : (
            <Tag color="green">Chưa sử dụng (1 lượt)</Tag>
          ),
      },
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [],
  );

  return (
    <>
      <Row justify="space-between" align="middle" style={{ marginBottom: 12 }}>
        <Col />
        <Col>
          <ActionBar
            onAdd={() => {
              form.resetFields();
              form.setFieldsValue({
                branch_id: branchId ?? branches[0]?.id ?? 1,
                valid_date: dayjs(),
                price: 120000,
                payment_method: 'CASH',
              });
              setSellOpen(true);
            }}
            addPerm={PERM.CONTRACT_MANAGE}
            addText="Bán vé ngày QR"
            onRefresh={() => table.refetch()}
            refreshing={table.isFetching}
          />
        </Col>
      </Row>

      <FilterBar
        searchPlaceholder="Tìm theo tên khách, SĐT, mã QR DP-..."
        colTextSearch={6}
        filters={[
          ...(branchId
            ? []
            : [
                {
                  name: 'branch_id',
                  placeholder: 'Chi nhánh',
                  options: branches.map((b) => ({ label: b.name, value: b.id })),
                  col: 4,
                },
              ]),
          {
            name: 'type',
            placeholder: 'Loại vé',
            options: DAY_PASS_TYPE_OPTIONS,
            col: 5,
          },
        ]}
      />

      <DataTable<DayPass>
        table={table}
        columns={columns}
        emptyText="Chưa có vé ngày hoặc vé khách QR nào được phát hành"
      />

      <Modal
        open={sellOpen}
        title="Bán vé tập ngày (Day Pass)"
        okText="Phát hành vé QR"
        cancelText="Hủy"
        onOk={() => form.submit()}
        onCancel={() => setSellOpen(false)}
        confirmLoading={createPassMut.isPending}
        width={540}
        forceRender
      >
        <Form
          form={form}
          layout="vertical"
          disabled={!canManage}
          onFinish={(v) => createPassMut.mutate(v)}
          style={{ marginTop: 12 }}
        >
          <Row gutter={12}>
            <Col xs={24}>
              <Alert
                type="info"
                showIcon
                style={{ marginBottom: 12 }}
                message="Vé khách mời (Guest Pass) do hội viên tự tạo trên cổng hội viên theo hạn mức gói tập."
              />
            </Col>
            <Col xs={24} sm={12}>
              <Form.Item
                name="branch_id"
                label="Chi nhánh tập luyện"
                rules={[{ required: true, message: 'Chọn chi nhánh' }]}
              >
                <BranchSelect activeOnly />
              </Form.Item>
            </Col>
            <Col xs={24} sm={12}>
              <Form.Item
                name="buyer_name"
                label="Họ và tên khách"
                rules={[{ required: true, whitespace: true, message: 'Nhập họ tên khách' }]}
              >
                <Input placeholder="Nguyễn Văn Khách" />
              </Form.Item>
            </Col>
            <Col xs={24} sm={12}>
              <Form.Item
                name="phone"
                label="Số điện thoại khách"
                rules={[{ required: true, whitespace: true, message: 'Nhập SĐT khách' }]}
              >
                <Input placeholder="0901234567" />
              </Form.Item>
            </Col>
            <Col xs={12}>
              <Form.Item
                name="price"
                label="Giá vé ngày (VND)"
                rules={[
                  { required: true, message: 'Nhập giá vé' },
                  { type: 'number', min: 0, message: 'Giá vé không được âm' },
                ]}
              >
                <MoneyInput />
              </Form.Item>
            </Col>
            <Col xs={12}>
              <Form.Item name="payment_method" label="Phương thức thanh toán">
                <Select options={COUNTER_PAYMENT_METHOD_OPTIONS} />
              </Form.Item>
            </Col>
            <Col xs={24}>
              <Form.Item
                name="valid_date"
                label="Ngày hiệu lực sử dụng"
                rules={[{ required: true }]}
              >
                <DatePicker
                  style={{ width: '100%' }}
                  format={DATE_FORMAT}
                  disabledDate={(d) => d.isBefore(dayjs(), 'day')}
                />
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Modal>

      <Modal
        open={!!issuedPass}
        title={
          <Space>
            <QrcodeOutlined style={{ color: '#2563eb' }} />
            <span>Vé tập QR 1 lần – {issuedPass?.buyer_name}</span>
          </Space>
        }
        okText="Đóng"
        cancelButtonProps={{ style: { display: 'none' } }}
        onOk={() => setIssuedPass(null)}
        onCancel={() => setIssuedPass(null)}
        width={480}
      >
        {issuedPass && (
          <Space direction="vertical" size={12} style={{ width: '100%', marginTop: 12 }}>
            <Alert
              type="success"
              showIcon
              message="Mã QR dùng 1 lần tại quầy Check-in hoặc màn hình Kiosk"
              description={
                <div style={{ marginTop: 6 }}>
                  <Tag
                    color="geekblue"
                    className="code-text"
                    style={{ fontSize: 16, padding: '6px 12px', fontWeight: 700 }}
                  >
                    {issuedPass.qr_token}
                  </Tag>
                  <Button
                    size="small"
                    type="link"
                    icon={<CopyOutlined />}
                    onClick={() => void copyToken(issuedPass.qr_token)}
                  >
                    Sao chép mã
                  </Button>
                </div>
              }
            />
            <Descriptions bordered size="small" column={1}>
              <Descriptions.Item label="Khách tập">
                {issuedPass.buyer_name} ({formatPhone(issuedPass.phone)})
              </Descriptions.Item>
              <Descriptions.Item label="Loại vé">
                {DAY_PASS_TYPE_LABELS[issuedPass.type] ?? issuedPass.type}
              </Descriptions.Item>
              <Descriptions.Item label="Ngày hiệu lực">
                {formatDate(issuedPass.valid_date)}
              </Descriptions.Item>
            </Descriptions>
          </Space>
        )}
      </Modal>
    </>
  );
}

export default function ContractsPassesPage() {
  const { can } = usePermission();
  const canManage = can(PERM.CONTRACT_MANAGE);

  return (
    <Card variant="borderless" className="page-card">
      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        <PageTitle
          title="Hợp đồng điện tử & Vé ngày QR"
          subtitle="Soạn mẫu hợp đồng điện tử, quản lý chữ ký số của hội viên và phát hành vé ngày / vé khách mời QR dùng 1 lần"
        />

        <Tabs
          defaultActiveKey="templates"
          items={[
            {
              key: 'templates',
              label: (
                <Space size={6}>
                  <FileTextOutlined />
                  <span>Mẫu hợp đồng điện tử</span>
                </Space>
              ),
              children: <ContractTemplatesTab canManage={canManage} />,
            },
            {
              key: 'member-contracts',
              label: (
                <Space size={6}>
                  <FileProtectOutlined />
                  <span>Hợp đồng hội viên đã ký</span>
                </Space>
              ),
              children: <MemberContractsTab />,
            },
            {
              key: 'day-passes',
              label: (
                <Space size={6}>
                  <QrcodeOutlined />
                  <span>Bán vé ngày / Vé khách QR</span>
                </Space>
              ),
              children: <DayPassesTab canManage={canManage} />,
            },
          ]}
        />
      </Space>
    </Card>
  );
}
