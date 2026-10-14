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

	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/storage/postgres"
)

// Chạy: GYMMASTER_TEST_DSN=postgres://gymmaster:gymmaster@localhost:5432/gymmaster?sslmode=disable make itest

type checkinITFixture struct {
	db       *gorm.DB
	svc      CheckinService
	ctx      context.Context
	branchID uint
	memberID uint
	subID    uint
}

func checkinITOpen(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("GYMMASTER_TEST_DSN")
	if dsn == "" {
		t.Skip("chưa đặt GYMMASTER_TEST_DSN")
	}
	db, err := gorm.Open(gormpg.Open(dsn), &gorm.Config{TranslateError: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("kết nối DB: %v", err)
	}
	return db
}

// checkinITSetup tạo chi nhánh (mở 00:00–23:59), gói SESSION, hội viên và một subscription ACTIVE
// còn `sessions` buổi; tự dọn dữ liệu khi test kết thúc.
func checkinITSetup(t *testing.T, sessions int) checkinITFixture {
	t.Helper()
	db := checkinITOpen(t)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)
	exec := func(dest *uint, sql string, args ...any) {
		t.Helper()
		if err := db.Raw(sql, args...).Scan(dest).Error; err != nil {
			t.Fatalf("seed: %v\n%s", err, sql)
		}
	}
	var adminID, branchID, planID, memberID, subID uint
	exec(&adminID, `SELECT id FROM users ORDER BY id LIMIT 1`)
	exec(&branchID, `INSERT INTO branches (code, name, open_time, close_time) VALUES (?, ?, '00:00', '23:59') RETURNING id`,
		"ITCK"+suffix, "IT Check-in "+suffix)
	exec(&planID, `INSERT INTO membership_plans (code, name, type, duration_days, session_count, price) VALUES (?, 'IT gói 10 buổi', 'SESSION', 30, 10, 500000) RETURNING id`,
		"ITCK"+suffix)
	exec(&memberID, `INSERT INTO members (full_name, phone, home_branch_id) VALUES ('IT Hội viên', ?, ?) RETURNING id`,
		"09"+fmt.Sprintf("%08d", time.Now().UnixNano()%100_000_000), branchID)
	exec(&subID, `INSERT INTO subscriptions (member_id, plan_id, branch_id, start_date, end_date, total_sessions, remaining_sessions, price, status)
		VALUES (?, ?, ?, (now() AT TIME ZONE 'Asia/Ho_Chi_Minh')::date - 1, (now() AT TIME ZONE 'Asia/Ho_Chi_Minh')::date + 29, 10, ?, 500000, 'ACTIVE') RETURNING id`,
		memberID, planID, branchID, sessions)

	t.Cleanup(func() {
		for _, st := range []struct {
			sql string
			arg uint
		}{
			{`DELETE FROM subscription_events WHERE subscription_id = ?`, subID},
			{`DELETE FROM checkins WHERE member_id = ?`, memberID},
			{`DELETE FROM subscriptions WHERE member_id = ?`, memberID},
			{`DELETE FROM members WHERE id = ?`, memberID},
		} {
			if err := db.Exec(st.sql, st.arg).Error; err != nil {
				t.Errorf("dọn dữ liệu: %v", err)
			}
		}
		db.Exec(`DELETE FROM membership_plans WHERE id = ?`, planID)
		db.Exec(`DELETE FROM branches WHERE id = ?`, branchID)
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})

	svc := NewCheckinService(repository.NewCheckinRepository(db), postgres.NewTxManager(db), NewSettingService(db))
	ctx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: adminID, Roles: []string{consts.RoleOwner}})
	return checkinITFixture{db: db, svc: svc, ctx: ctx, branchID: branchID, memberID: memberID, subID: subID}
}

func (f checkinITFixture) remaining(t *testing.T) int {
	var r int
	if err := f.db.Raw(`SELECT remaining_sessions FROM subscriptions WHERE id = ?`, f.subID).Scan(&r).Error; err != nil {
		t.Fatal(err)
	}
	return r
}

