import { Avatar, Space, Tag, Typography } from 'antd';
import {
  CheckCircleFilled,
  CloseCircleFilled,
  UserOutlined,
  WarningFilled,
} from '@ant-design/icons';
import type { ScanResult } from '@/api/checkin';
import { formatDate, formatDateTime, formatPhone } from '@/utils/format';
import './checkin.css';

export interface ResultCardProps {
  result: ScanResult;
  /** Bản gọn (modal check-in trong hồ sơ). */
  compact?: boolean;
}

/** Thẻ kết quả check-in lớn: xanh (ALLOWED) / đỏ (DENIED). */
export default function ResultCard({ result: r, compact }: ResultCardProps) {
  const ok = r.result === 'ALLOWED';
  const sub = r.subscription;
  const avatarSize = compact ? 72 : 112;

  return (
    <div className={`checkin-result ${ok ? 'checkin-result--ok' : 'checkin-result--deny'}`}>
      <div className="checkin-result__head">
        {ok ? <CheckCircleFilled /> : <CloseCircleFilled />}
        <span>{ok ? 'MỜI VÀO' : 'TỪ CHỐI'}</span>
      </div>
      <div className="checkin-result__body">
        <Avatar
          size={avatarSize}
          src={r.member?.avatar_url || undefined}
          icon={<UserOutlined />}
          style={{ flexShrink: 0, background: '#fff', color: '#9CA3AF', border: '3px solid #fff' }}
        />
        <div style={{ minWidth: 0 }}>
          {r.member ? (
            <>
              <div className="checkin-result__name">{r.member.full_name}</div>
              <Typography.Text className="checkin-result__meta">
                {r.member.code} · {formatPhone(r.member.phone)}
              </Typography.Text>
            </>
          ) : (
            <div className="checkin-result__name">Không xác định hội viên</div>
          )}
          {sub && (
            <div className="checkin-result__sub">
              <b>{sub.plan_name}</b>
              {sub.status === 'FROZEN' && ' (bảo lưu)'}
              <br />
              {sub.type === 'SESSION' && sub.remaining_sessions !== null
                ? `Còn ${sub.remaining_sessions} buổi · `
                : ''}
              {sub.days_left === 0 ? 'Hết hạn hôm nay' : `Còn ${sub.days_left} ngày`} (HSD{' '}
              {formatDate(sub.end_date)})
            </div>
          )}
        </div>
      </div>
      {!ok && r.deny_message && <div className="checkin-result__deny">{r.deny_message}</div>}
      {r.previous_checkin && (
        <Typography.Text className="checkin-result__meta">
          Lần trước: {formatDateTime(r.previous_checkin.checked_in_at)} tại{' '}
          {r.previous_checkin.branch_name}
        </Typography.Text>
      )}
      {r.warnings?.length > 0 && (
        <Space size={[6, 6]} wrap style={{ marginTop: 10 }}>
          {r.warnings.map((w) => (
            <Tag
              key={w}
              color="gold"
              icon={<WarningFilled />}
              style={{ fontSize: 14, padding: '2px 8px' }}
            >
              {w}
            </Tag>
          ))}
        </Space>
      )}
    </div>
  );
}
