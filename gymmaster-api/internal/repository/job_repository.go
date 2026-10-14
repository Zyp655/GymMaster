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
	// JobExpiredNotice: gói vừa được cron chuyển EXPIRED (để báo cho hội viên).
	JobExpiredNotice struct {
		SubscriptionID uint      `gorm:"column:subscription_id"`
		MemberID       uint      `gorm:"column:member_id"`
		MemberUserID   *uint     `gorm:"column:member_user_id"`
		MemberName     string    `gorm:"column:member_name"`
		MemberEmail    string    `gorm:"column:member_email"`
		PlanName       string    `gorm:"column:plan_name"`
		BranchID       uint      `gorm:"column:branch_id"`
		BranchName     string    `gorm:"column:branch_name"`
		EndDate        time.Time `gorm:"column:end_date"`
	}

	// JobRepository: truy vấn phục vụ job định kỳ và API xem lịch sử chạy job.
	JobRepository interface {
		DBNow(ctx context.Context) (time.Time, error)
		// ExpiredNotices: gói có sự kiện EXPIRED (không phải khôi phục gói cũ) từ mốc since
		// và hội viên chưa có gói khác còn hiệu lực/kế tiếp.
		ExpiredNotices(ctx context.Context, since time.Time) ([]JobExpiredNotice, error)
		// DeleteStaleRefreshTokens xóa refresh token hết hạn hoặc bị thu hồi trước mốc before.
		DeleteStaleRefreshTokens(ctx context.Context, before time.Time) (int64, error)
		ListRuns(ctx context.Context, job, status string, q dto.Query) ([]entity.JobRun, int64, error)
		LastRuns(ctx context.Context, jobs []string) (map[string]entity.JobRun, error)
	}

	jobRepository struct {
		db *gorm.DB
	}
)

func NewJobRepository(db *gorm.DB) JobRepository {
	return &jobRepository{db: db}
}

func (r *jobRepository) DBNow(ctx context.Context) (time.Time, error) {
	var t time.Time
	err := postgres.Conn(ctx, r.db).Raw("SELECT now()").Scan(&t).Error
	return t, err
}

func (r *jobRepository) ExpiredNotices(ctx context.Context, since time.Time) ([]JobExpiredNotice, error) {
	var rows []JobExpiredNotice
	err := postgres.Conn(ctx, r.db).Raw(`
SELECT DISTINCT ON (s.id) s.id AS subscription_id, s.member_id, m.user_id AS member_user_id, m.full_name AS member_name,
	m.email AS member_email, p.name AS plan_name, s.branch_id, b.name AS branch_name, s.end_date
FROM subscription_events e
JOIN subscriptions s ON s.id = e.subscription_id
JOIN members m ON m.id = s.member_id AND m.deleted_at IS NULL
JOIN membership_plans p ON p.id = s.plan_id
JOIN branches b ON b.id = s.branch_id
WHERE e.event_type = 'EXPIRED' AND e.created_at >= ?
  AND e.data->>'restored' IS NULL
  AND s.status = 'EXPIRED'
  AND NOT EXISTS (SELECT 1 FROM subscriptions n WHERE n.member_id = s.member_id AND n.id <> s.id
      AND n.status IN ('PENDING','ACTIVE','FROZEN') AND n.end_date >= s.end_date)
ORDER BY s.id`, since).Scan(&rows).Error
	return rows, err
}

func (r *jobRepository) DeleteStaleRefreshTokens(ctx context.Context, before time.Time) (int64, error) {
	res := postgres.Conn(ctx, r.db).Exec(
		"DELETE FROM refresh_tokens WHERE expires_at < ? OR (revoked_at IS NOT NULL AND revoked_at < ?)", before, before)
	return res.RowsAffected, res.Error
}

func (r *jobRepository) ListRuns(ctx context.Context, job, status string, q dto.Query) ([]entity.JobRun, int64, error) {
	db := postgres.Conn(ctx, r.db).Model(&entity.JobRun{}).Where("job = ?", job)
	if status != "" {
		db = db.Where("status = ?", status)
	}
	var total int64
	if err := db.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []entity.JobRun
	err := db.Scopes(q.Paginate).Find(&rows).Error
	return rows, total, err
}

func (r *jobRepository) LastRuns(ctx context.Context, jobs []string) (map[string]entity.JobRun, error) {
	out := map[string]entity.JobRun{}
	if len(jobs) == 0 {
		return out, nil
	}
	var rows []entity.JobRun
	err := postgres.Conn(ctx, r.db).Raw(
		"SELECT DISTINCT ON (job) * FROM job_runs WHERE job IN ? ORDER BY job, started_at DESC, id DESC", jobs).Scan(&rows).Error
	for _, row := range rows {
		out[row.Job] = row
	}
	return out, err
}
