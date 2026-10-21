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
		TrialPassID     *uint
		// StoredDenyMessage: thông điệp chi tiết lưu lúc quét (checkins.deny_message, rỗng với dữ liệu cũ).
		StoredDenyMessage string
		// HolidayDenyMessage: dữ liệu cũ (chưa lưu deny_message) – thông điệp suy ra từ lịch ngày lễ khi lượt bị từ chối
		// BRANCH_CLOSED rơi vào ngày lễ, để lịch sử hiển thị "Chi nhánh nghỉ lễ" thay vì "Ngoài giờ mở cửa".
		HolidayDenyMessage *string
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

// Lượt vào bằng vé ngày / vé tập thử không có hội viên (member_id NULL) → lấy thông tin hiển thị từ vé.
const checkinRowSelect = `c.id, COALESCE(c.member_id, 0) AS member_id, c.branch_id, c.subscription_id, c.device_id,
	c.day_pass_id, c.trial_pass_id, c.checked_in_at, c.checked_out_at, c.method, c.result, c.deny_reason,
	c.staff_id, c.note, c.created_at, c.deny_message AS stored_deny_message,
	COALESCE(m.member_code, CASE WHEN c.day_pass_id IS NOT NULL THEN 'DP-' || c.day_pass_id
		WHEN c.trial_pass_id IS NOT NULL THEN 'TRIAL-' || c.trial_pass_id ELSE '' END) AS member_code,
	COALESCE(m.full_name, CASE WHEN c.day_pass_id IS NOT NULL THEN dp.buyer_name || CASE WHEN dp.type = 'GUEST' THEN ' (Khách mời)' ELSE ' (Vé ngày)' END
		WHEN c.trial_pass_id IS NOT NULL THEN l.full_name || ' (Tập thử)' ELSE '' END, '') AS member_name,
	COALESCE(m.phone, dp.phone, l.phone, '') AS member_phone,
	COALESCE(m.avatar_url, '') AS member_avatar_url, b.name AS branch_name, mp.name AS plan_name, mp.type AS plan_type,
	u.full_name AS staff_name,
	CASE WHEN c.deny_reason = 'BRANCH_CLOSED' AND c.deny_message = '' THEN (
		SELECT CASE WHEN bh.closed THEN 'Chi nhánh nghỉ lễ (' || bh.name || '), không thể check-in'
			WHEN bh.open_time IS NOT NULL AND bh.close_time IS NOT NULL
				THEN 'Ngoài giờ mở cửa ngày lễ ' || bh.name || ' (' || bh.open_time || ' - ' || bh.close_time || ')' END
		FROM branch_holidays bh
		WHERE bh.date = (c.checked_in_at AT TIME ZONE 'Asia/Ho_Chi_Minh')::date AND (bh.branch_id = c.branch_id OR bh.branch_id IS NULL)
		ORDER BY bh.branch_id NULLS LAST, bh.id LIMIT 1) END AS holiday_deny_message`

const checkinRowPassJoins = `LEFT JOIN members m ON m.id = c.member_id
	LEFT JOIN day_passes dp ON dp.id = c.day_pass_id
	LEFT JOIN trial_passes tp ON tp.id = c.trial_pass_id
	LEFT JOIN leads l ON l.id = tp.lead_id`

