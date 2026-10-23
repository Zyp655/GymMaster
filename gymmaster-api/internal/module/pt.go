package module

import (
	"context"
	"sync"
	"time"

	"github.com/gogf/gf/v2/net/ghttp"

	ptctrl "gymmaster-api/internal/controller/pt"
	"gymmaster-api/internal/cron"
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
	// Ràng buộc lịch dùng chung: nghỉ lễ chi nhánh, lịch rảnh & nghỉ phép HLV (PT, lớp nhóm, cổng hội viên).
	service.SetScheduleGuard(service.NewScheduleGuard(d.DB))
	return s
}

// RegisterPT: huấn luyện cá nhân – hợp đồng PT, buổi PT, lịch dạy HLV, hoa hồng (/v1/admin).
func RegisterPT(d *Deps, admin *ghttp.RouterGroup) {
	pt := PTServiceOf(d)
	admin.Bind(ptctrl.NewV1(pt))

	cron.Register(cron.Job{
		Name:        "expire_pt_contracts",
		Spec:        "15 0 * * *",
		Description: "Chuyển hợp đồng PT quá hạn sang Hết hạn, hủy các buổi đã đặt sau hạn",
		Run:         func(ctx context.Context) (int, error) { return pt.ExpireContracts(ctx, time.Now()) },
	})
	cron.Register(cron.Job{
		Name:        "mark_pt_no_shows",
		Spec:        "*/30 * * * *",
		Description: "Đánh dấu buổi PT quá giờ mà chưa xác nhận để quản lý xử lý",
		Run:         func(ctx context.Context) (int, error) { return pt.MarkPTNoShows(ctx, time.Now()) },
	})
}
