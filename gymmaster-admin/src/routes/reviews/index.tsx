import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Card, Col, Rate, Row, Space, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { StarFilled } from '@ant-design/icons';
import { Link } from 'react-router-dom';
import reviewsApi, {
  RATING_OPTIONS,
  REVIEW_TARGET_COLORS,
  REVIEW_TARGET_LABELS,
  REVIEW_TARGET_OPTIONS,
} from '@/api/reviews';
import type { ReviewItem, ReviewListParams, ReviewTargetType } from '@/api/reviews';
import staffApi from '@/api/staff';
import ActionBar from '@/components/ActionBar';
import DataTable from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import PageTitle from '@/components/PageTitle';
import StatCard from '@/components/StatCard';
import { useBranchOptions } from '@/hooks/useBranchOptions';
import { useTableQuery } from '@/hooks/useTableQuery';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { formatDateTime, formatNumber } from '@/utils/format';

/** Trang Đánh giá HLV & lớp học (`/app/reviews`). */
export default function ReviewsPage() {
  const { branchId } = useWorkingBranch();
  const { branches } = useBranchOptions();

  const trainersQuery = useQuery({
    queryKey: [staffApi.key, 'trainers', 'reviews'],
    queryFn: () => staffApi.trainerOptions(branchId),
    staleTime: 5 * 60_000,
  });
  const trainerOptions = (trainersQuery.data ?? []).map((t) => ({
    value: t.id,
    label: t.full_name,
  }));

  const table = useTableQuery<ReviewItem, ReviewListParams>({
    queryKey: [reviewsApi.key, 'list'],
    queryFn: (p) => reviewsApi.list(p),
    filterKeys: ['branch_id', 'trainer_id', 'target_type', 'rating'],
    numberKeys: ['branch_id', 'trainer_id', 'rating'],
    defaults: { order_by: 'id', order_dir: 'DESC' },
    extraParams: branchId ? { branch_id: branchId } : {},
  });

  const avgRating = useMemo(() => {
    if (!table.items.length) return 0;
    const sum = table.items.reduce((acc, r) => acc + (r.rating || 0), 0);
    return Number((sum / table.items.length).toFixed(2));
  }, [table.items]);

  const columns = useMemo<ColumnsType<ReviewItem>>(
    () => [
      {
        title: 'Thời gian',
        dataIndex: 'created_at',
        className: 'text-nowrap',
        width: 155,
        render: (v: string) => formatDateTime(v),
      },
      {
        title: 'Hội viên',
        key: 'member',
        render: (_, r) => (
          <Space direction="vertical" size={0}>
            <Link to={`/app/members/${r.member_id}`}>
              {r.member_name || `Hội viên #${r.member_id}`}
            </Link>
            {r.member_code && (
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                {r.member_code}
              </Typography.Text>
            )}
          </Space>
        ),
      },
      {
        title: 'Huấn luyện viên',
        dataIndex: 'trainer_name',
        render: (v: string | undefined, r) => <b>{v || `HLV #${r.trainer_id}`}</b>,
      },
      {
        title: 'Loại buổi tập',
        dataIndex: 'target_type',
        width: 150,
        render: (t: ReviewTargetType, r) => (
          <Space direction="vertical" size={2}>
            <Tag color={REVIEW_TARGET_COLORS[t] ?? 'default'}>
              {REVIEW_TARGET_LABELS[t] ?? t}
            </Tag>
            {r.target_title && (
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                {r.target_title}
              </Typography.Text>
            )}
          </Space>
        ),
      },
      {
        title: 'Số sao',
        dataIndex: 'rating',
        width: 185,
        render: (rating: number) => (
          <Space size={6}>
            <Rate disabled value={rating} style={{ fontSize: 15 }} />
            <Typography.Text strong>{rating}/5</Typography.Text>
          </Space>
        ),
      },
      {
        title: 'Nhận xét của hội viên',
        dataIndex: 'comment',
        render: (c: string) =>
          c ? (
            <Typography.Text>{c}</Typography.Text>
          ) : (
            <Typography.Text type="secondary">Không có nhận xét</Typography.Text>
          ),
      },
      {
        title: 'Chi nhánh',
        dataIndex: 'branch_name',
        render: (v: string | undefined) => v || '—',
      },
    ],
    [],
  );

  return (
    <Card variant="borderless" className="page-card">
      <Row justify="space-between" align="middle" gutter={[8, 12]} className="page-head">
        <Col>
          <PageTitle
            title="Đánh giá HLV & lớp học"
            subtitle="Phản hồi thực tế của hội viên sau khi hoàn thành buổi tập PT hoặc tham gia lớp nhóm"
          />
        </Col>
        <Col>
          <ActionBar onRefresh={() => table.refetch()} refreshing={table.isFetching} />
        </Col>
      </Row>

      <Row gutter={[12, 12]} style={{ marginBottom: 16 }}>
        <Col xs={12} md={6}>
          <StatCard
            label="Tổng lượt đánh giá"
            value={table.total}
            caption="Theo bộ lọc hiện tại"
            icon={<StarFilled />}
            loading={table.isLoading}
          />
        </Col>
        <Col xs={12} md={6}>
          <StatCard
            label="Điểm trung bình trang"
            value={`${avgRating} ★`}
            caption={`${formatNumber(table.items.length)} đánh giá hiển thị`}
            icon={<StarFilled />}
            loading={table.isLoading}
          />
        </Col>
      </Row>

      <FilterBar
        searchPlaceholder="Tìm theo nhận xét, tên hội viên, tên HLV..."
        colTextSearch={5}
        filters={[
          ...(branchId
            ? []
            : [
                {
                  name: 'branch_id',
                  placeholder: 'Chi nhánh',
                  options: branches.map((b) => ({ label: b.name, value: b.id })),
                  col: 4,
                },
              ]),
          {
            name: 'trainer_id',
            placeholder: 'Huấn luyện viên',
            options: trainerOptions,
            col: 5,
          },
          {
            name: 'target_type',
            placeholder: 'Loại buổi tập',
            options: REVIEW_TARGET_OPTIONS,
            col: 4,
          },
          {
            name: 'rating',
            placeholder: 'Số sao',
            options: RATING_OPTIONS,
            col: 4,
          },
        ]}
      />

      <DataTable<ReviewItem>
        table={table}
        columns={columns}
        emptyText="Chưa có đánh giá nào từ hội viên"
      />
    </Card>
  );
}
