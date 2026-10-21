package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"gymmaster-api/internal/entity"
)

type CMSRepository struct {
	db *gorm.DB
}

func NewCMSRepository(db *gorm.DB) *CMSRepository {
	return &CMSRepository{db: db}
}

func (r *CMSRepository) DB() *gorm.DB { return r.db }

func (r *CMSRepository) ListTrainerProfiles(ctx context.Context, onlyPublished bool) ([]entity.TrainerPublicProfile, error) {
	var list []entity.TrainerPublicProfile
	q := r.db.WithContext(ctx).Order("featured DESC, id ASC")
	if onlyPublished {
		q = q.Where("is_published = TRUE")
	}
	err := q.Find(&list).Error
	return list, err
}

func (r *CMSRepository) FindTrainerProfileBySlug(ctx context.Context, slug string) (*entity.TrainerPublicProfile, error) {
	var item entity.TrainerPublicProfile
	err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *CMSRepository) UpsertTrainerProfile(ctx context.Context, p *entity.TrainerPublicProfile) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"slug", "headline", "bio", "specialties", "certifications", "experience_years", "avatar_url", "featured", "is_published", "updated_at"}),
	}).Create(p).Error
}

func (r *CMSRepository) DeleteTrainerProfile(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&entity.TrainerPublicProfile{}, id).Error
}

func (r *CMSRepository) ListTestimonials(ctx context.Context, status string) ([]entity.Testimonial, error) {
	var list []entity.Testimonial
	q := r.db.WithContext(ctx).Order("created_at DESC")
	if status != "" {
		q = q.Where("status = ?", status)
	}
	err := q.Find(&list).Error
	return list, err
}

func (r *CMSRepository) CreateTestimonial(ctx context.Context, t *entity.Testimonial) error {
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *CMSRepository) FindTestimonialByID(ctx context.Context, id uint) (*entity.Testimonial, error) {
	var item entity.Testimonial
	if err := r.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *CMSRepository) UpdateTestimonial(ctx context.Context, t *entity.Testimonial) error {
	return r.db.WithContext(ctx).Save(t).Error
}

func (r *CMSRepository) DeleteTestimonial(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&entity.Testimonial{}, id).Error
}

func (r *CMSRepository) ListPosts(ctx context.Context, status, category, search string) ([]entity.Post, error) {
	var list []entity.Post
	q := r.db.WithContext(ctx).Order("published_at DESC NULLS LAST, created_at DESC")
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if category != "" {
		q = q.Where("category = ?", category)
	}
	if search != "" {
		like := "%" + search + "%"
		q = q.Where("title ILIKE ? OR excerpt ILIKE ?", like, like)
	}
	err := q.Find(&list).Error
	return list, err
}

func (r *CMSRepository) FindPostBySlug(ctx context.Context, slug string) (*entity.Post, error) {
	var item entity.Post
	if err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *CMSRepository) FindPostByID(ctx context.Context, id uint) (*entity.Post, error) {
	var item entity.Post
	if err := r.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *CMSRepository) CreatePost(ctx context.Context, p *entity.Post) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *CMSRepository) UpdatePost(ctx context.Context, p *entity.Post) error {
	return r.db.WithContext(ctx).Save(p).Error
}

func (r *CMSRepository) DeletePost(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&entity.Post{}, id).Error
}

func (r *CMSRepository) CreateContactMessage(ctx context.Context, m *entity.ContactMessage) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *CMSRepository) ListContactMessages(ctx context.Context, status string) ([]entity.ContactMessage, error) {
	var list []entity.ContactMessage
	q := r.db.WithContext(ctx).Order("created_at DESC")
	if status != "" {
		q = q.Where("status = ?", status)
	}
	err := q.Find(&list).Error
	return list, err
}

func (r *CMSRepository) FindContactMessageByID(ctx context.Context, id uint) (*entity.ContactMessage, error) {
	var item entity.ContactMessage
	if err := r.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *CMSRepository) UpdateContactMessage(ctx context.Context, m *entity.ContactMessage) error {
	return r.db.WithContext(ctx).Save(m).Error
}

// EnsureConversation trả hội thoại của hợp đồng PT (tạo nếu chưa có). Hợp đồng là nguồn sự thật về người tham gia:
// nếu hợp đồng đã đổi HLV/hội viên, conversations.trainer_id/member_id được đồng bộ lại.
func (r *CMSRepository) EnsureConversation(ctx context.Context, contractID, memberID, trainerID uint) (*entity.Conversation, error) {
	var conv entity.Conversation
	err := r.db.WithContext(ctx).Where("pt_contract_id = ?", contractID).First(&conv).Error
	if err == nil {
		if conv.TrainerID != trainerID || conv.MemberID != memberID {
			if err := r.SyncConversationParticipants(ctx, conv.ID, memberID, trainerID); err != nil {
				return nil, err
			}
			conv.TrainerID, conv.MemberID = trainerID, memberID
		}
		return &conv, nil
	}
	conv = entity.Conversation{
		PTContractID: contractID,
		MemberID:     memberID,
		TrainerID:    trainerID,
	}
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "pt_contract_id"}},
		DoNothing: true,
	}).Create(&conv).Error; err != nil {
		return nil, err
	}
	if conv.ID == 0 {
		_ = r.db.WithContext(ctx).Where("pt_contract_id = ?", contractID).First(&conv).Error
	}
	return &conv, nil
}

