import { Grid } from 'antd';

/** `true` khi màn hình nhỏ hơn breakpoint `md` (768px) → trang list chuyển sang MobileList. */
export function useIsMobile(): boolean {
  const screens = Grid.useBreakpoint();
  // Lần render đầu screens rỗng → coi là desktop để tránh nháy.
  if (Object.keys(screens).length === 0) return false;
  return !screens.md;
}

export default useIsMobile;
