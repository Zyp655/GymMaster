// =================================================================================
// Interface controller – có thể sinh lại bằng `make ctrl` (gf gen ctrl).
// =================================================================================

package report

import (
	"context"

	v1 "gymmaster-api/api/report/v1"
)

type IReportV1 interface {
	Overview(ctx context.Context, req *v1.OverviewReq) (res *v1.OverviewRes, err error)
	Revenue(ctx context.Context, req *v1.RevenueReq) (res *v1.RevenueRes, err error)
	Members(ctx context.Context, req *v1.MembersReq) (res *v1.MembersRes, err error)
	Retention(ctx context.Context, req *v1.RetentionReq) (res *v1.RetentionRes, err error)
	CheckinHeatmap(ctx context.Context, req *v1.CheckinHeatmapReq) (res *v1.CheckinHeatmapRes, err error)
	ClassOccupancy(ctx context.Context, req *v1.ClassOccupancyReq) (res *v1.ClassOccupancyRes, err error)
	TrainerPerformance(ctx context.Context, req *v1.TrainerPerformanceReq) (res *v1.TrainerPerformanceRes, err error)
	SalesProducts(ctx context.Context, req *v1.SalesProductsReq) (res *v1.SalesProductsRes, err error)
	Subscriptions(ctx context.Context, req *v1.SubscriptionsReq) (res *v1.SubscriptionsRes, err error)
	Export(ctx context.Context, req *v1.ExportReq) (res *v1.ExportRes, err error)
}
