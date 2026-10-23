import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Alert,
  App,
  Button,
  Card,
  Col,
  Descriptions,
  Form,
  Input,
  Modal,
  Popconfirm,
  QRCode,
  Radio,
  Row,
  Select,
  Space,
  Steps,
  Table,
  Tabs,
  Tag,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  CopyOutlined,
  DeleteOutlined,
  DesktopOutlined,
  DownloadOutlined,
  FileProtectOutlined,
  KeyOutlined,
  LaptopOutlined,
  LockOutlined,
  LogoutOutlined,
  MobileOutlined,
  QrcodeOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
  StopOutlined,
  UnlockOutlined,
} from '@ant-design/icons';
import {
  DATA_REQUEST_TYPE_LABELS,
  DATA_REQUEST_TYPE_OPTIONS,
  securityPrivacyApi,
} from '@/api/operations-ext';
import type {
  ActiveSession,
  DataRequest,
  DataRequestStatus,
  DataRequestType,
  MFASetupData,
} from '@/api/operations-ext';
import DataTable from '@/components/DataTable';
import PageTitle from '@/components/PageTitle';
import StatusTag from '@/components/StatusTag';
import { PERM } from '@/constants/permissions';
import { DATA_REQUEST_STATUS_MAP, DATA_REQUEST_STATUS_OPTIONS } from '@/constants/status';
import { usePermission } from '@/hooks/usePermission';
import { useTableQuery } from '@/hooks/useTableQuery';
import { getErrorMessage } from '@/utils/errors';
import { formatDateTime, formatPhone } from '@/utils/format';
import { isMfaCodeComplete, mfaInputProps, normalizeMfaCode } from '../auth/mfaCode';
import type { MfaCodeMode } from '../auth/mfaCode';

/* ============================================================================
 * Tab 1: Xác thực 2 lớp (2FA TOTP)
 * ============================================================================ */

