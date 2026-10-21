import { useQuery } from '@tanstack/react-query';
import { Button, Card, Col, Progress, Result, Row, Table, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  DollarOutlined,
  FundOutlined,
  PercentageOutlined,
  ShoppingCartOutlined,
  ShoppingOutlined,
} from '@ant-design/icons';
import reportApi from '@/api/report';
import type { SalesProductRow } from '@/api/report';
import MoneyText from '@/components/Money/MoneyText';
import HelpLabel from '@/components/HelpLabel';
import StatCard from '@/components/StatCard';
import { REVENUE_HELP } from '@/constants/revenueBasis';
import { getErrorMessage } from '@/utils/errors';
import { formatNumber } from '@/utils/format';
import { rangeParams } from './types';
import type { ReportFilters } from './types';

export default function SalesTab(f: ReportFilters) {
  const query = useQuery({
    queryKey: [reportApi.key, 'sales-products', f.branchId ?? null, f.from, f.to],
    queryFn: () => reportApi.salesProducts({ ...rangeParams(f), limit: 20 }),
  });

  if (query.isError) {
    return (
      <Result
        status="warning"
        title="Không tải được báo cáo bán hàng POS"
        subTitle={getErrorMessage(query.error)}
        extra={<Button onClick={() => void query.refetch()}>Thử lại</Button>}
      />
    );
  }

  const d = query.data;
  const t = d?.totals;

  const productColumns: ColumnsType<SalesProductRow> = [
    {
      title: 'Sản phẩm',
      dataIndex: 'name',
      width: 300,
      render: (v: string, r) => (
        <div style={{ maxWidth: 284 }}>
          <Typography.Text
            strong
            ellipsis={{ tooltip: v }}
            style={{ maxWidth: 284, display: 'block' }}
          >
            {v}
          </Typography.Text>
          <span className="code-text" style={{ fontSize: 12, color: 'var(--gm-text-secondary)' }}>
            {r.sku}
          </span>
        </div>
      ),
    },
    { title: 'Danh mục', dataIndex: 'category_name', width: 150, render: (v: string) => v || '—' },
    {
      title: 'SL bán (thuần)',
      dataIndex: 'quantity',
      align: 'right',
      render: (v: number) => <b>{formatNumber(v)}</b>,
    },
    {
      title: 'Doanh thu',
      dataIndex: 'revenue',
      align: 'right',
      width: 140,
      render: (v: number) => (
        <span style={{ whiteSpace: 'nowrap' }}>
          <MoneyText value={v} strong />
        </span>
      ),
    },
    {
      title: 'Giá vốn',
      dataIndex: 'cost',
      align: 'right',
      render: (v: number) => <MoneyText value={v} />,
    },
    {
      title: 'Lãi gộp',
      dataIndex: 'gross_profit',
      align: 'right',
      render: (v: number) => <MoneyText value={v} strong />,
    },
    {
      title: 'Biên lãi',
      dataIndex: 'margin',
      align: 'right',
      render: (v: number) => `${v}%`,
    },
  ];

  return (
    <div>
      <Row gutter={[12, 12]} style={{ marginBottom: 16 }}>
        <Col xs={12} sm={8} lg={5}>
          <StatCard
            label={<HelpLabel help={REVENUE_HELP.sales}>SL bán thuần (đã trừ trả)</HelpLabel>}
            value={t?.quantity ?? 0}
            caption={`${formatNumber(d?.product_count ?? 0)} mặt hàng phát sinh`}
            icon={<ShoppingOutlined />}
            loading={query.isLoading}
          />
        </Col>
        <Col xs={12} sm={8} lg={5}>
          <StatCard
            label={<HelpLabel help={REVENUE_HELP.sales}>Doanh thu sản phẩm (đã trừ trả)</HelpLabel>}
            value={t?.revenue ?? 0}
            currency
            icon={<ShoppingCartOutlined />}
            loading={query.isLoading}
          />
        </Col>
        <Col xs={12} sm={8} lg={5}>
          <StatCard
            label="Giá vốn hàng bán"
            value={t?.cost ?? 0}
            currency
            icon={<DollarOutlined />}
            loading={query.isLoading}
          />
        </Col>
        <Col xs={12} sm={12} lg={5}>
          <StatCard
            label="Lợi nhuận gộp"
            value={t?.gross_profit ?? 0}
            currency
            icon={<FundOutlined />}
            loading={query.isLoading}
          />
        </Col>
        <Col xs={24} sm={12} lg={4}>
          <StatCard
            label="Biên lợi nhuận gộp"
            value={t?.margin ?? 0}
            valueSuffix="%"
            progress={t?.margin ?? 0}
            icon={<PercentageOutlined />}
            loading={query.isLoading}
          />
        </Col>
      </Row>

      <Row gutter={[12, 12]}>
        <Col xs={24} xxl={15}>
          <Card size="small" title="Top sản phẩm bán chạy" loading={query.isLoading}>
            <Table<SalesProductRow>
              size="small"
              rowKey="product_id"
              pagination={false}
              dataSource={d?.top ?? []}
              columns={productColumns}
              scroll={{ x: 'max-content' }}
              locale={{ emptyText: 'Chưa có sản phẩm bán ra trong kỳ' }}
            />
          </Card>
        </Col>
        <Col xs={24} xxl={9}>
          <Card size="small" title="Theo danh mục sản phẩm" loading={query.isLoading}>
            <Table
              size="small"
              rowKey="category_name"
              pagination={false}
              dataSource={d?.by_category ?? []}
              scroll={{ x: 'max-content' }}
              locale={{ emptyText: 'Chưa có dữ liệu' }}
              columns={[
                { title: 'Danh mục', dataIndex: 'category_name' },
                {
                  title: 'SL',
                  dataIndex: 'quantity',
                  align: 'right',
                  render: (v: number) => formatNumber(v),
                },
                {
                  title: 'Doanh thu',
                  dataIndex: 'revenue',
                  align: 'right',
                  render: (v: number) => <MoneyText value={v} strong />,
                },
                {
                  title: 'Lãi gộp',
                  dataIndex: 'gross_profit',
                  align: 'right',
                  render: (v: number) => <MoneyText value={v} />,
                },
                {
                  title: 'Tỷ trọng',
                  dataIndex: 'pct',
                  width: 110,
                  render: (pct: number) => <Progress percent={pct} size="small" />,
                },
              ]}
            />
          </Card>
        </Col>
      </Row>
    </div>
  );
}
