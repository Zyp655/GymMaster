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

	v1 "gymmaster-api/api/subscription/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/storage/postgres"
)

// Chạy: GYMMASTER_TEST_DSN=postgres://gymmaster:gymmaster@localhost:5432/gymmaster?sslmode=disable make itest

type subITFixture struct {
	db       *gorm.DB
	subs     SubscriptionService
	invoices InvoiceService
	ctx      context.Context
	memberID uint
	planID   uint
	branchID uint
}

func subITSetup(t *testing.T) *subITFixture {
	t.Helper()
	dsn := os.Getenv("GYMMASTER_TEST_DSN")
	if dsn == "" {
		t.Skip("chưa đặt GYMMASTER_TEST_DSN")
	}
	db, err := gorm.Open(gormpg.Open(dsn), &gorm.Config{TranslateError: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("kết nối DB: %v", err)
	}
	fx := &subITFixture{db: db}
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)

	// Chi nhánh + gói riêng cho test (không đụng dữ liệu thật, không tiêu thụ mã khuyến mãi).
	if err := db.Raw(`INSERT INTO branches (code, name, open_time, close_time) VALUES (?, ?, '00:00', '23:59') RETURNING id`,
		"ITS"+suffix[len(suffix)-6:], "IT sales "+suffix).Scan(&fx.branchID).Error; err != nil {
		t.Fatalf("seed branch: %v", err)
	}
	if err := db.Raw(`INSERT INTO membership_plans (code, name, type, duration_days, price, access_scope, max_freeze_days, max_freeze_times)
		VALUES (?, 'IT gói 3 tháng', 'TIME', 90, 1350000, 'SINGLE_BRANCH', 15, 1) RETURNING id`, "ITS"+suffix).Scan(&fx.planID).Error; err != nil {
		t.Fatalf("seed plan: %v", err)
	}
	if err := db.Raw(`INSERT INTO members (full_name, phone, home_branch_id) VALUES ('IT Sales', ?, ?) RETURNING id`,
		"039"+fmt.Sprintf("%07d", time.Now().UnixNano()%10_000_000), fx.branchID).Scan(&fx.memberID).Error; err != nil {
		t.Fatalf("seed member: %v", err)
	}
	var adminID uint
	db.Raw(`SELECT id FROM users ORDER BY id LIMIT 1`).Scan(&adminID)
	fx.ctx = reqctx.WithAuth(context.Background(), &token.Payload{UserID: adminID, Roles: []string{consts.RoleOwner}})

	t.Cleanup(func() {
		subQ := `SELECT id FROM subscriptions WHERE member_id = ?`
		invQ := `SELECT id FROM invoices WHERE member_id = ?`
		db.Exec(`DELETE FROM audit_logs WHERE (entity = 'subscription' AND entity_id IN (`+subQ+`))
			OR (entity = 'invoice' AND entity_id IN (`+invQ+`))
			OR (entity = 'subscription_freeze' AND entity_id IN (SELECT id FROM subscription_freezes WHERE subscription_id IN (`+subQ+`)))`,
			fx.memberID, fx.memberID, fx.memberID)
		for _, q := range []string{
			`DELETE FROM subscription_events WHERE subscription_id IN (` + subQ + `)`,
			`DELETE FROM subscription_freezes WHERE subscription_id IN (` + subQ + `)`,
			`DELETE FROM payments WHERE invoice_id IN (` + invQ + `)`,
			`DELETE FROM refunds WHERE invoice_id IN (` + invQ + `)`,
			`UPDATE subscriptions SET previous_id = NULL WHERE member_id = ?`,
			`DELETE FROM subscriptions WHERE member_id = ?`,
			`DELETE FROM invoices WHERE member_id = ?`,
			`DELETE FROM members WHERE id = ?`,
		} {
			if err := db.Exec(q, fx.memberID).Error; err != nil {
				t.Logf("cleanup: %v", err)
			}
		}
		db.Exec(`DELETE FROM notifications WHERE type IN ('FREEZE_REQUEST', 'FREEZE_RESULT') AND body LIKE 'IT Sales (%'`)
		db.Exec(`DELETE FROM membership_plans WHERE id = ?`, fx.planID)
		db.Exec(`DELETE FROM branches WHERE id = ?`, fx.branchID)
	})

	tx := postgres.NewTxManager(db)
	fx.invoices = NewInvoiceService(repository.NewInvoiceRepository(db), NewPromotionService(repository.NewPromotionRepository(db), nil), tx, NewAuditService(db))
	fx.subs = NewSubscriptionService(repository.NewSubscriptionRepository(db), fx.invoices, tx, NewAuditService(db), NewNotifier(db))
	return fx
}

