package kpi

import (
	"context"

	v1 "gymmaster-api/api/kpi/v1"
	"gymmaster-api/internal/service"
)

type AdminControllerV1 struct {
	svc service.KPIService
}

type MemberControllerV1 struct {
	svc service.KPIService
}

type PublicControllerV1 struct {
	svc service.KPIService
}

func NewAdminV1(svc service.KPIService) *AdminControllerV1 {
	return &AdminControllerV1{svc: svc}
}

func NewMemberV1(svc service.KPIService) *MemberControllerV1 {
	return &MemberControllerV1{svc: svc}
}

func NewPublicV1(svc service.KPIService) *PublicControllerV1 {
	return &PublicControllerV1{svc: svc}
}

// ---- Public & Member ------------------------------------------------------------

func (c *PublicControllerV1) PublicOccupancy(ctx context.Context, _ *v1.PublicOccupancyReq) (res v1.PublicOccupancyRes, err error) {
	return c.svc.BranchOccupancy(ctx)
}

func (c *MemberControllerV1) MemberOccupancy(ctx context.Context, _ *v1.MemberOccupancyReq) (res v1.MemberOccupancyRes, err error) {
	return c.svc.BranchOccupancy(ctx)
}

// ---- Admin ----------------------------------------------------------------------

func (c *AdminControllerV1) ListSalesTargets(ctx context.Context, req *v1.ListSalesTargetsReq) (res *v1.ListSalesTargetsRes, err error) {
	return c.svc.ListSalesTargets(ctx, req)
}

func (c *AdminControllerV1) CreateSalesTarget(ctx context.Context, req *v1.CreateSalesTargetReq) (res *v1.CreateSalesTargetRes, err error) {
	return c.svc.CreateSalesTarget(ctx, req)
}

func (c *AdminControllerV1) DeleteSalesTarget(ctx context.Context, req *v1.DeleteSalesTargetReq) (res *v1.DeleteKPIRes, err error) {
	return &v1.DeleteKPIRes{}, c.svc.DeleteSalesTarget(ctx, req.ID)
}

func (c *AdminControllerV1) KPIReport(ctx context.Context, req *v1.KPIReportReq) (res *v1.KPIReportRes, err error) {
	return c.svc.KPIReport(ctx, req)
}

func (c *AdminControllerV1) CohortReport(ctx context.Context, req *v1.CohortReportReq) (res *v1.CohortReportRes, err error) {
	return c.svc.CohortReport(ctx, req)
}
