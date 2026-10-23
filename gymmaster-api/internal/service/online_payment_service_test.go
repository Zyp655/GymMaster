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
		case "recorded_amount":
			a := v.(float64)
			t.RecordedAmount = &a
		case "refunded_amount":
			t.RefundedAmount = v.(float64)
		case "note":
			t.Note = v.(string)
		default:
			panic("fakeOnlinePaymentRepo.UpdateFields: " + k)
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

func (f *fakeOnlinePaymentRepo) FindPendingByInvoice(_ context.Context, invoiceID uint, since time.Time) (*entity.OnlinePaymentTxn, error) {
	var best *entity.OnlinePaymentTxn
	for _, t := range f.txns {
		if t.InvoiceID == invoiceID && t.Status == entity.OnlineTxnPending && !t.CreatedAt.Before(since) && (best == nil || t.ID > best.ID) {
			best = t
		}
	}
	if best == nil {
		return nil, nil
	}
	cp := *best
	return &cp, nil
}

func (f *fakeOnlinePaymentRepo) FindPortalCheckout(_ context.Context, memberID, planID, branchID uint) (*repository.PortalCheckout, error) {
	var best *repository.PortalCheckout
	for _, sub := range f.subFx.repo.subs {
		if sub.MemberID != memberID || sub.PlanID != planID || sub.BranchID != branchID || sub.Status != entity.SubStatusPending || sub.InvoiceID == nil {
			continue
		}
		inv, ok := f.subFx.invRep.invoices[*sub.InvoiceID]
		if !ok || inv.Status != entity.InvoiceStatusUnpaid || inv.Source != entity.InvoiceSourcePortal || inv.PaidAmount != 0 {
			continue
		}
		if best == nil || sub.ID > best.SubscriptionID {
			code := ""
			if inv.PromotionID != nil {
				if p, ok := f.subFx.promos.items[*inv.PromotionID]; ok {
					code = p.Code
				}
			}
			best = &repository.PortalCheckout{SubscriptionID: sub.ID, InvoiceID: inv.ID, InvoiceNo: inv.InvoiceNo, Total: inv.Total, PromotionCode: code}
		}
	}
	return best, nil
}

