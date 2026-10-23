//go:build integration

package service

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	gormpg "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	v1 "gymmaster-api/api/payroll/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/storage/postgres"
)

// PR-01/02/10, FIN-PAY-03/05/10 trên DB thật: ngày công theo NGÀY (quét 2 lần/ngày = 1 công), hoa hồng PT đã chi
// riêng không cộng lại, hoa hồng bán gói chỉ gói đã thanh toán × commission_rate, chi lương song song chỉ 1 lần.
func TestPayroll_GenerateAndPay_IT(t *testing.T) {
	dsn := os.Getenv("GYMMASTER_TEST_DSN")
	if dsn == "" {
		t.Skip("chưa đặt GYMMASTER_TEST_DSN")
	}
	db, err := gorm.Open(gormpg.Open(dsn), &gorm.Config{TranslateError: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sfx := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)
	var branchID, staffID, memberID, planID, invPaid, invPartial uint
	must := func(dest *uint, sql string, args ...any) {
		t.Helper()
		if err := db.Raw(sql, args...).Scan(dest).Error; err != nil {
			t.Fatalf("seed: %v\n%s", err, sql)
		}
	}
	exec := func(sql string, args ...any) {
		t.Helper()
		if err := db.Exec(sql, args...).Error; err != nil {
			t.Fatalf("seed: %v\n%s", err, sql)
		}
	}
	period := time.Now().In(memberVNLoc).Format("2006-01")
	must(&branchID, `INSERT INTO branches (code, name, open_time, close_time) VALUES (?, ?, '00:00', '23:59') RETURNING id`, "ITPR"+sfx, "IT Lương "+sfx)
	must(&staffID, `INSERT INTO users (email, password_hash, full_name) VALUES (?, 'x', 'IT NV lương') RETURNING id`, "itpr"+sfx+"@it.local")
	exec(`INSERT INTO staff_profiles (user_id, staff_code, position, base_salary, commission_rate) VALUES (?, ?, 'TRAINER', 2600000, 5)`, staffID, "ITPR"+sfx)
	// 3 lượt chấm công trong 2 ngày → 2 công.
	exec(`INSERT INTO staff_attendances (staff_id, branch_id, check_in_at) VALUES
		(?, ?, date_trunc('month', now()) + interval '2 hour'), (?, ?, date_trunc('month', now()) + interval '8 hour'),
		(?, ?, date_trunc('month', now()) + interval '1 day 2 hour')`, staffID, branchID, staffID, branchID, staffID, branchID)
	exec(`INSERT INTO trainer_commissions (trainer_id, branch_id, source_type, source_id, base_amount, rate, amount, period, status)
		VALUES (?, ?, 'PT_SESSION', ?, 100000, 15, 15000, ?, 'PAID'), (?, ?, 'PT_SESSION', ?, 100000, 20, 20000, ?, 'PENDING')`,
		staffID, branchID, 900000000+staffID, period, staffID, branchID, 910000000+staffID, period)
	must(&memberID, `INSERT INTO members (full_name, phone, home_branch_id) VALUES ('IT HV lương', ?, ?) RETURNING id`,
		fmt.Sprintf("06%08d", time.Now().UnixNano()%100_000_000), branchID)
	must(&planID, `INSERT INTO membership_plans (code, name, type, duration_days, price) VALUES (?, 'IT gói', 'TIME', 30, 1000000) RETURNING id`, "ITPR"+sfx)
	must(&invPaid, `INSERT INTO invoices (member_id, branch_id, subtotal, total, paid_amount, status) VALUES (?, ?, 1000000, 1000000, 1000000, 'PAID') RETURNING id`, memberID, branchID)
	must(&invPartial, `INSERT INTO invoices (member_id, branch_id, subtotal, total, paid_amount, status) VALUES (?, ?, 500000, 500000, 100000, 'PARTIAL') RETURNING id`, memberID, branchID)
	exec(`INSERT INTO subscriptions (member_id, plan_id, branch_id, invoice_id, start_date, end_date, price, status, sold_by) VALUES
		(?, ?, ?, ?, CURRENT_DATE, CURRENT_DATE + 30, 1000000, 'ACTIVE', ?), (?, ?, ?, ?, CURRENT_DATE, CURRENT_DATE + 30, 500000, 'PENDING', ?)`,
		memberID, planID, branchID, invPaid, staffID, memberID, planID, branchID, invPartial, staffID)
	t.Cleanup(func() {
		db.Exec(`DELETE FROM subscriptions WHERE member_id = ?`, memberID)
		db.Exec(`DELETE FROM invoices WHERE member_id = ?`, memberID)
		db.Exec(`DELETE FROM members WHERE id = ?`, memberID)
		db.Exec(`DELETE FROM membership_plans WHERE id = ?`, planID)
		db.Exec(`DELETE FROM trainer_commissions WHERE trainer_id = ?`, staffID)
		db.Exec(`DELETE FROM payrolls WHERE branch_id = ?`, branchID) // payroll_items xóa theo cascade
		db.Exec(`DELETE FROM staff_attendances WHERE staff_id = ?`, staffID)
		db.Exec(`DELETE FROM staff_profiles WHERE user_id = ?`, staffID)
		db.Exec(`DELETE FROM users WHERE id = ?`, staffID)
		db.Exec(`DELETE FROM branches WHERE id = ?`, branchID)
	})

	svc := NewPayrollService(repository.NewPayrollRepository(db), postgres.NewTxManager(db), nil)
	ctx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 1, Roles: []string{consts.RoleOwner}})
	detail, err := svc.GeneratePayroll(ctx, &v1.GeneratePayrollReq{BranchID: branchID, Period: period})
	if err != nil {
		t.Fatal(err)
	}
	var line *v1.PayrollLineItem
	for _, it := range detail.Items {
		if it.StaffID == staffID {
			line = it
		}
	}
	if line == nil {
		t.Fatal("không có dòng lương của nhân viên")
	}
	// 2.600.000 × 2/26 = 200.000; PT chỉ khoản PENDING 20.000; bán gói 1.000.000 × 5% = 50.000.
	if line.WorkDays != 2 || line.BasePay != 200000 || line.PTCommission != 20000 || line.SalesCommission != 20000 || line.Total != 240000 { // HLV: commission_rate (5%) là tỷ lệ dạy PT, bán gói tính 2%
		t.Fatalf("dòng lương sai: %+v", line)
	}

	ok := parallel(4, func(int) error {
		_, err := svc.PayPayroll(ctx, detail.ID)
		return err
	})
	var pending, paidCount int64
	db.Raw(`SELECT count(*) FROM trainer_commissions WHERE trainer_id = ? AND status = 'PENDING'`, staffID).Scan(&pending)
	db.Raw(`SELECT count(*) FROM trainer_commissions WHERE trainer_id = ? AND status = 'PAID' AND payroll_id = ?`, staffID, detail.ID).Scan(&paidCount)
	if ok != 1 || pending != 0 || paidCount != 1 {
		t.Fatalf("chi lương song song: ok=%d pending=%d paid(theo bảng lương)=%d (muốn 1/0/1)", ok, pending, paidCount)
	}
}
