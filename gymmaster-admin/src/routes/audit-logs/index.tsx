import { useCallback, useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  Button,
  Card,
  Col,
  Collapse,
  DatePicker,
  Descriptions,
  Divider,
  Drawer,
  Empty,
  Result,
  Row,
  Skeleton,
  Space,
  Table,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { EyeOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import auditApi, { auditActionLabel, auditEntityLabel } from '@/api/audit';
import type { AuditChange, AuditLog, AuditLogListParams } from '@/api/audit';
import ActionBar from '@/components/ActionBar';
import DataTable from '@/components/DataTable';
import DateText from '@/components/DateText';
import FilterBar from '@/components/FilterBar';
import MobileList from '@/components/MobileList';
import type { MobileFetchArgs } from '@/components/MobileList';
import PageTitle from '@/components/PageTitle';
import { useIsMobile } from '@/hooks/useIsMobile';
import { useTableQuery } from '@/hooks/useTableQuery';
import { useUrlParams } from '@/hooks/useUrlParams';
import { getErrorMessage } from '@/utils/errors';
import { DATE_FORMAT, formatDateTime, toApiDate } from '@/utils/format';

const TITLE = 'Nhật ký thao tác';
const SUBTITLE = 'Lịch sử thay đổi dữ liệu và thao tác nghiệp vụ trên hệ thống';

const ACTION_COLOR: Record<string, string> = {
  CREATE: 'green',
  UPDATE: 'blue',
  DELETE: 'red',
  VOID: 'red',
  CANCEL: 'red',
  SELL: 'cyan',
  PAYMENT: 'green',
  REFUND: 'purple',
  RENEW: 'geekblue',
  UPGRADE: 'gold',
  TRANSFER: 'orange',
  IMPORT: 'green',
  ADJUST: 'gold',
  LOCK: 'red',
  UNLOCK: 'green',
};

function formatDiffValue(val: unknown): string {
  if (val === null || val === undefined) return '—';
  if (typeof val === 'string') return val === '' ? '""' : val;
  if (typeof val === 'number' || typeof val === 'boolean') return String(val);
  try {
    return JSON.stringify(val);
  } catch {
    return String(val);
  }
}

function AuditDetailDrawer({ logId, onClose }: { logId: number | null; onClose: () => void }) {
  const query = useQuery({
    queryKey: [auditApi.key, 'detail', logId],
    queryFn: () => auditApi.get(logId!),
    enabled: !!logId,
  });

  const d = query.data;

  return (
    <Drawer
      open={!!logId}
      onClose={onClose}
      width={720}
      title={d ? `Chi tiết nhật ký #${d.id}` : 'Chi tiết nhật ký'}
    >
      {query.isLoading ? (
        <Skeleton active paragraph={{ rows: 8 }} />
      ) : query.isError || !d ? (
        <Result
          status="warning"
          title="Không tải được chi tiết nhật ký"
          subTitle={getErrorMessage(query.error)}
          extra={<Button onClick={() => void query.refetch()}>Thử lại</Button>}
        />
      ) : (
        <>
          <Descriptions
            size="small"
            column={{ xs: 1, sm: 2 }}
            bordered
            items={[
              {
                label: 'Thời gian',
                children: formatDateTime(d.created_at),
              },
              {
                label: 'Người thao tác',
                children: (
                  <span>
                    <b>{d.actor_name || 'Hệ thống'}</b>
                    {d.actor_email ? ` (${d.actor_email})` : ''}
                  </span>
                ),
              },
              {
                label: 'Hành động',
                children: (
                  <Tag color={ACTION_COLOR[d.action] ?? 'default'}>
                    {auditActionLabel(d.action)} ({d.action})
                  </Tag>
                ),
              },
              {
                label: 'Đối tượng',
                children: (
                  <span>
                    {auditEntityLabel(d.entity)}{' '}
                    <Typography.Text type="secondary" className="code-text">
                      ({d.entity}
                      {d.entity_id ? ` #${d.entity_id}` : ''})
                    </Typography.Text>
                  </span>
                ),
              },
              {
                label: 'Địa chỉ IP',
                children: <span className="code-text">{d.ip || '—'}</span>,
                span: 2,
              },
            ]}
          />

          <Divider orientation="left" plain>
            Các trường thay đổi ({d.changes?.length ?? 0})
          </Divider>

          {!d.changes?.length ? (
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description="Không có khác biệt trường cấp 1"
            />
          ) : (
            <Table<AuditChange>
              size="small"
              rowKey={(r, i) => r.field || String(i)}
              pagination={false}
              dataSource={d.changes}
              scroll={{ x: 'max-content' }}
              columns={[
                {
                  title: 'Trường',
                  dataIndex: 'field',
                  width: 180,
                  render: (v: string) => <span className="code-text">{v || '(gốc)'}</span>,
                },
                {
                  title: 'Trước',
                  dataIndex: 'before',
                  render: (v: unknown) => (
                    <Typography.Text type="secondary" style={{ wordBreak: 'break-all' }}>
                      {formatDiffValue(v)}
                    </Typography.Text>
                  ),
                },
                {
                  title: 'Sau',
                  dataIndex: 'after',
                  render: (v: unknown) => (
                    <Typography.Text strong style={{ wordBreak: 'break-all' }}>
                      {formatDiffValue(v)}
                    </Typography.Text>
                  ),
                },
              ]}
            />
          )}

          <Collapse
            style={{ marginTop: 16 }}
            size="small"
            items={[
              {
                key: 'before',
                label: 'Dữ liệu JSON trước (before)',
                children: (
                  <pre style={{ margin: 0, fontSize: 12, overflowX: 'auto', maxHeight: 280 }}>
                    {d.before ? JSON.stringify(d.before, null, 2) : 'null'}
                  </pre>
                ),
              },
              {
                key: 'after',
                label: 'Dữ liệu JSON sau (after)',
                children: (
                  <pre style={{ margin: 0, fontSize: 12, overflowX: 'auto', maxHeight: 280 }}>
                    {d.after ? JSON.stringify(d.after, null, 2) : 'null'}
                  </pre>
                ),
              },
            ]}
          />
        </>
      )}
    </Drawer>
  );
}

function useAuditFilterOptions() {
  const q = useQuery({
    queryKey: [auditApi.key, 'options'],
    queryFn: () => auditApi.options(),
    staleTime: 5 * 60_000,
  });

  const entityOptions = useMemo(
    () => (q.data?.entities ?? []).map((e) => ({ value: e, label: `${auditEntityLabel(e)} (${e})` })),
    [q.data?.entities],
  );

  const actionOptions = useMemo(
    () => (q.data?.actions ?? []).map((a) => ({ value: a, label: `${auditActionLabel(a)} (${a})` })),
    [q.data?.actions],
  );

  return { entityOptions, actionOptions };
}

function AuditLogsDesktop() {
  const url = useUrlParams();
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const { entityOptions, actionOptions } = useAuditFilterOptions();

  const table = useTableQuery<AuditLog, AuditLogListParams>({
    queryKey: [auditApi.key, 'list'],
    queryFn: (p) => auditApi.list(p),
    filterKeys: ['entity', 'action', 'from', 'to'],
    defaults: { order_by: 'a.id', order_dir: 'DESC' },
  });

  const from = url.get('from');
  const to = url.get('to');

  const columns = useMemo<ColumnsType<AuditLog>>(
    () => [
      {
        title: 'ID',
        dataIndex: 'id',
        width: 80,
        render: (v: number) => (
          <Button
            type="link"
            style={{ padding: 0 }}
            className="code-text"
            onClick={() => setSelectedId(v)}
          >
            #{v}
          </Button>
        ),
      },
      {
        title: 'Thời gian',
        dataIndex: 'created_at',
        width: 160,
        render: (v: string) => <DateText value={v} withTime />,
      },
      {
        title: 'Người thao tác',
        key: 'actor',
        render: (_, r) => (
          <Space direction="vertical" size={0}>
            <b>{r.actor_name || 'Hệ thống'}</b>
            {r.actor_email && (
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                {r.actor_email}
              </Typography.Text>
            )}
          </Space>
        ),
      },
      {
        title: 'Hành động',
        dataIndex: 'action',
        render: (v: string) => (
          <Tag color={ACTION_COLOR[v] ?? 'default'}>{auditActionLabel(v)}</Tag>
        ),
      },
      {
        title: 'Đối tượng',
        key: 'entity',
        render: (_, r) => (
          <span>
            {auditEntityLabel(r.entity)}{' '}
            {r.entity_id ? (
              <Typography.Text type="secondary" className="code-text">
                #{r.entity_id}
              </Typography.Text>
            ) : null}
          </span>
        ),
      },
      {
        title: 'IP',
        dataIndex: 'ip',
        responsive: ['md'],
        render: (v: string) => <span className="code-text">{v || '—'}</span>,
      },
      {
        title: '',
        key: 'actions',
        fixed: 'right',
        width: 60,
        render: (_, r) => (
          <Tooltip title="Xem chi tiết thay đổi">
            <Button
              type="text"
              size="small"
              icon={<EyeOutlined />}
              onClick={() => setSelectedId(r.id)}
              aria-label="Chi tiết"
            />
          </Tooltip>
        ),
      },
    ],
    [],
  );

  return (
    <Card variant="borderless" className="page-card">
      <Row justify="space-between" align="middle" gutter={[8, 12]} className="page-head">
        <Col>
          <PageTitle title={TITLE} subtitle={SUBTITLE} />
        </Col>
        <Col>
          <ActionBar onRefresh={() => table.refetch()} refreshing={table.isFetching} />
        </Col>
      </Row>

      <FilterBar
        searchPlaceholder="Tìm người thao tác, email, đối tượng..."
        colTextSearch={6}
        filters={[
          { name: 'entity', placeholder: 'Đối tượng', options: entityOptions, col: 4 },
          { name: 'action', placeholder: 'Hành động', options: actionOptions, col: 4 },
        ]}
      >
        <Col xs={24} sm={12} md={6}>
          <DatePicker.RangePicker
            style={{ width: '100%' }}
            format={DATE_FORMAT}
            placeholder={['Từ ngày', 'Đến ngày']}
            value={from || to ? [from ? dayjs(from) : null, to ? dayjs(to) : null] : null}
            allowEmpty={[true, true]}
            presets={[
              { label: 'Hôm nay', value: [dayjs(), dayjs()] },
              { label: '7 ngày qua', value: [dayjs().subtract(6, 'day'), dayjs()] },
              { label: 'Tháng này', value: [dayjs().startOf('month'), dayjs()] },
            ]}
            onChange={(v) =>
              url.set(
                { from: toApiDate(v?.[0]) ?? undefined, to: toApiDate(v?.[1]) ?? undefined },
                { resetPage: true },
              )
            }
          />
        </Col>
      </FilterBar>

      <DataTable<AuditLog>
        table={table}
        columns={columns}
        emptyText="Chưa có nhật ký thao tác nào"
        onRow={(r) => ({
          onClick: () => setSelectedId(r.id),
          style: { cursor: 'pointer' },
        })}
      />

      <AuditDetailDrawer logId={selectedId} onClose={() => setSelectedId(null)} />
    </Card>
  );
}

function AuditLogsMobile() {
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const { entityOptions, actionOptions } = useAuditFilterOptions();

  const fetchPage = useCallback(({ page, limit, keyword, filters }: MobileFetchArgs) => {
    const range = filters.range as [Dayjs | null, Dayjs | null] | undefined;
    return auditApi.list({
      page,
      limit,
      keyword: keyword || undefined,
      order_by: 'a.id',
      order_dir: 'DESC',
      entity: (filters.entity as string) || undefined,
      action: (filters.action as string) || undefined,
      from: toApiDate(range?.[0]) ?? undefined,
      to: toApiDate(range?.[1]) ?? undefined,
    });
  }, []);

  return (
    <>
      <MobileList<AuditLog>
        title={TITLE}
        queryKey={[auditApi.key, 'list']}
        fetchPage={fetchPage}
        onOpen={(r) => setSelectedId(r.id)}
        searchPlaceholder="Tìm người thao tác, đối tượng..."
        emptyText="Chưa có nhật ký thao tác nào"
        filters={[
          { name: 'entity', label: 'Đối tượng', options: entityOptions },
          { name: 'action', label: 'Hành động', options: actionOptions },
          { name: 'range', label: 'Khoảng ngày', type: 'daterange' },
        ]}
        row={{
          title: (r) =>
            `${auditEntityLabel(r.entity)}${r.entity_id ? ` #${r.entity_id}` : ''}`,
          subtitle: (r) => `${r.actor_name || 'Hệ thống'}${r.actor_email ? ` · ${r.actor_email}` : ''}`,
          meta: (r) => [formatDateTime(r.created_at), r.ip],
          tags: (r) => [
            {
              label: auditActionLabel(r.action),
              color: ACTION_COLOR[r.action] ?? 'default',
            },
          ],
        }}
      />
      <AuditDetailDrawer logId={selectedId} onClose={() => setSelectedId(null)} />
    </>
  );
}

export default function AuditLogsPage() {
  return useIsMobile() ? <AuditLogsMobile /> : <AuditLogsDesktop />;
}