func (f checkinITFixture) count(t *testing.T, sql string, args ...any) int64 {
	var n int64
	if err := f.db.Raw(sql, args...).Scan(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func TestCheckinIntegration_ScanFlow(t *testing.T) {
	f := checkinITSetup(t, 5)

	tok, _, err := f.svc.IssueQR(f.ctx, f.memberID)
	if err != nil {
		t.Fatal(err)
	}
	res, err := f.svc.Scan(f.ctx, tok, f.branchID)
	if err != nil || res.Result != "ALLOWED" {
		t.Fatalf("muốn ALLOWED: %+v err=%v", res, err)
	}
	if res.Subscription == nil || res.Subscription.ID != f.subID || *res.Subscription.RemainingSessions != 4 || res.Subscription.DaysLeft != 29 {
		t.Fatalf("subscription sai: %+v", res.Subscription)
	}
	if got := f.remaining(t); got != 4 {
		t.Fatalf("remaining_sessions trong DB = %d, muốn 4", got)
	}
	if n := f.count(t, `SELECT count(*) FROM subscription_events WHERE subscription_id = ? AND event_type = 'SESSION_USED'
		AND (data->>'checkin_id')::bigint = ? AND (data->>'remaining')::int = 4`, f.subID, res.Checkin.ID); n != 1 {
		t.Fatalf("thiếu subscription_events SESSION_USED")
	}

	// Quét lại → DUPLICATE, vẫn ghi log, không trừ buổi.
	again, err := f.svc.Scan(f.ctx, tok, f.branchID)
	if err != nil || again.DenyReason != "DUPLICATE" || again.PreviousCheckin == nil || again.PreviousCheckin.ID != res.Checkin.ID {
		t.Fatalf("muốn DUPLICATE: %+v err=%v", again, err)
	}
	if f.remaining(t) != 4 {
		t.Fatalf("DUPLICATE không được trừ buổi")
	}

	// QR hỏng hoàn toàn → DENIED, không ghi DB.
	bad, err := f.svc.Scan(f.ctx, "rác", f.branchID)
	if err != nil || bad.DenyReason != "QR_INVALID" || bad.Checkin != nil {
		t.Fatalf("QR hỏng: %+v err=%v", bad, err)
	}
	if n := f.count(t, `SELECT count(*) FROM checkins WHERE member_id = ?`, f.memberID); n != 2 {
		t.Fatalf("số bản ghi checkins = %d, muốn 2", n)
	}

	present, err := f.svc.Present(f.ctx, f.branchID)
	if err != nil || len(present) != 1 || present[0].CheckinID != res.Checkin.ID || present[0].PlanName == nil {
		t.Fatalf("present: %+v err=%v", present, err)
	}
	item, err := f.svc.Checkout(f.ctx, res.Checkin.ID)
	if err != nil || item.CheckedOutAt == nil {
		t.Fatalf("checkout: %+v err=%v", item, err)
	}
	if _, err := f.svc.Checkout(f.ctx, res.Checkin.ID); err == nil {
		t.Fatalf("checkout 2 lần phải lỗi")
	}

	page, err := f.svc.ListMemberCheckins(f.ctx, f.memberID, dto.PageReq{Page: 1, Limit: 10})
	if err != nil || page.Total != 2 || page.Items[0].Result != "DENIED" || page.Items[1].CheckedOutAt == nil {
		t.Fatalf("lịch sử: %+v err=%v", page, err)
	}
	stats, err := f.svc.Stats(f.ctx, f.branchID, "")
	if err != nil || stats.Allowed != 1 || stats.Denied != 1 || stats.Present != 0 || stats.UniqueMembers != 1 {
		t.Fatalf("stats: %+v err=%v", stats, err)
	}
}

// Hai lần quét đồng thời cùng hội viên, gói SESSION còn 1 buổi → đúng 1 ALLOWED.
func TestCheckinIntegration_ConcurrentScanLastSession(t *testing.T) {
	f := checkinITSetup(t, 1)
	tok, _, err := f.svc.IssueQR(f.ctx, f.memberID)
	if err != nil {
		t.Fatal(err)
	}

	const n = 2
	var (
		wg      sync.WaitGroup
		start   = make(chan struct{})
		results = make([]string, n)
		errs    = make([]error, n)
	)
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			res, err := f.svc.Scan(f.ctx, tok, f.branchID)
			errs[i] = err
			if res != nil {
				results[i] = res.Result + "/" + res.DenyReason
			}
		}(i)
	}
	close(start)
	wg.Wait()

	allowed := 0
	for i := range n {
		if errs[i] != nil {
			t.Fatalf("lần quét %d lỗi: %v", i, errs[i])
		}
		if results[i] == "ALLOWED/" {
			allowed++
		}
	}
	if allowed != 1 {
		t.Fatalf("số lượt ALLOWED = %d, muốn 1 (%v)", allowed, results)
	}
	if got := f.remaining(t); got != 0 {
		t.Fatalf("remaining_sessions = %d, muốn 0", got)
	}
	if c := f.count(t, `SELECT count(*) FROM checkins WHERE member_id = ?`, f.memberID); c != n {
		t.Fatalf("mọi lần quét đều phải ghi log: %d/%d", c, n)
	}
	if c := f.count(t, `SELECT count(*) FROM subscription_events WHERE subscription_id = ? AND event_type = 'SESSION_USED'`, f.subID); c != 1 {
		t.Fatalf("SESSION_USED = %d, muốn 1", c)
	}
	t.Logf("kết quả: %v", results)
}
