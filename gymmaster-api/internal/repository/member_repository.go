package repository

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/storage/postgres"
)

// Bộ lọc tình trạng gói của danh sách hội viên.
const (
	MemberSubActive   = "active"
	MemberSubExpiring = "expiring"
	MemberSubNone     = "none"
)

// MemberExpiringDays: gói còn ≤ số ngày này thì coi là "sắp hết hạn".
const MemberExpiringDays = 7

type (
	// MemberSubscription là gói đang hiệu lực (ACTIVE/FROZEN, end_date xa nhất) của hội viên.
	// Đọc trực tiếp bảng subscriptions/membership_plans (module bán gói quản lý ghi).
	MemberSubscription struct {
		ID                uint
		PlanID            uint
		PlanName          string
		Status            string
		StartDate         time.Time
		EndDate           time.Time
		RemainingSessions *int
	}

	// MemberRow là hội viên kèm tên chi nhánh và gói hiện tại (cho danh sách).
	MemberRow struct {
		entity.Member
		HomeBranchName string
		Subscription   *MemberSubscription
	}

	MemberCheckinStats struct {
		Total  int64
		LastAt *time.Time
	}

	// MemberListFilter là bộ lọc đã chuẩn hóa ở service.
	MemberListFilter struct {
		BranchIDs          []uint // nil = không giới hạn chi nhánh
		Status             string
		Gender             string
		Source             string
		SubscriptionStatus string
	}

	MemberRepository interface {
		List(ctx context.Context, q dto.Query, f MemberListFilter) ([]MemberRow, int64, error)
		Options(ctx context.Context, keyword string, branchIDs []uint, limit int) ([]entity.Member, error)
		FindByID(ctx context.Context, id uint) (*entity.Member, error)
		// ExistsPhone kiểm tra SĐT đã thuộc hội viên khác (chưa xóa) chưa.
		ExistsPhone(ctx context.Context, phone string, excludeID uint) (bool, error)
		// ExistingPhones trả tập SĐT đã tồn tại trong số phones (dùng cho import).
		ExistingPhones(ctx context.Context, phones []string) (map[string]bool, error)
		Create(ctx context.Context, m *entity.Member) error
		Update(ctx context.Context, m *entity.Member) error
		Delete(ctx context.Context, id uint, deletedBy *uint) error
		SetUserID(ctx context.Context, id uint, userID uint, updatedBy *uint) error
		SetQRSecret(ctx context.Context, id uint, secret string, updatedBy *uint) error

		FindBranch(ctx context.Context, id uint) (*entity.Branch, error)
		ActiveSubscription(ctx context.Context, memberID uint) (*MemberSubscription, error)
		// CountOpenSubscriptions đếm gói ACTIVE/FROZEN của hội viên.
		CountOpenSubscriptions(ctx context.Context, memberID uint) (int64, error)
		CheckinStats(ctx context.Context, memberID uint) (MemberCheckinStats, error)

		// Tài khoản cổng hội viên.
		RoleIDByCode(ctx context.Context, code string) (uint, error)
		UpdatePortalUser(ctx context.Context, userID uint, fields map[string]any) error
		RevokeUserTokens(ctx context.Context, userID uint, at time.Time) error
	}

	memberRepository struct {
		db *gorm.DB
	}
)

func NewMemberRepository(db *gorm.DB) MemberRepository {
	return &memberRepository{db: db}
}

// todayVN: ngày hiện tại theo giờ Việt Nam (không phụ thuộc timezone của session DB).
const todayVN = "(now() AT TIME ZONE 'Asia/Ho_Chi_Minh')::date"

// memberSubLateral chọn gói ACTIVE/FROZEN có ngày hết hạn xa nhất của mỗi hội viên.
const memberSubLateral = `LEFT JOIN LATERAL (
	SELECT s.id, s.plan_id, s.status, s.start_date, s.end_date, s.remaining_sessions
	FROM subscriptions s
	WHERE s.member_id = m.id AND s.status IN ('ACTIVE', 'FROZEN')
	ORDER BY s.end_date DESC, s.id DESC
	LIMIT 1
) sub ON true
LEFT JOIN membership_plans mp ON mp.id = sub.plan_id`

type memberListScan struct {
	entity.Member
	HomeBranchName       string
	SubID                *uint
	SubPlanID            *uint
	SubPlanName          *string
	SubStatus            *string
	SubStartDate         *time.Time
	SubEndDate           *time.Time
	SubRemainingSessions *int
}

