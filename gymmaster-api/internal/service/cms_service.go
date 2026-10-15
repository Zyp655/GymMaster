package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	v1 "gymmaster-api/api/cms/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/realtime"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
)

type CMSService struct {
	repo     *repository.CMSRepository
	notifier Notifier
}

func NewCMSService(repo *repository.CMSRepository, notifier Notifier) *CMSService {
	return &CMSService{repo: repo, notifier: notifier}
}

func (s *CMSService) enrichTrainerProfile(ctx context.Context, p entity.TrainerPublicProfile) *v1.PublicTrainerItem {
	item := &v1.PublicTrainerItem{
		ID:              p.ID,
		UserID:          p.UserID,
		Slug:            p.Slug,
		Headline:        p.Headline,
		Bio:             p.Bio,
		Specialties:     p.Specialties,
		Certifications:  p.Certifications,
		ExperienceYears: p.ExperienceYears,
		AvatarURL:       p.AvatarURL,
		Featured:        p.Featured,
		IsPublished:     p.IsPublished,
	}
	if s.repo.DB() != nil {
		var u entity.User
		if err := s.repo.DB().WithContext(ctx).First(&u, p.UserID).Error; err == nil {
			item.FullName = u.FullName
			if item.AvatarURL == "" {
				item.AvatarURL = u.AvatarURL
			}
		}
		var tp entity.TrainerProfile
		if err := s.repo.DB().WithContext(ctx).Where("user_id = ?", p.UserID).First(&tp).Error; err == nil {
			item.RatingAvg = tp.RatingAvg
			item.RatingCount = tp.RatingCount
			if item.Specialties == "" && len(tp.Specialties) > 0 {
				item.Specialties = strings.Join(tp.Specialties, ", ")
			}
			if item.Certifications == "" {
				item.Certifications = tp.Certifications
			}
			if item.Bio == "" {
				item.Bio = tp.Bio
			}
		}
	}
	return item
}

func (s *CMSService) PublicTrainers(ctx context.Context) (v1.PublicTrainersRes, error) {
	list, err := s.repo.ListTrainerProfiles(ctx, true)
	if err != nil {
		return nil, err
	}
	res := make(v1.PublicTrainersRes, 0, len(list))
	for _, p := range list {
		res = append(res, s.enrichTrainerProfile(ctx, p))
	}
	return res, nil
}

func (s *CMSService) PublicTrainerDetail(ctx context.Context, slug string) (*v1.PublicTrainerDetailRes, error) {
	p, err := s.repo.FindTrainerProfileBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy hồ sơ huấn luyện viên")
		}
		return nil, err
	}
	return &v1.PublicTrainerDetailRes{PublicTrainerItem: s.enrichTrainerProfile(ctx, *p)}, nil
}

func (s *CMSService) PublicClassTypes(ctx context.Context) (v1.PublicClassTypesRes, error) {
	var list []entity.ClassType
	if err := s.repo.DB().WithContext(ctx).Where("status = 'ACTIVE'").Order("id ASC").Find(&list).Error; err != nil {
		return nil, err
	}
	res := make(v1.PublicClassTypesRes, 0, len(list))
	for _, ct := range list {
		res = append(res, &v1.PublicClassTypeItem{
			ID:              ct.ID,
			Code:            ct.Code,
			Name:            ct.Name,
			Description:     ct.Description,
			DefaultDuration: ct.DefaultDurationMin,
			DefaultCapacity: ct.DefaultCapacity,
			Color:           ct.Color,
		})
	}
	return res, nil
}

