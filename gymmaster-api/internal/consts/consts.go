package consts

// Khóa lưu thông tin người dùng đã xác thực trong context của request.
const CtxAuthKey = "auth_payload"

// Header
const (
	HeaderAuthorization = "Authorization"
	BearerPrefix        = "Bearer "
)

// Role hệ thống (seed trong migration 000001).
const (
	RoleOwner        = "OWNER"
	RoleManager      = "MANAGER"
	RoleReceptionist = "RECEPTIONIST"
	RoleTrainer      = "TRAINER"
	RoleMember       = "MEMBER"
)

// Trạng thái dùng chung.
const (
	StatusActive   = "ACTIVE"
	StatusInactive = "INACTIVE"
	StatusLocked   = "LOCKED"
)

// MetaTagPermission là tên tag trong g.Meta khai báo quyền cần có, ví dụ:
//
//	g.Meta `path:"/branches" method:"get" perm:"branch.view"`
const MetaTagPermission = "perm"

// MetaTagRole là tag trong g.Meta yêu cầu người dùng có một trong các role (ngăn bởi dấu phẩy),
// dùng cho cổng hội viên: role:"MEMBER".
const MetaTagRole = "role"

// Định dạng ngày/giờ dùng trong API (giờ Việt Nam).
const (
	DateLayout = "2006-01-02"
	TimeZone   = "Asia/Ho_Chi_Minh"
)
