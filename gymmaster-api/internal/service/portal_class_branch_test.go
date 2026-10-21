package service

import (
	"testing"
	"time"

	v1 "gymmaster-api/api/portal/v1"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/repository"
)

func TestPortalDefaultClassBranch(t *testing.T) {
	d := func(s string) time.Time { x, _ := time.Parse("2006-01-02", s); return x }
	branches := []*v1.Branch{{ID: 1, IsHome: true}, {ID: 2}}
	const today = "2026-10-21"
	plan := func(branch uint, scope, status, start, end string) repository.PortalSubscriptionPlan {
		return repository.PortalSubscriptionPlan{BranchID: branch, AccessScope: scope, Status: status, StartDate: d(start), EndDate: d(end)}
	}
	cases := []struct {
		name  string
		plans []repository.PortalSubscriptionPlan
		want  uint
	}{
		{"gói 1 chi nhánh ở chi nhánh khác chi nhánh chính → chi nhánh của gói",
			[]repository.PortalSubscriptionPlan{plan(2, "SINGLE_BRANCH", entity.SubStatusActive, "2026-10-01", "2026-12-31")}, 2},
		{"gói toàn hệ thống → chi nhánh chính",
			[]repository.PortalSubscriptionPlan{plan(2, "ALL_BRANCHES", entity.SubStatusActive, "2026-10-01", "2026-12-31")}, 1},
		{"gói tại chi nhánh chính → chi nhánh chính",
			[]repository.PortalSubscriptionPlan{plan(2, "SINGLE_BRANCH", entity.SubStatusActive, "2026-10-01", "2026-12-31"),
				plan(1, "SINGLE_BRANCH", entity.SubStatusActive, "2026-10-01", "2026-12-31")}, 1},
		{"gói hết hạn / bảo lưu / chưa bắt đầu không tính → mặc định",
			[]repository.PortalSubscriptionPlan{plan(2, "SINGLE_BRANCH", entity.SubStatusFrozen, "2026-10-01", "2026-12-31"),
				plan(2, "SINGLE_BRANCH", entity.SubStatusActive, "2026-11-01", "2026-12-31"),
				plan(2, "SINGLE_BRANCH", entity.SubStatusActive, "2026-01-01", "2026-10-20")}, 1},
	}
	for _, tc := range cases {
		if got := portalDefaultClassBranch(tc.plans, branches, 1, today); got != tc.want {
			t.Errorf("%s: %d, muốn %d", tc.name, got, tc.want)
		}
	}
}