func (s *CMSService) PublicClassTypeDetail(ctx context.Context, id uint) (*v1.PublicClassTypeDetailRes, error) {
	var ct entity.ClassType
	if err := s.repo.DB().WithContext(ctx).First(&ct, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy bộ môn lớp học")
		}
		return nil, err
	}
	var schedules []entity.ClassSchedule
	_ = s.repo.DB().WithContext(ctx).
		Where("class_type_id = ? AND status = 'SCHEDULED' AND start_at >= ?", id, time.Now().Add(-2*time.Hour)).
		Order("start_at ASC").Limit(20).Find(&schedules).Error

	briefs := make([]*v1.PublicClassScheduleBrief, 0, len(schedules))
	for _, sc := range schedules {
		var branch entity.Branch
		var room entity.Room
		var trainer entity.User
		_ = s.repo.DB().WithContext(ctx).First(&branch, sc.BranchID).Error
		_ = s.repo.DB().WithContext(ctx).First(&room, sc.RoomID).Error
		_ = s.repo.DB().WithContext(ctx).First(&trainer, sc.TrainerID).Error
		var booked int64
		_ = s.repo.DB().WithContext(ctx).Model(&entity.ClassBooking{}).
			Where("schedule_id = ? AND status = 'BOOKED'", sc.ID).Count(&booked).Error
		briefs = append(briefs, &v1.PublicClassScheduleBrief{
			ID:          sc.ID,
			BranchID:    sc.BranchID,
			BranchName:  branch.Name,
			RoomName:    room.Name,
			TrainerName: trainer.FullName,
			StartAt:     sc.StartAt,
			EndAt:       sc.EndAt,
			Capacity:    sc.Capacity,
			BookedCount: int(booked),
		})
	}

	return &v1.PublicClassTypeDetailRes{
		ClassType: &v1.PublicClassTypeItem{
			ID:              ct.ID,
			Code:            ct.Code,
			Name:            ct.Name,
			Description:     ct.Description,
			DefaultDuration: ct.DefaultDurationMin,
			DefaultCapacity: ct.DefaultCapacity,
			Color:           ct.Color,
		},
		Schedules: briefs,
	}, nil
}

func (s *CMSService) PublicTestimonials(ctx context.Context) (v1.PublicTestimonialsRes, error) {
	list, err := s.repo.ListTestimonials(ctx, entity.TestimonialApproved)
	if err != nil {
		return nil, err
	}
	return mapTestimonials(list), nil
}

func (s *CMSService) SubmitTestimonial(ctx context.Context, req *v1.PublicSubmitTestimonialReq) (*v1.PublicSubmitTestimonialRes, error) {
	rating := req.Rating
	if rating < 1 || rating > 5 {
		rating = 5
	}
	t := &entity.Testimonial{
		AuthorName:      strings.TrimSpace(req.AuthorName),
		AuthorRole:      strings.TrimSpace(req.AuthorRole),
		Rating:          rating,
		Content:         strings.TrimSpace(req.Content),
		BeforeAfterNote: strings.TrimSpace(req.BeforeAfterNote),
		Status:          entity.TestimonialPending,
	}
	if err := s.repo.CreateTestimonial(ctx, t); err != nil {
		return nil, err
	}
	return &v1.PublicSubmitTestimonialRes{TestimonialItem: toTestimonialItem(*t)}, nil
}

func (s *CMSService) PublicPosts(ctx context.Context, category, search string) (v1.PublicPostsRes, error) {
	list, err := s.repo.ListPosts(ctx, entity.PostStatusPublished, category, search)
	if err != nil {
		return nil, err
	}
	return mapPosts(list), nil
}

func (s *CMSService) PublicPostDetail(ctx context.Context, slug string) (*v1.PublicPostDetailRes, error) {
	p, err := s.repo.FindPostBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy bài viết")
		}
		return nil, err
	}
	return &v1.PublicPostDetailRes{PostItem: toPostItem(*p)}, nil
}

func (s *CMSService) SubmitContactMessage(ctx context.Context, req *v1.PublicContactMessageReq) (*v1.PublicContactMessageRes, error) {
	msg := &entity.ContactMessage{
		BranchID: req.BranchID,
		FullName: strings.TrimSpace(req.FullName),
		Phone:    strings.TrimSpace(req.Phone),
		Email:    strings.TrimSpace(req.Email),
		Subject:  strings.TrimSpace(req.Subject),
		Message:  strings.TrimSpace(req.Message),
		Status:   entity.ContactStatusNew,
	}
	if err := s.repo.CreateContactMessage(ctx, msg); err != nil {
		return nil, err
	}
	if s.notifier != nil {
		_ = s.notifier.NotifyPermission(ctx, consts.PermLeadManage, req.BranchID, NotifyLeadNew,
			"Tin nhắn liên hệ mới từ Website",
			fmt.Sprintf("%s (%s): %s", msg.FullName, msg.Phone, msg.Subject),
			"/app/cms-marketing")
	}
	return &v1.PublicContactMessageRes{ContactMessageItem: toContactItem(*msg)}, nil
}

