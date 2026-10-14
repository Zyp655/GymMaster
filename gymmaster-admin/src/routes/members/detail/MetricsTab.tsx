import { useMemo, useState } from 'react';
import { Line } from '@ant-design/charts';
import { useQuery } from '@tanstack/react-query';
import {
  App,
  Button,
  Card,
  Col,
  DatePicker,
  Empty,
  Form,
  Input,
  InputNumber,
  Modal,
  Row,
  Segmented,
  Table,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import memberApi from '@/api/member';
import type { BodyMetric, BodyMetricInput, Member } from '@/api/member';
import ConfirmButton from '@/components/ConfirmButton';
import PermissionGate from '@/components/PermissionGate';
import { PERM } from '@/constants/permissions';
import { getErrorMessage, setFormError, showError } from '@/utils/errors';
import { DATETIME_FORMAT, formatDateTime } from '@/utils/format';
import { useAddMetric, useDeleteMetric } from '../queries';

type Series = 'weight' | 'fat' | 'bmi';
const SERIES: Record<Series, { label: string; pick: (m: BodyMetric) => number | null }> = {
  weight: { label: 'Cân nặng (kg)', pick: (m) => m.weight_kg },
  fat: { label: 'Mỡ cơ thể (%)', pick: (m) => m.body_fat_pct },
  bmi: { label: 'BMI', pick: (m) => m.bmi },
};

const num = (v?: number | null) => (v === null || v === undefined ? '—' : v);

interface FormValues extends Omit<BodyMetricInput, 'measured_at'> {
  measured_at: Dayjs;
}

/** Tab "Chỉ số cơ thể": biểu đồ theo thời gian + bảng + thêm/xóa lần đo. */
export default function MetricsTab({ member }: { member: Member }) {
  const [series, setSeries] = useState<Series[]>(['weight', 'fat', 'bmi']);
  const [open, setOpen] = useState(false);
  const del = useDeleteMetric(member.id);

  const query = useQuery({
    queryKey: [memberApi.key, 'metrics', member.id],
    queryFn: () => memberApi.metrics(member.id),
  });
  const metrics = useMemo(() => query.data ?? [], [query.data]);

  const chartData = useMemo(
    () =>
      metrics.flatMap((m) =>
        series
          .map((s) => ({
            date: formatDateTime(m.measured_at),
            value: SERIES[s].pick(m),
            type: SERIES[s].label,
          }))
          .filter((d) => d.value !== null),
      ),
    [metrics, series],
  );

  const columns: ColumnsType<BodyMetric> = [
    { title: 'Thời điểm đo', dataIndex: 'measured_at', render: (v: string) => formatDateTime(v) },
    { title: 'Cân nặng (kg)', dataIndex: 'weight_kg', align: 'right' },
    { title: 'Chiều cao (cm)', dataIndex: 'height_cm', align: 'right' },
    { title: 'BMI', dataIndex: 'bmi', align: 'right', render: (v: number) => v?.toFixed(1) },
    { title: 'Mỡ (%)', dataIndex: 'body_fat_pct', align: 'right', render: num },
    { title: 'Cơ (kg)', dataIndex: 'muscle_mass_kg', align: 'right', render: num },
    {
      title: 'Eo / Ngực / Mông (cm)',
      key: 'sizes',
      render: (_, r) => `${num(r.waist_cm)} / ${num(r.chest_cm)} / ${num(r.hip_cm)}`,
    },
    { title: 'Ghi chú', dataIndex: 'note', ellipsis: true },
    { title: 'Người đo', dataIndex: 'measured_by_name' },
    {
      title: '',
      key: 'actions',
      width: 48,
      fixed: 'right',
      render: (_, r) => (
        <ConfirmButton
          perm={PERM.MEMBER_METRIC}
          type="text"
          size="small"
          danger
          icon={<DeleteOutlined />}
          confirmTitle="Xóa lần đo này?"
          confirmContent={`Lần đo lúc ${formatDateTime(r.measured_at)} sẽ bị xóa.`}
          onConfirm={() => del.mutateAsync(r.id)}
          aria-label="Xóa"
        />
      ),
    },
  ];

  if (query.isError) {
    return (
      <Empty description={getErrorMessage(query.error, 'Không tải được chỉ số')}>
        <Button onClick={() => query.refetch()}>Thử lại</Button>
      </Empty>
    );
  }

  return (
    <>
      <Card
        size="small"
        title="Diễn biến chỉ số"
        loading={query.isLoading}
        extra={
          <Row gutter={8} align="middle" wrap={false}>
            <Col>
              <Segmented<string>
                size="small"
                options={[
                  { label: 'Tất cả', value: 'all' },
                  { label: 'Cân nặng', value: 'weight' },
                  { label: 'Mỡ', value: 'fat' },
                  { label: 'BMI', value: 'bmi' },
                ]}
                value={series.length > 1 ? 'all' : series[0]}
                onChange={(v) => setSeries(v === 'all' ? ['weight', 'fat', 'bmi'] : [v as Series])}
              />
            </Col>
            <PermissionGate perm={PERM.MEMBER_METRIC}>
              <Col>
                <Button
                  type="primary"
                  size="small"
                  icon={<PlusOutlined />}
                  onClick={() => setOpen(true)}
                >
                  Thêm chỉ số
                </Button>
              </Col>
            </PermissionGate>
          </Row>
        }
      >
        {chartData.length === 0 ? (
          <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="Chưa có dữ liệu đo" />
        ) : (
          <Line
            height={280}
            data={chartData}
            xField="date"
            yField="value"
            colorField="type"
            point={{ shapeField: 'circle', sizeField: 3 }}
            axis={{ x: { labelAutoRotate: true, title: false }, y: { title: false } }}
            legend={{ color: { position: 'top' } }}
          />
        )}
      </Card>

      <Table<BodyMetric>
        style={{ marginTop: 16 }}
        size="small"
        rowKey="id"
        loading={query.isFetching}
        columns={columns}
        dataSource={[...metrics].reverse()}
        pagination={metrics.length > 10 ? { pageSize: 10 } : false}
        scroll={{ x: 'max-content' }}
        locale={{ emptyText: 'Chưa có lần đo nào' }}
      />

      <MetricModal member={member} open={open} onClose={() => setOpen(false)} />
    </>
  );
}

function MetricModal({
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
  const add = useAddMetric(member.id);

  const onFinish = (v: FormValues) => {
    add.mutate(
      { ...v, measured_at: v.measured_at.format(), note: v.note?.trim() ?? '' },
      {
        onSuccess: (m) => {
          message.success(`Đã lưu chỉ số (BMI ${m.bmi})`);
          onClose();
        },
        onError: (e) => {
          if (
            !setFormError(form, e, {
              measured_at: /thời điểm|tương lai/i,
              weight_kg: /cân nặng/i,
              height_cm: /chiều cao/i,
            })
          )
            showError(e, 'Không thể lưu chỉ số');
        },
      },
    );
  };

  const latest = member.latest_metric;
  const n = (min: number, max: number, label: string) => ({
    type: 'number' as const,
    min,
    max,
    message: `${label} trong khoảng ${min}–${max}`,
  });

  return (
    <Modal
      open={open}
      title="Thêm chỉ số cơ thể"
      okText="Lưu"
      cancelText="Hủy"
      onOk={() => form.submit()}
      onCancel={onClose}
      confirmLoading={add.isPending}
      afterOpenChange={(o) => {
        if (!o) return;
        form.resetFields();
        form.setFieldsValue({ measured_at: dayjs(), height_cm: latest?.height_cm });
      }}
      width={640}
      forceRender
    >
      <Form<FormValues> form={form} layout="vertical" onFinish={onFinish} style={{ marginTop: 12 }}>
        <Row gutter={16}>
          <Col xs={24} sm={12}>
            <Form.Item
              name="measured_at"
              label="Thời điểm đo"
              rules={[{ required: true, message: 'Chọn thời điểm đo' }]}
            >
              <DatePicker
                showTime={{ format: 'HH:mm' }}
                format={DATETIME_FORMAT}
                style={{ width: '100%' }}
                disabledDate={(d) => d.isAfter(dayjs(), 'day')}
              />
            </Form.Item>
          </Col>
          <Col xs={12} sm={6}>
            <Form.Item
              name="weight_kg"
              label="Cân nặng (kg)"
              rules={[{ required: true, message: 'Nhập cân nặng' }, n(1, 500, 'Cân nặng')]}
            >
              <InputNumber style={{ width: '100%' }} step={0.1} />
            </Form.Item>
          </Col>
          <Col xs={12} sm={6}>
            <Form.Item
              name="height_cm"
              label="Chiều cao (cm)"
              rules={[{ required: true, message: 'Nhập chiều cao' }, n(50, 260, 'Chiều cao')]}
            >
              <InputNumber style={{ width: '100%' }} step={0.5} />
            </Form.Item>
          </Col>
          <Col xs={12} sm={8}>
            <Form.Item name="body_fat_pct" label="Mỡ (%)" rules={[n(0, 100, 'Tỷ lệ mỡ')]}>
              <InputNumber style={{ width: '100%' }} step={0.1} />
            </Form.Item>
          </Col>
          <Col xs={12} sm={8}>
            <Form.Item name="muscle_mass_kg" label="Cơ (kg)" rules={[n(0, 300, 'Khối cơ')]}>
              <InputNumber style={{ width: '100%' }} step={0.1} />
            </Form.Item>
          </Col>
          <Col xs={12} sm={8}>
            <Form.Item name="waist_cm" label="Eo (cm)" rules={[n(0, 999, 'Vòng eo')]}>
              <InputNumber style={{ width: '100%' }} />
            </Form.Item>
          </Col>
          <Col xs={12} sm={8}>
            <Form.Item name="chest_cm" label="Ngực (cm)" rules={[n(0, 999, 'Vòng ngực')]}>
              <InputNumber style={{ width: '100%' }} />
            </Form.Item>
          </Col>
          <Col xs={12} sm={8}>
            <Form.Item name="hip_cm" label="Mông (cm)" rules={[n(0, 999, 'Vòng mông')]}>
              <InputNumber style={{ width: '100%' }} />
            </Form.Item>
          </Col>
          <Col xs={24}>
            <Form.Item
              name="note"
              label="Ghi chú"
              rules={[{ max: 255, message: 'Tối đa 255 ký tự' }]}
            >
              <Input maxLength={255} placeholder="VD: Sau 1 tháng tập" />
            </Form.Item>
          </Col>
        </Row>
      </Form>
    </Modal>
  );
}
