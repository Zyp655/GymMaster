package module

import (
	"github.com/gogf/gf/v2/net/ghttp"

	inventoryctrl "gymmaster-api/internal/controller/inventory"
	posctrl "gymmaster-api/internal/controller/pos"
	productctrl "gymmaster-api/internal/controller/product"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/service"
)

// NewStockService dựng StockService dùng chung (cron cảnh báo tồn thấp gọi ListLowStock).
func NewStockService(d *Deps) service.StockService {
	return service.NewStockService(repository.NewStockRepository(d.DB), repository.NewProductRepository(d.DB),
		d.Tx, d.Audit, d.Notifier)
}

// RegisterPOS: sản phẩm – nhà cung cấp – kho (nhập/điều chuyển/kiểm kê/thẻ kho) – bán hàng tại quầy (/v1/admin).
// PosService đăng ký VoidHandler "PRODUCT" với d.Invoices (hủy hóa đơn POS → hoàn kho).
func RegisterPOS(d *Deps, admin *ghttp.RouterGroup) {
	products := repository.NewProductRepository(d.DB)
	stockRepo := repository.NewStockRepository(d.DB)
	posSvc := service.NewPosService(repository.NewPosRepository(d.DB), products, stockRepo, d.Invoices, d.Tx, d.Audit, d.Notifier)
	admin.Bind(
		productctrl.NewV1(service.NewProductService(products, d.Audit)),
		inventoryctrl.NewV1(service.NewStockService(stockRepo, products, d.Tx, d.Audit, d.Notifier)),
		posctrl.NewV1(posSvc),
	)
}
