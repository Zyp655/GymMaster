package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"gymmaster-api/internal/consts"
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
		// MemberBranchID: chi nhánh gốc của hội viên – dùng giới hạn phạm vi chi nhánh khi admin xử lý.
		MemberBranchID uint
		HandlerName    string
	}

	PrivacyRepository interface {
		FindUserByID(ctx context.Context, userID uint) (*entity.User, error)
		FindUserMFA(ctx context.Context, userID uint) (*entity.UserMFA, error)
		SaveUserMFA(ctx context.Context, mfa *entity.UserMFA) error
		DeleteUserMFA(ctx context.Context, userID uint) error
		// ConsumeTOTPStep ghi nhận bước thời gian TOTP vừa dùng – nguyên tử: chỉ thành công (true) khi bước này
		// mới hơn bước đã dùng gần nhất, nên một mã TOTP không thể dùng lại (kể cả hai request song song).
		ConsumeTOTPStep(ctx context.Context, userID uint, step int64) (bool, error)
		// ConsumeRecoveryCode xóa mã khôi phục (theo hash) khỏi danh sách bằng một câu UPDATE có điều kiện:
		// true = mã tồn tại và đã bị tiêu hủy bởi chính lời gọi này (hai lần đăng nhập song song chỉ một lần thành công,
		// và không ghi đè mất việc tiêu hủy mã khác).
		ConsumeRecoveryCode(ctx context.Context, userID uint, codeHash string) (bool, error)

		ListActiveSessions(ctx context.Context, userID uint, now time.Time) ([]entity.RefreshToken, error)
		RevokeSession(ctx context.Context, userID, sessionID uint, now time.Time) (bool, error)
		RevokeOtherSessions(ctx context.Context, userID, excludeSessionID uint, excludeTokenHash string, now time.Time) (int64, error)

		ListMemberConsents(ctx context.Context, memberID uint) ([]entity.Consent, error)
		UpsertConsent(ctx context.Context, c *entity.Consent) error

		// ListDataRequests: branchIDs rỗng/nil = không lọc chi nhánh (toàn hệ thống).
		ListDataRequests(ctx context.Context, q dto.Query, status, reqType string, branchIDs []uint) ([]DataRequestRow, int64, error)
		ListMemberDataRequests(ctx context.Context, memberID uint) ([]DataRequestRow, error)
		FindDataRequestByID(ctx context.Context, id uint) (*DataRequestRow, error)
		CreateDataRequest(ctx context.Context, dr *entity.DataRequest) error
		// HasPendingDataRequest: hội viên đã có yêu cầu cùng loại đang chờ xử lý.
		HasPendingDataRequest(ctx context.Context, memberID uint, reqType string) (bool, error)
		SaveDataRequestExport(ctx context.Context, id uint, data []byte) error
		// FindDataRequestExport trả bản trích xuất đã lưu (gorm.ErrRecordNotFound nếu không có).
		FindDataRequestExport(ctx context.Context, id uint) ([]byte, error)
		UpdateDataRequest(ctx context.Context, dr *entity.DataRequest) error

		FindMemberByID(ctx context.Context, id uint) (*entity.Member, error)
		FindMemberByUserID(ctx context.Context, userID uint) (*entity.Member, error)
		// AnonymizeMember ẩn danh hóa hồ sơ hội viên VÀ tài khoản đăng nhập liên kết (nếu có):
		// xóa thông tin cá nhân, khóa tài khoản, thu hồi mọi phiên/2FA/link đặt lại mật khẩu.
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

// SaveUserMFA ghi cấu hình MFA (upsert theo user_id). Không dùng Save(): bản ghi đã có mà struct mang CreatedAt rỗng
// (thiết lập lại MFA) sẽ bị UPDATE created_at = 0001-01-01. Khi trùng chỉ cập nhật các cột cấu hình + updated_at,
// giữ nguyên created_at.
func (r *privacyRepository) SaveUserMFA(ctx context.Context, mfa *entity.UserMFA) error {
	mfa.UpdatedAt = time.Now()
	return postgres.Conn(ctx, r.db).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"totp_secret_enc", "enabled_at", "recovery_codes_hash", "updated_at"}),
	}).Create(mfa).Error
}

func (r *privacyRepository) DeleteUserMFA(ctx context.Context, userID uint) error {
	return postgres.Conn(ctx, r.db).Where("user_id = ?", userID).Delete(&entity.UserMFA{}).Error
}

func (r *privacyRepository) ConsumeTOTPStep(ctx context.Context, userID uint, step int64) (bool, error) {
	res := postgres.Conn(ctx, r.db).Exec(
		`UPDATE user_mfa SET last_totp_step = ?, updated_at = now() WHERE user_id = ? AND last_totp_step < ?`,
		step, userID, step)
	return res.RowsAffected > 0, res.Error
}

