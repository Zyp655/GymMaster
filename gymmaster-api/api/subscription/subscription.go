// =================================================================================
// Interface controller – có thể sinh lại bằng `make ctrl` (gf gen ctrl).
// =================================================================================

package subscription

import (
	"context"

	v1 "gymmaster-api/api/subscription/v1"
)

type ISubscriptionV1 interface {
	Sell(ctx context.Context, req *v1.SellReq) (res *v1.SellRes, err error)
	Quote(ctx context.Context, req *v1.QuoteReq) (res *v1.QuoteRes, err error)
	List(ctx context.Context, req *v1.ListReq) (res *v1.ListRes, err error)
	Get(ctx context.Context, req *v1.GetReq) (res *v1.GetRes, err error)
	Events(ctx context.Context, req *v1.EventsReq) (res *v1.EventsRes, err error)
	MemberSubscriptions(ctx context.Context, req *v1.MemberSubscriptionsReq) (res *v1.MemberSubscriptionsRes, err error)
	Renew(ctx context.Context, req *v1.RenewReq) (res *v1.RenewRes, err error)
	RenewQuote(ctx context.Context, req *v1.RenewQuoteReq) (res *v1.RenewQuoteRes, err error)
	Upgrade(ctx context.Context, req *v1.UpgradeReq) (res *v1.UpgradeRes, err error)
	UpgradeQuote(ctx context.Context, req *v1.UpgradeQuoteReq) (res *v1.UpgradeQuoteRes, err error)
	Freeze(ctx context.Context, req *v1.FreezeReq) (res *v1.FreezeRes, err error)
	FreezeList(ctx context.Context, req *v1.FreezeListReq) (res *v1.FreezeListRes, err error)
	ApproveFreeze(ctx context.Context, req *v1.ApproveFreezeReq) (res *v1.ApproveFreezeRes, err error)
	RejectFreeze(ctx context.Context, req *v1.RejectFreezeReq) (res *v1.RejectFreezeRes, err error)
	EndFreeze(ctx context.Context, req *v1.EndFreezeReq) (res *v1.EndFreezeRes, err error)
	Transfer(ctx context.Context, req *v1.TransferReq) (res *v1.TransferRes, err error)
	Cancel(ctx context.Context, req *v1.CancelReq) (res *v1.CancelRes, err error)
}
