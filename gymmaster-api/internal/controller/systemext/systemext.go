package systemext

import (
	"context"

	v1 "gymmaster-api/api/systemext/v1"
	"gymmaster-api/internal/service"
)

type Controller struct {
	svc *service.SystemExtService
}

func New(svc *service.SystemExtService) *Controller {
	return &Controller{svc: svc}
}

func (c *Controller) GlobalSearch(ctx context.Context, req *v1.GlobalSearchReq) (*v1.GlobalSearchRes, error) {
	return c.svc.GlobalSearch(ctx, req.Q)
}

func (c *Controller) BulkMemberAction(ctx context.Context, req *v1.BulkMemberActionReq) (*v1.BulkMemberActionRes, error) {
	return c.svc.BulkMemberAction(ctx, req)
}

func (c *Controller) ListBranchTransfers(ctx context.Context, _ *v1.ListBranchTransfersReq) (v1.ListBranchTransfersRes, error) {
	return c.svc.ListBranchTransfers(ctx)
}

func (c *Controller) CreateBranchTransfer(ctx context.Context, req *v1.CreateBranchTransferReq) (*v1.CreateBranchTransferRes, error) {
	return c.svc.CreateBranchTransfer(ctx, req)
}

func (c *Controller) RefundOnlineTxn(ctx context.Context, req *v1.RefundOnlineTxnReq) (*v1.RefundOnlineTxnRes, error) {
	return c.svc.RefundOnlineTxn(ctx, req)
}

func (c *Controller) ListLegacyImports(ctx context.Context, _ *v1.ListLegacyImportJobsReq) (v1.ListLegacyImportJobsRes, error) {
	return c.svc.ListLegacyImports(ctx)
}

func (c *Controller) RunLegacyImport(ctx context.Context, req *v1.RunLegacyImportReq) (*v1.RunLegacyImportRes, error) {
	return c.svc.RunLegacyImport(ctx, req)
}

func (c *Controller) GetSetupWizardStatus(ctx context.Context, _ *v1.GetSetupWizardStatusReq) (*v1.GetSetupWizardStatusRes, error) {
	return c.svc.GetSetupWizardStatus(ctx)
}

func (c *Controller) CompleteSetupWizard(ctx context.Context, req *v1.CompleteSetupWizardReq) (*v1.CompleteSetupWizardRes, error) {
	return c.svc.CompleteSetupWizard(ctx, req)
}
