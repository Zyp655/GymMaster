import { useMemo, useState } from 'react';
import { Bar, Column, Line, Pie } from '@ant-design/charts';
import { useQuery } from '@tanstack/react-query';
import { Col, Row, Segmented, Table } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { DollarOutlined, RiseOutlined, RollbackOutlined } from '@ant-design/icons';
import reportApi from '@/api/report';
import type { RevenuePoint } from '@/api/report';
import { MoneyText } from '@/components/Money';
import StatCard from '@/components/StatCard';
import { formatDate, formatMoney } from '@/utils/format';
import { ChartCard, DeltaBadge, ExportButton, SwatchLegend } from './shared';
import { CATEGORICAL, NAVY, rangeParams, REVENUE_TYPES, shortMoney } from './types';
import type { ReportFilters } from './types';

const moneyTooltip = { valueFormatter: (v: number) => formatMoney(v) };

export default function RevenueTab(f: ReportFilters) {
  const [mode, setMode] = useState<'column' | 'line'>('column');
  const query = useQuery({
    queryKey: [reportApi.key, 'revenue', f],
    queryFn: () => reportApi.revenue({ ...rangeParams(f), group_by: f.groupBy }),
  });
  const data = query.data;

  // Kỳ trước cùng độ dài, cùng group_by → so sánh từng điểm.
  const prevQuery = useQuery({
    queryKey: [reportApi.key, 'revenue', 'previous', f.branchId, data?.previous.from, data?.previous.to, f.groupBy],
    enabled: !!data,
    queryFn: () =>
      reportApi.revenue({
        branch_id: f.branchId,
        from: data!.previous.from,
        to: data!.previous.to,
        group_by: f.groupBy,
      }),
  });

  const series = useMemo(() => data?.series ?? [], [data]);
  const hasRevenue = !!data && (data.totals.payments !== 0 || data.totals.refunds !== 0);

  const stacked = useMemo(
    () =>
      series.flatMap((p) =>
        REVENUE_TYPES.map((t) => ({ label: p.label, type: t.label, value: p[t.field] })),
      ),
    [series],
  );

  const compare = useMemo(() => {
    const prev = prevQuery.data?.series ?? [];
    return series.flatMap((p, i) => [
      { label: p.label, kind: 'Kỳ này', value: p.total },
      ...(prev[i] ? [{ label: p.label, kind: 'Kỳ trước', value: prev[i].total }] : []),
    ]);
  }, [series, prevQuery.data]);

  const byType = (data?.by_type ?? []).filter((x) => x.value !== 0);
  const byMethod = (data?.by_method ?? []).filter((x) => x.value !== 0);
  const typeColor = (key: string) => REVENUE_TYPES.find((t) => t.key === key)?.color ?? '#94a3b8';

  const columns: ColumnsType<RevenuePoint> = [
    { title: 'Kỳ', dataIndex: 'label', fixed: 'left', width: 110 },
    ...REVENUE_TYPES.map((t) => ({
      title: t.label,
      dataIndex: t.field,
      align: 'right' as const,
      render: (v: number) => <MoneyText value={v} />,
    })),
    { title: 'Tổng thu', dataIndex: 'payments', align: 'right', render: (v: number) => <MoneyText value={v} /> },
    { title: 'Hoàn trả', dataIndex: 'refunds', align: 'right', render: (v: number) => <MoneyText value={v} /> },
    { title: 'Thực thu', dataIndex: 'total', align: 'right', render: (v: number) => <MoneyText value={v} strong /> },
  ];

  const common = { loading: query.isLoading, error: query.error, onRetry: () => void query.refetch() };

  return (
    <div>
      <div className="rp-tab-head">
        <span className="rp-tab-head__note">
          Thực thu = tiền thu − tiền hoàn, phân bổ theo loại dòng hóa đơn.
        </span>
        <ExportButton type="revenue" branchId={f.branchId} from={f.from} to={f.to} />
      </div>

      <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
        <Col xs={24} md={8}>
          <StatCard
            label="Thực thu trong kỳ"
            icon={<RiseOutlined />}
            value={data?.totals.net}
            currency
            loading={query.isLoading}
            caption={
              data ? (
                <>
                  <DeltaBadge pct={data.previous.change_pct} />
                  <br />
                  Kỳ trước ({formatDate(data.previous.from)} – {formatDate(data.previous.to)}):{' '}
                  {formatMoney(data.previous.net)}
                </>
              ) : undefined
            }
          />
        </Col>
        <Col xs={12} md={8}>
          <StatCard label="Tổng tiền thu" icon={<DollarOutlined />} value={data?.totals.payments} currency loading={query.isLoading} caption="Theo ngày thu tiền" />
        </Col>
        <Col xs={12} md={8}>
          <StatCard label="Tổng tiền hoàn" icon={<RollbackOutlined />} value={data?.totals.refunds} currency loading={query.isLoading} caption="Theo ngày hoàn" />
        </Col>
      </Row>

      <Row gutter={[16, 16]}>
        <Col xs={24}>
          <ChartCard
            title="Doanh thu theo thời gian (tách loại)"
            {...common}
            empty={!hasRevenue}
            emptyText="Chưa phát sinh doanh thu trong kỳ"
            extra={
              <Segmented
                size="small"
                value={mode}
                onChange={(v) => setMode(v as 'column' | 'line')}
                options={[
                  { label: 'Cột chồng', value: 'column' },
                  { label: 'Đường', value: 'line' },
                ]}
              />
            }
          >
            {mode === 'column' ? (
              <Column
                height={300}
                data={stacked}
                xField="label"
                yField="value"
                colorField="type"
                stack
                scale={{ color: { domain: REVENUE_TYPES.map((t) => t.label), range: REVENUE_TYPES.map((t) => t.color) } }}
                style={{ radiusTopLeft: 3, radiusTopRight: 3, maxWidth: 28 }}
                axis={{ x: { title: false, labelAutoHide: true }, y: { title: false, labelFormatter: shortMoney } }}
                legend={{ color: { position: 'top', layout: { justifyContent: 'center' } } }}
                tooltip={moneyTooltip}
              />
            ) : (
              <Line
                height={300}
                data={stacked}
                xField="label"
                yField="value"
                colorField="type"
                scale={{ color: { domain: REVENUE_TYPES.map((t) => t.label), range: REVENUE_TYPES.map((t) => t.color) } }}
                style={{ lineWidth: 2 }}
                axis={{ x: { title: false, labelAutoHide: true }, y: { title: false, labelFormatter: shortMoney } }}
                legend={{ color: { position: 'top', layout: { justifyContent: 'center' } } }}
                tooltip={moneyTooltip}
              />
            )}
          </ChartCard>
        </Col>

        <Col xs={24} lg={14}>
          <ChartCard
            title="So sánh với kỳ trước"
            {...common}
            loading={query.isLoading || prevQuery.isLoading}
            empty={!hasRevenue && !(prevQuery.data?.totals.net)}
            extra={data ? <DeltaBadge pct={data.previous.change_pct} suffix="" /> : null}
          >
            <Line
              height={260}
              data={compare}
              xField="label"
              yField="value"
              colorField="kind"
              scale={{ color: { domain: ['Kỳ này', 'Kỳ trước'], range: [NAVY, '#B6BCC6'] } }}
              style={{ lineWidth: 2 }}
              axis={{ x: { title: false, labelAutoHide: true }, y: { title: false, labelFormatter: shortMoney } }}
              legend={{ color: { position: 'top', layout: { justifyContent: 'center' } } }}
              tooltip={moneyTooltip}
            />
          </ChartCard>
        </Col>
        <Col xs={24} lg={10}>
          <ChartCard title="Theo phương thức thanh toán" {...common} empty={byMethod.length === 0}>
            <Pie
              height={260}
              data={byMethod}
              angleField="value"
              colorField="label"
              innerRadius={0.6}
              scale={{
                color: {
                  domain: (data?.by_method ?? []).map((x) => x.label),
                  range: [...CATEGORICAL],
                },
              }}
              label={{ text: (d: { pct: number }) => `${d.pct}%`, position: 'outside', style: { fontSize: 12 } }}
              legend={{ color: { position: 'bottom', layout: { justifyContent: 'center' } } }}
              tooltip={{ title: 'label', items: [{ channel: 'y', name: 'Số tiền', valueFormatter: (v: number) => formatMoney(v) }] }}
            />
          </ChartCard>
        </Col>

        <Col xs={24} lg={10}>
          <ChartCard title="Cơ cấu theo loại" {...common} empty={byType.length === 0}>
            <Row gutter={16} align="middle">
              <Col xs={24} sm={12}>
                <Pie
                  height={220}
                  data={byType}
                  angleField="value"
                  colorField="label"
                  innerRadius={0.62}
                  scale={{ color: { domain: byType.map((x) => x.label), range: byType.map((x) => typeColor(x.key)) } }}
                  legend={false}
                  tooltip={{ title: 'label', items: [{ channel: 'y', name: 'Số tiền', valueFormatter: (v: number) => formatMoney(v) }] }}
                />
              </Col>
              <Col xs={24} sm={12}>
                <SwatchLegend
                  items={byType.map((x) => ({ label: x.label, color: typeColor(x.key), value: `${x.pct}%` }))}
                />
              </Col>
            </Row>
          </ChartCard>
        </Col>
        <Col xs={24} lg={14}>
          <ChartCard title="Theo chi nhánh" {...common} empty={!data?.by_branch.length}>
            <Bar
              height={Math.max(160, (data?.by_branch.length ?? 1) * 46)}
              data={data?.by_branch ?? []}
              xField="branch_name"
              yField="amount"
              style={{ fill: NAVY, maxWidth: 26, radiusTopRight: 3, radiusBottomRight: 3 }}
              axis={{ x: { title: false }, y: { title: false, labelFormatter: shortMoney } }}
              label={{ text: (d: { amount: number }) => shortMoney(d.amount), position: 'right', dx: 4, style: { fill: '#374151' } }}
              tooltip={{ items: [{ channel: 'y', name: 'Thực thu', valueFormatter: (v: number) => formatMoney(v) }] }}
            />
          </ChartCard>
        </Col>

        <Col xs={24}>
          <ChartCard title="Bảng số liệu" {...common} empty={series.length === 0} bodyStyle={{ padding: 0 }}>
            <Table<RevenuePoint>
              size="small"
              rowKey="period"
              dataSource={series}
              columns={columns}
              scroll={{ x: 'max-content' }}
              pagination={series.length > 31 ? { pageSize: 31, size: 'small' } : false}
              summary={() =>
                data ? (
                  <Table.Summary.Row style={{ fontWeight: 600, background: '#FAFBFC' }}>
                    <Table.Summary.Cell index={0}>Tổng</Table.Summary.Cell>
                    {REVENUE_TYPES.map((t, i) => (
                      <Table.Summary.Cell key={t.key} index={i + 1} align="right">
                        <MoneyText value={series.reduce((s, p) => s + p[t.field], 0)} />
                      </Table.Summary.Cell>
                    ))}
                    <Table.Summary.Cell index={6} align="right"><MoneyText value={data.totals.payments} /></Table.Summary.Cell>
                    <Table.Summary.Cell index={7} align="right"><MoneyText value={data.totals.refunds} /></Table.Summary.Cell>
                    <Table.Summary.Cell index={8} align="right"><MoneyText value={data.totals.net} strong /></Table.Summary.Cell>
                  </Table.Summary.Row>
                ) : null
              }
            />
          </ChartCard>
        </Col>
      </Row>
    </div>
  );
}
