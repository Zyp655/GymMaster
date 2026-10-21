/** Hàm thuần cho trang Cấu hình (có unit test trong scripts/unit.test.ts). */

export interface SettingDef {
  key: string;
  label: string;
  help: string;
  unit: string;
  min: number;
  max: number;
  /** Mảng số nguyên (VD ngày nhắc gia hạn). */
  list?: { maxItems: number };
  /** Cờ bật/tắt lưu dạng số 0/1 → hiển thị Switch. */
  boolean?: boolean;
}

/** Gợi ý khoảng hợp lệ dưới ô nhập (rỗng với cờ bật/tắt hoặc khóa không rõ giới hạn). */
export function rangeHint(def: SettingDef): string {
  if (def.boolean) return '';
  if (def.max >= Number.MAX_SAFE_INTEGER) return def.min > 0 ? `Tối thiểu ${def.min}` : '';
  return `Khoảng hợp lệ: ${def.min} – ${def.max}${def.unit ? ` ${def.unit}` : ''}`;
}

/** Chuẩn hóa danh sách mốc nhập trong Select tags: số nguyên, bỏ trùng, giảm dần. */
export function normalizeTagValues(vals: string[]): number[] {
  return [...new Set(vals.map((s) => Number(s.trim())).filter((n) => !Number.isNaN(n)))].sort(
    (a, b) => b - a,
  );
}
