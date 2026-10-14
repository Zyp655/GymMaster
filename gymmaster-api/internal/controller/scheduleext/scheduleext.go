package scheduleext

import (
	"context"

	v1 "gymmaster-api/api/scheduleext/v1"
	"gymmaster-api/internal/service"
)

type AdminControllerV1 struct {
	svc service.ScheduleExtService
}

type PublicControllerV1 struct {
	svc service.ScheduleExtService
}

func NewAdminV1(svc service.ScheduleExtService) *AdminControllerV1 {
	return &AdminControllerV1{svc: svc}
}

func NewPublicV1(svc service.ScheduleExtService) *PublicControllerV1 {
	return &PublicControllerV1{svc: svc}
}

// ---- Admin ----------------------------------------------------------------------

func (c *AdminControllerV1) GetTrainerAvailability(ctx context.Context, req *v1.GetTrainerAvailabilityReq) (res v1.GetTrainerAvailabilityRes, err error) {
	return c.svc.GetTrainerAvailability(ctx, req)
}

func (c *AdminControllerV1) SaveTrainerAvailability(ctx context.Context, req *v1.SaveTrainerAvailabilityReq) (res v1.SaveTrainerAvailabilityRes, err error) {
	return c.svc.SaveTrainerAvailability(ctx, req)
}

func (c *AdminControllerV1) ListLeaveRequests(ctx context.Context, req *v1.ListLeaveRequestsReq) (res *v1.ListLeaveRequestsRes, err error) {
	return c.svc.ListLeaveRequests(ctx, req)
}

func (c *AdminControllerV1) CreateLeaveRequest(ctx context.Context, req *v1.CreateLeaveRequestReq) (res *v1.CreateLeaveRequestRes, err error) {
	return c.svc.CreateLeaveRequest(ctx, req)
}

func (c *AdminControllerV1) ApproveLeaveRequest(ctx context.Context, req *v1.ApproveLeaveRequestReq) (res *v1.ApproveLeaveRequestRes, err error) {
	return c.svc.ApproveLeaveRequest(ctx, req)
}

func (c *AdminControllerV1) RejectLeaveRequest(ctx context.Context, req *v1.RejectLeaveRequestReq) (res *v1.RejectLeaveRequestRes, err error) {
	return c.svc.RejectLeaveRequest(ctx, req)
}

func (c *AdminControllerV1) GetLeaveImpacts(ctx context.Context, req *v1.GetLeaveImpactsReq) (res *v1.GetLeaveImpactsRes, err error) {
	return c.svc.GetLeaveImpacts(ctx, req)
}

func (c *AdminControllerV1) SubstituteClassTrainer(ctx context.Context, req *v1.SubstituteClassTrainerReq) (res *v1.SubstituteClassTrainerRes, err error) {
	return c.svc.SubstituteClassTrainer(ctx, req)
}

func (c *AdminControllerV1) ReassignPTSession(ctx context.Context, req *v1.ReassignPTSessionReq) (res *v1.ReassignPTSessionRes, err error) {
	return c.svc.ReassignPTSession(ctx, req)
}

func (c *AdminControllerV1) ListBranchHolidays(ctx context.Context, req *v1.ListBranchHolidaysReq) (res *v1.ListBranchHolidaysRes, err error) {
	return c.svc.ListBranchHolidays(ctx, req)
}

func (c *AdminControllerV1) CreateBranchHoliday(ctx context.Context, req *v1.CreateBranchHolidayReq) (res *v1.CreateBranchHolidayRes, err error) {
	return c.svc.CreateBranchHoliday(ctx, req)
}

func (c *AdminControllerV1) DeleteBranchHoliday(ctx context.Context, req *v1.DeleteBranchHolidayReq) (res *v1.DeleteScheduleExtRes, err error) {
	return &v1.DeleteScheduleExtRes{}, c.svc.DeleteBranchHoliday(ctx, req.ID)
}

// ---- Public ---------------------------------------------------------------------

func (c *PublicControllerV1) PublicHolidays(ctx context.Context, req *v1.PublicHolidaysReq) (res v1.PublicHolidaysRes, err error) {
	return c.svc.PublicHolidays(ctx, req)
}
