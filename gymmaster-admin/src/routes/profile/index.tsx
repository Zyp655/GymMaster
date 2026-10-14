import { useMemo, useState } from 'react';
import {
  Avatar,
  Button,
  Card,
  Col,
  Collapse,
  Descriptions,
  Row,
  Space,
  Tag,
  Typography,
} from 'antd';
import { KeyOutlined, MailOutlined, PhoneOutlined } from '@ant-design/icons';
import ChangePasswordModal from '@/components/ChangePasswordModal';
import PageTitle from '@/components/PageTitle';
import { PERMISSION_MODULE_LABELS, ROLE, ROLE_LABELS } from '@/constants/permissions';
import { useBranchOptions } from '@/hooks/useBranchOptions';
import { usePermission } from '@/hooks/usePermission';
import { formatPhone } from '@/utils/format';

/** Hồ sơ cá nhân: thông tin tài khoản, vai trò, chi nhánh, quyền + đổi mật khẩu. */
export default function ProfilePage() {
  const { user, isOwner } = usePermission();
  const { branches } = useBranchOptions();
  const [pwdOpen, setPwdOpen] = useState(false);

  const permGroups = useMemo(() => {
    const map = new Map<string, string[]>();
    (user?.permissions ?? []).forEach((code) => {
      const mod = code.split('.')[0];
      map.set(mod, [...(map.get(mod) ?? []), code]);
    });
    return [...map.entries()].sort((a, b) => a[0].localeCompare(b[0]));
  }, [user?.permissions]);

  if (!user) return null;
  const name = user.full_name || user.email;

  const branchText =
    user.branch_ids.length === 0 ? (
      <Tag color="green" bordered={false}>
        Toàn hệ thống
      </Tag>
    ) : (
      user.branch_ids.map((id) => (
        <Tag key={id} bordered={false}>
          {branches.find((b) => b.id === id)?.name ?? `Chi nhánh #${id}`}
        </Tag>
      ))
    );

  return (
    <Card variant="borderless" className="page-card">
      <div className="page-head">
        <PageTitle title="Hồ sơ cá nhân" back />
      </div>

      <Row gutter={[24, 24]}>
        <Col xs={24} md={8}>
          <Card style={{ borderRadius: 12, textAlign: 'center' }}>
            <Avatar
              size={88}
              src={user.avatar_url || undefined}
              className="user-avatar"
              style={{ fontSize: 34 }}
            >
              {name.charAt(0).toUpperCase()}
            </Avatar>
            <Typography.Title level={4} style={{ margin: '14px 0 4px' }}>
              {name}
            </Typography.Title>
            <Space wrap style={{ justifyContent: 'center' }}>
              {user.roles.map((r) => (
                <Tag key={r} color={r === ROLE.OWNER ? 'gold' : 'blue'} bordered={false}>
                  {ROLE_LABELS[r] ?? r}
                </Tag>
              ))}
            </Space>
            <div style={{ marginTop: 20 }}>
              <Button icon={<KeyOutlined />} onClick={() => setPwdOpen(true)} block>
                Đổi mật khẩu
              </Button>
            </div>
          </Card>
        </Col>

        <Col xs={24} md={16}>
          <Descriptions
            bordered
            size="middle"
            column={1}
            title="Thông tin tài khoản"
            items={[
              {
                key: 'email',
                label: (
                  <>
                    <MailOutlined /> Email
                  </>
                ),
                children: user.email,
              },
              {
                key: 'phone',
                label: (
                  <>
                    <PhoneOutlined /> Số điện thoại
                  </>
                ),
                children: formatPhone(user.phone),
              },
              { key: 'branches', label: 'Chi nhánh làm việc', children: branchText },
              {
                key: 'perms',
                label: 'Số quyền',
                children: isOwner ? 'Toàn quyền' : `${user.permissions.length} quyền`,
              },
            ]}
          />
          <Typography.Paragraph type="secondary" style={{ marginTop: 8, fontSize: 13 }}>
            Cần thay đổi họ tên, số điện thoại hoặc chi nhánh? Vui lòng liên hệ quản lý trung tâm.
          </Typography.Paragraph>

          {!isOwner && permGroups.length > 0 && (
            <Collapse
              style={{ marginTop: 8 }}
              items={[
                {
                  key: 'perms',
                  label: 'Quyền của tôi',
                  children: (
                    <Space direction="vertical" size={8} style={{ width: '100%' }}>
                      {permGroups.map(([mod, codes]) => (
                        <div key={mod}>
                          <Typography.Text strong>
                            {PERMISSION_MODULE_LABELS[mod] ?? mod}:{' '}
                          </Typography.Text>
                          {codes.map((c) => (
                            <Tag key={c} bordered={false} style={{ marginBottom: 4 }}>
                              {c}
                            </Tag>
                          ))}
                        </div>
                      ))}
                    </Space>
                  ),
                },
              ]}
            />
          )}
        </Col>
      </Row>

      <ChangePasswordModal open={pwdOpen} onClose={() => setPwdOpen(false)} />
    </Card>
  );
}
