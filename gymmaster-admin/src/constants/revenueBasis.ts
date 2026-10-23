/**
 * Cơ sở tính "doanh thu" của từng màn hình – các con số KHÁC NHAU có chủ đích, giải thích bằng tooltip (HelpLabel).
 * Giữ đồng bộ với backend: report_repository (Overview/RevenueFlows), finance_repository.CalcProfitLoss,
 * kpi_repository.ComputeKPIActuals, pos_repository (tổng kết ngày), report_repository.ProductSales.
 */
export const REVENUE_HELP = {
  /** Dashboard + Báo cáo doanh thu: dòng tiền thực thu. */
  cash:
    'Thực thu (dòng tiền): tổng phiếu thu theo NGÀY THU TIỀN trừ tiền hoàn theo NGÀY HOÀN, mọi loại hóa đơn. ' +
    'Hóa đơn chưa thu đủ chỉ tính phần đã thu.',
  /** "So với kỳ trước" trên Dashboard. */
  dashboardCompare:
    'So với cùng kỳ tháng trước: từ ngày 1 đến cùng ngày này của tháng trước (vd 01–21/09 so với 01–21/10).',
  /** "So với kỳ trước" trên Báo cáo. */
  reportCompare: 'So với kỳ liền trước có cùng số ngày (vd 01–31/10 so với 30/08–30/09).',
  /** Lãi/Lỗ (P&L). */
  pnl:
    'Doanh thu ghi nhận theo NGÀY LẬP HÓA ĐƠN (giá trị dòng hóa đơn đã thanh toán / thanh toán một phần / đã hoàn), ' +
    'trừ tiền hoàn theo ngày hoàn. Khác "Thực thu" ở Dashboard/Báo cáo vì tính theo ngày lập hóa đơn, kể cả phần chưa thu.',
  /** KPI doanh thu. */
  kpi:
    'Doanh thu KPI: số tiền ĐÃ THU (trừ hoàn) của các hóa đơn LẬP trong tháng KPI (không gồm hóa đơn hủy). ' +
    'Tiền thu tháng sau cho hóa đơn lập tháng này vẫn tính vào tháng này.',
  /** Báo cáo bán hàng (sản phẩm). */
  sales:
    'Theo hóa đơn POS đã thu (PAID/PARTIAL) LẬP trong kỳ; số lượng, doanh thu và giá vốn đã trừ toàn bộ hàng khách trả ' +
    'của chính hóa đơn đó (kể cả trả sau kỳ).',
  /** Tổng kết ngày POS. */
  posDaily:
    'Doanh thu thuần = tổng hóa đơn POS lập trong ngày − tiền hoàn trả hàng phát sinh trong ngày. ' +
    'SL bán = số lượng trên hóa đơn trong ngày − số lượng trả trong ngày (cùng cơ sở).',
  /** "Thu theo phương thức" của tổng kết ngày POS (pos_repository.PaymentsByMethod). */
  posByMethod:
    'Tiền GỘP đã thu theo NGÀY THU TIỀN cho đơn bán hàng POS (kể cả thu nợ đơn lập ngày trước), chia theo phương thức; ' +
    '"lần" = số phiếu thu. CHƯA trừ tiền hoàn trả hàng (xem "Hoàn trả" ở thẻ Doanh thu thuần) và không gồm phần khách ' +
    'còn nợ, nên có thể khác Doanh thu gộp/thuần.',
} as const;
