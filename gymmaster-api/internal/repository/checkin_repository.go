package repository

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/storage/postgres"
)

type (
	// CheckinRow là một lần check-in kèm thông tin hiển thị (hội viên, chi nhánh, gói, nhân viên).
	CheckinRow struct {
		entity.Checkin
		MemberCode      string
		MemberName      string
		MemberPhone     string
		MemberAvatarURL string
		BranchName      string
		PlanName        *string
		PlanType        *string
		StaffName       *string
	}

	// CheckinListFilter là bộ lọc lịch sử đã chuẩn hóa ở service.
	CheckinListFilter struct {
		BranchIDs []uint // nil = không giới hạn chi nhánh
		MemberID  uint
		Result    string
		Method    string
		From      *time.Time // checked_in_at >= From
		To        *time.Time // checked_in_at < To
	}

	// CheckinHourCount: số lượt theo giờ (giờ VN) và kết quả.
	CheckinHourCount struct {
		Hour   int
		Result string
		Count  int64
	}

	CheckinReasonCount struct {
		DenyReason string
		Count      int64
	}

	CheckinStatsRows struct {
		ByHour        []CheckinHourCount
		ByReason      []CheckinReasonCount
		UniqueMembers int64
		Present       int64
	}

	CheckinRepository interface {
		FindMember(ctx context.Context, id uint) (*entity.Member, error)
		// LockMember đọc và khóa dòng hội viên (FOR NO KEY UPDATE) để tuần tự hóa các lần
		// check-in đồng thời của cùng một hội viên trong transaction.
		LockMember(ctx context.Context, id uint) (*entity.Member, error)
		FindBranch(ctx context.Context, id uint) (*entity.Branch, error)
		// LockSubscriptions trả các gói ACTIVE/FROZEN của hội viên và khóa chúng (FOR UPDATE).
		LockSubscriptions(ctx context.Context, memberID uint) ([]entity.CheckinSubscriptionView, error)
		// LastAllowedSince trả lần check-in ALLOWED gần nhất của hội viên từ thời điểm since.
		LastAllowedSince(ctx context.Context, memberID uint, since time.Time) (*CheckinRow, error)
		Create(ctx context.Context, c *entity.Checkin) error
		// UseSession trừ 1 buổi của gói (chỉ khi còn buổi), trả số buổi còn lại.
		UseSession(ctx context.Context, subscriptionID uint) (int, error)
		CreateSubscriptionEvent(ctx context.Context, subscriptionID uint, eventType string, data map[string]any, actorID *uint) error

		FindByID(ctx context.Context, id uint) (*CheckinRow, error)
		// Checkout ghi giờ ra; trả false nếu đã checkout trước đó.
		Checkout(ctx context.Context, id uint, at time.Time) (bool, error)
		List(ctx context.Context, q dto.Query, f CheckinListFilter) ([]CheckinRow, int64, error)
		// Present: hội viên đang trong phòng tập (ALLOWED, chưa checkout, vào từ since).
		Present(ctx context.Context, branchIDs []uint, since time.Time) ([]CheckinRow, error)
		Stats(ctx context.Context, branchIDs []uint, from, to time.Time) (*CheckinStatsRows, error)
	}

	checkinRepository struct {
		db *gorm.DB
	}
)

func NewCheckinRepository(db *gorm.DB) CheckinRepository {
	return &checkinRepository{db: db}
}

const checkinRowSelect = `c.*, m.member_code, m.full_name AS member_name, m.phone AS member_phone,
	m.avatar_url AS member_avatar_url, b.name AS branch_name, mp.name AS plan_name, mp.type AS plan_type,
	u.full_name AS staff_name`

const checkinRowJoins = `JOIN members m ON m.id = c.member_id
	JOIN branches b ON b.id = c.branch_id
	LEFT JOIN subscriptions s ON s.id = c.subscription_id
	LEFT JOIN membership_plans mp ON mp.id = s.plan_id
	LEFT JOIN users u ON u.id = c.staff_id`

func (r *checkinRepository) base(ctx context.Context) *gorm.DB {
	return postgres.Conn(ctx, r.db).Table("checkins c").Select(checkinRowSelect).Joins(checkinRowJoins)
}

func (r *checkinRepository) FindMember(ctx context.Context, id uint) (*entity.Member, error) {
	var m entity.Member
	if err := postgres.Conn(ctx, r.db).First(&m, id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *checkinRepository) LockMember(ctx context.Context, id uint) (*entity.Member, error) {
	var m entity.Member
	err := postgres.Conn(ctx, r.db).Clauses(clause.Locking{Strength: "NO KEY UPDATE"}).First(&m, id).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *checkinRepository) FindBranch(ctx context.Context, id uint) (*entity.Branch, error) {
	var b entity.Branch
	if err := postgres.Conn(ctx, r.db).First(&b, id).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *checkinRepository) LockSubscriptions(ctx context.Context, memberID uint) ([]entity.CheckinSubscriptionView, error) {
	var rows []entity.CheckinSubscriptionView
	err := postgres.Conn(ctx, r.db).Table("subscriptions s").
		Select(`s.id, s.member_id, s.plan_id, s.branch_id, b.name AS branch_name, s.start_date, s.end_date,
			s.total_sessions, s.remaining_sessions, s.status, mp.name AS plan_name, mp.type AS plan_type, mp.access_scope`).
		Joins("JOIN membership_plans mp ON mp.id = s.plan_id").
		Joins("JOIN branches b ON b.id = s.branch_id").
		Where("s.member_id = ? AND s.status IN ?", memberID, []string{"ACTIVE", "FROZEN"}).
		Order("s.id").
		Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "s"}}).
		Scan(&rows).Error
	return rows, err
}