function MfaSecurityTab() {
  const { message } = App.useApp();
  const qc = useQueryClient();

  const [setupData, setSetupData] = useState<MFASetupData | null>(null);
  const [otpCode, setOtpCode] = useState('');
  const [disableCode, setDisableCode] = useState('');
  const [disableMode, setDisableMode] = useState<MfaCodeMode>('totp');
  const [disableModalOpen, setDisableModalOpen] = useState(false);
  const [recoveryCodes, setRecoveryCodes] = useState<string[]>([]);

  const mfaQuery = useQuery({
    queryKey: [securityPrivacyApi.mfaKey],
    queryFn: () => securityPrivacyApi.getMfaStatus(),
  });

  const setupMut = useMutation({
    mutationFn: () => securityPrivacyApi.setupMfa(),
    onSuccess: (res) => {
      setSetupData(res);
      if (res.recovery_codes?.length) {
        setRecoveryCodes(res.recovery_codes);
      }
      message.success('Đã khởi tạo khóa 2FA TOTP. Vui lòng quét mã QR để kích hoạt.');
    },
    onError: (err) => message.error(getErrorMessage(err)),
  });

  const enableMut = useMutation({
    mutationFn: (code: string) => securityPrivacyApi.enableMfa(code),
    onSuccess: (res) => {
      message.success('Đã kích hoạt xác thực 2 lớp (2FA) thành công');
      if (res?.recovery_codes?.length) {
        setRecoveryCodes(res.recovery_codes);
      }
      setSetupData(null);
      setOtpCode('');
      qc.invalidateQueries({ queryKey: [securityPrivacyApi.mfaKey] });
    },
    onError: (err) => message.error(getErrorMessage(err)),
  });

  const disableMut = useMutation({
    mutationFn: (code?: string) => securityPrivacyApi.disableMfa(code),
    onSuccess: () => {
      message.success('Đã tắt xác thực 2 lớp (2FA)');
      setDisableModalOpen(false);
      setDisableCode('');
      setSetupData(null);
      setRecoveryCodes([]);
      qc.invalidateQueries({ queryKey: [securityPrivacyApi.mfaKey] });
    },
    onError: (err) => message.error(getErrorMessage(err)),
  });

  const mfa = mfaQuery.data;
  const isEnabled = Boolean(mfa?.enabled);

  const handleCopyText = async (text: string, label: string) => {
    try {
      await navigator.clipboard.writeText(text);
      message.success(`Đã sao chép ${label}`);
    } catch {
      message.info(text);
    }
  };

  return (
    <Row gutter={[16, 16]}>
      <Col xs={24} lg={14}>
        <Card
          className="gm-card-wrap-head"
          loading={mfaQuery.isLoading}
          title={
            <Space>
              <SafetyCertificateOutlined />
              <span>Trạng thái xác thực 2 lớp (TOTP Authenticator)</span>
            </Space>
          }
          extra={
            <Button
              icon={<ReloadOutlined />}
              size="small"
              onClick={() => mfaQuery.refetch()}
              loading={mfaQuery.isFetching}
            >
              Làm mới
            </Button>
          }
        >
          <Alert
            type={isEnabled ? 'success' : 'warning'}
            showIcon
            icon={isEnabled ? <LockOutlined /> : <UnlockOutlined />}
            message={
              isEnabled
                ? 'Tài khoản của bạn đang được bảo vệ bằng xác thực 2 lớp (2FA TOTP)'
                : 'Tài khoản chưa bật xác thực 2 lớp (2FA)'
            }
            description={
              isEnabled
                ? `Mỗi khi đăng nhập từ thiết bị mới, hệ thống sẽ yêu cầu mã OTP 6 chữ số từ ứng dụng Google Authenticator hoặc Authy.${
                    mfa?.enabled_at ? ` Kích hoạt lúc: ${formatDateTime(mfa.enabled_at)}.` : ''
                  }`
                : 'Khuyến nghị bật xác thực 2 lớp đối với tài khoản Chủ phòng tập, Quản lý và Kế toán để bảo vệ dữ liệu tài chính & hội viên.'
            }
            style={{ marginBottom: 20 }}
          />

          {!isEnabled && !setupData && (
            <Space direction="vertical" size={12}>
              <Typography.Paragraph style={{ marginBottom: 4 }}>
                Sử dụng ứng dụng <strong>Google Authenticator</strong>,{' '}
                <strong>Microsoft Authenticator</strong> hoặc <strong>Authy</strong> trên điện thoại
                để nhận mã xác thực 6 chữ số đổi mới mỗi 30 giây.
              </Typography.Paragraph>
              <Button
                type="primary"
                icon={<QrcodeOutlined />}
                loading={setupMut.isPending}
                onClick={() => setupMut.mutate()}
              >
                Bắt đầu thiết lập 2FA
              </Button>
            </Space>
          )}

          {!isEnabled && setupData && (
            <div>
              <Steps
                size="small"
                current={otpCode.length === 6 ? 1 : 0}
                style={{ marginBottom: 20 }}
                items={[
                  { title: 'Quét mã QR / Nhập khóa' },
                  { title: 'Nhập mã 6 số xác nhận' },
                  { title: 'Lưu mã khôi phục' },
                ]}
              />

              <Row gutter={[20, 20]} align="middle">
                <Col xs={24} sm={9} style={{ textAlign: 'center' }}>
                  <div
                    style={{
                      display: 'inline-block',
                      padding: 12,
                      background: '#fff',
                      border: '1px solid var(--gm-border)',
                      borderRadius: 8,
                    }}
                  >
                    <QRCode
                      value={
                        setupData.otpauth_url ||
                        `otpauth://totp/GymMaster?secret=${setupData.secret}&issuer=GymMaster`
                      }
                      size={168}
                    />
                  </div>
                  <div style={{ marginTop: 8 }}>
                    <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                      Quét bằng ứng dụng Authenticator
                    </Typography.Text>
                  </div>
                </Col>

                <Col xs={24} sm={15}>
                  <Descriptions column={1} size="small" bordered>
                    <Descriptions.Item label="Khóa bí mật (Secret)">
                      <Space>
                        <Typography.Text code>{setupData.secret}</Typography.Text>
                        <Button
                          size="small"
                          type="text"
                          icon={<CopyOutlined />}
                          onClick={() => handleCopyText(setupData.secret, 'khóa bí mật 2FA')}
                        />
                      </Space>
                    </Descriptions.Item>
                    <Descriptions.Item label="Chuẩn OTP">
                      TOTP (RFC 6238) · 6 chữ số · Chu kỳ 30 giây
                    </Descriptions.Item>
                  </Descriptions>

                  <div style={{ marginTop: 16 }}>
                    <Typography.Text strong style={{ display: 'block', marginBottom: 8 }}>
                      Nhập mã xác thực 6 chữ số hiển thị trên điện thoại:
                    </Typography.Text>
                    <Space.Compact style={{ width: '100%', maxWidth: 340 }}>
                      <Input
                        value={otpCode}
                        onChange={(e) => setOtpCode(normalizeMfaCode(e.target.value, 'totp'))}
                        {...mfaInputProps('totp')}
                        placeholder="VD: 123456"
                        style={{ letterSpacing: 4, fontWeight: 600, fontSize: 16 }}
                        onPressEnter={() => {
                          if (otpCode.length === 6) enableMut.mutate(otpCode);
                        }}
                      />
                      <Button
                        type="primary"
                        icon={<CheckCircleOutlined />}
                        disabled={otpCode.length !== 6}
                        loading={enableMut.isPending}
                        onClick={() => enableMut.mutate(otpCode)}
                      >
                        Kích hoạt 2FA
                      </Button>
                    </Space.Compact>
                    <div style={{ marginTop: 8 }}>
                      <Button size="small" type="link" onClick={() => setSetupData(null)}>
                        Hủy thiết lập
                      </Button>
                    </div>
                  </div>
                </Col>
              </Row>
            </div>
          )}

          {isEnabled && (
            <Space>
              <Button
                danger
                icon={<UnlockOutlined />}
                onClick={() => {
                  setDisableCode('');
                  setDisableMode('totp');
                  setDisableModalOpen(true);
                }}
              >
                Tắt xác thực 2 lớp (2FA)
              </Button>
            </Space>
          )}
        </Card>
      </Col>

      <Col xs={24} lg={10}>
        <Card
          className="gm-card-wrap-head"
          title={
            <Space>
              <KeyOutlined />
              <span>Mã khôi phục dự phòng</span>
            </Space>
          }
        >
          <Typography.Paragraph type="secondary" style={{ fontSize: 13 }}>
            Mỗi mã khôi phục chỉ sử dụng được <strong>01 lần duy nhất</strong> trong trường hợp bạn
            mất điện thoại hoặc không thể truy cập ứng dụng tạo mã OTP.
          </Typography.Paragraph>

          {recoveryCodes.length > 0 || (mfa?.recovery_codes && mfa.recovery_codes.length > 0) ? (
            <div>
              <div
                style={{
                  display: 'grid',
                  gridTemplateColumns: 'repeat(2, minmax(0, 1fr))',
                  gap: 8,
                  padding: 12,
                  background: 'var(--gm-hover)',
                  border: '1px dashed var(--gm-border)',
                  borderRadius: 8,
                  marginBottom: 12,
                }}
              >
                {(recoveryCodes.length > 0 ? recoveryCodes : (mfa?.recovery_codes ?? [])).map(
                  (code) => (
                    <Typography.Text
                      key={code}
                      code
                      style={{ textAlign: 'center', fontSize: 13, padding: '2px 0' }}
                    >
                      {code}
                    </Typography.Text>
                  ),
                )}
              </div>
              <Button
                icon={<CopyOutlined />}
                size="small"
                onClick={() =>
                  handleCopyText(
                    (recoveryCodes.length > 0 ? recoveryCodes : (mfa?.recovery_codes ?? [])).join(
                      '\n',
                    ),
                    'danh sách mã khôi phục',
                  )
                }
              >
                Sao chép toàn bộ mã khôi phục
              </Button>
            </div>
          ) : (
            <Alert
              type="info"
              showIcon
              message="Mã khôi phục sẽ được hiển thị ngay khi bạn thiết lập hoặc kích hoạt 2FA."
            />
          )}
        </Card>
      </Col>

      <Modal
        open={disableModalOpen}
        title="Xác nhận tắt xác thực 2 lớp (2FA)"
        okText="Tắt 2FA"
        cancelText="Hủy"
        okButtonProps={{
          danger: true,
          loading: disableMut.isPending,
          disabled: !isMfaCodeComplete(disableCode, disableMode),
        }}
        onCancel={() => setDisableModalOpen(false)}
        onOk={() => disableMut.mutate(disableCode)}
        destroyOnHidden
      >
        <Alert
          type="warning"
          showIcon
          message="Việc tắt 2FA sẽ làm giảm mức độ bảo mật của tài khoản quản trị."
          style={{ marginBottom: 16 }}
        />
        <Typography.Text style={{ display: 'block', marginBottom: 8 }}>
          {disableMode === 'totp'
            ? 'Nhập mã OTP 6 chữ số hiện tại để xác nhận:'
            : 'Nhập một mã khôi phục (dạng XXXX-XXXX) để xác nhận:'}
        </Typography.Text>
        <Input
          // Đổi key theo chế độ → remount để bàn phím điện thoại đổi theo inputMode.
          key={disableMode}
          value={disableCode}
          onChange={(e) => setDisableCode(normalizeMfaCode(e.target.value, disableMode))}
          onPressEnter={() => {
            if (isMfaCodeComplete(disableCode, disableMode) && !disableMut.isPending) {
              disableMut.mutate(disableCode);
            }
          }}
          {...mfaInputProps(disableMode)}
          autoFocus
        />
        <Button
          type="link"
          size="small"
          style={{ padding: 0, marginTop: 8 }}
          onClick={() => {
            setDisableMode((m) => (m === 'totp' ? 'recovery' : 'totp'));
            setDisableCode('');
          }}
        >
          {disableMode === 'totp' ? 'Dùng mã khôi phục' : 'Dùng mã 6 số từ ứng dụng'}
        </Button>
      </Modal>
    </Row>
  );
}

