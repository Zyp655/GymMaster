import { Card, Col, Empty, Row, Skeleton, Space, Tag, Tooltip, Typography } from 'antd';
import { CheckCircleFilled } from '@ant-design/icons';
import type { MembershipPlan } from '@/api/plan';
import { ACCESS_SCOPE_LABEL } from '@/api/subscription';
import MoneyText from '@/components/Money/MoneyText';

export interface PlanCardsProps {
  plans: MembershipPlan[];
  loading?: boolean;
  value?: number;
  onChange: (plan: MembershipPlan) => void;
  /** Gói không cho chọn (vd. gói hiện tại khi nâng cấp). */
  disabledIds?: number[];
  /** Gói không cho chọn kèm lý do (hiện tooltip + nhãn), vd. gói rẻ hơn khi nâng cấp. */
  disabledReasons?: Record<number, { tag: string; tooltip: string }>;
}

/** Lưới thẻ các gói đang bán (giá, thời hạn, số buổi, gồm lớp) – bấm để chọn. */
export default function PlanCards({
  plans,
  loading,
  value,
  onChange,
  disabledIds,
  disabledReasons,
}: PlanCardsProps) {
  if (loading) return <Skeleton active paragraph={{ rows: 3 }} />;
  if (plans.length === 0)
    return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="Chưa có gói tập đang bán" />;

  return (
    <Row gutter={[12, 12]}>
      {plans.map((p) => {
        const selected = p.id === value;
        const isCurrent = disabledIds?.includes(p.id);
        const reason = isCurrent ? undefined : disabledReasons?.[p.id];
        const disabled = isCurrent || !!reason;
        return (
          <Col xs={24} sm={12} md={8} key={p.id}>
            <Tooltip title={reason?.tooltip}>
              <Card
                size="small"
                hoverable={!disabled}
                onClick={() => !disabled && onChange(p)}
                style={{
                  height: '100%',
                  cursor: disabled ? 'not-allowed' : 'pointer',
                  opacity: disabled ? 0.5 : 1,
                  borderColor: selected ? '#F97316' : undefined,
                  boxShadow: selected ? '0 0 0 2px rgba(249,115,22,0.25)' : undefined,
                }}
              >
                <Space direction="vertical" size={2} style={{ width: '100%' }}>
                  <Space style={{ justifyContent: 'space-between', width: '100%' }} align="start">
                    <Typography.Text strong>{p.name}</Typography.Text>
                    {selected && <CheckCircleFilled style={{ color: '#F97316', fontSize: 18 }} />}
                  </Space>
                  <Typography.Text type="secondary" className="code-text" style={{ fontSize: 12 }}>
                    {p.code}
                  </Typography.Text>
                  <Typography.Text strong style={{ fontSize: 18, color: '#0A3150' }}>
                    <MoneyText value={p.price} />
                  </Typography.Text>
                  <Typography.Text style={{ fontSize: 13 }}>
                    {p.duration_days} ngày
                    {p.type === 'SESSION' && p.session_count ? ` · ${p.session_count} buổi` : ''}
                  </Typography.Text>
                  <Space size={[4, 4]} wrap>
                    <Tag bordered={false}>{ACCESS_SCOPE_LABEL[p.access_scope]}</Tag>
                    {p.includes_classes && (
                      <Tag color="green" bordered={false}>
                        Gồm lớp nhóm
                      </Tag>
                    )}
                    {p.max_freeze_times > 0 && (
                      <Tag color="blue" bordered={false}>
                        Bảo lưu {p.max_freeze_days} ngày
                      </Tag>
                    )}
                    {isCurrent && <Tag bordered={false}>Gói hiện tại</Tag>}
                    {reason && (
                      <Tag color="default" bordered={false}>
                        {reason.tag}
                      </Tag>
                    )}
                  </Space>
                </Space>
              </Card>
            </Tooltip>
          </Col>
        );
      })}
    </Row>
  );
}
