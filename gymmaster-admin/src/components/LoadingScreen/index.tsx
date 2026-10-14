import { Spin } from 'antd';

/** Màn hình chờ toàn trang (khôi phục phiên lúc khởi động). */
export function LoadingScreen({ tip = 'Đang tải...' }: { tip?: string }) {
  return (
    <div className="loading-screen">
      <img src="/logo.svg" alt="GymMaster" className="loading-screen__logo" />
      <Spin size="large" />
      <div className="loading-screen__tip">{tip}</div>
    </div>
  );
}

/** Fallback khi tải chunk của trang (lazy route). */
export function RouteFallback() {
  return (
    <div style={{ display: 'flex', justifyContent: 'center', padding: '80px 0' }}>
      <Spin size="large" />
    </div>
  );
}

export default LoadingScreen;
