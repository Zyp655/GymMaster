import { useMemo, useState } from 'react';
import type { ReactNode } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  App,
  Button,
  Card,
  Col,
  InputNumber,
  Result,
  Row,
  Select,
  Skeleton,
  Space,
  Tag,
  Typography,
} from 'antd';
import {
  CalendarOutlined,
  IdcardOutlined,
  LoginOutlined,
  ReloadOutlined,
  SaveOutlined,
  SettingOutlined,
  ThunderboltOutlined,
  UndoOutlined,
} from '@ant-design/icons';
import settingApi from '@/api/setting';
import type { Setting, SettingValue } from '@/api/setting';
import DateText from '@/components/DateText';
import PageTitle from '@/components/PageTitle';
import { getErrorMessage, showError } from '@/utils/errors';

interface SettingDef {
  key: string;
  label: string;
  help: string;
  unit: string;
  min: number;
  max: number;
  /** Mảng số nguyên (VD ngày nhắc gia hạn). */
  list?: { maxItems: number };
}

interface GroupDef {
  key: string;
  title: string;
  icon: ReactNode;
  items: SettingDef[];
}

/** Khoảng hợp lệ khớp backend (docs/api/catalog.md §4). */
const GROUPS: GroupDef[] = [
  {
    key: 'checkin',
    title: 'Check-in',
    icon: <LoginOutlined />,
    items: [
      {
        key: 'checkin.duplicate_window_minutes',
        label: 'Chặn check-in lặp lại',
        help: 'Trong khoảng thời gian này, hội viên check-in lần nữa sẽ không bị trừ thêm buổi.',
        unit: 'phút',
        min: 1,
        max: 1440,
      },
      {
        key: 'checkin.qr_ttl_seconds',
        label: 'Thời hạn mã QR',
        help: 'Mã QR động trên ứng dụng hội viên hết hiệu lực sau số giây này.',
        unit: 'giây',
        min: 15,
        max: 600,
      },
    ],
  },
  {
    key: 'booking',
    title: 'Đặt lớp',
    icon: <CalendarOutlined />,
    items: [
      {
        key: 'booking.advance_days',
        label: 'Đặt lớp trước tối đa',
        help: 'Hội viên chỉ được đặt chỗ các buổi lớp trong số ngày tới.',
        unit: 'ngày',
        min: 1,
        max: 60,
      },
      {
        key: 'booking.cancel_before_minutes',
        label: 'Hủy đặt chỗ trước',
        help: 'Hạn chót hủy đặt chỗ tính trước giờ bắt đầu lớp. Quá hạn không được hủy.',
        unit: 'phút',
        min: 1,
        max: 10080,
      },
      {
        key: 'booking.no_show_limit',
        label: 'Số lần vắng mặt tối đa',
        help: 'Vắng mặt (đặt chỗ nhưng không đến) quá số lần này sẽ bị tạm khóa đặt lớp.',
        unit: 'lần',
        min: 1,
        max: 100,
      },
      {
        key: 'booking.block_days',
        label: 'Thời gian khóa đặt lớp',
        help: 'Số ngày hội viên bị khóa đặt lớp khi vượt giới hạn vắng mặt.',
        unit: 'ngày',
        min: 1,
        max: 365,
      },
    ],
  },
  {
    key: 'subscription',
    title: 'Gói tập',
    icon: <IdcardOutlined />,
    items: [
      {
        key: 'subscription.remind_before_days',
        label: 'Nhắc gia hạn trước khi hết hạn',
        help: 'Gửi thông báo nhắc gia hạn vào các mốc (số ngày trước hạn). Gõ số rồi nhấn Enter; tối đa 10 mốc, mỗi mốc 1–90 ngày.',
        unit: 'ngày',
        min: 1,
        max: 90,
        list: { maxItems: 10 },
      },
    ],
  },
  {
    key: 'pt',
    title: 'Huấn luyện cá nhân (PT)',
    icon: <ThunderboltOutlined />,
    items: [
      {
        key: 'pt.cancel_before_minutes',
        label: 'Hủy buổi PT trước',
        help: 'Hạn chót hủy buổi PT tính trước giờ hẹn. Quá hạn buổi tập bị tính là đã dùng.',
        unit: 'phút',
        min: 1,
        max: 10080,
      },
    ],
  },
];