func (s memberListScan) toRow() MemberRow {
	row := MemberRow{Member: s.Member, HomeBranchName: s.HomeBranchName}
	if s.SubID != nil {
		sub := &MemberSubscription{ID: *s.SubID, RemainingSessions: s.SubRemainingSessions}
		if s.SubPlanID != nil {
			sub.PlanID = *s.SubPlanID
		}
		if s.SubPlanName != nil {
			sub.PlanName = *s.SubPlanName
		}
		if s.SubStatus != nil {
			sub.Status = *s.SubStatus
		}
		if s.SubStartDate != nil {
			sub.StartDate = *s.SubStartDate
		}
		if s.SubEndDate != nil {
			sub.EndDate = *s.SubEndDate
		}
		row.Subscription = sub
	}
	return row
}

func (r *memberRepository) List(ctx context.Context, q dto.Query, f MemberListFilter) ([]MemberRow, int64, error) {
	tx := postgres.Conn(ctx, r.db).Table("members m").
		Joins("JOIN branches b ON b.id = m.home_branch_id").
		Joins(memberSubLateral).
		Where("m.deleted_at IS NULL")
	if q.Keyword != "" {
		kw := likeContains(q.Keyword)
		phoneKw := likeContains(normalizePhoneKeyword(q.Keyword))
		tx = tx.Where("m.full_name ILIKE ? OR m.phone LIKE ? OR m.member_code ILIKE ? OR m.email ILIKE ?", kw, phoneKw, kw, kw)
	}
	if f.BranchIDs != nil {
		tx = tx.Where("m.home_branch_id IN ?", f.BranchIDs)
	}
	if f.Status != "" {
		tx = tx.Where("m.status = ?", f.Status)
	}
	if f.Gender != "" {
		tx = tx.Where("m.gender = ?", f.Gender)
	}
	if f.Source != "" {
		tx = tx.Where("m.source = ?", f.Source)
	}
	switch f.SubscriptionStatus {
	case MemberSubActive:
		tx = tx.Where("sub.id IS NOT NULL")
	case MemberSubExpiring:
		tx = tx.Where("sub.id IS NOT NULL AND sub.end_date <= "+todayVN+" + ?::int", MemberExpiringDays)
	case MemberSubNone:
		tx = tx.Where("sub.id IS NULL")
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var scans []memberListScan
	err := tx.Select(`m.*, b.name AS home_branch_name,
			sub.id AS sub_id, sub.plan_id AS sub_plan_id, mp.name AS sub_plan_name, sub.status AS sub_status,
			sub.start_date AS sub_start_date, sub.end_date AS sub_end_date, sub.remaining_sessions AS sub_remaining_sessions`).
		Scopes(q.Paginate).Scan(&scans).Error
	if err != nil {
		return nil, 0, err
	}
	rows := make([]MemberRow, 0, len(scans))
	for _, s := range scans {
		rows = append(rows, s.toRow())
	}
	return rows, total, nil
}

// normalizePhoneKeyword cho phép tìm "+84 912..." hoặc "0912 345" khớp SĐT đã chuẩn hóa.
func normalizePhoneKeyword(kw string) string {
	s := strings.NewReplacer(" ", "", ".", "", "-", "").Replace(kw)
	if strings.HasPrefix(s, "+84") {
		s = "0" + s[3:]
	}
	return s
}

func (r *memberRepository) Options(ctx context.Context, keyword string, branchIDs []uint, limit int) ([]entity.Member, error) {
	tx := postgres.Conn(ctx, r.db).Model(&entity.Member{})
	if keyword != "" {
		kw := likeContains(keyword)
		tx = tx.Where("full_name ILIKE ? OR phone LIKE ? OR member_code ILIKE ?", kw, likeContains(normalizePhoneKeyword(keyword)), kw)
	}
	if branchIDs != nil {
		tx = tx.Where("home_branch_id IN ?", branchIDs)
	}
	var items []entity.Member
	// Ưu tiên khớp chính xác SĐT/mã, sau đó theo tên.
	order := "full_name ASC, id DESC"
	if keyword != "" {
		tx = tx.Order(gorm.Expr("CASE WHEN phone = ? OR upper(member_code) = upper(?) THEN 0 ELSE 1 END",
			normalizePhoneKeyword(keyword), keyword))
	}
	err := tx.Order(order).Limit(limit).Find(&items).Error
	return items, err
}

func (r *memberRepository) FindByID(ctx context.Context, id uint) (*entity.Member, error) {
	var m entity.Member
	if err := postgres.Conn(ctx, r.db).First(&m, id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *memberRepository) ExistsPhone(ctx context.Context, phone string, excludeID uint) (bool, error) {
	var n int64
	err := postgres.Conn(ctx, r.db).Model(&entity.Member{}).
		Where("phone = ? AND id <> ?", phone, excludeID).Count(&n).Error
	return n > 0, err
}

func (r *memberRepository) ExistingPhones(ctx context.Context, phones []string) (map[string]bool, error) {
	out := map[string]bool{}
	if len(phones) == 0 {
		return out, nil
	}
	var found []string
	if err := postgres.Conn(ctx, r.db).Model(&entity.Member{}).
		Where("phone IN ?", phones).Pluck("phone", &found).Error; err != nil {
		return nil, err
	}
	for _, p := range found {
		out[p] = true
	}
	return out, nil
}

func (r *memberRepository) Create(ctx context.Context, m *entity.Member) error {
	return postgres.Conn(ctx, r.db).Create(m).Error
}

func (r *memberRepository) Update(ctx context.Context, m *entity.Member) error {
	return postgres.Conn(ctx, r.db).Model(m).
		Select("full_name", "gender", "dob", "phone", "email", "id_card", "address", "avatar_url",
			"emergency_contact_name", "emergency_contact_phone", "home_branch_id", "source", "status", "note", "updated_by").
		Updates(m).Error
}

func (r *memberRepository) Delete(ctx context.Context, id uint, deletedBy *uint) error {
	conn := postgres.Conn(ctx, r.db)
	if err := conn.Model(&entity.Member{}).Where("id = ?", id).UpdateColumn("updated_by", deletedBy).Error; err != nil {
		return err
	}
	return conn.Delete(&entity.Member{}, id).Error
}

func (r *memberRepository) SetUserID(ctx context.Context, id uint, userID uint, updatedBy *uint) error {
	return postgres.Conn(ctx, r.db).Model(&entity.Member{}).Where("id = ?", id).
		Updates(map[string]any{"user_id": userID, "updated_by": updatedBy}).Error
}

func (r *memberRepository) SetQRSecret(ctx context.Context, id uint, secret string, updatedBy *uint) error {
	// qr_secret là trường chỉ đọc trong entity (tag "->") nên cập nhật qua Table để GORM không bỏ qua cột.
	return postgres.Conn(ctx, r.db).Table("members").Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]any{"qr_secret": secret, "updated_by": updatedBy}).Error
}