func TestSubIT_SellAndPayTwiceActivates(t *testing.T) {
	fx := subITSetup(t)
	sub, err := fx.subs.Sell(fx.ctx, v1.SellInput{MemberID: fx.memberID, PlanID: fx.planID, BranchID: fx.branchID},
		&v1.PaymentInput{Amount: 500000, Method: entity.PaymentCash})
	if err != nil {
		t.Fatalf("bán gói: %v", err)
	}
	if sub.Status != entity.SubStatusPending || sub.Invoice == nil || sub.Invoice.Status != entity.InvoiceStatusPartial || sub.Invoice.InvoiceNo == "" {
		t.Fatalf("sau lần thu 1: status=%s invoice=%+v", sub.Status, sub.Invoice)
	}
	if _, err := fx.invoices.AddPayment(fx.ctx, *sub.InvoiceID, PaymentInput{Amount: 850000, Method: entity.PaymentTransfer, Reference: "IT"}); err != nil {
		t.Fatalf("thu lần 2: %v", err)
	}
	got, err := fx.subs.Get(fx.ctx, sub.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != entity.SubStatusActive || got.Invoice.Status != entity.InvoiceStatusPaid || got.Invoice.PaidAmount != 1350000 || len(got.Invoice.Payments) != 2 {
		t.Fatalf("sau lần thu 2: status=%s invoice=%s paid=%v payments=%d", got.Status, got.Invoice.Status, got.Invoice.PaidAmount, len(got.Invoice.Payments))
	}
	var types []string
	for _, e := range got.Events {
		types = append(types, e.EventType)
	}
	if fmt.Sprint(types) != "[CREATED ACTIVATED]" {
		t.Fatalf("events = %v", types)
	}
	if got.StartDate != subFmt(subDateOf(time.Now())) {
		t.Fatalf("start_date = %s", got.StartDate)
	}

	// Gói thứ hai nối tiếp gói vừa kích hoạt.
	next, err := fx.subs.Sell(fx.ctx, v1.SellInput{MemberID: fx.memberID, PlanID: fx.planID, BranchID: fx.branchID}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if next.PreviousID == nil || *next.PreviousID != sub.ID || next.StartDate != subFmt(subAddDays(mustDate(t, got.EndDate), 1)) {
		t.Fatalf("nối gói sai: previous=%v start=%s (end cũ %s)", next.PreviousID, next.StartDate, got.EndDate)
	}
}

func mustDate(t *testing.T, s string) time.Time {
	d, err := time.Parse(consts.DateLayout, s)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestSubIT_ConcurrentPaymentsNeverExceedTotal(t *testing.T) {
	fx := subITSetup(t)
	sub, err := fx.subs.Sell(fx.ctx, v1.SellInput{MemberID: fx.memberID, PlanID: fx.planID, BranchID: fx.branchID}, nil)
	if err != nil {
		t.Fatal(err)
	}
	invID := *sub.InvoiceID

	// 2 request thu đủ tiền cùng lúc → chỉ 1 thành công.
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := range errs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = fx.invoices.AddPayment(fx.ctx, invID, PaymentInput{Amount: 1350000, Method: entity.PaymentCash})
		}(i)
	}
	wg.Wait()
	ok := 0
	for _, e := range errs {
		if e == nil {
			ok++
		}
	}
	var inv entity.Invoice
	fx.db.First(&inv, invID)
	var n int64
	fx.db.Model(&entity.Payment{}).Where("invoice_id = ?", invID).Count(&n)
	if ok != 1 || inv.PaidAmount != 1350000 || n != 1 || inv.Status != entity.InvoiceStatusPaid {
		t.Fatalf("ok=%d paid=%v payments=%d status=%s errs=%v", ok, inv.PaidAmount, n, inv.Status, errs)
	}

	// 10 request 300.000 cùng lúc cho hóa đơn 1.350.000 → đúng 4 thành công, paid = 1.200.000.
	sub2, err := fx.subs.Sell(fx.ctx, v1.SellInput{MemberID: fx.memberID, PlanID: fx.planID, BranchID: fx.branchID}, nil)
	if err != nil {
		t.Fatal(err)
	}
	inv2ID := *sub2.InvoiceID
	errs = make([]error, 10)
	for i := range errs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = fx.invoices.AddPayment(fx.ctx, inv2ID, PaymentInput{Amount: 300000, Method: entity.PaymentCash})
		}(i)
	}
	wg.Wait()
	ok = 0
	for _, e := range errs {
		if e == nil {
			ok++
		}
	}
	var inv2 entity.Invoice
	fx.db.First(&inv2, inv2ID)
	fx.db.Model(&entity.Payment{}).Where("invoice_id = ?", inv2ID).Count(&n)
	if ok != 4 || inv2.PaidAmount != 1200000 || n != 4 || inv2.PaidAmount > inv2.Total {
		t.Fatalf("ok=%d paid=%v payments=%d", ok, inv2.PaidAmount, n)
	}
}

