package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/storage/postgres"
)

type (
	TrainerAvailabilityRow struct {
		entity.TrainerAvailability
		BranchName string
	}

	LeaveRequestRow struct {
		entity.StaffLeaveRequest
		StaffName    string
		BranchName   string
		ApproverName string
	}

	ImpactedClassRow struct {
		entity.ClassSchedule
		ClassTypeName         string
		BranchName            string
		RoomName              string
		TrainerName           string
		SubstituteTrainerName string
	}

	ImpactedPTSessionRow struct {
		entity.PTSession
		MemberCode          string
		MemberName          string
		TrainerName         string
		OriginalTrainerName string
		BranchName          string
	}

	BranchHolidayRow struct {
		entity.BranchHoliday
		BranchName string
	}

	ScheduleExtRepository interface {
		ListTrainerAvailability(ctx context.Context, trainerID, branchID uint) ([]TrainerAvailabilityRow, error)
		ReplaceTrainerAvailability(ctx context.Context, trainerID, branchID uint, slots []entity.TrainerAvailability) error

		ListLeaveRequests(ctx context.Context, q dto.Query, staffID, branchID uint, status string) ([]LeaveRequestRow, int64, error)
		FindLeaveRequestByID(ctx context.Context, id uint) (*LeaveRequestRow, error)
		CreateLeaveRequest(ctx context.Context, lr *entity.StaffLeaveRequest) error
		UpdateLeaveRequest(ctx context.Context, lr *entity.StaffLeaveRequest) error

		ListImpactedClassSchedules(ctx context.Context, trainerID uint, fromAt, toAt time.Time) ([]ImpactedClassRow, error)
		ListImpactedPTSessions(ctx context.Context, trainerID uint, fromAt, toAt time.Time) ([]ImpactedPTSessionRow, error)
		FindClassScheduleByID(ctx context.Context, id uint) (*ImpactedClassRow, error)
		SetClassSubstituteTrainer(ctx context.Context, scheduleID uint, subTrainerID *uint, note string, updatedBy *uint) error
		FindPTSessionByID(ctx context.Context, id uint) (*ImpactedPTSessionRow, error)
		ReassignPTSessionTrainer(ctx context.Context, sessionID, newTrainerID uint, origTrainerID uint) error
		// TrainerEligible: user tồn tại, ACTIVE, là HLV (vai trò TRAINER hoặc hồ sơ vị trí TRAINER) và được phân công
		// tại chi nhánh branchID (hoặc toàn hệ thống).
		TrainerEligible(ctx context.Context, trainerID, branchID uint) (bool, error)
		// TrainerBusy trả "PT" nếu HLV có buổi PT khác (SCHEDULED/COMPLETED, trừ excludePTID) giao với [start, end),
		// "CLASS" nếu HLV dạy (chính hoặc dạy thay) lớp chưa hủy khác (trừ excludeClassID), "" nếu trống.
		TrainerBusy(ctx context.Context, trainerID uint, start, end time.Time, excludeClassID, excludePTID uint) (string, error)
		// ListScheduledClassesBetween / ListScheduledPTSessionsBetween: buổi lớp / PT SCHEDULED giao với [fromAt, toAt)
		// tại chi nhánh branchID (nil = mọi chi nhánh) – dùng cảnh báo khi thêm/sửa ngày lễ.
		ListScheduledClassesBetween(ctx context.Context, branchID *uint, fromAt, toAt time.Time) ([]ImpactedClassRow, error)
		ListScheduledPTSessionsBetween(ctx context.Context, branchID *uint, fromAt, toAt time.Time) ([]ImpactedPTSessionRow, error)

		ListBranchHolidays(ctx context.Context, q dto.Query, branchID uint, year int) ([]BranchHolidayRow, int64, error)
		ListUpcomingHolidays(ctx context.Context, branchID uint, fromDate time.Time) ([]BranchHolidayRow, error)
		FindBranchHolidayByID(ctx context.Context, id uint) (*BranchHolidayRow, error)
		CreateBranchHoliday(ctx context.Context, h *entity.BranchHoliday) error
		UpdateBranchHoliday(ctx context.Context, h *entity.BranchHoliday) error
		DeleteBranchHoliday(ctx context.Context, id uint) error
		ExtendActiveTimeSubscriptions(ctx context.Context, branchID *uint, holidayDate time.Time) (int64, error)
	}

	scheduleExtRepository struct {
		db *gorm.DB
	}
)

