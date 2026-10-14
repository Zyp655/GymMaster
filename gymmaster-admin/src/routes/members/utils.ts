/** Hàm tiện ích của module hội viên. */

/** Chuẩn hóa SĐT VN giống `service.NormalizeVNPhone` (bỏ khoảng trắng . -, +84/84 → 0). */
export function normalizeVNPhone(raw: string): string | null {
  let s = raw.replace(/[\s.\-()]/g, '');
  if (s.startsWith('+84')) s = `0${s.slice(3)}`;
  else if (s.startsWith('84') && s.length >= 11) s = `0${s.slice(2)}`;
  return /^0[35789]\d{8}$/.test(s) || /^02\d{9}$/.test(s) ? s : null;
}

/** Màu cảnh báo theo số ngày còn lại: ≤ 3 ngày đỏ, ≤ 7 ngày cam. */
export function expiryColor(daysLeft: number): string | undefined {
  if (daysLeft <= 3) return 'red';
  if (daysLeft <= 7) return 'orange';
  return undefined;
}

export function daysLeftText(daysLeft: number): string {
  if (daysLeft < 0) return 'Đã hết hạn';
  if (daysLeft === 0) return 'Hết hạn hôm nay';
  return `Còn ${daysLeft} ngày`;
}
