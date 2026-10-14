package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	v1 "gymmaster-api/api/branch/v1"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
)

type fakeBranchRepo struct {
	nextID   uint
	branches map[uint]*entity.Branch
	staff    map[uint]int64
}

func newFakeBranchRepo() *fakeBranchRepo {
	return &fakeBranchRepo{branches: map[uint]*entity.Branch{}, staff: map[uint]int64{}}
}

func (f *fakeBranchRepo) List(context.Context, dto.Query) ([]entity.Branch, int64, error) {
	out := []entity.Branch{}
	for _, b := range f.branches {
		out = append(out, *b)
	}
	return out, int64(len(out)), nil
}
func (f *fakeBranchRepo) FindByID(_ context.Context, id uint) (*entity.Branch, error) {
	if b, ok := f.branches[id]; ok {
		cp := *b
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *fakeBranchRepo) ExistsCode(_ context.Context, code string, excludeID uint) (bool, error) {
	for _, b := range f.branches {
		if b.Code == code && b.ID != excludeID {
			return true, nil
		}
	}
	return false, nil
}
func (f *fakeBranchRepo) Create(_ context.Context, b *entity.Branch) error {
	f.nextID++
	b.ID = f.nextID
	cp := *b
	f.branches[b.ID] = &cp
	return nil
}
func (f *fakeBranchRepo) Update(_ context.Context, b *entity.Branch) error {
	cp := *b
	f.branches[b.ID] = &cp
	return nil
}
func (f *fakeBranchRepo) Delete(_ context.Context, id uint, _ *uint) error {
	delete(f.branches, id)
	return nil
}
func (f *fakeBranchRepo) CountStaff(_ context.Context, id uint) (int64, error) {
	return f.staff[id], nil
}

func validBranch(code string) v1.BranchInput {
	return v1.BranchInput{Code: code, Name: "Chi nhánh " + code, OpenTime: "05:30", CloseTime: "22:00", Status: "ACTIVE"}
}

func TestBranchCreate(t *testing.T) {
	repo := newFakeBranchRepo()
	svc := NewBranchService(repo)
	ctx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 7})

	b, err := svc.Create(ctx, validBranch("HN01"))
	require.NoError(t, err)
	assert.Equal(t, "HN01", b.Code)
	assert.Equal(t, uint(7), *repo.branches[b.ID].CreatedBy, "ghi nhận người tạo")

	_, err = svc.Create(ctx, validBranch("HN01"))
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err), "trùng mã")

	in := validBranch("HN02")
	in.OpenTime, in.CloseTime = "22:00", "05:30"
	_, err = svc.Create(ctx, in)
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err), "giờ đóng cửa phải sau giờ mở cửa")
}

func TestBranchUpdate_KeepOwnCode(t *testing.T) {
	repo := newFakeBranchRepo()
	svc := NewBranchService(repo)
	ctx := context.Background()

	b, err := svc.Create(ctx, validBranch("HN01"))
	require.NoError(t, err)

	in := validBranch("HN01")
	in.Name = "Tên mới"
	updated, err := svc.Update(ctx, b.ID, in)
	require.NoError(t, err, "giữ nguyên mã của chính nó không bị coi là trùng")
	assert.Equal(t, "Tên mới", updated.Name)

	_, err = svc.Update(ctx, 999, in)
	assert.Equal(t, rescode.NotFound.Code(), codeOf(err))
}

func TestBranchDelete_BlockedWhenHasStaff(t *testing.T) {
	repo := newFakeBranchRepo()
	svc := NewBranchService(repo)
	ctx := context.Background()

	b, err := svc.Create(ctx, validBranch("HN01"))
	require.NoError(t, err)
	repo.staff[b.ID] = 3

	err = svc.Delete(ctx, b.ID)
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err))

	repo.staff[b.ID] = 0
	require.NoError(t, svc.Delete(ctx, b.ID))
	_, err = svc.Get(ctx, b.ID)
	assert.Equal(t, rescode.NotFound.Code(), codeOf(err))
}
