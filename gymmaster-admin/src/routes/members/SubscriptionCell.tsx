import { Tag, Typography } from 'antd';
import type { MemberActiveSubscription } from '@/api/member';
import { formatDate } from '@/utils/format';
import { daysLeftText, expiryColor } from './utils';

/** Ô "Gói hiện tại": tên gói + hạn dùng, tag đỏ/cam khi ≤ 7 ngày, "Chưa có gói" khi null. */
export default function SubscriptionCell({ sub }: { sub: MemberActiveSubscription | null }) {
  if (!sub) {
    return (
      <Tag bordered={false} color="default">
        Chưa có gói
      </Tag>
    );
  }
  const color = expiryColor(sub.days_left);
  return (
    <div style={{ lineHeight: 1.3 }}>
      <Typography.Text>{sub.plan_name}</Typography.Text>
      {sub.status === 'FROZEN' && (
        <Tag color="blue" bordered={false} style={{ marginLeft: 6 }}>
          Bảo lưu
        </Tag>
      )}
      <br />
      <Typography.Text type="secondary" style={{ fontSize: 12 }}>
        HSD {formatDate(sub.end_date)}
        {sub.remaining_sessions !== null && ` · còn ${sub.remaining_sessions} buổi`}
      </Typography.Text>
      {color && (
        <Tag color={color} bordered={false} style={{ marginLeft: 6, fontSize: 11 }}>
          {daysLeftText(sub.days_left)}
        </Tag>
      )}
    </div>
  );
}
