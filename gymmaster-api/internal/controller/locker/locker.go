package locker

import (
	"context"

	v1 "gymmaster-api/api/locker/v1"
	"gymmaster-api/internal/service"
)

type AdminControllerV1 struct {
	svc service.LockerService
}

type MemberControllerV1 struct {
	svc service.LockerService
}

func NewAdminV1(svc service.LockerService) *AdminControllerV1 {
	return &AdminControllerV1{svc: svc}
}

func NewMemberV1(svc service.LockerService) *MemberControllerV1 {
	return &MemberControllerV1{svc: svc}
}

func (c *AdminControllerV1) ListLockers(ctx context.Context, req *v1.ListLockersReq) (res v1.ListLockersRes, err error) {
	return c.svc.ListLockers(ctx, req)
}

func (c *AdminControllerV1) CreateLocker(ctx context.Context, req *v1.CreateLockerReq) (res *v1.CreateLockerRes, err error) {
	return c.svc.CreateLocker(ctx, req)
}

func (c *AdminControllerV1) UpdateLocker(ctx context.Context, req *v1.UpdateLockerReq) (res *v1.UpdateLockerRes, err error) {
	return c.svc.UpdateLocker(ctx, req)
}

func (c *AdminControllerV1) DeleteLocker(ctx context.Context, req *v1.DeleteLockerReq) (res *v1.DeleteLockerRes, err error) {
	if err := c.svc.DeleteLocker(ctx, req.ID); err != nil {
		return nil, err
	}
	return &v1.DeleteLockerRes{}, nil
}

func (c *AdminControllerV1) GetLockerBoard(ctx context.Context, req *v1.GetLockerBoardReq) (res *v1.GetLockerBoardRes, err error) {
	return c.svc.GetBoard(ctx, req)
}

func (c *AdminControllerV1) AssignLocker(ctx context.Context, req *v1.AssignLockerReq) (res *v1.AssignLockerRes, err error) {
	return c.svc.AssignLocker(ctx, req)
}

func (c *AdminControllerV1) ReleaseLocker(ctx context.Context, req *v1.ReleaseLockerReq) (res *v1.ReleaseLockerRes, err error) {
	return c.svc.ReleaseLocker(ctx, req.ID)
}

func (c *AdminControllerV1) CreateLockerRental(ctx context.Context, req *v1.CreateLockerRentalReq) (res *v1.CreateLockerRentalRes, err error) {
	return c.svc.CreateRental(ctx, req)
}

func (c *AdminControllerV1) ListLockerRentals(ctx context.Context, req *v1.ListLockerRentalsReq) (res v1.ListLockerRentalsRes, err error) {
	return c.svc.ListRentals(ctx, req)
}

func (c *MemberControllerV1) MemberLockers(ctx context.Context, _ *v1.MemberLockersReq) (res v1.MemberLockersRes, err error) {
	return c.svc.MemberLockers(ctx)
}
