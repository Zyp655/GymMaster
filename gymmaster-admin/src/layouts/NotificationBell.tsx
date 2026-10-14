import { useEffect, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { App, Badge, Button, Empty, List, Popover, Result, Spin, Tooltip, Typography } from 'antd';
import { BellOutlined, CheckOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { useNavigate } from 'react-router-dom';
import notificationApi, { toAppLink } from '@/api/notification';
import { appConfig } from '@/config';
import { useNotifications } from '@/hooks/useNotifications';
import { NotificationIcon } from '@/routes/notifications/meta';
import { useAppSelector } from '@/store';
import { formatDateTime } from '@/utils/format';
import { tokenStore } from '@/utils/token';

/** Chuông thông báo (dữ liệu từ `useNotifications` + kết nối SSE `/v1/notifications/stream`): 10 thông báo mới nhất + badge chưa đọc. */
export function NotificationBell() {
  const [open, setOpen] = useState(false);
  const { items, unread, isLoading, isError, refetch, markRead, markAllRead } = useNotifications({
    listEnabled: open,
  });
  const navigate = useNavigate();
  const qc = useQueryClient();
  const { notification } = App.useApp();
  const loggedIn = useAppSelector((s) => !!s.auth.user);

  useEffect(() => {
    if (!loggedIn || typeof EventSource === 'undefined') return;
    const accessToken = tokenStore.getAccessToken();
    if (!accessToken) return;

    const base = appConfig.apiUrl.replace(/\/+$/, '');
    const streamPath = base.endsWith('/v1')
      ? `${base}/notifications/stream`
      : `${base}/v1/notifications/stream`;
    const url = `${streamPath}?token=${encodeURIComponent(accessToken)}`;

    let es: EventSource | null = null;
    try {
      es = new EventSource(url);
    } catch {
      return;
    }

    const handleIncoming = (event: MessageEvent) => {
      void qc.invalidateQueries({ queryKey: [notificationApi.key] });
      if (!event.data) return;
      try {
        const payload = JSON.parse(String(event.data)) as {
          title?: string;
          body?: string;
          link?: string;
        };
        if (payload?.title) {
          notification.info({
            message: payload.title,
            description: payload.body,
            placement: 'bottomRight',
            duration: 5,
            onClick: () => {
              const target = toAppLink(payload.link);
              if (target) navigate(target);
            },
          });
        }
      } catch {
        // Sự kiện ping hoặc dữ liệu không phải JSON
      }
    };

    es.addEventListener('notification', handleIncoming as EventListener);
    es.onmessage = handleIncoming;
    es.onerror = () => {
      // Lỗi kết nối SSE -> đóng để tự động fallback về polling của useNotifications
      es?.close();
    };

    return () => {
      es?.close();
    };
  }, [loggedIn, qc, notification, navigate]);

  const go = (path: string) => {
    setOpen(false);
    navigate(path);
  };

  const content = (
    <div className="notif-pop">
      <div className="notif-head">
        <Typography.Text strong style={{ fontSize: 15 }}>
          Thông báo{' '}
          {unread > 0 ? (
            <Typography.Text type="secondary">({unread} chưa đọc)</Typography.Text>
          ) : null}
        </Typography.Text>
        {unread > 0 && (
          <Button type="link" size="small" icon={<CheckOutlined />} onClick={markAllRead}>
            Đánh dấu tất cả đã đọc
          </Button>
        )}
      </div>
      {isLoading ? (
        <div style={{ textAlign: 'center', padding: 32 }}>
          <Spin />
        </div>
      ) : isError ? (
        <Result
          status="warning"
          subTitle="Không tải được thông báo"
          extra={
            <Button size="small" onClick={refetch}>
              Thử lại
            </Button>
          }
        />
      ) : items.length === 0 ? (
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description="Chưa có thông báo"
          style={{ margin: '24px 0' }}
        />
      ) : (
        <List
          className="notif-list"
          dataSource={items}
          renderItem={(n) => (
            <List.Item
              className={n.read ? 'notif-item' : 'notif-item notif-item--unread'}
              onClick={() => {
                if (!n.read) markRead(n.id);
                const target = toAppLink(n.link);
                if (target) go(target);
              }}
            >
              <div className="notif-row">
                <NotificationIcon type={n.type} />
                <div className="notif-row__main">
                  <div className="notif-row__title">{n.title}</div>
                  {n.body && <div className="notif-row__body">{n.body}</div>}
                  <Tooltip title={formatDateTime(n.created_at)}>
                    <div className="notif-row__time">{dayjs(n.created_at).fromNow()}</div>
                  </Tooltip>
                </div>
              </div>
            </List.Item>
          )}
        />
      )}
      <div className="notif-foot">
        <Button type="link" onClick={() => go('/app/notifications')}>
          Xem tất cả
        </Button>
      </div>
    </div>
  );

  return (
    <Popover
      content={content}
      trigger="click"
      placement="bottomRight"
      arrow={false}
      open={open}
      onOpenChange={setOpen}
    >
      <Badge count={unread} size="small" offset={[-4, 4]} overflowCount={99}>
        <Button
          type="text"
          shape="circle"
          icon={<BellOutlined style={{ fontSize: 18 }} />}
          aria-label="Thông báo"
        />
      </Badge>
    </Popover>
  );
}

export default NotificationBell;
