//go:build integration

package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
	gormpg "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	v1 "gymmaster-api/api/report/v1"
	"gymmaster-api/internal/cron"
	"gymmaster-api/internal/repository"
)

// Chạy: GYMMASTER_TEST_DSN=postgres://gymmaster:gymmaster@localhost:5432/gymmaster?sslmode=disable make itest
// Dữ liệu được tạo trong một chi nhánh tạm riêng và dọn sạch khi test kết thúc; mọi báo cáo lọc theo chi nhánh đó.

type reportITFixture struct {
	db       *gorm.DB
	svc      ReportService
	repo     repository.ReportRepository
	branchID uint
	today    time.Time
}

func reportITOpen(t *testing.T) *gorm.DB {
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

// reportITSetup: chi nhánh tạm + gói + 2 hội viên + hóa đơn (gói 1.000.000 + 2 sản phẩm × 100.000) thu đủ, hoàn 120.000;
// 3 lượt check-in ALLOWED (07h, 07h, 18h) + 1 DENIED; 1 buổi lớp sức chứa 10 có 3 lượt đặt (ATTENDED, NO_SHOW, BOOKED).
func reportITSetup(t *testing.T) reportITFixture {
	t.Helper()
	db := reportITOpen(t)
	sfx := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)
	today := reportDay(time.Now())
	at := func(h, m int) time.Time { return today.Add(time.Duration(h)*time.Hour + time.Duration(m)*time.Minute) }
	ins := func(sql string, args ...any) uint {
		t.Helper()
		var id uint
		if err := db.Raw(sql+" RETURNING id", args...).Scan(&id).Error; err != nil {
			t.Fatalf("seed: %v\n%s", err, sql)
		}
		return id
	}
	exec := func(sql string, args ...any) {
		t.Helper()
		if err := db.Exec(sql, args...).Error; err != nil {
			t.Fatalf("seed: %v\n%s", err, sql)
		}
	}

	branch := ins(`INSERT INTO branches (code, name, open_time, close_time) VALUES (?, ?, '00:00', '23:59')`, "RPT"+sfx[:6], "IT Báo cáo "+sfx)
	plan := ins(`INSERT INTO membership_plans (code, name, type, duration_days, price) VALUES (?, 'IT gói tháng', 'TIME', 30, 1000000)`, "RPT"+sfx)
	m1 := ins(`INSERT INTO members (full_name, phone, gender, dob, source, home_branch_id) VALUES ('IT Report A', ?, 'MALE', '1995-05-05', 'FACEBOOK', ?)`, "081"+sfx[:7], branch)
	m2 := ins(`INSERT INTO members (full_name, phone, gender, source, home_branch_id) VALUES ('IT Report B', ?, 'FEMALE', 'REFERRAL', ?)`, "082"+sfx[:7], branch)
	trainer := ins(`INSERT INTO users (email, password_hash, full_name) VALUES (?, 'x', 'IT HLV báo cáo')`, "rpt"+sfx+"@it.local")

	inv := ins(`INSERT INTO invoices (member_id, branch_id, subtotal, discount, total, paid_amount, refunded_amount, status, source, issued_at)
		VALUES (?, ?, 1200000, 0, 1200000, 1200000, 120000, 'PAID', 'COUNTER', ?)`, m1, branch, at(9, 0))
	sub := ins(`INSERT INTO subscriptions (member_id, plan_id, branch_id, invoice_id, start_date, end_date, price, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, 1000000, 'ACTIVE', ?)`, m1, plan, branch, inv, today, today.AddDate(0, 0, 29), at(9, 0))
	exec(`INSERT INTO subscription_events (subscription_id, event_type, created_at) VALUES (?, 'CREATED', ?)`, sub, at(9, 0))
	cat := ins(`INSERT INTO product_categories (name) VALUES (?)`, "IT cat "+sfx)
	prod := ins(`INSERT INTO products (sku, name, category_id, cost_price, sell_price) VALUES (?, 'IT Whey', ?, 60000, 100000)`, "RPT"+sfx, cat)
	exec(`INSERT INTO invoice_items (invoice_id, item_type, ref_id, description, quantity, unit_price, amount) VALUES
		(?, 'MEMBERSHIP', ?, 'Gói', 1, 1000000, 1000000), (?, 'PRODUCT', ?, 'Whey', 2, 100000, 200000)`, inv, sub, inv, prod)
	exec(`INSERT INTO payments (invoice_id, amount, method, paid_at) VALUES (?, 700000, 'CASH', ?), (?, 500000, 'TRANSFER', ?)`, inv, at(9, 0), inv, at(9, 5))
	exec(`INSERT INTO refunds (invoice_id, amount, method, reason, created_at) VALUES (?, 120000, 'CASH', 'IT', ?)`, inv, at(10, 0))

	exec(`INSERT INTO checkins (member_id, branch_id, checked_in_at, method, result) VALUES
		(?, ?, ?, 'QR', 'ALLOWED'), (?, ?, ?, 'QR', 'ALLOWED'), (?, ?, ?, 'MANUAL', 'ALLOWED'), (?, ?, ?, 'QR', 'DENIED')`,
		m1, branch, at(7, 5), m2, branch, at(7, 40), m1, branch, at(18, 0), m2, branch, at(18, 30))

	ct := ins(`INSERT INTO class_types (code, name, color) VALUES (?, 'IT Yoga', '#22C55E')`, "RPT"+sfx)
	room := ins(`INSERT INTO rooms (branch_id, name, capacity) VALUES (?, 'IT phòng', 20)`, branch)
	sch := ins(`INSERT INTO class_schedules (class_type_id, branch_id, room_id, trainer_id, start_at, end_at, capacity, booked_count)
		VALUES (?, ?, ?, ?, ?, ?, 10, 3)`, ct, branch, room, trainer, at(6, 0), at(7, 0))
	m3 := ins(`INSERT INTO members (full_name, phone, home_branch_id) VALUES ('IT Report C', ?, ?)`, "083"+sfx[:7], branch)
	exec(`INSERT INTO class_bookings (schedule_id, member_id, status) VALUES (?, ?, 'ATTENDED'), (?, ?, 'NO_SHOW'), (?, ?, 'BOOKED')`,
		sch, m1, sch, m2, sch, m3)

	t.Cleanup(func() {
		for _, st := range []string{
			`DELETE FROM class_bookings WHERE schedule_id IN (SELECT id FROM class_schedules WHERE branch_id = @b)`,
			`DELETE FROM class_schedules WHERE branch_id = @b`,
			`DELETE FROM rooms WHERE branch_id = @b`,
			`DELETE FROM class_types WHERE id = @ct`,
			`DELETE FROM checkins WHERE branch_id = @b`,
			`DELETE FROM refunds WHERE invoice_id IN (SELECT id FROM invoices WHERE branch_id = @b)`,
			`DELETE FROM payments WHERE invoice_id IN (SELECT id FROM invoices WHERE branch_id = @b)`,
			`DELETE FROM invoice_items WHERE invoice_id IN (SELECT id FROM invoices WHERE branch_id = @b)`,
			`DELETE FROM subscription_events WHERE subscription_id IN (SELECT id FROM subscriptions WHERE branch_id = @b)`,
			`DELETE FROM subscriptions WHERE branch_id = @b`,
			`DELETE FROM invoices WHERE branch_id = @b`,
			`DELETE FROM products WHERE id = @p`,
			`DELETE FROM product_categories WHERE id = @cat`,
			`DELETE FROM members WHERE home_branch_id = @b`,
			`DELETE FROM membership_plans WHERE id = @plan`,
			`DELETE FROM users WHERE id = @tr`,
			`DELETE FROM branches WHERE id = @b`,
		} {
			if err := db.Exec(st, map[string]any{"b": branch, "ct": ct, "p": prod, "cat": cat, "plan": plan, "tr": trainer}).Error; err != nil {
				t.Errorf("dọn dữ liệu: %v\n%s", err, st)
			}
		}
	})
	repo := repository.NewReportRepository(db)
	return reportITFixture{db: db, svc: NewReportService(repo), repo: repo, branchID: branch, today: today}
}

