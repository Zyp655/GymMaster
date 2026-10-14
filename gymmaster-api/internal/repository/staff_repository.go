package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/storage/postgres"
)

// StaffFilter là bộ lọc danh sách nhân viên đã được service chuẩn hóa.
type StaffFilter struct {
	// BranchIDs nil = không lọc; khác nil = chỉ nhân viên có vai trò tại một trong các chi nhánh này.
	BranchIDs []uint
	Position  string
	Status    string
}

type (
	StaffRepository interface {
		List(ctx context.Context, q dto.Query, f StaffFilter) ([]entity.StaffAccount, int64, error)
		// FindByID tìm nhân viên (kèm vai trò, hồ sơ). Người dùng chỉ có vai trò MEMBER coi như không tồn tại.
		FindByID(ctx context.Context, id uint) (*entity.StaffAccount, error)
		EmailExists(ctx context.Context, email string, excludeID uint) (bool, error)
		NextStaffCode(ctx context.Context) (string, error)
		CreateUser(ctx context.Context, u *entity.User) error
		UpdateUser(ctx context.Context, u *entity.User) error
		SaveProfile(ctx context.Context, p *entity.StaffProfile) error
		SaveTrainerProfile(ctx context.Context, p *entity.TrainerProfile) error
		DeleteTrainerProfile(ctx context.Context, userID uint) error
		// ReplaceRoles xóa toàn bộ vai trò hiện có của user và gán danh sách mới.
		ReplaceRoles(ctx context.Context, userID uint, roles []entity.UserRole) error
		UpdateStatus(ctx context.Context, id uint, status string, by *uint) error
		UpdatePassword(ctx context.Context, id uint, passwordHash string, by *uint) error
		SoftDelete(ctx context.Context, id uint, by *uint) error
		// CountFutureAssignments đếm lịch lớp và buổi PT SCHEDULED chưa diễn ra mà nhân viên phụ trách.
		CountFutureAssignments(ctx context.Context, id uint, now time.Time) (classes int64, ptSessions int64, err error)
		DeleteFutureShifts(ctx context.Context, id uint, now time.Time) error
		// BranchNames trả tên các chi nhánh (chưa xóa) theo id.
		BranchNames(ctx context.Context, ids []uint) (map[uint]string, error)
		ActiveOptions(ctx context.Context, branchIDs []uint, trainersOnly bool) ([]entity.StaffAccount, error)
	}

	staffRepository struct {
		db *gorm.DB
	}
)

func NewStaffRepository(db *gorm.DB) StaffRepository {
	return &staffRepository{db: db}
}

// staffScope: người dùng có hồ sơ nhân viên hoặc có ít nhất một vai trò khác MEMBER.
const staffScopeSQL = `(EXISTS (SELECT 1 FROM staff_profiles sp0 WHERE sp0.user_id = users.id)
 OR EXISTS (SELECT 1 FROM user_roles ur0 JOIN roles r0 ON r0.id = ur0.role_id WHERE ur0.user_id = users.id AND r0.code <> 'MEMBER'))`

func (r *staffRepository) base(ctx context.Context) *gorm.DB {
	return postgres.Conn(ctx, r.db).Model(&entity.StaffAccount{}).Where(staffScopeSQL)
}

func (r *staffRepository) preload(tx *gorm.DB) *gorm.DB {
	return tx.Preload("UserRoles", func(db *gorm.DB) *gorm.DB { return db.Order("user_roles.id") }).
		Preload("UserRoles.Role").Preload("StaffProfile").Preload("TrainerProfile")
}

func staffWithBranches(tx *gorm.DB, branchIDs []uint) *gorm.DB {
	if branchIDs == nil {
		return tx
	}
	return tx.Where("EXISTS (SELECT 1 FROM user_roles urb WHERE urb.user_id = users.id AND urb.branch_id IN ?)", branchIDs)
}

func (r *staffRepository) List(ctx context.Context, q dto.Query, f StaffFilter) ([]entity.StaffAccount, int64, error) {
	tx := r.base(ctx).Joins("LEFT JOIN staff_profiles sp ON sp.user_id = users.id")
	tx = staffWithBranches(tx, f.BranchIDs)
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		tx = tx.Where("users.full_name ILIKE ? OR users.email ILIKE ? OR users.phone ILIKE ? OR sp.staff_code ILIKE ?", kw, kw, kw, kw)
	}
	if f.Position != "" {
		tx = tx.Where("sp.position = ?", f.Position)
	}
	if f.Status != "" {
		tx = tx.Where("users.status = ?", f.Status)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []entity.StaffAccount
	err := r.preload(tx.Select("users.*")).Scopes(q.Paginate).Find(&items).Error
	return items, total, err
}

func (r *staffRepository) FindByID(ctx context.Context, id uint) (*entity.StaffAccount, error) {
	var u entity.StaffAccount
	if err := r.preload(r.base(ctx)).Where("users.id = ?", id).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *staffRepository) EmailExists(ctx context.Context, email string, excludeID uint) (bool, error) {
	var n int64
	err := postgres.Conn(ctx, r.db).Model(&entity.User{}).
		Where("lower(email) = lower(?) AND id <> ?", email, excludeID).Count(&n).Error
	return n > 0, err
}

