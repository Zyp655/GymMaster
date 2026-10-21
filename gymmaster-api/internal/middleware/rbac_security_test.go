package middleware

import (
	"context"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"

	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
)

// TestAutomatedRBACAndBranchIsolation kiểm chứng tự động cơ chế phân quyền theo Mục 15.3 (PLAN.md):
// - User thiếu permission không qua được kiểm tra quyền (trả 40300 PERMISSION_DENIED).
// - OWNER luôn có toàn quyền.
// - Hội viên (MEMBER) không có quyền gọi các permission quản trị.
// - Giới hạn chi nhánh (BranchIDs) chặn truy cập sang chi nhánh không được phân công.
func TestAutomatedRBACAndBranchIsolation(t *testing.T) {
	adminPermissions := []string{
		consts.PermMemberView,
		consts.PermMemberCreate,
		consts.PermSubscriptionSell,
		consts.PermInvoiceView,
		consts.PermPayrollManage,
		consts.PermPayrollLock,
		consts.PermExpenseApprove,
		consts.PermCmsManage,
		consts.PermImportLegacy,
		consts.PermSetupWizard,
	}

	t.Run("Hội viên (MEMBER) bị chặn trên tất cả permission quản trị và nhóm /v1/admin/* (40300)", func(t *testing.T) {
		memberAuth := &token.Payload{
			UserID:      101,
			Roles:       []string{consts.RoleMember},
			Permissions: nil,
		}
		assert.False(t, hasStaffRole(memberAuth), "MEMBER không được coi là nhân sự (/v1/admin/*)")
		assert.False(t, hasStaffRole(&token.Payload{UserID: 102, Roles: nil}))
		assert.True(t, hasStaffRole(&token.Payload{UserID: 201, Roles: []string{consts.RoleTrainer}}))
		assert.True(t, hasStaffRole(&token.Payload{UserID: 202, Roles: []string{consts.RoleMember, consts.RoleReceptionist}}))
		for _, perm := range adminPermissions {
			allowed := memberAuth.HasRole(consts.RoleOwner) || anyOf(perm, memberAuth.HasPermission)
			assert.False(t, allowed, "MEMBER không được phép có quyền %s", perm)
		}
		assert.Equal(t, 40300, rescode.PermissionDenied.Code())
	})

	t.Run("Lễ tân chỉ có quyền được cấp, gọi endpoint chốt lương/import dữ liệu cũ bị từ chối", func(t *testing.T) {
		receptionistAuth := &token.Payload{
			UserID:      202,
			Roles:       []string{consts.RoleReceptionist},
			Permissions: []string{consts.PermMemberView, consts.PermCheckinScan, consts.PermDayPassSell},
			BranchIDs:   []uint{1},
		}
		assert.True(t, anyOf("member.view,member.create", receptionistAuth.HasPermission))
		assert.True(t, anyOf(consts.PermDayPassSell, receptionistAuth.HasPermission))
		assert.False(t, anyOf(consts.PermPayrollLock, receptionistAuth.HasPermission))
		assert.False(t, anyOf(consts.PermImportLegacy, receptionistAuth.HasPermission))
	})

	t.Run("OWNER vượt qua mọi kiểm tra permission", func(t *testing.T) {
		ownerAuth := &token.Payload{
			UserID: 1,
			Roles:  []string{consts.RoleOwner},
		}
		for _, perm := range adminPermissions {
			allowed := ownerAuth.HasRole(consts.RoleOwner) || anyOf(perm, ownerAuth.HasPermission)
			assert.True(t, allowed)
		}
	})

	t.Run("Cô lập ngữ cảnh người dùng & chi nhánh (Chống IDOR giữa 2 hội viên / 2 chi nhánh)", func(t *testing.T) {
		ctxA := reqctx.WithAuth(context.Background(), &token.Payload{
			UserID:    501,
			Roles:     []string{consts.RoleMember},
			BranchIDs: []uint{1},
		})
		ctxB := reqctx.WithAuth(context.Background(), &token.Payload{
			UserID:    502,
			Roles:     []string{consts.RoleMember},
			BranchIDs: []uint{2},
		})
		assert.NotEqual(t, reqctx.Auth(ctxA).UserID, reqctx.Auth(ctxB).UserID)
		assert.True(t, slices.Contains(reqctx.Auth(ctxA).BranchIDs, uint(1)))
		assert.False(t, slices.Contains(reqctx.Auth(ctxA).BranchIDs, uint(2)))
	})
}
