package service

import (
	"context"
	"errors"

	"gorm.io/gorm"

	v1 "gymmaster-api/api/branch/v1"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
)

type (
	BranchService interface {
		List(ctx context.Context, req *v1.ListReq) (*v1.ListRes, error)
		Get(ctx context.Context, id uint) (*v1.Branch, error)
		Create(ctx context.Context, in v1.BranchInput) (*v1.Branch, error)
		Update(ctx context.Context, id uint, in v1.BranchInput) (*v1.Branch, error)
		Delete(ctx context.Context, id uint) error
	}

	branchService struct {
		branchRepo repository.BranchRepository
	}
)

func NewBranchService(branchRepo repository.BranchRepository) BranchService {
	return &branchService{branchRepo: branchRepo}
}

var branchOrderable = []string{"id", "code", "name", "status", "created_at"}

func (s *branchService) List(ctx context.Context, req *v1.ListReq) (*v1.ListRes, error) {
	q := req.PageReq.ToQuery(branchOrderable, "id")
	if req.Status != "" {
		q.Filters["status"] = req.Status
	}
	items, total, err := s.branchRepo.List(ctx, q)
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(items, toBranchRes), total, q), nil
}

func (s *branchService) Get(ctx context.Context, id uint) (*v1.Branch, error) {
	b, err := s.find(ctx, id)
	if err != nil {
		return nil, err
	}
	return toBranchRes(*b), nil
}

func (s *branchService) Create(ctx context.Context, in v1.BranchInput) (*v1.Branch, error) {
	if err := s.validate(ctx, in, 0); err != nil {
		return nil, err
	}
	b := &entity.Branch{}
	applyBranchInput(b, in)
	b.CreatedBy = reqctx.UserID(ctx)
	b.UpdatedBy = b.CreatedBy
	if err := s.branchRepo.Create(ctx, b); err != nil {
		return nil, mapDuplicate(err, "Mã chi nhánh đã tồn tại")
	}
	return toBranchRes(*b), nil
}

func (s *branchService) Update(ctx context.Context, id uint, in v1.BranchInput) (*v1.Branch, error) {
	b, err := s.find(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.validate(ctx, in, id); err != nil {
		return nil, err
	}
	applyBranchInput(b, in)
	b.UpdatedBy = reqctx.UserID(ctx)
	if err := s.branchRepo.Update(ctx, b); err != nil {
		return nil, mapDuplicate(err, "Mã chi nhánh đã tồn tại")
	}
	return s.Get(ctx, id)
}

func (s *branchService) Delete(ctx context.Context, id uint) error {
	if _, err := s.find(ctx, id); err != nil {
		return err
	}
	n, err := s.branchRepo.CountStaff(ctx, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return rescode.New(rescode.Conflict, "Không thể xóa chi nhánh đang có nhân viên làm việc")
	}
	return s.branchRepo.Delete(ctx, id, reqctx.UserID(ctx))
}

func (s *branchService) find(ctx context.Context, id uint) (*entity.Branch, error) {
	b, err := s.branchRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy chi nhánh")
		}
		return nil, err
	}
	return b, nil
}

// validate kiểm tra các ràng buộc nghiệp vụ mà tag `v` không diễn đạt được.
func (s *branchService) validate(ctx context.Context, in v1.BranchInput, excludeID uint) error {
	if in.CloseTime <= in.OpenTime { // so sánh chuỗi "HH:mm" đúng thứ tự thời gian
		return rescode.New(rescode.BadRequest, "Giờ đóng cửa phải sau giờ mở cửa")
	}
	exists, err := s.branchRepo.ExistsCode(ctx, in.Code, excludeID)
	if err != nil {
		return err
	}
	if exists {
		return rescode.New(rescode.Conflict, "Mã chi nhánh đã tồn tại")
	}
	return nil
}

func applyBranchInput(b *entity.Branch, in v1.BranchInput) {
	b.Code = in.Code
	b.Name = in.Name
	b.Address = in.Address
	b.Phone = in.Phone
	b.Email = in.Email
	b.OpenTime = in.OpenTime
	b.CloseTime = in.CloseTime
	b.Status = in.Status
}

func toBranchRes(b entity.Branch) *v1.Branch {
	return &v1.Branch{
		ID:        b.ID,
		Code:      b.Code,
		Name:      b.Name,
		Address:   b.Address,
		Phone:     b.Phone,
		Email:     b.Email,
		OpenTime:  b.OpenTime,
		CloseTime: b.CloseTime,
		Status:    b.Status,
		CreatedAt: b.CreatedAt,
		UpdatedAt: b.UpdatedAt,
	}
}
