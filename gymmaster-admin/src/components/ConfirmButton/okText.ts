import type { ReactNode } from 'react';

/**
 * Chữ nút đồng ý của hộp xác nhận: okText truyền vào → chữ trên nút (nếu là chuỗi) → aria-label (nút chỉ có icon)
 * → "Xóa" (danger) / "Đồng ý". Trước đây mọi nút danger không truyền okText đều hiện "Xóa"
 * (vd "Từ chối" khoản chi, "Giải phóng" tủ đồ).
 */
export function resolveOkText(
  okText: string | undefined,
  children: ReactNode,
  danger?: boolean,
  ariaLabel?: string,
): string {
  if (okText) return okText;
  if (typeof children === 'string' && children.trim()) return children.trim();
  if (ariaLabel?.trim()) return ariaLabel.trim();
  return danger ? 'Xóa' : 'Đồng ý';
}
