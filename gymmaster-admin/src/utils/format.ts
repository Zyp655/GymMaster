import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';

export const DATE_FORMAT = 'DD/MM/YYYY';
export const DATETIME_FORMAT = 'DD/MM/YYYY HH:mm';
export const TIME_FORMAT = 'HH:mm';
/** Định dạng ngày gửi lên API. */
export const API_DATE_FORMAT = 'YYYY-MM-DD';

type DateInput = string | number | Date | Dayjs | null | undefined;

const EMPTY = '—';

const moneyFormatter = new Intl.NumberFormat('vi-VN', { maximumFractionDigits: 0 });

/** 1500000 → "1.500.000" */
export function formatNumber(value: number | string | null | undefined): string {
  if (value === null || value === undefined || value === '') return EMPTY;
  const n = Number(value);
  return Number.isFinite(n) ? moneyFormatter.format(n) : EMPTY;
}

/** 1500000 → "1.500.000 ₫" */
export function formatMoney(value: number | string | null | undefined, suffix = ' ₫'): string {
  const text = formatNumber(value);
  return text === EMPTY ? text : `${text}${suffix}`;
}

/** → "14/10/2026" */
export function formatDate(value: DateInput, format = DATE_FORMAT): string {
  if (value === null || value === undefined || value === '') return EMPTY;
  const d = dayjs(value);
  return d.isValid() ? d.format(format) : EMPTY;
}

/** → "14/10/2026 08:30" */
export function formatDateTime(value: DateInput): string {
  return formatDate(value, DATETIME_FORMAT);
}

/** Dayjs → "YYYY-MM-DD" cho API (null nếu rỗng). */
export function toApiDate(value: DateInput): string | null {
  if (!value) return null;
  const d = dayjs(value);
  return d.isValid() ? d.format(API_DATE_FORMAT) : null;
}

/** "0901234567" → "0901 234 567"; "02838221111" → "028 3822 1111". */
export function formatPhone(value: string | null | undefined): string {
  if (!value) return EMPTY;
  const digits = value.replace(/\D/g, '');
  if (digits.length === 10) return `${digits.slice(0, 4)} ${digits.slice(4, 7)} ${digits.slice(7)}`;
  if (digits.length === 11) return `${digits.slice(0, 3)} ${digits.slice(3, 7)} ${digits.slice(7)}`;
  return value;
}

/** Chuẩn hóa tiếng Việt để tìm kiếm không dấu. */
export function normalizeText(value: string): string {
  return value
    .normalize('NFD')
    .replace(/[̀-ͯ]/g, '')
    .replace(/đ/g, 'd')
    .replace(/Đ/g, 'D')
    .toLowerCase();
}
