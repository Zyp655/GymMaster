//go:build integration

package service

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gormpg "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/storage/postgres"
)

// Chạy: GYMMASTER_TEST_DSN=postgres://gymmaster:gymmaster@localhost:5432/gymmaster?sslmode=disable make itest
//
//  1. Hai request "hoàn thành" đồng thời cùng một buổi PT → đúng 1 thành công, 1 nhận 409;
//     chỉ 1 dòng trainer_commissions và used_sessions chỉ tăng 1 (khóa hợp đồng FOR UPDATE + UNIQUE(source)).
//  2. Hai request đặt buổi trùng giờ cùng một HLV (2 hợp đồng / 2 hội viên khác nhau) → 1 thành công, 1 nhận 409
//     (EXCLUDE ex_pt_trainer_overlap, SQLSTATE 23P01).

type ptITFixture struct {
	db        *gorm.DB
	svc       PTService
	ctx       context.Context
	trainerID uint
	branchID  uint
	memberIDs []uint
	contracts []uint
}

func ptITSetup(t *testing.T) *ptITFixture {
	t.Helper()
	dsn := os.Getenv("GYMMASTER_TEST_DSN")
	if dsn == "" {
		t.Skip("chưa đặt GYMMASTER_TEST_DSN")
	}
	db, err := gorm.Open(gormpg.Open(dsn), &gorm.Config{TranslateError: true, Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(10)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)
	exec := func(dest *uint, sql string, args ...any) {
		t.Helper()
		if err := db.Raw(sql, args...).Scan(dest).Error; err != nil {
			t.Fatalf("seed: %v\n%s", err, sql)
		}
	}
	f := &ptITFixture{db: db}
	var branchID, staffID, packageID uint
	exec(&branchID, `INSERT INTO branches (code, name, open_time, close_time) VALUES (?, ?, '00:00', '23:59') RETURNING id`,
		"ITPT"+suffix, "IT PT "+suffix)
	f.branchID = branchID
	exec(&f.trainerID, `INSERT INTO users (email, password_hash, full_name) VALUES (?, 'x', 'IT HLV PT') RETURNING id`,
		"it-pt-trainer-"+suffix+"@test.local")
	exec(&staffID, `INSERT INTO users (email, password_hash, full_name) VALUES (?, 'x', 'IT Lễ tân PT') RETURNING id`,
		"it-pt-staff-"+suffix+"@test.local")
	require.NoError(t, db.Exec(`INSERT INTO staff_profiles (user_id, position, commission_rate) VALUES (?, 'TRAINER', 10)`, f.trainerID).Error)
	require.NoError(t, db.Exec(`INSERT INTO user_roles (user_id, role_id, branch_id) SELECT ?, id, ? FROM roles WHERE code = 'TRAINER'`,
		f.trainerID, branchID).Error)
	exec(&packageID, `INSERT INTO pt_packages (code, name, session_count, session_minutes, validity_days, price)
		VALUES (?, 'IT PT 10', 10, 60, 45, 3500000) RETURNING id`, "ITPT"+suffix)
	phoneBase := time.Now().UnixNano() % 10_000_000
	for i := 0; i < 3; i++ {
		var memberID, contractID uint
		exec(&memberID, `INSERT INTO members (full_name, phone, home_branch_id) VALUES (?, ?, ?) RETURNING id`,
			fmt.Sprintf("IT HV PT %d", i), fmt.Sprintf("07%07d%02d", phoneBase, i), branchID)
		exec(&contractID, `INSERT INTO pt_contracts (member_id, trainer_id, pt_package_id, branch_id, total_sessions, session_minutes,
			start_date, end_date, price, status) VALUES (?, ?, ?, ?, 10, 60, CURRENT_DATE - 1, CURRENT_DATE + 40, 3500000, 'ACTIVE') RETURNING id`,
			memberID, f.trainerID, packageID, branchID)
		f.memberIDs = append(f.memberIDs, memberID)
		f.contracts = append(f.contracts, contractID)
	}

	t.Cleanup(func() {
		steps := []struct {
			sql  string
			args []any
		}{
			{`DELETE FROM trainer_commissions WHERE trainer_id = ?`, []any{f.trainerID}},
			{`DELETE FROM pt_sessions WHERE contract_id IN ?`, []any{f.contracts}},
			{`DELETE FROM pt_contracts WHERE id IN ?`, []any{f.contracts}},
			{`DELETE FROM pt_packages WHERE id = ?`, []any{packageID}},
			{`DELETE FROM members WHERE id IN ?`, []any{f.memberIDs}},
			{`DELETE FROM user_roles WHERE user_id = ?`, []any{f.trainerID}},
			{`DELETE FROM staff_profiles WHERE user_id = ?`, []any{f.trainerID}},
			{`DELETE FROM users WHERE id IN ?`, []any{[]uint{f.trainerID, staffID}}},
			{`DELETE FROM branches WHERE id = ?`, []any{branchID}},
		}
		for _, st := range steps {
			if err := db.Exec(st.sql, st.args...).Error; err != nil {
				t.Errorf("dọn dữ liệu: %v (%s)", err, st.sql)
			}
		}
		_ = sqlDB.Close()
	})

	tx := postgres.NewTxManager(db)
	invoices := NewInvoiceService(repository.NewInvoiceRepository(db), NewPromotionService(repository.NewPromotionRepository(db), nil), tx, nil)
	f.svc = NewPTService(repository.NewPTRepository(db), invoices, tx, NewSettingService(db), nil, nil)
	f.ctx = reqctx.WithAuth(context.Background(), &token.Payload{UserID: staffID, Roles: []string{consts.RoleOwner}})
	return f
}

func (f *ptITFixture) scalar(t *testing.T, sql string, args ...any) int64 {
	t.Helper()
	var n int64
	require.NoError(t, f.db.Raw(sql, args...).Scan(&n).Error)
	return n
}

func TestPTIntegration_ConcurrentComplete(t *testing.T) {
	f := ptITSetup(t)
	contractID := f.contracts[0]
	var sessionID uint
	require.NoError(t, f.db.Raw(`INSERT INTO pt_sessions (contract_id, trainer_id, member_id, branch_id, start_at, end_at)
		SELECT id, trainer_id, member_id, branch_id, now() - interval '90 minutes', now() - interval '30 minutes'
		FROM pt_contracts WHERE id = ? RETURNING id`, contractID).Scan(&sessionID).Error)

	const workers = 2
	var (
		wg        sync.WaitGroup
		start     = make(chan struct{})
		mu        sync.Mutex
		ok, confl int
		others    []error
	)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := f.svc.CompleteSession(f.ctx, sessionID, "Giáo án IT", "")
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				ok++
			case codeOf(err) == 40900:
				confl++
			default:
				others = append(others, err)
			}
		}()
	}
	close(start)
	wg.Wait()

	require.Empty(t, others)
	assert.Equal(t, 1, ok, "đúng 1 request hoàn thành thành công")
	assert.Equal(t, 1, confl, "request còn lại nhận 409")
	assert.EqualValues(t, 1, f.scalar(t, `SELECT COUNT(*) FROM trainer_commissions WHERE source_type = 'PT_SESSION' AND source_id = ?`, sessionID))
	assert.EqualValues(t, 1, f.scalar(t, `SELECT used_sessions FROM pt_contracts WHERE id = ?`, contractID))
	assert.EqualValues(t, 35000, f.scalar(t, `SELECT amount::bigint FROM trainer_commissions WHERE source_type = 'PT_SESSION' AND source_id = ?`, sessionID),
		"3.500.000 / 10 buổi × 10%")
}

