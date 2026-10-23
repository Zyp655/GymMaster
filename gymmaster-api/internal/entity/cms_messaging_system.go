package entity

import (
	"time"

	"gorm.io/datatypes"
)

// Trạng thái bài viết CMS, cảm nhận hội viên, tin nhắn liên hệ, chuyển chi nhánh & import.
const (
	PostStatusDraft     = "DRAFT"
	PostStatusPublished = "PUBLISHED"
	PostStatusArchived  = "ARCHIVED"

	TestimonialPending  = "PENDING"
	TestimonialApproved = "APPROVED"
	TestimonialRejected = "REJECTED"

	ContactStatusNew       = "NEW"
	ContactStatusContacted = "CONTACTED"
	ContactStatusClosed    = "CLOSED"

	NotifyChannelInApp = "IN_APP"
	NotifyChannelEmail = "EMAIL"

	ImportModeDryRun = "DRY_RUN"
	ImportModeCommit = "COMMIT"
)

// TrainerPublicProfile là hồ sơ HLV công khai trên trang chủ / landing page.
type TrainerPublicProfile struct {
	ID              uint   `gorm:"primaryKey;autoIncrement"`
	UserID          uint   `gorm:"not null;unique"`
	Slug            string `gorm:"type:varchar(120);not null;unique"`
	Headline        string `gorm:"type:varchar(255);not null;default:''"`
	Bio             string `gorm:"type:text;not null;default:''"`
	Specialties     string `gorm:"type:text;not null;default:''"`
	Certifications  string `gorm:"type:text;not null;default:''"`
	ExperienceYears int    `gorm:"not null;default:1"`
	AvatarURL       string `gorm:"column:avatar_url;type:varchar(500);not null;default:''"`
	Featured        bool   `gorm:"not null;default:false"`
	IsPublished     bool   `gorm:"not null;default:true"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (TrainerPublicProfile) TableName() string { return "trainer_public_profiles" }

// Testimonial là cảm nhận hội viên hiển thị trên landing page (sau khi admin duyệt).
type Testimonial struct {
	ID              uint `gorm:"primaryKey;autoIncrement"`
	MemberID        *uint
	AuthorName      string `gorm:"type:varchar(150);not null"`
	AuthorRole      string `gorm:"type:varchar(120);not null;default:''"`
	AvatarURL       string `gorm:"column:avatar_url;type:varchar(500);not null;default:''"`
	Rating          int    `gorm:"not null;default:5"`
	Content         string `gorm:"type:text;not null"`
	BeforeAfterNote string `gorm:"type:varchar(255);not null;default:''"`
	Status          string `gorm:"type:varchar(20);not null;default:'PENDING'"`
	ApprovedBy      *uint
	ApprovedAt      *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (Testimonial) TableName() string { return "testimonials" }

// Post là bài viết tin tức / kiến thức tập luyện (CMS Blog).
type Post struct {
	ID          uint   `gorm:"primaryKey;autoIncrement"`
	Slug        string `gorm:"type:varchar(180);not null;unique"`
	Title       string `gorm:"type:varchar(255);not null"`
	Excerpt     string `gorm:"type:text;not null;default:''"`
	CoverURL    string `gorm:"column:cover_url;type:varchar(500);not null;default:''"`
	Category    string `gorm:"type:varchar(60);not null;default:'TAP_LUYEN'"`
	Content     string `gorm:"type:text;not null"`
	Status      string `gorm:"type:varchar(20);not null;default:'DRAFT'"`
	AuthorID    *uint
	PublishedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (Post) TableName() string { return "posts" }

// ContactMessage là lời nhắn liên hệ gửi từ trang chủ.
type ContactMessage struct {
	ID          uint `gorm:"primaryKey;autoIncrement"`
	BranchID    *uint
	FullName    string `gorm:"type:varchar(150);not null"`
	Phone       string `gorm:"type:varchar(30);not null"`
	Email       string `gorm:"type:varchar(150);not null;default:''"`
	Subject     string `gorm:"type:varchar(255);not null;default:''"`
	Message     string `gorm:"type:text;not null"`
	Status      string `gorm:"type:varchar(20);not null;default:'NEW'"`
	HandledBy   *uint
	HandledNote string `gorm:"type:text;not null;default:''"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (ContactMessage) TableName() string { return "contact_messages" }

// Conversation là cuộc trò chuyện giữa HLV và hội viên gắn với một hợp đồng PT.
type Conversation struct {
	ID           uint `gorm:"primaryKey;autoIncrement"`
	PTContractID uint `gorm:"column:pt_contract_id;not null;unique"`
	MemberID     uint `gorm:"not null"`
	TrainerID    uint `gorm:"not null"`
	// TrainerJoinedAt: thời điểm HLV hiện tại tham gia hội thoại (đổi HLV hợp đồng); NULL = tham gia từ đầu.
	// Tin hội viên gửi trước mốc này (gửi HLV cũ) không tính "chưa đọc" cho HLV mới (C11).
	TrainerJoinedAt    *time.Time
	LastMessageAt      *time.Time
	LastMessagePreview string `gorm:"type:varchar(255);not null;default:''"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (Conversation) TableName() string { return "conversations" }

// Message là tin nhắn trong cuộc trò chuyện HLV ↔ Hội viên.
type Message struct {
	ID             uint   `gorm:"primaryKey;autoIncrement"`
	ConversationID uint   `gorm:"not null"`
	SenderUserID   uint   `gorm:"not null"`
	SenderRole     string `gorm:"type:varchar(20);not null;default:'MEMBER'"`
	Content        string `gorm:"type:text;not null"`
	AttachmentURL  string `gorm:"column:attachment_url;type:varchar(500);not null;default:''"`
	ReadAt         *time.Time
	CreatedAt      time.Time
}

func (Message) TableName() string { return "messages" }

// NotificationPreference là tùy chọn bật/tắt thông báo theo loại & kênh của người dùng.
type NotificationPreference struct {
	ID      uint   `gorm:"primaryKey;autoIncrement"`
	UserID  uint   `gorm:"not null"`
	Type    string `gorm:"type:varchar(50);not null"`
	Channel string `gorm:"type:varchar(20);not null;default:'IN_APP'"`
	// Không đặt default:true ở tag: GORM sẽ bỏ qua giá trị false (zero value) khi Create/Upsert
	// và DB dùng mặc định true → không tắt được thông báo (NT-01). Mặc định true đã có ở migration.
	Enabled   bool `gorm:"not null"`
	UpdatedAt time.Time
}

func (NotificationPreference) TableName() string { return "notification_preferences" }

// MemberBranchTransfer là lịch sử chuyển chi nhánh sinh hoạt chính của hội viên.
type MemberBranchTransfer struct {
	ID             uint `gorm:"primaryKey;autoIncrement"`
	MemberID       uint `gorm:"not null"`
	FromBranchID   uint `gorm:"not null"`
	ToBranchID     uint `gorm:"not null"`
	SubscriptionID *uint
	TransferFee    float64 `gorm:"type:numeric(14,2);not null;default:0"`
	Reason         string  `gorm:"type:text;not null;default:''"`
	Status         string  `gorm:"type:varchar(20);not null;default:'APPROVED'"`
	ApprovedBy     *uint
	ApprovedAt     *time.Time
	CreatedAt      time.Time
}

func (MemberBranchTransfer) TableName() string { return "member_branch_transfers" }

// LegacyImportJob là phiên chạy thử (dry-run) hoặc ghi chính thức (commit) khi chuyển đổi dữ liệu cũ.
type LegacyImportJob struct {
	ID          uint           `gorm:"primaryKey;autoIncrement"`
	BranchID    uint           `gorm:"not null"`
	FileName    string         `gorm:"type:varchar(255);not null"`
	Mode        string         `gorm:"type:varchar(20);not null;default:'DRY_RUN'"`
	Status      string         `gorm:"type:varchar(20);not null;default:'COMPLETED'"`
	TotalRows   int            `gorm:"not null;default:0"`
	OkRows      int            `gorm:"not null;default:0"`
	ErrorRows   int            `gorm:"not null;default:0"`
	SummaryJSON datatypes.JSON `gorm:"column:summary_json;type:jsonb;not null;default:'{}'"`
	ErrorsJSON  datatypes.JSON `gorm:"column:errors_json;type:jsonb;not null;default:'[]'"`
	CreatedBy   *uint
	CreatedAt   time.Time
}

func (LegacyImportJob) TableName() string { return "legacy_import_jobs" }