const KNOWN_KEYS = new Set(GROUPS.flatMap((g) => g.items.map((i) => i.key)));

const sameValue = (a: SettingValue | null, b: SettingValue) =>
  JSON.stringify(a) === JSON.stringify(b);

function validate(def: SettingDef, value: SettingValue | null): string | null {
  if (def.list) {
    const arr = Array.isArray(value) ? value : [];
    if (arr.length === 0) return 'Vui lòng nhập ít nhất một mốc';
    if (arr.length > def.list.maxItems) return `Tối đa ${def.list.maxItems} mốc`;
    if (arr.some((n) => !Number.isInteger(n) || n < def.min || n > def.max))
      return `Mỗi mốc phải là số nguyên từ ${def.min} đến ${def.max}`;
    return null;
  }
  if (typeof value !== 'number' || !Number.isInteger(value)) return 'Giá trị phải là số nguyên';
  if (value < def.min || value > def.max) return `Giá trị phải từ ${def.min} đến ${def.max}`;
  return null;
}

function SettingRow({ def, setting }: { def: SettingDef; setting?: Setting }) {
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const original = setting?.value ?? null;
  const [draft, setDraft] = useState<{ base: SettingValue | null; value: SettingValue | null }>({
    base: original,
    value: original,
  });
  // Server trả giá trị mới (lưu xong / làm mới) → đồng bộ lại ô nhập.
  const value = sameValue(draft.base, original as SettingValue) ? draft.value : original;
  const setValue = (v: SettingValue | null) => setDraft({ base: original, value: v });

  const save = useMutation({
    mutationFn: (v: SettingValue) => settingApi.update(def.key, v),
    onSuccess: (saved) => {
      queryClient.setQueryData<Setting[]>([settingApi.key, 'list'], (old) =>
        old?.map((s) => (s.key === saved.key ? saved : s)),
      );
      setDraft({ base: saved.value, value: saved.value });
      message.success(`Đã lưu "${def.label}"`);
    },
    onError: (e) => showError(e, 'Không thể lưu cấu hình'),
  });

  const error = value === null ? null : validate(def, value);
  const dirty = setting && value !== null && !sameValue(value, setting.value);

  if (!setting) {
    return (
      <Row gutter={[16, 8]} className="setting-row">
        <Col xs={24}>
          <Typography.Text type="secondary">
            {def.label}: chưa có trên máy chủ ({def.key})
          </Typography.Text>
        </Col>
      </Row>
    );
  }

  return (
    <Row gutter={[16, 8]} align="middle" className="setting-row" style={{ padding: '12px 0' }}>
      <Col xs={24} md={12}>
        <Typography.Text strong>{def.label}</Typography.Text>
        <div>
          <Typography.Text type="secondary" style={{ fontSize: 13 }}>
            {def.help}
          </Typography.Text>
        </div>
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
          Cập nhật <DateText value={setting.updated_at} withTime />
        </Typography.Text>
      </Col>
      <Col xs={24} md={12}>
        <Space.Compact style={{ width: '100%' }}>
          {def.list ? (
            <Select
              mode="tags"
              style={{ width: '100%' }}
              status={error ? 'error' : undefined}
              value={(Array.isArray(value) ? value : []).map(String)}
              tokenSeparators={[',', ' ']}
              notFoundContent={null}
              suffixIcon={null}
              placeholder="VD: 7, 3, 1"
              onChange={(vals: string[]) =>
                setValue(
                  [
                    ...new Set(vals.map((s) => Number(s.trim())).filter((n) => !Number.isNaN(n))),
                  ].sort((a, b) => b - a),
                )
              }
              tagRender={({ label, closable, onClose }) => (
                <Tag closable={closable} onClose={onClose} style={{ marginInlineEnd: 4 }}>
                  {label} {def.unit}
                </Tag>
              )}
            />
          ) : (
            <InputNumber
              style={{ width: '100%' }}
              min={def.min}
              max={def.max}
              precision={0}
              status={error ? 'error' : undefined}
              value={typeof value === 'number' ? value : null}
              onChange={(v) => setValue(v)}
              suffix={def.unit}
            />
          )}
          {dirty && (
            <Button
              icon={<UndoOutlined />}
              onClick={() => setValue(setting.value)}
              aria-label="Hoàn tác"
            />
          )}
          <Button
            type="primary"
            icon={<SaveOutlined />}
            disabled={!dirty || !!error}
            loading={save.isPending}
            onClick={() => value !== null && save.mutate(value)}
          >
            Lưu
          </Button>
        </Space.Compact>
        <Typography.Text type={error ? 'danger' : 'secondary'} style={{ fontSize: 12 }}>
          {error ?? `Khoảng hợp lệ: ${def.min} – ${def.max} ${def.unit}`}
        </Typography.Text>
      </Col>
    </Row>
  );
}