type fakePaymentNotifier struct {
	staff  []string
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

func (n *fakePaymentNotifier) NotifyPermission(_ context.Context, perm string, _ *uint, typ, title, body, _ string) error {
	n.staff = append(n.staff, perm+"|"+typ+"|"+title+"|"+body)
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
	subFx.invRep.pendingLookup = repo.FindPendingByInvoice
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

	// Thanh toán hệ thống (IPN) không gắn người thu → received_by NULL (tránh vi phạm FK users).
	assert.Nil(t, subFx.invRep.payments[checkoutRes.InvoiceID][0].ReceivedBy)

	// 5. Idempotency: VNPay gọi lại IPN lần 2 cùng TxnRef → 02 (đã xác nhận), không tạo thêm payment.
	ipnRes2, err := svc.HandleIPN(context.Background(), ipnParams)
	require.NoError(t, err)
	assert.Equal(t, "02", ipnRes2.RspCode)
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

	// 3. IPN thành công đến trễ (giao dịch đã bị cron chuyển EXPIRED) → VNPay ĐÃ trừ tiền khách nên vẫn phải ghi
	// nhận: hóa đơn còn nợ → thu, gói kích hoạt, trả 00 (trước đây trả 02 làm mất dấu khoản tiền).
	subFx.clock = subFx.clock.Add(16 * time.Minute)
	expiredCount, err := svc.ExpirePendingTxns(context.Background(), subFx.clock)
	require.NoError(t, err)
	assert.Equal(t, 1, expiredCount)

	validParams := buildSignedIPNParams(testVNPayCfg.HashSecret, checkoutRes.TxnRef, checkoutRes.Amount, "00")
	res, err := svc.HandleIPN(context.Background(), validParams)
	require.NoError(t, err)
	assert.Equal(t, "00", res.RspCode)
	assert.Equal(t, entity.InvoiceStatusPaid, subFx.invRep.invoices[checkoutRes.InvoiceID].Status)
	assert.Equal(t, entity.SubStatusActive, subFx.repo.subs[checkoutRes.SubscriptionID].Status)
}

func TestVNPayIPNResponseFor(t *testing.T) {
	assert.Equal(t, "00", VNPayIPNResponseFor(nil).RspCode)
	assert.Equal(t, "97", VNPayIPNResponseFor(rescode.New(rescode.PaymentInvalidSignature, "x")).RspCode)
	assert.Equal(t, "01", VNPayIPNResponseFor(rescode.New(rescode.NotFound, "x")).RspCode)
	assert.Equal(t, "04", VNPayIPNResponseFor(rescode.New(rescode.PaymentAmountMismatch, "x")).RspCode)
	assert.Equal(t, "02", VNPayIPNResponseFor(rescode.New(rescode.PaymentTxnExpired, "x")).RspCode)
	assert.Equal(t, "99", VNPayIPNResponseFor(fmt.Errorf("db down")).RspCode)
}

func staffCtx() context.Context {
	return reqctx.WithAuth(context.Background(), &token.Payload{UserID: 9, Roles: []string{consts.RoleOwner}})
}

// Bug QA2-3/4: thu tại quầy bị chặn khi có giao dịch VNPay đang chờ; IPN thu vượt / hóa đơn đã thu đủ không bao giờ
// trả 01 – ghi nhận tối đa số còn nợ và gắn cờ NEEDS_REFUND + báo nhân viên.
func TestOnlinePayment_PendingBlocksCounter_OverpayAndAlreadyPaid(t *testing.T) {
	svc, repo, subFx, notif := newOnlinePaymentFixture()
	ctx := memberPortalCtx(50)
	co, err := svc.CheckoutSubscription(ctx, &v1.CheckoutSubscriptionReq{PlanID: 1, BranchID: 1}, "127.0.0.1")
	require.NoError(t, err)
	require.Equal(t, 500000.0, co.Amount)

	// Gọi lại checkout cùng gói → dùng lại đơn + giao dịch PENDING (không tạo gói/giao dịch trùng).
	co2, err := svc.CheckoutSubscription(ctx, &v1.CheckoutSubscriptionReq{PlanID: 1, BranchID: 1}, "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, co.SubscriptionID, co2.SubscriptionID)
	assert.Equal(t, co.TxnRef, co2.TxnRef)
	assert.Len(t, repo.txns, 1)
	pendingSubs := 0
	for _, s := range subFx.repo.subs {
		if s.Status == entity.SubStatusPending {
			pendingSubs++
		}
	}
	assert.Equal(t, 1, pendingSubs)
	// Tạo URL VNPay cho hóa đơn lần nữa → vẫn giao dịch cũ.
	again, err := svc.CreateInvoiceVNPay(ctx, co.InvoiceID, "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, co.TxnRef, again.TxnRef)

	// Thu tại quầy khi hội viên đang thanh toán VNPay → 409.
	_, err = subFx.inv.AddPayment(staffCtx(), co.InvoiceID, PaymentInput{Amount: 200000, Method: entity.PaymentCash})
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err))

	// Hết hạn 15 phút → quầy thu được 200k (hóa đơn còn nợ 300k).
	subFx.clock = subFx.clock.Add(16 * time.Minute)
	_, err = subFx.inv.AddPayment(staffCtx(), co.InvoiceID, PaymentInput{Amount: 200000, Method: entity.PaymentCash})
	require.NoError(t, err)

	// IPN thành công đến trễ của giao dịch 500k → ghi nhận 300k, 200k cần hoàn, trả 00 (không phải 01).
	res, err := svc.HandleIPN(context.Background(), buildSignedIPNParams(testVNPayCfg.HashSecret, co.TxnRef, 500000, "00"))
	require.NoError(t, err)
	assert.Equal(t, "00", res.RspCode)
	txn := repo.txns[1]
	assert.Equal(t, entity.OnlineTxnNeedsRefund, txn.Status)
	require.NotNil(t, txn.RecordedAmount)
	assert.Equal(t, 300000.0, *txn.RecordedAmount)
	assert.Contains(t, txn.Note, "cần hoàn")
	inv := subFx.invRep.invoices[co.InvoiceID]
	assert.Equal(t, entity.InvoiceStatusPaid, inv.Status)
	assert.Equal(t, 500000.0, inv.PaidAmount)
	assert.Equal(t, entity.SubStatusActive, subFx.repo.subs[co.SubscriptionID].Status)
	require.Len(t, notif.staff, 1)
	assert.Contains(t, notif.staff[0], consts.PermInvoiceRefund)

	// Gọi lại IPN → 02, không ghi thêm.
	res, err = svc.HandleIPN(context.Background(), buildSignedIPNParams(testVNPayCfg.HashSecret, co.TxnRef, 500000, "00"))
	require.NoError(t, err)
	assert.Equal(t, "02", res.RspCode)
	assert.Len(t, subFx.invRep.payments[co.InvoiceID], 2)
}

func TestOnlinePayment_IPNForVoidInvoiceFlagsRefund(t *testing.T) {
	svc, repo, subFx, notif := newOnlinePaymentFixture()
	co, err := svc.CheckoutSubscription(memberPortalCtx(50), &v1.CheckoutSubscriptionReq{PlanID: 1, BranchID: 1}, "127.0.0.1")
	require.NoError(t, err)
	// Hóa đơn bị hủy trong lúc hội viên đang ở trang VNPay.
	require.NoError(t, subFx.inv.Void(staffCtx(), co.InvoiceID, "Khách đổi ý"))
	res, err := svc.HandleIPN(context.Background(), buildSignedIPNParams(testVNPayCfg.HashSecret, co.TxnRef, co.Amount, "00"))
	require.NoError(t, err)
	assert.Equal(t, "00", res.RspCode)
	assert.Equal(t, entity.OnlineTxnNeedsRefund, repo.txns[1].Status)
	assert.Equal(t, 0.0, *repo.txns[1].RecordedAmount)
	assert.Empty(t, subFx.invRep.payments[co.InvoiceID])
	require.Len(t, notif.staff, 1)

	// Danh sách hiển thị rõ trạng thái + số cần hoàn.
	list, err := svc.List(staffCtx(), &v1.ListOnlineTxnsReq{})
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	assert.Equal(t, entity.OnlineTxnNeedsRefund, list.Items[0].Status)
	assert.Equal(t, 0.0, list.Items[0].RecordedAmount)
	assert.Equal(t, co.Amount, list.Items[0].Refundable)

	// VerifyReturn sau khi giao dịch rơi vào NEEDS_REFUND không được báo Success=true.
	retRes, err := svc.VerifyReturn(context.Background(), buildSignedIPNParams(testVNPayCfg.HashSecret, co.TxnRef, co.Amount, "00"))
	require.NoError(t, err)
	assert.False(t, retRes.Success)
}

func TestOnlinePayment_CheckoutDifferentPromoVoidsOldPending(t *testing.T) {
	svc, _, subFx, _ := newOnlinePaymentFixture()
	ctx := memberPortalCtx(50)
	a, err := svc.CheckoutSubscription(ctx, &v1.CheckoutSubscriptionReq{PlanID: 2, BranchID: 1}, "127.0.0.1")
	require.NoError(t, err)
	b, err := svc.CheckoutSubscription(ctx, &v1.CheckoutSubscriptionReq{PlanID: 2, BranchID: 1, PromotionCode: "CHAODONG"}, "127.0.0.1")
	require.NoError(t, err)
	assert.NotEqual(t, a.SubscriptionID, b.SubscriptionID)
	assert.Equal(t, entity.InvoiceStatusVoid, subFx.invRep.invoices[a.InvoiceID].Status)
	assert.Equal(t, entity.SubStatusCancelled, subFx.repo.subs[a.SubscriptionID].Status)
	assert.Equal(t, 1215000.0, b.Amount)
}

// FIN-VNP-15 (B8): giao dịch EXPIRED đã nhận IPN thất bại → IPN 00 tới sau trả 02, giữ EXPIRED, không ghi tiền.
func TestOnlinePayment_ExpiredTxnWithFailedIPNRejectsLaterSuccess(t *testing.T) {
	svc, repo, subFx, _ := newOnlinePaymentFixture()
	checkoutRes, err := svc.CheckoutSubscription(memberPortalCtx(50), &v1.CheckoutSubscriptionReq{PlanID: 1}, "127.0.0.1")
	require.NoError(t, err)
	subFx.clock = subFx.clock.Add(16 * time.Minute)
	_, err = svc.ExpirePendingTxns(context.Background(), subFx.clock)
	require.NoError(t, err)

	res, err := svc.HandleIPN(context.Background(), buildSignedIPNParams(testVNPayCfg.HashSecret, checkoutRes.TxnRef, checkoutRes.Amount, "24"))
	require.NoError(t, err)
	assert.Equal(t, "00", res.RspCode)
	assert.Equal(t, entity.OnlineTxnExpired, repo.txns[1].Status)

	res, err = svc.HandleIPN(context.Background(), buildSignedIPNParams(testVNPayCfg.HashSecret, checkoutRes.TxnRef, checkoutRes.Amount, "00"))
	require.NoError(t, err)
	assert.Equal(t, "02", res.RspCode)
	assert.Equal(t, entity.OnlineTxnExpired, repo.txns[1].Status)
	assert.Empty(t, subFx.invRep.payments[checkoutRes.InvoiceID], "không được ghi tiền sau khi giao dịch đã chốt thất bại")
	assert.Equal(t, entity.InvoiceStatusUnpaid, subFx.invRep.invoices[checkoutRes.InvoiceID].Status)
}

// FE-05: trang kết quả với TxnRef không tồn tại (dù chữ ký hợp lệ) không được báo thành công.
func TestOnlinePayment_VerifyReturnUnknownTxnNotSuccess(t *testing.T) {
	svc, _, _, _ := newOnlinePaymentFixture()
	res, err := svc.VerifyReturn(context.Background(), buildSignedIPNParams(testVNPayCfg.HashSecret, "VNP_KHONG_TON_TAI", 1, "00"))
	require.NoError(t, err)
	assert.False(t, res.Success)
	assert.Zero(t, res.InvoiceID)
	assert.Contains(t, res.Message, "Không tìm thấy giao dịch")
}

// FE-06: gói giá 0đ không đăng ký trực tuyến được.
func TestOnlinePayment_CheckoutFreePlanRejected(t *testing.T) {
	svc, _, subFx, _ := newOnlinePaymentFixture()
	subFx.repo.plans[9] = &entity.MembershipPlan{BaseEntity: entity.BaseEntity{ID: 9}, Code: "FREE", Name: "Gói 0đ", Type: entity.PlanTypeTime,
		DurationDays: 30, Price: 0, AccessScope: entity.AccessSingleBranch, Status: consts.StatusActive}
	_, err := svc.CheckoutSubscription(memberPortalCtx(50), &v1.CheckoutSubscriptionReq{PlanID: 9, BranchID: 1}, "127.0.0.1")
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
	assert.Empty(t, subFx.repo.subs, "không tạo gói")
}
