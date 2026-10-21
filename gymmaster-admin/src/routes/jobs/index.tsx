import { useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  App,
  Button,
  Card,
  Col,
  Drawer,
  Form,
  Input,
  List,
  Modal,
  Result,
  Row,
  Segmented,
  Space,
  Table,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  HistoryOutlined,
  MailOutlined,
  PlayCircleOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import jobApi from '@/api/job';
import type { JobInfo, JobRun, JobRunStatus } from '@/api/job';
import ConfirmButton from '@/components/ConfirmButton';
import DateText from '@/components/DateText';
import PageTitle from '@/components/PageTitle';
import StatusTag from '@/components/StatusTag';
import { PERM } from '@/constants/permissions';
import type { StatusMeta } from '@/constants/status';
import { useIsMobile } from '@/hooks/useIsMobile';
import { usePermission } from '@/hooks/usePermission';
import { getErrorMessage, setFormError, showError } from '@/utils/errors';
import { formatDateTime, formatNumber } from '@/utils/format';

const JOB_RUN_STATUS_MAP: Record<string, StatusMeta> = {
  RUNNING: { label: 'Đang chạy', color: 'processing' },
  SUCCESS: { label: 'Thành công', color: 'green' },
  FAILED: { label: 'Thất bại', color: 'red' },
};

function JobRunsDrawer({ jobName, onClose }: { jobName: string | null; onClose: () => void }) {
  const [status, setStatus] = useState<JobRunStatus | 'ALL'>('ALL');
  const [page, setPage] = useState(1);

  const query = useQuery({
    queryKey: [jobApi.key, 'runs', jobName, status, page],
    queryFn: () =>
      jobApi.runs(jobName!, {
        page,
        limit: 20,
        status: status === 'ALL' ? undefined : status,
        order_by: 'id',
        order_dir: 'DESC',
      }),
    enabled: !!jobName,
  });

  const columns: ColumnsType<JobRun> = [
    { title: 'ID', dataIndex: 'id', width: 70, render: (v: number) => `#${v}` },
    {
      title: 'Bắt đầu',
      dataIndex: 'started_at',
      width: 155,
      render: (v: string) => formatDateTime(v),
    },
    {
      title: 'Thời lượng',
      dataIndex: 'duration_ms',
      align: 'right',
      width: 100,
      render: (v: number | null) => (v !== null && v !== undefined ? `${formatNumber(v)} ms` : '—'),
    },
    {
      title: 'Trạng thái',
      dataIndex: 'status',
      width: 115,
      render: (v: string) => <StatusTag status={v} map={JOB_RUN_STATUS_MAP} />,
    },
    {
      title: 'Ảnh hưởng',
      dataIndex: 'affected',
      align: 'right',
      width: 95,
      render: (v: number) => <b>{formatNumber(v)}</b>,
    },
    {
      title: 'Thông báo / Lỗi',
      dataIndex: 'message',
      render: (v: string) =>
        v ? (
          <Typography.Text type="danger">{v}</Typography.Text>
        ) : (
          <Typography.Text type="secondary">—</Typography.Text>
        ),
    },
  ];

  return (
    <Drawer
      open={!!jobName}
      onClose={onClose}
      width={720}
      title={jobName ? `Lịch sử chạy: ${jobName}` : 'Lịch sử chạy job'}
      extra={
        <Button
          size="small"
          icon={<ReloadOutlined spin={query.isFetching} />}
          onClick={() => void query.refetch()}
        >
          Làm mới
        </Button>
      }
    >
      <div style={{ marginBottom: 12 }}>
        <Segmented
          value={status}
          onChange={(v) => {
            setStatus(v as JobRunStatus | 'ALL');
            setPage(1);
          }}
          options={[
            { label: 'Tất cả', value: 'ALL' },
            { label: 'Thành công', value: 'SUCCESS' },
            { label: 'Thất bại', value: 'FAILED' },
            { label: 'Đang chạy', value: 'RUNNING' },
          ]}
        />
      </div>

      {query.isError ? (
        <Result
          status="warning"
          title="Không tải được lịch sử chạy job"
          subTitle={getErrorMessage(query.error)}
          extra={<Button onClick={() => void query.refetch()}>Thử lại</Button>}
        />
      ) : (
        <Table<JobRun>
          size="small"
          rowKey="id"
          loading={query.isLoading}
          dataSource={query.data?.items ?? []}
          columns={columns}
          scroll={{ x: 'max-content' }}
          locale={{ emptyText: 'Chưa có lượt chạy nào' }}
          pagination={
            (query.data?.total ?? 0) > 20
              ? {
                  current: page,
                  pageSize: 20,
                  total: query.data?.total,
                  onChange: setPage,
                  showSizeChanger: false,
                }
              : false
          }
        />
      )}
    </Drawer>
  );
}

function TestEmailModal({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { message } = App.useApp();
  const [form] = Form.useForm<{ to: string }>();

  const mutation = useMutation({
    mutationFn: (to: string) => jobApi.testEmail(to),
    onSuccess: (res) => {
      message.success(`Đã gửi email thử nghiệm tới ${res.to}`);
      form.resetFields();
      onClose();
    },
    onError: (err) => {
      if (!setFormError(form, err, { to: /email/i })) {
        showError(err, 'Không thể gửi email thử nghiệm');
      }
    },
  });

  const handleSubmit = async () => {
    const values = await form.validateFields();
    mutation.mutate(values.to.trim());
  };

  return (
    <Modal
      open={open}
      onCancel={onClose}
      onOk={() => void handleSubmit()}
      confirmLoading={mutation.isPending}
      title="Gửi email thử nghiệm (kiểm tra SMTP)"
      okText="Gửi thử"
      cancelText="Hủy"
      forceRender
    >
      <Form form={form} layout="vertical">
        <Form.Item
          name="to"
          label="Địa chỉ email người nhận"
          rules={[
            { required: true, whitespace: true, message: 'Vui lòng nhập địa chỉ email' },
            { type: 'email', message: 'Địa chỉ email không hợp lệ' },
          ]}
        >
          <Input
            placeholder="VD: admin@gymmaster.vn"
            autoFocus
            onPressEnter={() => void handleSubmit()}
          />
        </Form.Item>
      </Form>
    </Modal>
  );
}

export default function JobsPage() {
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const [historyJob, setHistoryJob] = useState<string | null>(null);
  const [emailModalOpen, setEmailModalOpen] = useState(false);

  const query = useQuery({
    queryKey: [jobApi.key, 'list'],
    queryFn: () => jobApi.list(),
  });

  const runJob = useMutation({
    mutationFn: (name: string) => jobApi.run(name),
    onSuccess: (res) => {
      void queryClient.invalidateQueries({ queryKey: [jobApi.key] });
      if (res.status === 'FAILED') {
        message.error(
          `Job "${res.job}" thất bại: ${res.message || 'Lỗi không xác định'} (ảnh hưởng ${formatNumber(res.affected)})`,
        );
      } else {
        message.success(
          `Đã chạy "${res.job}" thành công (${formatNumber(res.affected)} bản ghi bị ảnh hưởng, ${formatNumber(res.duration_ms ?? 0)} ms)`,
        );
      }
    },
    onError: (e) => showError(e, 'Không thể chạy job'),
  });

  const isMobile = useIsMobile();
  const { can } = usePermission();
  const canManage = can(PERM.JOB_MANAGE);

  const renderActions = (r: JobInfo, iconOnly: boolean) => (
    <Space size={6}>
      {canManage && (
        <Tooltip title={iconOnly ? 'Chạy ngay' : undefined}>
          <span>
            <ConfirmButton
              size="small"
              type="primary"
              icon={<PlayCircleOutlined />}
              disabled={r.running}
              confirmTitle={`Chạy ngay job "${r.name}"?`}
              confirmContent={r.description}
              okText="Chạy ngay"
              onConfirm={() => runJob.mutateAsync(r.name)}
              aria-label="Chạy ngay"
            >
              {iconOnly ? null : 'Chạy ngay'}
            </ConfirmButton>
          </span>
        </Tooltip>
      )}
      <Tooltip title={iconOnly ? 'Lịch sử chạy' : undefined}>
        <Button
          size="small"
          icon={<HistoryOutlined />}
          onClick={() => setHistoryJob(r.name)}
          aria-label="Lịch sử chạy"
        >
          {iconOnly ? null : 'Lịch sử'}
        </Button>
      </Tooltip>
    </Space>
  );

  const columns = useMemo<ColumnsType<JobInfo>>(
    () => [
      {
        title: 'Job định kỳ',
        key: 'name',
        width: 320,
        render: (_, r) => (
          <Space direction="vertical" size={2} style={{ maxWidth: 320 }}>
            <Space size={6}>
              <span className="code-text" style={{ fontWeight: 600 }}>
                {r.name}
              </span>
              {r.running && <Tag color="processing">Đang chạy</Tag>}
            </Space>
            <Typography.Paragraph
              type="secondary"
              style={{ fontSize: 12, marginBottom: 0 }}
              ellipsis={{ rows: 2, tooltip: r.description }}
            >
              {r.description}
            </Typography.Paragraph>
          </Space>
        ),
      },
      {
        title: 'Lịch Cron',
        dataIndex: 'spec',
        width: 130,
        render: (v: string) => <Tag className="code-text">{v}</Tag>,
      },
      {
        title: 'Trạng thái lịch',
        dataIndex: 'scheduled',
        width: 140,
        render: (scheduled: boolean) =>
          scheduled ? (
            <Tag color="green">Đang bật tự động</Tag>
          ) : (
            <Tag color="default">Tắt tự động</Tag>
          ),
      },
      {
        title: 'Lần chạy kế tiếp',
        dataIndex: 'next_run_at',
        width: 165,
        render: (v: string | null) => (v ? <DateText value={v} withTime /> : '—'),
      },
      {
        title: 'Lần chạy gần nhất',
        key: 'last_run',
        width: 230,
        render: (_, r) => {
          const lr = r.last_run;
          if (!lr) return <Typography.Text type="secondary">Chưa từng chạy</Typography.Text>;
          return (
            <Space direction="vertical" size={2}>
              <Space size={6} wrap>
                <StatusTag status={lr.status} map={JOB_RUN_STATUS_MAP} />
                <Typography.Text style={{ fontSize: 12 }}>
                  {formatDateTime(lr.started_at)}
                </Typography.Text>
                {lr.duration_ms !== null && (
                  <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                    ({formatNumber(lr.duration_ms)} ms · {formatNumber(lr.affected)} bản ghi)
                  </Typography.Text>
                )}
              </Space>
              {lr.message && (
                <Typography.Text type="danger" style={{ fontSize: 12 }}>
                  {lr.message}
                </Typography.Text>
              )}
            </Space>
          );
        },
      },
      {
        title: '',
        key: 'actions',
        fixed: 'right',
        width: 84,
        render: (_, r) => renderActions(r, true),
      },
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [runJob],
  );

  return (
    <Card variant="borderless" className="page-card">
      <Row justify="space-between" align="middle" gutter={[8, 12]} className="page-head">
        <Col>
          <PageTitle
            title="Job định kỳ"
            subtitle="Quản lý các tác vụ nền tự động của hệ thống, lịch sử thực thi và kiểm tra gửi email"
          />
        </Col>
        <Col>
          <Space wrap>
            <Button
              icon={<ReloadOutlined spin={query.isFetching} />}
              onClick={() => void query.refetch()}
            >
              Làm mới
            </Button>
            {canManage && (
              <Button icon={<MailOutlined />} onClick={() => setEmailModalOpen(true)}>
                Gửi email thử
              </Button>
            )}
          </Space>
        </Col>
      </Row>

      {query.isError ? (
        <Result
          status="warning"
          title="Không tải được danh sách job định kỳ"
          subTitle={getErrorMessage(query.error)}
          extra={<Button onClick={() => void query.refetch()}>Thử lại</Button>}
        />
      ) : isMobile ? (
        <List<JobInfo>
          loading={query.isLoading}
          dataSource={query.data ?? []}
          locale={{ emptyText: 'Chưa có job hệ thống nào' }}
          renderItem={(r) => (
            <List.Item style={{ display: 'block', paddingInline: 0 }}>
              <Space direction="vertical" size={6} style={{ width: '100%' }}>
                <Space size={6} wrap>
                  <span className="code-text" style={{ fontWeight: 600, wordBreak: 'break-all' }}>
                    {r.name}
                  </span>
                  {r.running && <Tag color="processing">Đang chạy</Tag>}
                  <Tag className="code-text">{r.spec}</Tag>
                </Space>
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                  {r.description}
                </Typography.Text>
                <Typography.Text style={{ fontSize: 12 }}>
                  Kế tiếp: {r.next_run_at ? formatDateTime(r.next_run_at) : '—'}
                  {' · '}Gần nhất:{' '}
                  {r.last_run ? (
                    <>
                      <StatusTag status={r.last_run.status} map={JOB_RUN_STATUS_MAP} />
                      {formatDateTime(r.last_run.started_at)}
                    </>
                  ) : (
                    'Chưa từng chạy'
                  )}
                </Typography.Text>
                {renderActions(r, false)}
              </Space>
            </List.Item>
          )}
        />
      ) : (
        <Table<JobInfo>
          size="small"
          rowKey="name"
          loading={query.isLoading}
          dataSource={query.data ?? []}
          columns={columns}
          pagination={false}
          scroll={{ x: 'max-content' }}
          locale={{ emptyText: 'Chưa có job hệ thống nào' }}
        />
      )}

      <JobRunsDrawer jobName={historyJob} onClose={() => setHistoryJob(null)} />
      <TestEmailModal open={emailModalOpen} onClose={() => setEmailModalOpen(false)} />
    </Card>
  );
}
