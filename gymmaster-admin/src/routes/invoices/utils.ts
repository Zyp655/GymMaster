import type { Invoice } from '@/api/invoice';

export type InvoiceLike = Pick<
  Invoice,
  'id' | 'invoice_no' | 'total' | 'paid_amount' | 'refunded_amount' | 'due' | 'status'
>;

/** Quyết định hiển thị nút theo trạng thái hóa đơn. */
export const canCollect = (i: InvoiceLike) =>
  i.status !== 'VOID' && i.status !== 'REFUNDED' && i.due > 0;
export const canVoid = (i: InvoiceLike) => i.status === 'UNPAID' && i.paid_amount === 0;
export const canRefund = (i: InvoiceLike) =>
  (i.status === 'PAID' || i.status === 'PARTIAL') && i.paid_amount - i.refunded_amount > 0;
