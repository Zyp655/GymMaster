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

func (s *onlinePaymentService) createTxn(ctx context.Context, inv *entity.Invoice, memberID uint, clientIP string) (*v1.PaymentURLRes, error) {
	if inv.Status != entity.InvoiceStatusUnpaid && inv.Status != entity.InvoiceStatusPartial {
		return nil, rescode.New(rescode.BadRequest, "Hóa đơn đã được thanh toán hoặc đã hủy")
	}
	due := roundMoneyVND(inv.Due())
	if due <= 0 {
		return nil, rescode.New(rescode.BadRequest, "Hóa đơn không còn số tiền cần thanh toán")
	}

	now := s.now()
	expiresAt := now.Add(onlineTxnTTL)
	txnRef := s.newTxnRef(now)
	orderInfo := fmt.Sprintf("Thanh toan hoa don %s", inv.InvoiceNo)
	if inv.InvoiceNo == "" {
		orderInfo = fmt.Sprintf("Thanh toan hoa don #%d", inv.ID)
	}

	paymentURL, err := vnpay.BuildPaymentURL(s.cfg, vnpay.BuildParams{
		TxnRef:     txnRef,
		Amount:     due,
		OrderInfo:  orderInfo,
		ClientIP:   clientIP,
		CreateDate: now,
		ExpireDate: expiresAt,
	})
	if err != nil {
		return nil, err
	}

	txn := &entity.OnlinePaymentTxn{
		InvoiceID: inv.ID,
		MemberID:  memberID,
		TxnRef:    txnRef,
		Amount:    due,
		Status:    entity.OnlineTxnPending,
		CreatedAt: now,
	}
	if err := s.repo.Create(ctx, txn); err != nil {
		return nil, err
	}
	return &v1.PaymentURLRes{
		TxnRef:     txnRef,
		PaymentURL: paymentURL,
		Amount:     due,
		ExpiresAt:  expiresAt,
	}, nil
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

func (s *onlinePaymentService) HandleIPN(ctx context.Context, params map[string]string) (*v1.VNPayIPNRes, error) {
	if !vnpay.VerifyQuery(s.cfg.HashSecret, params) {
		return nil, rescode.New(rescode.PaymentInvalidSignature, "Chữ ký VNPay không hợp lệ")
	}
	txnRef := strings.TrimSpace(params["vnp_TxnRef"])
	if txnRef == "" {
		return nil, rescode.New(rescode.BadRequest, "Thiếu mã giao dịch vnp_TxnRef")
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

		if txn.Status == entity.OnlineTxnSuccess {
			res = &v1.VNPayIPNRes{RspCode: "00", Message: "Confirm Success"}
			return nil
		}

		now := s.now()
		if txn.Status == entity.OnlineTxnExpired || (txn.Status == entity.OnlineTxnPending && !txn.CreatedAt.IsZero() && now.Sub(txn.CreatedAt) > onlineTxnTTL) {
			if txn.Status == entity.OnlineTxnPending {
				_ = s.repo.UpdateFields(ctx, txn.ID, map[string]any{"status": entity.OnlineTxnExpired})
			}
			return rescode.New(rescode.PaymentTxnExpired, "Giao dịch thanh toán trực tuyến đã hết hạn")
		}

		rawIPN, _ := json.Marshal(params)
		rspCode := strings.TrimSpace(params["vnp_ResponseCode"])
		providerTxn := strings.TrimSpace(params["vnp_TransactionNo"])
		bankCode := strings.TrimSpace(params["vnp_BankCode"])

		if rspCode != "00" {
			if err := s.repo.UpdateFields(ctx, txn.ID, map[string]any{
				"status":          entity.OnlineTxnFailed,
				"provider_txn_no": providerTxn,
				"bank_code":       bankCode,
				"raw_ipn":         datatypes.JSON(rawIPN),
			}); err != nil {
				return err
			}
			res = &v1.VNPayIPNRes{RspCode: "00", Message: "Confirm Failed Transaction"}
			return nil
		}

		if err := s.repo.UpdateFields(ctx, txn.ID, map[string]any{
			"status":          entity.OnlineTxnSuccess,
			"provider_txn_no": providerTxn,
			"bank_code":       bankCode,
			"raw_ipn":         datatypes.JSON(rawIPN),
			"paid_at":         now,
		}); err != nil {
			return err
		}

		ref := providerTxn
		if ref == "" {
			ref = txn.TxnRef
		}
		payCtx := reqctx.WithAuth(ctx, &token.Payload{Roles: []string{consts.RoleOwner}})
		inv, err := s.invoices.AddPayment(payCtx, txn.InvoiceID, PaymentInput{
			Amount:    txn.Amount,
			Method:    entity.PaymentVNPay,
			Reference: ref,
			Note:      "Thanh toán trực tuyến VNPay (" + txn.TxnRef + ")",
		})
		if err != nil {
			return err
		}

		if s.notifier != nil {
			if m, mErr := s.repo.FindMemberByID(ctx, txn.MemberID); mErr == nil && m != nil && m.UserID != nil {
				_ = s.notifier.Notify(
					ctx,
					[]uint{*m.UserID},
					NotifyPaymentSuccess,
					"Thanh toán thành công",
					fmt.Sprintf("Hóa đơn %s (%s) đã được thanh toán qua VNPay.", inv.InvoiceNo, formatMoneyVND(txn.Amount)),
					fmt.Sprintf("/me/invoices/%d", inv.ID),
				)
			}
		}

		res = &v1.VNPayIPNRes{RspCode: "00", Message: "Confirm Success"}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
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
		}
	})
	return dto.NewPageRes(items, total, q), nil
}

func (s *onlinePaymentService) ExpirePendingTxns(ctx context.Context, now time.Time) (int, error) {
	affected, err := s.repo.ExpirePendingBefore(ctx, now.Add(-onlineTxnTTL))
	return int(affected), err
}
