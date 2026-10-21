package payment

import (
	"context"
	"net/http"

	"github.com/gogf/gf/v2/frame/g"

	"gymmaster-api/api/payment"
	v1 "gymmaster-api/api/payment/v1"
	"gymmaster-api/internal/pkg/clientip"
	"gymmaster-api/internal/service"
)

type MemberControllerV1 struct {
	svc service.OnlinePaymentService
}

type PublicControllerV1 struct {
	svc service.OnlinePaymentService
}

type AdminControllerV1 struct {
	svc service.OnlinePaymentService
}

func NewMemberV1(svc service.OnlinePaymentService) payment.IMemberPaymentV1 {
	return &MemberControllerV1{svc: svc}
}

func NewPublicV1(svc service.OnlinePaymentService) payment.IPublicPaymentV1 {
	return &PublicControllerV1{svc: svc}
}

func NewAdminV1(svc service.OnlinePaymentService) payment.IAdminPaymentV1 {
	return &AdminControllerV1{svc: svc}
}

func NewV1(svc service.OnlinePaymentService) payment.IAdminPaymentV1 {
	return &AdminControllerV1{svc: svc}
}

func (c *MemberControllerV1) CreateInvoiceVNPay(ctx context.Context, req *v1.CreateInvoiceVNPayReq) (res *v1.PaymentURLRes, err error) {
	return c.svc.CreateInvoiceVNPay(ctx, req.ID, clientIP(ctx))
}

func (c *MemberControllerV1) CheckoutSubscription(ctx context.Context, req *v1.CheckoutSubscriptionReq) (res *v1.CheckoutSubscriptionRes, err error) {
	return c.svc.CheckoutSubscription(ctx, req, clientIP(ctx))
}

func (c *MemberControllerV1) CheckPromotion(ctx context.Context, req *v1.CheckPromotionReq) (res *v1.CheckPromotionRes, err error) {
	return c.svc.CheckPromotion(ctx, req)
}

// VNPayIPN trả JSON phẳng {"RspCode","Message"} với HTTP 200 theo đặc tả VNPay – ghi thẳng vào response
// để middleware Response bỏ qua envelope {code,message,data}.
func (c *PublicControllerV1) VNPayIPN(ctx context.Context, _ *v1.VNPayIPNReq) (res *v1.VNPayIPNRes, err error) {
	out, err := c.svc.HandleIPN(ctx, queryParams(ctx))
	if err != nil || out == nil {
		if err != nil {
			g.Log().Warningf(ctx, "VNPay IPN: %v", err)
		}
		out = service.VNPayIPNResponseFor(err)
	}
	if r := g.RequestFromCtx(ctx); r != nil {
		r.Response.ClearBuffer()
		r.Response.WriteHeader(http.StatusOK)
		r.Response.WriteJson(out)
		return nil, nil
	}
	return out, nil
}

func (c *PublicControllerV1) VNPayReturn(ctx context.Context, _ *v1.VNPayReturnReq) (res *v1.VNPayReturnRes, err error) {
	return c.svc.VerifyReturn(ctx, queryParams(ctx))
}

func (c *AdminControllerV1) ListOnlineTxns(ctx context.Context, req *v1.ListOnlineTxnsReq) (res *v1.ListOnlineTxnsRes, err error) {
	return c.svc.List(ctx, req)
}

func clientIP(ctx context.Context) string {
	r := g.RequestFromCtx(ctx)
	if r == nil {
		return "127.0.0.1"
	}
	return clientip.Real(r)
}

func queryParams(ctx context.Context) map[string]string {
	r := g.RequestFromCtx(ctx)
	if r == nil {
		return map[string]string{}
	}
	out := r.GetQueryMapStrStr()
	if out == nil {
		out = map[string]string{}
	}
	if r.Request != nil && r.URL != nil {
		for k, vals := range r.URL.Query() {
			if len(vals) > 0 {
				out[k] = vals[0]
			}
		}
	}
	return out
}