// Các truy vấn cho cron chỉ đọc – kiểm tra cú pháp SQL trên DB thật mà không đổi dữ liệu người khác.
func TestSubIT_CronQueries(t *testing.T) {
	fx := subITSetup(t)
	repo := repository.NewSubscriptionRepository(fx.db)
	ctx := context.Background()
	today := subDateOf(time.Now())
	sub, err := fx.subs.Sell(fx.ctx, v1.SellInput{MemberID: fx.memberID, PlanID: fx.planID, BranchID: fx.branchID},
		&v1.PaymentInput{Amount: 1350000, Method: entity.PaymentCash})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.IDsToExpire(ctx, today); err != nil {
		t.Fatalf("IDsToExpire: %v", err)
	}
	if _, err := repo.StalePending(ctx, time.Now()); err != nil {
		t.Fatalf("StalePending: %v", err)
	}
	if _, err := repo.FreezeIDsToStart(ctx, today); err != nil {
		t.Fatalf("FreezeIDsToStart: %v", err)
	}
	if _, err := repo.FreezeIDsToEnd(ctx, today); err != nil {
		t.Fatalf("FreezeIDsToEnd: %v", err)
	}
	end := mustDate(t, sub.EndDate)
	rows, err := fx.subs.ListExpiring(ctx, []int{subDaysBetween(today, end)}, today)
	if err != nil {
		t.Fatalf("ListExpiring: %v", err)
	}
	found := false
	for _, r := range rows {
		if r.SubscriptionID == sub.ID && r.MemberName == "IT Sales" && r.DaysLeft == subDaysBetween(today, end) {
			found = true
		}
	}
	if !found {
		t.Fatalf("ListExpiring không thấy gói %d: %+v", sub.ID, rows)
	}
	// Gói nối tiếp → không còn trong danh sách nhắc gia hạn.
	if _, err := fx.subs.Sell(fx.ctx, v1.SellInput{MemberID: fx.memberID, PlanID: fx.planID, BranchID: fx.branchID}, nil); err != nil {
		t.Fatal(err)
	}
	rows, _ = fx.subs.ListExpiring(ctx, []int{subDaysBetween(today, end)}, today)
	for _, r := range rows {
		if r.SubscriptionID == sub.ID {
			t.Fatalf("gói đã có gói nối tiếp vẫn bị nhắc")
		}
	}
	// Bảo lưu + danh sách chờ duyệt.
	fr, err := fx.subs.RequestFreeze(fx.ctx, sub.ID, v1.FreezeInput{StartDate: subFmt(subAddDays(today, 1)), EndDate: subFmt(subAddDays(today, 3))})
	if err != nil {
		t.Fatalf("RequestFreeze: %v", err)
	}
	if fr.Days != 3 || fr.MemberName != "IT Sales" {
		t.Fatalf("freeze = %+v", fr)
	}
	if _, err := fx.subs.ApproveFreeze(fx.ctx, fr.ID, "ok"); err != nil {
		t.Fatalf("ApproveFreeze: %v", err)
	}
	ids, err := repo.FreezeIDsToStart(ctx, subAddDays(today, 1))
	if err != nil || !containsUint(ids, fr.ID) {
		t.Fatalf("FreezeIDsToStart(today+1) = %v, %v", ids, err)
	}
}

func containsUint(s []uint, v uint) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
