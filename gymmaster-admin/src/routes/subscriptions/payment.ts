import type { PaymentInput, PaymentMethod } from '@/api/invoice';

/** Giá trị form của khối thanh toán (đặt dưới tên `payment`). */
export interface PaymentFormValues {
  amount?: number | null;
  method?: PaymentMethod;
  reference?: string;
  note?: string;
}

export const DEFAULT_PAYMENT: PaymentFormValues = { method: 'CASH', reference: '', note: '' };

/** Số tiền 0/trống = chưa thu (không gửi `payment`). */
export function toPaymentInput(v?: PaymentFormValues): PaymentInput | undefined {
  if (!v?.amount || v.amount <= 0) return undefined;
  return {
    amount: v.amount,
    method: v.method ?? 'CASH',
    reference: v.reference?.trim() || '',
    note: v.note?.trim() || '',
  };
}
