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

	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/storage/postgres"
)

// Chạy: GYMMASTER_TEST_DSN=postgres://gymmaster:gymmaster@localhost:5432/gymmaster?sslmode=disable make itest
//
// Chứng minh chống overbooking: 50 goroutine đặt chỗ đồng thời vào lớp capacity 10
// → đúng 10 BOOKED + 40 WAITLIST (vị trí 1..40 không trùng), booked_count = 10.
// Sau đó 5 goroutine hủy đồng thời 5 lượt BOOKED → 5 người đầu waitlist được đẩy lên, booked_count vẫn = 10.

const (
	bookingITMembers  = 50
	bookingITCapacity = 10
)

type bookingITFixture struct {
	db         *gorm.DB
	svc        BookingService
	scheduleID uint
	memberIDs  []uint
}

func bookingITSetup(t *testing.T) *bookingITFixture {
	t.Helper()
	dsn := os.Getenv("GYMMASTER_TEST_DSN")
	if dsn == "" {
		t.Skip("chưa đặt GYMMASTER_TEST_DSN")
	}
	db, err := gorm.Open(gormpg.Open(dsn), &gorm.Config{TranslateError: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("kết nối DB: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(20)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)
	exec := func(dest *uint, sql string, args ...any) {
		t.Helper()
		if err := db.Raw(sql, args...).Scan(dest).Error; err != nil {
			t.Fatalf("seed: %v\n%s", err, sql)
		}
	}
	var branchID, roomID, trainerID, typeID, planID, scheduleID uint
	exec(&branchID, `INSERT INTO branches (code, name, open_time, close_time) VALUES (?, ?, '00:00', '23:59') RETURNING id`,
		"ITBK"+suffix, "IT Booking "+suffix)
	exec(&roomID, `INSERT INTO rooms (branch_id, name, capacity) VALUES (?, 'IT phòng', 50) RETURNING id`, branchID)
	exec(&trainerID, `INSERT INTO users (email, password_hash, full_name) VALUES (?, 'x', 'IT HLV') RETURNING id`,
		"it-booking-"+suffix+"@test.local")
	exec(&typeID, `INSERT INTO class_types (code, name) VALUES (?, 'IT Yoga') RETURNING id`, "ITBK"+suffix)
	exec(&planID, `INSERT INTO membership_plans (code, name, type, duration_days, price, includes_classes)
		VALUES (?, 'IT gói lớp', 'TIME', 30, 500000, true) RETURNING id`, "ITBK"+suffix)
	// Lớp bắt đầu sau 1 ngày (trong hạn đặt trước).
	exec(&scheduleID, `INSERT INTO class_schedules (class_type_id, branch_id, room_id, trainer_id, start_at, end_at, capacity)
		VALUES (?, ?, ?, ?, now() + interval '1 day', now() + interval '1 day 1 hour', ?) RETURNING id`,
		typeID, branchID, roomID, trainerID, bookingITCapacity)

	f := &bookingITFixture{db: db, scheduleID: scheduleID}
	phoneBase := time.Now().UnixNano() % 10_000_000
	for i := 0; i < bookingITMembers; i++ {
		var memberID, subID uint
		exec(&memberID, `INSERT INTO members (full_name, phone, home_branch_id) VALUES (?, ?, ?) RETURNING id`,
			fmt.Sprintf("IT HV %02d", i), fmt.Sprintf("08%07d%02d", phoneBase, i), branchID)
		exec(&subID, `INSERT INTO subscriptions (member_id, plan_id, branch_id, start_date, end_date, price, status)
			VALUES (?, ?, ?, CURRENT_DATE - 1, CURRENT_DATE + 29, 500000, 'ACTIVE') RETURNING id`, memberID, planID, branchID)
		f.memberIDs = append(f.memberIDs, memberID)
	}

	t.Cleanup(func() {
		steps := []struct {
			sql  string
			args []any
		}{
			{`DELETE FROM class_bookings WHERE schedule_id = ?`, []any{scheduleID}},
			{`DELETE FROM class_schedules WHERE id = ?`, []any{scheduleID}},
			{`DELETE FROM subscriptions WHERE member_id IN ?`, []any{f.memberIDs}},
			{`DELETE FROM members WHERE id IN ?`, []any{f.memberIDs}},
			{`DELETE FROM membership_plans WHERE id = ?`, []any{planID}},
			{`DELETE FROM class_types WHERE id = ?`, []any{typeID}},
			{`DELETE FROM rooms WHERE id = ?`, []any{roomID}},
			{`DELETE FROM users WHERE id = ?`, []any{trainerID}},
			{`DELETE FROM branches WHERE id = ?`, []any{branchID}},
		}
		for _, st := range steps {
			if err := db.Exec(st.sql, st.args...).Error; err != nil {
				t.Errorf("dọn dữ liệu: %v (%s)", err, st.sql)
			}
		}
		_ = sqlDB.Close()
	})

	f.svc = NewBookingService(
		repository.NewClassScheduleRepository(db),
		repository.NewClassBookingRepository(db),
		postgres.NewTxManager(db),
		NewSettingService(db),
		NewNotifier(db),
	)
	return f
}

func (f *bookingITFixture) count(t *testing.T, sql string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := f.db.Raw(sql, args...).Scan(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func (f *bookingITFixture) bookedCount(t *testing.T) int64 {
	return f.count(t, `SELECT booked_count FROM class_schedules WHERE id = ?`, f.scheduleID)
}

func TestBookingConcurrency_NoOverbooking(t *testing.T) {
	f := bookingITSetup(t)
	ctx := context.Background()

	var (
		wg    sync.WaitGroup
		mu    sync.Mutex
		errs  []error
		start = make(chan struct{})
	)
	for _, mid := range f.memberIDs {
		wg.Add(1)
		go func(memberID uint) {
			defer wg.Done()
			<-start
			if _, err := f.svc.Book(ctx, f.scheduleID, memberID, entity.BookingSourcePortal); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}(mid)
	}
	close(start)
	wg.Wait()
	if len(errs) > 0 {
		t.Fatalf("%d lượt đặt lỗi, ví dụ: %v", len(errs), errs[0])
	}

	booked := f.count(t, `SELECT count(*) FROM class_bookings WHERE schedule_id = ? AND status = 'BOOKED'`, f.scheduleID)
	waitlist := f.count(t, `SELECT count(*) FROM class_bookings WHERE schedule_id = ? AND status = 'WAITLIST'`, f.scheduleID)
	positions := f.count(t, `SELECT count(DISTINCT waitlist_position) FROM class_bookings WHERE schedule_id = ? AND status = 'WAITLIST'`, f.scheduleID)
	maxPos := f.count(t, `SELECT COALESCE(MAX(waitlist_position), 0) FROM class_bookings WHERE schedule_id = ? AND status = 'WAITLIST'`, f.scheduleID)
	t.Logf("BOOKED=%d WAITLIST=%d booked_count=%d", booked, waitlist, f.bookedCount(t))
	if booked != bookingITCapacity || waitlist != bookingITMembers-bookingITCapacity {
		t.Fatalf("muốn %d BOOKED + %d WAITLIST, có %d + %d", bookingITCapacity, bookingITMembers-bookingITCapacity, booked, waitlist)
	}
	if bc := f.bookedCount(t); bc != bookingITCapacity {
		t.Fatalf("booked_count = %d, muốn %d", bc, bookingITCapacity)
	}
	if positions != waitlist || maxPos != waitlist {
		t.Fatalf("vị trí waitlist phải là 1..%d không trùng (distinct=%d, max=%d)", waitlist, positions, maxPos)
	}

	// Hủy đồng thời 5 lượt BOOKED (nhân viên hủy hộ) → 5 người đầu waitlist được đẩy lên.
	var bookedIDs []uint
	if err := f.db.Raw(`SELECT id FROM class_bookings WHERE schedule_id = ? AND status = 'BOOKED' ORDER BY id LIMIT 5`, f.scheduleID).
		Scan(&bookedIDs).Error; err != nil {
		t.Fatal(err)
	}
	var firstWaitlist []uint
	if err := f.db.Raw(`SELECT id FROM class_bookings WHERE schedule_id = ? AND status = 'WAITLIST' ORDER BY waitlist_position LIMIT 5`, f.scheduleID).
		Scan(&firstWaitlist).Error; err != nil {
		t.Fatal(err)
	}
	start2 := make(chan struct{})
	for _, id := range bookedIDs {
		wg.Add(1)
		go func(bookingID uint) {
			defer wg.Done()
			<-start2
			if err := f.svc.Cancel(ctx, bookingID, false); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}(id)
	}
	close(start2)
	wg.Wait()
	if len(errs) > 0 {
		t.Fatalf("hủy lỗi: %v", errs[0])
	}
	if bc := f.bookedCount(t); bc != bookingITCapacity {
		t.Fatalf("sau khi hủy + đẩy waitlist, booked_count = %d, muốn %d", bc, bookingITCapacity)
	}
	if n := f.count(t, `SELECT count(*) FROM class_bookings WHERE schedule_id = ? AND status = 'BOOKED'`, f.scheduleID); n != bookingITCapacity {
		t.Fatalf("số BOOKED = %d, muốn %d", n, bookingITCapacity)
	}
	if n := f.count(t, `SELECT count(*) FROM class_bookings WHERE id IN ? AND status = 'BOOKED'`, firstWaitlist); n != 5 {
		t.Fatalf("5 người đầu waitlist phải được đẩy lên, có %d", n)
	}
}
