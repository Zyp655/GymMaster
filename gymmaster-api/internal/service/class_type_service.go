package service

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"

	v1 "gymmaster-api/api/class/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
)

type (
	ClassTypeService interface {
		List(ctx context.Context, req *v1.ListTypeReq) (*v1.ListTypeRes, error)
		Options(ctx context.Context) ([]*v1.ClassTypeOption, error)
		Get(ctx context.Context, id uint) (*v1.ClassType, error)
		Create(ctx context.Context, in v1.ClassTypeInput) (*v1.ClassType, error)
		Update(ctx context.Context, id uint, in v1.ClassTypeInput) (*v1.ClassType, error)
		Delete(ctx context.Context, id uint) error
	}

	classTypeService struct {
		repo  repository.ClassTypeRepository
		audit AuditService
		now   func() time.Time
	}
)

const classTypeCodeConflictMsg = "Mã loại lớp đã tồn tại"

var classTypeOrderable = []string{"id", "code", "name", "level", "status", "created_at"}

func NewClassTypeService(repo repository.ClassTypeRepository, audit AuditService) ClassTypeService {
	return &classTypeService{repo: repo, audit: audit, now: time.Now}
}

func (s *classTypeService) List(ctx context.Context, req *v1.ListTypeReq) (*v1.ListTypeRes, error) {
	q := req.PageReq.ToQuery(classTypeOrderable, "id")
	if req.Status != "" {
		q.Filters["status"] = req.Status
	}
	if req.Level != "" {
		q.Filters["level"] = req.Level
	}
	items, total, err := s.repo.List(ctx, q)
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(items, classTypeToRes), total, q), nil
}

func (s *classTypeService) Options(ctx context.Context) ([]*v1.ClassTypeOption, error) {
	items, err := s.repo.ActiveOptions(ctx)
	if err != nil {
		return nil, err
	}
	return dto.Map(items, func(t entity.ClassType) *v1.ClassTypeOption {
		return &v1.ClassTypeOption{
			ID: t.ID, Code: t.Code, Name: t.Name, Color: t.Color, Level: t.Level,
			DefaultDurationMin: t.DefaultDurationMin, DefaultCapacity: t.DefaultCapacity,
		}
	}), nil
}

func (s *classTypeService) Get(ctx context.Context, id uint) (*v1.ClassType, error) {
	t, err := s.find(ctx, id)
	if err != nil {
		return nil, err
	}
	return classTypeToRes(*t), nil
}

func (s *classTypeService) Create(ctx context.Context, in v1.ClassTypeInput) (*v1.ClassType, error) {
	in = classTypeNormalize(in)
	if !classTypeColorRe.MatchString(in.Color) {
		return nil, rescode.New(rescode.BadRequest, classTypeColorMsg)
	}
	if err := s.checkCode(ctx, in.Code, 0); err != nil {
		return nil, err
	}
	t := &entity.ClassType{}
	classTypeApply(t, in)
	t.CreatedBy = reqctx.UserID(ctx)
	t.UpdatedBy = t.CreatedBy
	if err := s.repo.Create(ctx, t); err != nil {
		return nil, mapDuplicate(err, classTypeCodeConflictMsg)
	}
	s.audit.Log(ctx, AuditCreate, "class_type", t.ID, nil, t)
	return s.Get(ctx, t.ID)
}

func (s *classTypeService) Update(ctx context.Context, id uint, in v1.ClassTypeInput) (*v1.ClassType, error) {
	t, err := s.find(ctx, id)
	if err != nil {
		return nil, err
	}
	before := *t
	in = classTypeNormalize(in)
	if !classTypeColorRe.MatchString(in.Color) {
		return nil, rescode.New(rescode.BadRequest, classTypeColorMsg)
	}
	if err := s.checkCode(ctx, in.Code, id); err != nil {
		return nil, err
	}
	classTypeApply(t, in)
	t.UpdatedBy = reqctx.UserID(ctx)
	if err := s.repo.Update(ctx, t); err != nil {
		return nil, mapDuplicate(err, classTypeCodeConflictMsg)
	}
	s.audit.Log(ctx, AuditUpdate, "class_type", id, before, t)
	return s.Get(ctx, id)
}

func (s *classTypeService) Delete(ctx context.Context, id uint) error {
	t, err := s.find(ctx, id)
	if err != nil {
		return err
	}
	n, err := s.repo.CountFutureSchedules(ctx, id, s.now())
	if err != nil {
		return err
	}
	if n > 0 {
		return rescode.New(rescode.Conflict, "Không thể xóa: loại lớp còn lịch sắp diễn ra")
	}
	if err := s.repo.Delete(ctx, id, reqctx.UserID(ctx)); err != nil {
		return err
	}
	s.audit.Log(ctx, AuditDelete, "class_type", id, t, nil)
	return nil
}

func (s *classTypeService) find(ctx context.Context, id uint) (*entity.ClassType, error) {
	t, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy loại lớp")
		}
		return nil, err
	}
	return t, nil
}

func (s *classTypeService) checkCode(ctx context.Context, code string, excludeID uint) error {
	exists, err := s.repo.ExistsCode(ctx, code, excludeID)
	if err != nil {
		return err
	}
	if exists {
		return rescode.New(rescode.Conflict, classTypeCodeConflictMsg)
	}
	return nil
}

// classTypeColorRe: màu hiển thị dạng #RRGGBB.
var classTypeColorRe = regexp.MustCompile(`^#[0-9A-F]{6}$`)

const classTypeColorMsg = "Màu phải có dạng #RRGGBB (ví dụ #F97316)"

func classTypeNormalize(in v1.ClassTypeInput) v1.ClassTypeInput {
	in.Code = strings.ToUpper(strings.TrimSpace(in.Code))
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)
	in.Color = strings.ToUpper(strings.TrimSpace(in.Color))
	if in.Color == "" {
		in.Color = "#F97316"
	}
	if in.DefaultDurationMin == 0 {
		in.DefaultDurationMin = 60
	}
	if in.DefaultCapacity == 0 {
		in.DefaultCapacity = 20
	}
	if in.Level == "" {
		in.Level = entity.ClassLevelAll
	}
	if in.Status == "" {
		in.Status = consts.StatusActive
	}
	return in
}

func classTypeApply(t *entity.ClassType, in v1.ClassTypeInput) {
	t.Code = in.Code
	t.Name = in.Name
	t.Description = in.Description
	t.Color = in.Color
	t.DefaultDurationMin = in.DefaultDurationMin
	t.DefaultCapacity = in.DefaultCapacity
	t.Level = in.Level
	t.Status = in.Status
}

func classTypeToRes(t entity.ClassType) *v1.ClassType {
	return &v1.ClassType{
		ID: t.ID, Code: t.Code, Name: t.Name, Description: t.Description, Color: t.Color,
		DefaultDurationMin: t.DefaultDurationMin, DefaultCapacity: t.DefaultCapacity,
		Level: t.Level, Status: t.Status, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}
}
