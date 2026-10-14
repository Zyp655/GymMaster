// =================================================================================
// Interface controller – có thể sinh lại bằng `make ctrl` (gf gen ctrl).
// =================================================================================

package pt

import (
	"context"

	v1 "gymmaster-api/api/pt/v1"
)

type IPtV1 interface {
	Quote(ctx context.Context, req *v1.QuoteReq) (res *v1.QuoteRes, err error)
	Sell(ctx context.Context, req *v1.SellReq) (res *v1.SellRes, err error)
	ContractList(ctx context.Context, req *v1.ContractListReq) (res *v1.ContractListRes, err error)
	ContractGet(ctx context.Context, req *v1.ContractGetReq) (res *v1.ContractGetRes, err error)
	MemberContracts(ctx context.Context, req *v1.MemberContractsReq) (res *v1.MemberContractsRes, err error)
	ChangeTrainerCheck(ctx context.Context, req *v1.ChangeTrainerCheckReq) (res *v1.ChangeTrainerCheckRes, err error)
	ChangeTrainer(ctx context.Context, req *v1.ChangeTrainerReq) (res *v1.ChangeTrainerRes, err error)
	CancelContract(ctx context.Context, req *v1.CancelContractReq) (res *v1.CancelContractRes, err error)

	SessionList(ctx context.Context, req *v1.SessionListReq) (res *v1.SessionListRes, err error)
	SessionGet(ctx context.Context, req *v1.SessionGetReq) (res *v1.SessionGetRes, err error)
	SessionCreate(ctx context.Context, req *v1.SessionCreateReq) (res *v1.SessionCreateRes, err error)
	SessionUpdate(ctx context.Context, req *v1.SessionUpdateReq) (res *v1.SessionUpdateRes, err error)
	SessionCancel(ctx context.Context, req *v1.SessionCancelReq) (res *v1.SessionCancelRes, err error)
	SessionComplete(ctx context.Context, req *v1.SessionCompleteReq) (res *v1.SessionCompleteRes, err error)
	SessionNoShow(ctx context.Context, req *v1.SessionNoShowReq) (res *v1.SessionNoShowRes, err error)
	TrainerSchedule(ctx context.Context, req *v1.TrainerScheduleReq) (res *v1.TrainerScheduleRes, err error)

	CommissionList(ctx context.Context, req *v1.CommissionListReq) (res *v1.CommissionListRes, err error)
	CommissionSummary(ctx context.Context, req *v1.CommissionSummaryReq) (res *v1.CommissionSummaryRes, err error)
	CommissionPay(ctx context.Context, req *v1.CommissionPayReq) (res *v1.CommissionPayRes, err error)
}