// ==================== ADMIN CMS ====================

func (s *CMSService) AdminListTrainerProfiles(ctx context.Context) (v1.AdminListTrainerProfilesRes, error) {
	list, err := s.repo.ListTrainerProfiles(ctx, false)
	if err != nil {
		return nil, err
	}
	res := make(v1.AdminListTrainerProfilesRes, 0, len(list))
	for _, p := range list {
		res = append(res, s.enrichTrainerProfile(ctx, p))
	}
	return res, nil
}

func (s *CMSService) AdminSaveTrainerProfile(ctx context.Context, req *v1.AdminSaveTrainerProfileReq) (*v1.AdminSaveTrainerProfileRes, error) {
	p := &entity.TrainerPublicProfile{
		UserID:          req.UserID,
		Slug:            strings.TrimSpace(req.Slug),
		Headline:        strings.TrimSpace(req.Headline),
		Bio:             strings.TrimSpace(req.Bio),
		Specialties:     strings.TrimSpace(req.Specialties),
		Certifications:  strings.TrimSpace(req.Certifications),
		ExperienceYears: req.ExperienceYears,
		AvatarURL:       strings.TrimSpace(req.AvatarURL),
		Featured:        req.Featured,
		IsPublished:     req.IsPublished,
	}
	if err := s.repo.UpsertTrainerProfile(ctx, p); err != nil {
		return nil, err
	}
	return &v1.AdminSaveTrainerProfileRes{PublicTrainerItem: s.enrichTrainerProfile(ctx, *p)}, nil
}

func (s *CMSService) AdminDeleteTrainerProfile(ctx context.Context, id uint) error {
	return s.repo.DeleteTrainerProfile(ctx, id)
}

func (s *CMSService) AdminListTestimonials(ctx context.Context, status string) (v1.AdminListTestimonialsRes, error) {
	list, err := s.repo.ListTestimonials(ctx, status)
	if err != nil {
		return nil, err
	}
	return mapTestimonials(list), nil
}

func (s *CMSService) AdminUpdateTestimonialStatus(ctx context.Context, id uint, status string) (*v1.TestimonialItem, error) {
	t, err := s.repo.FindTestimonialByID(ctx, id)
	if err != nil {
		return nil, rescode.New(rescode.NotFound, "Không tìm thấy cảm nhận hội viên")
	}
	now := time.Now()
	t.Status = status
	if auth := reqctx.Auth(ctx); auth != nil {
		t.ApprovedBy = &auth.UserID
	}
	t.ApprovedAt = &now
	if err := s.repo.UpdateTestimonial(ctx, t); err != nil {
		return nil, err
	}
	return toTestimonialItem(*t), nil
}

func (s *CMSService) AdminDeleteTestimonial(ctx context.Context, id uint) error {
	return s.repo.DeleteTestimonial(ctx, id)
}

func (s *CMSService) AdminListPosts(ctx context.Context, status, category string) (v1.AdminListPostsRes, error) {
	list, err := s.repo.ListPosts(ctx, status, category, "")
	if err != nil {
		return nil, err
	}
	return mapPosts(list), nil
}

