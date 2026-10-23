package v1

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

// ==================== PUBLIC LANDING / MARKETING (SPRINT 29) ====================

type PublicTrainerItem struct {
	ID              uint   `json:"id"`
	UserID          uint   `json:"user_id"`
	FullName        string `json:"full_name"`
	Slug            string `json:"slug"`
	Headline        string `json:"headline"`
	Bio             string `json:"bio"`
	Specialties     string `json:"specialties"`
	Certifications  string `json:"certifications"`
	ExperienceYears int    `json:"experience_years"`
	AvatarURL       string `json:"avatar_url"`
	Featured        bool   `json:"featured"`
	isPublished     bool
	IsPublished     bool    `json:"is_published"`
	RatingAvg       float64 `json:"rating_avg"`
	RatingCount     int     `json:"rating_count"`
}

type PublicTrainersReq struct {
	g.Meta `path:"/trainers" method:"get" tags:"Công khai – Nội dung" summary:"Danh sách hồ sơ HLV công khai"`
}

type PublicTrainersRes []*PublicTrainerItem

type PublicTrainerDetailReq struct {
	g.Meta `path:"/trainers/{slug}" method:"get" tags:"Công khai – Nội dung" summary:"Chi tiết hồ sơ HLV công khai theo slug"`
	Slug   string `json:"slug" in:"path" v:"required"`
}

type PublicTrainerDetailRes struct {
	*PublicTrainerItem
}

type PublicClassTypeItem struct {
	ID              uint   `json:"id"`
	Code            string `json:"code"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	DefaultDuration int    `json:"default_duration"`
	DefaultCapacity int    `json:"default_capacity"`
	Color           string `json:"color"`
}

type PublicClassScheduleBrief struct {
	ID          uint      `json:"id"`
	BranchID    uint      `json:"branch_id"`
	BranchName  string    `json:"branch_name"`
	RoomName    string    `json:"room_name"`
	TrainerName string    `json:"trainer_name"`
	StartAt     time.Time `json:"start_at"`
	EndAt       time.Time `json:"end_at"`
	Capacity    int       `json:"capacity"`
	BookedCount int       `json:"booked_count"`
}

type PublicClassTypesReq struct {
	g.Meta `path:"/class-types" method:"get" tags:"Công khai – Nội dung" summary:"Danh sách bộ môn lớp học công khai"`
}

type PublicClassTypesRes []*PublicClassTypeItem

type PublicClassTypeDetailReq struct {
	g.Meta `path:"/class-types/{id}" method:"get" tags:"Công khai – Nội dung" summary:"Chi tiết bộ môn lớp học và lịch sắp tới"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}

type PublicClassTypeDetailRes struct {
	ClassType *PublicClassTypeItem        `json:"class_type"`
	Schedules []*PublicClassScheduleBrief `json:"schedules"`
}

