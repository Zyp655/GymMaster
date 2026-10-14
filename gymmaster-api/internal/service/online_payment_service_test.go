package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	v1 "gymmaster-api/api/payment/v1"
	"gymmaster-api/internal/config"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/pkg/vnpay"
	"gymmaster-api/internal/repository"
)

type fakeOnlinePaymentRepo struct {
	nextID uint
	txns   map[uint]*entity.OnlinePaymentTxn
	subFx  *subTestFixture
}

func newFakeOnlinePaymentRepo(subFx *subTestFixture) *fakeOnlinePaymentRepo {
	return &fakeOnlinePaymentRepo{
		txns:  map[uint]*entity.OnlinePaymentTxn{},
		subFx: subFx,
	}
}

func (f *fakeOnlinePaymentRepo) Create(_ context.Context, txn *entity.OnlinePaymentTxn) error {
	f.nextID++
	txn.ID = f.nextID
	cp := *txn
	f.txns[txn.ID] = &cp
	return nil
}

func (f *fakeOnlinePaymentRepo) FindByTxnRef(_ context.Context, txnRef string, _ bool) (*entity.OnlinePaymentTxn, error) {
	for _, t := range f.txns {
		if t.TxnRef == txnRef {
			cp := *t
			return &cp, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeOnlinePaymentRepo) UpdateFields(_ context.Context, id uint, fields map[string]any) error {
	t, ok := f.txns[id]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	for k, v := range fields {
		switch k {
		case "status":
			t.Status = v.(string)
		case "provider_txn_no":
			t.ProviderTxnNo = v.(string)
		case "bank_code":
			t.BankCode = v.(string)
		case "raw_ipn":
			t.RawIPN = v.(datatypes.JSON)
		case "paid_at":
			tm := v.(time.Time)
			t.PaidAt = &tm
		}
	}
	return nil
}

func (f *fakeOnlinePaymentRepo) List(_ context.Context, _ dto.Query, flt repository.OnlineTxnListFilter) ([]repository.OnlineTxnRow, int64, error) {
	var rows []repository.OnlineTxnRow
	for _, t := range f.txns {
		if flt.Status != "" && t.Status != flt.Status {
			continue
		}
		rows = append(rows, repository.OnlineTxnRow{OnlinePaymentTxn: *t})
	}
	return rows, int64(len(rows)), nil
}

func (f *fakeOnlinePaymentRepo) ExpirePendingBefore(_ context.Context, before time.Time) (int64, error) {
	var affected int64
	for _, t := range f.txns {
		if t.Status == entity.OnlineTxnPending && t.CreatedAt.Before(before) {
			t.Status = entity.OnlineTxnExpired
			affected++
		}
	}
	return affected, nil
}

func (f *fakeOnlinePaymentRepo) FindMemberByUserID(_ context.Context, userID uint) (*entity.Member, error) {
	for _, m := range f.subFx.repo.members {
		if m.UserID != nil && *m.UserID == userID {
			cp := *m
			return &cp, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeOnlinePaymentRepo) FindMemberByID(_ context.Context, id uint) (*entity.Member, error) {
	return f.subFx.repo.FindMember(context.Background(), id)
}

func (f *fakeOnlinePaymentRepo) FindPlanByID(_ context.Context, id uint) (*entity.MembershipPlan, error) {
	return f.subFx.repo.FindPlan(context.Background(), id)
}

func (f *fakeOnlinePaymentRepo) SetInvoiceSource(_ context.Context, invoiceID uint, source string) error {
	if inv, ok := f.subFx.invRep.invoices[invoiceID]; ok {
		inv.Source = source
	}
	return nil
}

type fakePaymentNotifier struct {
	events []struct {
		userIDs []uint
		typ     string
		title   string
		body    string
	}
}

func (n *fakePaymentNotifier) Notify(_ context.Context, userIDs []uint, typ, title, body, _ string) error {
	n.events = append(n.events, struct {
		userIDs []uint
		typ     string
		title   string
		body    string
	}{userIDs: userIDs, typ: typ, title: title, body: body})
	return nil
}

func (n *fakePaymentNotifier) NotifyPermission(context.Context, string, *uint, string, string, string, string) error {
	return nil
}

var testVNPayCfg = config.VNPayCfg{
	TmnCode:    "GYMMASTR",
	HashSecret: "GYMMASTER_VNPAY_SANDBOX_SECRET_KEY_32",
	PaymentURL: "https://sandbox.vnpayment.vn/paymentv2/vpcpay.html",
	ReturnURL:  "http://localhost:3000/payment/result",
}

func newOnlinePaymentFixture() (*onlinePaymentService, *fakeOnlinePaymentRepo, *subTestFixture, *fakePaymentNotifier) {
	subFx := newSubFixture()
	repo := newFakeOnlinePaymentRepo(subFx)
	notif := &fakePaymentNotifier{}
	promos := NewPromotionService(subFx.promos, nil).(*promotionService)
	promos.now = func() time.Time { return subFx.clock }

	svc := NewOnlinePaymentService(repo, subFx.inv, subFx.svc, promos, subNoTx{}, notif, testVNPayCfg).(*onlinePaymentService)
	svc.now = func() time.Time { return subFx.clock }
	return svc, repo, subFx, notif
}

func memberPortalCtx(userID uint) context.Context {
	return reqctx.WithAuth(context.Background(), &token.Payload{
		UserID: userID,
		Roles:  []string{consts.RoleMember},
	})
}

func buildSignedIPNParams(secret, txnRef string, amountVND float64, rspCode string) map[string]string {
	params := map[string]string{
		"vnp_TmnCode":       "GYMMASTR",
		"vnp_Amount":        fmt.Sprintf("%d", int64(amountVND*100)),
		"vnp_BankCode":      "NCB",
		"vnp_OrderInfo":     "Thanh toan hoa don",
		"vnp_ResponseCode":  rspCode,
		"vnp_TransactionNo": "14098765",
		"vnp_TxnRef":        txnRef,
	}
	params["vnp_SecureHash"] = vnpay.SignParams(secret, params)
	return params
}

func TestOnlinePayment_CheckoutAndIPNActivatesSubscription(t *testing.T) {
	svc, repo, subFx, notif := newOnlinePaymentFixture()
	ctx := memberPortalCtx(50) // Member 1 có UserID = 50

	// 1. Kiểm tra mã khuyến mãi CHAODONG (giảm 10% gói M3 1.350.000 -> còn 1.215.000).
	promoRes, err := svc.CheckPromotion(ctx, &v1.CheckPromotionReq{
		Code:   "CHAODONG",
		PlanID: 2,
	})
	require.NoError(t, err)
	assert.Equal(t, 135000.0, promoRes.Discount)
	assert.Equal(t, 1215000.0, promoRes.FinalAmount)

	// 2. Hội viên tự checkout gói M3 kèm mã CHAODONG.
	checkoutRes, err := svc.CheckoutSubscription(ctx, &v1.CheckoutSubscriptionReq{
		PlanID:        2,
		BranchID:      1,
		PromotionCode: "CHAODONG",
	}, "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, 1215000.0, checkoutRes.Amount)
	assert.NotEmpty(t, checkoutRes.TxnRef)
	assert.NotEmpty(t, checkoutRes.PaymentURL)

	// Trước khi IPN: gói ở trạng thái PENDING, hóa đơn UNPAID nguồn PORTAL.
	sub := subFx.repo.subs[checkoutRes.SubscriptionID]
	inv := subFx.invRep.invoices[checkoutRes.InvoiceID]
	assert.Equal(t, entity.SubStatusPending, sub.Status)
	assert.Equal(t, entity.InvoiceStatusUnpaid, inv.Status)
	assert.Equal(t, entity.InvoiceSourcePortal, inv.Source)

	// 3. VerifyReturn chỉ xác thực chữ ký, KHÔNG cập nhật trạng thái DB.
	ipnParams := buildSignedIPNParams(testVNPayCfg.HashSecret, checkoutRes.TxnRef, checkoutRes.Amount, "00")
	retRes, err := svc.VerifyReturn(context.Background(), ipnParams)
	require.NoError(t, err)
	assert.True(t, retRes.Success)
	assert.Equal(t, checkoutRes.InvoiceID, retRes.InvoiceID)
	assert.Equal(t, entity.SubStatusPending, subFx.repo.subs[checkoutRes.SubscriptionID].Status, "VerifyReturn không được đổi trạng thái")

	// 4. HandleIPN thành công → giao dịch SUCCESS, hóa đơn PAID, gói ACTIVE, bắn thông báo.
	ipnRes, err := svc.HandleIPN(context.Background(), ipnParams)
	require.NoError(t, err)
	assert.Equal(t, "00", ipnRes.RspCode)
	assert.Equal(t, entity.OnlineTxnSuccess, repo.txns[1].Status)
	assert.Equal(t, entity.InvoiceStatusPaid, subFx.invRep.invoices[checkoutRes.InvoiceID].Status)
	assert.Equal(t, entity.SubStatusActive, subFx.repo.subs[checkoutRes.SubscriptionID].Status)
	require.Len(t, subFx.invRep.payments[checkoutRes.InvoiceID], 1)
	assert.Equal(t, entity.PaymentVNPay, subFx.invRep.payments[checkoutRes.InvoiceID][0].Method)
	require.Len(t, notif.events, 1)
	assert.Equal(t, NotifyPaymentSuccess, notif.events[0].typ)

	// 5. Idempotency: VNPay gọi lại IPN lần 2 cùng TxnRef → trả 00, không tạo thêm payment.
	ipnRes2, err := svc.HandleIPN(context.Background(), ipnParams)
	require.NoError(t, err)
	assert.Equal(t, "00", ipnRes2.RspCode)
	assert.Len(t, subFx.invRep.payments[checkoutRes.InvoiceID], 1, "IPN gọi trùng không được ghi trùng payment")
}

func TestOnlinePayment_IPNErrors_Signature_Amount_Expired(t *testing.T) {
	svc, _, subFx, _ := newOnlinePaymentFixture()
	ctx := memberPortalCtx(50)

	checkoutRes, err := svc.CheckoutSubscription(ctx, &v1.CheckoutSubscriptionReq{
		PlanID: 1, // 500.000 đ
	}, "127.0.0.1")
	require.NoError(t, err)

	// 1. Sai chữ ký HMAC → 44001 PaymentInvalidSignature.
	badSigParams := buildSignedIPNParams("WRONG_SECRET", checkoutRes.TxnRef, checkoutRes.Amount, "00")
	_, err = svc.HandleIPN(context.Background(), badSigParams)
	assert.Equal(t, rescode.PaymentInvalidSignature.Code(), codeOf(err))

	// 2. Sai số tiền → 44002 PaymentAmountMismatch.
	badAmountParams := buildSignedIPNParams(testVNPayCfg.HashSecret, checkoutRes.TxnRef, 100000, "00")
	_, err = svc.HandleIPN(context.Background(), badAmountParams)
	assert.Equal(t, rescode.PaymentAmountMismatch.Code(), codeOf(err))

	// 3. Quá 15 phút (hoặc đã bị cron chuyển sang EXPIRED) → 44003 PaymentTxnExpired.
	subFx.clock = subFx.clock.Add(16 * time.Minute)
	expiredCount, err := svc.ExpirePendingTxns(context.Background(), subFx.clock)
	require.NoError(t, err)
	assert.Equal(t, 1, expiredCount)

	validParams := buildSignedIPNParams(testVNPayCfg.HashSecret, checkoutRes.TxnRef, checkoutRes.Amount, "00")
	_, err = svc.HandleIPN(context.Background(), validParams)
	assert.Equal(t, rescode.PaymentTxnExpired.Code(), codeOf(err))
	assert.Equal(t, entity.SubStatusPending, subFx.repo.subs[checkoutRes.SubscriptionID].Status)
}