func (s *CMSService) AdminCreatePost(ctx context.Context, req *v1.AdminCreatePostReq) (*v1.AdminCreatePostRes, error) {
	now := time.Now()
	status := req.Status
	if status == "" {
		status = entity.PostStatusPublished
	}
	var pubAt *time.Time
	if status == entity.PostStatusPublished {
		pubAt = &now
	}
	var authorID *uint
	if auth := reqctx.Auth(ctx); auth != nil {
		authorID = &auth.UserID
	}
	p := &entity.Post{
		Slug:        strings.TrimSpace(req.Slug),
		Title:       strings.TrimSpace(req.Title),
		Excerpt:     strings.TrimSpace(req.Excerpt),
		CoverURL:    strings.TrimSpace(req.CoverURL),
		Category:    strings.TrimSpace(req.Category),
		Content:     req.Content,
		Status:      status,
		AuthorID:    authorID,
		PublishedAt: pubAt,
	}
	if err := s.repo.CreatePost(ctx, p); err != nil {
		return nil, err
	}
	return &v1.AdminCreatePostRes{PostItem: toPostItem(*p)}, nil
}

func (s *CMSService) AdminUpdatePost(ctx context.Context, req *v1.AdminUpdatePostReq) (*v1.AdminUpdatePostRes, error) {
	p, err := s.repo.FindPostByID(ctx, req.ID)
	if err != nil {
		return nil, rescode.New(rescode.NotFound, "Không tìm thấy bài viết")
	}
	if req.Slug != "" {
		p.Slug = strings.TrimSpace(req.Slug)
	}
	if req.Title != "" {
		p.Title = strings.TrimSpace(req.Title)
	}
	p.Excerpt = strings.TrimSpace(req.Excerpt)
	p.CoverURL = strings.TrimSpace(req.CoverURL)
	if req.Category != "" {
		p.Category = req.Category
	}
	if req.Content != "" {
		p.Content = req.Content
	}
	if req.Status != "" {
		p.Status = req.Status
		if req.Status == entity.PostStatusPublished && p.PublishedAt == nil {
			now := time.Now()
			p.PublishedAt = &now
		}
	}
	if err := s.repo.UpdatePost(ctx, p); err != nil {
		return nil, err
	}
	return &v1.AdminUpdatePostRes{PostItem: toPostItem(*p)}, nil
}

func (s *CMSService) AdminDeletePost(ctx context.Context, id uint) error {
	return s.repo.DeletePost(ctx, id)
}

func (s *CMSService) AdminListContactMessages(ctx context.Context, status string) (v1.AdminListContactMessagesRes, error) {
	list, err := s.repo.ListContactMessages(ctx, status)
	if err != nil {
		return nil, err
	}
	res := make(v1.AdminListContactMessagesRes, 0, len(list))
	for _, m := range list {
		res = append(res, toContactItem(m))
	}
	return res, nil
}

func (s *CMSService) AdminUpdateContactMessage(ctx context.Context, req *v1.AdminUpdateContactMessageReq) (*v1.AdminUpdateContactMessageRes, error) {
	m, err := s.repo.FindContactMessageByID(ctx, req.ID)
	if err != nil {
		return nil, rescode.New(rescode.NotFound, "Không tìm thấy lời nhắn liên hệ")
	}
	m.Status = req.Status
	m.HandledNote = strings.TrimSpace(req.HandledNote)
	if auth := reqctx.Auth(ctx); auth != nil {
		m.HandledBy = &auth.UserID
	}
	if err := s.repo.UpdateContactMessage(ctx, m); err != nil {
		return nil, err
	}
	return &v1.AdminUpdateContactMessageRes{ContactMessageItem: toContactItem(*m)}, nil
}

// ==================== NHẮN TIN HLV ↔ HỘI VIÊN (MỤC 15.1) ====================

