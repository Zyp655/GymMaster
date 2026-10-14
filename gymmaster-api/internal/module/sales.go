package module

import (
	"github.com/gogf/gf/v2/net/ghttp"

	invoicectrl "gymmaster-api/internal/controller/invoice"
	subscriptionctrl "gymmaster-api/internal/controller/subscription"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/service"
)

// RegisterSales: bán gói tập – đăng ký – hóa đơn – thanh toán – hoàn tiền (/v1/admin).
// InvoiceService/SubscriptionService dùng chung lấy từ d (tạo trong cmd); SubscriptionService tự đăng ký
// PaidHandler/VoidHandler cho dòng MEMBERSHIP và FEE khi khởi tạo.
func RegisterSales(d *Deps, admin *ghttp.RouterGroup) {
	subs := d.Subscriptions
	if subs == nil {
		subs = service.NewSubscriptionService(repository.NewSubscriptionRepository(d.DB), d.Invoices, d.Tx, d.Audit, d.Notifier)
		d.Subscriptions = subs
	}
	admin.Bind(
		subscriptionctrl.NewV1(subs),
		invoicectrl.NewV1(d.Invoices),
	)
}
