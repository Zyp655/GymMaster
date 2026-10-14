package service

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	v1 "gymmaster-api/api/ptpackage/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
)

type (
	// PTPackageService quản lý danh mục gói huấn luyện cá nhân.
	PTPackageService interface {
		List(ctx context.Context, req *v1.ListReq) (*v1.ListRes, error)
		// Options trả các gói PT đang bán (ACTIVE) cho dropdown bán PT.
		Options(ctx context.Context) ([]*v1.PTPackage, error)
		Get(ctx context.Context, id uint) (*v1.PTPackage, error)
		Create(ctx context.Context, in v1.PTPackageInput) (*v1.PTPackage, error)
		Update(ctx context.Context, id uint, in v1.PTPackageInput) (*v1.PTPackage, error)
		UpdateStatus(ctx context.Context, id uint, status string) (*v1.PTPackage, error)
		Delete(ctx context.Context, id uint) error
	}

	ptPackageService struct {
		repo  repository.PTPackageRepository
		audit AuditService
	}
)

func NewPTPackageService(repo repository.PTPackageRepository, audit AuditService) PTPackageService {
	return &ptPackageService{repo: repo, audit: audit}
}

const auditEntityPTPackage = "pt_package"

var ptPackageOrderable = []string{"id", "code", "name", "session_count", "validity_days", "price", "status", "created_at"}

func (s *ptPackageService) List(ctx context.Context, req *v1.ListReq) (*v1.ListRes, error) {
	q := req.PageReq.ToQuery(ptPackageOrderable, "price")
	if req.OrderBy == "" {
		q.OrderDir = "ASC"
	}
	if req.Status != "" {
		q.Filters["status"] = req.Status
	}
	items, total, err := s.repo.List(ctx, q)
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(items, toPTPackageRes), total, q), nil
}

func (s *ptPackageService) Options(ctx context.Context) ([]*v1.PTPackage, error) {
	items, err := s.repo.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	return dto.Map(items, toPTPackageRes), nil
}

func (s *ptPackageService) Get(ctx context.Context, id uint) (*v1.PTPackage, error) {
	p, err := s.find(ctx, id)
	if err != nil {
		return nil, err
	}
	return toPTPackageRes(*p), nil
}

func (s *ptPackageService) Create(ctx context.Context, in v1.PTPackageInput) (*v1.PTPackage, error) {
	in, err := normalizePTPackageInput(in)
	if err != nil {
		return nil, err
	}
	if err := s.checkCode(ctx, in.Code, 0); err != nil {
		return nil, err
	}
	p := &entity.PTPackage{}
	applyPTPackageInput(p, in)
	p.CreatedBy = reqctx.UserID(ctx)
	p.UpdatedBy = p.CreatedBy
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, mapDuplicate(err, "Mã gói PT đã tồn tại")
	}
	res := toPTPackageRes(*p)
	auditCatalog(ctx, s.audit, AuditCreate, auditEntityPTPackage, p.ID, nil, res)
	return res, nil
}

func (s *ptPackageService) Update(ctx context.Context, id uint, in v1.PTPackageInput) (*v1.PTPackage, error) {
	p, err := s.find(ctx, id)
	if err != nil {
		return nil, err
	}
	in, err = normalizePTPackageInput(in)
	if err != nil {
		return nil, err
	}
	if err := s.checkCode(ctx, in.Code, id); err != nil {
		return nil, err
	}
	before := toPTPackageRes(*p)
	applyPTPackageInput(p, in)
	p.UpdatedBy = reqctx.UserID(ctx)
	if err := s.repo.Update(ctx, p); err != nil {
		return nil, mapDuplicate(err, "Mã gói PT đã tồn tại")
	}
	res, err := s.Get(ctx, id)
	if err == nil {
		auditCatalog(ctx, s.audit, AuditUpdate, auditEntityPTPackage, id, before, res)
	}
	return res, err
}

