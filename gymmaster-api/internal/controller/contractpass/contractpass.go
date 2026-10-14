package contractpass

import (
	"context"

	v1 "gymmaster-api/api/contractpass/v1"
	"gymmaster-api/internal/service"
)

type AdminControllerV1 struct {
	svc service.ContractPassService
}

type MemberControllerV1 struct {
	svc service.ContractPassService
}

func NewAdminV1(svc service.ContractPassService) *AdminControllerV1 {
	return &AdminControllerV1{svc: svc}
}

func NewMemberV1(svc service.ContractPassService) *MemberControllerV1 {
	return &MemberControllerV1{svc: svc}
}

// ---- Admin ----------------------------------------------------------------------

func (c *AdminControllerV1) ListTemplates(ctx context.Context, req *v1.ListContractTemplatesReq) (res *v1.ListContractTemplatesRes, err error) {
	return c.svc.ListTemplates(ctx, req)
}

func (c *AdminControllerV1) CreateTemplate(ctx context.Context, req *v1.CreateContractTemplateReq) (res *v1.CreateContractTemplateRes, err error) {
	return c.svc.CreateTemplate(ctx, req.ContractTemplateInput)
}

func (c *AdminControllerV1) UpdateTemplate(ctx context.Context, req *v1.UpdateContractTemplateReq) (res *v1.UpdateContractTemplateRes, err error) {
	return c.svc.UpdateTemplate(ctx, req.ID, req.ContractTemplateInput)
}

func (c *AdminControllerV1) DeleteTemplate(ctx context.Context, req *v1.DeleteContractTemplateReq) (res *v1.DeleteContractPassRes, err error) {
	return &v1.DeleteContractPassRes{}, c.svc.DeleteTemplate(ctx, req.ID)
}

func (c *AdminControllerV1) ListMemberContracts(ctx context.Context, req *v1.ListMemberContractsReq) (res *v1.ListMemberContractsRes, err error) {
	return c.svc.ListMemberContracts(ctx, req)
}

func (c *AdminControllerV1) CreateMemberContract(ctx context.Context, req *v1.CreateMemberContractReq) (res *v1.CreateMemberContractRes, err error) {
	return c.svc.CreateMemberContract(ctx, req)
}

func (c *AdminControllerV1) ListDayPasses(ctx context.Context, req *v1.ListDayPassesReq) (res *v1.ListDayPassesRes, err error) {
	return c.svc.ListDayPasses(ctx, req)
}

func (c *AdminControllerV1) CreateDayPass(ctx context.Context, req *v1.CreateDayPassReq) (res *v1.CreateDayPassRes, err error) {
	return c.svc.CreateDayPass(ctx, req)
}

// ---- Member ---------------------------------------------------------------------

func (c *MemberControllerV1) MemberContracts(ctx context.Context, _ *v1.MemberContractsReq) (res v1.MemberContractsRes, err error) {
	return c.svc.MemberContracts(ctx)
}

func (c *MemberControllerV1) SignMemberContract(ctx context.Context, req *v1.SignMemberContractReq) (res *v1.SignMemberContractRes, err error) {
	return c.svc.SignMemberContract(ctx, req)
}

func (c *MemberControllerV1) MemberGuestPasses(ctx context.Context, _ *v1.MemberGuestPassesReq) (res *v1.MemberGuestPassesRes, err error) {
	return c.svc.MemberGuestPasses(ctx)
}

func (c *MemberControllerV1) CreateGuestPass(ctx context.Context, req *v1.CreateGuestPassReq) (res *v1.CreateGuestPassRes, err error) {
	return c.svc.CreateGuestPass(ctx, req)
}
