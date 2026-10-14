import { useMemo, useState } from 'react';
import { Button, Card, Col, Progress, Row, Space, Tooltip, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { DeleteOutlined, EditOutlined, SafetyCertificateOutlined } from '@ant-design/icons';
import promotionApi from '@/api/promotion';
import type { Promotion, PromotionListParams } from '@/api/promotion';
import ActionBar from '@/components/ActionBar';
import ConfirmButton from '@/components/ConfirmButton';
import DataTable from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import { MoneyText } from '@/components/Money';
import PageTitle from '@/components/PageTitle';
import StatusTag from '@/components/StatusTag';
import { PERM } from '@/constants/permissions';
import { useIsMobile } from '@/hooks/useIsMobile';
import { usePermission } from '@/hooks/usePermission';
import { useTableQuery } from '@/hooks/useTableQuery';
import { formatDateTime } from '@/utils/format';
import PromotionCheckModal from './PromotionCheckModal';
import PromotionForm from './PromotionForm';
import PromotionsMobile from './PromotionsMobile';
import {
  APPLIES_TO_LABELS,
  APPLIES_TO_OPTIONS,
  VALIDITY_MAP,
  VALIDITY_OPTIONS,
  discountText,
} from './constants';
import { useDeletePromotion } from './queries';

function UsageCell({ r }: { r: Promotion }) {
  if (!r.usage_limit) {
    return (
      <span>
        {r.used_count} <Typography.Text type="secondary">/ không giới hạn</Typography.Text>
      </span>
    );
  }
  const pct = Math.min(100, Math.round((r.used_count / r.usage_limit) * 100));
  return (
    <div style={{ minWidth: 120 }}>
      <div style={{ fontSize: 12 }}>
        {r.used_count} / {r.usage_limit}
      </div>
      <Progress
        percent={pct}
        size="small"
        showInfo={false}
        status={pct >= 100 ? 'exception' : 'normal'}
        style={{ margin: 0 }}
      />
    </div>
  );
}

function PromotionsDesktop() {
  const { can } = usePermission();
  const canManage = can(PERM.PROMOTION_MANAGE);
  const remove = useDeletePromotion();
  const [modal, setModal] = useState<{ open: boolean; record: Promotion | null }>({
    open: false,
    record: null,
  });
  const [checkCode, setCheckCode] = useState<string | null>(null);
  const openForm = (record: Promotion | null = null) => setModal({ open: true, record });

  const table = useTableQuery<Promotion, PromotionListParams>({
    queryKey: [promotionApi.key, 'list'],
    queryFn: (params) => promotionApi.list(params),
    filterKeys: ['validity', 'applies_to'],
    defaults: { order_by: 'id', order_dir: 'DESC' },
  });

  const columns = useMemo<ColumnsType<Promotion>>(
    () => [
      {
        title: 'Mã',
        dataIndex: 'code',
        sorter: true,
        render: (c: string) => <span className="code-text">{c}</span>,
      },
      {
        title: 'Tên chương trình',
        dataIndex: 'name',
        sorter: true,
        render: (name: string, r) => (
          <Button
            type="link"
            size="small"
            style={{ padding: 0, maxWidth: 240 }}
            onClick={() => openForm(r)}
          >
            <Typography.Text ellipsis={{ tooltip: name }} style={{ color: 'inherit' }}>
              {name}
            </Typography.Text>
          </Button>
        ),
      },
      {
        title: 'Hiệu lực',
        dataIndex: 'validity',
        render: (v: string) => <StatusTag status={v} map={VALIDITY_MAP} />,
      },
      {
        title: 'Ưu đãi',
        dataIndex: 'value',
        sorter: true,
        render: (_, r) => (
          <div>
            <div>{discountText(r)}</div>
            {r.min_amount > 0 && (
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                Đơn từ <MoneyText value={r.min_amount} />
              </Typography.Text>
            )}
          </div>
        ),
      },
      {
        title: 'Áp dụng cho',
        dataIndex: 'applies_to',
        render: (v: string) => APPLIES_TO_LABELS[v] ?? v,
      },
      {
        title: 'Thời gian',
        dataIndex: 'start_at',
        sorter: true,
        className: 'text-nowrap',
        render: (_, r) => (
          <div style={{ fontSize: 13 }}>
            <div>{formatDateTime(r.start_at)}</div>
            <Typography.Text type="secondary">→ {formatDateTime(r.end_at)}</Typography.Text>
          </div>
        ),
      },
      {
        title: 'Đã dùng',
        dataIndex: 'used_count',
        sorter: true,
        render: (_, r) => <UsageCell r={r} />,
      },
      {
        title: '',
        key: 'actions',
        fixed: 'right',
        width: 110,
        render: (_, r) => (
          <Space size={0}>
            <Tooltip title="Kiểm tra mã">
              <Button
                type="text"
                size="small"
                icon={<SafetyCertificateOutlined />}
                onClick={() => setCheckCode(r.code)}
                aria-label="Kiểm tra mã"
              />
            </Tooltip>
            {canManage && (
              <Tooltip title="Sửa">
                <Button
                  type="text"
                  size="small"
                  icon={<EditOutlined />}
                  onClick={() => openForm(r)}
                />
              </Tooltip>
            )}
            <ConfirmButton
              perm={PERM.PROMOTION_MANAGE}
              type="text"
              size="small"
              danger
              icon={<DeleteOutlined />}
              confirmTitle={`Xóa mã ${r.code}?`}
              confirmContent="Mã đã được sử dụng không thể xóa – hãy chuyển sang Ngừng áp dụng."
              onConfirm={() => remove.mutateAsync(r)}
              aria-label="Xóa"
            />
          </Space>
        ),
      },
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [canManage],
  );

  return (
    <Card variant="borderless" className="page-card">
      <Row justify="space-between" align="middle" gutter={[8, 12]} className="page-head">
        <Col>
          <PageTitle title="Khuyến mãi" subtitle="Mã giảm giá cho gói tập, gói PT và bán hàng" />
        </Col>
        <Col>
          <ActionBar
            onAdd={() => openForm()}
            addPerm={PERM.PROMOTION_MANAGE}
            addText="Thêm khuyến mãi"
            showDelete={false}
            onRefresh={() => table.refetch()}
            refreshing={table.isFetching}
          >
            <Button icon={<SafetyCertificateOutlined />} onClick={() => setCheckCode('')}>
              Kiểm tra mã
            </Button>
          </ActionBar>
        </Col>
      </Row>

      <FilterBar
        searchPlaceholder="Tìm theo mã, tên..."
        filters={[
          { name: 'validity', placeholder: 'Hiệu lực', options: VALIDITY_OPTIONS, col: 5 },
          { name: 'applies_to', placeholder: 'Áp dụng cho', options: APPLIES_TO_OPTIONS },
        ]}
      />

      <DataTable<Promotion> table={table} columns={columns} emptyText="Chưa có khuyến mãi nào" />

      <PromotionForm
        open={modal.open}
        record={modal.record}
        onClose={() => setModal((m) => ({ ...m, open: false }))}
      />
      <PromotionCheckModal
        open={checkCode !== null}
        initialCode={checkCode || undefined}
        onClose={() => setCheckCode(null)}
      />
    </Card>
  );
}

export default function PromotionsPage() {
  return useIsMobile() ? <PromotionsMobile /> : <PromotionsDesktop />;
}
