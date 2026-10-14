package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/storage/postgres"
)

type (
	DataRequestRow struct {
		entity.DataRequest
		MemberCode  string
		MemberName  string
		MemberPhone string
		HandlerName string
	}

	PrivacyRepository interface {
		FindUserByID(ctx context.Context, userID uint) (*entity.User, error)
		FindUserMFA(ctx context.Context, userID uint) (*entity.UserMFA, error)
		SaveUserMFA(ctx context.Context, mfa *entity.UserMFA) error
		DeleteUserMFA(ctx context.Context, userID uint) error

		ListActiveSessions(ctx context.Context, userID uint, now time.Time) ([]entity.RefreshToken, error)
		RevokeSession(ctx context.Context, userID, sessionID uint, now time.Time) (bool, error)
		RevokeOtherSessions(ctx context.Context, userID, excludeSessionID uint, excludeTokenHash string, now time.Time) (int64, error)

		ListMemberConsents(ctx context.Context, memberID uint) ([]entity.Consent, error)
		UpsertConsent(ctx context.Context, c *entity.Consent) error

		ListDataRequests(ctx context.Context, q dto.Query, status, reqType string) ([]DataRequestRow, int64, error)
		ListMemberDataRequests(ctx context.Context, memberID uint) ([]DataRequestRow, error)
		FindDataRequestByID(ctx context.Context, id uint) (*DataRequestRow, error)
		CreateDataRequest(ctx context.Context, dr *entity.DataRequest) error
		UpdateDataRequest(ctx context.Context, dr *entity.DataRequest) error

		FindMemberByID(ctx context.Context, id uint) (*entity.Member, error)
		FindMemberByUserID(ctx context.Context, userID uint) (*entity.Member, error)
		AnonymizeMember(ctx context.Context, memberID uint, updatedBy *uint) error
	}

	privacyRepository struct {
		db *gorm.DB
	}
)

func NewPrivacyRepository(db *gorm.DB) PrivacyRepository {
	return &privacyRepository{db: db}
}

func (r *privacyRepository) FindUserByID(ctx context.Context, userID uint) (*entity.User, error) {
	var u entity.User
	if err := postgres.Conn(ctx, r.db).First(&u, userID).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *privacyRepository) FindUserMFA(ctx context.Context, userID uint) (*entity.UserMFA, error) {
	var mfa entity.UserMFA
	if err := postgres.Conn(ctx, r.db).Where("user_id = ?", userID).First(&mfa).Error; err != nil {
		return nil, err
	}
	return &mfa, nil
}

func (r *privacyRepository) SaveUserMFA(ctx context.Context, mfa *entity.UserMFA) error {
	return postgres.Conn(ctx, r.db).Save(mfa).Error
}

func (r *privacyRepository) DeleteUserMFA(ctx context.Context, userID uint) error {
	return postgres.Conn(ctx, r.db).Where("user_id = ?", userID).Delete(&entity.UserMFA{}).Error
}

func (r *privacyRepository) ListActiveSessions(ctx context.Context, userID uint, now time.Time) ([]entity.RefreshToken, error) {
	var items []entity.RefreshToken
	err := postgres.Conn(ctx, r.db).
		Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", userID, now).
		Order("COALESCE(last_used_at, created_at) DESC, id DESC").
		Find(&items).Error
	return items, err
}

func (r *privacyRepository) RevokeSession(ctx context.Context, userID, sessionID uint, now time.Time) (bool, error) {
	res := postgres.Conn(ctx, r.db).Model(&entity.RefreshToken{}).
		Where("id = ? AND user_id = ? AND revoked_at IS NULL", sessionID, userID).
		UpdateColumn("revoked_at", now)
	return res.RowsAffected > 0, res.Error
}

func (r *privacyRepository) RevokeOtherSessions(ctx context.Context, userID, excludeSessionID uint, excludeTokenHash string, now time.Time) (int64, error) {
	tx := postgres.Conn(ctx, r.db).Model(&entity.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID)
	if excludeSessionID > 0 {
		tx = tx.Where("id <> ?", excludeSessionID)
	}
	if excludeTokenHash != "" {
		tx = tx.Where("token_hash <> ?", excludeTokenHash)
	}
	res := tx.UpdateColumn("revoked_at", now)
	return res.RowsAffected, res.Error
}

