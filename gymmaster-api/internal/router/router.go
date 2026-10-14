package router

import (
	"github.com/gogf/gf/v2/net/ghttp"

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
		v1.Group("/", func(authed *ghttp.RouterGroup) {
			authed.Middleware(d.MW.Authentication())
			module.RegisterAuth(d, v1, authed)
		})

		v1.Group("/admin", func(admin *ghttp.RouterGroup) {
			admin.Middleware(d.MW.Authentication(), d.MW.Authorization())
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
			// [admin-modules]
		})

		v1.Group("/member", func(member *ghttp.RouterGroup) {
			member.Middleware(d.MW.Authentication(), d.MW.Authorization())
			module.RegisterMemberNotification(d, member)
			// [member-modules]
		})

		v1.Group("/public", func(public *ghttp.RouterGroup) {
			module.RegisterCatalogPublic(d, public)
			// [public-modules]
		})
	})
}
