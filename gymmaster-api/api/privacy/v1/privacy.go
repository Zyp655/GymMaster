package v1

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"

	"gymmaster-api/internal/dto"
)

type MFAStatusRes struct {
	Enabled                bool       `json:"enabled"`
	SetupPending           bool       `json:"setup_pending"`
	EnabledAt              *time.Time `json:"enabled_at"`
	RecoveryCodesRemaining int        `json:"recovery_codes_remaining"`
}

type SetupMFARes struct {
	Secret     string `json:"secret"`
	OTPAuthURL string `json:"otpauth_url"`
}

type EnableMFARes struct {
	Enabled       bool       `json:"enabled"`
	EnabledAt     *time.Time `json:"enabled_at"`
	RecoveryCodes []string   `json:"recovery_codes"`
}

type VerifyMFARes struct {
	Verified     bool `json:"verified"`
	UsedRecovery bool `json:"used_recovery,omitempty"`
}

type SessionItem struct {
	ID         uint       `json:"id"`
	UserID     uint       `json:"user_id"`
	UserAgent  string     `json:"user_agent"`
	IP         string     `json:"ip"`
	DeviceName string     `json:"device_name"`
	LastUsedAt *time.Time `json:"last_used_at"`
	ExpiresAt  time.Time  `json:"expires_at"`
	CreatedAt  time.Time  `json:"created_at"`
	IsCurrent  bool       `json:"is_current" dc:"Phiên của access token đang gọi API"`
}

type ConsentItem struct {
	Type      string    `json:"type"`
	Granted   bool      `json:"granted"`
	GrantedAt time.Time `json:"granted_at"`
	IP        string    `json:"ip,omitempty"`
}

