package v1

import "github.com/gogf/gf/v2/frame/g"

type UserInfo struct {
	ID          uint     `json:"id"`
	Email       string   `json:"email"`
	FullName    string   `json:"full_name"`
	Phone       string   `json:"phone"`
	AvatarURL   string   `json:"avatar_url"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	BranchIDs   []uint   `json:"branch_ids" dc:"Rỗng = làm việc trên toàn hệ thống"`
}

type TokenRes struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresIn    int       `json:"expires_in" dc:"Số giây access token còn hiệu lực"`
	User         *UserInfo `json:"user"`
}

type LoginReq struct {
	g.Meta   `path:"/auth/login" method:"post" tags:"Xác thực" summary:"Đăng nhập bằng email + mật khẩu"`
	Email    string `json:"email"    v:"required|email#Vui lòng nhập email|Email không hợp lệ"`
	Password string `json:"password" v:"required#Vui lòng nhập mật khẩu"`
}
type LoginRes = TokenRes

type RefreshReq struct {
	g.Meta       `path:"/auth/refresh" method:"post" tags:"Xác thực" summary:"Cấp lại access token (xoay vòng refresh token)"`
	RefreshToken string `json:"refresh_token" v:"required#Thiếu refresh token"`
}
type RefreshRes = TokenRes

type LogoutReq struct {
	g.Meta       `path:"/auth/logout" method:"post" tags:"Xác thực" summary:"Đăng xuất (thu hồi refresh token)"`
	RefreshToken string `json:"refresh_token"`
}
type LogoutRes struct{}

type MeReq struct {
	g.Meta `path:"/auth/me" method:"get" tags:"Xác thực" summary:"Thông tin người dùng hiện tại"`
}
type MeRes = UserInfo

type ChangePasswordReq struct {
	g.Meta      `path:"/auth/change-password" method:"put" tags:"Xác thực" summary:"Đổi mật khẩu"`
	OldPassword string `json:"old_password" v:"required#Vui lòng nhập mật khẩu hiện tại"`
	NewPassword string `json:"new_password" v:"required|length:8,72|different:OldPassword#Vui lòng nhập mật khẩu mới|Mật khẩu mới từ 8 đến 72 ký tự|Mật khẩu mới phải khác mật khẩu hiện tại"`
}
type ChangePasswordRes struct{}
