package report

import (
	"context"

	v1 "gymmaster-api/api/report/v1"
)

func (c *ControllerV1) SalesProducts(ctx context.Context, req *v1.SalesProductsReq) (res *v1.SalesProductsRes, err error) {
	return c.reportService.SalesProducts(ctx, req)
}
