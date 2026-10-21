package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/rescode"
)

func TestCheckinRules_AllScenarios(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 15, 10, 30, 0, 0, time.UTC)
	remaining := 5

	validInput := func() *CheckinRuleInput {
		return &CheckinRuleInput{
			Now:      now,
			BranchID: 1,
			Member: &entity.Member{
				Status: entity.MemberStatusActive,
			},
			Subscription: &entity.Subscription{
				BranchID:          1,
				Status:            entity.SubStatusActive,
				StartDate:         now.AddDate(0, -1, 0),
				EndDate:           now.AddDate(0, 1, 0),
				RemainingSessions: &remaining,
			},
			Plan: &entity.MembershipPlan{
				Type:        entity.PlanTypeTime,
				AccessScope: entity.AccessSingleBranch,
			},
			AccessHoursWindow: "08:00-17:00",
			CooldownMinutes:   60,
		}
	}

	t.Run("Hợp lệ tất cả các rule -> nil violation", func(t *testing.T) {
		v := EvaluateCheckinRules(ctx, validInput())
		assert.Nil(t, v)
	})

	t.Run("Hội viên bị khóa -> MEMBER_INACTIVE", func(t *testing.T) {
		in := validInput()
		in.Member.Status = entity.MemberStatusBanned
		v := EvaluateCheckinRules(ctx, in)
		require.NotNil(t, v)
		assert.Equal(t, entity.CheckinDenyMemberInactive, v.DenyReason)
	})

	t.Run("Chi nhánh nghỉ lễ đóng cửa -> BRANCH_CLOSED (42004)", func(t *testing.T) {
		in := validInput()
		in.BranchHolidayClosed = true
		in.BranchHolidayName = "Tết Nguyên Đán"
		v := EvaluateCheckinRules(ctx, in)
		require.NotNil(t, v)
		assert.Equal(t, rescode.BranchHolidayClosed.Code(), v.Code.Code())
		assert.Equal(t, entity.CheckinDenyBranchClosed, v.DenyReason)
	})

	t.Run("Chưa ký hợp đồng điện tử bắt buộc -> CONTRACT_UNSIGNED (42001)", func(t *testing.T) {
		in := validInput()
		in.RequireSignedContract = true
		in.HasSignedContract = false
		v := EvaluateCheckinRules(ctx, in)
		require.NotNil(t, v)
		assert.Equal(t, rescode.ContractUnsigned.Code(), v.Code.Code())
	})

	t.Run("Quá hạn kỳ trả góp -> INSTALLMENT_OVERDUE (42003)", func(t *testing.T) {
		in := validInput()
		in.HasOverdueInstallment = true
		v := EvaluateCheckinRules(ctx, in)
		require.NotNil(t, v)
		assert.Equal(t, rescode.InstallmentOverdue.Code(), v.Code.Code())
	})

	t.Run("Sai chi nhánh với gói SINGLE_BRANCH -> WRONG_BRANCH", func(t *testing.T) {
		in := validInput()
		in.BranchID = 2
		v := EvaluateCheckinRules(ctx, in)
		require.NotNil(t, v)
		assert.Equal(t, entity.CheckinDenyWrongBranch, v.DenyReason)
	})

	t.Run("Ngoài khung giờ Off-Peak -> OUTSIDE_ACCESS_HOURS (42002)", func(t *testing.T) {
		in := validInput()
		in.Now = time.Date(2026, 10, 15, 19, 15, 0, 0, time.UTC)
		v := EvaluateCheckinRules(ctx, in)
		require.NotNil(t, v)
		assert.Equal(t, rescode.OutsideAccessHours.Code(), v.Code.Code())
	})

	t.Run("Gói theo buổi hết lượt -> NO_SESSIONS_LEFT", func(t *testing.T) {
		in := validInput()
		in.Plan.Type = entity.PlanTypeSession
		zero := 0
		in.Subscription.RemainingSessions = &zero
		v := EvaluateCheckinRules(ctx, in)
		require.NotNil(t, v)
		assert.Equal(t, entity.CheckinDenyNoSessionsLeft, v.DenyReason)
	})

	t.Run("Check-in trùng trong 60 phút -> DUPLICATE", func(t *testing.T) {
		in := validInput()
		last := now.Add(-15 * time.Minute)
		in.LastCheckinAt = &last
		v := EvaluateCheckinRules(ctx, in)
		require.NotNil(t, v)
		assert.Equal(t, entity.CheckinDenyDuplicate, v.DenyReason)
	})
}

func TestOffPeakHoursRule_AccessHoursJSON(t *testing.T) {
	ctx := context.Background()
	vn := time.FixedZone("ICT", 7*3600)
	weekdayMorning := time.Date(2026, 10, 21, 10, 0, 0, 0, vn) // Thứ 4
	weekdayEvening := time.Date(2026, 10, 21, 19, 0, 0, 0, vn)
	sunday := time.Date(2026, 10, 25, 10, 0, 0, 0, vn)
	slots := `[{"dow":[1,2,3,4,5],"from":"09:00","to":"16:00"}]`

	cases := []struct {
		name string
		now  time.Time
		raw  string
		deny bool
	}{
		{"không cấu hình", weekdayEvening, `[]`, false},
		{"trong khung ngày thường", weekdayMorning, slots, false},
		{"ngoài giờ ngày thường", weekdayEvening, slots, true},
		{"chủ nhật không thuộc dow", sunday, slots, true},
		{"dạng chuỗi HH:MM-HH:MM", weekdayEvening, `["09:00-16:00"]`, true},
		{"dạng chuỗi trong khung", weekdayMorning, `["09:00-16:00"]`, false},
		{"JSON lỗi → không chặn", weekdayEvening, `{bad`, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v := EvaluateCheckinRules(ctx, &CheckinRuleInput{Now: c.now, AccessHoursJSON: c.raw}, OffPeakHoursRule{})
			if c.deny {
				require.NotNil(t, v)
				assert.Equal(t, entity.CheckinDenyOutsideAccessHours, v.DenyReason)
			} else {
				assert.Nil(t, v)
			}
		})
	}
}