func (r *staffRepository) NextStaffCode(ctx context.Context) (string, error) {
	var code string
	err := postgres.Conn(ctx, r.db).Raw("SELECT 'NV' || lpad(nextval('staff_code_seq')::text, 4, '0')").Scan(&code).Error
	return code, err
}

func (r *staffRepository) CreateUser(ctx context.Context, u *entity.User) error {
	return postgres.Conn(ctx, r.db).Omit("UserRoles").Create(u).Error
}

func (r *staffRepository) UpdateUser(ctx context.Context, u *entity.User) error {
	return postgres.Conn(ctx, r.db).Model(u).Omit("UserRoles").
		Select("email", "phone", "full_name", "updated_by").Updates(u).Error
}

func (r *staffRepository) SaveProfile(ctx context.Context, p *entity.StaffProfile) error {
	return postgres.Conn(ctx, r.db).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"position", "gender", "dob", "id_card", "address",
			"hire_date", "base_salary", "commission_rate", "note"}),
	}).Create(p).Error
}

func (r *staffRepository) SaveTrainerProfile(ctx context.Context, p *entity.TrainerProfile) error {
	return postgres.Conn(ctx, r.db).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"level", "specialties", "bio", "certifications", "years_experience"}),
	}).Create(p).Error
}

func (r *staffRepository) DeleteTrainerProfile(ctx context.Context, userID uint) error {
	return postgres.Conn(ctx, r.db).Where("user_id = ?", userID).Delete(&entity.TrainerProfile{}).Error
}

func (r *staffRepository) ReplaceRoles(ctx context.Context, userID uint, roles []entity.UserRole) error {
	db := postgres.Conn(ctx, r.db)
	if err := db.Where("user_id = ?", userID).Delete(&entity.UserRole{}).Error; err != nil {
		return err
	}
	for i := range roles {
		roles[i].ID = 0
		roles[i].UserID = userID
		if err := db.Omit("Role").Create(&roles[i]).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *staffRepository) UpdateStatus(ctx context.Context, id uint, status string, by *uint) error {
	return postgres.Conn(ctx, r.db).Model(&entity.User{}).Where("id = ?", id).
		Updates(map[string]any{"status": status, "updated_by": by}).Error
}

func (r *staffRepository) UpdatePassword(ctx context.Context, id uint, passwordHash string, by *uint) error {
	return postgres.Conn(ctx, r.db).Model(&entity.User{}).Where("id = ?", id).
		Updates(map[string]any{"password_hash": passwordHash, "updated_by": by}).Error
}

func (r *staffRepository) SoftDelete(ctx context.Context, id uint, by *uint) error {
	db := postgres.Conn(ctx, r.db)
	if err := db.Model(&entity.User{}).Where("id = ?", id).UpdateColumn("updated_by", by).Error; err != nil {
		return err
	}
	return db.Delete(&entity.User{}, id).Error
}

func (r *staffRepository) CountFutureAssignments(ctx context.Context, id uint, now time.Time) (int64, int64, error) {
	db := postgres.Conn(ctx, r.db)
	var classes, pts int64
	if err := db.Table("class_schedules").
		Where("trainer_id = ? AND status = 'SCHEDULED' AND start_at > ?", id, now).Count(&classes).Error; err != nil {
		return 0, 0, err
	}
	if err := db.Table("pt_sessions").
		Where("trainer_id = ? AND status = 'SCHEDULED' AND start_at > ?", id, now).Count(&pts).Error; err != nil {
		return 0, 0, err
	}
	return classes, pts, nil
}

func (r *staffRepository) DeleteFutureShifts(ctx context.Context, id uint, now time.Time) error {
	return postgres.Conn(ctx, r.db).Where("staff_id = ? AND start_at > ?", id, now).Delete(&entity.Shift{}).Error
}

func (r *staffRepository) BranchNames(ctx context.Context, ids []uint) (map[uint]string, error) {
	out := map[uint]string{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []entity.Branch
	if err := postgres.Conn(ctx, r.db).Select("id", "name").Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, b := range rows {
		out[b.ID] = b.Name
	}
	return out, nil
}

func (r *staffRepository) ActiveOptions(ctx context.Context, branchIDs []uint, trainersOnly bool) ([]entity.StaffAccount, error) {
	tx := r.base(ctx).Where("users.status = 'ACTIVE'")
	tx = staffWithBranches(tx, branchIDs)
	if trainersOnly {
		tx = tx.Where("EXISTS (SELECT 1 FROM staff_profiles spt WHERE spt.user_id = users.id AND spt.position = 'TRAINER')")
	} else {
		tx = tx.Where("EXISTS (SELECT 1 FROM staff_profiles spt WHERE spt.user_id = users.id)")
	}
	var items []entity.StaffAccount
	err := tx.Preload("StaffProfile").Preload("TrainerProfile").Order("users.full_name").Limit(500).Find(&items).Error
	return items, err
}
