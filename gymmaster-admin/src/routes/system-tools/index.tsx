import { useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Col,
  Collapse,
  Form,
  Input,
  InputNumber,
  Row,
  Select,
  Space,
  Steps,
  Table,
  Tabs,
  Tag,
  Typography,
  message,
} from 'antd';
import {
  CheckCircleOutlined,
  CloudUploadOutlined,
  RocketOutlined,
  SwapOutlined,
} from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import {
  cmsSystemApi,
  type BranchTransferItem,
  type LegacyImportJobItem,
  type LegacyImportRow,
} from '@/api/cms-system';
import { formatDateTime, formatMoney } from '@/utils/format';

const SAMPLE_LEGACY_ROWS: LegacyImportRow[] = [
  {
    full_name: 'Phạm Hoàng Long',
    phone: '0918222333',
    email: 'long.pham@example.com',
    gender: 'MALE',
    plan_code: 'GOLD-12M',
    sub_start_date: '2026-01-01',
    sub_end_date: '2026-12-31',
    debt_amount: 0,
  },
  {
    full_name: 'Lê Thu Hà',
    phone: '0987654111',
    email: 'ha.le@example.com',
    gender: 'FEMALE',
    plan_code: 'SILVER-6M',
    sub_start_date: '2026-06-01',
    sub_end_date: '2026-12-01',
    // Hợp đồng PT chuyển sang: pt_trainer_email bắt buộc khi có pt_package_code.
    pt_package_code: 'PT10',
    pt_remaining_sessions: 6,
    pt_trainer_email: 'hlv.cg@gymmaster.vn',
    debt_amount: 1500000,
  },
];

