package module

import (
	"github.com/gogf/gf/v2/net/ghttp"

	landingctrl "gymmaster-api/internal/controller/landing"
	planctrl "gymmaster-api/internal/controller/plan"
	promotionctrl "gymmaster-api/internal/controller/promotion"
	ptpackagectrl "gymmaster-api/internal/controller/ptpackage"
	settingctrl "gymmaster-api/internal/controller/setting"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/service"
)

// RegisterCatalog: danh mục bán – gói tập, gói PT, khuyến mãi, cấu hình nghiệp vụ (/v1/admin).
func RegisterCatalog(d *Deps, admin *ghttp.RouterGroup) {
	admin.Bind(
		planctrl.NewV1(service.NewMembershipPlanService(repository.NewMembershipPlanRepository(d.DB), d.Audit)),
		ptpackagectrl.NewV1(service.NewPTPackageService(repository.NewPTPackageRepository(d.DB), d.Audit)),
		promotionctrl.NewV1(service.NewPromotionService(repository.NewPromotionRepository(d.DB), d.Audit)),
		settingctrl.NewV1(service.NewSettingAdminService(d.Settings, d.Audit)),
	)
}

// RegisterCatalogPublic: bảng giá và chi nhánh cho trang công khai (/v1/public, không cần đăng nhập).
func RegisterCatalogPublic(d *Deps, public *ghttp.RouterGroup) {
	svc := service.NewLandingService(
		repository.NewMembershipPlanRepository(d.DB),
		repository.NewPTPackageRepository(d.DB),
		repository.NewBranchRepository(d.DB),
	)
	public.Bind(landingctrl.NewV1(svc))
}