type TestimonialItem struct {
	ID              uint       `json:"id"`
	MemberID        *uint      `json:"member_id,omitempty"`
	AuthorName      string     `json:"author_name"`
	AuthorRole      string     `json:"author_role"`
	AvatarURL       string     `json:"avatar_url"`
	Rating          int        `json:"rating"`
	Content         string     `json:"content"`
	BeforeAfterNote string     `json:"before_after_note"`
	Status          string     `json:"status"`
	ApprovedAt      *time.Time `json:"approved_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

type PublicTestimonialsReq struct {
	g.Meta `path:"/testimonials" method:"get" tags:"Công khai – Nội dung" summary:"Danh sách cảm nhận hội viên đã duyệt"`
}

type PublicTestimonialsRes []*TestimonialItem

type PublicSubmitTestimonialReq struct {
	g.Meta          `path:"/testimonials" method:"post" tags:"Công khai – Nội dung" summary:"Gửi cảm nhận hội viên"`
	AuthorName      string `json:"author_name" v:"required|length:2,150"`
	AuthorRole      string `json:"author_role"`
	Rating          int    `json:"rating" d:"5" v:"between:1,5"`
	Content         string `json:"content" v:"required|length:10,2000"`
	BeforeAfterNote string `json:"before_after_note"`
}

type PublicSubmitTestimonialRes struct {
	*TestimonialItem
}

type PostItem struct {
	ID          uint       `json:"id"`
	Slug        string     `json:"slug"`
	Title       string     `json:"title"`
	Excerpt     string     `json:"excerpt"`
	CoverURL    string     `json:"cover_url"`
	Category    string     `json:"category"`
	Content     string     `json:"content"`
	Status      string     `json:"status"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type PublicPostsReq struct {
	g.Meta   `path:"/posts" method:"get" tags:"Công khai – Nội dung" summary:"Danh sách bài viết Blog/Tin tức đã xuất bản"`
	Category string `json:"category" in:"query"`
	Search   string `json:"search" in:"query"`
}

type PublicPostsRes []*PostItem

type PublicPostDetailReq struct {
	g.Meta `path:"/posts/{slug}" method:"get" tags:"Công khai – Nội dung" summary:"Chi tiết bài viết Blog theo slug"`
	Slug   string `json:"slug" in:"path" v:"required"`
}

type PublicPostDetailRes struct {
	*PostItem
}

type ContactMessageItem struct {
	ID          uint      `json:"id"`
	BranchID    *uint     `json:"branch_id,omitempty"`
	FullName    string    `json:"full_name"`
	Phone       string    `json:"phone"`
	Email       string    `json:"email"`
	Subject     string    `json:"subject"`
	Message     string    `json:"message"`
	Status      string    `json:"status"`
	HandledNote string    `json:"handled_note"`
	CreatedAt   time.Time `json:"created_at"`
}

type PublicContactMessageReq struct {
	g.Meta   `path:"/contact-messages" method:"post" tags:"Công khai – Nội dung" summary:"Gửi lời nhắn liên hệ từ trang chủ"`
	BranchID *uint  `json:"branch_id"`
	FullName string `json:"full_name" v:"required|length:2,150"`
	Phone    string `json:"phone" v:"required|length:9,25"`
	Email    string `json:"email"`
	Subject  string `json:"subject"`
	Message  string `json:"message" v:"required|length:5,2000"`
}

type PublicContactMessageRes struct {
	*ContactMessageItem
}

// ==================== ADMIN CMS MANAGEMENT ====================

type AdminListTrainerProfilesReq struct {
	g.Meta `path:"/trainer-profiles" method:"get" perm:"cms.manage,staff.view" tags:"Quản trị – Nội dung" summary:"Danh sách hồ sơ HLV công khai"`
}

type AdminListTrainerProfilesRes []*PublicTrainerItem

type AdminSaveTrainerProfileReq struct {
	g.Meta          `path:"/trainer-profiles" method:"post" perm:"cms.manage" tags:"Quản trị – Nội dung" summary:"Tạo/Cập nhật hồ sơ HLV công khai"`
	UserID          uint   `json:"user_id" v:"required|min:1"`
	Slug            string `json:"slug" v:"required"`
	Headline        string `json:"headline"`
	Bio             string `json:"bio"`
	Specialties     string `json:"specialties"`
	Certifications  string `json:"certifications"`
	ExperienceYears int    `json:"experience_years" d:"2"`
	AvatarURL       string `json:"avatar_url"`
	Featured        bool   `json:"featured"`
	IsPublished     bool   `json:"is_published" d:"true"`
}

type AdminSaveTrainerProfileRes struct {
	*PublicTrainerItem
}

type AdminDeleteTrainerProfileReq struct {
	g.Meta `path:"/trainer-profiles/{id}" method:"delete" perm:"cms.manage" tags:"Quản trị – Nội dung" summary:"Xóa hồ sơ HLV công khai"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}

type AdminDeleteTrainerProfileRes struct{}

type AdminListTestimonialsReq struct {
	g.Meta `path:"/testimonials" method:"get" perm:"cms.manage" tags:"Quản trị – Nội dung" summary:"Danh sách cảm nhận hội viên"`
	Status string `json:"status" in:"query"`
}

type AdminListTestimonialsRes []*TestimonialItem

type AdminApproveTestimonialReq struct {
	g.Meta `path:"/testimonials/{id}/approve" method:"post" perm:"cms.manage" tags:"Quản trị – Nội dung" summary:"Duyệt cảm nhận hội viên"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}

type AdminApproveTestimonialRes struct {
	*TestimonialItem
}

type AdminRejectTestimonialReq struct {
	g.Meta `path:"/testimonials/{id}/reject" method:"post" perm:"cms.manage" tags:"Quản trị – Nội dung" summary:"Từ chối cảm nhận hội viên"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}

type AdminRejectTestimonialRes struct {
	*TestimonialItem
}

type AdminDeleteTestimonialReq struct {
	g.Meta `path:"/testimonials/{id}" method:"delete" perm:"cms.manage" tags:"Quản trị – Nội dung" summary:"Xóa cảm nhận hội viên"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}

type AdminDeleteTestimonialRes struct{}

type AdminListPostsReq struct {
	g.Meta   `path:"/posts" method:"get" perm:"cms.manage" tags:"Quản trị – Nội dung" summary:"Danh sách bài viết Blog"`
	Status   string `json:"status" in:"query"`
	Category string `json:"category" in:"query"`
}

type AdminListPostsRes []*PostItem

type AdminCreatePostReq struct {
	g.Meta   `path:"/posts" method:"post" perm:"cms.manage" tags:"Quản trị – Nội dung" summary:"Tạo bài viết Blog mới"`
	Slug     string `json:"slug" v:"required|length:3,180|regex:^[a-z0-9]+(-[a-z0-9]+)*$#Vui lòng nhập đường dẫn bài viết|Đường dẫn bài viết phải từ 3 đến 180 ký tự|Đường dẫn bài viết chỉ gồm chữ thường không dấu (a-z), số và dấu gạch ngang '-'"`
	Title    string `json:"title" v:"required|max-length:255#Vui lòng nhập tiêu đề bài viết|Tiêu đề tối đa 255 ký tự"`
	Excerpt  string `json:"excerpt"`
	CoverURL string `json:"cover_url" v:"max-length:500#Ảnh bìa tối đa 500 ký tự"`
	Category string `json:"category" d:"TAP_LUYEN" v:"in:TAP_LUYEN,DINH_DUONG,SU_KIEN#Chuyên mục bài viết không hợp lệ"`
	Content  string `json:"content" v:"required#Vui lòng nhập nội dung bài viết"`
	Status   string `json:"status" d:"PUBLISHED" v:"in:DRAFT,PUBLISHED,ARCHIVED#Trạng thái bài viết không hợp lệ"`
}

type AdminCreatePostRes struct {
	*PostItem
}

type AdminUpdatePostReq struct {
	g.Meta   `path:"/posts/{id}" method:"put" perm:"cms.manage" tags:"Quản trị – Nội dung" summary:"Cập nhật bài viết Blog"`
	ID       uint   `json:"id" in:"path" v:"required|min:1"`
	Slug     string `json:"slug" v:"length:3,180|regex:^[a-z0-9]+(-[a-z0-9]+)*$#Đường dẫn bài viết phải từ 3 đến 180 ký tự|Đường dẫn bài viết chỉ gồm chữ thường không dấu (a-z), số và dấu gạch ngang '-'"`
	Title    string `json:"title" v:"required|max-length:255#Vui lòng nhập tiêu đề bài viết|Tiêu đề tối đa 255 ký tự"`
	Excerpt  string `json:"excerpt"`
	CoverURL string `json:"cover_url" v:"max-length:500#Ảnh bìa tối đa 500 ký tự"`
	Category string `json:"category" v:"in:TAP_LUYEN,DINH_DUONG,SU_KIEN#Chuyên mục bài viết không hợp lệ"`
	Content  string `json:"content"`
	Status   string `json:"status" v:"in:DRAFT,PUBLISHED,ARCHIVED#Trạng thái bài viết không hợp lệ"`
}

type AdminUpdatePostRes struct {
	*PostItem
}

type AdminDeletePostReq struct {
	g.Meta `path:"/posts/{id}" method:"delete" perm:"cms.manage" tags:"Quản trị – Nội dung" summary:"Xóa bài viết Blog"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}

type AdminDeletePostRes struct{}

type AdminListContactMessagesReq struct {
	g.Meta `path:"/contact-messages" method:"get" perm:"cms.manage,lead.view" tags:"Quản trị – Nội dung" summary:"Danh sách lời nhắn liên hệ"`
	Status string `json:"status" in:"query"`
}

type AdminListContactMessagesRes []*ContactMessageItem

type AdminUpdateContactMessageReq struct {
	g.Meta      `path:"/contact-messages/{id}" method:"put" perm:"cms.manage,lead.manage" tags:"Quản trị – Nội dung" summary:"Cập nhật xử lý lời nhắn liên hệ"`
	ID          uint   `json:"id" in:"path" v:"required|min:1"`
	Status      string `json:"status" v:"required|in:NEW,CONTACTED,CLOSED"`
	HandledNote string `json:"handled_note"`
}

type AdminUpdateContactMessageRes struct {
	*ContactMessageItem
}

// ==================== NHẮN TIN HLV ↔ HỘI VIÊN, TÙY CHỌN THÔNG BÁO & LỊCH .ICS (MỤC 15.1) ====================

type ConversationItem struct {
	ID                 uint       `json:"id"`
	PTContractID       uint       `json:"pt_contract_id"`
	MemberID           uint       `json:"member_id"`
	MemberName         string     `json:"member_name"`
	MemberCode         string     `json:"member_code"`
	TrainerID          uint       `json:"trainer_id"`
	TrainerName        string     `json:"trainer_name"`
	ContractStatus     string     `json:"contract_status"`
	RemainingSessions  int        `json:"remaining_sessions"`
	LastMessageAt      *time.Time `json:"last_message_at,omitempty"`
	LastMessagePreview string     `json:"last_message_preview"`
	UnreadCount        int64      `json:"unread_count"`
}

type MessageItem struct {
	ID             uint       `json:"id"`
	ConversationID uint       `json:"conversation_id"`
	SenderUserID   uint       `json:"sender_user_id"`
	SenderName     string     `json:"sender_name"`
	SenderRole     string     `json:"sender_role"`
	Content        string     `json:"content"`
	AttachmentURL  string     `json:"attachment_url,omitempty"`
	ReadAt         *time.Time `json:"read_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

type ListConversationsReq struct {
	g.Meta `path:"/conversations" method:"get" tags:"Tin nhắn PT & Lịch" summary:"Danh sách hội thoại HLV – Hội viên (tin nhắn mới nhất trước)"`
	Page   int `json:"page"  in:"query" v:"min:0" dc:"Trang (mặc định 1)"`
	Limit  int `json:"limit" in:"query" v:"min:0|max:100" dc:"Số hội thoại mỗi trang (mặc định 50, tối đa 100)"`
}

type ListConversationsRes []*ConversationItem

type ListMessagesReq struct {
	g.Meta `path:"/conversations/{id}/messages" method:"get" tags:"Tin nhắn PT & Lịch" summary:"Danh sách tin nhắn trong hội thoại"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}

type ListMessagesRes []*MessageItem

type SendMessageReq struct {
	g.Meta        `path:"/conversations/{id}/messages" method:"post" tags:"Tin nhắn PT & Lịch" summary:"Gửi tin nhắn trong hợp đồng PT"`
	ID            uint   `json:"id" in:"path" v:"required|min:1"`
	Content       string `json:"content" v:"required|length:1,2000#Vui lòng nhập nội dung tin nhắn|Tin nhắn tối đa 2000 ký tự"`
	AttachmentURL string `json:"attachment_url"`
}

type SendMessageRes struct {
	*MessageItem
}

type NotificationPrefItem struct {
	Type      string `json:"type"`
	Label     string `json:"label"`
	Channel   string `json:"channel"`
	Enabled   bool   `json:"enabled"`
	Mandatory bool   `json:"mandatory"`
}

type GetNotificationPreferencesReq struct {
	g.Meta `path:"/notification-preferences" method:"get" tags:"Tin nhắn PT & Lịch" summary:"Xem tùy chọn thông báo"`
}

type GetNotificationPreferencesRes []*NotificationPrefItem

type SaveNotificationPreferenceInput struct {
	Type    string `json:"type" v:"required"`
	Channel string `json:"channel" v:"required|in:IN_APP,EMAIL"`
	Enabled bool   `json:"enabled"`
}

type SaveNotificationPreferencesReq struct {
	g.Meta      `path:"/notification-preferences" method:"put" tags:"Tin nhắn PT & Lịch" summary:"Lưu tùy chọn thông báo"`
	Preferences []SaveNotificationPreferenceInput `json:"preferences" v:"required"`
}

type SaveNotificationPreferencesRes []*NotificationPrefItem

type CalendarTokenRes struct {
	Token  string `json:"token"`
	ICSURL string `json:"ics_url"`
}

type GetCalendarTokenReq struct {
	g.Meta `path:"/calendar-token" method:"get" tags:"Tin nhắn PT & Lịch" summary:"Lấy token đồng bộ lịch .ics"`
}

type GetCalendarTokenRes struct {
	*CalendarTokenRes
}

type RotateCalendarTokenReq struct {
	g.Meta `path:"/calendar-token/rotate" method:"post" tags:"Tin nhắn PT & Lịch" summary:"Đổi token đồng bộ lịch .ics"`
}

type RotateCalendarTokenRes struct {
	*CalendarTokenRes
}
