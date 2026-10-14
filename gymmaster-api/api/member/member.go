// =================================================================================
// Interface controller – có thể sinh lại bằng `make ctrl` (gf gen ctrl).
// =================================================================================

package member

import (
	"context"

	v1 "gymmaster-api/api/member/v1"
)

type IMemberV1 interface {
	List(ctx context.Context, req *v1.ListReq) (res *v1.ListRes, err error)
	Options(ctx context.Context, req *v1.OptionsReq) (res *v1.OptionsRes, err error)
	Get(ctx context.Context, req *v1.GetReq) (res *v1.GetRes, err error)
	Create(ctx context.Context, req *v1.CreateReq) (res *v1.CreateRes, err error)
	Update(ctx context.Context, req *v1.UpdateReq) (res *v1.UpdateRes, err error)
	Delete(ctx context.Context, req *v1.DeleteReq) (res *v1.DeleteRes, err error)
	CreatePortalAccount(ctx context.Context, req *v1.CreatePortalAccountReq) (res *v1.CreatePortalAccountRes, err error)
	LockPortalAccount(ctx context.Context, req *v1.LockPortalAccountReq) (res *v1.LockPortalAccountRes, err error)
	ResetQR(ctx context.Context, req *v1.ResetQRReq) (res *v1.ResetQRRes, err error)
	ListBodyMetrics(ctx context.Context, req *v1.ListBodyMetricsReq) (res *v1.ListBodyMetricsRes, err error)
	CreateBodyMetric(ctx context.Context, req *v1.CreateBodyMetricReq) (res *v1.CreateBodyMetricRes, err error)
	DeleteBodyMetric(ctx context.Context, req *v1.DeleteBodyMetricReq) (res *v1.DeleteBodyMetricRes, err error)
	ImportTemplate(ctx context.Context, req *v1.ImportTemplateReq) (res *v1.ImportTemplateRes, err error)
	Import(ctx context.Context, req *v1.ImportReq) (res *v1.ImportRes, err error)
}