func NewScheduleExtRepository(db *gorm.DB) ScheduleExtRepository {
	return &scheduleExtRepository{db: db}
}

// ---- Trainer Availability -------------------------------------------------------

func (r *scheduleExtRepository) ListTrainerAvailability(ctx context.Context, trainerID, branchID uint) ([]TrainerAvailabilityRow, error) {
	tx := postgres.Conn(ctx, r.db).Table("trainer_availabilities ta").
		Select("ta.*, COALESCE(b.name, '') AS branch_name").
		Joins("LEFT JOIN branches b ON b.id = ta.branch_id").
		Where("ta.trainer_id = ?", trainerID)
	if branchID > 0 {
		tx = tx.Where("ta.branch_id = ?", branchID)
	}
	var rows []TrainerAvailabilityRow
	err := tx.Order("ta.dow ASC, ta.start_time ASC, ta.id ASC").Scan(&rows).Error
	return rows, err
}

// ReplaceTrainerAvailability thay toàn bộ khung giờ rảnh của HLV TẠI MỘT chi nhánh (branchID bắt buộc > 0);
// lịch rảnh ở chi nhánh khác không bị đụng tới. Mọi slot được gán về branchID.
func (r *scheduleExtRepository) ReplaceTrainerAvailability(ctx context.Context, trainerID, branchID uint, slots []entity.TrainerAvailability) error {
	if branchID == 0 {
		return gorm.ErrMissingWhereClause
	}
	conn := postgres.Conn(ctx, r.db)
	if err := conn.Where("trainer_id = ? AND branch_id = ?", trainerID, branchID).Delete(&entity.TrainerAvailability{}).Error; err != nil {
		return err
	}
	for _, s := range slots {
		row := s
		row.ID = 0
		row.TrainerID = trainerID
		row.BranchID = branchID
		if err := conn.Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

// ---- Leave Requests -------------------------------------------------------------

const leaveRequestSelect = `lr.*, COALESCE(su.full_name, '') AS staff_name,
	COALESCE(b.name, '') AS branch_name, COALESCE(au.full_name, '') AS approver_name`

func (r *scheduleExtRepository) baseLeaveRequest(ctx context.Context) *gorm.DB {
	return postgres.Conn(ctx, r.db).Table("staff_leave_requests lr").
		Select(leaveRequestSelect).
		Joins("LEFT JOIN users su ON su.id = lr.staff_id").
		Joins("LEFT JOIN branches b ON b.id = lr.branch_id").
		Joins("LEFT JOIN users au ON au.id = lr.approved_by")
}

func (r *scheduleExtRepository) ListLeaveRequests(ctx context.Context, q dto.Query, staffID, branchID uint, status string) ([]LeaveRequestRow, int64, error) {
	tx := r.baseLeaveRequest(ctx)
	if staffID > 0 {
		tx = tx.Where("lr.staff_id = ?", staffID)
	}
	if branchID > 0 {
		tx = tx.Where("lr.branch_id = ?", branchID)
	}
	if status != "" {
		tx = tx.Where("lr.status = ?", status)
	}
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		tx = tx.Where("su.full_name ILIKE ? OR lr.reason ILIKE ?", like, like)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []LeaveRequestRow
	if err := q.Paginate(tx).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *scheduleExtRepository) FindLeaveRequestByID(ctx context.Context, id uint) (*LeaveRequestRow, error) {
	var row LeaveRequestRow
	res := r.baseLeaveRequest(ctx).Where("lr.id = ?", id).Limit(1).Find(&row)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}

func (r *scheduleExtRepository) CreateLeaveRequest(ctx context.Context, lr *entity.StaffLeaveRequest) error {
	return postgres.Conn(ctx, r.db).Create(lr).Error
}

func (r *scheduleExtRepository) UpdateLeaveRequest(ctx context.Context, lr *entity.StaffLeaveRequest) error {
	return postgres.Conn(ctx, r.db).Save(lr).Error
}

// ---- Impacts & Substitution -----------------------------------------------------

const impactedClassSelect = `cs.*, COALESCE(ct.name, '') AS class_type_name,
	COALESCE(b.name, '') AS branch_name, COALESCE(rm.name, '') AS room_name,
	COALESCE(tu.full_name, '') AS trainer_name, COALESCE(stu.full_name, '') AS substitute_trainer_name`

func (r *scheduleExtRepository) baseImpactedClass(ctx context.Context) *gorm.DB {
	return postgres.Conn(ctx, r.db).Table("class_schedules cs").
		Select(impactedClassSelect).
		Joins("LEFT JOIN class_types ct ON ct.id = cs.class_type_id").
		Joins("LEFT JOIN branches b ON b.id = cs.branch_id").
		Joins("LEFT JOIN rooms rm ON rm.id = cs.room_id").
		Joins("LEFT JOIN users tu ON tu.id = cs.trainer_id").
		Joins("LEFT JOIN users stu ON stu.id = cs.substitute_trainer_id")
}

func (r *scheduleExtRepository) ListImpactedClassSchedules(ctx context.Context, trainerID uint, fromAt, toAt time.Time) ([]ImpactedClassRow, error) {
	var rows []ImpactedClassRow
	err := r.baseImpactedClass(ctx).
		Where("cs.trainer_id = ? AND cs.status = 'SCHEDULED' AND cs.start_at < ? AND cs.end_at > ?", trainerID, toAt, fromAt).
		Order("cs.start_at ASC").
		Scan(&rows).Error
	return rows, err
}

func (r *scheduleExtRepository) FindClassScheduleByID(ctx context.Context, id uint) (*ImpactedClassRow, error) {
	var row ImpactedClassRow
	res := r.baseImpactedClass(ctx).Where("cs.id = ?", id).Limit(1).Find(&row)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}

func (r *scheduleExtRepository) SetClassSubstituteTrainer(ctx context.Context, scheduleID uint, subTrainerID *uint, note string, updatedBy *uint) error {
	updates := map[string]any{
		"substitute_trainer_id": subTrainerID,
		"updated_at":            time.Now(),
		"updated_by":            updatedBy,
	}
	if note != "" {
		updates["note"] = note
	}
	return postgres.Conn(ctx, r.db).Model(&entity.ClassSchedule{}).Where("id = ?", scheduleID).Updates(updates).Error
}

const impactedPTSelect = `ps.*, COALESCE(m.member_code, '') AS member_code,
	COALESCE(m.full_name, '') AS member_name, COALESCE(tu.full_name, '') AS trainer_name,
	COALESCE(otu.full_name, '') AS original_trainer_name, COALESCE(b.name, '') AS branch_name`

func (r *scheduleExtRepository) baseImpactedPT(ctx context.Context) *gorm.DB {
	return postgres.Conn(ctx, r.db).Table("pt_sessions ps").
		Select(impactedPTSelect).
		Joins("LEFT JOIN members m ON m.id = ps.member_id").
		Joins("LEFT JOIN users tu ON tu.id = ps.trainer_id").
		Joins("LEFT JOIN users otu ON otu.id = ps.original_trainer_id").
		Joins("LEFT JOIN branches b ON b.id = ps.branch_id")
}

func (r *scheduleExtRepository) ListImpactedPTSessions(ctx context.Context, trainerID uint, fromAt, toAt time.Time) ([]ImpactedPTSessionRow, error) {
	var rows []ImpactedPTSessionRow
	err := r.baseImpactedPT(ctx).
		Where("ps.trainer_id = ? AND ps.status = 'SCHEDULED' AND ps.start_at < ? AND ps.end_at > ?", trainerID, toAt, fromAt).
		Order("ps.start_at ASC").
		Scan(&rows).Error
	return rows, err
}

func (r *scheduleExtRepository) FindPTSessionByID(ctx context.Context, id uint) (*ImpactedPTSessionRow, error) {
	var row ImpactedPTSessionRow
	res := r.baseImpactedPT(ctx).Where("ps.id = ?", id).Limit(1).Find(&row)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}

func (r *scheduleExtRepository) ReassignPTSessionTrainer(ctx context.Context, sessionID, newTrainerID uint, origTrainerID uint) error {
	return postgres.Conn(ctx, r.db).Model(&entity.PTSession{}).
		Where("id = ?", sessionID).
		Updates(map[string]any{
			"trainer_id":          newTrainerID,
			"original_trainer_id": origTrainerID,
			"updated_at":          time.Now(),
		}).Error
}

func (r *scheduleExtRepository) TrainerEligible(ctx context.Context, trainerID, branchID uint) (bool, error) {
	var n int64
	err := postgres.Conn(ctx, r.db).Table("users u").
		Where("u.id = ? AND u.deleted_at IS NULL AND u.status = 'ACTIVE'", trainerID).
		Where(`(EXISTS (SELECT 1 FROM staff_profiles sp WHERE sp.user_id = u.id AND sp.position = 'TRAINER')
			OR EXISTS (SELECT 1 FROM user_roles ur JOIN roles r ON r.id = ur.role_id WHERE ur.user_id = u.id AND r.code = 'TRAINER'))`).
		Where(`EXISTS (SELECT 1 FROM user_roles ur JOIN roles r ON r.id = ur.role_id
			WHERE ur.user_id = u.id AND r.code <> 'MEMBER' AND (ur.branch_id = ? OR ur.branch_id IS NULL))`, branchID).
		Count(&n).Error
	return n > 0, err
}

func (r *scheduleExtRepository) TrainerBusy(ctx context.Context, trainerID uint, start, end time.Time, excludeClassID, excludePTID uint) (string, error) {
	conn := postgres.Conn(ctx, r.db)
	var n int64
	if err := conn.Table("pt_sessions").
		Where("trainer_id = ? AND status IN ('SCHEDULED', 'COMPLETED') AND start_at < ? AND end_at > ? AND id <> ?",
			trainerID, end, start, excludePTID).
		Count(&n).Error; err != nil {
		return "", err
	}
	if n > 0 {
		return "PT", nil
	}
	if err := conn.Table("class_schedules").
		Where("COALESCE(substitute_trainer_id, trainer_id) = ? AND status <> 'CANCELLED' AND start_at < ? AND end_at > ? AND id <> ?",
			trainerID, end, start, excludeClassID).
		Count(&n).Error; err != nil {
		return "", err
	}
	if n > 0 {
		return "CLASS", nil
	}
	return "", nil
}

func (r *scheduleExtRepository) ListScheduledClassesBetween(ctx context.Context, branchID *uint, fromAt, toAt time.Time) ([]ImpactedClassRow, error) {
	tx := r.baseImpactedClass(ctx).
		Where("cs.status = 'SCHEDULED' AND cs.start_at < ? AND cs.end_at > ?", toAt, fromAt)
	if branchID != nil && *branchID > 0 {
		tx = tx.Where("cs.branch_id = ?", *branchID)
	}
	var rows []ImpactedClassRow
	err := tx.Order("cs.start_at ASC").Limit(500).Scan(&rows).Error
	return rows, err
}

func (r *scheduleExtRepository) ListScheduledPTSessionsBetween(ctx context.Context, branchID *uint, fromAt, toAt time.Time) ([]ImpactedPTSessionRow, error) {
	tx := r.baseImpactedPT(ctx).
		Where("ps.status = 'SCHEDULED' AND ps.start_at < ? AND ps.end_at > ?", toAt, fromAt)
	if branchID != nil && *branchID > 0 {
		tx = tx.Where("ps.branch_id = ?", *branchID)
	}
	var rows []ImpactedPTSessionRow
	err := tx.Order("ps.start_at ASC").Limit(500).Scan(&rows).Error
	return rows, err
}

// ---- Branch Holidays ------------------------------------------------------------

const branchHolidaySelect = `bh.*, COALESCE(b.name, 'Toàn hệ thống') AS branch_name`

func (r *scheduleExtRepository) baseBranchHoliday(ctx context.Context) *gorm.DB {
	return postgres.Conn(ctx, r.db).Table("branch_holidays bh").
		Select(branchHolidaySelect).
		Joins("LEFT JOIN branches b ON b.id = bh.branch_id")
}

func (r *scheduleExtRepository) ListBranchHolidays(ctx context.Context, q dto.Query, branchID uint, year int) ([]BranchHolidayRow, int64, error) {
	tx := r.baseBranchHoliday(ctx)
	if branchID > 0 {
		tx = tx.Where("(bh.branch_id IS NULL OR bh.branch_id = ?)", branchID)
	}
	if year > 0 {
		tx = tx.Where("EXTRACT(YEAR FROM bh.date) = ?", year)
	}
	if q.Keyword != "" {
		tx = tx.Where("bh.name ILIKE ?", "%"+q.Keyword+"%")
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []BranchHolidayRow
	if err := q.Paginate(tx).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *scheduleExtRepository) ListUpcomingHolidays(ctx context.Context, branchID uint, fromDate time.Time) ([]BranchHolidayRow, error) {
	tx := r.baseBranchHoliday(ctx).Where("bh.date >= ?", fromDate.Format("2006-01-02"))
	if branchID > 0 {
		tx = tx.Where("(bh.branch_id IS NULL OR bh.branch_id = ?)", branchID)
	}
	var rows []BranchHolidayRow
	err := tx.Order("bh.date ASC, bh.id ASC").Limit(50).Scan(&rows).Error
	return rows, err
}

func (r *scheduleExtRepository) FindBranchHolidayByID(ctx context.Context, id uint) (*BranchHolidayRow, error) {
	var row BranchHolidayRow
	res := r.baseBranchHoliday(ctx).Where("bh.id = ?", id).Limit(1).Find(&row)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}

func (r *scheduleExtRepository) CreateBranchHoliday(ctx context.Context, h *entity.BranchHoliday) error {
	return postgres.Conn(ctx, r.db).Create(h).Error
}

// UpdateBranchHoliday ghi các trường có thể sửa (Select để lưu cả giá trị zero như closed=false, giờ NULL).
func (r *scheduleExtRepository) UpdateBranchHoliday(ctx context.Context, h *entity.BranchHoliday) error {
	return postgres.Conn(ctx, r.db).Model(h).
		Select("branch_id", "date", "name", "closed", "open_time", "close_time", "updated_at").
		Updates(h).Error
}

func (r *scheduleExtRepository) DeleteBranchHoliday(ctx context.Context, id uint) error {
	return postgres.Conn(ctx, r.db).Delete(&entity.BranchHoliday{}, id).Error
}

func (r *scheduleExtRepository) ExtendActiveTimeSubscriptions(ctx context.Context, branchID *uint, holidayDate time.Time) (int64, error) {
	conn := postgres.Conn(ctx, r.db)
	dStr := holidayDate.Format("2006-01-02")
	tx := conn.Model(&entity.Subscription{}).
		Where("status = 'ACTIVE' AND start_date <= ? AND end_date >= ?", dStr, dStr).
		Where("plan_id IN (SELECT id FROM membership_plans WHERE type = 'TIME')")
	if branchID != nil && *branchID > 0 {
		tx = tx.Where("branch_id = ?", *branchID)
	}
	res := tx.UpdateColumns(map[string]any{
		"end_date":   gorm.Expr("end_date + INTERVAL '1 day'"),
		"updated_at": time.Now(),
	})
	return res.RowsAffected, res.Error
}
