package service

import (
	"context"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	rolev1 "gymmaster-api/api/role/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
)

// fakeRoleBranchAssign: role_id → danh sách branch_id đang gán (nil = toàn hệ thống).
var fakeRoleBranchAssign = map[*fakeRoleRepo]map[uint][]*uint{}

func (f *fakeRoleRepo) CountUsersOutsideBranches(_ context.Context, id uint, branchIDs []uint) (int64, error) {
	var n int64
	for _, b := range fakeRoleBranchAssign[f][id] {
		if b == nil || !slices.Contains(branchIDs, *b) {
			n++
		}
	}
	return n, nil
}

// A1-01/REG-A1: non-OWNER có role.update không được sửa vai trò MEMBER; role-admin chi nhánh
// không được sửa vai trò đang gán cho nhân viên chi nhánh khác.
func TestRoleService_A1_MemberRoleAndBranchScope(t *testing.T) {
	repo := newFakeRoleRepo()
	repo.roles[5] = &entity.Role{BaseEntity: entity.BaseEntity{ID: 5}, Code: consts.RoleMember, Name: "Hội viên", IsSystem: true}
	repo.roles[6] = &entity.Role{BaseEntity: entity.BaseEntity{ID: 6}, Code: "CUSTOM", Name: "Tùy biến"}
	svc := NewRoleService(repo, subNoTx{})
	ctx := reqctx.WithAuth(context.Background(), &token.Payload{
		UserID: 9, Roles: []string{"ROLE_ADMIN"}, Permissions: []string{"member.view", "role.update"},
		BranchIDs: []uint{1},
	})

	_, err := svc.Update(ctx, &rolev1.UpdateReq{ID: 5, Name: "Hội viên", Permissions: []string{"member.view"}})
	require.Error(t, err)
	assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(err))
	assert.Empty(t, repo.roles[5].Permissions)

	_, err = svc.Update(ctx, &rolev1.UpdateReq{ID: 1, Name: "Chủ", Permissions: []string{"member.view"}})
	assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(err))

	// Vai trò đang gán cho NV CN2 → chặn.
	cn1, cn2 := uint(1), uint(2)
	fakeRoleBranchAssign[repo] = map[uint][]*uint{6: {&cn1, &cn2}}
	_, err = svc.Update(ctx, &rolev1.UpdateReq{ID: 6, Name: "Tùy biến", Permissions: []string{"member.view"}})
	assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(err))

	// Chỉ gán trong CN1 → được.
	fakeRoleBranchAssign[repo] = map[uint][]*uint{6: {&cn1}}
	_, err = svc.Update(ctx, &rolev1.UpdateReq{ID: 6, Name: "Tùy biến", Permissions: []string{"member.view"}})
	require.NoError(t, err)

	// OWNER vẫn sửa được vai trò MEMBER.
	owner := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 1, Roles: []string{consts.RoleOwner}})
	_, err = svc.Update(owner, &rolev1.UpdateReq{ID: 5, Name: "Hội viên", Permissions: []string{}})
	require.NoError(t, err)
}
