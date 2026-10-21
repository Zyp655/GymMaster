package repository

import (
	"context"
	"strings"
	"testing"

	"gorm.io/datatypes"

	"gymmaster-api/internal/entity"
)

// Update phải ghi cả guest_passes_per_month và access_hours (lỗi cũ: thiếu trong danh sách Select → sửa gói không lưu).
func TestMembershipPlanRepository_UpdatePersistsGuestPassesAndAccessHours(t *testing.T) {
	db := newDryRunDB(t)
	sqls := captureSQL(t, db)
	repo := NewMembershipPlanRepository(db)
	p := &entity.MembershipPlan{Code: "P", Name: "P", GuestPassesPerMonth: 3, AccessHours: datatypes.JSON(`[{"from":"08:00","to":"10:00"}]`)}
	p.ID = 5
	if err := repo.Update(context.Background(), p); err != nil {
		t.Fatalf("update: %v", err)
	}
	if len(*sqls) == 0 {
		t.Fatal("no SQL captured")
	}
	sql := (*sqls)[len(*sqls)-1]
	for _, col := range []string{`"guest_passes_per_month"`, `"access_hours"`, `"transfer_fee"`} {
		if !strings.Contains(sql, col) {
			t.Errorf("UPDATE thiếu cột %s: %s", col, sql)
		}
	}
}

// HasOverdueInstallment chỉ tính kỳ còn mở của hóa đơn còn nợ.
func TestCheckinRepository_HasOverdueInstallmentIgnoresSettledInvoices(t *testing.T) {
	db := newDryRunDB(t)
	sqls := captureSQL(t, db)
	repo := NewCheckinRepository(db).(CheckinRulesRepository)
	_, _ = repo.HasOverdueInstallment(context.Background(), 7, "2026-10-01")
	if len(*sqls) == 0 {
		t.Fatal("no SQL captured")
	}
	sql := (*sqls)[len(*sqls)-1]
	for _, want := range []string{"JOIN invoices inv", "inv.status IN ('UNPAID', 'PARTIAL')", "s.status IN ('PENDING', 'OVERDUE')"} {
		if !strings.Contains(sql, want) {
			t.Errorf("thiếu %q trong %s", want, sql)
		}
	}
}
