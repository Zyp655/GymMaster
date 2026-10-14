import { Card, Col, Descriptions, Empty, Row, Statistic, Tag, Typography } from 'antd';
import { GENDER_LABEL, SOURCE_LABEL } from '@/api/member';
import type { Member } from '@/api/member';
import DateText from '@/components/DateText';
import { formatDate, formatDateTime, formatPhone } from '@/utils/format';
import { daysLeftText, expiryColor } from '../utils';

const dash = (v?: string | null) => (v ? v : '—');

/** Tab "Tổng quan": thông tin chi tiết + thống kê check-in + gói hiện tại + lần đo gần nhất. */
export default function OverviewTab({ member: m }: { member: Member }) {
  const sub = m.active_subscription;
  const metric = m.latest_metric;
  return (
    <Row gutter={[16, 16]}>
      <Col xs={24} lg={15}>
        <Card size="small" title="Thông tin cá nhân">
          <Descriptions
            size="small"
            column={{ xs: 1, sm: 2, md: 2, lg: 2, xl: 2, xxl: 2 }}
            items={[
              { label: 'Họ tên', children: m.full_name },
              {
                label: 'Mã hội viên',
                children: <span className="code-text">{m.member_code}</span>,
              },
              { label: 'Số điện thoại', children: formatPhone(m.phone) },
              { label: 'Email', children: dash(m.email) },
              { label: 'Giới tính', children: GENDER_LABEL[m.gender] },
              { label: 'Ngày sinh', children: m.dob ? formatDate(m.dob) : '—' },
              { label: 'CCCD', children: dash(m.id_card) },
              { label: 'Nguồn', children: SOURCE_LABEL[m.source] },
              { label: 'Địa chỉ', children: dash(m.address), span: { xs: 1, sm: 2 } },
              {
                label: 'Liên hệ khẩn cấp',
                span: { xs: 1, sm: 2 },
                children:
                  m.emergency_contact_name || m.emergency_contact_phone
                    ? [
                        m.emergency_contact_name,
                        m.emergency_contact_phone && formatPhone(m.emergency_contact_phone),
                      ]
                        .filter(Boolean)
                        .join(' · ')
                    : '—',
              },
              {
                label: 'Tài khoản cổng',
                children: m.portal_account ? (
                  <>
                    {m.portal_account.email}{' '}
                    <Tag
                      color={m.portal_account.status === 'ACTIVE' ? 'green' : 'red'}
                      bordered={false}
                    >
                      {m.portal_account.status === 'ACTIVE' ? 'Hoạt động' : 'Đã khóa'}
                    </Tag>
                  </>
                ) : (
                  'Chưa cấp'
                ),
              },
              {
                label: 'Khóa đặt lớp đến',
                children: m.booking_blocked_until ? formatDateTime(m.booking_blocked_until) : '—',
              },
              { label: 'Ngày tạo', children: <DateText value={m.created_at} withTime /> },
              { label: 'Cập nhật', children: <DateText value={m.updated_at} withTime /> },
              { label: 'Ghi chú', children: dash(m.note), span: { xs: 1, sm: 2 } },
            ]}
          />
        </Card>
      </Col>
      <Col xs={24} lg={9}>
        <Row gutter={[16, 16]}>
          <Col span={24}>
            <Card size="small" title="Gói hiện tại">
              {sub ? (
                <>
                  <Typography.Title level={5} style={{ margin: 0 }}>
                    {sub.plan_name}{' '}
                    {sub.status === 'FROZEN' && (
                      <Tag color="blue" bordered={false}>
                        Bảo lưu
                      </Tag>
                    )}
                  </Typography.Title>
                  <Typography.Text type="secondary">
                    {formatDate(sub.start_date)} → {formatDate(sub.end_date)}
                  </Typography.Text>
                  <div style={{ marginTop: 8 }}>
                    <Tag color={expiryColor(sub.days_left) ?? 'green'} bordered={false}>
                      {daysLeftText(sub.days_left)}
                    </Tag>
                    {sub.remaining_sessions !== null && (
                      <Tag bordered={false}>Còn {sub.remaining_sessions} buổi</Tag>
                    )}
                  </div>
                </>
              ) : (
                <Empty
                  image={Empty.PRESENTED_IMAGE_SIMPLE}
                  description="Chưa có gói đang hiệu lực"
                />
              )}
            </Card>
          </Col>
          <Col span={24}>
            <Card size="small" title="Check-in">
              <Row gutter={16}>
                <Col span={10}>
                  <Statistic title="Tổng lượt vào" value={m.stats?.total_checkins ?? 0} />
                </Col>
                <Col span={14}>
                  <Statistic
                    title="Lần gần nhất"
                    value={m.stats?.last_checkin_at ? formatDateTime(m.stats.last_checkin_at) : '—'}
                    valueStyle={{ fontSize: 16 }}
                  />
                </Col>
              </Row>
            </Card>
          </Col>
          <Col span={24}>
            <Card size="small" title="Chỉ số gần nhất">
              {metric ? (
                <Row gutter={8}>
                  <Col span={8}>
                    <Statistic title="Cân nặng" value={metric.weight_kg} suffix="kg" />
                  </Col>
                  <Col span={8}>
                    <Statistic title="BMI" value={metric.bmi} precision={1} />
                  </Col>
                  <Col span={8}>
                    <Statistic
                      title="Mỡ"
                      value={metric.body_fat_pct ?? '—'}
                      suffix={metric.body_fat_pct !== null ? '%' : undefined}
                    />
                  </Col>
                  <Col span={24}>
                    <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                      Đo lúc {formatDateTime(metric.measured_at)}
                    </Typography.Text>
                  </Col>
                </Row>
              ) : (
                <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="Chưa có lần đo nào" />
              )}
            </Card>
          </Col>
        </Row>
      </Col>
    </Row>
  );
}
