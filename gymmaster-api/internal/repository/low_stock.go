package repository

import "fmt"

// LowStockCond là định nghĩa DUY NHẤT của "tồn thấp" (dashboard, báo cáo, tồn kho, /inventory/low-stock,
// danh sách sản phẩm, POS, cron cảnh báo) – xem docs/api/pos-inventory.md mục "Tồn thấp":
//
//	một DÒNG TỒN (stock_levels đã tồn tại – sản phẩm đã từng nhập về chi nhánh đó) của sản phẩm ACTIVE chưa xóa,
//	tại chi nhánh ACTIVE chưa xóa, có min_stock > 0 và quantity < min_stock (tồn đúng bằng mức tối thiểu là ĐỦ).
//
// Sản phẩm chưa từng nhập về chi nhánh (không có dòng stock_levels) KHÔNG tính là tồn thấp ở chi nhánh đó.
// Tham số là alias SQL của products, stock_levels và branches trong truy vấn gọi.
func LowStockCond(p, sl, b string) string {
	return fmt.Sprintf(`(%[2]s.product_id IS NOT NULL AND %[1]s.deleted_at IS NULL AND %[1]s.status = 'ACTIVE'`+
		` AND %[3]s.deleted_at IS NULL AND %[3]s.status = 'ACTIVE'`+
		` AND %[1]s.min_stock > 0 AND %[2]s.quantity < %[1]s.min_stock)`, p, sl, b)
}

// lowStockSQL: điều kiện tồn thấp với alias chuẩn p (products), sl (stock_levels), b (branches).
var lowStockSQL = LowStockCond("p", "sl", "b")
