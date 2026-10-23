package module

import (
	"context"

	"github.com/gogf/gf/v2/net/ghttp"

	loyaltyctrl "gymmaster-api/internal/controller/loyalty"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/service"
)

// RegisterLoyalty: tích điểm thưởng, đổi quà, huy hiệu, thử thách & giới thiệu bạn bè (/v1/admin & /v1/member).
func RegisterLoyalty(d *Deps, admin, member *ghttp.RouterGroup) {
	svc := service.NewLoyaltyService(repository.NewLoyaltyRepository(d.DB), d.Tx, d.Audit)
	// Tích điểm tự động: check-in & đánh giá gọi qua hook dùng chung; thanh toán gói tập/PT qua hook hóa đơn.
	service.SetLoyaltyEarner(svc)
	if d.Invoices != nil {
		referrerOf := func(ctx context.Context, memberID uint) (*uint, error) {
			var m entity.Member
			err := d.DB.WithContext(ctx).Select("referred_by").Where("id = ?", memberID).Take(&m).Error
			return m.ReferredBy, err
		}
		h := service.NewLoyaltyInvoicePaidHandler(svc, referrerOf)
		d.Invoices.RegisterPaidHandler(entity.InvoiceItemMembership, h)
		d.Invoices.RegisterPaidHandler(entity.InvoiceItemPT, h)
	}
	if admin != nil {
		admin.Bind(loyaltyctrl.NewAdminV1(svc))
	}
	if member != nil {
		member.Bind(loyaltyctrl.NewMemberV1(svc))
	}
}
