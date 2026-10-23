package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "gymmaster-api/api/engagement/v1"
	"gymmaster-api/internal/entity"
)

// NU-01: gợi ý calo & macro theo mục tiêu (giảm mỡ -20%, duy trì 0, tăng cơ +10%).
func TestBuildNutritionTargets_Formula(t *testing.T) {
	dob := time.Date(1996, 1, 1, 0, 0, 0, 0, time.UTC)
	m := &entity.Member{Dob: &dob, Gender: entity.MemberGenderMale}
	bm := &entity.BodyMetric{WeightKg: 70, HeightCm: 175}
	now := time.Date(2026, 10, 22, 3, 0, 0, 0, time.UTC)
	// 30 tuổi, nam: BMR = 700 + 1093.75 - 150 + 5 = 1648.75 → 1649; TDEE = round(1648.75 × 1.55) = 2556.
	res := &v1.MemberNutritionOverviewRes{}
	buildNutritionTargets(res, m, bm, now, "", "")
	assert.Equal(t, 1649, res.BMR)
	assert.Equal(t, 2556, res.TDEE)
	assert.Equal(t, 2556, res.RecommendedCalories, "trường cũ giữ nguyên = TDEE")
	assert.Equal(t, 1.55, res.ActivityFactor)
	assert.True(t, res.HasBodyMetric)
	assert.Equal(t, "MAINTAIN", res.Goal)
	assert.Equal(t, "DEFAULT", res.GoalSource)
	require.Len(t, res.GoalTargets, 3)

	cut, keep, bulk := res.GoalTargets[0], res.GoalTargets[1], res.GoalTargets[2]
	// Giảm mỡ: 2556 × 0.8 = 2044.8 → 2045; đạm 2.0×70 = 140g; béo 25% = 56.8 → 57g; carb (2045-560-511.25)/4 = 243.4 → 243g.
	assert.Equal(t, v1.NutritionGoalTarget{Goal: "CUT", Label: "Giảm mỡ", AdjustmentPct: -20, Calories: 2045, ProteinG: 140, CarbsG: 243, FatG: 57}, *cut)
	// Duy trì: 2556; đạm 1.6×70 = 112g; béo 30% = 85.2 → 85g; carb (2556-448-766.8)/4 = 335.3 → 335g.
	assert.Equal(t, v1.NutritionGoalTarget{Goal: "MAINTAIN", Label: "Duy trì", AdjustmentPct: 0, Calories: 2556, ProteinG: 112, CarbsG: 335, FatG: 85}, *keep)
	// Tăng cơ: 2556 × 1.1 = 2811.6 → 2812; đạm 1.8×70 = 126g; béo 25% = 78.1 → 78g; carb (2812-504-703)/4 = 401.25 → 401g.
	assert.Equal(t, v1.NutritionGoalTarget{Goal: "BULK", Label: "Tăng cơ", AdjustmentPct: 10, Calories: 2812, ProteinG: 126, CarbsG: 401, FatG: 78}, *bulk)
	assert.Same(t, keep, res.GoalTarget)

	// Macro khớp năng lượng (sai số làm tròn ≤ 10 kcal).
	for _, g := range res.GoalTargets {
		kcal := g.ProteinG*4 + g.CarbsG*4 + g.FatG*9
		assert.InDelta(t, g.Calories, kcal, 10, g.Goal)
	}

	res2 := &v1.MemberNutritionOverviewRes{}
	buildNutritionTargets(res2, m, bm, now, "CUT", "QUERY")
	assert.Equal(t, "CUT", res2.Goal)
	assert.Equal(t, 2045, res2.GoalTarget.Calories)
}

func TestBuildNutritionTargets_NoBodyMetricAndFloor(t *testing.T) {
	now := time.Date(2026, 10, 22, 3, 0, 0, 0, time.UTC)
	res := &v1.MemberNutritionOverviewRes{}
	buildNutritionTargets(res, &entity.Member{}, nil, now, "BULK", "QUERY")
	assert.False(t, res.HasBodyMetric)
	assert.Equal(t, 0, res.BMR)
	assert.Equal(t, 2000, res.TDEE)
	// Chưa có cân nặng: đạm theo % năng lượng – tăng cơ 2200 kcal, đạm 25% = 137.5 → 138g (làm tròn nửa lên).
	assert.Equal(t, 2200, res.GoalTarget.Calories)
	assert.Equal(t, 138, res.GoalTarget.ProteinG)

	// Sàn 1200 kcal khi giảm mỡ từ TDEE thấp.
	small := nutritionGoalTarget(nutritionGoalRules[0], 1300, 40)
	assert.Equal(t, 1200, small.Calories)
}

func TestNutritionGoalParseAndInfer(t *testing.T) {
	for in, want := range map[string]string{"": "", "cut": "CUT", " Bulk ": "BULK", "MAINTAIN": "MAINTAIN"} {
		got, err := parseNutritionGoal(in)
		assert.NoError(t, err, in)
		assert.Equal(t, want, got, in)
	}
	_, err := parseNutritionGoal("LOSE_ALL")
	assert.Equal(t, 40000, codeOf(err))

	assert.Equal(t, "CUT", inferNutritionGoal("Giảm mỡ bụng"))
	assert.Equal(t, "CUT", inferNutritionGoal("Siết cơ trước giải"))
	assert.Equal(t, "BULK", inferNutritionGoal("Tăng cơ"))
	assert.Equal(t, "MAINTAIN", inferNutritionGoal("Giảm mỡ, tăng cơ"))
	assert.Equal(t, "MAINTAIN", inferNutritionGoal("Duy trì vóc dáng"))
	assert.Equal(t, "", inferNutritionGoal("Ăn lành mạnh"))
}
