import dayjs from 'dayjs';

const currencyFormatter = new Intl.NumberFormat('vi-VN', {
  style: 'currency',
  currency: 'VND',
  maximumFractionDigits: 0,
});

const numberFormatter = new Intl.NumberFormat('vi-VN', {
  maximumFractionDigits: 1,
});

export function formatCurrency(value: number | null | undefined): string {
  return currencyFormatter.format(value ?? 0);
}

export function formatNumber(value: number | null | undefined): string {
  return numberFormatter.format(value ?? 0);
}

export function formatDate(
  value: string | Date | null | undefined,
  pattern = 'DD/MM/YYYY',
): string {
  if (!value) return '';
  return dayjs(value).format(pattern);
}

export function formatDateTime(value: string | Date | null | undefined): string {
  return formatDate(value, 'DD/MM/YYYY HH:mm');
}

export function initials(name: string | null | undefined): string {
  if (!name) return '?';
  const parts = name.trim().split(/\s+/);
  const first = parts[0]?.[0] ?? '';
  const last = parts.length > 1 ? (parts[parts.length - 1]?.[0] ?? '') : '';
  return (first + last).toUpperCase();
}