func (r *checkinRepository) LastAllowedSince(ctx context.Context, memberID uint, since time.Time) (*CheckinRow, error) {
	var rows []CheckinRow
	err := r.base(ctx).
		Where("c.member_id = ? AND c.result = ? AND c.checked_in_at >= ?", memberID, entity.CheckinAllowed, since).
		Order("c.checked_in_at DESC").Limit(1).Scan(&rows).Error
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return &rows[0], nil
}

func (r *checkinRepository) Create(ctx context.Context, c *entity.Checkin) error {
	return postgres.Conn(ctx, r.db).Create(c).Error
}

func (r *checkinRepository) UseSession(ctx context.Context, subscriptionID uint) (int, error) {
	var remaining []int
	err := postgres.Conn(ctx, r.db).Raw(
		`UPDATE subscriptions SET remaining_sessions = remaining_sessions - 1
		 WHERE id = ? AND remaining_sessions > 0 RETURNING remaining_sessions`, subscriptionID).
		Scan(&remaining).Error
	if err != nil {
		return 0, err
	}
	if len(remaining) == 0 {
		return 0, gorm.ErrRecordNotFound
	}
	return remaining[0], nil
}

func (r *checkinRepository) CreateSubscriptionEvent(ctx context.Context, subscriptionID uint, eventType string, data map[string]any, actorID *uint) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return postgres.Conn(ctx, r.db).Exec(
		`INSERT INTO subscription_events (subscription_id, event_type, data, actor_id) VALUES (?, ?, ?::jsonb, ?)`,
		subscriptionID, eventType, string(raw), actorID).Error
}

func (r *checkinRepository) FindByID(ctx context.Context, id uint) (*CheckinRow, error) {
	var rows []CheckinRow
	if err := r.base(ctx).Where("c.id = ?", id).Limit(1).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &rows[0], nil
}

func (r *checkinRepository) Checkout(ctx context.Context, id uint, at time.Time) (bool, error) {
	res := postgres.Conn(ctx, r.db).Model(&entity.Checkin{}).
		Where("id = ? AND checked_out_at IS NULL", id).
		Update("checked_out_at", at)
	return res.RowsAffected > 0, res.Error
}

func (r *checkinRepository) List(ctx context.Context, q dto.Query, f CheckinListFilter) ([]CheckinRow, int64, error) {
	db := postgres.Conn(ctx, r.db).Table("checkins c").Joins("JOIN members m ON m.id = c.member_id")
	if f.BranchIDs != nil {
		db = db.Where("c.branch_id IN ?", f.BranchIDs)
	}
	if f.MemberID > 0 {
		db = db.Where("c.member_id = ?", f.MemberID)
	}
	if f.Result != "" {
		db = db.Where("c.result = ?", f.Result)
	}
	if f.Method != "" {
		db = db.Where("c.method = ?", f.Method)
	}
	if f.From != nil {
		db = db.Where("c.checked_in_at >= ?", *f.From)
	}
	if f.To != nil {
		db = db.Where("c.checked_in_at < ?", *f.To)
	}
	if q.Keyword != "" {
		kw := "%" + strings.ToLower(q.Keyword) + "%"
		db = db.Where("(lower(m.full_name) LIKE ? OR lower(m.member_code) LIKE ? OR m.phone LIKE ?)", kw, kw, kw)
	}
	var total int64
	if err := db.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []CheckinRow
	err := db.Select(checkinRowSelect).
		Joins("JOIN branches b ON b.id = c.branch_id").
		Joins("LEFT JOIN subscriptions s ON s.id = c.subscription_id").
		Joins("LEFT JOIN membership_plans mp ON mp.id = s.plan_id").
		Joins("LEFT JOIN users u ON u.id = c.staff_id").
		Scopes(q.Paginate).Order("c.id DESC").Scan(&rows).Error
	return rows, total, err
}

func (r *checkinRepository) Present(ctx context.Context, branchIDs []uint, since time.Time) ([]CheckinRow, error) {
	db := r.base(ctx).Where("c.result = ? AND c.checked_out_at IS NULL AND c.checked_in_at >= ?", entity.CheckinAllowed, since)
	if branchIDs != nil {
		db = db.Where("c.branch_id IN ?", branchIDs)
	}
	var rows []CheckinRow
	err := db.Order("c.checked_in_at DESC").Scan(&rows).Error
	return rows, err
}

