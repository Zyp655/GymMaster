package router

import (
	"github.com/gogf/gf/v2/net/ghttp"

	notifctrl "gymmaster-api/internal/controller/notification"
	"gymmaster-api/internal/module"
)

// Register khai báo các nhóm route theo đối tượng sử dụng:
//
//	/v1/auth/*    xác thực (một phần public)
//	/v1/admin/*   nhân viên trung tâm – quyền kiểm tra theo tag `perm` của từng API
//	/v1/member/*  cổng hội viên – API khai báo role:"MEMBER" trong g.Meta
//	/v1/public/*  dữ liệu công khai cho landing page (không cần đăng nhập)
//
// Mỗi module thêm đúng MỘT dòng Register vào nhóm tương ứng bên dưới.
func Register(s *ghttp.Server, d *module.Deps) {
	s.Group("/v1", func(v1 *ghttp.RouterGroup) {
		streamHandler := notifctrl.StreamHandler(d.TokenMaker)
		v1.GET("/notifications/stream", streamHandler)
		v1.GET("/admin/notifications/stream", streamHandler)
		v1.GET("/member/notifications/stream", streamHandler)

		v1.Group("/", func(authed *ghttp.RouterGroup) {
			authed.Middleware(d.MW.Authentication())
			module.RegisterAuth(d, v1, authed)
		})

		admin := v1.Group("/admin")
		admin.Middleware(d.MW.Authentication(), d.MW.Authorization())

		member := v1.Group("/member")
		member.Middleware(d.MW.Authentication(), d.MW.Authorization())

		public := v1.Group("/public")

		module.RegisterBranch(d, admin)
		module.RegisterRole(d, admin)
		module.RegisterCatalog(d, admin)
		module.RegisterUpload(d, admin)
		module.RegisterMember(d, admin)
		module.RegisterStaff(d, admin)
		module.RegisterRoom(d, admin)
		module.RegisterShift(d, admin)
		module.RegisterCheckin(d, admin)
		module.RegisterClass(d, admin)
		module.RegisterSales(d, admin)
		module.RegisterPOS(d, admin)
		module.RegisterPT(d, admin)
		module.RegisterReport(d, admin)
		module.RegisterSystem(d, admin)

		module.RegisterMemberNotification(d, member)
		module.RegisterPortal(d, member)

		module.RegisterCatalogPublic(d, public)

		// Giai đoạn 2 (Sprint 13–16): Thanh toán VNPay, Giáo án tập luyện, Đánh giá, Thiết bị & CRM Leads
		module.RegisterPayment(d, admin, member, public)
		module.RegisterWorkout(d, admin, member)
		module.RegisterReview(d, admin, member)
		module.RegisterEquipment(d, admin)
		module.RegisterLead(d, admin, public)

		// Giai đoạn 3 (Sprint 18–23): Thiết bị Kiosk, Tủ đồ, Lương, Tài chính P&L, Điểm thưởng & Dinh dưỡng
		module.RegisterDevice(d, admin, public, v1)
		module.RegisterLocker(d, admin, member)
		module.RegisterPayroll(d, admin)
		module.RegisterFinance(d, admin, member)
		module.RegisterLoyalty(d, admin, member)
		module.RegisterEngagement(d, admin, member)

		// Giai đoạn 4 (Sprint 24–27): Hợp đồng điện tử, Lịch rảnh/Nghỉ lễ, KPI/Cohort, 2FA & Nghị định 13/2023
		module.RegisterContractPass(d, admin, member)
		module.RegisterScheduleExt(d, admin, public)
		module.RegisterKPI(d, admin, member, public)
		module.RegisterPrivacy(d, admin, member)

		// Giai đoạn 5 (Sprint 29–30 & Mục 15.1–15.2): CMS Landing/Blog, Tin nhắn PT, Lịch .ics, Import dữ liệu cũ & Setup Wizard
		module.RegisterCMS(d, admin, member, public)
		module.RegisterSystemExt(d, admin)
	})
}
