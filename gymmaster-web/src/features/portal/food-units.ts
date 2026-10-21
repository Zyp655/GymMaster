/**
 * Quy đổi khẩu phần nhật ký ăn uống. Server lưu `qty` = SỐ KHẨU PHẦN theo `food.unit`
 * (calories = food.calories × qty, cột numeric(10,4)).
 * - Món có đơn vị theo gam ("100g", "50g") → người dùng LUÔN nhập số gam; quy đổi gam / N.
 * - Món theo khẩu phần ("1 bát", "1 quả") → nhập số khẩu phần.
 */

/** Tối đa 100 khẩu phần mỗi bản ghi (khớp giới hạn API). */
export const MAX_SERVINGS = 100;

export type FoodQtyMode = 'gram' | 'serving';

/** Số gam của 1 khẩu phần nếu đơn vị dạng "100g" / "50 g" / "12,5g"; ngược lại null. */
export function gramsPerServing(unit: string | null | undefined): number | null {
  const match = /^\s*(\d+(?:[.,]\d+)?)\s*(g|gr|gam|gram)\s*$/i.exec(unit ?? '');
  if (!match) return null;
  const grams = Number(match[1].replace(',', '.'));
  return Number.isFinite(grams) && grams > 0 ? grams : null;
}

export function qtyMode(unit: string | null | undefined): FoodQtyMode {
  return gramsPerServing(unit) ? 'gram' : 'serving';
}

function round(value: number, digits: number): number {
  const f = 10 ** digits;
  return Math.round((value + Number.EPSILON) * f) / f;
}

/** Số nhập trên form (gam hoặc khẩu phần) → số khẩu phần gửi API (4 chữ số thập phân). */
export function toServings(unit: string | null | undefined, input: number): number {
  const grams = gramsPerServing(unit);
  return round(grams ? input / grams : input, 4);
}

/** Số khẩu phần từ API → giá trị hiển thị/nhập lại trên form (gam hoặc khẩu phần, 2 chữ số). */
export function fromServings(unit: string | null | undefined, servings: number): number {
  const grams = gramsPerServing(unit);
  return round(grams ? servings * grams : servings, 2);
}

/** Giá trị nhập tối đa theo đơn vị của món (100 khẩu phần). */
export function maxInput(unit: string | null | undefined): number {
  const grams = gramsPerServing(unit);
  return grams ? MAX_SERVINGS * grams : MAX_SERVINGS;
}

/** Giá trị mặc định khi chọn món: 1 khẩu phần (VD: 100 g cho món "100g"). */
export function defaultInput(unit: string | null | undefined): number {
  return gramsPerServing(unit) ?? 1;
}
