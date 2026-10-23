package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "gymmaster-api/api/locker/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
)

// LK-07: không gán tủ cho hội viên không ACTIVE.
func TestLockerService_AssignRejectsInactiveMember(t *testing.T) {
	repo := newFakeLockerRepo()
	repo.members[11] = &entity.Member{BaseEntity: entity.BaseEntity{ID: 11}, MemberCode: "HV00011", FullName: "Ngừng", Status: entity.MemberStatusInactive}
	svc := NewLockerService(repo, nil, nil)
	ctx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 1, Roles: []string{consts.RoleOwner}})
	l, err := svc.CreateLocker(ctx, &v1.CreateLockerReq{BranchID: 1, Code: "Z-01", Size: "SMALL", MonthlyFee: lockerFeePtr(0)})
	require.NoError(t, err)

	_, err = svc.AssignLocker(ctx, &v1.AssignLockerReq{ID: l.ID, MemberID: 11, Type: entity.LockerRentalDaily})
	require.Error(t, err)
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
	assert.Equal(t, entity.LockerStatusAvailable, repo.lockers[l.ID].Status)

	_, err = svc.AssignLocker(ctx, &v1.AssignLockerReq{ID: l.ID, MemberID: 10, Type: entity.LockerRentalDaily})
	require.NoError(t, err)
}
