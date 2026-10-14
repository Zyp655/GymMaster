import { useState } from 'react';
import { Badge, Button, Empty, List, Popover, Result, Spin, Tooltip, Typography } from 'antd';
import { BellOutlined, CheckOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { useNavigate } from 'react-router-dom';
import { toAppLink } from '@/api/notification';
import { useNotifications } from '@/hooks/useNotifications';
import { formatDateTime } from '@/utils/format';
import { NotificationIcon } from '@/routes/notifications/meta';

/** Chuông thông báo (dữ liệu từ `useNotifications`): 10 thông báo mới nhất + badge chưa đọc. */
export function NotificationBell() {
  const [open, setOpen] = useState(false);
  const { items, unread, isLoading, isError, refetch, markRead, markAllRead } = useNotifications(
    { listEnabled: open },
  );
  const navigate = useNavigate();

  const go = (path: string) => {
    setOpen(false);
    navigate(path);
  };

  const content = (
    <div className="notif-pop">
      <div className="notif-head">
        <Typography.Text strong style={{ fontSize: 15 }}>
          Thông báo {unread > 0 ? <Typography.Text type="secondary">({unread} chưa đọc)</Typography.Text> : null}
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
