package service

import (
	"context"
	"encoding/json"

	"github.com/gogf/gf/v2/frame/g"
	"gorm.io/gorm"

	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/clientip"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/storage/postgres"
)

// Hành động audit thường dùng.
const (
	AuditCreate = "CREATE"
	AuditUpdate = "UPDATE"
	AuditDelete = "DELETE"
)

// AuditService ghi nhật ký thao tác quan trọng. Ghi trong cùng transaction của nghiệp vụ
// (nếu ctx đang có transaction) để nhật ký và dữ liệu luôn nhất quán.
// Lỗi ghi audit chỉ được log, không làm hỏng nghiệp vụ chính.
type AuditService interface {
	Log(ctx context.Context, action, entityName string, entityID uint, before, after any)
}

type auditService struct {
	db *gorm.DB
}

func NewAuditService(db *gorm.DB) AuditService {
	return &auditService{db: db}
}

func (s *auditService) Log(ctx context.Context, action, entityName string, entityID uint, before, after any) {
	row := &entity.AuditLog{
		ActorID: reqctx.UserID(ctx),
		Action:  action,
		Entity:  entityName,
		Before:  toJSON(before),
		After:   toJSON(after),
	}
	if entityID > 0 {
		row.EntityID = &entityID
	}
	if r := g.RequestFromCtx(ctx); r != nil {
		row.IP = clientip.Real(r)
	}
	if err := postgres.Conn(ctx, s.db).Create(row).Error; err != nil {
		g.Log().Warningf(ctx, "audit log: %v", err)
	}
}

func toJSON(v any) []byte {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return b
}