const checkinRowJoins = checkinRowPassJoins + `
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
			s.total_sessions, s.remaining_sessions, s.status, mp.name AS plan_name, mp.type AS plan_type, mp.access_scope,
			COALESCE(mp.access_hours::text, '') AS access_hours`).
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
	db := postgres.Conn(ctx, r.db).Table("checkins c").Joins(checkinRowPassJoins)
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
		// Tìm cả lượt vào bằng vé: mã QR vé (DP-…, TP-…) và mã hiển thị DP-<id> / TRIAL-<id>.
		db = db.Where(`(lower(m.full_name) LIKE ? OR lower(m.member_code) LIKE ? OR m.phone LIKE ?
			OR lower(dp.buyer_name) LIKE ? OR dp.phone LIKE ? OR lower(l.full_name) LIKE ? OR l.phone LIKE ?
			OR lower(dp.qr_token) LIKE ? OR lower(tp.qr_token) LIKE ?
			OR lower('dp-' || c.day_pass_id::text) LIKE ? OR lower('trial-' || c.trial_pass_id::text) LIKE ?)`,
			kw, kw, kw, kw, kw, kw, kw, kw, kw, kw, kw)
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
	// Lượt khách duy nhất: hội viên + khách vé ngày / vé tập thử (member_id NULL, mỗi vé là một khách).
	err = conn.Table("checkins c").Scopes(scope).Where("c.result = ?", entity.CheckinAllowed).
		Select(`count(DISTINCT COALESCE('m' || c.member_id::text, 'd' || c.day_pass_id::text, 't' || c.trial_pass_id::text))`).
		Scan(&out.UniqueMembers).Error
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
	// FindTrialPassByToken đọc vé và khóa dòng vé (FOR UPDATE) khi chạy trong transaction.
	FindTrialPassByToken(ctx context.Context, token string) (*CheckinTrialPassRow, error)
	// MarkTrialPassUsed đánh dấu đã dùng có điều kiện (used_at IS NULL); trả false nếu vé đã được dùng.
	MarkTrialPassUsed(ctx context.Context, id uint, at time.Time) (bool, error)
}

func (r *checkinRepository) FindTrialPassByToken(ctx context.Context, token string) (*CheckinTrialPassRow, error) {
	var row CheckinTrialPassRow
	err := postgres.Conn(ctx, r.db).Table("trial_passes tp").
		Select("tp.id, tp.lead_id, tp.branch_id, b.name AS branch_name, tp.valid_date, tp.qr_token, tp.used_at, l.full_name, l.phone").
		Joins("JOIN leads l ON l.id = tp.lead_id AND l.deleted_at IS NULL").
		Joins("JOIN branches b ON b.id = tp.branch_id").
		Where("tp.qr_token = ?", token).
		Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "tp"}}).
		Take(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *checkinRepository) MarkTrialPassUsed(ctx context.Context, id uint, at time.Time) (bool, error) {
	res := postgres.Conn(ctx, r.db).Table("trial_passes").Where("id = ? AND used_at IS NULL", id).Update("used_at", at)
	return res.RowsAffected > 0, res.Error
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
	// FindDayPassByToken đọc vé và khóa dòng vé (FOR UPDATE) khi chạy trong transaction.
	FindDayPassByToken(ctx context.Context, token string) (*CheckinDayPassRow, error)
	// MarkDayPassUsed đánh dấu đã dùng có điều kiện (used_at IS NULL); trả false nếu vé đã được dùng.
	MarkDayPassUsed(ctx context.Context, id uint, at time.Time) (bool, error)
}

func (r *checkinRepository) FindDayPassByToken(ctx context.Context, token string) (*CheckinDayPassRow, error) {
	var row CheckinDayPassRow
	err := postgres.Conn(ctx, r.db).Table("day_passes dp").
		Select("dp.id, dp.branch_id, b.name AS branch_name, dp.buyer_name, dp.phone, dp.type, dp.valid_date, dp.qr_token, dp.used_at").
		Joins("JOIN branches b ON b.id = dp.branch_id").
		Where("dp.qr_token = ?", token).
		Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "dp"}}).
		Take(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *checkinRepository) MarkDayPassUsed(ctx context.Context, id uint, at time.Time) (bool, error) {
	res := postgres.Conn(ctx, r.db).Table("day_passes").Where("id = ? AND used_at IS NULL", id).Update("used_at", at)
	return res.RowsAffected > 0, res.Error
}

// CheckinPassRecord là một lượt quét vé ngày / vé tập thử (không gắn hội viên).
type CheckinPassRecord struct {
	BranchID    uint
	DayPassID   *uint
	TrialPassID *uint
	DeviceID    *uint
	CheckedInAt time.Time
	Method      string
	Result      string
	DenyReason  string
	DenyMessage string
	StaffID     *uint
	Note        string
}

// CheckinPassRepository ghi lượt quét vé vào bảng checkins (member_id NULL) để xuất hiện trong lịch sử / thống kê.
type CheckinPassRepository interface {
	CreatePassCheckin(ctx context.Context, rec *CheckinPassRecord) (uint, error)
}