func (r *privacyRepository) ConsumeRecoveryCode(ctx context.Context, userID uint, codeHash string) (bool, error) {
	// jsonb - text: bỏ phần tử chuỗi khỏi mảng; jsonb_exists thay cho toán tử "?" (trùng placeholder của GORM).
	res := postgres.Conn(ctx, r.db).Exec(
		`UPDATE user_mfa SET recovery_codes_hash = recovery_codes_hash - ?::text, updated_at = now()
		 WHERE user_id = ? AND enabled_at IS NOT NULL AND jsonb_exists(recovery_codes_hash, ?)`,
		codeHash, userID, codeHash)
	return res.RowsAffected > 0, res.Error
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
	COALESCE(m.home_branch_id, 0) AS member_branch_id,
	COALESCE(u.full_name, '') AS handler_name`

func (r *privacyRepository) baseDataRequest(ctx context.Context) *gorm.DB {
	return postgres.Conn(ctx, r.db).Table("data_requests dr").
		Select(dataRequestSelect).
		Joins("LEFT JOIN members m ON m.id = dr.member_id").
		Joins("LEFT JOIN users u ON u.id = dr.handled_by")
}

func (r *privacyRepository) ListDataRequests(ctx context.Context, q dto.Query, status, reqType string, branchIDs []uint) ([]DataRequestRow, int64, error) {
	tx := r.baseDataRequest(ctx)
	if len(branchIDs) > 0 {
		tx = tx.Where("m.home_branch_id IN ?", branchIDs)
	}
	if status != "" {
		tx = tx.Where("dr.status = ?", status)
	}
	if reqType != "" {
		tx = tx.Where("dr.type = ?", reqType)
	}
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		tx = tx.Where("(m.full_name ILIKE ? OR m.member_code ILIKE ? OR dr.reason ILIKE ?)", like, like, like)
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

func (r *privacyRepository) HasPendingDataRequest(ctx context.Context, memberID uint, reqType string) (bool, error) {
	var n int64
	err := postgres.Conn(ctx, r.db).Model(&entity.DataRequest{}).
		Where("member_id = ? AND type = ? AND status = ?", memberID, reqType, entity.DataRequestPending).
		Count(&n).Error
	return n > 0, err
}

func (r *privacyRepository) SaveDataRequestExport(ctx context.Context, id uint, data []byte) error {
	return postgres.Conn(ctx, r.db).Exec(`UPDATE data_requests SET export_data = ?::jsonb WHERE id = ?`, string(data), id).Error
}

func (r *privacyRepository) FindDataRequestExport(ctx context.Context, id uint) ([]byte, error) {
	var rows []struct{ ExportData *string }
	if err := postgres.Conn(ctx, r.db).Raw(
		`SELECT export_data::text AS export_data FROM data_requests WHERE id = ?`, id).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 || rows[0].ExportData == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return []byte(*rows[0].ExportData), nil
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
	conn := postgres.Conn(ctx, r.db)
	var m entity.Member
	if err := conn.Select("id", "user_id").First(&m, memberID).Error; err != nil {
		return err
	}
	if err := conn.Model(&entity.Member{}).
		Where("id = ?", memberID).
		Updates(map[string]any{
			"full_name": "Hội viên đã ẩn danh",
			// members.phone là UNIQUE (uq_members_phone) → dùng giá trị giữ chỗ duy nhất thay cho chuỗi rỗng,
			// nếu không hồ sơ ẩn danh thứ hai sẽ lỗi trùng khóa. Không phải SĐT hợp lệ nên không ai đăng ký trùng được.
			"phone":                   fmt.Sprintf("ANON-%d", memberID),
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
		}).Error; err != nil {
		return err
	}
	if m.UserID == nil {
		return nil
	}
	return r.anonymizeUser(ctx, *m.UserID, updatedBy, now)
}

// anonymizeUser xóa thông tin cá nhân của tài khoản đăng nhập và vô hiệu hóa hoàn toàn: email thay bằng
// địa chỉ không định tuyến được (giữ ràng buộc unique), mật khẩu không thể khớp, trạng thái LOCKED,
// thu hồi refresh token, xóa 2FA và vô hiệu link đặt lại mật khẩu.
func (r *privacyRepository) anonymizeUser(ctx context.Context, userID uint, updatedBy *uint, now time.Time) error {
	conn := postgres.Conn(ctx, r.db)
	if err := conn.Model(&entity.User{}).Where("id = ?", userID).
		Updates(map[string]any{
			"email":          fmt.Sprintf("deleted-user-%d@anonymized.invalid", userID),
			"full_name":      "Người dùng đã ẩn danh",
			"phone":          "",
			"avatar_url":     "",
			"password_hash":  "!anonymized", // không phải chuỗi bcrypt hợp lệ → không thể đăng nhập
			"status":         consts.StatusLocked,
			"calendar_token": nil,
			"updated_by":     updatedBy,
			"updated_at":     now,
		}).Error; err != nil {
		return err
	}
	if err := conn.Model(&entity.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		UpdateColumn("revoked_at", now).Error; err != nil {
		return err
	}
	if err := conn.Model(&entity.PasswordReset{}).
		Where("user_id = ? AND used_at IS NULL", userID).
		UpdateColumn("used_at", now).Error; err != nil {
		return err
	}
	return conn.Where("user_id = ?", userID).Delete(&entity.UserMFA{}).Error
}
