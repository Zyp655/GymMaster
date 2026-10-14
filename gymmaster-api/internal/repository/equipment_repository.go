package repository

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/storage/postgres"
)

type (
	EquipmentRow struct {
		entity.Equipment
		BranchName string
		RoomName   string
	}

	EquipmentFilter struct {
		BranchIDs []uint
		RoomID    uint
		Category  string
		Status    string
		Search    string
	}

	MaintenanceLogRow struct {
		entity.MaintenanceLog
		EquipmentCode string
		EquipmentName string
		BranchID      uint
		BranchName    string
		ReporterName  string
	}

	MaintenanceLogFilter struct {
		BranchIDs   []uint
		EquipmentID uint
		Type        string
		Status      string
	}

	MaintenanceBranchCostRow struct {
		BranchID   uint
		BranchName string
		TotalCost  float64
		TotalLogs  int64
	}

	MaintenanceMonthCostRow struct {
		Month     string
		TotalCost float64
		TotalLogs int64
	}

	MaintenanceSummaryData struct {
		TotalCost float64
		TotalLogs int64
		ByBranch  []MaintenanceBranchCostRow
		ByMonth   []MaintenanceMonthCostRow
	}

	EquipmentRepository interface {
		ListEquipments(ctx context.Context, q dto.Query, f EquipmentFilter) ([]EquipmentRow, int64, error)
		FindEquipmentByID(ctx context.Context, id uint) (*EquipmentRow, error)
		ExistsEquipmentCode(ctx context.Context, code string, excludeID uint) (bool, error)
		CreateEquipment(ctx context.Context, e *entity.Equipment) error
		UpdateEquipment(ctx context.Context, e *entity.Equipment) error
		DeleteEquipment(ctx context.Context, id uint, deletedBy *uint) error

		ListMaintenanceLogs(ctx context.Context, q dto.Query, f MaintenanceLogFilter) ([]MaintenanceLogRow, int64, error)
		ListEquipmentLogs(ctx context.Context, equipmentID uint) ([]MaintenanceLogRow, error)
		FindMaintenanceLogByID(ctx context.Context, id uint) (*MaintenanceLogRow, error)
		CreateMaintenanceLog(ctx context.Context, l *entity.MaintenanceLog) error
		UpdateMaintenanceLog(ctx context.Context, l *entity.MaintenanceLog) error
		DeleteMaintenanceLog(ctx context.Context, id uint) error
		CountOpenLogsExcluding(ctx context.Context, equipmentID, excludeLogID uint) (int64, error)
		MaintenanceSummary(ctx context.Context, branchIDs []uint, from, to *time.Time) (*MaintenanceSummaryData, error)
		ListDueMaintenance(ctx context.Context, today time.Time) ([]EquipmentRow, error)
	}

	equipmentRepository struct {
		db *gorm.DB
	}
)

func NewEquipmentRepository(db *gorm.DB) EquipmentRepository {
	return &equipmentRepository{db: db}
}

const equipmentRowSelect = `eq.*, COALESCE(b.name, '') AS branch_name, COALESCE(rm.name, '') AS room_name`
const equipmentRowJoins = `LEFT JOIN branches b ON b.id = eq.branch_id
	LEFT JOIN rooms rm ON rm.id = eq.room_id`

func (r *equipmentRepository) baseEquipment(ctx context.Context) *gorm.DB {
	return postgres.Conn(ctx, r.db).Table("equipments eq").
		Select(equipmentRowSelect).
		Joins(equipmentRowJoins).
		Where("eq.deleted_at IS NULL")
}

