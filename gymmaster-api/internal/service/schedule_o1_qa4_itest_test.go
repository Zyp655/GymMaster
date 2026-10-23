//go:build integration

package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	gormpg "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"gymmaster-api/internal/repository"
)

// Chạy: GYMMASTER_TEST_DSN=postgres://gymmaster:gymmaster@localhost:5432/<db>?sslmode=disable go test -tags integration \
//   -run O1QA4 ./internal/service/
//
// Kiểm chứng SQL thật của nhóm O1 (QA vòng 4): EXCLUDE/overlap theo HLV thực dạy (C1), lọc + tên người dạy thay (C3),
// thứ hạng danh sách chờ (UI-01), UNIQUE ngày lễ (C4) và cộng bù / hoàn tác gói khi xóa ngày lễ (HO-05).
// Mọi dữ liệu nằm trong một transaction được rollback khi kết thúc.

type o1ITFixture struct {
	tx                        *gorm.DB
	branchID, room1, room2    uint
	trainer2, trainer3, ctype uint
	planID                    uint
}

func o1ITSetup(t *testing.T) *o1ITFixture {
	t.Helper()
	dsn := os.Getenv("GYMMASTER_TEST_DSN")
	if dsn == "" {
		t.Skip("chưa đặt GYMMASTER_TEST_DSN")
	}
	db, err := gorm.Open(gormpg.Open(dsn), &gorm.Config{TranslateError: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("kết nối DB: %v", err)
	}
	tx := db.Begin()
	t.Cleanup(func() { tx.Rollback() })
	f := &o1ITFixture{tx: tx}
	sfx := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)
	f.branchID = f.id(t, `INSERT INTO branches (code, name, open_time, close_time) VALUES (?, ?, '00:00', '23:59') RETURNING id`, "ITO1"+sfx, "IT O1 "+sfx)
	f.room1 = f.id(t, `INSERT INTO rooms (branch_id, name, capacity) VALUES (?, 'IT P1', 50) RETURNING id`, f.branchID)
	f.room2 = f.id(t, `INSERT INTO rooms (branch_id, name, capacity) VALUES (?, 'IT P2', 50) RETURNING id`, f.branchID)
	f.trainer2 = f.id(t, `INSERT INTO users (email, password_hash, full_name) VALUES (?, 'x', 'IT HLV t2') RETURNING id`, "it-o1-t2-"+sfx+"@test.local")
	f.trainer3 = f.id(t, `INSERT INTO users (email, password_hash, full_name) VALUES (?, 'x', 'IT HLV t3') RETURNING id`, "it-o1-t3-"+sfx+"@test.local")
	f.ctype = f.id(t, `INSERT INTO class_types (code, name) VALUES (?, 'IT Yoga') RETURNING id`, "ITO1"+sfx)
	f.planID = f.id(t, `INSERT INTO membership_plans (code, name, type, duration_days, price) VALUES (?, 'IT gói', 'TIME', 30, 100000) RETURNING id`, "ITO1"+sfx)
	return f
}

func (f *o1ITFixture) id(t *testing.T, sql string, args ...any) uint {
	t.Helper()
	var v uint
	if err := f.tx.Raw(sql, args...).Scan(&v).Error; err != nil {
		t.Fatalf("seed: %v\n%s", err, sql)
	}
	return v
}

func (f *o1ITFixture) class(t *testing.T, room, trainer uint, sub *uint, start time.Time) uint {
	return f.id(t, `INSERT INTO class_schedules (class_type_id, branch_id, room_id, trainer_id, substitute_trainer_id, start_at, end_at, capacity)
		VALUES (?, ?, ?, ?, ?, ?, ?, 10) RETURNING id`, f.ctype, f.branchID, room, trainer, sub, start, start.Add(time.Hour))
}

