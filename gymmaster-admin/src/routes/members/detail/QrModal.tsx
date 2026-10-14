import { useEffect, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { App, Button, Modal, Progress, QRCode, Space, Typography } from 'antd';
import { CopyOutlined, ReloadOutlined } from '@ant-design/icons';
import memberApi from '@/api/member';
import type { Member } from '@/api/member';
import { getErrorMessage } from '@/utils/errors';

/** Làm mới mã khi còn ≤ N giây. */
const REFRESH_BEFORE = 5;

/** Hiển thị QR check-in động của hội viên (`GET /admin/members/{id}/qr`), tự làm mới trước khi hết hạn. */
export default function QrModal({
  member,
  open,
  onClose,
}: {
  member: Member;
  open: boolean;
  onClose: () => void;
}) {
  const { message } = App.useApp();
  const [now, setNow] = useState(() => Date.now());

  const query = useQuery({
    queryKey: [memberApi.key, 'qr', member.id],
    queryFn: () => memberApi.qr(member.id),
    enabled: open,
    gcTime: 0,
    staleTime: Infinity,
  });
  const qr = query.data;

  useEffect(() => {
    if (!open) return;
    const t = setInterval(() => setNow(Date.now()), 500);
    return () => clearInterval(t);
  }, [open]);

  const expiresAt = qr ? new Date(qr.expires_at).getTime() : 0;
  const left = Math.max(0, Math.ceil((expiresAt - now) / 1000));
  const { refetch, isFetching } = query;

  useEffect(() => {
    if (open && qr && left <= REFRESH_BEFORE && !isFetching) void refetch();
  }, [open, qr, left, isFetching, refetch]);

  const copy = async () => {
    if (!qr) return;
    try {
      await navigator.clipboard.writeText(qr.token);
      message.success('Đã sao chép mã');
    } catch {
      message.error('Trình duyệt không cho phép sao chép');
    }
  };

  return (
    <Modal
      open={open}
      title={`Mã QR check-in · ${member.full_name}`}
      onCancel={onClose}
      footer={<Button onClick={onClose}>Đóng</Button>}
      width={420}
    >
      <div style={{ textAlign: 'center' }}>
        {query.isError ? (
          <Space direction="vertical">
            <Typography.Text type="danger">
              {getErrorMessage(query.error, 'Không lấy được mã QR')}
            </Typography.Text>
            <Button onClick={() => refetch()}>Thử lại</Button>
          </Space>
        ) : (
          <>
            <QRCode
              value={qr?.token ?? '-'}
              size={240}
              status={!qr ? 'loading' : left === 0 ? 'expired' : 'active'}
              onRefresh={() => refetch()}
              style={{ margin: '8px auto' }}
            />
            {qr && (
              <>
                <Typography.Paragraph
                  code
                  copyable={false}
                  style={{ wordBreak: 'break-all', marginTop: 8 }}
                >
                  {qr.token}
                </Typography.Paragraph>
                <Progress
                  percent={Math.round((left / qr.ttl_seconds) * 100)}
                  showInfo={false}
                  size="small"
                  strokeColor={left <= 10 ? '#F97316' : '#0A3150'}
                />
                <Typography.Text type="secondary">
                  {left > 0 ? `Hết hạn sau ${left} giây – tự làm mới` : 'Mã đã hết hạn'}
                </Typography.Text>
                <div style={{ marginTop: 12 }}>
                  <Space>
                    <Button icon={<CopyOutlined />} onClick={copy}>
                      Sao chép mã
                    </Button>
                    <Button icon={<ReloadOutlined spin={isFetching} />} onClick={() => refetch()}>
                      Làm mới
                    </Button>
                  </Space>
                </div>
              </>
            )}
          </>
        )}
      </div>
    </Modal>
  );
}
