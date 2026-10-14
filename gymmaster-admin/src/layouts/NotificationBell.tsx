import { Badge, Button, Empty, List, Popover, Spin, Typography } from 'antd';
import { BellOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import DateText from '@/components/DateText';
import { useNotifications } from '@/hooks/useNotifications';

/** Chuông thông báo (dữ liệu từ `useNotifications`). */
export function NotificationBell() {
  const { items, unread, isLoading, markRead, markAllRead } = useNotifications();
  const navigate = useNavigate();

  const content = (
    <div style={{ width: 320 }}>
      <div className="notif-head">
        <Typography.Text strong>Thông báo</Typography.Text>
        {unread > 0 && (
          <Button type="link" size="small" onClick={markAllRead}>
            Đánh dấu đã đọc
          </Button>
        )}
      </div>
      {isLoading ? (
        <div style={{ textAlign: 'center', padding: 24 }}>
          <Spin />
        </div>
      ) : items.length === 0 ? (
        <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="Chưa có thông báo" />
      ) : (
        <List
          className="notif-list"
          dataSource={items}
          renderItem={(n) => (
            <List.Item
              className={n.read ? 'notif-item' : 'notif-item notif-item--unread'}
              onClick={() => {
                markRead(n.id);
                if (n.link) navigate(n.link);
              }}
            >
              <List.Item.Meta
                title={n.title}
                description={
                  <>
                    {n.body && <div>{n.body}</div>}
                    <DateText value={n.created_at} withTime />
                  </>
                }
              />
            </List.Item>
          )}
        />
      )}
    </div>
  );

  return (
    <Popover content={content} trigger="click" placement="bottomRight" arrow={false}>
      <Badge count={unread} size="small" offset={[-4, 4]}>
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