/* ============================================================================
 * Tab 2: Phiên đăng nhập đang hoạt động
 * ============================================================================ */

function ActiveSessionsTab() {
  const { message } = App.useApp();
  const qc = useQueryClient();

  const sessionsQuery = useQuery({
    queryKey: [securityPrivacyApi.sessionKey],
    queryFn: () => securityPrivacyApi.listSessions(),
  });

  const revokeMut = useMutation({
    mutationFn: (id: number | string) => securityPrivacyApi.revokeSession(id),
    onSuccess: () => {
      message.success('Đã thu hồi phiên đăng nhập thiết bị');
      qc.invalidateQueries({ queryKey: [securityPrivacyApi.sessionKey] });
    },
    onError: (err) => message.error(getErrorMessage(err)),
  });

  const revokeOthersMut = useMutation({
    mutationFn: () => securityPrivacyApi.revokeOtherSessions(),
    onSuccess: () => {
      message.success('Đã đăng xuất toàn bộ các thiết bị khác');
      qc.invalidateQueries({ queryKey: [securityPrivacyApi.sessionKey] });
    },
    onError: (err) => message.error(getErrorMessage(err)),
  });

  const sessions = sessionsQuery.data ?? [];

  const renderDeviceIcon = (ua = '', name = '') => {
    const text = `${ua} ${name}`.toLowerCase();
    if (text.includes('iphone') || text.includes('android') || text.includes('mobile')) {
      return <MobileOutlined style={{ color: '#0284c7' }} />;
    }
    if (text.includes('mac') || text.includes('windows') || text.includes('laptop')) {
      return <LaptopOutlined style={{ color: '#0f766e' }} />;
    }
    return <DesktopOutlined style={{ color: '#475569' }} />;
  };

  const columns: ColumnsType<ActiveSession> = [
    {
      title: 'Thiết bị / Trình duyệt',
      key: 'device',
      render: (_, r) => {
        const isCurrent = Boolean(r.is_current || r.current);
        return (
          <Space align="start">
            <span style={{ fontSize: 18, marginTop: 2 }}>
              {renderDeviceIcon(r.user_agent, r.device_name)}
            </span>
            <div>
              <Space size={6}>
                <Typography.Text strong>{r.device_name || 'Thiết bị trình duyệt'}</Typography.Text>
                {isCurrent && <Tag color="green">Phiên hiện tại</Tag>}
              </Space>
              {r.user_agent && (
                <div
                  style={{ fontSize: 12, color: 'var(--gm-text-secondary)', maxWidth: 460 }}
                  className="truncate"
                >
                  {r.user_agent}
                </div>
              )}
            </div>
          </Space>
        );
      },
    },
    {
      title: 'Địa chỉ IP',
      dataIndex: 'ip',
      width: 150,
      render: (v?: string) => (v ? <Typography.Text code>{v}</Typography.Text> : '—'),
    },
    {
      title: 'Hoạt động gần nhất',
      dataIndex: 'last_used_at',
      width: 170,
      render: (v?: string | null, r?: ActiveSession) => formatDateTime(v || r?.created_at),
    },
    {
      title: 'Đăng nhập lúc',
      dataIndex: 'created_at',
      width: 170,
      render: (v: string) => formatDateTime(v),
    },
    {
      title: 'Hết hạn',
      dataIndex: 'expires_at',
      width: 170,
      render: (v?: string) => (v ? formatDateTime(v) : '—'),
    },
    {
      title: 'Thao tác',
      key: 'actions',
      width: 140,
      align: 'right',
      render: (_, r) => {
        const isCurrent = Boolean(r.is_current || r.current);
        if (isCurrent) {
          return (
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              Đang sử dụng
            </Typography.Text>
          );
        }
        return (
          <Popconfirm
            title="Thu hồi phiên đăng nhập này?"
            description="Thiết bị này sẽ bị đăng xuất ngay lập tức."
            okText="Thu hồi"
            cancelText="Hủy"
            okButtonProps={{ danger: true }}
            onConfirm={() => revokeMut.mutate(r.id)}
          >
            <Button size="small" danger icon={<LogoutOutlined />} loading={revokeMut.isPending}>
              Thu hồi
            </Button>
          </Popconfirm>
        );
      },
    },
  ];

  return (
    <Card
      title={
        <Space>
          <DesktopOutlined />
          <span>Danh sách thiết bị & phiên đăng nhập đang hoạt động</span>
        </Space>
      }
      extra={
        <Space>
          <Button
            icon={<ReloadOutlined />}
            onClick={() => sessionsQuery.refetch()}
            loading={sessionsQuery.isFetching}
          >
            Làm mới
          </Button>
          <Popconfirm
            title="Đăng xuất tất cả thiết bị khác?"
            description="Chỉ giữ lại phiên làm việc hiện tại của bạn."
            okText="Đăng xuất thiết bị khác"
            cancelText="Hủy"
            okButtonProps={{ danger: true }}
            onConfirm={() => revokeOthersMut.mutate()}
          >
            <Button
              danger
              icon={<StopOutlined />}
              loading={revokeOthersMut.isPending}
              disabled={sessions.length <= 1}
            >
              Thu hồi tất cả thiết bị khác
            </Button>
          </Popconfirm>
        </Space>
      }
    >
      <Table<ActiveSession>
        rowKey="id"
        loading={sessionsQuery.isLoading}
        columns={columns}
        dataSource={sessions}
        pagination={false}
        scroll={{ x: 920 }}
      />
    </Card>
  );
}

