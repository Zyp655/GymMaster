/** Chuẩn hóa mã giới thiệu từ query `?ref=`: chỉ chữ/số/-/_ , tối đa 32 ký tự; không hợp lệ → null. */
export function normalizeReferralCode(raw: string | string[] | null | undefined): string | null {
  const value = (Array.isArray(raw) ? raw[0] : raw)?.trim();
  if (!value || !/^[A-Za-z0-9_-]{1,32}$/.test(value)) return null;
  return value;
}