func TestPTIntegration_ConcurrentBookingSameTrainer(t *testing.T) {
	f := ptITSetup(t)
	// Ngày mai 10:00 giờ VN (chi nhánh mở 00:00–23:59, trong hạn hợp đồng).
	tomorrow := time.Now().In(subVNLoc).AddDate(0, 0, 1)
	slot := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 10, 0, 0, 0, subVNLoc)

	var (
		wg        sync.WaitGroup
		start     = make(chan struct{})
		mu        sync.Mutex
		ok, confl int
		others    []error
	)
	for i := 1; i <= 2; i++ {
		contractID := f.contracts[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := f.svc.CreateSession(f.ctx, contractID, slot.Format(time.RFC3339))
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				ok++
			case codeOf(err) == 40900:
				confl++
			default:
				others = append(others, err)
			}
		}()
	}
	close(start)
	wg.Wait()

	require.Empty(t, others)
	assert.Equal(t, 1, ok, "1 request đặt thành công")
	assert.Equal(t, 1, confl, "1 request nhận 409 (trùng giờ HLV)")
	assert.EqualValues(t, 1, f.scalar(t, `SELECT COUNT(*) FROM pt_sessions WHERE trainer_id = ? AND status = 'SCHEDULED'`, f.trainerID))

	// Lớp bảo vệ cuối cùng: ghi thẳng một buổi chồng giờ (bỏ qua kiểm tra ở service) → EXCLUDE 23P01 → 409.
	err := repository.NewPTRepository(f.db).CreateSession(context.Background(), &entity.PTSession{
		ContractID: f.contracts[0], TrainerID: f.trainerID, MemberID: f.memberIDs[0], BranchID: f.branchID,
		StartAt: slot.Add(30 * time.Minute), EndAt: slot.Add(90 * time.Minute), Status: entity.PTSessionScheduled, Source: entity.PTSourceCounter,
	})
	require.Error(t, err)
	assert.True(t, isExclusionViolation(err))
	assert.Equal(t, 40900, codeOf(ptMapOverlap(err)))
}