/* ============================================================================
 * Tab 3: Yêu cầu dữ liệu cá nhân – Nghị định 13/2023/NĐ-CP
 * ============================================================================ */

interface ProcessFormValues {
  status: 'DONE' | 'REJECTED';
  note?: string;
}

function DataPrivacyRequestsTab() {
  const { message } = App.useApp();
  const { can } = usePermission();
  // Duyệt/từ chối: API yêu cầu `data_request.process` (OWNER/MANAGER được cấp ở migration 000036).
  const canManage = can(PERM.DATA_REQUEST_PROCESS);
  const qc = useQueryClient();

  const [processingRow, setProcessingRow] = useState<DataRequest | null>(null);
  const [form] = Form.useForm<ProcessFormValues>();

  const table = useTableQuery<
    DataRequest,
    { page?: number; limit?: number; type?: DataRequestType; status?: DataRequestStatus }
  >({
    queryKey: [securityPrivacyApi.dataRequestKey],
    queryFn: (p) => securityPrivacyApi.listDataRequests(p),
    filterKeys: ['type', 'status'],
  });

  const processMut = useMutation({
    mutationFn: (values: ProcessFormValues) =>
      securityPrivacyApi.processDataRequest(processingRow!.id, values),
    onSuccess: () => {
      message.success('Đã xử lý yêu cầu dữ liệu cá nhân');
      setProcessingRow(null);
      qc.invalidateQueries({ queryKey: [securityPrivacyApi.dataRequestKey] });
    },
    onError: (err) => message.error(getErrorMessage(err)),
  });

  const openProcessModal = (row: DataRequest, defaultStatus: 'DONE' | 'REJECTED' = 'DONE') => {
    setProcessingRow(row);
    form.setFieldsValue({
      status: defaultStatus,
      note: row.note ?? '',
    });
  };

  const columns: ColumnsType<DataRequest> = [
    {
      title: 'Mã YC',
      dataIndex: 'id',
      width: 80,
      render: (id: number) => <Typography.Text code>#{id}</Typography.Text>,
    },
    {
      title: 'Hội viên',
      key: 'member',
      width: 210,
      render: (_, r) => (
        <div>
          <div style={{ fontWeight: 600 }}>{r.member_name || `Hội viên #${r.member_id}`}</div>
          <div style={{ fontSize: 12, color: 'var(--gm-text-secondary)' }}>
            {r.member_code ? `${r.member_code} · ` : ''}
            {r.member_phone ? formatPhone(r.member_phone) : ''}
          </div>
        </div>
      ),
    },
    {
      title: 'Loại yêu cầu (NĐ 13/2023)',
      dataIndex: 'type',
      width: 230,
      render: (type: DataRequestType) =>
        type === 'DELETE' ? (
          <Tag color="red" icon={<DeleteOutlined />}>
            {DATA_REQUEST_TYPE_LABELS.DELETE}
          </Tag>
        ) : (
          <Tag color="blue" icon={<DownloadOutlined />}>
            {DATA_REQUEST_TYPE_LABELS.EXPORT}
          </Tag>
        ),
    },
    {
      title: 'Lý do hội viên gửi',
      dataIndex: 'reason',
      ellipsis: true,
      render: (v?: string) => v || '—',
    },
    {
      title: 'Trạng thái',
      dataIndex: 'status',
      width: 140,
      render: (st: DataRequestStatus) => <StatusTag status={st} map={DATA_REQUEST_STATUS_MAP} />,
    },
    {
      title: 'Tệp xuất / Kết quả',
      dataIndex: 'file_url',
      width: 150,
      render: (url?: string, r?: DataRequest) => {
        if (url) {
          return (
            <Button
              size="small"
              type="link"
              icon={<DownloadOutlined />}
              href={url}
              target="_blank"
              rel="noreferrer"
            >
              Tải tệp dữ liệu
            </Button>
          );
        }
        if (r?.status === 'DONE' && r.type === 'DELETE') {
          return <Tag color="purple">Đã ẩn danh hóa</Tag>;
        }
        return '—';
      },
    },
    {
      title: 'Ngày gửi / Hoàn tất',
      key: 'dates',
      width: 185,
      render: (_, r) => (
        <div style={{ fontSize: 12 }}>
          <div>Gửi: {formatDateTime(r.created_at)}</div>
          {r.done_at && <div style={{ color: '#16a34a' }}>Xong: {formatDateTime(r.done_at)}</div>}
        </div>
      ),
    },
    {
      title: 'Ghi chú xử lý',
      dataIndex: 'note',
      width: 180,
      ellipsis: true,
      render: (v?: string, r?: DataRequest) => (
        <div>
          {v ? <div>{v}</div> : '—'}
          {r?.handled_by_name && (
            <div style={{ fontSize: 11, color: 'var(--gm-text-secondary)' }}>
              Bởi: {r.handled_by_name}
            </div>
          )}
        </div>
      ),
    },
    {
      title: 'Thao tác',
      key: 'actions',
      width: 150,
      fixed: 'right',
      render: (_, r) => {
        if (!canManage || r.status === 'DONE' || r.status === 'REJECTED') {
          return '—';
        }
        return (
          <Space size={6}>
            <Button
              size="small"
              type="primary"
              icon={<CheckCircleOutlined />}
              onClick={() => openProcessModal(r, 'DONE')}
            >
              Xử lý
            </Button>
            <Button
              size="small"
              danger
              icon={<CloseCircleOutlined />}
              onClick={() => openProcessModal(r, 'REJECTED')}
            >
              Từ chối
            </Button>
          </Space>
        );
      },
    },
  ];

  return (
    <Card>
      <Alert
        type="info"
        showIcon
        icon={<FileProtectOutlined />}
        message="Tuân thủ Nghị định 13/2023/NĐ-CP về Bảo vệ dữ liệu cá nhân"
        description="Khi duyệt yêu cầu XÓA DỮ LIỆU, hệ thống sẽ ẩn danh hóa thông tin định danh cá nhân (Họ tên, SĐT, Email, CCCD, Sinh trắc học) của hội viên nhưng vẫn giữ lại chứng từ hóa đơn kế toán theo luật thuế."
        style={{ marginBottom: 16 }}
      />

      <Space wrap style={{ marginBottom: 16 }}>
        <Select
          allowClear
          placeholder="Loại yêu cầu"
          style={{ width: 260 }}
          value={table.params.type}
          options={DATA_REQUEST_TYPE_OPTIONS}
          onChange={(v) => table.setFilters({ type: v })}
        />
        <Select
          allowClear
          placeholder="Trạng thái xử lý"
          style={{ width: 180 }}
          value={table.params.status}
          options={DATA_REQUEST_STATUS_OPTIONS}
          onChange={(v) => table.setFilters({ status: v })}
        />
        <Button
          icon={<ReloadOutlined />}
          onClick={() => table.refetch()}
          loading={table.isFetching}
        >
          Làm mới
        </Button>
      </Space>

      <DataTable<DataRequest> table={table} columns={columns} scroll={{ x: 1320 }} />

      <Modal
        open={Boolean(processingRow)}
        title={
          processingRow
            ? `Xử lý yêu cầu dữ liệu #${processingRow.id} – ${
                processingRow.member_name || `Hội viên #${processingRow.member_id}`
              }`
            : 'Xử lý yêu cầu dữ liệu'
        }
        okText="Xác nhận thực thi"
        cancelText="Hủy"
        confirmLoading={processMut.isPending}
        onCancel={() => setProcessingRow(null)}
        onOk={() => form.submit()}
        destroyOnHidden
      >
        {processingRow && (
          <div style={{ marginBottom: 16 }}>
            <Descriptions size="small" column={1} bordered>
              <Descriptions.Item label="Loại yêu cầu">
                {DATA_REQUEST_TYPE_LABELS[processingRow.type]}
              </Descriptions.Item>
              <Descriptions.Item label="Lý do hội viên">
                {processingRow.reason || '—'}
              </Descriptions.Item>
            </Descriptions>
            {processingRow.type === 'DELETE' && (
              <Alert
                type="warning"
                showIcon
                style={{ marginTop: 12 }}
                message="Cảnh báo ẩn danh hóa dữ liệu hội viên"
                description="Nếu chọn Phê duyệt & Hoàn tất, thông tin cá nhân của hội viên sẽ được ẩn danh hóa vĩnh viễn và không thể khôi phục."
              />
            )}
          </div>
        )}

        <Form
          form={form}
          layout="vertical"
          onFinish={(values) => processMut.mutate(values)}
          initialValues={{ status: 'DONE' }}
        >
          <Form.Item
            name="status"
            label="Quyết định xử lý"
            rules={[{ required: true, message: 'Vui lòng chọn quyết định xử lý' }]}
          >
            <Radio.Group>
              <Radio.Button value="DONE">
                Phê duyệt & Thực thi (
                {processingRow?.type === 'DELETE' ? 'Ẩn danh hóa' : 'Xuất file'})
              </Radio.Button>
              <Radio.Button value="REJECTED">Từ chối yêu cầu</Radio.Button>
            </Radio.Group>
          </Form.Item>

          <Form.Item name="note" label="Ghi chú xử lý / Lý do từ chối">
            <Input.TextArea
              rows={3}
              placeholder="Ghi chú nội bộ hoặc phản hồi gửi cho hội viên..."
            />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}

/* ============================================================================
 * Main Page: Bảo mật & Quyền riêng tư dữ liệu (/app/security-privacy)
 * ============================================================================ */

export default function SecurityPrivacyPage() {
  const { can } = usePermission();
  // 2FA & phiên đăng nhập: mọi nhân viên (API không yêu cầu quyền). Yêu cầu dữ liệu cá nhân:
  // API: xem danh sách cần `privacy.manage` hoặc `data_request.process`; xử lý cần `data_request.process`.
  const canPrivacy = can([PERM.PRIVACY_MANAGE, PERM.DATA_REQUEST_PROCESS]);
  return (
    <Card variant="borderless" className="page-card">
      <PageTitle
        title="Bảo mật 2FA, Phiên đăng nhập & Dữ liệu cá nhân"
        subtitle="Thiết lập xác thực 2 lớp (TOTP), giám sát thiết bị đăng nhập và xử lý yêu cầu dữ liệu theo Nghị định 13/2023/NĐ-CP"
      />

      <Tabs
        defaultActiveKey="mfa"
        items={[
          {
            key: 'mfa',
            label: (
              <Space>
                <SafetyCertificateOutlined />
                <span>Xác thực 2 lớp (2FA TOTP)</span>
              </Space>
            ),
            children: <MfaSecurityTab />,
          },
          {
            key: 'sessions',
            label: (
              <Space>
                <DesktopOutlined />
                <span>Phiên đăng nhập đang hoạt động</span>
              </Space>
            ),
            children: <ActiveSessionsTab />,
          },
          ...(canPrivacy
            ? [
                {
                  key: 'privacy',
                  label: (
                    <Space>
                      <FileProtectOutlined />
                      <span>Yêu cầu dữ liệu cá nhân (NĐ 13/2023)</span>
                    </Space>
                  ),
                  children: <DataPrivacyRequestsTab />,
                },
              ]
            : []),
        ]}
      />
    </Card>
  );
}
