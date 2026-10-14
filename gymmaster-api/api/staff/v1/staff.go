package v1

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"

	"gymmaster-api/internal/dto"
)

// StaffRole là một vai trò của nhân viên, có thể gắn với một chi nhánh (branch_id = null: toàn hệ thống).
type StaffRole struct {
	RoleID     uint   `json:"role_id"`
	RoleCode   string `json:"role_code"`
	RoleName   string `json:"role_name"`
	BranchID   *uint  `json:"branch_id"`
	BranchName string `json:"branch_name"`
}

type StaffProfile struct {
	StaffCode      string  `json:"staff_code"`
	Position       string  `json:"position"`
	Gender         string  `json:"gender"`
	Dob            *string `json:"dob"`
	IDCard         string  `json:"id_card"`
	Address        string  `json:"address"`
	HireDate       string  `json:"hire_date"`
	BaseSalary     float64 `json:"base_salary"`
	CommissionRate float64 `json:"commission_rate"`
	Note           string  `json:"note"`
}

type TrainerProfile struct {
	Level           string   `json:"level"`
	Specialties     []string `json:"specialties"`
	Bio             string   `json:"bio"`
	Certifications  string   `json:"certifications"`
	YearsExperience int      `json:"years_experience"`
}

type Staff struct {
	ID             uint            `json:"id"`
	Email          string          `json:"email"`
	Phone          string          `json:"phone"`
	FullName       string          `json:"full_name"`
	AvatarURL      string          `json:"avatar_url"`
	Status         string          `json:"status"`
	LastLoginAt    *time.Time      `json:"last_login_at"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	StaffProfile   *StaffProfile   `json:"staff_profile"`
	Roles          []StaffRole     `json:"roles"`
	TrainerProfile *TrainerProfile `json:"trainer_profile"`
}

// RoleAssignment: một dòng vai trò × chi nhánh khi tạo nhân viên / gán lại vai trò.
type RoleAssignment struct {
	RoleCode string `json:"role_code" v:"required#Vui lòng chọn vai trò"`
	BranchID *uint  `json:"branch_id" dc:"null = toàn hệ thống (chỉ OWNER được gán)"`
}

type TrainerProfileInput struct {
	Level           string   `json:"level"            v:"in:JUNIOR,SENIOR,MASTER#Cấp độ HLV không hợp lệ" d:"JUNIOR"`
	Specialties     []string `json:"specialties"      dc:"Chuyên môn, VD: [\"Yoga\",\"Giảm cân\"]"`
	Bio             string   `json:"bio"              v:"max-length:2000#Giới thiệu tối đa 2000 ký tự"`
	Certifications  string   `json:"certifications"   v:"max-length:2000#Chứng chỉ tối đa 2000 ký tự"`
	YearsExperience int      `json:"years_experience" v:"between:0,60#Số năm kinh nghiệm từ 0 đến 60"`
}

// StaffInput là phần thông tin dùng chung cho tạo/sửa nhân viên.
type StaffInput struct {
	FullName       string               `json:"full_name"       v:"required|max-length:150#Vui lòng nhập họ tên|Họ tên tối đa 150 ký tự"`
	Email          string               `json:"email"           v:"required|email|max-length:150#Vui lòng nhập email|Email không hợp lệ|Email tối đa 150 ký tự"`
	Phone          string               `json:"phone"           v:"regex:^[0-9+ ]{8,20}$#Số điện thoại không hợp lệ"`
	Position       string               `json:"position"        v:"required|in:MANAGER,RECEPTIONIST,TRAINER,OTHER#Vui lòng chọn chức vụ|Chức vụ không hợp lệ"`
	Gender         string               `json:"gender"          v:"in:MALE,FEMALE,OTHER#Giới tính không hợp lệ" d:"OTHER"`
	Dob            string               `json:"dob"             v:"date-format:Y-m-d#Ngày sinh phải có dạng YYYY-MM-DD"`
	IDCard         string               `json:"id_card"         v:"max-length:20#Số CCCD tối đa 20 ký tự"`
	Address        string               `json:"address"         v:"max-length:255#Địa chỉ tối đa 255 ký tự"`
	HireDate       string               `json:"hire_date"       v:"date-format:Y-m-d#Ngày vào làm phải có dạng YYYY-MM-DD" dc:"Mặc định: hôm nay"`
	BaseSalary     float64              `json:"base_salary"     v:"min:0#Lương cơ bản không được âm"`
	CommissionRate float64              `json:"commission_rate" v:"between:0,100#Tỷ lệ hoa hồng từ 0 đến 100"`
	Note           string               `json:"note"            v:"max-length:2000#Ghi chú tối đa 2000 ký tự"`
	TrainerProfile *TrainerProfileInput `json:"trainer_profile" dc:"Chỉ dùng khi position = TRAINER"`
}

type ListReq struct {
	g.Meta `path:"/staff" method:"get" tags:"Nhân sự" summary:"Danh sách nhân viên" perm:"staff.view"`
	dto.PageReq
	BranchID uint   `json:"branch_id" in:"query" dc:"Lọc theo chi nhánh"`
	Position string `json:"position"  in:"query" v:"in:MANAGER,RECEPTIONIST,TRAINER,OTHER#Chức vụ không hợp lệ"`
	Status   string `json:"status"    in:"query" v:"in:ACTIVE,LOCKED#Trạng thái không hợp lệ"`
}
type ListRes = dto.PageRes[*Staff]

type GetReq struct {
	g.Meta `path:"/staff/{id}" method:"get" tags:"Nhân sự" summary:"Chi tiết nhân viên" perm:"staff.view"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type GetRes = Staff

type CreateReq struct {
	g.Meta `path:"/staff" method:"post" tags:"Nhân sự" summary:"Thêm nhân viên (tạo tài khoản + hồ sơ + vai trò)" perm:"staff.create"`
	StaffInput
	Password string           `json:"password" v:"required|length:8,72#Vui lòng nhập mật khẩu ban đầu|Mật khẩu từ 8 đến 72 ký tự"`
	Roles    []RoleAssignment `json:"roles"    v:"required#Vui lòng gán ít nhất một vai trò"`
}
type CreateRes = Staff

type UpdateReq struct {
	g.Meta `path:"/staff/{id}" method:"put" tags:"Nhân sự" summary:"Cập nhật thông tin nhân viên" perm:"staff.update"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
	StaffInput
}
type UpdateRes = Staff

type AssignRolesReq struct {
	g.Meta `path:"/staff/{id}/roles" method:"put" tags:"Nhân sự" summary:"Gán lại vai trò cho nhân viên" perm:"staff.update"`
	ID     uint             `json:"id"    in:"path" v:"required|min:1"`
	Roles  []RoleAssignment `json:"roles" v:"required#Vui lòng gán ít nhất một vai trò"`
}
type AssignRolesRes = Staff

type LockReq struct {
	g.Meta `path:"/staff/{id}/lock" method:"post" tags:"Nhân sự" summary:"Khóa tài khoản (thu hồi mọi phiên đăng nhập)" perm:"staff.update"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type LockRes = Staff

type UnlockReq struct {
	g.Meta `path:"/staff/{id}/unlock" method:"post" tags:"Nhân sự" summary:"Mở khóa tài khoản" perm:"staff.update"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type UnlockRes = Staff

type ResetPasswordReq struct {
	g.Meta      `path:"/staff/{id}/reset-password" method:"post" tags:"Nhân sự" summary:"Đặt lại mật khẩu" perm:"staff.update"`
	ID          uint   `json:"id"           in:"path" v:"required|min:1"`
	NewPassword string `json:"new_password" v:"required|length:8,72#Vui lòng nhập mật khẩu mới|Mật khẩu từ 8 đến 72 ký tự"`
}
type ResetPasswordRes struct{}

type DeleteReq struct {
	g.Meta `path:"/staff/{id}" method:"delete" tags:"Nhân sự" summary:"Xóa nhân viên (xóa mềm)" perm:"staff.delete"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type DeleteRes struct{}

// StaffOption dùng cho dropdown chọn nhân viên (xếp ca...).
type StaffOption struct {
	ID        uint   `json:"id"`
	FullName  string `json:"full_name"`
	StaffCode string `json:"staff_code"`
	Position  string `json:"position"`
}

type OptionsReq struct {
	g.Meta   `path:"/staff/options" method:"get" tags:"Nhân sự" summary:"Danh sách nhân viên đang hoạt động (dropdown)" perm:"staff.view,shift.view,shift.manage"`
	BranchID uint `json:"branch_id" in:"query" dc:"Lọc theo chi nhánh"`
}
type OptionsRes []*StaffOption

// TrainerOption dùng cho dropdown chọn HLV (lớp học, PT).
type TrainerOption struct {
	ID          uint     `json:"id"`
	FullName    string   `json:"full_name"`
	Level       string   `json:"level"`
	Specialties []string `json:"specialties"`
}

type TrainerOptionsReq struct {
	g.Meta   `path:"/trainers/options" method:"get" tags:"Nhân sự" summary:"Danh sách HLV đang hoạt động (dropdown)" perm:"staff.view,class.view,pt.view"`
	BranchID uint `json:"branch_id" in:"query" dc:"Lọc theo chi nhánh"`
}
type TrainerOptionsRes []*TrainerOption