func (r *privacyRepository) ListMemberConsents(ctx context.Context, memberID uint) ([]entity.Consent, error) {
	var items []entity.Consent
	err := postgres.Conn(ctx, r.db).Where("member_id = ?", memberID).Order("type ASC").Find(&items).Error
	return items, err
}

func (r *privacyRepository) UpsertConsent(ctx context.Context, c *entity.Consent) error {
	conn := postgres.Conn(ctx, r.db)
	var existing entity.Consent
	if err := conn.Where("member_id = ? AND type = ?", c.MemberID, c.Type).First(&existing).Error; err == nil {
		existing.Granted = c.Granted
		existing.GrantedAt = c.GrantedAt
		existing.IP = c.IP
		if err := conn.Save(&existing).Error; err != nil {
			return err
		}
		*c = existing
		return nil
	}
	return conn.Create(c).Error
}

const dataRequestSelect = `dr.*, COALESCE(m.member_code, '') AS member_code,
	COALESCE(m.full_name, '') AS member_name, COALESCE(m.phone, '') AS member_phone,
	COALESCE(u.full_name, '') AS handler_name`

func (r *privacyRepository) baseDataRequest(ctx context.Context) *gorm.DB {
	return postgres.Conn(ctx, r.db).Table("data_requests dr").
		Select(dataRequestSelect).
		Joins("LEFT JOIN members m ON m.id = dr.member_id").
		Joins("LEFT JOIN users u ON u.id = dr.handled_by")
}

func (r *privacyRepository) ListDataRequests(ctx context.Context, q dto.Query, status, reqType string) ([]DataRequestRow, int64, error) {
	tx := r.baseDataRequest(ctx)
	if status != "" {
		tx = tx.Where("dr.status = ?", status)
	}
	if reqType != "" {
		tx = tx.Where("dr.type = ?", reqType)
	}
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		tx = tx.Where("m.full_name ILIKE ? OR m.member_code ILIKE ? OR dr.reason ILIKE ?", like, like, like)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []DataRequestRow
	if err := q.Paginate(tx).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *privacyRepository) ListMemberDataRequests(ctx context.Context, memberID uint) ([]DataRequestRow, error) {
	var rows []DataRequestRow
	err := r.baseDataRequest(ctx).Where("dr.member_id = ?", memberID).Order("dr.id DESC").Scan(&rows).Error
	return rows, err
}

func (r *privacyRepository) FindDataRequestByID(ctx context.Context, id uint) (*DataRequestRow, error) {
	var row DataRequestRow
	res := r.baseDataRequest(ctx).Where("dr.id = ?", id).Limit(1).Find(&row)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}

func (r *privacyRepository) CreateDataRequest(ctx context.Context, dr *entity.DataRequest) error {
	return postgres.Conn(ctx, r.db).Create(dr).Error
}

func (r *privacyRepository) UpdateDataRequest(ctx context.Context, dr *entity.DataRequest) error {
	return postgres.Conn(ctx, r.db).Save(dr).Error
}

func (r *privacyRepository) FindMemberByID(ctx context.Context, id uint) (*entity.Member, error) {
	var m entity.Member
	if err := postgres.Conn(ctx, r.db).First(&m, id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *privacyRepository) FindMemberByUserID(ctx context.Context, userID uint) (*entity.Member, error) {
	var m entity.Member
	if err := postgres.Conn(ctx, r.db).Where("user_id = ?", userID).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *privacyRepository) AnonymizeMember(ctx context.Context, memberID uint, updatedBy *uint) error {
	now := time.Now()
	return postgres.Conn(ctx, r.db).Model(&entity.Member{}).
		Where("id = ?", memberID).
		Updates(map[string]any{
			"full_name":               "Hội viên đã ẩn danh",
			"phone":                   "",
			"email":                   "",
			"id_card":                 "",
			"address":                 "",
			"avatar_url":              "",
			"emergency_contact_name":  "",
			"emergency_contact_phone": "",
			"dob":                     nil,
			"status":                  entity.MemberStatusInactive,
			"updated_by":              updatedBy,
			"updated_at":              now,
		}).Error
}
