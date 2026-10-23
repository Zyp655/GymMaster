//go:build integration

package service

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	gormpg "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	v1 "gymmaster-api/api/loyalty/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/storage/postgres"
)

// Chạy: GYMMASTER_TEST_DSN=postgres://gymmaster:gymmaster@localhost:5432/<db>?sslmode=disable go test -tags integration ...
// FIN-LOY-03: đổi quà song song không làm điểm âm; LY-03: tham gia thử thách song song chỉ thưởng 1 lần;
// EarnPoints song song cùng nguồn chỉ cộng 1 lần.

type loyaltyITFixture struct {
	db       *gorm.DB
	svc      LoyaltyService
	branchID uint
	userID   uint
	memberID uint
}

func loyaltyITSetup(t *testing.T, points int) *loyaltyITFixture {
	t.Helper()
	dsn := os.Getenv("GYMMASTER_TEST_DSN")
	if dsn == "" {
		t.Skip("chưa đặt GYMMASTER_TEST_DSN")
	}
	db, err := gorm.Open(gormpg.Open(dsn), &gorm.Config{TranslateError: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(20)
	sfx := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)
	f := &loyaltyITFixture{db: db}
	must := func(dest *uint, sql string, args ...any) {
		t.Helper()
		if err := db.Raw(sql, args...).Scan(dest).Error; err != nil {
			t.Fatalf("seed: %v\n%s", err, sql)
		}
	}
	must(&f.branchID, `INSERT INTO branches (code, name, open_time, close_time) VALUES (?, ?, '00:00', '23:59') RETURNING id`, "ITLY"+sfx, "IT Loyalty "+sfx)
	must(&f.userID, `INSERT INTO users (email, password_hash, full_name) VALUES (?, 'x', 'IT HV điểm') RETURNING id`, "itly"+sfx+"@it.local")
	must(&f.memberID, `INSERT INTO members (user_id, member_code, full_name, phone, home_branch_id, loyalty_points) VALUES (?, ?, 'IT HV điểm', ?, ?, ?) RETURNING id`,
		f.userID, "ITLY"+sfx, fmt.Sprintf("07%08d", time.Now().UnixNano()%100_000_000), f.branchID, points)
	t.Cleanup(func() {
		db.Exec(`DELETE FROM challenge_participants WHERE member_id = ?`, f.memberID)
		db.Exec(`DELETE FROM challenges WHERE branch_id = ?`, f.branchID)
		db.Exec(`DELETE FROM checkins WHERE member_id = ?`, f.memberID)
		db.Exec(`DELETE FROM loyalty_ledgers WHERE member_id = ?`, f.memberID)
		db.Exec(`DELETE FROM promotions WHERE code LIKE ?`, "RWD-ITLY"+sfx+"%")
		db.Exec(`DELETE FROM members WHERE id = ?`, f.memberID)
		db.Exec(`DELETE FROM users WHERE id = ?`, f.userID)
		db.Exec(`DELETE FROM branches WHERE id = ?`, f.branchID)
		_ = sqlDB.Close()
	})
	f.svc = NewLoyaltyService(repository.NewLoyaltyRepository(db), postgres.NewTxManager(db), nil)
	return f
}

func (f *loyaltyITFixture) points(t *testing.T) (int, int64) {
	t.Helper()
	var pts int
	var ledgers int64
	f.db.Raw(`SELECT loyalty_points FROM members WHERE id = ?`, f.memberID).Scan(&pts)
	f.db.Raw(`SELECT count(*) FROM loyalty_ledgers WHERE member_id = ?`, f.memberID).Scan(&ledgers)
	return pts, ledgers
}

func parallel(n int, fn func(i int) error) (ok int) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if fn(i) == nil {
				mu.Lock()
				ok++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	return ok
}

func TestLoyaltyRedeem_ConcurrentNoNegative_IT(t *testing.T) {
	f := loyaltyITSetup(t, 100)
	var rewardID uint
	if err := f.db.Raw(`INSERT INTO loyalty_rewards (name, cost_points, reward_type, stock) VALUES ('IT quà', 100, 'VOUCHER', 5) RETURNING id`).Scan(&rewardID).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.db.Exec(`DELETE FROM loyalty_rewards WHERE id = ?`, rewardID) })
	ctx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: f.userID, Roles: []string{consts.RoleMember}})

	ok := parallel(5, func(int) error {
		_, err := f.svc.RedeemReward(ctx, &v1.RedeemRewardReq{RewardID: rewardID})
		return err
	})
	pts, ledgers := f.points(t)
	var stock int
	f.db.Raw(`SELECT stock FROM loyalty_rewards WHERE id = ?`, rewardID).Scan(&stock)
	if ok != 1 || pts != 0 || ledgers != 1 || stock != 4 {
		t.Fatalf("đổi quà song song: ok=%d điểm=%d ledger=%d tồn=%d (muốn 1/0/1/4)", ok, pts, ledgers, stock)
	}
}

func TestLoyaltyJoinChallenge_ConcurrentRewardOnce_IT(t *testing.T) {
	f := loyaltyITSetup(t, 0)
	var chID uint
	if err := f.db.Raw(`INSERT INTO challenges (branch_id, name, metric, target, start_date, end_date, reward_points)
		VALUES (?, 'IT thử thách', 'CHECKINS', 1, CURRENT_DATE - 1, CURRENT_DATE + 7, 100) RETURNING id`, f.branchID).Scan(&chID).Error; err != nil {
		t.Fatal(err)
	}
	f.db.Exec(`INSERT INTO checkins (member_id, branch_id, method, result) VALUES (?, ?, 'MANUAL', 'ALLOWED')`, f.memberID, f.branchID)
	ctx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: f.userID, Roles: []string{consts.RoleMember}})

	parallel(8, func(int) error {
		_, err := f.svc.JoinChallenge(ctx, &v1.JoinChallengeReq{ID: chID})
		return err
	})
	pts, ledgers := f.points(t)
	if pts != 100 || ledgers != 1 {
		t.Fatalf("tham gia song song: điểm=%d ledger=%d (muốn 100/1)", pts, ledgers)
	}
}

func TestLoyaltyEarnPoints_ConcurrentIdempotent_IT(t *testing.T) {
	f := loyaltyITSetup(t, 0)
	ref := uint(987654321)
	ok := parallel(8, func(int) error {
		awarded, _, err := f.svc.EarnPoints(context.Background(), f.memberID, 10, entity.LoyaltyReasonCheckin, "CHECKIN", &ref, "Check-in")
		if err != nil {
			return err
		}
		if !awarded {
			return fmt.Errorf("đã cộng")
		}
		return nil
	})
	pts, ledgers := f.points(t)
	if ok != 1 || pts != 10 || ledgers != 1 {
		t.Fatalf("EarnPoints song song: awarded=%d điểm=%d ledger=%d (muốn 1/10/1)", ok, pts, ledgers)
	}
	// Số điểm không hợp lệ → lỗi.
	if _, _, err := f.svc.EarnPoints(context.Background(), f.memberID, 0, entity.LoyaltyReasonCheckin, "CHECKIN", nil, ""); err == nil {
		t.Fatal("points = 0 phải lỗi")
	}
}
