package device

import (
	"context"
	"strings"

	"github.com/gogf/gf/v2/frame/g"

	v1 "gymmaster-api/api/device/v1"
	"gymmaster-api/internal/service"
)

type AdminControllerV1 struct {
	svc service.DeviceService
}

type PublicControllerV1 struct {
	svc service.DeviceService
}

func NewAdminV1(svc service.DeviceService) *AdminControllerV1 {
	return &AdminControllerV1{svc: svc}
}

func NewPublicV1(svc service.DeviceService) *PublicControllerV1 {
	return &PublicControllerV1{svc: svc}
}

func (c *AdminControllerV1) ListAccessDevices(ctx context.Context, req *v1.ListAccessDevicesReq) (res v1.ListAccessDevicesRes, err error) {
	return c.svc.List(ctx, req)
}

func (c *AdminControllerV1) CreateAccessDevice(ctx context.Context, req *v1.CreateAccessDeviceReq) (res *v1.CreateAccessDeviceRes, err error) {
	return c.svc.Create(ctx, req)
}

func (c *AdminControllerV1) UpdateAccessDevice(ctx context.Context, req *v1.UpdateAccessDeviceReq) (res *v1.UpdateAccessDeviceRes, err error) {
	return c.svc.Update(ctx, req)
}

func (c *AdminControllerV1) DeleteAccessDevice(ctx context.Context, req *v1.DeleteAccessDeviceReq) (res *v1.DeleteAccessDeviceRes, err error) {
	if err := c.svc.Delete(ctx, req.ID); err != nil {
		return nil, err
	}
	return &v1.DeleteAccessDeviceRes{}, nil
}

func (c *AdminControllerV1) RotateDeviceKey(ctx context.Context, req *v1.RotateDeviceKeyReq) (res *v1.RotateDeviceKeyRes, err error) {
	return c.svc.RotateKey(ctx, req.ID)
}

func extractDeviceKey(ctx context.Context, bodyKey string) string {
	if k := strings.TrimSpace(bodyKey); k != "" {
		return k
	}
	if r := g.RequestFromCtx(ctx); r != nil {
		return strings.TrimSpace(r.Header.Get("X-Device-Key"))
	}
	return ""
}

func (c *PublicControllerV1) DeviceCheckin(ctx context.Context, req *v1.DeviceCheckinReq) (res *v1.DeviceCheckinRes, err error) {
	req.DeviceKey = extractDeviceKey(ctx, req.DeviceKey)
	return c.svc.DeviceCheckin(ctx, req)
}

func (c *PublicControllerV1) DeviceHeartbeat(ctx context.Context, req *v1.DeviceHeartbeatReq) (res *v1.DeviceHeartbeatRes, err error) {
	req.DeviceKey = extractDeviceKey(ctx, req.DeviceKey)
	return c.svc.DeviceHeartbeat(ctx, req)
}
