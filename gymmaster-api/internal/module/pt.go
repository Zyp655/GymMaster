package module

import (
	"sync"

	"github.com/gogf/gf/v2/net/ghttp"

	ptctrl "gymmaster-api/internal/controller/pt"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/service"
)

var (
	ptMu       sync.Mutex
	ptServices = map[*Deps]service.PTService{}
)

// PTServiceOf trả PTService dùng chung cho một bộ Deps (tạo MỘT lần – đăng ký PaidHandler/VoidHandler "PT"
// vào d.Invoices). Module khác (cron, cổng hội viên) lấy service PT qua hàm này:
//
//	pt := module.PTServiceOf(d)
//	n, err := pt.ExpireContracts(ctx, time.Now())
func PTServiceOf(d *Deps) service.PTService {
	ptMu.Lock()
	defer ptMu.Unlock()
	if s, ok := ptServices[d]; ok {
		return s
	}
	s := service.NewPTService(repository.NewPTRepository(d.DB), d.Invoices, d.Tx, d.Settings, d.Audit, d.Notifier)
	ptServices[d] = s
	return s
}

// RegisterPT: huấn luyện cá nhân – hợp đồng PT, buổi PT, lịch dạy HLV, hoa hồng (/v1/admin).
func RegisterPT(d *Deps, admin *ghttp.RouterGroup) {
	admin.Bind(ptctrl.NewV1(PTServiceOf(d)))
}
