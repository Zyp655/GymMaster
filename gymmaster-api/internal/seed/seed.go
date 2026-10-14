// Package seed tạo dữ liệu khởi tạo/demo. Chạy lặp lại nhiều lần không bị trùng (idempotent).
package seed

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"gorm.io/gorm"

	"gymmaster-api/internal/config"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/hash"
)

func Run(ctx context.Context, db *gorm.DB, cfg *config.Config) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := seedBranches(ctx, tx); err != nil {
			return err
		}
		return seedOwner(ctx, tx, cfg.Seed)
	})
}

func seedBranches(ctx context.Context, tx *gorm.DB) error {
	branches := []entity.Branch{
		{Code: "HN01", Name: "GymMaster Cầu Giấy", Address: "144 Xuân Thủy, Cầu Giấy, Hà Nội", Phone: "02437547461", Email: "caugiay@gymmaster.vn", OpenTime: "05:30", CloseTime: "22:00", Status: consts.StatusActive},
		{Code: "HN02", Name: "GymMaster Hai Bà Trưng", Address: "1 Đại Cồ Việt, Hai Bà Trưng, Hà Nội", Phone: "02438692222", Email: "hbt@gymmaster.vn", OpenTime: "05:30", CloseTime: "22:00", Status: consts.StatusActive},
		{Code: "HCM01", Name: "GymMaster Quận 1", Address: "86 Lê Thánh Tôn, Quận 1, TP.HCM", Phone: "02838221111", Email: "q1@gymmaster.vn", OpenTime: "06:00", CloseTime: "22:30", Status: consts.StatusActive},
	}
	for _, b := range branches {
		var n int64
		if err := tx.Model(&entity.Branch{}).Where("code = ?", b.Code).Count(&n).Error; err != nil {
			return err
		}
		if n == 0 {
			if err := tx.Create(&b).Error; err != nil {
				return err
			}
			g.Log().Infof(ctx, "seed: tạo chi nhánh %s", b.Code)
		}
	}
	return nil
}

func seedOwner(ctx context.Context, tx *gorm.DB, cfg config.SeedCfg) error {
	var n int64
	if err := tx.Model(&entity.User{}).Where("lower(email) = lower(?)", cfg.AdminEmail).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return nil
	}

	hashed, err := hash.HashPassword(cfg.AdminPassword)
	if err != nil {
		return err
	}
	user := entity.User{Email: cfg.AdminEmail, PasswordHash: hashed, FullName: "Quản trị viên", Phone: "0900000000", Status: consts.StatusActive}
	if err := tx.Omit("UserRoles").Create(&user).Error; err != nil {
		return err
	}
	var owner entity.Role
	if err := tx.Where("code = ?", consts.RoleOwner).First(&owner).Error; err != nil {
		return err
	}
	if err := tx.Omit("Role").Create(&entity.UserRole{UserID: user.ID, RoleID: owner.ID}).Error; err != nil {
		return err
	}
	g.Log().Infof(ctx, "seed: tạo tài khoản OWNER %s", cfg.AdminEmail)
	return nil
}
