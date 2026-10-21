package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gogf/gf/v2/errors/gerror"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	v1 "gymmaster-api/api/payment/v1"
	subv1 "gymmaster-api/api/subscription/v1"
	"gymmaster-api/internal/config"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/pkg/vnpay"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/storage/postgres"
)

const (
	onlineTxnTTL        = 15 * time.Minute
	JobExpireOnlineTxns = "expire_online_txns"
)

var txnRefSeq uint64

type (
	OnlinePaymentService interface {
		CreateInvoiceVNPay(ctx context.Context, invoiceID uint, clientIP string) (*v1.PaymentURLRes, error)
		CheckoutSubscription(ctx context.Context, req *v1.CheckoutSubscriptionReq, clientIP string) (*v1.CheckoutSubscriptionRes, error)
		CheckPromotion(ctx context.Context, req *v1.CheckPromotionReq) (*v1.CheckPromotionRes, error)
		HandleIPN(ctx context.Context, params map[string]string) (*v1.VNPayIPNRes, error)
		VerifyReturn(ctx context.Context, params map[string]string) (*v1.VNPayReturnRes, error)
		List(ctx context.Context, req *v1.ListOnlineTxnsReq) (*v1.ListOnlineTxnsRes, error)
		ExpirePendingTxns(ctx context.Context, now time.Time) (int, error)
	}

	onlinePaymentService struct {
		repo          repository.OnlinePaymentRepository
		invoices      InvoiceService
		subscriptions SubscriptionService
		promotions    PromotionService
		tx            postgres.TxManager
		notifier      Notifier
		cfg           config.VNPayCfg
		now           func() time.Time
	}
)

func NewOnlinePaymentService(
	repo repository.OnlinePaymentRepository,
	invoices InvoiceService,
	subscriptions SubscriptionService,
	promotions PromotionService,
	tx postgres.TxManager,
	notifier Notifier,
	cfg config.VNPayCfg,
) OnlinePaymentService {
	return &onlinePaymentService{
		repo:          repo,
		invoices:      invoices,
		subscriptions: subscriptions,
		promotions:    promotions,
		tx:            tx,
		notifier:      notifier,
		cfg:           cfg,
		now:           time.Now,
	}
}

func (s *onlinePaymentService) currentMember(ctx context.Context) (*entity.Member, uint, error) {
	p := reqctx.Auth(ctx)
	if p == nil || p.UserID == 0 {
		return nil, 0, rescode.New(rescode.Unauthorized, "Bạn chưa đăng nhập")
	}
	m, err := s.repo.FindMemberByUserID(ctx, p.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, rescode.New(rescode.NotFound, "Tài khoản chưa được liên kết với hội viên")
		}
		return nil, 0, err
	}
	return m, p.UserID, nil
}

func (s *onlinePaymentService) newTxnRef(now time.Time) string {
	seq := atomic.AddUint64(&txnRefSeq, 1) % 10000
	return fmt.Sprintf("VNP%s%04d", now.In(subVNLoc).Format("060102150405"), seq)
}