func (r *equipmentRepository) ListEquipments(ctx context.Context, q dto.Query, f EquipmentFilter) ([]EquipmentRow, int64, error) {
	tx := r.baseEquipment(ctx)
	if f.BranchIDs != nil {
		tx = tx.Where("eq.branch_id IN ?", f.BranchIDs)
	}
	if f.RoomID > 0 {
		tx = tx.Where("eq.room_id = ?", f.RoomID)
	}
	if f.Category != "" {
		tx = tx.Where("eq.category = ?", f.Category)
	}
	if f.Status != "" {
		tx = tx.Where("eq.status = ?", f.Status)
	}
	kw := strings.TrimSpace(f.Search)
	if kw == "" {
		kw = q.Keyword
	}
	if kw != "" {
		like := "%" + kw + "%"
		tx = tx.Where("eq.code ILIKE ? OR eq.name ILIKE ? OR eq.note ILIKE ?", like, like, like)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []EquipmentRow
	if err := q.Paginate(tx).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *equipmentRepository) FindEquipmentByID(ctx context.Context, id uint) (*EquipmentRow, error) {
	var row EquipmentRow
	res := r.baseEquipment(ctx).Where("eq.id = ?", id).Limit(1).Find(&row)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}

func (r *equipmentRepository) ExistsEquipmentCode(ctx context.Context, code string, excludeID uint) (bool, error) {
	tx := postgres.Conn(ctx, r.db).Model(&entity.Equipment{}).Where("UPPER(code) = UPPER(?)", strings.TrimSpace(code))
	if excludeID > 0 {
		tx = tx.Where("id <> ?", excludeID)
	}
	var count int64
	if err := tx.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *equipmentRepository) CreateEquipment(ctx context.Context, e *entity.Equipment) error {
	return postgres.Conn(ctx, r.db).Create(e).Error
}

func (r *equipmentRepository) UpdateEquipment(ctx context.Context, e *entity.Equipment) error {
	return postgres.Conn(ctx, r.db).Save(e).Error
}

func (r *equipmentRepository) DeleteEquipment(ctx context.Context, id uint, deletedBy *uint) error {
	now := time.Now()
	return postgres.Conn(ctx, r.db).Model(&entity.Equipment{}).Where("id = ?", id).
		Updates(map[string]any{"deleted_at": now, "updated_by": deletedBy, "updated_at": now}).Error
}

const maintLogSelect = `ml.*, COALESCE(eq.code, '') AS equipment_code, COALESCE(eq.name, '') AS equipment_name,
	COALESCE(eq.branch_id, 0) AS branch_id, COALESCE(b.name, '') AS branch_name, COALESCE(u.full_name, '') AS reporter_name`

const maintLogJoins = `JOIN equipments eq ON eq.id = ml.equipment_id
	LEFT JOIN branches b ON b.id = eq.branch_id
	LEFT JOIN users u ON u.id = ml.reported_by`

func (r *equipmentRepository) baseMaintLog(ctx context.Context) *gorm.DB {
	return postgres.Conn(ctx, r.db).Table("maintenance_logs ml").Select(maintLogSelect).Joins(maintLogJoins)
}

func (r *equipmentRepository) ListMaintenanceLogs(ctx context.Context, q dto.Query, f MaintenanceLogFilter) ([]MaintenanceLogRow, int64, error) {
	tx := r.baseMaintLog(ctx)
	if f.BranchIDs != nil {
		tx = tx.Where("eq.branch_id IN ?", f.BranchIDs)
	}
	if f.EquipmentID > 0 {
		tx = tx.Where("ml.equipment_id = ?", f.EquipmentID)
	}
	if f.Type != "" {
		tx = tx.Where("ml.type = ?", f.Type)
	}
	if f.Status != "" {
		tx = tx.Where("ml.status = ?", f.Status)
	}
	if q.Keyword != "" {
		like := "%" + strings.TrimSpace(q.Keyword) + "%"
		tx = tx.Where("ml.description ILIKE ? OR eq.name ILIKE ? OR eq.code ILIKE ?", like, like, like)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []MaintenanceLogRow
	if err := q.Paginate(tx).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *equipmentRepository) ListEquipmentLogs(ctx context.Context, equipmentID uint) ([]MaintenanceLogRow, error) {
	var rows []MaintenanceLogRow
	err := r.baseMaintLog(ctx).Where("ml.equipment_id = ?", equipmentID).Order("ml.created_at DESC, ml.id DESC").Scan(&rows).Error
	return rows, err
}

func (r *equipmentRepository) FindMaintenanceLogByID(ctx context.Context, id uint) (*MaintenanceLogRow, error) {
	var row MaintenanceLogRow
	res := r.baseMaintLog(ctx).Where("ml.id = ?", id).Limit(1).Find(&row)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}

func (r *equipmentRepository) CreateMaintenanceLog(ctx context.Context, l *entity.MaintenanceLog) error {
	return postgres.Conn(ctx, r.db).Create(l).Error
}

func (r *equipmentRepository) UpdateMaintenanceLog(ctx context.Context, l *entity.MaintenanceLog) error {
	return postgres.Conn(ctx, r.db).Save(l).Error
}

func (r *equipmentRepository) DeleteMaintenanceLog(ctx context.Context, id uint) error {
	return postgres.Conn(ctx, r.db).Delete(&entity.MaintenanceLog{}, id).Error
}

func (r *equipmentRepository) CountOpenLogsExcluding(ctx context.Context, equipmentID, excludeLogID uint) (int64, error) {
	tx := postgres.Conn(ctx, r.db).Model(&entity.MaintenanceLog{}).
		Where("equipment_id = ? AND status IN ?", equipmentID, []string{entity.MaintenanceStatusOpen, entity.MaintenanceStatusInProgress})
	if excludeLogID > 0 {
		tx = tx.Where("id <> ?", excludeLogID)
	}
	var count int64
	err := tx.Count(&count).Error
	return count, err
}

func (r *equipmentRepository) MaintenanceSummary(ctx context.Context, branchIDs []uint, from, to *time.Time) (*MaintenanceSummaryData, error) {
	base := func() *gorm.DB {
		tx := postgres.Conn(ctx, r.db).Table("maintenance_logs ml").
			Joins("JOIN equipments eq ON eq.id = ml.equipment_id").
			Joins("LEFT JOIN branches b ON b.id = eq.branch_id")
		if branchIDs != nil {
			tx = tx.Where("eq.branch_id IN ?", branchIDs)
		}
		if from != nil {
			tx = tx.Where("ml.created_at >= ?", *from)
		}
		if to != nil {
			tx = tx.Where("ml.created_at < ?", *to)
		}
		return tx
	}

	out := &MaintenanceSummaryData{}
	if err := base().Select("COALESCE(SUM(ml.cost), 0) AS total_cost, COUNT(*) AS total_logs").Scan(out).Error; err != nil {
		return nil, err
	}
	if err := base().
		Select("eq.branch_id AS branch_id, COALESCE(b.name, '') AS branch_name, COALESCE(SUM(ml.cost), 0) AS total_cost, COUNT(*) AS total_logs").
		Group("eq.branch_id, b.name").
		Order("eq.branch_id ASC").
		Scan(&out.ByBranch).Error; err != nil {
		return nil, err
	}
	if err := base().
		Select("TO_CHAR(ml.created_at AT TIME ZONE 'Asia/Ho_Chi_Minh', 'YYYY-MM') AS month, COALESCE(SUM(ml.cost), 0) AS total_cost, COUNT(*) AS total_logs").
		Group("month").
		Order("month ASC").
		Scan(&out.ByMonth).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *equipmentRepository) ListDueMaintenance(ctx context.Context, today time.Time) ([]EquipmentRow, error) {
	dateStr := today.Format(consts.DateLayout)
	var rows []EquipmentRow
	err := r.baseEquipment(ctx).
		Where("eq.status = ? AND eq.next_maintenance_date IS NOT NULL AND eq.next_maintenance_date <= ?", entity.EquipmentStatusActive, dateStr).
		Order("eq.branch_id ASC, eq.next_maintenance_date ASC, eq.id ASC").
		Scan(&rows).Error
	return rows, err
}