func TestO1QA4_ActualTrainerOverlapAndFilter(t *testing.T) {
	f := o1ITSetup(t)
	ctx := context.Background()
	repo := repository.NewClassScheduleRepository(f.tx)
	start := time.Now().Add(72 * time.Hour).Truncate(time.Hour)
	a := f.class(t, f.room1, f.trainer2, &f.trainer3, start)

	// C1: HLV 3 đang dạy thay lớp A → bận; HLV 2 không còn thực dạy → rảnh.
	if busy, err := repo.TrainerOverlap(ctx, f.trainer3, start.Add(15*time.Minute), start.Add(75*time.Minute), 0); err != nil || !busy {
		t.Fatalf("HLV dạy thay phải bận: %v %v", busy, err)
	}
	if busy, err := repo.TrainerOverlap(ctx, f.trainer2, start, start.Add(time.Hour), 0); err != nil || busy {
		t.Fatalf("HLV chính đã giao dạy thay phải rảnh: %v %v", busy, err)
	}
	// Lớp bảo vệ cuối: EXCLUDE theo COALESCE(substitute_trainer_id, trainer_id) (migration 000046).
	sp := f.tx.SavePoint("ex")
	err := f.tx.Exec(`INSERT INTO class_schedules (class_type_id, branch_id, room_id, trainer_id, start_at, end_at, capacity)
		VALUES (?, ?, ?, ?, ?, ?, 10)`, f.ctype, f.branchID, f.room2, f.trainer3, start.Add(15*time.Minute), start.Add(75*time.Minute)).Error
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23P01" || pgErr.ConstraintName != "ex_class_trainer_overlap" {
		t.Fatalf("muốn 23P01 ex_class_trainer_overlap, nhận %v", err)
	}
	sp.RollbackTo("ex")

	// C3: lọc theo HLV thực dạy + tên người dạy thay.
	items, err := repo.List(ctx, repository.ClassScheduleFilter{From: start.Add(-time.Hour), To: start.Add(time.Hour), TrainerID: f.trainer3})
	if err != nil || len(items) != 1 || items[0].ID != a || items[0].SubstituteTrainerName != "IT HLV t3" || items[0].TrainerName != "IT HLV t2" {
		t.Fatalf("lọc theo HLV dạy thay sai: %+v %v", items, err)
	}
	items, err = repo.List(ctx, repository.ClassScheduleFilter{From: start.Add(-time.Hour), To: start.Add(time.Hour), TrainerID: f.trainer2})
	if err != nil || len(items) != 0 {
		t.Fatalf("HLV chính không còn thực dạy lớp A: %+v %v", items, err)
	}
	pt := repository.NewPTRepository(f.tx)
	slots, err := pt.TrainerClasses(ctx, f.trainer3, start.Add(-time.Hour), start.Add(time.Hour))
	if err != nil || len(slots) != 1 || slots[0].TrainerID != f.trainer3 || slots[0].TrainerName != "IT HLV t3" {
		t.Fatalf("lịch dạy HLV dạy thay sai: %+v %v", slots, err)
	}
}

func TestO1QA4_WaitlistRank(t *testing.T) {
	f := o1ITSetup(t)
	ctx := context.Background()
	start := time.Now().Add(72 * time.Hour).Truncate(time.Hour)
	sch := f.class(t, f.room1, f.trainer2, &f.trainer3, start)
	var ids []uint
	for i, st := range []struct {
		status string
		pos    any
	}{{"BOOKED", nil}, {"CANCELLED", nil}, {"WAITLIST", 5}, {"CANCELLED", nil}, {"WAITLIST", 8}} {
		m := f.id(t, `INSERT INTO members (full_name, phone, home_branch_id) VALUES (?, ?, ?) RETURNING id`,
			fmt.Sprintf("IT O1 HV %d", i), fmt.Sprintf("09%08d", time.Now().UnixNano()%100_000_000+int64(i)), f.branchID)
		ids = append(ids, f.id(t, `INSERT INTO class_bookings (schedule_id, member_id, status, waitlist_position) VALUES (?, ?, ?, ?) RETURNING id`,
			sch, m, st.status, st.pos))
	}
	repo := repository.NewClassBookingRepository(f.tx)
	for idx, want := range map[int]int{2: 1, 4: 2} {
		d, err := repo.FindDetail(ctx, ids[idx])
		if err != nil || d.WaitlistRank == nil || *d.WaitlistRank != want {
			t.Fatalf("booking %d: waitlist_rank muốn %d, nhận %v (pos %v) err=%v", ids[idx], want, d.WaitlistRank, d.WaitlistPosition, err)
		}
		if d.TrainerID != f.trainer3 || d.TrainerName != "IT HLV t3" {
			t.Fatalf("booking phải hiện HLV thực dạy: %+v", d)
		}
	}
	if d, err := repo.FindDetail(ctx, ids[0]); err != nil || d.WaitlistRank != nil {
		t.Fatalf("BOOKED không có waitlist_rank: %+v %v", d, err)
	}
}