func (s *CMSService) ListConversations(ctx context.Context) (v1.ListConversationsRes, error) {
	auth := reqctx.Auth(ctx)
	if auth == nil {
		return nil, rescode.New(rescode.Unauthorized, "Bạn chưa đăng nhập")
	}

	var contracts []entity.PTContract
	q := s.repo.DB().WithContext(ctx).Order("id DESC")
	if auth.HasRole(consts.RoleMember) {
		var member entity.Member
		if err := s.repo.DB().WithContext(ctx).Where("user_id = ?", auth.UserID).First(&member).Error; err != nil {
			return v1.ListConversationsRes{}, nil
		}
		q = q.Where("member_id = ?", member.ID)
	} else if auth.HasRole(consts.RoleTrainer) && !auth.HasRole(consts.RoleOwner) && !auth.HasRole(consts.RoleManager) {
		q = q.Where("trainer_id = ?", auth.UserID)
	}
	if err := q.Limit(30).Find(&contracts).Error; err != nil {
		return nil, err
	}

	res := make(v1.ListConversationsRes, 0, len(contracts))
	for _, c := range contracts {
		conv, err := s.repo.EnsureConversation(ctx, c.ID, c.MemberID, c.TrainerID)
		if err != nil {
			continue
		}
		var member entity.Member
		var trainer entity.User
		_ = s.repo.DB().WithContext(ctx).First(&member, c.MemberID).Error
		_ = s.repo.DB().WithContext(ctx).First(&trainer, c.TrainerID).Error

		remaining := c.TotalSessions - c.UsedSessions
		if remaining < 0 {
			remaining = 0
		}
		res = append(res, &v1.ConversationItem{
			ID:                 conv.ID,
			PTContractID:       c.ID,
			MemberID:           c.MemberID,
			MemberName:         member.FullName,
			MemberCode:         member.MemberCode,
			TrainerID:          c.TrainerID,
			TrainerName:        trainer.FullName,
			ContractStatus:     c.Status,
			RemainingSessions:  remaining,
			LastMessageAt:      conv.LastMessageAt,
			LastMessagePreview: conv.LastMessagePreview,
			UnreadCount:        s.repo.CountUnreadMessages(ctx, conv.ID, auth.UserID),
		})
	}
	return res, nil
}

func (s *CMSService) ListMessages(ctx context.Context, conversationID uint) (v1.ListMessagesRes, error) {
	auth := reqctx.Auth(ctx)
	if auth == nil {
		return nil, rescode.New(rescode.Unauthorized, "Bạn chưa đăng nhập")
	}
	conv, err := s.repo.FindConversationByID(ctx, conversationID)
	if err != nil {
		return nil, rescode.New(rescode.NotFound, "Không tìm thấy cuộc trò chuyện")
	}
	_ = s.repo.MarkMessagesRead(ctx, conv.ID, auth.UserID, time.Now())

	msgs, err := s.repo.ListMessages(ctx, conv.ID)
	if err != nil {
		return nil, err
	}
	res := make(v1.ListMessagesRes, 0, len(msgs))
	for _, m := range msgs {
		var u entity.User
		_ = s.repo.DB().WithContext(ctx).First(&u, m.SenderUserID).Error
		res = append(res, &v1.MessageItem{
			ID:             m.ID,
			ConversationID: m.ConversationID,
			SenderUserID:   m.SenderUserID,
			SenderName:     u.FullName,
			SenderRole:     m.SenderRole,
			Content:        m.Content,
			AttachmentURL:  m.AttachmentURL,
			ReadAt:         m.ReadAt,
			CreatedAt:      m.CreatedAt,
		})
	}
	return res, nil
}

