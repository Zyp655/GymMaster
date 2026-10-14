// =================================================================================
// Interface controller – có thể sinh lại bằng `make ctrl` (gf gen ctrl).
// =================================================================================

package portal

import (
	"context"

	v1 "gymmaster-api/api/portal/v1"
)

// IPortalV1: cổng hội viên (/v1/member/*, role MEMBER).
type IPortalV1 interface {
	GetProfile(ctx context.Context, req *v1.GetProfileReq) (res *v1.GetProfileRes, err error)
	UpdateProfile(ctx context.Context, req *v1.UpdateProfileReq) (res *v1.UpdateProfileRes, err error)
	Upload(ctx context.Context, req *v1.UploadReq) (res *v1.UploadRes, err error)
	QR(ctx context.Context, req *v1.QRReq) (res *v1.QRRes, err error)
	ListSubscriptions(ctx context.Context, req *v1.ListSubscriptionsReq) (res *v1.ListSubscriptionsRes, err error)
	RequestFreeze(ctx context.Context, req *v1.RequestFreezeReq) (res *v1.RequestFreezeRes, err error)
	ListCheckins(ctx context.Context, req *v1.ListCheckinsReq) (res *v1.ListCheckinsRes, err error)
	ListBranches(ctx context.Context, req *v1.ListBranchesReq) (res *v1.ListBranchesRes, err error)
	ListClasses(ctx context.Context, req *v1.ListClassesReq) (res *v1.ListClassesRes, err error)
	BookClass(ctx context.Context, req *v1.BookClassReq) (res *v1.BookClassRes, err error)
	ListBookings(ctx context.Context, req *v1.ListBookingsReq) (res *v1.ListBookingsRes, err error)
	CancelBooking(ctx context.Context, req *v1.CancelBookingReq) (res *v1.CancelBookingRes, err error)
	ListPTContracts(ctx context.Context, req *v1.ListPTContractsReq) (res *v1.ListPTContractsRes, err error)
	ListPTSessions(ctx context.Context, req *v1.ListPTSessionsReq) (res *v1.ListPTSessionsRes, err error)
	BookPTSession(ctx context.Context, req *v1.BookPTSessionReq) (res *v1.BookPTSessionRes, err error)
	CancelPTSession(ctx context.Context, req *v1.CancelPTSessionReq) (res *v1.CancelPTSessionRes, err error)
	TrainerAvailability(ctx context.Context, req *v1.TrainerAvailabilityReq) (res *v1.TrainerAvailabilityRes, err error)
	ListBodyMetrics(ctx context.Context, req *v1.ListBodyMetricsReq) (res *v1.ListBodyMetricsRes, err error)
	ListInvoices(ctx context.Context, req *v1.ListInvoicesReq) (res *v1.ListInvoicesRes, err error)
	GetInvoice(ctx context.Context, req *v1.GetInvoiceReq) (res *v1.GetInvoiceRes, err error)
}