func TestO1QA4_HolidayUniqueAndExtensionRevert(t *testing.T) {
	f := o1ITSetup(t)
	ctx := context.Background()
	repo := repository.NewScheduleExtRepository(f.tx)
	day := time.Now().AddDate(0, 0, 10)
	dStr := day.Format("2006-01-02")
	m := f.id(t, `INSERT INTO members (full_name, phone, home_branch_id) VALUES ('IT O1 gói', ?, ?) RETURNING id`,
		fmt.Sprintf("07%08d", time.Now().UnixNano()%100_000_000), f.branchID)
	sub := f.id(t, `INSERT INTO subscriptions (member_id, plan_id, branch_id, start_date, end_date, price, status)
		VALUES (?, ?, ?, CURRENT_DATE - 1, CURRENT_DATE + 29, 100000, 'ACTIVE') RETURNING id`, m, f.planID, f.branchID)
	endDate := func() string {
		var s string
		f.tx.Raw(`SELECT end_date::text FROM subscriptions WHERE id = ?`, sub).Scan(&s)
		return s
	}
	orig := endDate()

	h1 := f.id(t, `INSERT INTO branch_holidays (branch_id, date, name, extend_subs) VALUES (?, ?, 'IT lễ CN', true) RETURNING id`, f.branchID, dStr)
	// C4: UNIQUE (COALESCE(branch_id, 0), date).
	sp := f.tx.SavePoint("dup")
	if err := f.tx.Exec(`INSERT INTO branch_holidays (branch_id, date, name) VALUES (?, ?, 'trùng')`, f.branchID, dStr).Error; !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("lễ trùng chi nhánh/ngày phải vi phạm unique, nhận %v", err)
	}
	sp.RollbackTo("dup")
	if dup, err := repo.BranchHolidayExists(ctx, &f.branchID, day, 0); err != nil || !dup {
		t.Fatalf("BranchHolidayExists: %v %v", dup, err)
	}

	n, err := repo.ExtendActiveTimeSubscriptions(ctx, h1, &f.branchID, day)
	if err != nil || n < 1 {
		t.Fatalf("cộng bù: %d %v", n, err)
	}
	extended := endDate()
	if extended == orig {
		t.Fatalf("end_date chưa được cộng bù")
	}
	// HO-05: lễ toàn hệ thống cùng ngày không cộng thêm lần nữa cho gói đã được cộng bù.
	var exists int64
	f.tx.Raw(`SELECT count(*) FROM branch_holidays WHERE branch_id IS NULL AND date = ?`, dStr).Scan(&exists)
	if exists == 0 {
		h2 := f.id(t, `INSERT INTO branch_holidays (date, name, extend_subs) VALUES (?, 'IT lễ toàn HT', true) RETURNING id`, dStr)
		if _, err := repo.ExtendActiveTimeSubscriptions(ctx, h2, nil, day); err != nil {
			t.Fatal(err)
		}
		if endDate() != extended {
			t.Fatalf("gói bị cộng bù 2 lần cho cùng ngày: %s → %s", extended, endDate())
		}
	}
	// Xóa lễ → hoàn tác.
	if _, err := repo.RevertHolidayExtensions(ctx, h1); err != nil {
		t.Fatal(err)
	}
	if endDate() != orig {
		t.Fatalf("hoàn tác cộng bù sai: muốn %s, nhận %s", orig, endDate())
	}
	if _, err := repo.RevertHolidayExtensions(ctx, h1); err != nil || endDate() != orig {
		t.Fatalf("hoàn tác lần 2 không được trừ thêm: %s %v", endDate(), err)
	}
}