func (r *memberRepository) FindBranch(ctx context.Context, id uint) (*entity.Branch, error) {
	var b entity.Branch
	if err := postgres.Conn(ctx, r.db).First(&b, id).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *memberRepository) ActiveSubscription(ctx context.Context, memberID uint) (*MemberSubscription, error) {
	var scans []memberListScan
	err := postgres.Conn(ctx, r.db).Table("members m").
		Joins(memberSubLateral).
		Where("m.id = ?", memberID).
		Select(`m.id, sub.id AS sub_id, sub.plan_id AS sub_plan_id, mp.name AS sub_plan_name, sub.status AS sub_status,
			sub.start_date AS sub_start_date, sub.end_date AS sub_end_date, sub.remaining_sessions AS sub_remaining_sessions`).
		Scan(&scans).Error
	if err != nil || len(scans) == 0 {
		return nil, err
	}
	return scans[0].toRow().Subscription, nil
}

func (r *memberRepository) CountOpenSubscriptions(ctx context.Context, memberID uint) (int64, error) {
	var n int64
	err := postgres.Conn(ctx, r.db).Table("subscriptions").
		Where("member_id = ? AND status IN ('ACTIVE', 'FROZEN')", memberID).Count(&n).Error
	return n, err
}

func (r *memberRepository) CheckinStats(ctx context.Context, memberID uint) (MemberCheckinStats, error) {
	var out struct {
		Total  int64
		LastAt *time.Time
	}
	err := postgres.Conn(ctx, r.db).Table("checkins").
		Select("count(*) AS total, max(checked_in_at) AS last_at").
		Where("member_id = ? AND result = 'ALLOWED'", memberID).
		Scan(&out).Error
	return MemberCheckinStats{Total: out.Total, LastAt: out.LastAt}, err
}

func (r *memberRepository) RoleIDByCode(ctx context.Context, code string) (uint, error) {
	var role entity.Role
	if err := postgres.Conn(ctx, r.db).Select("id").Where("code = ?", code).First(&role).Error; err != nil {
		return 0, err
	}
	return role.ID, nil
}

func (r *memberRepository) UpdatePortalUser(ctx context.Context, userID uint, fields map[string]any) error {
	return postgres.Conn(ctx, r.db).Model(&entity.User{}).Where("id = ?", userID).Updates(fields).Error
}

func (r *memberRepository) RevokeUserTokens(ctx context.Context, userID uint, at time.Time) error {
	return postgres.Conn(ctx, r.db).Model(&entity.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		UpdateColumn("revoked_at", at).Error
}
