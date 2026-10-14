package module

import (
	"sync"

	"github.com/gogf/gf/v2/net/ghttp"

	financectrl "gymmaster-api/internal/controller/finance"
	"gymmaster-api/internal/pkg/einvoice"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/service"
)

var (
	financeMu       sync.Mutex
	financeServices = map[*Deps]service.FinanceService{}
)

func FinanceServiceOf(d *Deps) service.FinanceService {
	financeMu.Lock()
	defer financeMu.Unlock()
	if s, ok := financeServices[d]; ok {
		return s
	}
	s := service.NewFinanceService(
		repository.NewFinanceRepository(d.DB),
		d.Invoices,
		einvoice.NewMockProvider(),
		d.Tx,
		d.Audit,
	)
	financeServices[d] = s
	return s
}

// RegisterFinance đăng ký route chi phí, đối soát ca, báo cáo P&L, trả góp & hóa đơn điện tử (/v1/admin & /v1/member).
func RegisterFinance(d *Deps, admin, member *ghttp.RouterGroup) {
	svc := FinanceServiceOf(d)
	if admin != nil {
		admin.Bind(financectrl.NewAdminV1(svc))
	}
	if member != nil {
		member.Bind(financectrl.NewMemberV1(svc))
	}
}
