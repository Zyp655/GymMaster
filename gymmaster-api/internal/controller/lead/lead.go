package lead

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"

	v1 "gymmaster-api/api/lead/v1"
	"gymmaster-api/internal/pkg/clientip"
	"gymmaster-api/internal/service"
)

type (
	AdminControllerV1 struct {
		svc service.LeadService
	}

	PublicControllerV1 struct {
		svc service.LeadService
	}
)

func NewAdminV1(svc service.LeadService) *AdminControllerV1 {
	return &AdminControllerV1{svc: svc}
}

func NewPublicV1(svc service.LeadService) *PublicControllerV1 {
	return &PublicControllerV1{svc: svc}
}

// ---- Public (/v1/public) --------------------------------------------------------

func (c *PublicControllerV1) CreateLead(ctx context.Context, req *v1.PublicCreateLeadReq) (*v1.PublicCreateLeadRes, error) {
	ip := ""
	if r := g.RequestFromCtx(ctx); r != nil {
		ip = clientip.Real(r)
	}
	return c.svc.PublicCreateLead(ctx, req, ip)
}

func (c *PublicControllerV1) ClassSchedules(ctx context.Context, req *v1.PublicClassSchedulesReq) (v1.PublicClassSchedulesRes, error) {
	return c.svc.PublicClassSchedules(ctx, req)
}

// ---- Admin (/v1/admin) ----------------------------------------------------------

func (c *AdminControllerV1) ListLeads(ctx context.Context, req *v1.ListLeadsReq) (*v1.ListLeadsRes, error) {
	return c.svc.ListLeads(ctx, req)
}

func (c *AdminControllerV1) GetLead(ctx context.Context, req *v1.GetLeadReq) (*v1.GetLeadRes, error) {
	return c.svc.GetLead(ctx, req.ID)
}

func (c *AdminControllerV1) CreateLead(ctx context.Context, req *v1.CreateLeadReq) (*v1.CreateLeadRes, error) {
	return c.svc.CreateLead(ctx, req)
}

func (c *AdminControllerV1) UpdateLead(ctx context.Context, req *v1.UpdateLeadReq) (*v1.UpdateLeadRes, error) {
	return c.svc.UpdateLead(ctx, req)
}

func (c *AdminControllerV1) UpdateLeadStatus(ctx context.Context, req *v1.UpdateLeadStatusReq) (*v1.UpdateLeadStatusRes, error) {
	return c.svc.UpdateLeadStatus(ctx, req.ID, req.Status)
}

func (c *AdminControllerV1) DeleteLead(ctx context.Context, req *v1.DeleteLeadReq) (*v1.DeleteLeadRes, error) {
	if err := c.svc.DeleteLead(ctx, req.ID); err != nil {
		return nil, err
	}
	return &v1.DeleteLeadRes{}, nil
}

func (c *AdminControllerV1) ListActivities(ctx context.Context, req *v1.ListLeadActivitiesReq) (v1.ListLeadActivitiesRes, error) {
	return c.svc.ListActivities(ctx, req.ID)
}

func (c *AdminControllerV1) AddActivity(ctx context.Context, req *v1.AddLeadActivityReq) (*v1.AddLeadActivityRes, error) {
	return c.svc.AddActivity(ctx, req)
}

func (c *AdminControllerV1) IssueTrialPass(ctx context.Context, req *v1.IssueTrialPassReq) (*v1.IssueTrialPassRes, error) {
	return c.svc.IssueTrialPass(ctx, req)
}

func (c *AdminControllerV1) ConvertLead(ctx context.Context, req *v1.ConvertLeadReq) (*v1.ConvertLeadRes, error) {
	return c.svc.ConvertLead(ctx, req)
}

func (c *AdminControllerV1) AtRiskMembers(ctx context.Context, req *v1.AtRiskMembersReq) (v1.AtRiskMembersRes, error) {
	return c.svc.AtRiskMembers(ctx, req)
}

func (c *AdminControllerV1) LeadsReport(ctx context.Context, req *v1.LeadsReportReq) (*v1.LeadsReportRes, error) {
	return c.svc.LeadsReport(ctx, req)
}
