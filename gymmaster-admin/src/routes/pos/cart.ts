import { useCallback, useEffect, useRef, useState } from 'react';
import type { MemberOption } from '@/api/member';
import type { PaymentMethod } from '@/api/invoice';
import type { PosProduct } from '@/api/pos';
import { openInvoicePrint } from '@/routes/subscriptions/utils';

/** Một dòng giỏ: giữ ảnh chụp sản phẩm để hiển thị được cả khi danh sách chưa tải xong. */
export interface CartLine {
  product: PosProduct;
  quantity: number;
}

export interface PaymentLine {
  key: string;
  method: PaymentMethod;
  amount: number | null;
  reference: string;
}

export interface CartState {
  lines: CartLine[];
  member: MemberOption | null;
  /** Mã khuyến mãi đã bấm "Áp dụng" (đã kiểm tra hợp lệ ở lần cuối). */
  promoCode: string;
  payments: PaymentLine[];
  note: string;
}

let seq = 0;
export const newPaymentLine = (method: PaymentMethod = 'CASH', amount: number | null = null) => ({
  key: `${Date.now()}-${++seq}`,
  method,
  amount,
  reference: '',
});

export const emptyCart = (): CartState => ({
  lines: [],
  member: null,
  promoCode: '',
  payments: [newPaymentLine()],
  note: '',
});

const storageKey = (branchId: number) => `gm.pos.cart.${branchId}`;

function readCart(branchId: number): CartState {
  try {
    const raw = localStorage.getItem(storageKey(branchId));
    if (!raw) return emptyCart();
    const parsed = JSON.parse(raw) as Partial<CartState>;
    return {
      ...emptyCart(),
      ...parsed,
      lines: Array.isArray(parsed.lines) ? parsed.lines.filter((l) => l?.product?.id) : [],
      payments:
        Array.isArray(parsed.payments) && parsed.payments.length > 0
          ? parsed.payments
          : [newPaymentLine()],
    };
  } catch {
    return emptyCart();
  }
}

/** Đơn giá áp dụng: giá hội viên khi có hội viên đang hoạt động và sản phẩm có giá hội viên. */
export function unitPrice(p: Pick<PosProduct, 'sell_price' | 'member_price'>, member: MemberOption | null) {
  if (member && member.status === 'ACTIVE' && p.member_price !== null && p.member_price !== undefined)
    return p.member_price;
  return p.sell_price;
}

/** Giỏ hàng của chi nhánh, lưu localStorage để F5 không mất. */
export function useCart(branchId: number) {
  const [cart, setCart] = useState<CartState>(() => readCart(branchId));
  const cartRef = useRef(cart);
  cartRef.current = cart;

  useEffect(() => {
    try {
      localStorage.setItem(storageKey(branchId), JSON.stringify(cart));
    } catch {
      /* bỏ qua (chế độ riêng tư) */
    }
  }, [branchId, cart]);

  const patch = useCallback((p: Partial<CartState>) => setCart((c) => ({ ...c, ...p })), []);

  /** Thêm `qty` sản phẩm; trả về false nếu vượt tồn. */
  const add = useCallback((product: PosProduct, qty = 1): boolean => {
    const current = cartRef.current.lines.find((l) => l.product.id === product.id)?.quantity ?? 0;
    if (current + qty > product.stock) return false;
    setCart((c) => {
      const idx = c.lines.findIndex((l) => l.product.id === product.id);
      const lines =
        idx >= 0
          ? c.lines.map((l, i) => (i === idx ? { product, quantity: l.quantity + qty } : l))
          : [...c.lines, { product, quantity: qty }];
      return { ...c, lines };
    });
    return true;
  }, []);

  const setQty = useCallback((productId: number, qty: number) => {
    setCart((c) => ({
      ...c,
      lines: c.lines
        .map((l) =>
          l.product.id === productId
            ? { ...l, quantity: Math.max(0, Math.min(qty, l.product.stock)) }
            : l,
        )
        .filter((l) => l.quantity > 0),
    }));
  }, []);

  const remove = useCallback((productId: number) => {
    setCart((c) => ({ ...c, lines: c.lines.filter((l) => l.product.id !== productId) }));
  }, []);

  /** Cập nhật ảnh chụp sản phẩm (giá, tồn) theo danh sách mới nhất. */
  const sync = useCallback((products: PosProduct[]) => {
    setCart((c) => {
      let changed = false;
      const lines = c.lines.map((l) => {
        const p = products.find((x) => x.id === l.product.id);
        if (!p) return l;
        if (JSON.stringify(p) === JSON.stringify(l.product)) return l;
        changed = true;
        return { ...l, product: p };
      });
      return changed ? { ...c, lines } : c;
    });
  }, []);

  const reset = useCallback(() => setCart(emptyCart()), []);

  return { cart, setCart, patch, add, setQty, remove, sync, reset };
}

export interface PaymentTotals {
  received: number;
  nonCash: number;
  cash: number;
  /** Còn thiếu (> 0 thì chưa thanh toán được). */
  missing: number;
  change: number;
  /** Lỗi chặn thanh toán (chuyển khoản/thẻ vượt tổng). */
  error: string | null;
}

export function paymentTotals(payments: PaymentLine[], total: number): PaymentTotals {
  const amt = (p: PaymentLine) => Math.max(0, p.amount ?? 0);
  const received = payments.reduce((s, p) => s + amt(p), 0);
  const cash = payments.filter((p) => p.method === 'CASH').reduce((s, p) => s + amt(p), 0);
  const nonCash = received - cash;
  const change = Math.max(0, received - total);
  return {
    received,
    cash,
    nonCash,
    missing: Math.max(0, total - received),
    change,
    error:
      nonCash > total
        ? 'Chỉ tiền mặt mới được đưa thừa: số tiền chuyển khoản/thẻ vượt quá tổng hóa đơn'
        : null,
  };
}

const PRINT_SIZE_KEY = 'gm.invoicePrintSize';

/** In hóa đơn khổ 80mm (máy in nhiệt ở quầy) qua trang in có sẵn `/print/invoices/:id`. */
export function printReceipt80(invoiceId: number) {
  try {
    localStorage.setItem(PRINT_SIZE_KEY, 'r80');
  } catch {
    /* bỏ qua */
  }
  openInvoicePrint(invoiceId);
}
