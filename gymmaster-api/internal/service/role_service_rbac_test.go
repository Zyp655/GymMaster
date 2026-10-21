package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	rolev1 "gymmaster-api/api/role/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
)

// Chống leo thang đặc quyền khi người không phải OWNER quản lý vai trò.
func TestRoleService_NonOwnerPrivilegeEscalation(t *testing.T) {
	repo := newFakeRoleRepo()
	audit := &fakeStaffAudit{}
	svc := NewRoleService(repo, subNoTx{}, audit)
	// Quản lý chỉ có member.view + role.update, đang giữ vai trò MANAGER.
	ctx := reqctx.WithAuth(context.Background(), &token.Payload{
		UserID: 9, Roles: []string{consts.RoleManager}, Permissions: []string{"member.view", "role.update"},
	})

	// Không được tạo vai trò có quyền mình không có.
	_, err := svc.Create(ctx, &rolev1.CreateReq{Code: "X1", Name: "X1", Permissions: []string{"member.view", "member.create"}})
	require.Error(t, err)
	assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(err))

	// Tạo vai trò với quyền con của mình → được, có audit.
	created, err := svc.Create(ctx, &rolev1.CreateReq{Code: "X2", Name: "X2", Permissions: []string{"member.view"}})
	require.NoError(t, err)
	assert.Contains(t, audit.actions, AuditCreate)

	// Không được nâng quyền vai trò vượt quá quyền của mình.
	_, err = svc.Update(ctx, &rolev1.UpdateReq{ID: created.ID, Name: "X2", Permissions: []string{"member.view", "member.create"}})
	require.Error(t, err)
	assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(err))

	// Không được sửa vai trò OWNER.
	_, err = svc.Update(ctx, &rolev1.UpdateReq{ID: 1, Name: "Chủ", Permissions: []string{"member.view"}})
	require.Error(t, err)
	assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(err))

	// Không được sửa vai trò chính mình đang giữ.
	selfCtx := reqctx.WithAuth(context.Background(), &token.Payload{
		UserID: 9, Roles: []string{consts.RoleReceptionist}, Permissions: []string{"member.view", "role.update"},
	})
	_, err = svc.Update(selfCtx, &rolev1.UpdateReq{ID: 2, Name: "Lễ tân", Permissions: []string{"member.view"}})
	require.Error(t, err)
	assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(err))

	// Sửa vai trò khác trong phạm vi quyền → được.
	_, err = svc.Update(ctx, &rolev1.UpdateReq{ID: 2, Name: "Lễ tân", Permissions: []string{"member.view"}})
	require.NoError(t, err)
	assert.Contains(t, audit.actions, AuditUpdate)
}