/** Cấu hình hệ thống – mỗi mục lưu riêng (`PUT /admin/settings/{key}`). */
export default function SettingsPage() {
  const query = useQuery({ queryKey: [settingApi.key, 'list'], queryFn: () => settingApi.list() });
  const byKey = useMemo(() => new Map((query.data ?? []).map((s) => [s.key, s])), [query.data]);
  const others = (query.data ?? []).filter((s) => !KNOWN_KEYS.has(s.key));

  return (
    <Card variant="borderless" className="page-card">
      <Row justify="space-between" align="middle" gutter={[8, 12]} className="page-head">
        <Col>
          <PageTitle title="Cấu hình" subtitle="Tham số vận hành áp dụng cho toàn hệ thống" />
        </Col>
        <Col>
          <Button icon={<ReloadOutlined spin={query.isFetching} />} onClick={() => query.refetch()}>
            Làm mới
          </Button>
        </Col>
      </Row>

      {query.isLoading ? (
        <Skeleton active paragraph={{ rows: 8 }} />
      ) : query.isError ? (
        <Result
          status="warning"
          title="Không tải được cấu hình"
          subTitle={getErrorMessage(query.error)}
          extra={
            <Button type="primary" onClick={() => query.refetch()}>
              Thử lại
            </Button>
          }
        />
      ) : (
        <Row gutter={[16, 16]}>
          {GROUPS.map((g) => (
            <Col xs={24} xl={12} key={g.key}>
              <Card
                size="small"
                title={
                  <Space>
                    {g.icon}
                    {g.title}
                  </Space>
                }
                styles={{ body: { paddingTop: 0, paddingBottom: 0 } }}
              >
                {g.items.map((def, i) => (
                  <div
                    key={def.key}
                    style={{ borderTop: i ? '1px solid var(--gm-border)' : undefined }}
                  >
                    <SettingRow def={def} setting={byKey.get(def.key)} />
                  </div>
                ))}
              </Card>
            </Col>
          ))}
          {others.length > 0 && (
            <Col xs={24} xl={12}>
              <Card
                size="small"
                title={
                  <Space>
                    <SettingOutlined />
                    Khác
                  </Space>
                }
                styles={{ body: { paddingTop: 0, paddingBottom: 0 } }}
              >
                {others.map((s, i) => (
                  <div
                    key={s.key}
                    style={{ borderTop: i ? '1px solid var(--gm-border)' : undefined }}
                  >
                    <SettingRow
                      setting={s}
                      def={{
                        key: s.key,
                        label: s.description || s.key,
                        help: s.key,
                        unit: '',
                        min: 0,
                        max: Number.MAX_SAFE_INTEGER,
                        list: Array.isArray(s.value) ? { maxItems: 100 } : undefined,
                      }}
                    />
                  </div>
                ))}
              </Card>
            </Col>
          )}
        </Row>
      )}
    </Card>
  );
}
