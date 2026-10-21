package privacy

import (
	"context"
	"fmt"

	"github.com/gogf/gf/v2/frame/g"

	v1 "gymmaster-api/api/privacy/v1"
	"gymmaster-api/internal/service"
)

type SecurityControllerV1 struct {
	svc service.PrivacyService
}

type AdminControllerV1 struct {
	svc service.PrivacyService
}

type MemberControllerV1 struct {
	svc service.PrivacyService
}

func NewSecurityV1(svc service.PrivacyService) *SecurityControllerV1 {
	return &SecurityControllerV1{svc: svc}
}

func NewAdminV1(svc service.PrivacyService) *AdminControllerV1 {
	return &AdminControllerV1{svc: svc}
}

func NewMemberV1(svc service.PrivacyService) *MemberControllerV1 {
	return &MemberControllerV1{svc: svc}
}

// ---- Security (Shared between Admin & Member) -----------------------------------

func (c *SecurityControllerV1) GetMFAStatus(ctx context.Context, _ *v1.GetMFAStatusReq) (res *v1.MFAStatusRes, err error) {
	return c.svc.GetMFAStatus(ctx)
}

func (c *SecurityControllerV1) SetupMFA(ctx context.Context, _ *v1.SetupMFAReq) (res *v1.SetupMFARes, err error) {
	return c.svc.SetupMFA(ctx)
}

func (c *SecurityControllerV1) EnableMFA(ctx context.Context, req *v1.EnableMFAReq) (res *v1.EnableMFARes, err error) {
	return c.svc.EnableMFA(ctx, req)
}

func (c *SecurityControllerV1) DisableMFA(ctx context.Context, req *v1.DisableMFAReq) (res *v1.DisableMFARes, err error) {
	return c.svc.DisableMFA(ctx, req)
}

func (c *SecurityControllerV1) VerifyMFA(ctx context.Context, req *v1.VerifyMFAReq) (res *v1.VerifyMFARes, err error) {
	return c.svc.VerifyMFA(ctx, req)
}

func (c *SecurityControllerV1) ListSessions(ctx context.Context, _ *v1.ListSessionsReq) (res v1.ListSessionsRes, err error) {
	return c.svc.ListSessions(ctx)
}

func (c *SecurityControllerV1) RevokeSession(ctx context.Context, req *v1.RevokeSessionReq) (res *v1.DeletePrivacyRes, err error) {
	return &v1.DeletePrivacyRes{}, c.svc.RevokeSession(ctx, req.ID)
}

func (c *SecurityControllerV1) RevokeOtherSessions(ctx context.Context, req *v1.RevokeOtherSessionsReq) (res *v1.RevokeOtherSessionsRes, err error) {
	return c.svc.RevokeOtherSessions(ctx, req)
}

// ---- Member Privacy -------------------------------------------------------------

func (c *MemberControllerV1) MemberConsents(ctx context.Context, _ *v1.MemberConsentsReq) (res v1.MemberConsentsRes, err error) {
	return c.svc.MemberConsents(ctx)
}

func (c *MemberControllerV1) UpdateMemberConsents(ctx context.Context, req *v1.UpdateMemberConsentsReq) (res v1.UpdateMemberConsentsRes, err error) {
	return c.svc.UpdateMemberConsents(ctx, req)
}

func (c *MemberControllerV1) MemberDataRequests(ctx context.Context, _ *v1.MemberDataRequestsReq) (res v1.MemberDataRequestsRes, err error) {
	return c.svc.MemberDataRequests(ctx)
}

func (c *MemberControllerV1) CreateDataRequest(ctx context.Context, req *v1.CreateDataRequestReq) (res *v1.CreateDataRequestRes, err error) {
	return c.svc.CreateDataRequest(ctx, req)
}

// ---- Admin Privacy --------------------------------------------------------------

func (c *AdminControllerV1) ListDataRequests(ctx context.Context, req *v1.ListDataRequestsReq) (res *v1.ListDataRequestsRes, err error) {
	return c.svc.ListDataRequests(ctx, req)
}

func (c *AdminControllerV1) ProcessDataRequest(ctx context.Context, req *v1.ProcessDataRequestReq) (res *v1.ProcessDataRequestRes, err error) {
	return c.svc.ProcessDataRequest(ctx, req)
}

// DownloadDataRequest ghi thẳng file JSON vào response (middleware Response bỏ qua khi handler đã tự ghi).
func (c *MemberControllerV1) DownloadDataRequest(ctx context.Context, req *v1.DownloadDataRequestReq) (res *v1.DownloadDataRequestRes, err error) {
	data, filename, err := c.svc.DownloadDataRequest(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	r := g.RequestFromCtx(ctx)
	h := r.Response.Header()
	h.Set("Content-Type", "application/json; charset=utf-8")
	h.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	h.Set("Cache-Control", "no-store")
	r.Response.Write(data)
	return nil, nil
}
