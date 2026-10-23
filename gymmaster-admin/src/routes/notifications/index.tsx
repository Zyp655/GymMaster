import {
  App,
  Button,
  Card,
  Col,
  Empty,
  List,
  Pagination,
  Result,
  Row,
  Segmented,
  Select,
  Space,
  Tag,
  Tooltip,
} from 'antd';
import { CheckOutlined, ReloadOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import dayjs from 'dayjs';
import { useNavigate } from 'react-router-dom';
import notificationApi, {
  NOTIFICATION_TYPE_LABEL,
  notificationTypeLabel,
  toAppLink,
} from '@/api/notification';
import type { Notification, NotificationListParams } from '@/api/notification';
import PageTitle from '@/components/PageTitle';
import { useTableQuery } from '@/hooks/useTableQuery';
import { getErrorMessage, showError } from '@/utils/errors';
import { formatDateTime } from '@/utils/format';
import { NotificationIcon } from './meta';

const TYPE_OPTIONS = Object.entries(NOTIFICATION_TYPE_LABEL).map(([value, label]) => ({
  value,
  label,
}));

/** Trang "Tất cả thông báo" – phân trang server, lọc chưa đọc / loại. */
export default function NotificationsPage() {
  const { message } = App.useApp();
  const navigate = useNavigate();
  const qc = useQueryClient();

  const table = useTableQuery<Notification, NotificationListParams>({
    queryKey: [notificationApi.key, 'list'],
    queryFn: (p) => notificationApi.list(p),
    filterKeys: ['unread', 'type'],
    defaults: { limit: 20 },
  });
  const unreadOnly = table.params.unread === ('true' as unknown as boolean);

  const count = useQuery({
    queryKey: [notificationApi.key, 'unread-count'],
    queryFn: () => notificationApi.unreadCount(),
  });

  const invalidate = () => qc.invalidateQueries({ queryKey: [notificationApi.key] });
  const readOne = useMutation({
    mutationFn: (id: number) => notificationApi.markRead(id),
    onSettled: invalidate,
  });
  const readAll = useMutation({
    mutationFn: () => notificationApi.markAllRead(),
    onSuccess: (r) => message.success(`Đã đánh dấu ${r.updated} thông báo là đã đọc`),
    onError: (e) => showError(e, 'Không thể đánh dấu đã đọc'),
    onSettled: invalidate,
  });

  const open = (n: Notification) => {
    if (!n.is_read) readOne.mutate(n.id);
    const target = toAppLink(n.link);
    if (target) navigate(target);
  };

  const unread = count.data?.count ?? 0;

  return (
    <Card variant="borderless" className="page-card">
      <Row justify="space-between" align="middle" gutter={[8, 12]} className="page-head">
        <Col>
          <PageTitle
            title="Thông báo"
            subtitle={unread > 0 ? `${unread} thông báo chưa đọc` : 'Bạn đã đọc hết thông báo'}
          />
        </Col>
        <Col>
          <Space.Compact>
            <Button
              icon={<ReloadOutlined />}
              loading={table.isFetching}
              onClick={() => void table.refetch()}
            >
              Làm mới
            </Button>
            <Button
              type="primary"
              icon={<CheckOutlined />}
              disabled={unread === 0}
              loading={readAll.isPending}
              onClick={() => readAll.mutate()}
            >
              Đánh dấu tất cả đã đọc
            </Button>
          </Space.Compact>
        </Col>
      </Row>

      <Row gutter={[12, 12]} style={{ marginBottom: 16 }}>
        <Col>
          <Segmented
            value={unreadOnly ? 'unread' : 'all'}
            options={[
              { label: 'Tất cả', value: 'all' },
              { label: `Chưa đọc${unread ? ` (${unread})` : ''}`, value: 'unread' },
            ]}
            onChange={(v) => table.setFilters({ unread: v === 'unread' ? 'true' : undefined })}
          />
        </Col>
        <Col xs={24} sm={10} md={6}>
          <Select
            allowClear
            style={{ width: '100%' }}
            placeholder="Loại thông báo"
            options={TYPE_OPTIONS}
            value={table.params.type || undefined}
            onChange={(v?: string) => table.setFilters({ type: v })}
          />
        </Col>
      </Row>

      <Card
        style={{ borderRadius: 10, border: '1px solid var(--gm-border)' }}
        styles={{ body: { padding: 8 } }}
      >
        {table.isError ? (
          <Result
            status="warning"
            title="Không tải được thông báo"
            subTitle={getErrorMessage(table.error)}
            extra={<Button onClick={() => void table.refetch()}>Thử lại</Button>}
          />
        ) : (
          <List<Notification>
            loading={table.isLoading}
            dataSource={table.items}
            locale={{
              emptyText: (
                <Empty
                  image={Empty.PRESENTED_IMAGE_SIMPLE}
                  description={unreadOnly ? 'Không có thông báo chưa đọc' : 'Chưa có thông báo'}
                />
              ),
            }}
            renderItem={(n) => (
              <List.Item
                className={n.is_read ? 'notif-page-item' : 'notif-page-item notif-item--unread'}
                onClick={() => open(n)}
                extra={
                  !n.is_read ? (
                    <Button
                      size="small"
                      type="link"
                      onClick={(e) => {
                        e.stopPropagation();
                        readOne.mutate(n.id);
                      }}
                    >
                      Đã đọc
                    </Button>
                  ) : null
                }
              >
                <div className="notif-row">
                  <NotificationIcon type={n.type} size={38} />
                  <div className="notif-row__main">
                    <div className="notif-row__title">
                      {n.title}{' '}
                      <Tag bordered={false} style={{ marginLeft: 6, fontWeight: 400 }}>
                        {notificationTypeLabel(n.type)}
                      </Tag>
                    </div>
                    {n.body && (
                      <div className="notif-row__body notif-row__body--full">{n.body}</div>
                    )}
                    <Tooltip title={formatDateTime(n.created_at)}>
                      <div className="notif-row__time">
                        {dayjs(n.created_at).fromNow()} · {formatDateTime(n.created_at)}
                      </div>
                    </Tooltip>
                  </div>
                </div>
              </List.Item>
            )}
          />
        )}
      </Card>

      {table.total > (table.params.limit ?? 20) && (
        <Pagination
          style={{ marginTop: 16, textAlign: 'right' }}
          align="end"
          current={table.params.page}
          pageSize={table.params.limit}
          total={table.total}
          showSizeChanger
          pageSizeOptions={[10, 20, 50]}
          showTotal={(t) => `${t} thông báo`}
          onChange={(p, size) => table.setPage(p, size)}
        />
      )}
    </Card>
  );
}
