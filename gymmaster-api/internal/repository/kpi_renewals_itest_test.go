//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// D2 (FIN-RPT-07): KPI "Gia hạn" chỉ đếm sự kiện RENEWED – không đếm gói nối tiếp do nâng cấp/chuyển nhượng.
func TestComputeKPIActuals_RenewalsOnlyRenewedEvents_IT(t *testing.T) {
	db := openITDB(t)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)
	var branchID, planID, memberID, s1, s2, s3 uint
	must := func(dest *uint, sql string, args ...any) {
		t.Helper()
		if err := db.Raw(sql, args...).Scan(dest).Error; err != nil {
			t.Fatalf("seed: %v\n%s", err, sql)
		}
	}
	must(&branchID, `INSERT INTO branches (code, name, open_time, close_time) VALUES (?, ?, '06:00', '22:00') RETURNING id`,
		"ITKR"+suffix, "IT KPI "+suffix)
	must(&planID, `INSERT INTO membership_plans (code, name, type, duration_days, price) VALUES (?, 'IT gói', 'TIME', 30, 500000) RETURNING id`,
		"ITKR"+suffix)
	must(&memberID, `INSERT INTO members (full_name, phone, home_branch_id) VALUES ('IT KPI', ?, ?) RETURNING id`,
		fmt.Sprintf("09%08d", time.Now().UnixNano()%100_000_000), branchID)
	sub := func(dest *uint, prev any) {
		must(dest, `INSERT INTO subscriptions (member_id, plan_id, branch_id, start_date, end_date, price, status, previous_id)
			VALUES (?, ?, ?, CURRENT_DATE, CURRENT_DATE + 30, 500000, 'ACTIVE', ?) RETURNING id`, memberID, planID, branchID, prev)
	}
	sub(&s1, nil)
	sub(&s2, s1) // gói gia hạn (có RENEWED trên gói cũ)
	sub(&s3, s2) // gói nâng cấp (UPGRADED) – không phải gia hạn
	db.Exec(`INSERT INTO subscription_events (subscription_id, event_type) VALUES (?, 'RENEWED'), (?, 'UPGRADED')`, s1, s2)
	t.Cleanup(func() {
		db.Exec(`DELETE FROM subscription_events WHERE subscription_id IN ?`, []uint{s1, s2, s3})
		db.Exec(`DELETE FROM subscriptions WHERE member_id = ?`, memberID)
		db.Exec(`DELETE FROM members WHERE id = ?`, memberID)
		db.Exec(`DELETE FROM membership_plans WHERE id = ?`, planID)
		db.Exec(`DELETE FROM branches WHERE id = ?`, branchID)
	})

	now := time.Now()
	out, err := NewKPIRepository(db).ComputeKPIActuals(context.Background(), branchID, now.Add(-time.Hour), now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if out.Renewals != 1 {
		t.Fatalf("Renewals = %v, muốn 1 (chỉ sự kiện RENEWED)", out.Renewals)
	}
}
