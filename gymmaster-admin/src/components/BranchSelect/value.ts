/**
 * Ánh xạ giá trị giữa BranchSelect (bên ngoài) và antd Select (bên trong) – hàm thuần để test được.
 *
 * antd cảnh báo khi option có value = null nên lựa chọn "Tất cả chi nhánh" dùng 0 bên trong
 * (id chi nhánh luôn ≥ 1). CHỈ ánh xạ null ↔ 0 khi ô thật sự có lựa chọn "Tất cả" (allowAll && user toàn hệ thống);
 * ngược lại null/undefined → undefined để Select hiện placeholder (trước đây null luôn thành 0 → ô hiện chữ "0").
 */
export const ALL_BRANCHES = 0;

export type BranchSelectValue = number | number[] | null | undefined;

export function toSelectValue(
  value: BranchSelectValue,
  showAll: boolean,
): number | number[] | undefined {
  if (value === null || value === undefined)
    return showAll && value === null ? ALL_BRANCHES : undefined;
  return value;
}

export function fromSelectValue(
  value: BranchSelectValue,
  showAll: boolean,
): number | number[] | null | undefined {
  if (showAll && value === ALL_BRANCHES) return null;
  // Bỏ chọn (allowClear) → undefined; 0 không phải id chi nhánh hợp lệ → undefined.
  if (value === null || value === ALL_BRANCHES) return undefined;
  return value;
}