func reportITNear(t *testing.T, name string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1 {
		t.Errorf("%s = %.0f, muốn %.0f", name, got, want)
	}
}

func TestReportIT_RevenueAndProducts(t *testing.T) {
	fx := reportITSetup(t)
	ctx := context.Background()
	day := fx.today.Format("2006-01-02")
	f := v1.Filter{BranchID: fx.branchID, From: day, To: day}

	rev, err := fx.svc.Revenue(ctx, &v1.RevenueReq{Filter: f, GroupBy: v1.GroupBy{GroupBy: "day"}})
	if err != nil {
		t.Fatal(err)
	}
	reportITNear(t, "payments", rev.Totals.Payments, 1200000)
	reportITNear(t, "refunds", rev.Totals.Refunds, 120000)
	reportITNear(t, "net", rev.Totals.Net, 1080000)
	if len(rev.Series) != 1 {
		t.Fatalf("series = %d điểm, muốn 1", len(rev.Series))
	}
	// Phân bổ theo tỷ trọng dòng hóa đơn: gói 5/6, sản phẩm 1/6.
	reportITNear(t, "membership", rev.Series[0].Membership, 900000)
	reportITNear(t, "product", rev.Series[0].Product, 180000)
	methods := map[string]float64{}
	for _, m := range rev.ByMethod {
		methods[m.Key] = m.Value
	}
	reportITNear(t, "CASH", methods["CASH"], 580000)
	reportITNear(t, "TRANSFER", methods["TRANSFER"], 500000)
	if len(rev.ByBranch) != 1 || rev.ByBranch[0].BranchID != fx.branchID {
		t.Errorf("by_branch = %+v", rev.ByBranch)
	}

	// group_by=week/month dùng cùng mốc với date_trunc của Postgres.
	for _, gb := range []string{"week", "month"} {
		r, err := fx.svc.Revenue(ctx, &v1.RevenueReq{Filter: f, GroupBy: v1.GroupBy{GroupBy: gb}})
		if err != nil {
			t.Fatal(err)
		}
		reportITNear(t, gb+" series total", r.Series[0].Total, 1080000)
	}

	prod, err := fx.svc.SalesProducts(ctx, &v1.SalesProductsReq{Filter: f, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(prod.Top) != 1 || prod.Top[0].Quantity != 2 {
		t.Fatalf("top = %+v", prod.Top)
	}
	reportITNear(t, "product revenue", prod.Top[0].Revenue, 200000)
	reportITNear(t, "product cost", prod.Top[0].Cost, 120000) // không có thẻ kho → products.cost_price × qty
	reportITNear(t, "gross profit", prod.Totals.GrossProfit, 80000)

	subs, err := fx.svc.Subscriptions(ctx, &v1.SubscriptionsReq{Filter: f, GroupBy: v1.GroupBy{GroupBy: "day"}})
	if err != nil {
		t.Fatal(err)
	}
	if subs.Counts.New != 1 || len(subs.ByPlan) != 1 || subs.ByPlan[0].Sold != 1 {
		t.Errorf("subscriptions = %+v", subs)
	}
	reportITNear(t, "plan collected", subs.ByPlan[0].Collected, 900000)
}

func TestReportIT_CheckinsClassesMembers(t *testing.T) {
	fx := reportITSetup(t)
	ctx := context.Background()
	day := fx.today.Format("2006-01-02")
	f := v1.Filter{BranchID: fx.branchID, From: day, To: day}

	hm, err := fx.svc.CheckinHeatmap(ctx, &v1.CheckinHeatmapReq{Filter: f})
	if err != nil {
		t.Fatal(err)
	}
	dow := (int(fx.today.Weekday())+6)%7 + 1
	row := hm.Matrix[dow-1]
	if hm.Total != 3 || hm.Denied != 1 || hm.UniqueMembers != 2 || row.Hours[7] != 2 || row.Hours[18] != 1 {
		t.Errorf("heatmap total=%d denied=%d unique=%d h7=%d h18=%d", hm.Total, hm.Denied, hm.UniqueMembers, row.Hours[7], row.Hours[18])
	}
	if hm.Peak == nil || hm.Peak.Hour != 7 || len(hm.Daily) != 1 || hm.Daily[0].Count != 3 {
		t.Errorf("peak/daily = %+v %+v", hm.Peak, hm.Daily)
	}

	occ, err := fx.svc.ClassOccupancy(ctx, &v1.ClassOccupancyReq{Filter: f})
	if err != nil {
		t.Fatal(err)
	}
	s := occ.Summary
	if s.Sessions != 1 || s.Capacity != 10 || s.Booked != 3 || s.Attended != 1 || s.NoShow != 1 || s.FillRate != 30 || s.AttendanceRate != 50 {
		t.Errorf("occupancy summary = %+v", s)
	}
	if len(occ.ByClassType) != 1 || occ.ByClassType[0].Label != "IT Yoga" || len(occ.ByTimeSlot) != 1 || occ.ByTimeSlot[0].Key != "06:00" {
		t.Errorf("occupancy groups = %+v / %+v", occ.ByClassType, occ.ByTimeSlot)
	}

	tp, err := fx.svc.TrainerPerformance(ctx, &v1.TrainerPerformanceReq{Filter: f})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, it := range tp.Items {
		if it.TrainerName == "IT HLV báo cáo" {
			found = true
			if it.ClassAttendees != 1 {
				t.Errorf("class_attendees = %d", it.ClassAttendees)
			}
		}
	}
	if !found && time.Now().After(fx.today.Add(6*time.Hour)) {
		t.Errorf("không thấy HLV trong báo cáo: %+v", tp.Items)
	}

	mem, err := fx.svc.Members(ctx, &v1.MembersReq{Filter: f, GroupBy: v1.GroupBy{GroupBy: "day"}, Population: "new"})
	if err != nil {
		t.Fatal(err)
	}
	if mem.NewTotal != 3 || mem.TotalMembers != 3 {
		t.Errorf("members new=%d total=%d", mem.NewTotal, mem.TotalMembers)
	}
	status := map[string]int64{}
	for _, st := range mem.Status {
		status[st.Key] = st.Count
	}
	if status["ACTIVE"] != 1 || status["NONE"] != 2 {
		t.Errorf("status = %v", status)
	}

	ov, err := fx.svc.Overview(ctx, &v1.OverviewReq{BranchID: fx.branchID})
	if err != nil {
		t.Fatal(err)
	}
	if ov.Checkin.Today != 3 || ov.Classes.Today != 1 || ov.Members.Active != 1 {
		t.Errorf("overview = %+v", ov)
	}
	reportITNear(t, "overview revenue today", ov.Revenue.Today, 1080000)

	ret, err := fx.svc.Retention(ctx, &v1.RetentionReq{Filter: v1.Filter{BranchID: fx.branchID}, WindowDays: 30})
	if err != nil {
		t.Fatal(err)
	}
	if len(ret.Months) == 0 {
		t.Error("retention thiếu tháng")
	}
}

func TestReportIT_Export(t *testing.T) {
	fx := reportITSetup(t)
	day := fx.today.Format("2006-01-02")
	for _, typ := range []string{"revenue", "members", "checkins", "subscriptions", "invoices"} {
		data, name, err := fx.svc.Export(context.Background(), &v1.ExportReq{Filter: v1.Filter{BranchID: fx.branchID, From: day, To: day}, Type: typ})
		if err != nil {
			t.Fatalf("%s: %v", typ, err)
		}
		f, err := excelize.OpenReader(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("%s (%s): file không đọc được: %v", typ, name, err)
		}
		if f.SheetCount < 2 || f.GetSheetName(0) != "Tổng hợp" {
			t.Errorf("%s: sheets = %v", typ, f.GetSheetList())
		}
		rows, _ := f.GetRows(f.GetSheetName(1))
		if len(rows) < 5 {
			t.Errorf("%s: sheet chi tiết chỉ có %d dòng", typ, len(rows))
		}
		_ = f.Close()
	}
}

// Runner + store PostgreSQL: ghi job_runs, chống chạy chồng bằng advisory lock giữa hai "instance".
func TestReportIT_JobRunnerAdvisoryLock(t *testing.T) {
	db := reportITOpen(t)
	name := fmt.Sprintf("it_job_%d", time.Now().UnixNano()%1_000_000)
	t.Cleanup(func() { db.Exec("DELETE FROM job_runs WHERE job = ?", name) })

	started, release := make(chan struct{}), make(chan struct{})
	regA := cron.NewRegistry()
	if err := regA.Add(cron.Job{Name: name, Spec: "@daily", Run: func(ctx context.Context) (int, error) {
		close(started)
		<-release
		return 7, nil
	}}); err != nil {
		t.Fatal(err)
	}
	regB := cron.NewRegistry()
	_ = regB.Add(cron.Job{Name: name, Spec: "@daily", Run: func(ctx context.Context) (int, error) { return 0, errors.New("không được chạy") }})
	a, b := cron.NewRunner(regA, cron.NewPGStore(db)), cron.NewRunner(regB, cron.NewPGStore(db))

	done := make(chan error, 1)
	go func() {
		run, err := a.Run(context.Background(), name)
		if err == nil && (run.Status != cron.StatusSuccess || run.Affected != 7) {
			err = fmt.Errorf("run = %+v", run)
		}
		done <- err
	}()
	<-started
	if _, err := b.Run(context.Background(), name); !errors.Is(err, cron.ErrJobRunning) {
		t.Errorf("instance B: err = %v, muốn ErrJobRunning", err)
	}
	if _, err := a.Run(context.Background(), name); !errors.Is(err, cron.ErrJobRunning) {
		t.Errorf("cùng instance: err = %v, muốn ErrJobRunning", err)
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	// Sau khi nhả khóa, instance B chạy được (và job lỗi → FAILED).
	run, err := b.Run(context.Background(), name)
	if err != nil || run.Status != cron.StatusFailed || run.Message != "không được chạy" {
		t.Fatalf("B sau khi nhả khóa: run=%+v err=%v", run, err)
	}
	var n int64
	db.Raw("SELECT count(*) FROM job_runs WHERE job = ? AND status <> 'RUNNING' AND finished_at IS NOT NULL", name).Scan(&n)
	if n != 2 {
		t.Errorf("job_runs = %d, muốn 2", n)
	}
}
