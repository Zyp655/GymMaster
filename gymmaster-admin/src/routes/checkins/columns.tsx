import { Space, Tag, Tooltip, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { Link } from 'react-router-dom';
import { CHECKIN_RESULT_MAP } from '@/api/checkin';
import type { CheckinItem } from '@/api/checkin';
import DateText from '@/components/DateText';
import { MemberAvatar } from '@/components/MemberSelect';
import { formatPhone } from '@/utils/format';
import { methodLabel } from './methods';

/** Thời lượng tập "1 giờ 15 phút" giữa hai thời điểm. */
export function durationText(from: string, to: string): string {
  const mins = Math.max(0, Math.round((new Date(to).getTime() - new Date(from).getTime()) / 60000));
  const h = Math.floor(mins / 60);
  return h > 0 ? `${h} giờ ${mins % 60} phút` : `${mins} phút`;
}

/** Cột bảng lượt ra vào – dùng cho trang Lịch sử ra vào và tab hồ sơ hội viên. */
export function checkinColumns({ showMember = true } = {}): ColumnsType<CheckinItem> {
  return [
    {
      title: 'Thời điểm',
      dataIndex: 'checked_in_at',
      sorter: true,
      width: 160,
      render: (v: string) => <DateText value={v} format="DD/MM/YYYY HH:mm:ss" />,
    },
    ...(showMember
      ? ([
          {
            title: 'Hội viên',
            dataIndex: 'member_name',
            render: (_: unknown, r: CheckinItem) => (
              <Space size={8}>
                <MemberAvatar src={r.member_avatar_url} name={r.member_name} />
                <div style={{ lineHeight: 1.25 }}>
                  {/* Lượt vào bằng vé ngày / vé tập thử không gắn hội viên (member_id = 0). */}
                  {r.member_id ? (
                    <Link to={`/app/members/${r.member_id}`}>{r.member_name}</Link>
                  ) : (
                    <Typography.Text>{r.member_name}</Typography.Text>
                  )}
                  <br />
                  <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                    {r.member_code} · {formatPhone(r.member_phone)}
                  </Typography.Text>
                </div>
              </Space>
            ),
          },
        ] as ColumnsType<CheckinItem>)
      : []),
    { title: 'Chi nhánh', dataIndex: 'branch_name', width: 150 },
    {
      title: 'Kết quả',
      dataIndex: 'result',
      width: 240,
      // Giới hạn bề rộng để lý do từ chối dài tự xuống dòng, không đẩy các cột sau ra khỏi khung nhìn.
      render: (_, r) => (
        <div style={{ maxWidth: 240, whiteSpace: 'normal' }}>
          <Tag color={CHECKIN_RESULT_MAP[r.result].color} bordered={false}>
            {CHECKIN_RESULT_MAP[r.result].label}
          </Tag>
          {r.result === 'DENIED' && (
            <Typography.Text type="danger" style={{ fontSize: 12 }}>
              {r.deny_message}
            </Typography.Text>
          )}
        </div>
      ),
    },
    {
      title: 'Gói',
      dataIndex: 'plan_name',
      render: (v: string | null) =>
        v ? (
          <div style={{ maxWidth: 180, whiteSpace: 'normal' }}>{v}</div>
        ) : (
          <Typography.Text type="secondary">—</Typography.Text>
        ),
    },
    {
      title: 'Phương thức',
      dataIndex: 'method',
      width: 130,
      render: (m: string) => <span style={{ whiteSpace: 'nowrap' }}>{methodLabel(m)}</span>,
    },
    {
      title: 'Ra về',
      dataIndex: 'checked_out_at',
      render: (v: string | null, r) =>
        v ? (
          <Tooltip title={`Thời gian tập: ${durationText(r.checked_in_at, v)}`}>
            <span>
              <DateText value={v} format="HH:mm" relative={false} />
            </span>
          </Tooltip>
        ) : r.result === 'ALLOWED' ? (
          <Typography.Text type="secondary">Chưa checkout</Typography.Text>
        ) : (
          '—'
        ),
    },
    {
      title: 'Nhân viên',
      dataIndex: 'staff_name',
      render: (v: string | null, r) => (
        <div style={{ maxWidth: 200, whiteSpace: 'normal' }}>
          {v ?? '—'}
          {r.note && (
            <>
              <br />
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                {r.note}
              </Typography.Text>
            </>
          )}
        </div>
      ),
    },
  ];
}
