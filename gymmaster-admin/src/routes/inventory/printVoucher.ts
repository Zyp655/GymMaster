import type { Voucher } from '@/api/inventory';
import { STOCK_TX_TYPE_MAP, VOUCHER_KIND_MAP } from '@/constants/status';
import { formatDateTime, formatMoney, formatNumber } from '@/utils/format';

const esc = (v: unknown) =>
  String(v ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');

/** In phiếu kho đơn giản (A4/A5) – mở cửa sổ mới chỉ chứa phiếu rồi gọi hộp thoại in. */
export function printVoucher(v: Voucher) {
  const isTransfer = v.kind === 'TRANSFER';
  const lines = isTransfer ? v.lines.filter((l) => l.type === 'TRANSFER_OUT') : v.lines;
  const showCost = v.kind === 'IMPORT' || v.kind === 'ADJUST';
  const rows = lines
    .map(
      (l, i) => `<tr>
        <td>${i + 1}</td>
        <td>${esc(l.product_sku)}</td>
        <td>${esc(l.product_name)}${v.kind === 'ADJUST' ? `<div class="m">${esc(l.note)}</div>` : ''}</td>
        <td>${esc(l.unit)}</td>
        <td class="n">${formatNumber(isTransfer ? Math.abs(l.quantity) : l.quantity)}</td>
        ${showCost ? `<td class="n">${formatNumber(l.unit_cost)}</td><td class="n">${formatNumber(Math.round(Math.abs(l.quantity) * l.unit_cost))}</td>` : ''}
      </tr>`,
    )
    .join('');
  const kind = VOUCHER_KIND_MAP[v.kind]?.label ?? v.kind;
  const html = `<!doctype html><html lang="vi"><head><meta charset="utf-8"><title>${esc(kind)} ${esc(v.voucher_no)}</title>
<style>
  @page { size: A5 landscape; margin: 10mm; }
  body { font-family: Arial, sans-serif; font-size: 12px; color: #111; margin: 0; padding: 12px; }
  h1 { text-align: center; font-size: 18px; margin: 6px 0 2px; text-transform: uppercase; }
  .c { text-align: center; } .m { color: #555; font-size: 11px; }
  .meta { display: grid; grid-template-columns: 110px 1fr 110px 1fr; gap: 3px 8px; margin: 12px 0; }
  table { width: 100%; border-collapse: collapse; }
  th, td { border: 1px solid #999; padding: 4px 6px; vertical-align: top; }
  th { background: #f2f2f2; } .n { text-align: right; white-space: nowrap; }
  .sign { display: grid; grid-template-columns: 1fr 1fr 1fr; text-align: center; margin-top: 24px; }
</style></head><body>
  <div class="c"><b>GYMMASTER</b> · ${esc(v.branch_name)}</div>
  <h1>${esc(kind)}</h1>
  <div class="c m">Số: <b>${esc(v.voucher_no)}</b> · ${esc(formatDateTime(v.created_at))}</div>
  <div class="meta">
    <span>${isTransfer ? 'Chi nhánh xuất:' : 'Chi nhánh:'}</span><b>${esc(v.branch_name)}</b>
    ${isTransfer ? `<span>Chi nhánh nhận:</span><b>${esc(v.to_branch_name)}</b>` : v.supplier_name ? `<span>Nhà cung cấp:</span><b>${esc(v.supplier_name)}</b>` : '<span></span><span></span>'}
    <span>Người lập:</span><span>${esc(v.created_by_name)}</span>
    <span>Ghi chú:</span><span>${esc(v.note) || '—'}</span>
  </div>
  <table>
    <thead><tr><th>#</th><th>SKU</th><th>Sản phẩm</th><th>ĐVT</th><th>${v.kind === 'ADJUST' ? 'Chênh lệch' : 'Số lượng'}</th>
    ${showCost ? '<th>Đơn giá</th><th>Thành tiền</th>' : ''}</tr></thead>
    <tbody>${rows}</tbody>
  </table>
  <p style="text-align:right">Tổng số lượng: <b>${formatNumber(v.total_quantity)}</b> · Tổng giá trị: <b>${esc(formatMoney(v.total_value))}</b></p>
  <div class="sign"><div><b>Người lập phiếu</b><div class="m">(Ký, ghi rõ họ tên)</div></div>
  <div><b>${isTransfer ? 'Người nhận' : 'Thủ kho'}</b><div class="m">(Ký, ghi rõ họ tên)</div></div>
  <div><b>Quản lý</b><div class="m">(Ký, ghi rõ họ tên)</div></div></div>
  <script>window.onload = function () { setTimeout(function () { window.print(); }, 200); };</script>
</body></html>`;
  const w = window.open('', '_blank', 'width=900,height=700');
  if (!w) return false;
  w.document.open();
  w.document.write(html);
  w.document.close();
  return true;
}

export const txTypeLabel = (t: string) => STOCK_TX_TYPE_MAP[t]?.label ?? t;
