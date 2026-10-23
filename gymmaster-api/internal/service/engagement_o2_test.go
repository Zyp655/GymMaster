package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"gymmaster-api/internal/entity"
)

// NU-01: tuổi trong Mifflin-St Jeor tính theo ngày sinh (chưa tới sinh nhật thì chưa cộng tuổi).
func TestCalculateMifflinCalories_AgeByBirthday(t *testing.T) {
	dob := time.Date(1995, 12, 15, 0, 0, 0, 0, time.UTC)
	m := &entity.Member{Dob: &dob, Gender: entity.MemberGenderFemale}
	bm := &entity.BodyMetric{WeightKg: 60, HeightCm: 165}
	now := time.Date(2026, 10, 22, 3, 0, 0, 0, time.UTC)
	// 30 tuổi: (600 + 1031.25 - 150 - 161) × 1.55 = 2046
	assert.Equal(t, 2046, calculateMifflinCalories(m, bm, now))
	assert.Equal(t, 30, ageOnDate(dob, now))
	// Ngày sinh nhật (giờ VN 15/12 00:30) → đủ 31 tuổi.
	assert.Equal(t, 31, ageOnDate(dob, time.Date(2026, 12, 14, 17, 30, 0, 0, time.UTC)))
	assert.Equal(t, 30, ageOnDate(dob, time.Date(2026, 12, 14, 16, 30, 0, 0, time.UTC)))
}
