package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gymmaster-api/internal/pkg/rescode"
)

// fakeBranchOutside: số lớp/PT tương lai nằm ngoài giờ theo từng chi nhánh (giả lập kết quả repository).
var fakeBranchOutside = map[*fakeBranchRepo][2]int64{}

func (f *fakeBranchRepo) CountOutsideHours(_ context.Context, _ uint, _, _ string, _ time.Time) (int64, int64, error) {
	v := fakeBranchOutside[f]
	return v[0], v[1], nil
}

// OR-05: đổi giờ mở cửa khi còn lớp/PT tương lai ngoài giờ mới → 409 kèm số lượng; force=true thì cho lưu.
func TestBranchUpdate_HoursConflictWithScheduled(t *testing.T) {
	repo := newFakeBranchRepo()
	svc := NewBranchService(repo)
	ctx := context.Background()
	b, err := svc.Create(ctx, validBranch("HN09"))
	require.NoError(t, err)

	fakeBranchOutside[repo] = [2]int64{2, 1}
	in := validBranch("HN09")
	in.OpenTime = "10:00"
	_, err = svc.Update(ctx, b.ID, in)
	require.Error(t, err)
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err))
	assert.Contains(t, err.Error(), "2 lớp nhóm và 1 buổi PT")
	assert.Equal(t, "05:30", repo.branches[b.ID].OpenTime)

	// Không đổi giờ → không kiểm tra.
	in2 := validBranch("HN09")
	in2.Name = "Đổi tên"
	_, err = svc.Update(ctx, b.ID, in2)
	require.NoError(t, err)

	in.Force = true
	res, err := svc.Update(ctx, b.ID, in)
	require.NoError(t, err)
	assert.Equal(t, "10:00", res.OpenTime)
}
