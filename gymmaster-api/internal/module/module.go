// Package module chứa phần "lắp ráp" (composition root) cho từng module nghiệp vụ.
//
// Mỗi module tự khởi tạo repository → service → controller và đăng ký route của mình,
// thay vì dồn toàn bộ vào một file router khổng lồ. Thêm module mới = thêm một file ở đây
// và một dòng trong router.Register.
package module

import (
	"gorm.io/gorm"

	"gymmaster-api/internal/config"
	"gymmaster-api/internal/middleware"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/service"
	"gymmaster-api/internal/storage/postgres"
)

// Deps là các phụ thuộc hạ tầng và service dùng chung, được tạo một lần trong cmd.
type Deps struct {
	Config     *config.Config
	DB         *gorm.DB
	Tx         postgres.TxManager
	TokenMaker token.Maker
	MW         middleware.Middleware

	// Service dùng chung giữa các module.
	Settings service.SettingService
	Audit    service.AuditService
	Notifier service.Notifier

	// Bán hàng (module sales): Promotions áp mã khuyến mãi; Invoices là dịch vụ hóa đơn DÙNG CHUNG –
	// module PT/POS lập hóa đơn qua đây và đăng ký Invoices.RegisterPaidHandler/RegisterVoidHandler
	// cho item_type của mình. Subscriptions dùng cho cron (ExpireDue, ApplyDueFreezes, ListExpiring)
	// và cổng hội viên.
	Promotions    service.PromotionService
	Invoices      service.InvoiceService
	Subscriptions service.SubscriptionService
}
