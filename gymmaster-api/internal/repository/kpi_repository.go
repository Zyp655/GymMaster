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
	BranchOccupancyRow struct {
		BranchID     uint
		BranchCode   string
		BranchName   string
		Capacity     int
		PresentCount int
	}

	SalesTargetRow struct {
		entity.SalesTarget
		BranchName string
		StaffName  string
	}

	KPIActuals struct {
		Revenue       float64
		NewMembers    float64
		Renewals      float64
		PTSalesAmount float64
		PTSalesCount  float64
	}

	CohortMemberRow struct {
		ID          uint
		JoinedMonth string
	}

	KPIRepository interface {
		ListBranchOccupancy(ctx context.Context, since time.Time) ([]BranchOccupancyRow, error)

		ListSalesTargets(ctx context.Context, q dto.Query, branchID uint, period, metric string) ([]SalesTargetRow, int64, error)
		ListTargetsForPeriod(ctx context.Context, branchID uint, period string) ([]entity.SalesTarget, error)
		FindSalesTargetByID(ctx context.Context, id uint) (*SalesTargetRow, error)
		UpsertSalesTarget(ctx context.Context, st *entity.SalesTarget) error
		DeleteSalesTarget(ctx context.Context, id uint) error

		// BranchExists: chi nhánh tồn tại và chưa bị xóa mềm.
		BranchExists(ctx context.Context, branchID uint) (bool, error)
		// StaffBranchStatus: exists = tài khoản nhân viên (có vai trò khác MEMBER) đang ACTIVE, chưa xóa;
		// inBranch = có vai trò tại branchID hoặc vai trò toàn hệ thống (branch_id NULL).
		StaffBranchStatus(ctx context.Context, staffID, branchID uint) (exists, inBranch bool, err error)

		ComputeKPIActuals(ctx context.Context, branchID uint, from, to time.Time) (KPIActuals, error)
		ListCohortMembers(ctx context.Context, branchID uint, since time.Time) ([]CohortMemberRow, error)
		ListActiveMemberMonths(ctx context.Context, memberIDs []uint, since time.Time) (map[uint]map[string]bool, error)
	}

	kpiRepository struct {
		db *gorm.DB
	}
)

func NewKPIRepository(db *gorm.DB) KPIRepository {
	return &kpiRepository{db: db}
}

func (r *kpiRepository) ListBranchOccupancy(ctx context.Context, since time.Time) ([]BranchOccupancyRow, error) {
	var rows []BranchOccupancyRow
	err := postgres.Conn(ctx, r.db).Table("branches b").
		Select(`b.id AS branch_id, b.code AS branch_code, b.name AS branch_name, COALESCE(b.capacity, 120) AS capacity,
			(SELECT COUNT(DISTINCT COALESCE('m' || ck.member_id::text, 'd' || ck.day_pass_id::text, 't' || ck.trial_pass_id::text)) FROM checkins ck
			 WHERE ck.branch_id = b.id AND ck.result = 'ALLOWED'
			   AND ck.checked_in_at >= ? AND ck.checked_out_at IS NULL) AS present_count`, since).
		Where("b.deleted_at IS NULL AND b.status = 'ACTIVE'").
		Order("b.id ASC").
		Scan(&rows).Error
	return rows, err
}

const salesTargetSelect = `st.*, COALESCE(b.name, '') AS branch_name, COALESCE(u.full_name, '') AS staff_name`

func (r *kpiRepository) baseSalesTarget(ctx context.Context) *gorm.DB {
	return postgres.Conn(ctx, r.db).Table("sales_targets st").
		Select(salesTargetSelect).
		Joins("LEFT JOIN branches b ON b.id = st.branch_id").
		Joins("LEFT JOIN users u ON u.id = st.staff_id")
}

