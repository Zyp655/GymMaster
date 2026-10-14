package module

import (
	"context"
	"fmt"
	"strings"

	"github.com/gogf/gf/v2/net/ghttp"

	"gymmaster-api/internal/consts"
	inventoryctrl "gymmaster-api/internal/controller/inventory"
	posctrl "gymmaster-api/internal/controller/pos"
	productctrl "gymmaster-api/internal/controller/product"
	"gymmaster-api/internal/cron"
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

	stock := NewStockService(d)
	cron.Register(cron.Job{
		Name:        "low_stock_digest",
		Spec:        "0 7 * * *",
		Description: "Tổng hợp sản phẩm dưới mức tồn tối thiểu, gửi thông báo cho người quản lý kho từng chi nhánh",
		Run: func(ctx context.Context) (int, error) {
			items, err := stock.ListLowStock(ctx, nil)
			if err != nil {
				return 0, err
			}
			byBranch := map[uint][]string{}
			for _, it := range items {
				byBranch[it.BranchID] = append(byBranch[it.BranchID], it.ProductName)
			}
			for branchID, names := range byBranch {
				id := branchID
				body := fmt.Sprintf("%d sản phẩm dưới mức tồn tối thiểu: %s", len(names), strings.Join(names, ", "))
				if err := d.Notifier.NotifyPermission(ctx, consts.PermInventoryView, &id, service.NotifyLowStock,
					"Cảnh báo tồn kho thấp", body, "/app/inventory?low_stock=true"); err != nil {
					return len(items), err
				}
			}
			return len(items), nil
		},
	})
}
