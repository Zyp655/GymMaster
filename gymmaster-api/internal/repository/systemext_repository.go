package repository

import (
	"context"

	"gorm.io/gorm"

	"gymmaster-api/internal/entity"
)

type SystemExtRepository struct {
	db *gorm.DB
}

func NewSystemExtRepository(db *gorm.DB) *SystemExtRepository {
	return &SystemExtRepository{db: db}
}

func (r *SystemExtRepository) DB() *gorm.DB { return r.db }

func (r *SystemExtRepository) ListBranchTransfers(ctx context.Context) ([]entity.MemberBranchTransfer, error) {
	var list []entity.MemberBranchTransfer
	err := r.db.WithContext(ctx).Order("created_at DESC").Limit(100).Find(&list).Error
	return list, err
}

func (r *SystemExtRepository) ListLegacyImports(ctx context.Context) ([]entity.LegacyImportJob, error) {
	var list []entity.LegacyImportJob
	err := r.db.WithContext(ctx).Order("created_at DESC").Limit(50).Find(&list).Error
	return list, err
}

func (r *SystemExtRepository) CreateLegacyImportJob(ctx context.Context, job *entity.LegacyImportJob) error {
	return r.db.WithContext(ctx).Create(job).Error
}
