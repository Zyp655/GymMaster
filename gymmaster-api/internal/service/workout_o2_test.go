package service

import (
	"context"
	"testing"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "gymmaster-api/api/workout/v1"
	"gymmaster-api/internal/pkg/rescode"
)

// WO-03: nhật ký tập không nhận thời điểm ở tương lai.
func TestWorkoutParsePerformedAt_RejectsFuture(t *testing.T) {
	now := time.Date(2026, 10, 22, 3, 0, 0, 0, time.UTC) // 10:00 giờ VN
	_, err := workoutParsePerformedAt("2026-10-27", now)
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
	_, err = workoutParsePerformedAt("2026-10-22T18:00", now)
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
	_, err = workoutParsePerformedAt(now.Add(time.Hour).Format(time.RFC3339), now)
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))

	got, err := workoutParsePerformedAt("2026-10-22", now) // hôm nay
	require.NoError(t, err)
	assert.False(t, got.After(now))
	_, err = workoutParsePerformedAt("2026-10-21T19:30", now)
	require.NoError(t, err)
	_, err = workoutParsePerformedAt(now.Add(2*time.Minute).Format(time.RFC3339), now) // lệch đồng hồ nhỏ
	require.NoError(t, err)
}

// FD-05: mức tạ có giới hạn trên (≤ 500 kg).
func TestCreateWorkoutLogReq_WeightUpperBound(t *testing.T) {
	ctx := context.Background()
	err := g.Validator().Data(&v1.CreateWorkoutLogReq{ExerciseID: 1, Sets: 3, Reps: 10, Weight: 1000}).Run(ctx)
	require.Error(t, err)
	assert.Contains(t, err.FirstError().Error(), "500 kg")
	assert.NoError(t, g.Validator().Data(&v1.CreateWorkoutLogReq{ExerciseID: 1, Sets: 3, Reps: 10, Weight: 62.5}).Run(ctx))
}

// Thông báo/khảo sát (engagement) vẫn hẹn giờ được ở tương lai: dùng parseFlexibleTime, không chặn tương lai.
func TestParseFlexibleTime_AllowsFuture(t *testing.T) {
	now := time.Date(2026, 10, 22, 3, 0, 0, 0, time.UTC)
	got, err := parseFlexibleTime("2026-12-01T08:00", now)
	require.NoError(t, err)
	assert.True(t, got.After(now))
	_, err = parseFlexibleTime("abc", now)
	assert.Error(t, err)
}
