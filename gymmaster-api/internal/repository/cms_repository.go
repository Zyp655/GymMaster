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

func (r *CMSRepository) EnsureConversation(ctx context.Context, contractID, memberID, trainerID uint) (*entity.Conversation, error) {
	var conv entity.Conversation
	err := r.db.WithContext(ctx).Where("pt_contract_id = ?", contractID).First(&conv).Error
	if err == nil {
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

func (r *CMSRepository) MarkMessagesRead(ctx context.Context, conversationID, readerUserID uint, now time.Time) error {
	return r.db.WithContext(ctx).Model(&entity.Message{}).
		Where("conversation_id = ? AND sender_user_id <> ? AND read_at IS NULL", conversationID, readerUserID).
		Update("read_at", now).Error
}

func (r *CMSRepository) CountUnreadMessages(ctx context.Context, conversationID, readerUserID uint) int64 {
	var count int64
	_ = r.db.WithContext(ctx).Model(&entity.Message{}).
		Where("conversation_id = ? AND sender_user_id <> ? AND read_at IS NULL", conversationID, readerUserID).
		Count(&count).Error
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