func (r *checkinRepository) Stats(ctx context.Context, branchIDs []uint, from, to time.Time) (*CheckinStatsRows, error) {
	scope := func(db *gorm.DB) *gorm.DB {
		db = db.Where("c.checked_in_at >= ? AND c.checked_in_at < ?", from, to)
		if branchIDs != nil {
			db = db.Where("c.branch_id IN ?", branchIDs)
		}
		return db
	}
	conn := postgres.Conn(ctx, r.db)
	out := &CheckinStatsRows{}
	err := conn.Table("checkins c").Scopes(scope).
		Select("extract(hour FROM c.checked_in_at AT TIME ZONE 'Asia/Ho_Chi_Minh')::int AS hour, c.result, count(*) AS count").
		Group("1, 2").Order("1").Scan(&out.ByHour).Error
	if err != nil {
		return nil, err
	}
	err = conn.Table("checkins c").Scopes(scope).Where("c.result = ?", entity.CheckinDenied).
		Select("c.deny_reason, count(*) AS count").Group("c.deny_reason").Order("count DESC").
		Scan(&out.ByReason).Error
	if err != nil {
		return nil, err
	}
	err = conn.Table("checkins c").Scopes(scope).Where("c.result = ?", entity.CheckinAllowed).
		Select("count(DISTINCT c.member_id)").Scan(&out.UniqueMembers).Error
	if err != nil {
		return nil, err
	}
	err = conn.Table("checkins c").Scopes(scope).
		Where("c.result = ? AND c.checked_out_at IS NULL", entity.CheckinAllowed).
		Count(&out.Present).Error
	return out, err
}

// CheckinTrialPassRow là thông tin vé tập thử kèm tên khách tiềm năng và chi nhánh.
type CheckinTrialPassRow struct {
	ID         uint       `gorm:"column:id"`
	LeadID     uint       `gorm:"column:lead_id"`
	BranchID   uint       `gorm:"column:branch_id"`
	BranchName string     `gorm:"column:branch_name"`
	ValidDate  time.Time  `gorm:"column:valid_date"`
	QRToken    string     `gorm:"column:qr_token"`
	UsedAt     *time.Time `gorm:"column:used_at"`
	FullName   string     `gorm:"column:full_name"`
	Phone      string     `gorm:"column:phone"`
}

// CheckinTrialPassRepository mở rộng cho phép quét mã QR vé tập thử (TP-...) tại quầy check-in.
type CheckinTrialPassRepository interface {
	FindTrialPassByToken(ctx context.Context, token string) (*CheckinTrialPassRow, error)
	MarkTrialPassUsed(ctx context.Context, id uint, at time.Time) error
}

func (r *checkinRepository) FindTrialPassByToken(ctx context.Context, token string) (*CheckinTrialPassRow, error) {
	var row CheckinTrialPassRow
	err := postgres.Conn(ctx, r.db).Table("trial_passes tp").
		Select("tp.id, tp.lead_id, tp.branch_id, b.name AS branch_name, tp.valid_date, tp.qr_token, tp.used_at, l.full_name, l.phone").
		Joins("JOIN leads l ON l.id = tp.lead_id AND l.deleted_at IS NULL").
		Joins("JOIN branches b ON b.id = tp.branch_id").
		Where("tp.qr_token = ?", token).
		Take(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *checkinRepository) MarkTrialPassUsed(ctx context.Context, id uint, at time.Time) error {
	return postgres.Conn(ctx, r.db).Table("trial_passes").Where("id = ? AND used_at IS NULL", id).Update("used_at", at).Error
}

// CheckinDayPassRow là thông tin vé ngày / vé khách mời (DP-...).
type CheckinDayPassRow struct {
	ID         uint       `gorm:"column:id"`
	BranchID   uint       `gorm:"column:branch_id"`
	BranchName string     `gorm:"column:branch_name"`
	BuyerName  string     `gorm:"column:buyer_name"`
	Phone      string     `gorm:"column:phone"`
	Type       string     `gorm:"column:type"`
	ValidDate  time.Time  `gorm:"column:valid_date"`
	QRToken    string     `gorm:"column:qr_token"`
	UsedAt     *time.Time `gorm:"column:used_at"`
}

// CheckinDayPassRepository mở rộng cho phép quét mã QR vé ngày / vé khách (DP-...) tại quầy check-in.
type CheckinDayPassRepository interface {
	FindDayPassByToken(ctx context.Context, token string) (*CheckinDayPassRow, error)
	MarkDayPassUsed(ctx context.Context, id uint, at time.Time) error
}

func (r *checkinRepository) FindDayPassByToken(ctx context.Context, token string) (*CheckinDayPassRow, error) {
	var row CheckinDayPassRow
	err := postgres.Conn(ctx, r.db).Table("day_passes dp").
		Select("dp.id, dp.branch_id, b.name AS branch_name, dp.buyer_name, dp.phone, dp.type, dp.valid_date, dp.qr_token, dp.used_at").
		Joins("JOIN branches b ON b.id = dp.branch_id").
		Where("dp.qr_token = ?", token).
		Take(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *checkinRepository) MarkDayPassUsed(ctx context.Context, id uint, at time.Time) error {
	return postgres.Conn(ctx, r.db).Table("day_passes").Where("id = ? AND used_at IS NULL", id).Update("used_at", at).Error
}

