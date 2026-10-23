import type { Locker, LockerBoardSummary, LockerBoardZone } from '@/api/operations-ext';

/** Một chi nhánh trên sơ đồ tủ: thống kê riêng + các khu vực (tủ cùng mã ở chi nhánh khác không bị trộn). */
export interface LockerBoardBranch {
  branchId: number;
  branchName: string;
  summary: LockerBoardSummary;
  zones: LockerBoardZone[];
}

/** Đếm tủ theo trạng thái. */
export function summarizeLockers(lockers: Locker[]): LockerBoardSummary {
  const s: LockerBoardSummary = {
    total: lockers.length,
    available: 0,
    occupied: 0,
    rented: 0,
    broken: 0,
  };
  for (const l of lockers) {
    if (l.status === 'AVAILABLE') s.available++;
    else if (l.status === 'OCCUPIED') s.occupied++;
    else if (l.status === 'RENTED') s.rented++;
    else if (l.status === 'BROKEN') s.broken++;
  }
  return s;
}

/**
 * Nhóm tủ theo chi nhánh → khu vực (giữ thứ tự xuất hiện từ API). Dùng khi xem "Tất cả chi nhánh":
 * mỗi chi nhánh đều có "A-01", "Khu Nam – Tầng 2"... nên nhóm chỉ theo khu sẽ trộn tủ của nhiều nơi.
 */
export function groupLockersByBranch(
  lockers: Locker[],
  branchNameOf?: (id: number) => string | undefined,
): LockerBoardBranch[] {
  const branches = new Map<number, { name: string; zones: Map<string, Locker[]>; all: Locker[] }>();
  for (const lk of lockers) {
    let b = branches.get(lk.branch_id);
    if (!b) {
      const name = lk.branch_name || branchNameOf?.(lk.branch_id) || `Chi nhánh #${lk.branch_id}`;
      b = { name, zones: new Map(), all: [] };
      branches.set(lk.branch_id, b);
    }
    b.all.push(lk);
    const zone = lk.zone || 'Khu A';
    const list = b.zones.get(zone);
    if (list) list.push(lk);
    else b.zones.set(zone, [lk]);
  }
  return Array.from(branches.entries()).map(([branchId, b]) => ({
    branchId,
    branchName: b.name,
    summary: summarizeLockers(b.all),
    zones: Array.from(b.zones.entries()).map(([zone, list]) => ({ zone, lockers: list })),
  }));
}