func (r *kpiRepository) ListSalesTargets(ctx context.Context, q dto.Query, branchID uint, period, metric string) ([]SalesTargetRow, int64, error) {
	tx := r.baseSalesTarget(ctx)
	if branchID > 0 {
		tx = tx.Where("st.branch_id = ?", branchID)
	}
	if period != "" {
		tx = tx.Where("st.period = ?", period)
	}
	if metric != "" {
		tx = tx.Where("st.metric = ?", metric)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []SalesTargetRow
	if err := q.Paginate(tx).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *kpiRepository) ListTargetsForPeriod(ctx context.Context, branchID uint, period string) ([]entity.SalesTarget, error) {
	tx := postgres.Conn(ctx, r.db).Model(&entity.SalesTarget{}).Where("period = ?", period)
	if branchID > 0 {
		tx = tx.Where("branch_id = ?", branchID)
	}
	var items []entity.SalesTarget
	err := tx.Find(&items).Error
	return items, err
}

func (r *kpiRepository) FindSalesTargetByID(ctx context.Context, id uint) (*SalesTargetRow, error) {
	var row SalesTargetRow
	res := r.baseSalesTarget(ctx).Where("st.id = ?", id).Limit(1).Find(&row)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}

func (r *kpiRepository) UpsertSalesTarget(ctx context.Context, st *entity.SalesTarget) error {
	conn := postgres.Conn(ctx, r.db)
	tx := conn.Model(&entity.SalesTarget{}).
		Where("branch_id = ? AND period = ? AND metric = ?", st.BranchID, st.Period, st.Metric)
	if st.StaffID != nil && *st.StaffID > 0 {
		tx = tx.Where("staff_id = ?", *st.StaffID)
	} else {
		tx = tx.Where("staff_id IS NULL")
	}
	var existing entity.SalesTarget
	if err := tx.First(&existing).Error; err == nil {
		existing.TargetValue = st.TargetValue
		if err := conn.Save(&existing).Error; err != nil {
			return err
		}
		*st = existing
		return nil
	}
	return conn.Create(st).Error
}

func (r *kpiRepository) BranchExists(ctx context.Context, branchID uint) (bool, error) {
	var n int64
	err := postgres.Conn(ctx, r.db).Table("branches").Where("id = ? AND deleted_at IS NULL", branchID).Count(&n).Error
	return n > 0, err
}

func (r *kpiRepository) StaffBranchStatus(ctx context.Context, staffID, branchID uint) (exists, inBranch bool, err error) {
	var row struct {
		Total    int64
		InBranch int64
	}
	err = postgres.Conn(ctx, r.db).Table("users u").
		Select("COUNT(*) AS total, COUNT(*) FILTER (WHERE ur.branch_id = ? OR ur.branch_id IS NULL) AS in_branch", branchID).
		Joins("JOIN user_roles ur ON ur.user_id = u.id").
		Joins("JOIN roles r ON r.id = ur.role_id").
		Where("u.id = ? AND u.deleted_at IS NULL AND u.status = 'ACTIVE' AND r.code <> 'MEMBER'", staffID).
		Scan(&row).Error
	return row.Total > 0, row.InBranch > 0, err
}

func (r *kpiRepository) DeleteSalesTarget(ctx context.Context, id uint) error {
	return postgres.Conn(ctx, r.db).Delete(&entity.SalesTarget{}, id).Error
}

func (r *kpiRepository) ComputeKPIActuals(ctx context.Context, branchID uint, from, to time.Time) (KPIActuals, error) {
	conn := postgres.Conn(ctx, r.db)
	var out KPIActuals

	// 1. REVENUE: net paid amount on non-void invoices in [from, to)
	invTx := conn.Table("invoices").
		Select("COALESCE(SUM(paid_amount - refunded_amount), 0)").
		Where("status <> 'VOID' AND issued_at >= ? AND issued_at < ?", from, to)
	if branchID > 0 {
		invTx = invTx.Where("branch_id = ?", branchID)
	}
	if err := invTx.Scan(&out.Revenue).Error; err != nil {
		return out, err
	}

	// 2. NEW_MEMBERS: members created in [from, to)
	var newMembers int64
	memTx := conn.Table("members").
		Where("deleted_at IS NULL AND created_at >= ? AND created_at < ?", from, to)
	if branchID > 0 {
		memTx = memTx.Where("home_branch_id = ?", branchID)
	}
	if err := memTx.Count(&newMembers).Error; err != nil {
		return out, err
	}
	out.NewMembers = float64(newMembers)

	// 3. RENEWALS: subscriptions created in [from, to) with previous_id IS NOT NULL or RENEWED event
	var renewals int64
	renTx := conn.Table("subscriptions s").
		Where("s.status <> 'CANCELLED' AND s.created_at >= ? AND s.created_at < ?", from, to).
		Where("(s.previous_id IS NOT NULL OR EXISTS (SELECT 1 FROM subscription_events se WHERE se.subscription_id = s.id AND se.event_type = 'RENEWED'))")
	if branchID > 0 {
		renTx = renTx.Where("s.branch_id = ?", branchID)
	}
	if err := renTx.Count(&renewals).Error; err != nil {
		return out, err
	}
	out.Renewals = float64(renewals)

	// 4. PT_SALES: pt_contracts created in [from, to)
	var ptStats struct {
		Cnt int64
		Sum float64
	}
	ptTx := conn.Table("pt_contracts").
		Select("COUNT(*) AS cnt, COALESCE(SUM(price), 0) AS sum").
		Where("status <> 'CANCELLED' AND created_at >= ? AND created_at < ?", from, to)
	if branchID > 0 {
		ptTx = ptTx.Where("branch_id = ?", branchID)
	}
	if err := ptTx.Scan(&ptStats).Error; err != nil {
		return out, err
	}
	out.PTSalesCount = float64(ptStats.Cnt)
	out.PTSalesAmount = ptStats.Sum
	return out, nil
}

func (r *kpiRepository) ListCohortMembers(ctx context.Context, branchID uint, since time.Time) ([]CohortMemberRow, error) {
	tx := postgres.Conn(ctx, r.db).Table("members").
		Select("id, TO_CHAR(created_at AT TIME ZONE 'Asia/Ho_Chi_Minh', 'YYYY-MM') AS joined_month").
		Where("deleted_at IS NULL AND created_at >= ?", since)
	if branchID > 0 {
		tx = tx.Where("home_branch_id = ?", branchID)
	}
	var rows []CohortMemberRow
	err := tx.Scan(&rows).Error
	return rows, err
}

func (r *kpiRepository) ListActiveMemberMonths(ctx context.Context, memberIDs []uint, since time.Time) (map[uint]map[string]bool, error) {
	active := map[uint]map[string]bool{}
	if len(memberIDs) == 0 {
		return active, nil
	}
	conn := postgres.Conn(ctx, r.db)

	var checkinRows []struct {
		MemberID uint
		Ym       string
	}
	if err := conn.Table("checkins").
		Select(" DISTINCT member_id, TO_CHAR(checked_in_at AT TIME ZONE 'Asia/Ho_Chi_Minh', 'YYYY-MM') AS ym").
		Where("member_id IN ? AND result = 'ALLOWED' AND checked_in_at >= ?", memberIDs, since).
		Scan(&checkinRows).Error; err != nil {
		return nil, err
	}
	for _, r := range checkinRows {
		if active[r.MemberID] == nil {
			active[r.MemberID] = map[string]bool{}
		}
		active[r.MemberID][r.Ym] = true
	}

	var subRows []struct {
		MemberID  uint
		StartDate time.Time
		EndDate   time.Time
	}
	if err := conn.Table("subscriptions").
		Select("member_id, start_date, end_date").
		Where("member_id IN ? AND status IN ('ACTIVE', 'EXPIRED', 'FROZEN') AND end_date >= ?", memberIDs, since.Format("2006-01-02")).
		Scan(&subRows).Error; err == nil {
		for _, sr := range subRows {
			if active[sr.MemberID] == nil {
				active[sr.MemberID] = map[string]bool{}
			}
			cur := time.Date(sr.StartDate.Year(), sr.StartDate.Month(), 1, 0, 0, 0, 0, time.UTC)
			endM := time.Date(sr.EndDate.Year(), sr.EndDate.Month(), 1, 0, 0, 0, 0, time.UTC)
			for !cur.After(endM) {
				active[sr.MemberID][cur.Format("2006-01")] = true
				cur = cur.AddDate(0, 1, 0)
			}
		}
	}
	return active, nil
}
