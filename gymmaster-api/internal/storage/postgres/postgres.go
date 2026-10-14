package postgres

import (
	"context"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"gymmaster-api/internal/config"
)

// Open mở kết nối GORM tới PostgreSQL. Schema KHÔNG được tạo ở đây
// (không AutoMigrate) – schema do golang-migrate quản lý trong thư mục migrations/.
func Open(cfg config.DbCfg) (*gorm.DB, error) {
	logLevel := logger.Warn
	if cfg.LogSQL {
		logLevel = logger.Info
	}

	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger:         logger.Default.LogMode(logLevel),
		TranslateError: true, // map lỗi unique/fk của Postgres sang gorm.ErrDuplicatedKey/ErrForeignKeyViolated
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, err
	}
	return db, nil
}
