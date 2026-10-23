/**
 * Ký tự vô hình (zero-width, BOM, soft hyphen, dấu điều hướng bidi…): trim() không loại bỏ
 * nên chuỗi chỉ gồm các ký tự này trông như rỗng nhưng vẫn qua được kiểm tra `.trim()`.
 */
const INVISIBLE_CHARS = /[­͏؜ᅟᅠ឴឵᠎​-‏‪-‮⁠-⁯ㅤ﻿ﾠ]/g;

/** Bỏ ký tự vô hình + khoảng trắng hai đầu – chỉ dùng để kiểm tra có nội dung hiển thị hay không. */
export function stripInvisible(value: string): string {
  return value.replace(INVISIBLE_CHARS, '').trim();
}

/** true khi chuỗi có ít nhất một ký tự nhìn thấy được. */
export function hasVisibleText(value: string | null | undefined): boolean {
  return !!value && stripInvisible(value).length > 0;
}