export default function SystemToolsPage() {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const [wizardForm] = Form.useForm();
  const [transferForm] = Form.useForm();
  const [importBranchId, setImportBranchId] = useState<number>(1);
  const [importJson, setImportJson] = useState<string>(
    JSON.stringify(SAMPLE_LEGACY_ROWS, null, 2),
  );

  const wizardQuery = useQuery({
    queryKey: ['admin-setup-wizard'],
    queryFn: () => cmsSystemApi.getSetupWizardStatus(),
  });

  const importsQuery = useQuery({
    queryKey: ['admin-legacy-imports'],
    queryFn: () => cmsSystemApi.listLegacyImports(),
  });

  const transfersQuery = useQuery({
    queryKey: ['admin-branch-transfers'],
    queryFn: () => cmsSystemApi.listBranchTransfers(),
  });

  const completeWizardMut = useMutation({
    mutationFn: (v: Record<string, unknown>) => cmsSystemApi.completeSetupWizard(v),
    onSuccess: () => {
      message.success('Đã lưu cấu hình Thiết lập ban đầu');
      qc.invalidateQueries({ queryKey: ['admin-setup-wizard'] });
    },
  });

  const runImportMut = useMutation({
    mutationFn: (mode: 'DRY_RUN' | 'COMMIT') => {
      let rows: LegacyImportRow[] = [];
      try {
        rows = JSON.parse(importJson);
      } catch {
        throw new Error('Dữ liệu JSON không hợp lệ');
      }
      return cmsSystemApi.runLegacyImport({
        branch_id: importBranchId,
        file_name: `chuyen-doi-du-lieu-${mode.toLowerCase()}.xlsx`,
        mode,
        rows,
      });
    },
    onSuccess: (res) => {
      if (res.error_rows > 0) {
        message.warning(`Phát hiện ${res.error_rows} dòng lỗi khi kiểm tra`);
      } else {
        message.success(
          res.mode === 'DRY_RUN'
            ? `Chạy thử thành công: ${res.ok_rows}/${res.total_rows} dòng hợp lệ!`
            : `Đã ghi chính thức ${res.ok_rows} hồ sơ hội viên (source = IMPORT)!`,
        );
      }
      qc.invalidateQueries({ queryKey: ['admin-legacy-imports'] });
    },
    onError: (err: Error) => message.error(err.message),
  });

  const createTransferMut = useMutation({
    mutationFn: (v: {
      member_id: number;
      to_branch_id: number;
      transfer_fee?: number;
      reason?: string;
    }) => cmsSystemApi.createBranchTransfer(v),
    onSuccess: () => {
      message.success('Đã chuyển chi nhánh chính và cập nhật gói tập đang chạy');
      transferForm.resetFields();
      qc.invalidateQueries({ queryKey: ['admin-branch-transfers'] });
    },
  });

  const steps = wizardQuery.data?.steps ?? [];

  return (
    <Card
      className="gm-card-wrap-head"
      title="Thiết lập Ban đầu, Chuyển đổi Dữ liệu Cũ & Công cụ Hệ thống"
    >
      <Tabs
        items={[
          {
            key: 'wizard',
            label: '1. Wizard Thiết lập Ban đầu (6 bước)',
            children: (
              <Space direction="vertical" size={20} style={{ width: '100%' }}>
                <Alert
                  type={wizardQuery.data?.completed ? 'success' : 'info'}
                  showIcon
                  message={
                    wizardQuery.data?.completed
                      ? 'Hệ thống đã hoàn tất toàn bộ 6 bước thiết lập ban đầu!'
                      : 'Hướng dẫn thiết lập trung tâm lần đầu cho Chủ phòng tập (OWNER)'
                  }
                  description="Bạn có thể bấm vào từng bước để mở trang quản lý chi tiết hoặc điền nhanh cấu hình bên dưới."
                />
                <Steps
                  labelPlacement="vertical"
                  size="small"
                  current={steps.filter((s) => s.completed).length}
                  items={steps.map((s) => ({
                    title: s.title,
                    status: s.completed ? 'finish' : 'process',
                    description: (
                      <Button
                        type="link"
                        size="small"
                        style={{ padding: 0, height: 'auto', whiteSpace: 'normal' }}
                        onClick={() => navigate(s.route)}
                      >
                        Mở trang ({s.count} bản ghi)
                      </Button>
                    ),
                  }))}
                />
                <Card size="small" title="Khai báo nhanh thông tin vận hành trung tâm">
                  <Form
                    form={wizardForm}
                    layout="vertical"
                    initialValues={{
                      center_name: 'Hệ thống GymMaster Fitness & Yoga',
                      hotline: '1900 6868',
                      opening_hours: '05:30 - 22:00',
                      cancel_class_before_hours: 2,
                    }}
                    onFinish={(v) => completeWizardMut.mutate(v)}
                  >
                    <Row gutter={16}>
                      <Col xs={24} md={8}>
                        <Form.Item name="center_name" label="Tên thương hiệu trung tâm">
                          <Input />
                        </Form.Item>
                      </Col>
                      <Col xs={24} md={6}>
                        <Form.Item name="hotline" label="Hotline CSKH">
                          <Input />
                        </Form.Item>
                      </Col>
                      <Col xs={24} md={6}>
                        <Form.Item name="opening_hours" label="Giờ mở cửa tiêu chuẩn">
                          <Input />
                        </Form.Item>
                      </Col>
                      <Col xs={24} md={4}>
                        <Form.Item name="cancel_class_before_hours" label="Hủy lớp trước (giờ)">
                          <InputNumber min={1} max={24} style={{ width: '100%' }} />
                        </Form.Item>
                      </Col>
                    </Row>
                    <Space wrap>
                      <Button
                        type="primary"
                        icon={<RocketOutlined />}
                        htmlType="submit"
                        loading={completeWizardMut.isPending}
                      >
                        Lưu & Hoàn tất Wizard
                      </Button>
                      <Button
                        onClick={() => completeWizardMut.mutate({ skip: true })}
                        loading={completeWizardMut.isPending}
                      >
                        Bỏ qua & Quay lại sau
                      </Button>
                    </Space>
                  </Form>
                </Card>
              </Space>
            ),
          },
          {
            key: 'import',
            label: '2. Chuyển đổi Dữ liệu Cũ (Dry-Run & Commit)',
            children: (
              <Row gutter={16}>
                <Col xs={24} lg={10}>
                  <Card
                    size="small"
                    title="Dữ liệu Hội viên + Gói đang chạy + Công nợ (Excel/JSON)"
                    extra={
                      <Button
                        size="small"
                        onClick={() =>
                          setImportJson(JSON.stringify(SAMPLE_LEGACY_ROWS, null, 2))
                        }
                      >
                        Nạp dữ liệu mẫu
                      </Button>
                    }
                  >
                    <Space direction="vertical" style={{ width: '100%' }} size={12}>
                      <div>
                        <Typography.Text strong>Chi nhánh tiếp nhận ID: </Typography.Text>
                        <InputNumber
                          min={1}
                          value={importBranchId}
                          onChange={(v) => setImportBranchId(Number(v || 1))}
                        />
                      </div>
                      <Input.TextArea
                        rows={11}
                        value={importJson}
                        onChange={(e) => setImportJson(e.target.value)}
                        style={{ fontFamily: 'monospace', fontSize: 12 }}
                      />
                      <Space wrap>
                        <Button
                          icon={<CheckCircleOutlined />}
                          loading={runImportMut.isPending}
                          onClick={() => runImportMut.mutate('DRY_RUN')}
                        >
                          1. Chạy thử (Dry-Run)
                        </Button>
                        <Button
                          type="primary"
                          icon={<CloudUploadOutlined />}
                          loading={runImportMut.isPending}
                          onClick={() => runImportMut.mutate('COMMIT')}
                        >
                          2. Ghi chính thức (Commit)
                        </Button>
                      </Space>
                    </Space>
                  </Card>
                </Col>
                <Col xs={24} lg={14}>
                  <Card size="small" title="Lịch sử các phiên Import & Báo cáo Lỗi">
                    <Table<LegacyImportJobItem>
                      rowKey="id"
                      size="small"
                      loading={importsQuery.isLoading}
                      dataSource={importsQuery.data ?? []}
                      columns={[
                        { title: 'Tệp dữ liệu', dataIndex: 'file_name' },
                        {
                          title: 'Chế độ',
                          dataIndex: 'mode',
                          render: (m: string) => (
                            <Tag color={m === 'COMMIT' ? 'green' : 'geekblue'}>
                              {m === 'COMMIT' ? 'Ghi chính thức' : 'Chạy thử'}
                            </Tag>
                          ),
                        },
                        { title: 'Tổng dòng', dataIndex: 'total_rows' },
                        {
                          title: 'Hợp lệ / Lỗi',
                          render: (_, r) => (
                            <Space wrap>
                              <Tag color="success">{r.ok_rows} OK</Tag>
                              {r.error_rows > 0 && <Tag color="error">{r.error_rows} lỗi</Tag>}
                            </Space>
                          ),
                        },
                        {
                          title: 'Thời gian',
                          dataIndex: 'created_at',
                          render: (v: string) => formatDateTime(v),
                        },
                      ]}
                      expandable={{
                        expandedRowRender: (r) =>
                          r.errors?.length ? (
                            <ul>
                              {r.errors.map((e, i) => (
                                <li key={i} style={{ color: '#dc2626' }}>
                                  Dòng {e.row} ({e.phone || 'trống SĐT'}): {e.message}
                                </li>
                              ))}
                            </ul>
                          ) : (
                            <Typography.Text type="success">
                              Tất cả các dòng đều hợp lệ (0 lỗi).
                            </Typography.Text>
                          ),
                      }}
                    />
                  </Card>
                </Col>
              </Row>
            ),
          },
          {
            key: 'transfers',
            label: '3. Chuyển Chi nhánh Hội viên',
            children: (
              <Row gutter={16}>
                <Col xs={24} md={8}>
                  <Card size="small" title="Lập lệnh chuyển chi nhánh chính">
                    <Form
                      form={transferForm}
                      layout="vertical"
                      onFinish={(v) => createTransferMut.mutate(v)}
                    >
                      <Form.Item
                        name="member_id"
                        label="ID Hội viên"
                        rules={[{ required: true }]}
                      >
                        <InputNumber min={1} style={{ width: '100%' }} />
                      </Form.Item>
                      <Form.Item
                        name="to_branch_id"
                        label="Chuyển tới Chi nhánh ID"
                        rules={[{ required: true }]}
                      >
                        <Select
                          options={[
                            { value: 1, label: 'Cơ sở 1 – Cầu Giấy' },
                            { value: 2, label: 'Cơ sở 2 – Đống Đa' },
                          ]}
                        />
                      </Form.Item>
                      <Form.Item name="transfer_fee" label="Phí chuyển chi nhánh (VNĐ)" initialValue={0}>
                        <InputNumber min={0} step={50000} style={{ width: '100%' }} />
                      </Form.Item>
                      <Form.Item name="reason" label="Lý do chuyển">
                        <Input.TextArea rows={2} placeholder="Chuyển nơi ở / nơi làm việc..." />
                      </Form.Item>
                      <Button
                        type="primary"
                        icon={<SwapOutlined />}
                        htmlType="submit"
                        loading={createTransferMut.isPending}
                        block
                      >
                        Xác nhận chuyển chi nhánh
                      </Button>
                    </Form>
                  </Card>
                </Col>
                <Col xs={24} md={16}>
                  <Table<BranchTransferItem>
                    rowKey="id"
                    size="small"
                    loading={transfersQuery.isLoading}
                    dataSource={transfersQuery.data ?? []}
                    columns={[
                      {
                        title: 'Hội viên',
                        render: (_, r) => `${r.member_name} (${r.member_code})`,
                      },
                      { title: 'Từ chi nhánh', dataIndex: 'from_branch_name' },
                      { title: 'Sang chi nhánh', dataIndex: 'to_branch_name' },
                      {
                        title: 'Phí chuyển',
                        dataIndex: 'transfer_fee',
                        render: (v: number) => formatMoney(v),
                      },
                      { title: 'Lý do', dataIndex: 'reason' },
                      {
                        title: 'Thời gian',
                        dataIndex: 'created_at',
                        render: (v: string) => formatDateTime(v),
                      },
                    ]}
                  />
                </Col>
              </Row>
            ),
          },
          {
            key: 'guide',
            label: '4. Hướng dẫn Sử dụng theo Vai trò',
            children: (
              <Collapse
                defaultActiveKey={['owner', 'receptionist']}
                items={[
                  {
                    key: 'owner',
                    label: '1. Chủ phòng tập & Quản lý (OWNER / MANAGER)',
                    children: (
                      <p>
                        Thiết lập chi nhánh, gói tập, duyệt yêu cầu bảo lưu, duyệt chi phí vận
                        hành, tính & khóa bảng lương tháng (`/app/payrolls`), xem báo cáo Lãi/Lỗ P&L
                        (`/app/finance`) và ma trận giữ chân hội viên Cohort (`/app/kpi-analytics`).
                      </p>
                    ),
                  },
                  {
                    key: 'receptionist',
                    label: '2. Lễ tân & Thu ngân (RECEPTIONIST)',
                    children: (
                      <p>
                        Sử dụng phím tắt <strong>Ctrl + K</strong> để tra cứu nhanh hội viên; quét
                        mã QR động / vé tập thử (`TP-...`) / vé ngày (`DP-...`) tại `/app/checkin`;
                        gán tủ đồ tại `/app/lockers`; chăm sóc khách tiềm năng tại `/app/leads` và
                        đối soát tiền mặt cuối ca tại `/app/finance`.
                      </p>
                    ),
                  },
                  {
                    key: 'trainer',
                    label: '3. Huấn luyện viên (TRAINER)',
                    children: (
                      <p>
                        Soạn giáo án mẫu & nhân bản cho học viên tại `/app/workout-plans`; hoàn
                        thành buổi dạy PT để nhận hoa hồng tự động; trao đổi trực tuyến với học viên
                        qua mục Tin nhắn PT (`/app/cms-marketing`) và đăng ký lịch rảnh / nghỉ phép
                        tại `/app/trainer-schedules`.
                      </p>
                    ),
                  },
                ]}
              />
            ),
          },
        ]}
      />
    </Card>
  );
}
