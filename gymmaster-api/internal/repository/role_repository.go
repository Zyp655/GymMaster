package repository

import (
	"context"

	"gorm.io/gorm"

	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/storage/postgres"
)

type (
	RoleRepository interface {
		List(ctx context.Context) ([]entity.Role, error)
		FindByID(ctx context.Context, id uint) (*entity.Role, error)
		FindByCode(ctx context.Context, code string) (*entity.Role, error)
		ExistsCode(ctx context.Context, code string) (bool, error)
		Create(ctx context.Context, role *entity.Role) error
		Update(ctx context.Context, role *entity.Role) error
		// ReplacePermissions thay toàn bộ quyền của role bằng danh sách mới.
		ReplacePermissions(ctx context.Context, role *entity.Role, perms []entity.Permission) error
		Delete(ctx context.Context, id uint) error
		CountUsers(ctx context.Context, id uint) (int64, error)
		// CountUsersOutsideBranches đếm lượt gán role cho người dùng ngoài các chi nhánh branchIDs
		// (kể cả gán toàn hệ thống, branch_id NULL).
		CountUsersOutsideBranches(ctx context.Context, id uint, branchIDs []uint) (int64, error)

		ListPermissions(ctx context.Context) ([]entity.Permission, error)
		FindPermissionsByCodes(ctx context.Context, codes []string) ([]entity.Permission, error)
	}

	roleRepository struct {
		db *gorm.DB
	}
)

func NewRoleRepository(db *gorm.DB) RoleRepository {
	return &roleRepository{db: db}
}

func (r *roleRepository) List(ctx context.Context) ([]entity.Role, error) {
	var roles []entity.Role
	err := postgres.Conn(ctx, r.db).Preload("Permissions", func(db *gorm.DB) *gorm.DB {
		return db.Order("code")
	}).Order("is_system DESC, id").Find(&roles).Error
	return roles, err
}

func (r *roleRepository) FindByID(ctx context.Context, id uint) (*entity.Role, error) {
	var role entity.Role
	if err := postgres.Conn(ctx, r.db).Preload("Permissions").First(&role, id).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *roleRepository) FindByCode(ctx context.Context, code string) (*entity.Role, error) {
	var role entity.Role
	if err := postgres.Conn(ctx, r.db).Where("code = ?", code).First(&role).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *roleRepository) ExistsCode(ctx context.Context, code string) (bool, error) {
	var n int64
	err := postgres.Conn(ctx, r.db).Model(&entity.Role{}).Where("code = ?", code).Count(&n).Error
	return n > 0, err
}

func (r *roleRepository) Create(ctx context.Context, role *entity.Role) error {
	return postgres.Conn(ctx, r.db).Omit("Permissions").Create(role).Error
}

func (r *roleRepository) Update(ctx context.Context, role *entity.Role) error {
	return postgres.Conn(ctx, r.db).Model(role).Omit("Permissions").
		Select("name", "description", "updated_by").Updates(role).Error
}

func (r *roleRepository) ReplacePermissions(ctx context.Context, role *entity.Role, perms []entity.Permission) error {
	return postgres.Conn(ctx, r.db).Model(role).Association("Permissions").Replace(perms)
}

func (r *roleRepository) Delete(ctx context.Context, id uint) error {
	return postgres.Conn(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM role_permissions WHERE role_id = ?", id).Error; err != nil {
			return err
		}
		return tx.Delete(&entity.Role{}, id).Error
	})
}

func (r *roleRepository) CountUsers(ctx context.Context, id uint) (int64, error) {
	var n int64
	err := postgres.Conn(ctx, r.db).Model(&entity.UserRole{}).Where("role_id = ?", id).Count(&n).Error
	return n, err
}

func (r *roleRepository) CountUsersOutsideBranches(ctx context.Context, id uint, branchIDs []uint) (int64, error) {
	var n int64
	q := postgres.Conn(ctx, r.db).Model(&entity.UserRole{}).Where("role_id = ?", id)
	if len(branchIDs) > 0 {
		q = q.Where("(branch_id IS NULL OR branch_id NOT IN ?)", branchIDs)
	}
	err := q.Count(&n).Error
	return n, err
}

func (r *roleRepository) ListPermissions(ctx context.Context) ([]entity.Permission, error) {
	var perms []entity.Permission
	err := postgres.Conn(ctx, r.db).Order("module, id").Find(&perms).Error
	return perms, err
}

func (r *roleRepository) FindPermissionsByCodes(ctx context.Context, codes []string) ([]entity.Permission, error) {
	perms := []entity.Permission{}
	if len(codes) == 0 {
		return perms, nil
	}
	err := postgres.Conn(ctx, r.db).Where("code IN ?", codes).Find(&perms).Error
	return perms, err
}
