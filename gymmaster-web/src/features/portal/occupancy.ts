import type { BranchOccupancy } from './types';

/**
 * Chuẩn hóa dữ liệu mật độ: API trả `present_count`; các tên cũ (present/current_count/
 * active_count) chỉ giữ làm phương án dự phòng để UI luôn đọc `present_count`.
 */
export const normalizeOccupancy = (o: BranchOccupancy): BranchOccupancy => ({
  ...o,
  present_count: o.present_count ?? o.present ?? o.current_count ?? o.active_count ?? 0,
});

export const toOccupancyList = (
  res: BranchOccupancy[] | BranchOccupancy | { items?: BranchOccupancy[] } | null | undefined,
): BranchOccupancy[] => {
  if (!res) return [];
  if (Array.isArray(res)) return res.map(normalizeOccupancy);
  if ('items' in res && Array.isArray(res.items)) return res.items.map(normalizeOccupancy);
  if ('branch_id' in res || 'capacity' in res) return [normalizeOccupancy(res as BranchOccupancy)];
  return [];
};