// SyncConversationParticipants ghi lại member_id/trainer_id của hội thoại theo hợp đồng PT (sau khi đổi HLV).
func (r *CMSRepository) SyncConversationParticipants(ctx context.Context, convID, memberID, trainerID uint) error {
	return r.db.WithContext(ctx).Model(&entity.Conversation{}).Where("id = ?", convID).
		Updates(map[string]any{"member_id": memberID, "trainer_id": trainerID}).Error
}

// ConversationListFilter: phạm vi danh sách hội thoại (đã chuẩn hóa theo vai trò ở service).
type ConversationListFilter struct {
	BranchIDs []uint // nil = mọi chi nhánh (quản lý)
	TrainerID uint
	MemberID  uint
	Offset    int
	Limit     int
}

// ConversationListRow: hợp đồng PT kèm hội thoại (nếu đã có) và thông tin hiển thị.
type ConversationListRow struct {
	ContractID         uint
	MemberID           uint
	TrainerID          uint
	Status             string
	TotalSessions      int
	UsedSessions       int
	ConvID             *uint
	LastMessageAt      *time.Time
	LastMessagePreview string
	MemberName         string
	MemberCode         string
	MemberUserID       *uint
	TrainerName        string
}

// ListConversationContracts: hợp đồng PT trong phạm vi, hội thoại có tin nhắn mới nhất trước (chưa nhắn → theo hợp đồng
// mới nhất), phân trang theo Offset/Limit.
func (r *CMSRepository) ListConversationContracts(ctx context.Context, f ConversationListFilter) ([]ConversationListRow, error) {
	q := r.db.WithContext(ctx).Table("pt_contracts c").
		Select(`c.id AS contract_id, c.member_id, c.trainer_id, c.status, c.total_sessions, c.used_sessions,
			cv.id AS conv_id, cv.last_message_at, COALESCE(cv.last_message_preview, '') AS last_message_preview,
			m.full_name AS member_name, m.member_code, m.user_id AS member_user_id, COALESCE(u.full_name, '') AS trainer_name`).
		Joins("JOIN members m ON m.id = c.member_id").
		Joins("LEFT JOIN users u ON u.id = c.trainer_id").
		Joins("LEFT JOIN conversations cv ON cv.pt_contract_id = c.id")
	if f.BranchIDs != nil {
		q = q.Where("c.branch_id IN ?", f.BranchIDs)
	}
	if f.TrainerID > 0 {
		q = q.Where("c.trainer_id = ?", f.TrainerID)
	}
	if f.MemberID > 0 {
		q = q.Where("c.member_id = ?", f.MemberID)
	}
	var rows []ConversationListRow
	err := q.Order("cv.last_message_at DESC NULLS LAST").Order("c.id DESC").Offset(f.Offset).Limit(f.Limit).Scan(&rows).Error
	return rows, err
}

func (r *CMSRepository) FindConversationByID(ctx context.Context, id uint) (*entity.Conversation, error) {
	var conv entity.Conversation
	if err := r.db.WithContext(ctx).First(&conv, id).Error; err != nil {
		return nil, err
	}
	return &conv, nil
}

func (r *CMSRepository) ListMessages(ctx context.Context, conversationID uint) ([]entity.Message, error) {
	var list []entity.Message
	err := r.db.WithContext(ctx).Where("conversation_id = ?", conversationID).Order("created_at ASC").Find(&list).Error
	return list, err
}

func (r *CMSRepository) MarkMessagesRead(ctx context.Context, conversationID, readerUserID uint, readerRole string, now time.Time) error {
	q := r.db.WithContext(ctx).Model(&entity.Message{}).
		Where("conversation_id = ? AND sender_user_id <> ? AND read_at IS NULL", conversationID, readerUserID)
	if readerRole != "" {
		q = q.Where("sender_role <> ?", readerRole)
	}
	return q.Update("read_at", now).Error
}

func (r *CMSRepository) CountUnreadMessages(ctx context.Context, conversationID, readerUserID uint, readerRole string) int64 {
	var count int64
	q := r.db.WithContext(ctx).Model(&entity.Message{}).
		Where("conversation_id = ? AND sender_user_id <> ? AND read_at IS NULL", conversationID, readerUserID)
	if readerRole != "" {
		q = q.Where("sender_role <> ?", readerRole)
	}
	_ = q.Count(&count).Error
	return count
}

func (r *CMSRepository) CreateMessage(ctx context.Context, m *entity.Message, preview string, now time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(m).Error; err != nil {
			return err
		}
		return tx.Model(&entity.Conversation{}).Where("id = ?", m.ConversationID).Updates(map[string]any{
			"last_message_at":      now,
			"last_message_preview": preview,
			"updated_at":           now,
		}).Error
	})
}

func (r *CMSRepository) ListNotificationPrefs(ctx context.Context, userID uint) ([]entity.NotificationPreference, error) {
	var list []entity.NotificationPreference
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&list).Error
	return list, err
}

func (r *CMSRepository) UpsertNotificationPref(ctx context.Context, pref *entity.NotificationPreference) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "type"}, {Name: "channel"}},
		DoUpdates: clause.AssignmentColumns([]string{"enabled", "updated_at"}),
	}).Create(pref).Error
}
