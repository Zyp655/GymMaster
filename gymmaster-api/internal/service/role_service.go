package service

import (
	"context"
	"errors"
	"slices"

	"gorm.io/gorm"

	v1 "gymmaster-api/api/role/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/storage/postgres"
)

type (
	RoleService interface {
		ListPermissions(ctx context.Context) ([]*v1.Permission, error)
		List(ctx context.Context) ([]*v1.Role, error)
		Get(ctx context.Context, id uint) (*v1.Role, error)
		Create(ctx context.Context, req *v1.CreateReq) (*v1.Role, error)
		Update(ctx context.Context, req *v1.UpdateReq) (*v1.Role, error)
		Delete(ctx context.Context, id uint) error
	}

	roleService struct {
		roleRepo repository.RoleRepository
		tx       postgres.TxManager
	}
)

func NewRoleService(roleRepo repository.RoleRepository, tx postgres.TxManager) RoleService {
	return &roleService{roleRepo: roleRepo, tx: tx}
}

func (s *roleService) ListPermissions(ctx context.Context) ([]*v1.Permission, error) {
	perms, err := s.roleRepo.ListPermissions(ctx)
	if err != nil {
		return nil, err
	}
	return dto.Map(perms, toPermissionRes), nil
}

func (s *roleService) List(ctx context.Context) ([]*v1.Role, error) {
	roles, err := s.roleRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	return dto.Map(roles, toRoleRes), nil
}

func (s *roleService) Get(ctx context.Context, id uint) (*v1.Role, error) {
	role, err := s.find(ctx, id)
	if err != nil {
		return nil, err
	}
	return toRoleRes(*role), nil
}

func (s *roleService) Create(ctx context.Context, req *v1.CreateReq) (*v1.Role, error) {
	exists, err := s.roleRepo.ExistsCode(ctx, req.Code)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, rescode.New(rescode.Conflict, "Mã vai trò đã tồn tại")
	}
	perms, err := s.resolvePermissions(ctx, req.Permissions)
	if err != nil {
		return nil, err
	}

	role := &entity.Role{Code: req.Code, Name: req.Name, Description: req.Description}
	role.CreatedBy = reqctx.UserID(ctx)
	role.UpdatedBy = role.CreatedBy
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.roleRepo.Create(ctx, role); err != nil {
			return mapDuplicate(err, "Mã vai trò đã tồn tại")
		}
		return s.roleRepo.ReplacePermissions(ctx, role, perms)
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, role.ID)
}

func (s *roleService) Update(ctx context.Context, req *v1.UpdateReq) (*v1.Role, error) {
	role, err := s.find(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	perms, err := s.resolvePermissions(ctx, req.Permissions)
	if err != nil {
		return nil, err
	}
	// OWNER luôn giữ toàn quyền để không thể tự khóa mình khỏi hệ thống.
	if role.Code == consts.RoleOwner {
		if perms, err = s.roleRepo.ListPermissions(ctx); err != nil {
			return nil, err
		}
	}

	role.Name = req.Name
	role.Description = req.Description
	role.UpdatedBy = reqctx.UserID(ctx)
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.roleRepo.Update(ctx, role); err != nil {
			return err
		}
		return s.roleRepo.ReplacePermissions(ctx, role, perms)
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, role.ID)
}

func (s *roleService) Delete(ctx context.Context, id uint) error {
	role, err := s.find(ctx, id)
	if err != nil {
		return err
	}
	if role.IsSystem {
		return rescode.New(rescode.Conflict, "Không thể xóa vai trò hệ thống")
	}
	n, err := s.roleRepo.CountUsers(ctx, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return rescode.New(rescode.Conflict, "Không thể xóa vai trò đang được gán cho người dùng")
	}
	return s.roleRepo.Delete(ctx, id)
}

func (s *roleService) find(ctx context.Context, id uint) (*entity.Role, error) {
	role, err := s.roleRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy vai trò")
		}
		return nil, err
	}
	return role, nil
}

// resolvePermissions đổi danh sách mã quyền sang entity, báo lỗi nếu có mã không tồn tại.
func (s *roleService) resolvePermissions(ctx context.Context, codes []string) ([]entity.Permission, error) {
	slices.Sort(codes)
	codes = slices.Compact(codes)
	perms, err := s.roleRepo.FindPermissionsByCodes(ctx, codes)
	if err != nil {
		return nil, err
	}
	if len(perms) != len(codes) {
		return nil, rescode.New(rescode.BadRequest, "Danh sách quyền có mã không hợp lệ")
	}
	return perms, nil
}

func toRoleRes(r entity.Role) *v1.Role {
	perms := make([]string, 0, len(r.Permissions))
	for _, p := range r.Permissions {
		perms = append(perms, p.Code)
	}
	slices.Sort(perms)
	return &v1.Role{
		ID:          r.ID,
		Code:        r.Code,
		Name:        r.Name,
		Description: r.Description,
		IsSystem:    r.IsSystem,
		Permissions: perms,
	}
}

func toPermissionRes(p entity.Permission) *v1.Permission {
	return &v1.Permission{ID: p.ID, Code: p.Code, Module: p.Module, Name: p.Name}
}
