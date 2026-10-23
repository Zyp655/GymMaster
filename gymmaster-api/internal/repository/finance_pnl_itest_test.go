//go:build integration

package repository

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	gormpg "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openITDB(t *testing.T) *gorm.DB {
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

// B2 (FIN-RPT-01/05): hóa đơn 1.350.000 thu 300.000 rồi xóa nợ 1.050.000 → doanh thu P&L chỉ 300.000
// (= invoices.total), không phải Σ invoice_items.amount.
func TestCalcProfitLoss_ExcludesWriteOff_IT(t *testing.T) {
	db := openITDB(t)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)
	var branchID, inv1, inv2 uint
	must := func(dest *uint, sql string, args ...any) {
		t.Helper()
		if err := db.Raw(sql, args...).Scan(dest).Error; err != nil {
			t.Fatalf("seed: %v\n%s", err, sql)
		}
	}
	must(&branchID, `INSERT INTO branches (code, name, open_time, close_time) VALUES (?, ?, '06:00', '22:00') RETURNING id`,
		"ITPL"+suffix, "IT P&L "+suffix)
	t.Cleanup(func() {
		db.Exec(`DELETE FROM invoices WHERE branch_id = ?`, branchID)
		db.Exec(`DELETE FROM branches WHERE id = ?`, branchID)
	})
	// HĐ 1: gói 1.350.000, thu 300.000, xóa nợ 1.050.000 → total 300.000, PAID.
	must(&inv1, `INSERT INTO invoices (branch_id, subtotal, discount, write_off_amount, total, paid_amount, status)
		VALUES (?, 1350000, 0, 1050000, 300000, 300000, 'PAID') RETURNING id`, branchID)
	db.Exec(`INSERT INTO invoice_items (invoice_id, item_type, description, unit_price, amount) VALUES (?, 'MEMBERSHIP', 'Gói', 1350000, 1350000)`, inv1)
	// HĐ 2: 2 dòng (gói 900.000 + sản phẩm 100.000), thu 500.000, xóa nợ 500.000 → phân bổ 450.000 / 50.000.
	must(&inv2, `INSERT INTO invoices (branch_id, subtotal, discount, write_off_amount, total, paid_amount, status)
		VALUES (?, 1000000, 0, 500000, 500000, 500000, 'PAID') RETURNING id`, branchID)
	db.Exec(`INSERT INTO invoice_items (invoice_id, item_type, description, unit_price, amount) VALUES
		(?, 'MEMBERSHIP', 'Gói', 900000, 900000), (?, 'PRODUCT', 'Nước', 100000, 100000)`, inv2, inv2)

	loc := time.FixedZone("ICT", 7*3600)
	now := time.Now().In(loc)
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	out, err := NewFinanceRepository(db).CalcProfitLoss(context.Background(), []uint{branchID}, day, day)
	if err != nil {
		t.Fatal(err)
	}
	if out.MembershipRevenue != 750000 || out.POSRevenue != 50000 {
		t.Fatalf("doanh thu sai: membership=%v pos=%v (muốn 750000 / 50000)", out.MembershipRevenue, out.POSRevenue)
	}
}
