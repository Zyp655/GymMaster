package v1

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"

	"gymmaster-api/internal/dto"
)

type ContractTemplateItem struct {
	ID        uint      `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	BodyHTML  string    `json:"body_html"`
	Version   int       `json:"version"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ContractTemplateInput struct {
	Code     string `json:"code"      v:"required|max-length:50#Vui lòng nhập mã mẫu hợp đồng"`
	Name     string `json:"name"      v:"required|max-length:150#Vui lòng nhập tên mẫu hợp đồng"`
	BodyHTML string `json:"body_html" v:"required#Vui lòng nhập nội dung HTML hợp đồng"`
	Version  int    `json:"version"   d:"1" v:"min:1"`
	Status   string `json:"status"    d:"ACTIVE" v:"in:ACTIVE,INACTIVE"`
}

type MemberContractItem struct {
	ID              uint       `json:"id"`
	MemberID        uint       `json:"member_id"`
	MemberCode      string     `json:"member_code,omitempty"`
	MemberName      string     `json:"member_name,omitempty"`
	MemberPhone     string     `json:"member_phone,omitempty"`
	TemplateID      uint       `json:"template_id"`
	TemplateCode    string     `json:"template_code,omitempty"`
	TemplateName    string     `json:"template_name,omitempty"`
	TemplateVersion int        `json:"template_version"`
	SubscriptionID  *uint      `json:"subscription_id"`
	RenderedHTML    string     `json:"rendered_html"`
	Status          string     `json:"status"`
	SignedAt        *time.Time `json:"signed_at"`
	SignatureURL    string     `json:"signature_url"`
	PDFURL          string     `json:"pdf_url"`
	IP              string     `json:"ip"`
	UserAgent       string     `json:"user_agent"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type DayPassItem struct {
	ID              uint       `json:"id"`
	BranchID        uint       `json:"branch_id"`
	BranchName      string     `json:"branch_name,omitempty"`
	BuyerName       string     `json:"buyer_name"`
	Phone           string     `json:"phone"`
	MemberID        *uint      `json:"member_id"`
	SponsorMemberID *uint      `json:"sponsor_member_id"`
	SponsorName     string     `json:"sponsor_name,omitempty"`
	Type            string     `json:"type"`
	ValidDate       string     `json:"valid_date"`
	QRToken         string     `json:"qr_token"`
	Price           float64    `json:"price"`
	InvoiceID       *uint      `json:"invoice_id"`
	UsedAt          *time.Time `json:"used_at"`
	CreatedBy       *uint      `json:"created_by"`
	CreatedAt       time.Time  `json:"created_at"`
}

// ---- Admin (/v1/admin) ----------------------------------------------------------

type ListContractTemplatesReq struct {
	g.Meta `path:"/contract-templates" method:"get" tags:"Hợp đồng & Vé ngày" summary:"Danh sách mẫu hợp đồng điện tử" perm:"contract.manage"`
	dto.PageReq
	Status string `json:"status" in:"query"`
}
type ListContractTemplatesRes = dto.PageRes[*ContractTemplateItem]

type CreateContractTemplateReq struct {
	g.Meta `path:"/contract-templates" method:"post" tags:"Hợp đồng & Vé ngày" summary:"Tạo mẫu hợp đồng điện tử" perm:"contract.manage"`
	ContractTemplateInput
}
type CreateContractTemplateRes = ContractTemplateItem

type UpdateContractTemplateReq struct {
	g.Meta `path:"/contract-templates/{id}" method:"put" tags:"Hợp đồng & Vé ngày" summary:"Cập nhật mẫu hợp đồng điện tử" perm:"contract.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
	ContractTemplateInput
}
type UpdateContractTemplateRes = ContractTemplateItem

type DeleteContractTemplateReq struct {
	g.Meta `path:"/contract-templates/{id}" method:"delete" tags:"Hợp đồng & Vé ngày" summary:"Xóa mẫu hợp đồng điện tử" perm:"contract.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type DeleteContractPassRes struct{}

type ListMemberContractsReq struct {
	g.Meta   `path:"/member-contracts" method:"get" tags:"Hợp đồng & Vé ngày" summary:"Danh sách hợp đồng điện tử của hội viên" perm:"contract.manage,member.view"`
	dto.PageReq
	MemberID uint   `json:"member_id" in:"query"`
	Status   string `json:"status"    in:"query"`
}
type ListMemberContractsRes = dto.PageRes[*MemberContractItem]

type CreateMemberContractReq struct {
	g.Meta         `path:"/member-contracts" method:"post" tags:"Hợp đồng & Vé ngày" summary:"Khởi tạo hợp đồng điện tử cho hội viên" perm:"contract.manage"`
	MemberID       uint  `json:"member_id"       v:"required|min:1#Vui lòng chọn hội viên"`
	TemplateID     uint  `json:"template_id"`
	SubscriptionID *uint `json:"subscription_id"`
}
type CreateMemberContractRes = MemberContractItem

type ListDayPassesReq struct {
	g.Meta   `path:"/day-passes" method:"get" tags:"Hợp đồng & Vé ngày" summary:"Danh sách vé ngày & vé khách" perm:"contract.manage,checkin.view"`
	dto.PageReq
	BranchID uint   `json:"branch_id" in:"query"`
	Type     string `json:"type"      in:"query"`
}
type ListDayPassesRes = dto.PageRes[*DayPassItem]

type CreateDayPassReq struct {
	g.Meta    `path:"/day-passes" method:"post" tags:"Hợp đồng & Vé ngày" summary:"Bán vé tập theo ngày (Day Pass)" perm:"contract.manage,checkin.scan"`
	BranchID  uint    `json:"branch_id"  v:"required|min:1#Vui lòng chọn chi nhánh"`
	BuyerName string  `json:"buyer_name" v:"required|max-length:150#Vui lòng nhập họ tên khách"`
	Phone     string  `json:"phone"      v:"required|max-length:20#Vui lòng nhập số điện thoại"`
	Price     float64 `json:"price"      d:"100000" v:"min:0"`
	ValidDate string  `json:"valid_date"`
}
type CreateDayPassRes = DayPassItem

// ---- Member (/v1/member) --------------------------------------------------------

type MemberContractsReq struct {
	g.Meta `path:"/contracts" method:"get" tags:"Cổng hội viên" summary:"Hợp đồng điện tử của tôi" role:"MEMBER"`
}
type MemberContractsRes []*MemberContractItem

type SignMemberContractReq struct {
	g.Meta        `path:"/contracts/{id}/sign" method:"post" tags:"Cổng hội viên" summary:"Ký hợp đồng điện tử" role:"MEMBER"`
	ID            uint   `json:"id"             in:"path" v:"required|min:1"`
	SignatureData string `json:"signature_data"           v:"required#Vui lòng ký tên xác nhận"`
}
type SignMemberContractRes = MemberContractItem

type MemberGuestPassesReq struct {
	g.Meta `path:"/guest-passes" method:"get" tags:"Cổng hội viên" summary:"Danh sách vé mời bạn bè (Guest Pass)" role:"MEMBER"`
}
type MemberGuestPassesRes struct {
	Quota         int            `json:"quota"`
	UsedThisMonth int            `json:"used_this_month"`
	Remaining     int            `json:"remaining"`
	Passes        []*DayPassItem `json:"passes"`
	Items         []*DayPassItem `json:"items"`
}

type CreateGuestPassReq struct {
	g.Meta    `path:"/guest-passes" method:"post" tags:"Cổng hội viên" summary:"Tạo vé mời bạn bè đi tập cùng" role:"MEMBER"`
	BuyerName string `json:"buyer_name" v:"required|max-length:150#Vui lòng nhập họ tên bạn bè"`
	Phone     string `json:"phone"      v:"required|max-length:20#Vui lòng nhập số điện thoại bạn bè"`
	ValidDate string `json:"valid_date"`
	BranchID  uint   `json:"branch_id"`
}
type CreateGuestPassRes = DayPassItem