func (s *ptPackageService) UpdateStatus(ctx context.Context, id uint, status string) (*v1.PTPackage, error) {
	p, err := s.find(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.repo.UpdateStatus(ctx, id, status, reqctx.UserID(ctx)); err != nil {
		return nil, err
	}
	auditCatalog(ctx, s.audit, AuditUpdate, auditEntityPTPackage, id, map[string]any{"status": p.Status}, map[string]any{"status": status})
	return s.Get(ctx, id)
}

func (s *ptPackageService) Delete(ctx context.Context, id uint) error {
	p, err := s.find(ctx, id)
	if err != nil {
		return err
	}
	n, err := s.repo.CountContracts(ctx, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return rescode.New(rescode.Conflict, "Gói PT đã có hợp đồng nên không thể xóa, hãy chuyển sang trạng thái Ngừng bán")
	}
	if err := s.repo.Delete(ctx, id, reqctx.UserID(ctx)); err != nil {
		return err
	}
	auditCatalog(ctx, s.audit, AuditDelete, auditEntityPTPackage, id, toPTPackageRes(*p), nil)
	return nil
}

func (s *ptPackageService) find(ctx context.Context, id uint) (*entity.PTPackage, error) {
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy gói PT")
		}
		return nil, err
	}
	return p, nil
}

func (s *ptPackageService) checkCode(ctx context.Context, code string, excludeID uint) error {
	exists, err := s.repo.ExistsCode(ctx, code, excludeID)
	if err != nil {
		return err
	}
	if exists {
		return rescode.New(rescode.Conflict, "Mã gói PT đã tồn tại")
	}
	return nil
}

// normalizePTPackageInput kiểm tra ràng buộc (khớp CHECK của migration 000004) và chuẩn hóa.
func normalizePTPackageInput(in v1.PTPackageInput) (v1.PTPackageInput, error) {
	in.Code = strings.ToUpper(strings.TrimSpace(in.Code))
	in.Name = strings.TrimSpace(in.Name)
	if in.Status == "" {
		in.Status = consts.StatusActive
	}
	if in.SessionMinutes == 0 {
		in.SessionMinutes = 60
	}
	if in.SessionCount <= 0 {
		return in, rescode.New(rescode.BadRequest, "Số buổi phải lớn hơn 0")
	}
	if in.SessionMinutes < 15 || in.SessionMinutes > 240 {
		return in, rescode.New(rescode.BadRequest, "Thời lượng mỗi buổi phải từ 15 đến 240 phút")
	}
	if in.ValidityDays <= 0 {
		return in, rescode.New(rescode.BadRequest, "Thời hạn sử dụng phải lớn hơn 0 ngày")
	}
	if in.Price < 0 {
		return in, rescode.New(rescode.BadRequest, "Giá gói không được âm")
	}
	in.Price = roundMoneyVND(in.Price)
	return in, nil
}

func applyPTPackageInput(p *entity.PTPackage, in v1.PTPackageInput) {
	p.Code = in.Code
	p.Name = in.Name
	p.Description = in.Description
	p.SessionCount = in.SessionCount
	p.SessionMinutes = in.SessionMinutes
	p.ValidityDays = in.ValidityDays
	p.Price = in.Price
	p.Status = in.Status
}

func toPTPackageRes(p entity.PTPackage) *v1.PTPackage {
	var perSession float64
	if p.SessionCount > 0 {
		perSession = roundMoneyVND(p.Price / float64(p.SessionCount))
	}
	return &v1.PTPackage{
		ID:              p.ID,
		Code:            p.Code,
		Name:            p.Name,
		Description:     p.Description,
		SessionCount:    p.SessionCount,
		SessionMinutes:  p.SessionMinutes,
		ValidityDays:    p.ValidityDays,
		Price:           p.Price,
		PricePerSession: perSession,
		Status:          p.Status,
		CreatedAt:       p.CreatedAt,
		UpdatedAt:       p.UpdatedAt,
	}
}
