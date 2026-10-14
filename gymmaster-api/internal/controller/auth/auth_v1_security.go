package auth

import (
	"context"

	v1 "gymmaster-api/api/auth/v1"
)

func (c *ControllerV1) ForgotPassword(ctx context.Context, req *v1.ForgotPasswordReq) (res *v1.ForgotPasswordRes, err error) {
	return c.authService.ForgotPassword(ctx, req.Email, req.Portal, clientInfo(ctx))
}

func (c *ControllerV1) ResetPassword(ctx context.Context, req *v1.ResetPasswordReq) (res *v1.ResetPasswordRes, err error) {
	return c.authService.ResetPassword(ctx, req.Token, req.NewPassword)
}

func (c *ControllerV1) RegisterMember(ctx context.Context, req *v1.RegisterMemberReq) (res *v1.RegisterMemberRes, err error) {
	return c.authService.RegisterMember(ctx, req, clientInfo(ctx))
}

func (c *ControllerV1) VerifyMemberOTP(ctx context.Context, req *v1.VerifyMemberOTPReq) (res *v1.VerifyMemberOTPRes, err error) {
	return c.authService.VerifyMemberOTP(ctx, req, clientInfo(ctx))
}
