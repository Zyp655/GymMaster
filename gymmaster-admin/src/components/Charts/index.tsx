import { lazy, Suspense } from 'react';
import { Skeleton } from 'antd';
import type { ColumnConfig, LineConfig, PieConfig } from '@ant-design/charts';

/**
 * Biểu đồ tải lười: thư viện @ant-design/charts (~1,4 MB) chỉ tải khi biểu đồ thực sự hiển thị,
 * phần còn lại của trang (KPI, bảng) render ngay. Dùng thay cho import trực tiếp từ '@ant-design/charts'.
 */
// Import động qua ./chartsImpl (re-export tĩnh) để Rollup vẫn tree-shake phần không dùng (graphs/G6...).
const ColumnImpl = lazy(() => import('./chartsImpl').then((m) => ({ default: m.Column })));
const LineImpl = lazy(() => import('./chartsImpl').then((m) => ({ default: m.Line })));
const PieImpl = lazy(() => import('./chartsImpl').then((m) => ({ default: m.Pie })));

function ChartFallback({ height }: { height?: number }) {
  return (
    <div
      role="status"
      aria-label="Đang tải biểu đồ"
      style={{ height: height ?? 240, display: 'flex', alignItems: 'center' }}
    >
      <Skeleton.Node active style={{ width: '100%', height: height ?? 240 }}>
        <span />
      </Skeleton.Node>
    </div>
  );
}

export function Column(props: ColumnConfig) {
  return (
    <Suspense fallback={<ChartFallback height={props.height} />}>
      <ColumnImpl {...props} />
    </Suspense>
  );
}

export function Line(props: LineConfig) {
  return (
    <Suspense fallback={<ChartFallback height={props.height} />}>
      <LineImpl {...props} />
    </Suspense>
  );
}

export function Pie(props: PieConfig) {
  return (
    <Suspense fallback={<ChartFallback height={props.height} />}>
      <PieImpl {...props} />
    </Suspense>
  );
}
