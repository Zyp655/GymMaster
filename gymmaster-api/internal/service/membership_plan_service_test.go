package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "gymmaster-api/api/plan/v1"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/rescode"
)

func TestNormalizePlanInput_AccessHours(t *testing.T) {
	base := v1.PlanInput{Code: "off", Name: "Off-peak", Type: entity.PlanTypeTime, DurationDays: 30, Price: 100000, GuestPassesPerMonth: 2}

	in := base
	in.AccessHours = []v1.AccessHoursSlot{{Dow: []int{1, 2}, From: "8:00", To: " 9:5"}}
	out, err := normalizePlanInput(in)
	require.NoError(t, err)
	assert.Equal(t, "08:00", out.AccessHours[0].From)
	assert.Equal(t, "09:05", out.AccessHours[0].To)
	assert.Equal(t, "8:00", in.AccessHours[0].From, "không ghi đè slice của caller")

	// Lưu vào entity giữ vé mời + khung giờ đã chuẩn hóa.
	var p entity.MembershipPlan
	applyPlanInput(&p, out)
	assert.Equal(t, 2, p.GuestPassesPerMonth)
	assert.JSONEq(t, `[{"dow":[1,2],"from":"08:00","to":"09:05"}]`, string(p.AccessHours))
	res := toPlanRes(p)
	assert.Equal(t, 2, res.GuestPassesPerMonth)
	assert.Equal(t, "08:00", res.AccessHours[0].From)

	for _, bad := range []v1.AccessHoursSlot{{From: "10:00", To: "09:00"}, {From: "25:00", To: "26:00"}, {From: "x", To: "09:00"}, {From: "08:00", To: "09:00", Dow: []int{8}}} {
		in := base
		in.AccessHours = []v1.AccessHoursSlot{bad}
		_, err := normalizePlanInput(in)
		assert.Equal(t, rescode.BadRequest.Code(), codeOf(err), bad)
	}
}

// B9 (FIN-PL-06/07): khung giờ chồng nhau trong cùng ngày và dow trùng → 400; khung liền nhau / khác ngày hợp lệ.
func TestNormalizePlanInput_AccessHoursOverlapAndDupDow(t *testing.T) {
	base := v1.PlanInput{Code: "off", Name: "Off-peak", Type: entity.PlanTypeTime, DurationDays: 30, Price: 100000}
	bad := [][]v1.AccessHoursSlot{
		{{Dow: []int{1, 1, 1}, From: "06:00", To: "09:00"}},
		{{Dow: []int{0, 7}, From: "06:00", To: "09:00"}}, // 0 và 7 cùng là CN
		{{From: "06:00", To: "10:00"}, {From: "09:00", To: "12:00"}},
		{{Dow: []int{1, 2}, From: "06:00", To: "10:00"}, {Dow: []int{2, 3}, From: "08:00", To: "09:00"}},
		{{From: "06:00", To: "10:00"}, {Dow: []int{6}, From: "07:00", To: "08:00"}}, // dow rỗng = mọi ngày
	}
	for _, slots := range bad {
		in := base
		in.AccessHours = slots
		_, err := normalizePlanInput(in)
		assert.Equal(t, rescode.BadRequest.Code(), codeOf(err), slots)
	}
	ok := [][]v1.AccessHoursSlot{
		{{From: "06:00", To: "09:00"}, {From: "09:00", To: "12:00"}},                                     // liền nhau
		{{Dow: []int{1, 2}, From: "06:00", To: "10:00"}, {Dow: []int{6, 0}, From: "06:00", To: "10:00"}}, // khác ngày
	}
	for _, slots := range ok {
		in := base
		in.AccessHours = slots
		_, err := normalizePlanInput(in)
		assert.NoError(t, err, slots)
	}
}