func (s *CMSService) SendMessage(ctx context.Context, req *v1.SendMessageReq) (*v1.SendMessageRes, error) {
	auth := reqctx.Auth(ctx)
	if auth == nil {
		return nil, rescode.New(rescode.Unauthorized, "Bạn chưa đăng nhập")
	}
	conv, err := s.repo.FindConversationByID(ctx, req.ID)
	if err != nil {
		return nil, rescode.New(rescode.NotFound, "Không tìm thấy cuộc trò chuyện")
	}
	var contract entity.PTContract
	if err := s.repo.DB().WithContext(ctx).First(&contract, conv.PTContractID).Error; err != nil {
		return nil, rescode.New(rescode.NotFound, "Không tìm thấy hợp đồng PT")
	}
	if contract.Status != entity.PTContractActive {
		return nil, rescode.New(rescode.BadRequest, "Chỉ được nhắn tin khi hợp đồng PT còn hiệu lực (ACTIVE)")
	}

	senderRole := "TRAINER"
	var recipientUserID uint = conv.TrainerID
	if auth.HasRole(consts.RoleMember) {
		senderRole = "MEMBER"
		recipientUserID = conv.TrainerID
	} else {
		var member entity.Member
		if err := s.repo.DB().WithContext(ctx).First(&member, conv.MemberID).Error; err == nil && member.UserID != nil {
			recipientUserID = *member.UserID
		}
	}

	now := time.Now()
	content := strings.TrimSpace(req.Content)
	preview := content
	if len(preview) > 120 {
		preview = preview[:120] + "..."
	}
	msg := &entity.Message{
		ConversationID: conv.ID,
		SenderUserID:   auth.UserID,
		SenderRole:     senderRole,
		Content:        content,
		AttachmentURL:  strings.TrimSpace(req.AttachmentURL),
		CreatedAt:      now,
	}
	if err := s.repo.CreateMessage(ctx, msg, preview, now); err != nil {
		return nil, err
	}

	var sender entity.User
	_ = s.repo.DB().WithContext(ctx).First(&sender, auth.UserID).Error

	if recipientUserID > 0 && recipientUserID != auth.UserID {
		realtime.Default().Publish(realtime.Event{
			ID:        msg.ID,
			UserID:    recipientUserID,
			Type:      "PT_MESSAGE",
			Title:     fmt.Sprintf("Tin nhắn mới từ %s", sender.FullName),
			Body:      preview,
			Link:      "/me/messages",
			CreatedAt: now,
		})
	}

	return &v1.SendMessageRes{
		MessageItem: &v1.MessageItem{
			ID:             msg.ID,
			ConversationID: msg.ConversationID,
			SenderUserID:   msg.SenderUserID,
			SenderName:     sender.FullName,
			SenderRole:     msg.SenderRole,
			Content:        msg.Content,
			AttachmentURL:  msg.AttachmentURL,
			CreatedAt:      msg.CreatedAt,
		},
	}, nil
}

// ==================== TÙY CHỌN THÔNG BÁO & LỊCH .ICS ====================

var notificationCatalog = []struct {
	Type      string
	Label     string
	Mandatory bool
}{
	{Type: "PAYMENT_SUCCESS", Label: "Biên lai thanh toán & hóa đơn", Mandatory: true},
	{Type: "SYSTEM", Label: "Cảnh báo bảo mật & tài khoản", Mandatory: true},
	{Type: "SUBSCRIPTION_EXPIRING", Label: "Nhắc gói tập sắp hết hạn", Mandatory: false},
	{Type: "CLASS_REMINDER", Label: "Nhắc lịch lớp học & đôn chỗ Waitlist", Mandatory: false},
	{Type: "PT_REMINDER", Label: "Nhắc lịch tập PT & tin nhắn HLV", Mandatory: false},
	{Type: "BIRTHDAY", Label: "Ưu đãi sinh nhật & quà tặng Loyalty", Mandatory: false},
}

func (s *CMSService) GetNotificationPreferences(ctx context.Context) (v1.GetNotificationPreferencesRes, error) {
	auth := reqctx.Auth(ctx)
	if auth == nil {
		return nil, rescode.New(rescode.Unauthorized, "Bạn chưa đăng nhập")
	}
	saved, _ := s.repo.ListNotificationPrefs(ctx, auth.UserID)
	prefMap := make(map[string]bool)
	for _, p := range saved {
		prefMap[p.Type+"|"+p.Channel] = p.Enabled
	}

	res := make(v1.GetNotificationPreferencesRes, 0, len(notificationCatalog)*2)
	for _, cat := range notificationCatalog {
		for _, ch := range []string{entity.NotifyChannelInApp, entity.NotifyChannelEmail} {
			enabled := true
			if v, ok := prefMap[cat.Type+"|"+ch]; ok {
				enabled = v
			}
			if cat.Mandatory {
				enabled = true
			}
			res = append(res, &v1.NotificationPrefItem{
				Type:      cat.Type,
				Label:     cat.Label,
				Channel:   ch,
				Enabled:   enabled,
				Mandatory: cat.Mandatory,
			})
		}
	}
	return res, nil
}

