package repository

import (
	"context"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/storage/postgres"
)

type (
	// AuditLogRow: dòng audit_logs kèm thông tin người thao tác.
	AuditLogRow struct {
		ID         uint           `gorm:"column:id"`
		ActorID    *uint          `gorm:"column:actor_id"`
		ActorName  string         `gorm:"column:actor_name"`
		ActorEmail string         `gorm:"column:actor_email"`
		Action     string         `gorm:"column:action"`
		Entity     string         `gorm:"column:entity"`
		EntityID   *uint          `gorm:"column:entity_id"`
		Before     datatypes.JSON `gorm:"column:before"`
		After      datatypes.JSON `gorm:"column:after"`
		IP         string         `gorm:"column:ip"`
		CreatedAt  time.Time      `gorm:"column:created_at"`
	}

	AuditLogFilter struct {
		ActorID  uint
		Entity   string
		EntityID uint
		Action   string
		From     *time.Time
		To       *time.Time // không bao gồm
	}

	AuditLogRepository interface {
		List(ctx context.Context, f AuditLogFilter, q dto.Query) ([]AuditLogRow, int64, error)
		Get(ctx context.Context, id uint) (*AuditLogRow, error)
		Entities(ctx context.Context) ([]string, error)
		Actions(ctx context.Context) ([]string, error)
	}

	auditLogRepository struct {
		db *gorm.DB
	}
)

func NewAuditLogRepository(db *gorm.DB) AuditLogRepository {
	return &auditLogRepository{db: db}
}

func (r *auditLogRepository) base(ctx context.Context) *gorm.DB {
	return postgres.Conn(ctx, r.db).Table("audit_logs a").
		Joins("LEFT JOIN users u ON u.id = a.actor_id")
}

func (r *auditLogRepository) List(ctx context.Context, f AuditLogFilter, q dto.Query) ([]AuditLogRow, int64, error) {
	db := r.base(ctx)
	if f.ActorID > 0 {
		db = db.Where("a.actor_id = ?", f.ActorID)
	}
	if f.Entity != "" {
		db = db.Where("a.entity = ?", f.Entity)
	}
	if f.EntityID > 0 {
		db = db.Where("a.entity_id = ?", f.EntityID)
	}
	if f.Action != "" {
		db = db.Where("a.action = ?", f.Action)
	}
	if f.From != nil {
		db = db.Where("a.created_at >= ?", *f.From)
	}
	if f.To != nil {
		db = db.Where("a.created_at < ?", *f.To)
	}
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		db = db.Where("(u.full_name ILIKE ? OR u.email ILIKE ? OR a.entity ILIKE ? OR a.action ILIKE ?)", kw, kw, kw, kw)
	}
	var total int64
	if err := db.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []AuditLogRow
	// Danh sách không trả before/after (nặng) – xem chi tiết qua Get.
	err := db.Select(`a.id, a.actor_id, COALESCE(u.full_name, '') AS actor_name, COALESCE(u.email, '') AS actor_email,
		a.action, a.entity, a.entity_id, a.ip, a.created_at`).
		Scopes(q.Paginate).Order("a.id DESC").Scan(&rows).Error
	return rows, total, err
}

func (r *auditLogRepository) Get(ctx context.Context, id uint) (*AuditLogRow, error) {
	var rows []AuditLogRow
	err := r.base(ctx).Select(`a.id, a.actor_id, COALESCE(u.full_name, '') AS actor_name, COALESCE(u.email, '') AS actor_email,
		a.action, a.entity, a.entity_id, a.before, a.after, a.ip, a.created_at`).
		Where("a.id = ?", id).Limit(1).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &rows[0], nil
}

func (r *auditLogRepository) Entities(ctx context.Context) ([]string, error) {
	var out []string
	err := postgres.Conn(ctx, r.db).Raw("SELECT DISTINCT entity FROM audit_logs ORDER BY entity").Scan(&out).Error
	return out, err
}

func (r *auditLogRepository) Actions(ctx context.Context) ([]string, error) {
	var out []string
	err := postgres.Conn(ctx, r.db).Raw("SELECT DISTINCT action FROM audit_logs ORDER BY action").Scan(&out).Error
	return out, err
}
