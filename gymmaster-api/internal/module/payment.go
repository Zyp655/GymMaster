package module

import (
	"context"
	"sync"
	"time"

	"github.com/gogf/gf/v2/net/ghttp"

	paymentctrl "gymmaster-api/internal/controller/payment"
	"gymmaster-api/internal/cron"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/service"
)

var (
	paymentMu       sync.Mutex
	paymentServices = map[*Deps]service.OnlinePaymentService{}
	paymentJobOnce  sync.Once
)

// OnlinePaymentServiceOf trả OnlinePaymentService dùng chung cho bộ Deps và đăng ký cron expire_online_txns.
func OnlinePaymentServiceOf(d *Deps) service.OnlinePaymentService {
	paymentMu.Lock()
	defer paymentMu.Unlock()
	if s, ok := paymentServices[d]; ok {
		return s
	}
	subs := d.Subscriptions
	if subs == nil {
		subs = service.NewSubscriptionService(repository.NewSubscriptionRepository(d.DB), d.Invoices, d.Tx, d.Audit, d.Notifier)
		d.Subscriptions = subs
	}
	promos := d.Promotions
	if promos == nil {
		promos = service.NewPromotionService(repository.NewPromotionRepository(d.DB), d.Audit)
		d.Promotions = promos
	}
	s := service.NewOnlinePaymentService(
		repository.NewOnlinePaymentRepository(d.DB),
		d.Invoices,
		subs,
		promos,
		d.Tx,
		d.Notifier,
		d.Config.VNPay,
	)
	paymentServices[d] = s

	paymentJobOnce.Do(func() {
		cron.Register(cron.Job{
			Name:        service.JobExpireOnlineTxns,
			Spec:        "*/15 * * * *",
			Description: "Chuyển các giao dịch thanh toán trực tuyến VNPay chờ quá 15 phút sang Hết hạn",
			Run: func(ctx context.Context) (int, error) {
				return s.ExpirePendingTxns(ctx, time.Now())
			},
		})
	})
	return s
}

// RegisterPayment đăng ký toàn bộ route thanh toán trực tuyến cho admin, member và public.
func RegisterPayment(d *Deps, admin, member, public *ghttp.RouterGroup) {
	RegisterPaymentAdmin(d, admin)
	RegisterPaymentMember(d, member)
	RegisterPaymentPublic(d, public)
}

func RegisterPaymentAdmin(d *Deps, admin *ghttp.RouterGroup) {
	if admin == nil {
		return
	}
	admin.Bind(paymentctrl.NewAdminV1(OnlinePaymentServiceOf(d)))
}

func RegisterPaymentMember(d *Deps, member *ghttp.RouterGroup) {
	if member == nil {
		return
	}
	member.Bind(paymentctrl.NewMemberV1(OnlinePaymentServiceOf(d)))
}

func RegisterPaymentPublic(d *Deps, public *ghttp.RouterGroup) {
	if public == nil {
		return
	}
	public.Bind(paymentctrl.NewPublicV1(OnlinePaymentServiceOf(d)))
}
