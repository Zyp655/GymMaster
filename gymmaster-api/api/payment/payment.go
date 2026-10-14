package payment

import (
	"context"

	v1 "gymmaster-api/api/payment/v1"
)

type IMemberPaymentV1 interface {
	CreateInvoiceVNPay(ctx context.Context, req *v1.CreateInvoiceVNPayReq) (res *v1.PaymentURLRes, err error)
	CheckoutSubscription(ctx context.Context, req *v1.CheckoutSubscriptionReq) (res *v1.CheckoutSubscriptionRes, err error)
	CheckPromotion(ctx context.Context, req *v1.CheckPromotionReq) (res *v1.CheckPromotionRes, err error)
}

type IPublicPaymentV1 interface {
	VNPayIPN(ctx context.Context, req *v1.VNPayIPNReq) (res *v1.VNPayIPNRes, err error)
	VNPayReturn(ctx context.Context, req *v1.VNPayReturnReq) (res *v1.VNPayReturnRes, err error)
}

type IAdminPaymentV1 interface {
	ListOnlineTxns(ctx context.Context, req *v1.ListOnlineTxnsReq) (res *v1.ListOnlineTxnsRes, err error)
}
