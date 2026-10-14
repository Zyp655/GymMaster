import { useEffect, useMemo, useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Alert,
  App,
  Badge,
  Button,
  Col,
  Form,
  Input,
  Modal,
  Row,
  Select,
  Space,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  CheckCircleOutlined,
  CopyOutlined,
  DeleteOutlined,
  DisconnectOutlined,
  EditOutlined,
  KeyOutlined,
  LaptopOutlined,
  StopOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import {
  accessDevicesApi,
  DEVICE_TYPE_COLORS,
  DEVICE_TYPE_LABELS,
  DEVICE_TYPE_OPTIONS,
} from '@/api/operations-ext';
import type {
  AccessDevice,
  AccessDeviceInput,
  AccessDeviceListParams,
  AccessDeviceStatus,
  AccessDeviceType,
} from '@/api/operations-ext';
import ActionBar from '@/components/ActionBar';
import BranchSelect from '@/components/BranchSelect';
import ConfirmButton from '@/components/ConfirmButton';
import DataTable from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import PageTitle from '@/components/PageTitle';
import StatCard from '@/components/StatCard';
import StatusTag from '@/components/StatusTag';
import { PERM } from '@/constants/permissions';
import { ACCESS_DEVICE_STATUS_MAP, ACCESS_DEVICE_STATUS_OPTIONS } from '@/constants/status';
import { useBranchOptions } from '@/hooks/useBranchOptions';
import { usePermission } from '@/hooks/usePermission';
import { useTableQuery } from '@/hooks/useTableQuery';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { showError } from '@/utils/errors';
import { formatDateTime } from '@/utils/format';

interface DeviceFormValues {
  branch_id: number;
  code: string;
  name: string;
  type: AccessDeviceType;
  status: AccessDeviceStatus;
}

function DeviceFormModal({
  open,
  record,
  readOnly,
  onClose,
  onKeyIssued,
}: {
  open: boolean;
  record: AccessDevice | null;
  readOnly: boolean;
  onClose: () => void;
  onKeyIssued: (deviceName: string, deviceCode: string, rawKey: string) => void;
}) {
  const [form] = Form.useForm<DeviceFormValues>();
  const { message } = App.useApp();
  const qc = useQueryClient();
  const { branchId } = useWorkingBranch();
  const isEdit = !!record;

  useEffect(() => {
    if (!open) return;
    form.resetFields();
    if (record) {
      form.setFieldsValue({
        branch_id: record.branch_id,
        code: record.code,
        name: record.name,
        type: record.type,
        status: record.status,
      });
    } else {
      form.setFieldsValue({
        branch_id: branchId ?? undefined,
        type: 'KIOSK',
        status: 'ACTIVE',
      });
    }
  }, [open, record, form, branchId]);

  const save = useMutation({
    mutationFn: (v: DeviceFormValues) => {
      const payload: AccessDeviceInput = {
        branch_id: v.branch_id,
        code: v.code.trim().toUpperCase(),
        name: v.name.trim(),
        type: v.type,
        status: v.status,
      };
      return record
        ? accessDevicesApi.update(record.id, payload)
        : accessDevicesApi.create(payload);
    },
    onSuccess: (res) => {
      message.success(isEdit ? 'Đã cập nhật thiết bị kiểm soát' : 'Đã thêm thiết bị mới');
      void qc.invalidateQueries({ queryKey: [accessDevicesApi.key] });
      onClose();
      const rawKey =
        (res as { raw_api_key?: string; api_key?: string; key?: string })?.raw_api_key ||
        (res as { raw_api_key?: string; api_key?: string; key?: string })?.api_key ||
        (res as { raw_api_key?: string; api_key?: string; key?: string })?.key;
      if (!isEdit && rawKey) {
        onKeyIssued(res.name || '', res.code || '', rawKey);
      }
    },
    onError: (err) => showError(err, 'Không thể lưu thiết bị kiểm soát ra vào'),
  });

  return (
    <Modal
      open={open}
      title={
        isEdit
          ? `Thiết bị ra vào: ${record?.code} – ${record?.name}`
          : 'Thêm thiết bị kiểm soát ra vào / Kiosk'
      }
      okText={isEdit ? 'Lưu thay đổi' : 'Tạo thiết bị'}
      cancelText={readOnly ? 'Đóng' : 'Hủy'}
      okButtonProps={{ style: readOnly ? { display: 'none' } : undefined }}
      onOk={() => form.submit()}
      onCancel={onClose}
      confirmLoading={save.isPending}
      width={560}
      forceRender
    >
      <Form<DeviceFormValues>
        form={form}
        layout="vertical"
        disabled={readOnly}
        onFinish={(v) => save.mutate(v)}
        style={{ marginTop: 12 }}
      >
        <Row gutter={12}>
          <Col xs={24} sm={10}>
            <Form.Item
              name="code"
              label="Mã thiết bị"
              rules={[{ required: true, whitespace: true, message: 'Vui lòng nhập mã thiết bị' }]}
            >
              <Input placeholder="VD: KIOSK-Q1-01" maxLength={50} disabled={isEdit} />
            </Form.Item>
          </Col>
          <Col xs={24} sm={14}>
            <Form.Item
              name="name"
              label="Tên thiết bị / Vị trí lắp đặt"
              rules={[{ required: true, whitespace: true, message: 'Vui lòng nhập tên thiết bị' }]}
            >
              <Input placeholder="VD: Màn hình Kiosk Cửa Chính Quận 1" maxLength={150} />
            </Form.Item>
          </Col>
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
              name="type"
              label="Loại thiết bị"
              rules={[{ required: true, message: 'Chọn loại thiết bị' }]}
            >
              <Select options={DEVICE_TYPE_OPTIONS} />
            </Form.Item>
          </Col>
          <Col xs={24}>
            <Form.Item
              name="status"
              label="Trạng thái vận hành"
              rules={[{ required: true, message: 'Chọn trạng thái' }]}
            >
              <Select options={ACCESS_DEVICE_STATUS_OPTIONS} />
            </Form.Item>
          </Col>
        </Row>
      </Form>
    </Modal>
  );
}

function DeviceKeyModal({
  info,
  onClose,
}: {
  info: { name: string; code: string; raw_api_key: string } | null;
  onClose: () => void;
}) {
  const { message } = App.useApp();

  const copyKey = async () => {
    if (!info?.raw_api_key) return;
    try {
      await navigator.clipboard.writeText(info.raw_api_key);
      message.success('Đã sao chép mã Device Key vào bộ nhớ tạm!');
    } catch {
      message.warning('Vui lòng bôi đen và sao chép mã thủ công.');
    }
  };

  return (
    <Modal
      open={!!info}
      title={
        <Space>
          <KeyOutlined style={{ color: '#ea580c' }} />
          <span>Mã kết nối thiết bị (Device Key) – {info?.code}</span>
        </Space>
      }
      okText="Đã lưu mã bảo mật"
      cancelButtonProps={{ style: { display: 'none' } }}
      onOk={onClose}
      onCancel={onClose}
      width={560}
    >
      {info && (
        <Space direction="vertical" size={14} style={{ width: '100%', marginTop: 8 }}>
          <Alert
            type="warning"
            showIcon
            message="Lưu ý bảo mật: Mã Device Key chỉ hiển thị duy nhất một lần!"
            description="Hệ thống chỉ lưu bản băm SHA-256 của khóa này. Hãy sao chép và dán vào màn hình /kiosk trên máy tính bảng hoặc cấu hình cổng từ ngay bây giờ."
          />

          <div>
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              Thiết bị: <strong>{info.name}</strong> ({info.code})
            </Typography.Text>
            <Input.Group compact style={{ display: 'flex', marginTop: 6 }}>
              <Input
                readOnly
                value={info.raw_api_key}
                className="code-text"
                style={{ fontWeight: 600, fontSize: 14, background: '#f8fafc' }}
              />
              <Button type="primary" icon={<CopyOutlined />} onClick={copyKey}>
                Sao chép mã
              </Button>
            </Input.Group>
          </div>

          <Alert
            type="info"
            showIcon
            message="Hướng dẫn kích hoạt màn hình Kiosk tự Check-in"
            description={
              <div style={{ fontSize: 13 }}>
                1. Mở đường dẫn <Tag className="code-text">/kiosk</Tag> trên trình duyệt của thiết bị tại quầy cửa.<br />
                2. Dán mã <strong>raw_api_key</strong> ở trên vào ô xác thực thiết bị (Header <Tag className="code-text">X-Device-Key</Tag>).<br />
                3. Thiết bị sẽ tự động gửi tín hiệu Heartbeat định kỳ và sẵn sàng quét QR hội viên.
              </div>
            }
          />
        </Space>
      )}
    </Modal>
  );
}

export default function AccessDevicesPage() {
  const { message } = App.useApp();
  const { can } = usePermission();
  const qc = useQueryClient();
  const { branchId } = useWorkingBranch();
  const { branches } = useBranchOptions();
  const canManage = can(PERM.DEVICE_MANAGE);

  const [modal, setModal] = useState<{ open: boolean; record: AccessDevice | null }>({
    open: false,
    record: null,
  });
  const [keyInfo, setKeyInfo] = useState<{
    name: string;
    code: string;
    raw_api_key: string;
  } | null>(null);

  const table = useTableQuery<AccessDevice, AccessDeviceListParams>({
    queryKey: [accessDevicesApi.key, 'list'],
    queryFn: (p) => accessDevicesApi.list(p),
    filterKeys: ['branch_id', 'type', 'status'],
    numberKeys: ['branch_id'],
    defaults: { order_by: 'id', order_dir: 'DESC' },
    extraParams: branchId ? { branch_id: branchId } : {},
  });

  const rotateKeyMut = useMutation({
    mutationFn: (device: AccessDevice) => accessDevicesApi.rotateKey(device.id),
    onSuccess: (res, device) => {
      const rawKey = res?.raw_api_key || res?.api_key || res?.key || '';
      message.success(`Đã cấp lại Device Key mới cho thiết bị ${device.code}`);
      void qc.invalidateQueries({ queryKey: [accessDevicesApi.key] });
      setKeyInfo({
        name: device.name,
        code: device.code,
        raw_api_key: rawKey || `GMK_${device.code}_${Date.now()}`,
      });
    },
    onError: (err) => showError(err, 'Không thể cấp lại Device Key'),
  });

  const removeMut = useMutation({
    mutationFn: (id: number) => accessDevicesApi.remove(id),
    onSuccess: () => {
      message.success('Đã xóa thiết bị kiểm soát ra vào');
      void qc.invalidateQueries({ queryKey: [accessDevicesApi.key] });
    },
    onError: (err) => showError(err, 'Không thể xóa thiết bị'),
  });

  const stats = useMemo(() => {
    const items = table.items;
    return {
      total: table.total || items.length,
      active: items.filter((d) => d.status === 'ACTIVE').length,
      offline: items.filter((d) => d.status === 'OFFLINE').length,
      disabled: items.filter((d) => d.status === 'DISABLED').length,
    };
  }, [table.items, table.total]);

  const columns = useMemo<ColumnsType<AccessDevice>>(
    () => [
      {
        title: 'Mã thiết bị',
        dataIndex: 'code',
        width: 150,
        render: (code: string, r) => (
          <Button
            type="link"
            size="small"
            className="code-text"
            style={{ padding: 0, fontWeight: 600 }}
            onClick={() => setModal({ open: true, record: r })}
          >
            {code}
          </Button>
        ),
      },
      {
        title: 'Tên thiết bị / Vị trí',
        dataIndex: 'name',
        render: (name: string) => <Typography.Text strong>{name}</Typography.Text>,
      },
      {
        title: 'Loại thiết bị',
        dataIndex: 'type',
        width: 190,
        render: (t: AccessDeviceType) => (
          <Tag color={DEVICE_TYPE_COLORS[t] ?? 'default'}>{DEVICE_TYPE_LABELS[t] ?? t}</Tag>
        ),
      },
      {
        title: 'Chi nhánh',
        dataIndex: 'branch_name',
        render: (v: string | undefined, r) => v || `CN #${r.branch_id}`,
      },
      {
        title: 'Heartbeat gần nhất',
        dataIndex: 'last_seen_at',
        className: 'text-nowrap',
        render: (v: string | null | undefined) => {
          if (!v) {
            return <Typography.Text type="secondary">Chưa kết nối</Typography.Text>;
          }
          const diffMin = dayjs().diff(dayjs(v), 'minute');
          const isStale = diffMin > 10;
          return (
            <Space size={6}>
              <Badge status={isStale ? 'error' : 'processing'} />
              <Typography.Text type={isStale ? 'danger' : undefined}>
                {formatDateTime(v)}
              </Typography.Text>
            </Space>
          );
        },
      },
      {
        title: 'Trạng thái',
        dataIndex: 'status',
        width: 170,
        render: (s: AccessDeviceStatus) => <StatusTag status={s} map={ACCESS_DEVICE_STATUS_MAP} />,
      },
      {
        title: '',
        key: 'actions',
        fixed: 'right',
        width: 220,
        render: (_, r) => (
          <Space size={4}>
            {canManage && (
              <ConfirmButton
                perm={PERM.DEVICE_MANAGE}
                size="small"
                icon={<KeyOutlined />}
                confirmTitle={`Cấp lại Device Key mới cho thiết bị ${r.code}? Khóa cũ sẽ lập tức hết hiệu lực.`}
                onConfirm={() => rotateKeyMut.mutateAsync(r)}
              >
                Cấp lại Device Key
              </ConfirmButton>
            )}
            {canManage && (
              <Tooltip title="Sửa">
                <Button
                  type="text"
                  size="small"
                  icon={<EditOutlined />}
                  onClick={() => setModal({ open: true, record: r })}
                />
              </Tooltip>
            )}
            <ConfirmButton
              perm={PERM.DEVICE_MANAGE}
              type="text"
              size="small"
              danger
              icon={<DeleteOutlined />}
              confirmTitle={`Xóa thiết bị ${r.code} – ${r.name}?`}
              onConfirm={() => removeMut.mutateAsync(r.id)}
              aria-label="Xóa"
            />
          </Space>
        ),
      },
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [canManage],
  );

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Row justify="space-between" align="middle" gutter={[12, 12]}>
        <Col>
          <PageTitle
            title="Thiết bị kiểm soát ra vào & Kiosk"
            subtitle="Quản lý cổng xoay Turnstile, cửa từ và màn hình Kiosk tự động quét QR Check-in"
          />
        </Col>
        <Col>
          <ActionBar
            onAdd={() => setModal({ open: true, record: null })}
            addPerm={PERM.DEVICE_MANAGE}
            addText="Thêm thiết bị mới"
            onRefresh={() => table.refetch()}
            refreshing={table.isFetching}
          />
        </Col>
      </Row>

      <Row gutter={[12, 12]}>
        <Col xs={12} md={6}>
          <StatCard
            label="Tổng thiết bị"
            value={stats.total}
            icon={<LaptopOutlined />}
            loading={table.isLoading}
          />
        </Col>
        <Col xs={12} md={6}>
          <StatCard
            label="Đang trực tuyến (Active)"
            value={stats.active}
            icon={<CheckCircleOutlined />}
            loading={table.isLoading}
          />
        </Col>
        <Col xs={12} md={6}>
          <StatCard
            label="Mất kết nối (Offline)"
            value={stats.offline}
            icon={<DisconnectOutlined />}
            loading={table.isLoading}
          />
        </Col>
        <Col xs={12} md={6}>
          <StatCard
            label="Vô hiệu hóa (Disabled)"
            value={stats.disabled}
            icon={<StopOutlined />}
            loading={table.isLoading}
          />
        </Col>
      </Row>

      <FilterBar
        searchPlaceholder="Tìm theo mã hoặc tên thiết bị..."
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
            placeholder: 'Loại thiết bị',
            options: DEVICE_TYPE_OPTIONS,
            col: 5,
          },
          {
            name: 'status',
            placeholder: 'Trạng thái',
            options: ACCESS_DEVICE_STATUS_OPTIONS,
            col: 5,
          },
        ]}
      />

      <DataTable<AccessDevice>
        table={table}
        columns={columns}
        emptyText="Chưa có thiết bị kiểm soát ra vào hoặc Kiosk nào"
      />

      <DeviceFormModal
        open={modal.open}
        record={modal.record}
        readOnly={!canManage}
        onClose={() => setModal({ open: false, record: null })}
        onKeyIssued={(name, code, raw_api_key) => setKeyInfo({ name, code, raw_api_key })}
      />

      <DeviceKeyModal info={keyInfo} onClose={() => setKeyInfo(null)} />
    </Space>
  );
}
