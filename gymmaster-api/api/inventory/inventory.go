// =================================================================================
// Interface controller – viết tay theo mẫu api/branch/branch.go.
// =================================================================================

package inventory

import (
	"context"

	v1 "gymmaster-api/api/inventory/v1"
)

type IInventoryV1 interface {
	Import(ctx context.Context, req *v1.ImportReq) (res *v1.ImportRes, err error)
	Transfer(ctx context.Context, req *v1.TransferReq) (res *v1.TransferRes, err error)
	Adjust(ctx context.Context, req *v1.AdjustReq) (res *v1.AdjustRes, err error)
	Stock(ctx context.Context, req *v1.StockReq) (res *v1.StockRes, err error)
	Transactions(ctx context.Context, req *v1.TransactionListReq) (res *v1.TransactionListRes, err error)
	Voucher(ctx context.Context, req *v1.VoucherGetReq) (res *v1.VoucherGetRes, err error)
	LowStock(ctx context.Context, req *v1.LowStockReq) (res *v1.LowStockRes, err error)
}