func (s *CMSService) SaveNotificationPreferences(ctx context.Context, req *v1.SaveNotificationPreferencesReq) (v1.SaveNotificationPreferencesRes, error) {
	auth := reqctx.Auth(ctx)
	if auth == nil {
		return nil, rescode.New(rescode.Unauthorized, "Bạn chưa đăng nhập")
	}
	mandatorySet := map[string]bool{"PAYMENT_SUCCESS": true, "SYSTEM": true}
	now := time.Now()
	for _, in := range req.Preferences {
		enabled := in.Enabled
		if mandatorySet[in.Type] {
			enabled = true
		}
		_ = s.repo.UpsertNotificationPref(ctx, &entity.NotificationPreference{
			UserID:    auth.UserID,
			Type:      in.Type,
			Channel:   in.Channel,
			Enabled:   enabled,
			UpdatedAt: now,
		})
	}
	prefs, err := s.GetNotificationPreferences(ctx)
	return v1.SaveNotificationPreferencesRes(prefs), err
}

func (s *CMSService) GetOrRotateCalendarToken(ctx context.Context, rotate bool) (*v1.CalendarTokenRes, error) {
	auth := reqctx.Auth(ctx)
	if auth == nil {
		return nil, rescode.New(rescode.Unauthorized, "Bạn chưa đăng nhập")
	}
	var u entity.User
	if err := s.repo.DB().WithContext(ctx).First(&u, auth.UserID).Error; err != nil {
		return nil, err
	}
	if rotate || u.CalendarToken == nil || *u.CalendarToken == "" {
		raw := make([]byte, 16)
		_, _ = rand.Read(raw)
		tok := "ICS-" + hex.EncodeToString(raw)
		if err := s.repo.DB().WithContext(ctx).Model(&entity.User{}).Where("id = ?", u.ID).Update("calendar_token", tok).Error; err != nil {
			return nil, err
		}
		u.CalendarToken = &tok
	}
	return &v1.CalendarTokenRes{
		Token:  *u.CalendarToken,
		ICSURL: fmt.Sprintf("/v1/public/calendar.ics?token=%s", *u.CalendarToken),
	}, nil
}

func (s *CMSService) BuildCalendarICS(ctx context.Context, tokenStr string) (string, error) {
	if strings.TrimSpace(tokenStr) == "" {
		return "", rescode.New(rescode.BadRequest, "Thiếu calendar token")
	}
	var u entity.User
	if err := s.repo.DB().WithContext(ctx).Where("calendar_token = ?", tokenStr).First(&u).Error; err != nil {
		return "", rescode.New(rescode.Unauthorized, "Calendar token không hợp lệ hoặc đã bị thu hồi")
	}

	var b strings.Builder
	b.WriteString("BEGIN:VCALENDAR\r\n")
	b.WriteString("VERSION:2.0\r\n")
	b.WriteString("PRODID:-//GymMaster//Fitness Calendar//VI\r\n")
	b.WriteString("CALSCALE:GREGORIAN\r\n")
	b.WriteString("X-WR-CALNAME:Lịch tập GymMaster - " + u.FullName + "\r\n")

	var member entity.Member
	if err := s.repo.DB().WithContext(ctx).Where("user_id = ?", u.ID).First(&member).Error; err == nil {
		var ptSessions []entity.PTSession
		_ = s.repo.DB().WithContext(ctx).Where("member_id = ? AND status = 'SCHEDULED'", member.ID).Order("start_at ASC").Limit(50).Find(&ptSessions).Error
		for _, pt := range ptSessions {
			writeICSEvent(&b, fmt.Sprintf("pt-%d@gymmaster.vn", pt.ID), "Buổi tập PT 1-1 tại GymMaster", pt.WorkoutNote, pt.StartAt, pt.EndAt)
		}
		var bookings []entity.ClassBooking
		_ = s.repo.DB().WithContext(ctx).Where("member_id = ? AND status = 'BOOKED'", member.ID).Limit(50).Find(&bookings).Error
		for _, bk := range bookings {
			var sc entity.ClassSchedule
			if err := s.repo.DB().WithContext(ctx).First(&sc, bk.ScheduleID).Error; err == nil {
				var ct entity.ClassType
				_ = s.repo.DB().WithContext(ctx).First(&ct, sc.ClassTypeID).Error
				writeICSEvent(&b, fmt.Sprintf("class-%d@gymmaster.vn", sc.ID), "Lớp nhóm: "+ct.Name, ct.Description, sc.StartAt, sc.EndAt)
			}
		}
	} else {
		var ptSessions []entity.PTSession
		_ = s.repo.DB().WithContext(ctx).Where("trainer_id = ? AND status = 'SCHEDULED'", u.ID).Order("start_at ASC").Limit(50).Find(&ptSessions).Error
		for _, pt := range ptSessions {
			writeICSEvent(&b, fmt.Sprintf("trainer-pt-%d@gymmaster.vn", pt.ID), "Lịch dạy PT 1-1", pt.TrainerNote, pt.StartAt, pt.EndAt)
		}
	}

	b.WriteString("END:VCALENDAR\r\n")
	return b.String(), nil
}