// createTxn tạo (hoặc dùng lại) giao dịch VNPay cho số còn nợ của hóa đơn. Chạy trong transaction, khóa hóa đơn
// để hai yêu cầu đồng thời không tạo hai giao dịch PENDING:
//   - đã có giao dịch PENDING còn hạn của chính hội viên với đúng số còn nợ → trả lại URL thanh toán của giao dịch đó;
//   - PENDING khác (số tiền đã đổi) → chuyển EXPIRED (bị thay thế) rồi tạo giao dịch mới. Nếu hội viên vẫn thanh toán
//     giao dịch cũ, IPN vẫn ghi nhận tiền (hoặc gắn cờ cần hoàn) – không mất tiền.
func (s *onlinePaymentService) createTxn(ctx context.Context, inv *entity.Invoice, memberID uint, clientIP string) (*v1.PaymentURLRes, error) {
	var out *v1.PaymentURLRes
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		// Khóa hóa đơn bằng ngữ cảnh hệ thống (hội viên không có phạm vi chi nhánh quản trị).
		locked, err := s.invoices.LockInvoice(reqctx.WithAuth(ctx, nil), inv.ID)
		if err != nil {
			return err
		}
		if locked.Status != entity.InvoiceStatusUnpaid && locked.Status != entity.InvoiceStatusPartial {
			return rescode.New(rescode.BadRequest, "Hóa đơn đã được thanh toán hoặc đã hủy")
		}
		due := roundMoneyVND(locked.Due())
		if due <= 0 {
			return rescode.New(rescode.BadRequest, "Hóa đơn không còn số tiền cần thanh toán")
		}
		orderInfo := fmt.Sprintf("Thanh toan hoa don %s", locked.InvoiceNo)
		if locked.InvoiceNo == "" {
			orderInfo = fmt.Sprintf("Thanh toan hoa don #%d", locked.ID)
		}

		now := s.now()
		pending, err := s.repo.FindPendingByInvoice(ctx, locked.ID, now.Add(-onlineTxnTTL))
		if err != nil {
			return err
		}
		if pending != nil {
			if pending.MemberID == memberID && roundMoneyVND(pending.Amount) == due {
				expiresAt := pending.CreatedAt.Add(onlineTxnTTL)
				paymentURL, err := vnpay.BuildPaymentURL(s.cfg, vnpay.BuildParams{
					TxnRef: pending.TxnRef, Amount: pending.Amount, OrderInfo: orderInfo, ClientIP: clientIP,
					CreateDate: pending.CreatedAt, ExpireDate: expiresAt,
				})
				if err != nil {
					return err
				}
				out = &v1.PaymentURLRes{TxnRef: pending.TxnRef, PaymentURL: paymentURL, Amount: pending.Amount, ExpiresAt: expiresAt}
				return nil
			}
			if err := s.repo.UpdateFields(ctx, pending.ID, map[string]any{
				"status": entity.OnlineTxnExpired, "note": "Thay bằng giao dịch mới (số tiền cần thanh toán đã thay đổi)",
			}); err != nil {
				return err
			}
		}

		expiresAt := now.Add(onlineTxnTTL)
		txnRef := s.newTxnRef(now)
		paymentURL, err := vnpay.BuildPaymentURL(s.cfg, vnpay.BuildParams{
			TxnRef:     txnRef,
			Amount:     due,
			OrderInfo:  orderInfo,
			ClientIP:   clientIP,
			CreateDate: now,
			ExpireDate: expiresAt,
		})
		if err != nil {
			return err
		}
		txn := &entity.OnlinePaymentTxn{
			InvoiceID: locked.ID,
			MemberID:  memberID,
			TxnRef:    txnRef,
			Amount:    due,
			Status:    entity.OnlineTxnPending,
			CreatedAt: now,
		}
		if err := s.repo.Create(ctx, txn); err != nil {
			return err
		}
		out = &v1.PaymentURLRes{TxnRef: txnRef, PaymentURL: paymentURL, Amount: due, ExpiresAt: expiresAt}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *onlinePaymentService) CreateInvoiceVNPay(ctx context.Context, invoiceID uint, clientIP string) (*v1.PaymentURLRes, error) {
	member, _, err := s.currentMember(ctx)
	if err != nil {
		return nil, err
	}
	inv, err := s.invoices.FindInvoice(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	if inv.MemberID == nil || *inv.MemberID != member.ID {
		return nil, rescode.New(rescode.NotFound, "Không tìm thấy hóa đơn")
	}
	return s.createTxn(ctx, inv, member.ID, clientIP)
}

func (s *onlinePaymentService) CheckoutSubscription(ctx context.Context, req *v1.CheckoutSubscriptionReq, clientIP string) (*v1.CheckoutSubscriptionRes, error) {
	member, userID, err := s.currentMember(ctx)
	if err != nil {
		return nil, err
	}
	branchID := req.BranchID
	if branchID == 0 {
		branchID = member.HomeBranchID
	}

	// Nâng quyền tạm thời trên context nội bộ để SubscriptionService.Sell lập gói PENDING + hóa đơn UNPAID cho chính hội viên này.
	sellCtx := reqctx.WithAuth(ctx, &token.Payload{
		UserID:      userID,
		Roles:       []string{consts.RoleMember},
		Permissions: []string{consts.PermSubscriptionSell, consts.PermInvoiceCollect},
		BranchIDs:   []uint{branchID},
	})

	var out *v1.CheckoutSubscriptionRes
	err = s.tx.WithTx(sellCtx, func(txCtx context.Context) error {
		// Đã có đơn đăng ký trực tuyến chờ thanh toán cho đúng gói/chi nhánh → dùng lại (không tạo gói PENDING trùng).
		// Khác mã khuyến mãi → hủy đơn cũ (trả lượt KM, gói cũ chuyển CANCELLED) rồi lập đơn mới.
		if existing, err := s.repo.FindPortalCheckout(txCtx, member.ID, req.PlanID, branchID); err != nil {
			return err
		} else if existing != nil {
			if strings.EqualFold(strings.TrimSpace(existing.PromotionCode), strings.TrimSpace(req.PromotionCode)) {
				invEntity, err := s.invoices.FindInvoice(txCtx, existing.InvoiceID)
				if err != nil {
					return err
				}
				payRes, err := s.createTxn(txCtx, invEntity, member.ID, clientIP)
				if err != nil {
					return err
				}
				out = &v1.CheckoutSubscriptionRes{
					SubscriptionID: existing.SubscriptionID, InvoiceID: existing.InvoiceID, InvoiceNo: existing.InvoiceNo,
					Amount: payRes.Amount, TxnRef: payRes.TxnRef, PaymentURL: payRes.PaymentURL,
				}
				return nil
			}
			if err := s.invoices.Void(reqctx.WithAuth(txCtx, nil), existing.InvoiceID, "Hội viên đăng ký lại gói trực tuyến với mã khuyến mãi khác"); err != nil {
				return err
			}
		}
		subDetail, err := s.subscriptions.Sell(txCtx, subv1.SellInput{
			MemberID:      member.ID,
			PlanID:        req.PlanID,
			BranchID:      branchID,
			PromotionCode: req.PromotionCode,
			Note:          "Đăng ký gói tập trực tuyến qua cổng hội viên",
		}, nil)
		if err != nil {
			return err
		}
		if subDetail.Invoice == nil {
			return rescode.New(rescode.Internal, "Không thể khởi tạo hóa đơn cho gói tập")
		}
		_ = s.repo.SetInvoiceSource(txCtx, subDetail.Invoice.ID, entity.InvoiceSourcePortal)

		res := &v1.CheckoutSubscriptionRes{
			SubscriptionID: subDetail.ID,
			InvoiceID:      subDetail.Invoice.ID,
			InvoiceNo:      subDetail.Invoice.InvoiceNo,
			Amount:         subDetail.Invoice.Total,
		}
		if subDetail.Invoice.Total > 0 && subDetail.Invoice.Status != entity.InvoiceStatusPaid {
			invEntity, err := s.invoices.FindInvoice(txCtx, subDetail.Invoice.ID)
			if err != nil {
				return err
			}
			payRes, err := s.createTxn(txCtx, invEntity, member.ID, clientIP)
			if err != nil {
				return err
			}
			res.TxnRef = payRes.TxnRef
			res.PaymentURL = payRes.PaymentURL
			res.Amount = payRes.Amount
		}
		out = res
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *onlinePaymentService) CheckPromotion(ctx context.Context, req *v1.CheckPromotionReq) (*v1.CheckPromotionRes, error) {
	if _, _, err := s.currentMember(ctx); err != nil {
		return nil, err
	}
	amount := req.Amount
	if amount <= 0 && req.PlanID > 0 {
		plan, err := s.repo.FindPlanByID(ctx, req.PlanID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, rescode.New(rescode.NotFound, "Không tìm thấy gói tập")
			}
			return nil, err
		}
		amount = plan.Price
	}
	promo, discount, err := s.promotions.Validate(ctx, req.Code, entity.AppliesMembership, amount, s.now())
	if err != nil {
		return nil, err
	}
	return &v1.CheckPromotionRes{
		Code:        promo.Code,
		Name:        promo.Name,
		Discount:    discount,
		FinalAmount: roundMoneyVND(amount) - discount,
	}, nil
}

// Mã phản hồi IPN theo đặc tả VNPay.
const (
	VNPayRspSuccess          = "00"
	VNPayRspOrderNotFound    = "01"
	VNPayRspAlreadyConfirmed = "02"
	VNPayRspInvalidAmount    = "04"
	VNPayRspInvalidChecksum  = "97"
	VNPayRspUnknown          = "99"
)

// VNPayIPNResponseFor đổi lỗi nghiệp vụ của HandleIPN sang phản hồi chuẩn VNPay (RspCode/Message).
func VNPayIPNResponseFor(err error) *v1.VNPayIPNRes {
	if err == nil {
		return &v1.VNPayIPNRes{RspCode: VNPayRspSuccess, Message: "Confirm Success"}
	}
	switch gerror.Code(err).Code() {
	case rescode.PaymentInvalidSignature.Code():
		return &v1.VNPayIPNRes{RspCode: VNPayRspInvalidChecksum, Message: "Invalid Checksum"}
	case rescode.NotFound.Code():
		return &v1.VNPayIPNRes{RspCode: VNPayRspOrderNotFound, Message: "Order not found"}
	case rescode.PaymentAmountMismatch.Code():
		return &v1.VNPayIPNRes{RspCode: VNPayRspInvalidAmount, Message: "Invalid amount"}
	case rescode.PaymentTxnExpired.Code():
		return &v1.VNPayIPNRes{RspCode: VNPayRspAlreadyConfirmed, Message: "Order already confirmed (transaction expired)"}
	}
	return &v1.VNPayIPNRes{RspCode: VNPayRspUnknown, Message: "Unknown error"}
}

func (s *onlinePaymentService) HandleIPN(ctx context.Context, params map[string]string) (*v1.VNPayIPNRes, error) {
	if !vnpay.VerifyQuery(s.cfg.HashSecret, params) {
		return nil, rescode.New(rescode.PaymentInvalidSignature, "Chữ ký VNPay không hợp lệ")
	}
	txnRef := strings.TrimSpace(params["vnp_TxnRef"])
	if txnRef == "" {
		return nil, rescode.New(rescode.NotFound, "Thiếu mã giao dịch vnp_TxnRef")
	}

	var res *v1.VNPayIPNRes
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		txn, err := s.repo.FindByTxnRef(ctx, txnRef, true)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return rescode.New(rescode.NotFound, "Không tìm thấy giao dịch thanh toán")
			}
			return err
		}

		vnpAmount, err := strconv.ParseInt(strings.TrimSpace(params["vnp_Amount"]), 10, 64)
		if err != nil || math.Abs(float64(vnpAmount)/100-txn.Amount) > 0.01 {
			return rescode.New(rescode.PaymentAmountMismatch, "Số tiền thanh toán không khớp với giao dịch")
		}

		switch txn.Status {
		case entity.OnlineTxnSuccess, entity.OnlineTxnFailed, entity.OnlineTxnNeedsRefund, entity.OnlineTxnRefunded:
			// VNPay gọi lại IPN cho giao dịch đã xử lý → 02 (Order already confirmed), không ghi trùng.
			res = &v1.VNPayIPNRes{RspCode: VNPayRspAlreadyConfirmed, Message: "Order already confirmed"}
			return nil
		}

		now := s.now()
		rawIPN, _ := json.Marshal(params)
		rspCode := strings.TrimSpace(params["vnp_ResponseCode"])
		providerTxn := strings.TrimSpace(params["vnp_TransactionNo"])
		bankCode := strings.TrimSpace(params["vnp_BankCode"])

		if rspCode != "00" {
			// Thanh toán không thành công: giao dịch PENDING → FAILED; giao dịch đã hết hạn/bị thay thế giữ EXPIRED.
			fields := map[string]any{"provider_txn_no": providerTxn, "bank_code": bankCode, "raw_ipn": datatypes.JSON(rawIPN)}
			if txn.Status == entity.OnlineTxnPending {
				fields["status"] = entity.OnlineTxnFailed
			}
			if err := s.repo.UpdateFields(ctx, txn.ID, fields); err != nil {
				return err
			}
			res = &v1.VNPayIPNRes{RspCode: VNPayRspSuccess, Message: "Confirm Failed Transaction"}
			return nil
		}

		// VNPay xác nhận ĐÃ trừ tiền khách. Kể cả giao dịch đã quá hạn/bị thay thế (IPN đến trễ) cũng phải ghi nhận
		// tiền – không bao giờ trả lỗi làm mất dấu khoản tiền: ghi vào hóa đơn tối đa số còn nợ, phần còn lại
		// (hoặc toàn bộ nếu hóa đơn đã thu đủ/đã hủy/gói đã hủy) gắn cờ NEEDS_REFUND để nhân viên hoàn.
		// IPN là tác vụ hệ thống: KHÔNG gắn người dùng (payload nil = toàn quyền chi nhánh, người thu/actor = NULL).
		// Không dùng UserID 0 vì sẽ vi phạm FK payments.received_by / audit_logs.actor_id.
		sysCtx := reqctx.WithAuth(ctx, nil)
		cur, err := s.invoices.LockInvoice(sysCtx, txn.InvoiceID)
		if err != nil {
			return err
		}
		var due float64
		if cur.Status == entity.InvoiceStatusUnpaid || cur.Status == entity.InvoiceStatusPartial {
			due = max(0, roundMoneyVND(cur.Due()))
		}
		record := math.Min(roundMoneyVND(txn.Amount), due)
		refundReason := ""
		var inv *entity.Invoice
		if record > 0 {
			ref := providerTxn
			if ref == "" {
				ref = txn.TxnRef
			}
			inv, err = s.invoices.AddPayment(sysCtx, txn.InvoiceID, PaymentInput{
				Amount:    record,
				Method:    entity.PaymentVNPay,
				Reference: ref,
				Note:      "Thanh toán trực tuyến VNPay (" + txn.TxnRef + ")",
			})
			if err != nil {
				if !onlineIsBusinessErr(err) {
					return err // lỗi hệ thống → 99, VNPay sẽ gọi lại IPN
				}
				// Bị từ chối nghiệp vụ (vd gói đã hủy) → không ghi vào hóa đơn, cần hoàn toàn bộ.
				refundReason = "Không ghi nhận được vào hóa đơn: " + err.Error()
				record, inv = 0, nil
			}
		}
		status := entity.OnlineTxnSuccess
		note := ""
		excess := roundMoneyVND(txn.Amount - record)
		if excess > 0 {
			status = entity.OnlineTxnNeedsRefund
			switch {
			case refundReason != "":
				note = refundReason
			case record > 0:
				note = fmt.Sprintf("Khách trả vượt số còn nợ: đã ghi nhận %s vào hóa đơn, cần hoàn %s", formatMoneyVND(record), formatMoneyVND(excess))
			case cur.Status == entity.InvoiceStatusVoid:
				note = "Hóa đơn đã bị hủy trước khi nhận tiền VNPay – cần hoàn toàn bộ cho khách"
			default:
				note = "Hóa đơn đã được thanh toán đủ trước khi nhận tiền VNPay – cần hoàn toàn bộ cho khách"
			}
		}
		if err := s.repo.UpdateFields(ctx, txn.ID, map[string]any{
			"status":          status,
			"recorded_amount": record,
			"note":            invoiceTruncate(note, 255),
			"provider_txn_no": providerTxn,
			"bank_code":       bankCode,
			"raw_ipn":         datatypes.JSON(rawIPN),
			"paid_at":         now,
		}); err != nil {
			return err
		}

		if s.notifier != nil {
			invNo := cur.InvoiceNo
			if inv != nil && inv.InvoiceNo != "" {
				invNo = inv.InvoiceNo
			}
			link := fmt.Sprintf("/me/invoices?invoice=%d", txn.InvoiceID)
			if m, mErr := s.repo.FindMemberByID(ctx, txn.MemberID); mErr == nil && m != nil && m.UserID != nil {
				if record > 0 {
					_ = s.notifier.Notify(ctx, []uint{*m.UserID}, NotifyPaymentSuccess, "Thanh toán thành công",
						fmt.Sprintf("Hóa đơn %s đã ghi nhận %s thanh toán qua VNPay.", invNo, formatMoneyVND(record)), link)
				}
				if excess > 0 {
					_ = s.notifier.Notify(ctx, []uint{*m.UserID}, NotifyPaymentSuccess, "Đã nhận tiền VNPay – sẽ hoàn phần chưa ghi nhận",
						fmt.Sprintf("Giao dịch %s: %s chưa được ghi vào hóa đơn %s. Phòng tập sẽ liên hệ hoàn tiền cho bạn.", txn.TxnRef, formatMoneyVND(excess), invNo), link)
				}
			}
			if excess > 0 {
				branchID := cur.BranchID
				_ = s.notifier.NotifyPermission(ctx, consts.PermInvoiceRefund, &branchID, NotifySystem,
					"Giao dịch VNPay cần hoàn tiền",
					fmt.Sprintf("Giao dịch %s (hóa đơn %s): %s.", txn.TxnRef, invNo, note),
					"/app/payments?status="+entity.OnlineTxnNeedsRefund)
			}
		}

		res = &v1.VNPayIPNRes{RspCode: VNPayRspSuccess, Message: "Confirm Success"}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

// onlineIsBusinessErr: lỗi nghiệp vụ có mã 4xxxx (từ chối hợp lệ) – khác lỗi hệ thống/DB.
func onlineIsBusinessErr(err error) bool {
	c := gerror.Code(err).Code()
	return c >= 40000 && c < 50000
}

func (s *onlinePaymentService) VerifyReturn(ctx context.Context, params map[string]string) (*v1.VNPayReturnRes, error) {
	if !vnpay.VerifyQuery(s.cfg.HashSecret, params) {
		return nil, rescode.New(rescode.PaymentInvalidSignature, "Chữ ký VNPay không hợp lệ")
	}
	txnRef := strings.TrimSpace(params["vnp_TxnRef"])
	rspCode := strings.TrimSpace(params["vnp_ResponseCode"])
	var amount float64
	if vnpAmount, err := strconv.ParseInt(strings.TrimSpace(params["vnp_Amount"]), 10, 64); err == nil {
		amount = float64(vnpAmount) / 100
	}

	out := &v1.VNPayReturnRes{
		Success:      rspCode == "00",
		TxnRef:       txnRef,
		Amount:       amount,
		ResponseCode: rspCode,
		Message:      "Thanh toán thành công",
	}
	if !out.Success {
		out.Message = "Giao dịch thanh toán không thành công"
	}

	if txn, err := s.repo.FindByTxnRef(ctx, txnRef, false); err == nil && txn != nil {
		out.InvoiceID = txn.InvoiceID
		out.Amount = txn.Amount
		if inv, invErr := s.invoices.FindInvoice(ctx, txn.InvoiceID); invErr == nil && inv != nil {
			out.InvoiceNo = inv.InvoiceNo
		}
		switch txn.Status {
		case entity.OnlineTxnNeedsRefund:
			out.Success = false
			out.Message = "Hóa đơn đã được xử lý trước đó, phòng tập sẽ liên hệ đối soát hoàn tiền"
		case entity.OnlineTxnFailed, entity.OnlineTxnExpired:
			out.Success = false
			out.Message = "Giao dịch thanh toán không thành công hoặc đã hết hạn"
		}
	}
	return out, nil
}

var onlineTxnOrderable = []string{"id", "created_at", "paid_at", "amount", "status"}

func (s *onlinePaymentService) List(ctx context.Context, req *v1.ListOnlineTxnsReq) (*v1.ListOnlineTxnsRes, error) {
	q := req.PageReq.ToQuery(onlineTxnOrderable, "id")
	q.OrderBy = "t." + q.OrderBy
	f := repository.OnlineTxnListFilter{
		BranchIDs: reqctx.AllowedBranches(ctx),
		Status:    req.Status,
		Search:    req.Search,
	}
	rows, total, err := s.repo.List(ctx, q, f)
	if err != nil {
		return nil, err
	}
	items := dto.Map(rows, func(r repository.OnlineTxnRow) *v1.OnlineTxnItem {
		return &v1.OnlineTxnItem{
			ID:            r.ID,
			InvoiceID:     r.InvoiceID,
			InvoiceNo:     r.InvoiceNo,
			MemberID:      r.MemberID,
			MemberCode:    r.MemberCode,
			MemberName:    r.MemberName,
			TxnRef:        r.TxnRef,
			Amount:        r.Amount,
			Status:        r.Status,
			ProviderTxnNo: r.ProviderTxnNo,
			BankCode:      r.BankCode,
			CreatedAt:     r.CreatedAt,
			PaidAt:        r.PaidAt,
			RecordedAmount: func() float64 {
				if r.RecordedAmount == nil {
					if r.Status == entity.OnlineTxnSuccess || r.Status == entity.OnlineTxnRefunded {
						return r.Amount
					}
					return 0
				}
				return *r.RecordedAmount
			}(),
			RefundedAmount: r.RefundedAmount,
			Refundable:     roundMoneyVND(r.Amount - r.RefundedAmount),
			Note:           r.Note,
		}
	})
	return dto.NewPageRes(items, total, q), nil
}

func (s *onlinePaymentService) ExpirePendingTxns(ctx context.Context, now time.Time) (int, error) {
	affected, err := s.repo.ExpirePendingBefore(ctx, now.Add(-onlineTxnTTL))
	return int(affected), err
}
