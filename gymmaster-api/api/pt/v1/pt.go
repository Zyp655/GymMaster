package v1

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"

	invoicev1 "gymmaster-api/api/invoice/v1"
	"gymmaster-api/internal/dto"
)

// =================================================================================
// Đối tượng trả về
// =================================================================================

// Contract là một dòng danh sách hợp đồng PT.
type Contract struct {
	ID                uint      `json:"id"`
	MemberID          uint      `json:"member_id"`
	MemberCode        string    `json:"member_code"`
	MemberName        string    `json:"member_name"`
	MemberPhone       string    `json:"member_phone"`
	TrainerID         uint      `json:"trainer_id"`
	TrainerName       string    `json:"trainer_name"`
	PTPackageID       uint      `json:"pt_package_id"`
	PackageCode       string    `json:"package_code"`
	PackageName       string    `json:"package_name"`
	BranchID          uint      `json:"branch_id"`
	BranchName        string    `json:"branch_name"`
	InvoiceID         *uint     `json:"invoice_id"`
	InvoiceNo         string    `json:"invoice_no"`
	InvoiceStatus     string    `json:"invoice_status"`
	TotalSessions     int       `json:"total_sessions"`
	UsedSessions      int       `json:"used_sessions"      dc:"Số buổi đã trừ (hoàn thành + vắng mặt/hủy muộn bị tính)"`
	ScheduledSessions int       `json:"scheduled_sessions" dc:"Số buổi đang đặt lịch (SCHEDULED)"`
	RemainingSessions int       `json:"remaining_sessions" dc:"= total − used"`
	BookableSessions  int       `json:"bookable_sessions"  dc:"Còn đặt được = total − used − scheduled"`
	SessionMinutes    int       `json:"session_minutes"`
	StartDate         string    `json:"start_date"         dc:"YYYY-MM-DD"`
	EndDate           string    `json:"end_date"           dc:"YYYY-MM-DD"`
	Price             float64   `json:"price"              dc:"Giá trị hợp đồng (số tiền thực bán sau khuyến mãi)"`
	PricePerSession   float64   `json:"price_per_session"  dc:"= round(price / total_sessions) – cơ sở tính hoa hồng"`
	Status            string    `json:"status"             dc:"PENDING | ACTIVE | COMPLETED | EXPIRED | CANCELLED"`
	DaysLeft          *int      `json:"days_left"          dc:"Số ngày còn lại đến end_date (PENDING/ACTIVE)"`
	SoldBy            *uint     `json:"sold_by"`
	SoldByName        string    `json:"sold_by_name"`
	Note              string    `json:"note"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type PackageRef struct {
	ID             uint    `json:"id"`
	Code           string  `json:"code"`
	Name           string  `json:"name"`
	SessionCount   int     `json:"session_count"`
	SessionMinutes int     `json:"session_minutes"`
	ValidityDays   int     `json:"validity_days"`
	Price          float64 `json:"price"`
	Status         string  `json:"status"`
}

type MemberRef struct {
	ID         uint   `json:"id"`
	MemberCode string `json:"member_code"`
	FullName   string `json:"full_name"`
	Phone      string `json:"phone"`
	Email      string `json:"email"`
	Status     string `json:"status"`
}

// ContractDetail là chi tiết hợp đồng kèm gói, hội viên, hóa đơn và các buổi tập.
type ContractDetail struct {
	Contract
	Package  PackageRef               `json:"package"`
	Member   MemberRef                `json:"member"`
	Invoice  *invoicev1.InvoiceDetail `json:"invoice"`
	Sessions []*Session               `json:"sessions"`
}

// Session là một buổi PT.
type Session struct {
	ID            uint       `json:"id"`
	ContractID    uint       `json:"contract_id"`
	TrainerID     uint       `json:"trainer_id"`
	TrainerName   string     `json:"trainer_name"`
	MemberID      uint       `json:"member_id"`
	MemberCode    string     `json:"member_code"`
	MemberName    string     `json:"member_name"`
	MemberPhone   string     `json:"member_phone"`
	BranchID      uint       `json:"branch_id"`
	BranchName    string     `json:"branch_name"`
	PackageName   string     `json:"package_name"`
	StartAt       time.Time  `json:"start_at"`
	EndAt         time.Time  `json:"end_at"`
	Status        string     `json:"status"  dc:"SCHEDULED | COMPLETED | CANCELLED | NO_SHOW"`
	Counted       bool       `json:"counted" dc:"true = buổi đã trừ vào hợp đồng (COMPLETED, hoặc NO_SHOW đã chốt tính buổi)"`
	WorkoutNote   string     `json:"workout_note"`
	TrainerNote   string     `json:"trainer_note"`
	CompletedAt   *time.Time `json:"completed_at" dc:"Thời điểm chốt buổi (hoàn thành / tính buổi vắng mặt)"`
	CancelledAt   *time.Time `json:"cancelled_at"`
	CancelReason  string     `json:"cancel_reason"`
	Source        string     `json:"source"  dc:"COUNTER | PORTAL | TRAINER"`
	CreatedBy     *uint      `json:"created_by"`
	CreatedByName string     `json:"created_by_name"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// Commission là một dòng hoa hồng HLV.
type Commission struct {
	ID           uint       `json:"id"`
	TrainerID    uint       `json:"trainer_id"`
	TrainerName  string     `json:"trainer_name"`
	BranchID     uint       `json:"branch_id"`
	BranchName   string     `json:"branch_name"`
	SourceType   string     `json:"source_type" dc:"PT_SESSION | CLASS"`
	SourceID     uint       `json:"source_id"`
	SessionStart *time.Time `json:"session_start_at" dc:"Giờ bắt đầu buổi PT (source PT_SESSION)"`
	ContractID   *uint      `json:"contract_id"`
	MemberName   string     `json:"member_name"`
	BaseAmount   float64    `json:"base_amount" dc:"Giá mỗi buổi"`
	Rate         float64    `json:"rate"        dc:"% hoa hồng"`
	Amount       float64    `json:"amount"`
	Period       string     `json:"period"      dc:"YYYY-MM"`
	Status       string     `json:"status"      dc:"PENDING | PAID"`
	PaidAt       *time.Time `json:"paid_at"`
	PaidBy       *uint      `json:"paid_by"`
	PaidByName   string     `json:"paid_by_name"`
	CreatedAt    time.Time  `json:"created_at"`
}

// =================================================================================
// Hợp đồng PT
// =================================================================================

type PaymentInput struct {
	Amount    float64 `json:"amount"    dc:"Số tiền thu (> 0, ≤ tổng hóa đơn)"`
	Method    string  `json:"method"    dc:"CASH | TRANSFER | CARD"`
	Reference string  `json:"reference" dc:"Mã giao dịch"`
	Note      string  `json:"note"`
}

type RefundInput struct {
	Amount float64 `json:"amount"`
	Method string  `json:"method" dc:"CASH | TRANSFER | CARD"`
}

type SellInput struct {
	MemberID      uint   `json:"member_id"      v:"required|min:1#Vui lòng chọn hội viên|Vui lòng chọn hội viên"`
	PTPackageID   uint   `json:"pt_package_id"  v:"required|min:1#Vui lòng chọn gói PT|Vui lòng chọn gói PT"`
	TrainerID     uint   `json:"trainer_id"     v:"required|min:1#Vui lòng chọn huấn luyện viên|Vui lòng chọn huấn luyện viên"`
	BranchID      uint   `json:"branch_id"      v:"required|min:1#Vui lòng chọn chi nhánh|Vui lòng chọn chi nhánh"`
	StartDate     string `json:"start_date"     v:"date-format:Y-m-d#Ngày bắt đầu phải có dạng YYYY-MM-DD" dc:"Bỏ trống = hôm nay"`
	PromotionCode string `json:"promotion_code" v:"max-length:30#Mã khuyến mãi tối đa 30 ký tự"`
	Note          string `json:"note"           v:"max-length:500#Ghi chú tối đa 500 ký tự"`
}

type QuoteReq struct {
	g.Meta `path:"/pt-contracts/quote" method:"post" tags:"Huấn luyện cá nhân (PT)" summary:"Xem trước bán gói PT (không ghi DB)" perm:"pt.sell"`
	SellInput
}

type PromotionApplied struct {
	ID       uint    `json:"id"`
	Code     string  `json:"code"`
	Name     string  `json:"name"`
	Discount float64 `json:"discount"`
}

type QuoteRes struct {
	MemberID        uint              `json:"member_id"`
	BranchID        uint              `json:"branch_id"`
	TrainerID       uint              `json:"trainer_id"`
	TrainerName     string            `json:"trainer_name"`
	Package         PackageRef        `json:"package"`
	StartDate       string            `json:"start_date"`
	EndDate         string            `json:"end_date"`
	TotalSessions   int               `json:"total_sessions"`
	SessionMinutes  int               `json:"session_minutes"`
	Subtotal        float64           `json:"subtotal"`
	Discount        float64           `json:"discount"`
	Total           float64           `json:"total"`
	PricePerSession float64           `json:"price_per_session" dc:"= round(total / total_sessions)"`
	Promotion       *PromotionApplied `json:"promotion"`
}

type SellReq struct {
	g.Meta `path:"/pt-contracts" method:"post" tags:"Huấn luyện cá nhân (PT)" summary:"Bán gói PT (tạo hợp đồng + hóa đơn)" perm:"pt.sell"`
	SellInput
	Payment *PaymentInput `json:"payment" dc:"Thu tiền ngay (tùy chọn)"`
}
type SellRes = ContractDetail

type ContractListReq struct {
	g.Meta `path:"/pt-contracts" method:"get" tags:"Huấn luyện cá nhân (PT)" summary:"Danh sách hợp đồng PT" perm:"pt.view"`
	dto.PageReq
	BranchID    uint   `json:"branch_id"     in:"query"`
	TrainerID   uint   `json:"trainer_id"    in:"query"`
	MemberID    uint   `json:"member_id"     in:"query"`
	PTPackageID uint   `json:"pt_package_id" in:"query"`
	Status      string `json:"status"        in:"query" v:"in:PENDING,ACTIVE,COMPLETED,EXPIRED,CANCELLED#Trạng thái không hợp lệ"`
}
type ContractListRes = dto.PageRes[*Contract]

type ContractGetReq struct {
	g.Meta `path:"/pt-contracts/{id}" method:"get" tags:"Huấn luyện cá nhân (PT)" summary:"Chi tiết hợp đồng PT (kèm buổi tập, hóa đơn)" perm:"pt.view"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type ContractGetRes = ContractDetail

type MemberContractsReq struct {
	g.Meta `path:"/members/{id}/pt-contracts" method:"get" tags:"Huấn luyện cá nhân (PT)" summary:"Hợp đồng PT của hội viên" perm:"pt.view"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type MemberContractsRes []*Contract

type ChangeTrainerInput struct {
	TrainerID uint   `json:"trainer_id" v:"required|min:1#Vui lòng chọn huấn luyện viên mới|Vui lòng chọn huấn luyện viên mới"`
	Note      string `json:"note"       v:"max-length:255#Ghi chú tối đa 255 ký tự"`
}

// SessionConflict là một buổi sẽ bị trùng lịch nếu chuyển sang HLV mới.
type SessionConflict struct {
	SessionID uint      `json:"session_id"`
	StartAt   time.Time `json:"start_at"`
	EndAt     time.Time `json:"end_at"`
	Reason    string    `json:"reason" dc:"Lý do trùng: buổi PT khác / lớp nhóm"`
}

type ChangeTrainerCheckReq struct {
	g.Meta `path:"/pt-contracts/{id}/change-trainer/check" method:"post" tags:"Huấn luyện cá nhân (PT)" summary:"Kiểm tra trùng lịch trước khi đổi HLV (không ghi DB)" perm:"pt.sell"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
	ChangeTrainerInput
}

type ChangeTrainerCheckRes struct {
	OK            bool               `json:"ok"              dc:"true = đổi được"`
	TrainerID     uint               `json:"trainer_id"`
	TrainerName   string             `json:"trainer_name"`
	MovedSessions int                `json:"moved_sessions"  dc:"Số buổi SCHEDULED tương lai sẽ chuyển sang HLV mới"`
	Conflicts     []*SessionConflict `json:"conflicts"`
}

type ChangeTrainerReq struct {
	g.Meta `path:"/pt-contracts/{id}/change-trainer" method:"post" tags:"Huấn luyện cá nhân (PT)" summary:"Đổi HLV của hợp đồng" perm:"pt.sell"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
	ChangeTrainerInput
}
type ChangeTrainerRes = ContractDetail

type CancelContractInput struct {
	Reason string       `json:"reason" v:"required|max-length:255#Vui lòng nhập lý do hủy|Lý do tối đa 255 ký tự"`
	Refund *RefundInput `json:"refund" dc:"Hoàn tiền kèm theo (tùy chọn)"`
}

type CancelContractReq struct {
	g.Meta `path:"/pt-contracts/{id}/cancel" method:"post" tags:"Huấn luyện cá nhân (PT)" summary:"Hủy hợp đồng PT (+ hoàn tiền)" perm:"pt.cancel_contract"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
	CancelContractInput
}
type CancelContractRes = ContractDetail

// =================================================================================
// Buổi PT
// =================================================================================

type SessionListReq struct {
	g.Meta     `path:"/pt-sessions" method:"get" tags:"Huấn luyện cá nhân (PT)" summary:"Lịch buổi PT theo khoảng ngày" perm:"pt.view"`
	From       string `json:"from"        in:"query" v:"required|date-format:Y-m-d#Vui lòng chọn từ ngày|Từ ngày phải có dạng YYYY-MM-DD"`
	To         string `json:"to"          in:"query" v:"required|date-format:Y-m-d#Vui lòng chọn đến ngày|Đến ngày phải có dạng YYYY-MM-DD"`
	BranchID   uint   `json:"branch_id"   in:"query"`
	TrainerID  uint   `json:"trainer_id"  in:"query"`
	MemberID   uint   `json:"member_id"   in:"query"`
	ContractID uint   `json:"contract_id" in:"query"`
	Status     string `json:"status"      in:"query" v:"in:SCHEDULED,COMPLETED,CANCELLED,NO_SHOW#Trạng thái không hợp lệ"`
}
type SessionListRes []*Session

type SessionGetReq struct {
	g.Meta `path:"/pt-sessions/{id}" method:"get" tags:"Huấn luyện cá nhân (PT)" summary:"Chi tiết buổi PT" perm:"pt.view"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type SessionGetRes = Session

type SessionCreateReq struct {
	g.Meta     `path:"/pt-sessions" method:"post" tags:"Huấn luyện cá nhân (PT)" summary:"Đặt buổi PT" perm:"pt.schedule"`
	ContractID uint   `json:"contract_id" v:"required|min:1#Vui lòng chọn hợp đồng PT|Vui lòng chọn hợp đồng PT"`
	StartAt    string `json:"start_at"    v:"required#Vui lòng chọn giờ bắt đầu" dc:"RFC3339, hoặc 'YYYY-MM-DDTHH:mm' (giờ Việt Nam)"`
}
type SessionCreateRes = Session

type SessionUpdateReq struct {
	g.Meta  `path:"/pt-sessions/{id}" method:"put" tags:"Huấn luyện cá nhân (PT)" summary:"Đổi giờ buổi PT" perm:"pt.schedule"`
	ID      uint   `json:"id"       in:"path" v:"required|min:1"`
	StartAt string `json:"start_at" v:"required#Vui lòng chọn giờ bắt đầu" dc:"RFC3339, hoặc 'YYYY-MM-DDTHH:mm' (giờ Việt Nam)"`
}
type SessionUpdateRes = Session

type SessionCancelReq struct {
	g.Meta      `path:"/pt-sessions/{id}/cancel" method:"post" tags:"Huấn luyện cá nhân (PT)" summary:"Hủy buổi PT" perm:"pt.schedule"`
	ID          uint   `json:"id"            in:"path" v:"required|min:1"`
	Reason      string `json:"reason"        v:"required|max-length:255#Vui lòng nhập lý do hủy|Lý do tối đa 255 ký tự"`
	CountAsUsed bool   `json:"count_as_used" dc:"Chỉ có tác dụng khi hủy muộn: true = tính là đã dùng buổi (NO_SHOW)"`
}
type SessionCancelRes = Session

type SessionCompleteReq struct {
	g.Meta      `path:"/pt-sessions/{id}/complete" method:"post" tags:"Huấn luyện cá nhân (PT)" summary:"Xác nhận hoàn thành buổi PT (ghi giáo án, sinh hoa hồng)" perm:"pt.complete"`
	ID          uint   `json:"id"           in:"path" v:"required|min:1"`
	WorkoutNote string `json:"workout_note" v:"max-length:5000#Giáo án tối đa 5000 ký tự"`
	TrainerNote string `json:"trainer_note" v:"max-length:5000#Nhận xét tối đa 5000 ký tự"`
}
type SessionCompleteRes = Session

type SessionNoShowReq struct {
	g.Meta `path:"/pt-sessions/{id}/no-show" method:"post" tags:"Huấn luyện cá nhân (PT)" summary:"Đánh dấu hội viên vắng mặt (tính buổi, không hoa hồng)" perm:"pt.complete"`
	ID     uint   `json:"id"   in:"path" v:"required|min:1"`
	Note   string `json:"note" v:"max-length:255#Ghi chú tối đa 255 ký tự"`
}
type SessionNoShowRes = Session

// ScheduleItem là một mục trong lịch dạy của HLV (buổi PT hoặc buổi lớp nhóm).
type ScheduleItem struct {
	Type        string    `json:"type"         dc:"PT | CLASS"`
	ID          uint      `json:"id"           dc:"pt_sessions.id hoặc class_schedules.id"`
	Title       string    `json:"title"        dc:"PT: tên hội viên; CLASS: tên loại lớp"`
	StartAt     time.Time `json:"start_at"`
	EndAt       time.Time `json:"end_at"`
	Status      string    `json:"status"`
	BranchID    uint      `json:"branch_id"`
	BranchName  string    `json:"branch_name"`
	TrainerID   uint      `json:"trainer_id"`
	TrainerName string    `json:"trainer_name"`
	ContractID  *uint     `json:"contract_id"  dc:"Chỉ PT"`
	MemberID    *uint     `json:"member_id"    dc:"Chỉ PT"`
	RoomName    string    `json:"room_name"    dc:"Chỉ CLASS"`
	Color       string    `json:"color"        dc:"Chỉ CLASS: màu loại lớp"`
	BookedCount *int      `json:"booked_count" dc:"Chỉ CLASS"`
	Capacity    *int      `json:"capacity"     dc:"Chỉ CLASS"`
}

type TrainerScheduleReq struct {
	g.Meta    `path:"/trainer-schedule" method:"get" tags:"Huấn luyện cá nhân (PT)" summary:"Lịch dạy của HLV (buổi PT + lớp nhóm)" perm:"pt.view"`
	TrainerID uint   `json:"trainer_id" in:"query" dc:"Bỏ trống = chính mình"`
	From      string `json:"from"       in:"query" v:"required|date-format:Y-m-d#Vui lòng chọn từ ngày|Từ ngày phải có dạng YYYY-MM-DD"`
	To        string `json:"to"         in:"query" v:"required|date-format:Y-m-d#Vui lòng chọn đến ngày|Đến ngày phải có dạng YYYY-MM-DD"`
}
type TrainerScheduleRes []*ScheduleItem

// =================================================================================
// Hoa hồng
// =================================================================================

type CommissionListReq struct {
	g.Meta `path:"/commissions" method:"get" tags:"Huấn luyện cá nhân (PT)" summary:"Danh sách hoa hồng HLV" perm:"commission.view"`
	dto.PageReq
	TrainerID  uint   `json:"trainer_id"  in:"query"`
	BranchID   uint   `json:"branch_id"   in:"query"`
	Period     string `json:"period"      in:"query" v:"regex:^\\d{4}-\\d{2}$#Kỳ phải có dạng YYYY-MM"`
	Status     string `json:"status"      in:"query" v:"in:PENDING,PAID#Trạng thái không hợp lệ"`
	SourceType string `json:"source_type" in:"query" v:"in:PT_SESSION,CLASS#Nguồn không hợp lệ"`
}
type CommissionListRes = dto.PageRes[*Commission]

type CommissionSummaryReq struct {
	g.Meta   `path:"/commissions/summary" method:"get" tags:"Huấn luyện cá nhân (PT)" summary:"Tổng hợp hoa hồng theo HLV trong kỳ" perm:"commission.view"`
	Period   string `json:"period"    in:"query" v:"required|regex:^\\d{4}-\\d{2}$#Vui lòng chọn kỳ|Kỳ phải có dạng YYYY-MM"`
	BranchID uint   `json:"branch_id" in:"query"`
}

type CommissionSummaryItem struct {
	TrainerID     uint    `json:"trainer_id"`
	TrainerName   string  `json:"trainer_name"`
	StaffCode     string  `json:"staff_code"`
	Sessions      int     `json:"sessions"       dc:"Số buổi PT / lớp sinh hoa hồng"`
	BaseTotal     float64 `json:"base_total"     dc:"Tổng doanh thu buổi (Σ base_amount)"`
	TotalAmount   float64 `json:"total_amount"`
	PaidAmount    float64 `json:"paid_amount"`
	PendingAmount float64 `json:"pending_amount"`
}

type CommissionSummaryRes struct {
	Period        string                   `json:"period"`
	BranchID      uint                     `json:"branch_id"`
	Items         []*CommissionSummaryItem `json:"items"`
	Sessions      int                      `json:"sessions"`
	TotalAmount   float64                  `json:"total_amount"`
	PaidAmount    float64                  `json:"paid_amount"`
	PendingAmount float64                  `json:"pending_amount"`
}

type CommissionPayReq struct {
	g.Meta    `path:"/commissions/pay" method:"post" tags:"Huấn luyện cá nhân (PT)" summary:"Chi trả hoa hồng của HLV trong kỳ" perm:"commission.pay"`
	TrainerID uint   `json:"trainer_id" v:"required|min:1#Vui lòng chọn huấn luyện viên|Vui lòng chọn huấn luyện viên"`
	Period    string `json:"period"     v:"required|regex:^\\d{4}-\\d{2}$#Vui lòng chọn kỳ|Kỳ phải có dạng YYYY-MM"`
	BranchID  uint   `json:"branch_id"  dc:"Tùy chọn: chỉ chi trả hoa hồng phát sinh tại chi nhánh này"`
}

type CommissionPayRes struct {
	TrainerID   uint      `json:"trainer_id"`
	TrainerName string    `json:"trainer_name"`
	Period      string    `json:"period"`
	Count       int       `json:"count"  dc:"Số dòng hoa hồng đã chuyển PAID"`
	Amount      float64   `json:"amount" dc:"Tổng tiền đã chi"`
	PaidAt      time.Time `json:"paid_at"`
}