func writeICSEvent(b *strings.Builder, uid, summary, desc string, startAt, endAt time.Time) {
	b.WriteString("BEGIN:VEVENT\r\n")
	b.WriteString("UID:" + uid + "\r\n")
	b.WriteString("DTSTAMP:" + time.Now().UTC().Format("20060102T150405Z") + "\r\n")
	b.WriteString("DTSTART:" + startAt.UTC().Format("20060102T150405Z") + "\r\n")
	b.WriteString("DTEND:" + endAt.UTC().Format("20060102T150405Z") + "\r\n")
	b.WriteString("SUMMARY:" + strings.ReplaceAll(summary, "\n", " ") + "\r\n")
	if desc != "" {
		b.WriteString("DESCRIPTION:" + strings.ReplaceAll(desc, "\n", " ") + "\r\n")
	}
	b.WriteString("END:VEVENT\r\n")
}

func toTestimonialItem(t entity.Testimonial) *v1.TestimonialItem {
	return &v1.TestimonialItem{
		ID:              t.ID,
		MemberID:        t.MemberID,
		AuthorName:      t.AuthorName,
		AuthorRole:      t.AuthorRole,
		AvatarURL:       t.AvatarURL,
		Rating:          t.Rating,
		Content:         t.Content,
		BeforeAfterNote: t.BeforeAfterNote,
		Status:          t.Status,
		ApprovedAt:      t.ApprovedAt,
		CreatedAt:       t.CreatedAt,
	}
}

func mapTestimonials(list []entity.Testimonial) []*v1.TestimonialItem {
	res := make([]*v1.TestimonialItem, 0, len(list))
	for _, t := range list {
		res = append(res, toTestimonialItem(t))
	}
	return res
}

func toPostItem(p entity.Post) *v1.PostItem {
	return &v1.PostItem{
		ID:          p.ID,
		Slug:        p.Slug,
		Title:       p.Title,
		Excerpt:     p.Excerpt,
		CoverURL:    p.CoverURL,
		Category:    p.Category,
		Content:     p.Content,
		Status:      p.Status,
		PublishedAt: p.PublishedAt,
		CreatedAt:   p.CreatedAt,
	}
}

func mapPosts(list []entity.Post) []*v1.PostItem {
	res := make([]*v1.PostItem, 0, len(list))
	for _, p := range list {
		res = append(res, toPostItem(p))
	}
	return res
}

func toContactItem(m entity.ContactMessage) *v1.ContactMessageItem {
	return &v1.ContactMessageItem{
		ID:          m.ID,
		BranchID:    m.BranchID,
		FullName:    m.FullName,
		Phone:       m.Phone,
		Email:       m.Email,
		Subject:     m.Subject,
		Message:     m.Message,
		Status:      m.Status,
		HandledNote: m.HandledNote,
		CreatedAt:   m.CreatedAt,
	}
}