func (r *checkinRepository) CreatePassCheckin(ctx context.Context, rec *CheckinPassRecord) (uint, error) {
	var ids []uint
	err := postgres.Conn(ctx, r.db).Raw(
		`INSERT INTO checkins (member_id, branch_id, day_pass_id, trial_pass_id, device_id, checked_in_at, method, result, deny_reason, deny_message, staff_id, note)
		 VALUES (NULL, ?, ?, ?, ?, ?, ?, ?, ?, LEFT(?, 255), ?, ?) RETURNING id`,
		rec.BranchID, rec.DayPassID, rec.TrialPassID, rec.DeviceID, rec.CheckedInAt, rec.Method, rec.Result,
		rec.DenyReason, rec.DenyMessage, rec.StaffID, rec.Note).Scan(&ids).Error
	if err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, gorm.ErrRecordNotFound
	}
	return ids[0], nil
}

// CheckinDenyMessageWriter lưu thông điệp từ chối chi tiết của một lượt check-in hội viên (checkins.deny_message).
type CheckinDenyMessageWriter interface {
	SetDenyMessage(ctx context.Context, id uint, msg string) error
}

func (r *checkinRepository) SetDenyMessage(ctx context.Context, id uint, msg string) error {
	return postgres.Conn(ctx, r.db).Exec(`UPDATE checkins SET deny_message = LEFT(?, 255) WHERE id = ?`, msg, id).Error
}

// CheckinRulesRepository cung cấp dữ liệu cho chuỗi CheckinRule (nghỉ lễ, hợp đồng, trả góp).
type CheckinRulesRepository interface {
	// FindBranchHoliday trả lịch nghỉ lễ của chi nhánh (ưu tiên lịch riêng chi nhánh, sau đó toàn hệ thống) trong ngày day (YYYY-MM-DD); nil nếu không có.
	FindBranchHoliday(ctx context.Context, branchID uint, day string) (*entity.BranchHoliday, error)
	// HasSignedContract: hội viên đã ký ít nhất một hợp đồng điện tử.
	HasSignedContract(ctx context.Context, memberID uint) (bool, error)
	// HasOverdueInstallment: hội viên có kỳ trả góp OVERDUE hoặc chưa trả quá hạn (due_date < overdueBefore).
	HasOverdueInstallment(ctx context.Context, memberID uint, overdueBefore string) (bool, error)
}

func (r *checkinRepository) FindBranchHoliday(ctx context.Context, branchID uint, day string) (*entity.BranchHoliday, error) {
	var rows []entity.BranchHoliday
	err := postgres.Conn(ctx, r.db).
		Where("date = ? AND (branch_id = ? OR branch_id IS NULL)", day, branchID).
		Order("branch_id NULLS LAST").Order("id").Limit(1).Find(&rows).Error
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return &rows[0], nil
}

func (r *checkinRepository) HasSignedContract(ctx context.Context, memberID uint) (bool, error) {
	var n int64
	err := postgres.Conn(ctx, r.db).Table("member_contracts").
		Where("member_id = ? AND status = 'SIGNED'", memberID).Count(&n).Error
	return n > 0, err
}

func (r *checkinRepository) HasOverdueInstallment(ctx context.Context, memberID uint, overdueBefore string) (bool, error) {
	var n int64
	// Chỉ tính kỳ còn mở (PENDING/OVERDUE) của hóa đơn còn nợ – bỏ qua kỳ đã tất toán/hủy và hóa đơn đã
	// thu đủ/hủy/hoàn (dữ liệu cũ có thể còn kỳ PENDING lệch trạng thái hóa đơn).
	err := postgres.Conn(ctx, r.db).Table("installment_schedules s").
		Joins("JOIN installment_plans ip ON ip.id = s.plan_id").
		Joins("JOIN invoices inv ON inv.id = ip.invoice_id").
		Where("ip.member_id = ? AND s.status IN ('PENDING', 'OVERDUE') AND (s.status = 'OVERDUE' OR s.due_date < ?)", memberID, overdueBefore).
		Where("inv.status IN ('UNPAID', 'PARTIAL')").
		Count(&n).Error
	return n > 0, err
}
