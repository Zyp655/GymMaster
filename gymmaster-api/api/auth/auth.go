// =================================================================================
// Interface controller – có thể sinh lại bằng `make ctrl` (gf gen ctrl).
// =================================================================================

package auth

import (
	"context"

	v1 "gymmaster-api/api/auth/v1"
)

type IAuthV1 interface {
	Login(ctx context.Context, req *v1.LoginReq) (res *v1.LoginRes, err error)
	Refresh(ctx context.Context, req *v1.RefreshReq) (res *v1.RefreshRes, err error)
	Logout(ctx context.Context, req *v1.LogoutReq) (res *v1.LogoutRes, err error)
	Me(ctx context.Context, req *v1.MeReq) (res *v1.MeRes, err error)
	ChangePassword(ctx context.Context, req *v1.ChangePasswordReq) (res *v1.ChangePasswordRes, err error)
	ForgotPassword(ctx context.Context, req *v1.ForgotPasswordReq) (res *v1.ForgotPasswordRes, err error)
	ResetPassword(ctx context.Context, req *v1.ResetPasswordReq) (res *v1.ResetPasswordRes, err error)
	RegisterMember(ctx context.Context, req *v1.RegisterMemberReq) (res *v1.RegisterMemberRes, err error)
	VerifyMemberOTP(ctx context.Context, req *v1.VerifyMemberOTPReq) (res *v1.VerifyMemberOTPRes, err error)
}

