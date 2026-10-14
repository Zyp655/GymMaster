package inventory

import (
	"context"

	v1 "gymmaster-api/api/inventory/v1"
)

func (c *ControllerV1) Import(ctx context.Context, req *v1.ImportReq) (res *v1.ImportRes, err error) {
	return c.stockService.Import(ctx, req)
}

func (c *ControllerV1) Transfer(ctx context.Context, req *v1.TransferReq) (res *v1.TransferRes, err error) {
	return c.stockService.Transfer(ctx, req)
}

func (c *ControllerV1) Adjust(ctx context.Context, req *v1.AdjustReq) (res *v1.AdjustRes, err error) {
	return c.stockService.Adjust(ctx, req)
}

func (c *ControllerV1) Voucher(ctx context.Context, req *v1.VoucherGetReq) (res *v1.VoucherGetRes, err error) {
	return c.stockService.Voucher(ctx, req.VoucherNo)
}
