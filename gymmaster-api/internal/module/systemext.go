package module

import (
	"github.com/gogf/gf/v2/net/ghttp"

	"gymmaster-api/internal/controller/systemext"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/service"
)

func RegisterSystemExt(d *Deps, admin *ghttp.RouterGroup) {
	repo := repository.NewSystemExtRepository(d.DB)
	svc := service.NewSystemExtService(repo, d.Notifier, d.Audit)
	// Hoàn VNPay đi qua dịch vụ hóa đơn dùng chung để chạy RefundHandler (gói tập PENDING…).
	svc.SetInvoiceService(d.Invoices)
	ctrl := systemext.New(svc)
	admin.Bind(ctrl)
}