type DataRequestItem struct {
	ID          uint           `json:"id"`
	MemberID    uint           `json:"member_id"`
	MemberCode  string         `json:"member_code,omitempty"`
	MemberName  string         `json:"member_name,omitempty"`
	MemberPhone string         `json:"member_phone,omitempty"`
	Type        string         `json:"type"`
	Status      string         `json:"status"`
	Reason      string         `json:"reason"`
	FileURL     string         `json:"file_url" dc:"Yêu cầu EXPORT đã xong: đường dẫn tải bản trích xuất (GET, cần access token)"`
	HandledBy   *uint          `json:"handled_by"`
	HandlerName string         `json:"handler_name,omitempty"`
	Note        string         `json:"note"`
	ExportData  map[string]any `json:"export_data,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	DoneAt      *time.Time     `json:"done_at"`
}

// ---- Security (Shared between /v1/admin and /v1/member) -------------------------

type GetMFAStatusReq struct {
	g.Meta `path:"/security/mfa" method:"get" tags:"Bảo mật & Quyền riêng tư" summary:"Trạng thái xác thực 2 lớp (2FA TOTP)"`
}

type SetupMFAReq struct {
	g.Meta `path:"/security/mfa/setup" method:"post" tags:"Bảo mật & Quyền riêng tư" summary:"Khởi tạo khóa bí mật TOTP & mã QR"`
}

type EnableMFAReq struct {
	g.Meta `path:"/security/mfa/enable" method:"post" tags:"Bảo mật & Quyền riêng tư" summary:"Xác nhận mã TOTP và kích hoạt 2FA"`
	Code   string `json:"code" v:"required#Vui lòng nhập mã xác thực 6 số"`
}

type DisableMFAReq struct {
	g.Meta `path:"/security/mfa/disable" method:"post" tags:"Bảo mật & Quyền riêng tư" summary:"Tắt xác thực 2 lớp (2FA)"`
	Code   string `json:"code" v:"required#Vui lòng nhập mã TOTP hoặc mã khôi phục"`
}
type DisableMFARes struct {
	Enabled bool `json:"enabled"`
}

type VerifyMFAReq struct {
	g.Meta `path:"/security/mfa/verify" method:"post" tags:"Bảo mật & Quyền riêng tư" summary:"Xác thực mã 2FA hoặc mã khôi phục"`
	Code   string `json:"code" v:"required#Vui lòng nhập mã xác thực"`
}

type ListSessionsReq struct {
	g.Meta `path:"/security/sessions" method:"get" tags:"Bảo mật & Quyền riêng tư" summary:"Danh sách phiên đăng nhập đang hoạt động"`
}
type ListSessionsRes []*SessionItem

type RevokeSessionReq struct {
	g.Meta `path:"/security/sessions/{id}" method:"delete" tags:"Bảo mật & Quyền riêng tư" summary:"Thu hồi một phiên đăng nhập"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type DeletePrivacyRes struct{}

type RevokeOtherSessionsReq struct {
	g.Meta           `path:"/security/sessions/revoke-others" method:"post" tags:"Bảo mật & Quyền riêng tư" summary:"Đăng xuất khỏi tất cả thiết bị khác (giữ phiên hiện tại)"`
	CurrentSessionID uint   `json:"current_session_id" dc:"Không bắt buộc: phiên hiện tại lấy từ access token (claim sid); chỉ cần với token cũ"`
	RefreshToken     string `json:"refresh_token"      dc:"Không bắt buộc: refresh token của phiên hiện tại (dùng khi access token không có sid)"`
}
type RevokeOtherSessionsRes struct {
	RevokedCount int64 `json:"revoked_count"`
}

// ---- Member Privacy (/v1/member) ------------------------------------------------

type MemberConsentsReq struct {
	g.Meta `path:"/consents" method:"get" tags:"Cổng hội viên" summary:"Trạng thái đồng ý xử lý dữ liệu cá nhân (Nghị định 13/2023)" role:"MEMBER"`
}
type MemberConsentsRes = []*ConsentItem

type UpdateMemberConsentsReq struct {
	g.Meta    `path:"/consents" method:"put" tags:"Cổng hội viên" summary:"Cập nhật tùy chọn đồng ý dữ liệu cá nhân" role:"MEMBER"`
	Terms     *bool         `json:"terms"`
	Marketing *bool         `json:"marketing"`
	Photo     *bool         `json:"photo"`
	Consents  []ConsentItem `json:"consents"`
	Items     []ConsentItem `json:"items"`
}
type UpdateMemberConsentsRes = []*ConsentItem

type MemberDataRequestsReq struct {
	g.Meta `path:"/data-requests" method:"get" tags:"Cổng hội viên" summary:"Danh sách yêu cầu xuất/xóa dữ liệu cá nhân của tôi" role:"MEMBER"`
}
type MemberDataRequestsRes []*DataRequestItem

type CreateDataRequestReq struct {
	g.Meta `path:"/data-requests" method:"post" tags:"Cổng hội viên" summary:"Gửi yêu cầu xuất hoặc xóa dữ liệu cá nhân" role:"MEMBER"`
	Type   string `json:"type"   v:"required|in:EXPORT,DELETE#Loại yêu cầu không hợp lệ"`
	Reason string `json:"reason" v:"max-length:500"`
}
type CreateDataRequestRes = DataRequestItem

type DownloadDataRequestReq struct {
	g.Meta `path:"/data-requests/{id}/download" method:"get" tags:"Cổng hội viên" summary:"Tải bản trích xuất dữ liệu cá nhân (JSON) của yêu cầu EXPORT đã hoàn tất" role:"MEMBER"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type DownloadDataRequestRes struct{}

// ---- Admin Privacy (/v1/admin) --------------------------------------------------

type ListDataRequestsReq struct {
	g.Meta `path:"/data-requests" method:"get" tags:"Bảo mật & Quyền riêng tư" summary:"Danh sách yêu cầu dữ liệu cá nhân (Nghị định 13/2023)" perm:"privacy.manage,data_request.process"`
	dto.PageReq
	Status   string `json:"status"    in:"query"`
	Type     string `json:"type"      in:"query"`
	BranchID uint   `json:"branch_id" in:"query" dc:"Lọc theo chi nhánh gốc của hội viên; 0 = mọi chi nhánh được phép"`
}
type ListDataRequestsRes = dto.PageRes[*DataRequestItem]

type ProcessDataRequestReq struct {
	g.Meta `path:"/data-requests/{id}/process" method:"post" tags:"Bảo mật & Quyền riêng tư" summary:"Duyệt hoặc từ chối yêu cầu dữ liệu cá nhân" perm:"data_request.process"`
	ID     uint   `json:"id"     in:"path" v:"required|min:1"`
	Action string `json:"action"           v:"required|in:APPROVE,REJECT#Hành động xử lý không hợp lệ"`
	Note   string `json:"note"             v:"max-length:255"`
}
type ProcessDataRequestRes = DataRequestItem
