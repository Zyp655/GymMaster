package class

import (
	"context"

	v1 "gymmaster-api/api/class/v1"
)

func (c *ControllerV1) ListType(ctx context.Context, req *v1.ListTypeReq) (res *v1.ListTypeRes, err error) {
	return c.typeService.List(ctx, req)
}

func (c *ControllerV1) TypeOptions(ctx context.Context, req *v1.TypeOptionsReq) (res *v1.TypeOptionsRes, err error) {
	items, err := c.typeService.Options(ctx)
	if err != nil {
		return nil, err
	}
	out := v1.TypeOptionsRes(items)
	return &out, nil
}

func (c *ControllerV1) GetType(ctx context.Context, req *v1.GetTypeReq) (res *v1.GetTypeRes, err error) {
	return c.typeService.Get(ctx, req.ID)
}

func (c *ControllerV1) CreateType(ctx context.Context, req *v1.CreateTypeReq) (res *v1.CreateTypeRes, err error) {
	return c.typeService.Create(ctx, req.ClassTypeInput)
}

func (c *ControllerV1) UpdateType(ctx context.Context, req *v1.UpdateTypeReq) (res *v1.UpdateTypeRes, err error) {
	return c.typeService.Update(ctx, req.ID, req.ClassTypeInput)
}

func (c *ControllerV1) DeleteType(ctx context.Context, req *v1.DeleteTypeReq) (res *v1.DeleteTypeRes, err error) {
	return nil, c.typeService.Delete(ctx, req.ID)
}
