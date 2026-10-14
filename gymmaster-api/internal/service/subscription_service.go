package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	invoicev1 "gymmaster-api/api/invoice/v1"
	v1 "gymmaster-api/api/subscription/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/storage/postgres"
)

// Audit của module gói tập.
const (
	auditEntitySubscription = "subscription"
	auditEntityFreeze       = "subscription_freeze"
	AuditSell               = "SELL"
	AuditRenew              = "RENEW"
	AuditUpgrade            = "UPGRADE"
	AuditTransfer           = "TRANSFER"
	AuditCancel             = "CANCEL"
	AuditFreezeRequest      = "FREEZE_REQUEST"
	AuditFreezeApprove      = "FREEZE_APPROVE"
	AuditFreezeReject       = "FREEZE_REJECT"
	AuditFreezeEnd          = "FREEZE_END"
)

// subPendingTTLDays: gói PENDING chưa thu đồng nào quá số ngày này thì cron hủy (và hủy hóa đơn).
const subPendingTTLDays = 3

type (
	// SubscriptionService: bán gói, gia hạn, nâng cấp, bảo lưu, chuyển nhượng, hủy và các tác vụ định kỳ.
	SubscriptionService interface {
		Sell(ctx context.Context, in v1.SellInput, pay *v1.PaymentInput) (*v1.SubscriptionDetail, error)
		Quote(ctx context.Context, in v1.SellInput) (*v1.QuoteRes, error)
		Renew(ctx context.Context, id uint, in v1.RenewInput) (*v1.SubscriptionDetail, error)
		RenewQuote(ctx context.Context, id uint, in v1.RenewInput) (*v1.QuoteRes, error)
		Upgrade(ctx context.Context, id uint, in v1.UpgradeInput) (*v1.SubscriptionDetail, error)
		UpgradeQuote(ctx context.Context, id uint, planID uint) (*v1.UpgradeQuoteRes, error)
		Transfer(ctx context.Context, id uint, in v1.TransferInput) (*v1.SubscriptionDetail, error)
		Cancel(ctx context.Context, id uint, in v1.CancelInput) (*v1.SubscriptionDetail, error)

		RequestFreeze(ctx context.Context, id uint, in v1.FreezeInput) (*v1.Freeze, error)
		ApproveFreeze(ctx context.Context, freezeID uint, note string) (*v1.Freeze, error)
		RejectFreeze(ctx context.Context, freezeID uint, note string) (*v1.Freeze, error)
		// EndFreezeEarly: kết thúc sớm bảo lưu đang áp dụng (hoàn lại ngày chưa dùng) hoặc hủy yêu cầu chưa áp dụng.
		EndFreezeEarly(ctx context.Context, freezeID uint, note string) (*v1.Freeze, error)
		ListFreezes(ctx context.Context, req *v1.FreezeListReq) (*v1.FreezeListRes, error)

		List(ctx context.Context, req *v1.ListReq) (*v1.ListRes, error)
		Get(ctx context.Context, id uint) (*v1.SubscriptionDetail, error)
		Events(ctx context.Context, id uint) ([]*v1.Event, error)
		ListByMember(ctx context.Context, memberID uint) ([]*v1.Subscription, error)

		// Tác vụ định kỳ (cron gọi với today theo giờ Việt Nam; ctx không cần người dùng).
		// ExpireDue: ACTIVE/FROZEN có end_date < today → EXPIRED; PENDING quá 3 ngày chưa thu đồng nào
		// → CANCELLED + hủy hóa đơn. Trả tổng số gói đã đổi trạng thái.
		ExpireDue(ctx context.Context, today time.Time) (int, error)
		// ApplyDueFreezes: kết thúc bảo lưu đã hết hạn (FROZEN → ACTIVE) rồi áp các bảo lưu APPROVED đến ngày
		// (ACTIVE → FROZEN). Trả số bảo lưu đã xử lý.
		ApplyDueFreezes(ctx context.Context, today time.Time) (int, error)
		// ListExpiring: gói ACTIVE hết hạn đúng sau N ngày (N thuộc days) và chưa có gói nối tiếp – để nhắc gia hạn.
		ListExpiring(ctx context.Context, days []int, today time.Time) ([]repository.ExpiringSubscription, error)

		// Cổng hội viên (memberID lấy từ tài khoản đăng nhập).
		ListMemberSubscriptions(ctx context.Context, memberID uint) ([]*v1.Subscription, error)
		ListMemberInvoices(ctx context.Context, memberID uint, page dto.PageReq) (*invoicev1.ListRes, error)
		RequestFreezeByMember(ctx context.Context, memberID, subscriptionID uint, in v1.FreezeInput) (*v1.Freeze, error)
	}

	subscriptionService struct {
		repo     repository.SubscriptionRepository
		invoices InvoiceService
		tx       postgres.TxManager
		audit    AuditService
		notifier Notifier
		now      func() time.Time
	}
)

// NewSubscriptionService tạo service và đăng ký PaidHandler/VoidHandler cho dòng MEMBERSHIP và FEE
// vào InvoiceService dùng chung. audit/notifier có thể nil (test).
func NewSubscriptionService(repo repository.SubscriptionRepository, invoices InvoiceService, tx postgres.TxManager,
	audit AuditService, notifier Notifier) SubscriptionService {
	s := &subscriptionService{repo: repo, invoices: invoices, tx: tx, audit: audit, notifier: notifier, now: time.Now}
	for _, t := range []string{entity.InvoiceItemMembership, entity.InvoiceItemFee} {
		invoices.RegisterPaidHandler(t, s.onInvoicePaid)
		invoices.RegisterVoidHandler(t, s.onInvoiceVoid)
	}
	return s
}

// ---- Ngày (giờ Việt Nam) --------------------------------------------------------

var subVNLoc = func() *time.Location {
	loc, err := time.LoadLocation(consts.TimeZone)
	if err != nil {
		return time.FixedZone("ICT", 7*3600)
	}
	return loc
}()

// subDateOf trả ngày (theo lịch Việt Nam) của thời điểm t, biểu diễn 00:00 UTC – dạng dùng cho cột DATE.
func subDateOf(t time.Time) time.Time {
	t = t.In(subVNLoc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// subNorm chuẩn hóa một giá trị DATE về 00:00 UTC (giữ nguyên năm/tháng/ngày).
func subNorm(d time.Time) time.Time {
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
}

func subAddDays(d time.Time, n int) time.Time { return subNorm(d).AddDate(0, 0, n) }

// subDaysBetween = b − a (số ngày).
func subDaysBetween(a, b time.Time) int {
	return int(math.Round(subNorm(b).Sub(subNorm(a)).Hours() / 24))
}

// subEndDate: ngày kết thúc = start + duration − 1.
func subEndDate(start time.Time, durationDays int) time.Time {
	return subAddDays(start, durationDays-1)
}

func subFmt(d time.Time) string { return subNorm(d).Format(consts.DateLayout) }

func subParseDate(s, label string) (time.Time, error) {
	d, err := time.Parse(consts.DateLayout, strings.TrimSpace(s))
	if err != nil {
		return time.Time{}, rescode.New(rescode.BadRequest, label+" phải có dạng YYYY-MM-DD")
	}
	return d, nil
}

// subInstantRange đổi khoảng ngày from/to (YYYY-MM-DD, tính cả hai đầu, giờ VN) thành [from 00:00, to+1 00:00).
func subInstantRange(from, to string) (*time.Time, *time.Time, error) {
	var f, t *time.Time
	if strings.TrimSpace(from) != "" {
		d, err := subParseDate(from, "Từ ngày")
		if err != nil {
			return nil, nil, err
		}
		v := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, subVNLoc)
		f = &v
	}
	if strings.TrimSpace(to) != "" {
		d, err := subParseDate(to, "Đến ngày")
		if err != nil {
			return nil, nil, err
		}
		v := time.Date(d.Year(), d.Month(), d.Day()+1, 0, 0, 0, 0, subVNLoc)
		t = &v
	}
	if f != nil && t != nil && !f.Before(*t) {
		return nil, nil, rescode.New(rescode.BadRequest, "Từ ngày phải trước hoặc bằng đến ngày")
	}
	return f, t, nil
}

// subChainStart: ngày bắt đầu khi nối tiếp gói latest (nil = không nối) – end_date gói cũ + 1, không sớm hơn hôm nay.
func subChainStart(today time.Time, latest *entity.Subscription) time.Time {
	if latest == nil {
		return subNorm(today)
	}
	next := subAddDays(latest.EndDate, 1)
	if next.Before(subNorm(today)) {
		return subNorm(today)
	}
	return next
}

// subRemaining tính giá trị còn lại của gói tại ngày today (dùng cho nâng cấp):
//   - TIME   : price × (số ngày còn lại / tổng số ngày), ngày còn lại tính cả hôm nay;
//   - SESSION: price × (buổi còn lại / tổng buổi).
func subRemaining(sub *entity.Subscription, planType string, today time.Time) (value float64, totalDays, remainingDays int) {
	totalDays = subDaysBetween(sub.StartDate, sub.EndDate) + 1
	switch {
	case subNorm(today).Before(subNorm(sub.StartDate)):
		remainingDays = totalDays
	default:
		remainingDays = subDaysBetween(today, sub.EndDate) + 1
	}
	remainingDays = max(0, min(remainingDays, totalDays))
	if planType == entity.PlanTypeSession && sub.TotalSessions != nil && *sub.TotalSessions > 0 && sub.RemainingSessions != nil {
		value = sub.Price * float64(*sub.RemainingSessions) / float64(*sub.TotalSessions)
	} else if totalDays > 0 {
		value = sub.Price * float64(remainingDays) / float64(totalDays)
	}
	return roundMoneyVND(value), totalDays, remainingDays
}

func (s *subscriptionService) today() time.Time { return subDateOf(s.now()) }

// ---- Lỗi / tra cứu --------------------------------------------------------------

var (
	errSubNotFound    = rescode.New(rescode.NotFound, "Không tìm thấy gói tập của hội viên")
	errSubOutOfScope  = rescode.New(rescode.PermissionDenied, "Gói tập thuộc chi nhánh ngoài phạm vi của bạn")
	errFreezeNotFound = rescode.New(rescode.NotFound, "Không tìm thấy yêu cầu bảo lưu")
)

func subMapNotFound(err error, nf error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nf
	}
	return err
}

// loadSub đọc gói và kiểm tra phạm vi chi nhánh.
func (s *subscriptionService) loadSub(ctx context.Context, id uint, lock bool) (*entity.Subscription, error) {
	sub, err := s.repo.FindByID(ctx, id, lock)
	if err != nil {
		return nil, subMapNotFound(err, errSubNotFound)
	}
	if !reqctx.CanAccessBranch(ctx, sub.BranchID) {
		return nil, errSubOutOfScope
	}
	return sub, nil
}

func (s *subscriptionService) activeMember(ctx context.Context, id uint, label string) (*entity.Member, error) {
	m, err := s.repo.FindMember(ctx, id)
	if err != nil {
		return nil, subMapNotFound(err, rescode.New(rescode.NotFound, "Không tìm thấy "+label))
	}
	if m.Status != entity.MemberStatusActive {
		return nil, rescode.New(rescode.BadRequest, strings.ToUpper(label[:1])+label[1:]+" đang không hoạt động")
	}
	return m, nil
}

func (s *subscriptionService) sellablePlan(ctx context.Context, id uint) (*entity.MembershipPlan, error) {
	p, err := s.repo.FindPlan(ctx, id)
	if err != nil {
		return nil, subMapNotFound(err, rescode.New(rescode.NotFound, "Không tìm thấy gói tập"))
	}
	if p.DeletedAt.Valid {
		return nil, rescode.New(rescode.NotFound, "Không tìm thấy gói tập")
	}
	if p.Status != consts.StatusActive {
		return nil, rescode.New(rescode.BadRequest, "Gói tập "+p.Name+" đã ngừng bán")
	}
	return p, nil
}

func (s *subscriptionService) event(ctx context.Context, subID uint, typ string, data map[string]any) error {
	raw := []byte("{}")
	if data != nil {
		if b, err := json.Marshal(data); err == nil {
			raw = b
		}
	}
	return s.repo.CreateEvent(ctx, &entity.SubscriptionEvent{
		SubscriptionID: subID, EventType: typ, Data: datatypes.JSON(raw), ActorID: reqctx.UserID(ctx),
	})
}

func (s *subscriptionService) logAudit(ctx context.Context, action, entityName string, id uint, before, after any) {
	auditCatalog(ctx, s.audit, action, entityName, id, before, after)
}

// subPayment chuẩn hóa thông tin thu tiền kèm theo (nil = không thu).
func subPayment(p *v1.PaymentInput) (*PaymentInput, error) {
	if p == nil || (p.Amount == 0 && p.Method == "") {
		return nil, nil
	}
	if roundMoneyVND(p.Amount) <= 0 {
		return nil, rescode.New(rescode.BadRequest, "Số tiền thu phải lớn hơn 0")
	}
	if err := invoiceValidMethod(p.Method); err != nil {
		return nil, err
	}
	return &PaymentInput{Amount: p.Amount, Method: p.Method, Reference: p.Reference, Note: p.Note}, nil
}

// ---- Bán gói / gia hạn ----------------------------------------------------------

type subSale struct {
	member     *entity.Member
	plan       *entity.MembershipPlan
	branchID   uint
	start, end time.Time
	previousID *uint
	chained    bool
}

// prepareSale kiểm tra hội viên/gói/chi nhánh và tính ngày bắt đầu – kết thúc.
// Không truyền startStr → nối tiếp gói ACTIVE/FROZEN cùng phạm vi có end_date muộn nhất (nếu có).
// minStart (gia hạn) bảo đảm gói mới không bắt đầu trước ngày này.
func (s *subscriptionService) prepareSale(ctx context.Context, memberID, planID, branchID uint, startStr string, minStart *time.Time) (*subSale, error) {
	if !reqctx.CanAccessBranch(ctx, branchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền bán gói tại chi nhánh này")
	}
	branch, err := s.repo.FindBranch(ctx, branchID)
	if err != nil {
		return nil, subMapNotFound(err, rescode.New(rescode.BadRequest, "Chi nhánh không tồn tại"))
	}
	if branch.Status != consts.StatusActive {
		return nil, rescode.New(rescode.BadRequest, "Chi nhánh đang ngừng hoạt động")
	}
	member, err := s.activeMember(ctx, memberID, "hội viên")
	if err != nil {
		return nil, err
	}
	plan, err := s.sellablePlan(ctx, planID)
	if err != nil {
		return nil, err
	}
	today := s.today()
	sale := &subSale{member: member, plan: plan, branchID: branchID}
	if strings.TrimSpace(startStr) != "" {
		d, err := subParseDate(startStr, "Ngày bắt đầu")
		if err != nil {
			return nil, err
		}
		if d.Before(today) {
			return nil, rescode.New(rescode.BadRequest, "Ngày bắt đầu không được ở quá khứ")
		}
		sale.start = d
	} else {
		latest, err := s.repo.LatestOpen(ctx, memberID, plan.AccessScope, branchID, today)
		if err != nil {
			return nil, err
		}
		sale.start = subChainStart(today, latest)
		if latest != nil {
			sale.previousID, sale.chained = &latest.ID, true
		}
	}
	if minStart != nil && sale.start.Before(*minStart) {
		sale.start = *minStart
	}
	sale.end = subEndDate(sale.start, plan.DurationDays)
	return sale, nil
}

func subMembershipItem(plan *entity.MembershipPlan, start, end time.Time) InvoiceItemDraft {
	return InvoiceItemDraft{
		ItemType:    entity.InvoiceItemMembership,
		Description: fmt.Sprintf("Gói %s – %s (%s - %s)", plan.Code, plan.Name, invoiceDescDate(start), invoiceDescDate(end)),
		Quantity:    1,
		UnitPrice:   plan.Price,
	}
}

func subPlanRef(p *entity.MembershipPlan) v1.PlanRef {
	return v1.PlanRef{ID: p.ID, Code: p.Code, Name: p.Name, Type: p.Type, DurationDays: p.DurationDays, SessionCount: p.SessionCount,
		Price: p.Price, AccessScope: p.AccessScope, MaxFreezeDays: p.MaxFreezeDays, MaxFreezeTimes: p.MaxFreezeTimes,
		TransferFee: p.TransferFee, Status: p.Status}
}

func (s *subscriptionService) quote(ctx context.Context, sale *subSale, promoCode string) (*v1.QuoteRes, error) {
	pv, err := s.invoices.Preview(ctx, InvoiceDraft{
		MemberID: &sale.member.ID, BranchID: sale.branchID,
		Items:         []InvoiceItemDraft{subMembershipItem(sale.plan, sale.start, sale.end)},
		PromotionCode: promoCode, AppliesTo: entity.AppliesMembership,
	})
	if err != nil {
		return nil, err
	}
	res := &v1.QuoteRes{
		MemberID: sale.member.ID, BranchID: sale.branchID, Plan: subPlanRef(sale.plan),
		StartDate: subFmt(sale.start), EndDate: subFmt(sale.end), Chained: sale.chained, PreviousID: sale.previousID,
		TotalSessions: sale.plan.SessionCount, Subtotal: pv.Subtotal, Discount: pv.Discount, Total: pv.Total,
	}
	if pv.Promotion != nil {
		res.Promotion = &v1.PromotionApplied{ID: pv.Promotion.ID, Code: pv.Promotion.Code, Name: pv.Promotion.Name, Discount: pv.PromotionDiscount}
	}
	return res, nil
}

func (s *subscriptionService) Quote(ctx context.Context, in v1.SellInput) (*v1.QuoteRes, error) {
	sale, err := s.prepareSale(ctx, in.MemberID, in.PlanID, in.BranchID, in.StartDate, nil)
	if err != nil {
		return nil, err
	}
	return s.quote(ctx, sale, in.PromotionCode)
}

type subSellArgs struct {
	sale      *subSale
	promoCode string
	note      string
	source    string
	payment   *PaymentInput
	renewFrom *entity.Subscription // gia hạn từ gói này (event RENEWED)
}

// sell tạo gói PENDING + hóa đơn (+ thu tiền) trong MỘT transaction. Trả id gói mới.
func (s *subscriptionService) sell(ctx context.Context, a subSellArgs) (uint, error) {
	sale := a.sale
	var subID uint
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if a.renewFrom != nil {
			old, err := s.repo.FindByID(ctx, a.renewFrom.ID, true)
			if err != nil {
				return err
			}
			if err := subCheckRenewable(old); err != nil {
				return err
			}
		}
		sub := &entity.Subscription{
			MemberID: sale.member.ID, PlanID: sale.plan.ID, BranchID: sale.branchID,
			StartDate: sale.start, EndDate: sale.end, Price: sale.plan.Price, Status: entity.SubStatusPending,
			PreviousID: sale.previousID, SoldBy: reqctx.UserID(ctx), Note: invoiceTruncate(strings.TrimSpace(a.note), 500),
		}
		if sale.plan.Type == entity.PlanTypeSession && sale.plan.SessionCount != nil {
			n := *sale.plan.SessionCount
			sub.TotalSessions, sub.RemainingSessions = &n, &n
		}
		if sub.PreviousID == nil && a.renewFrom != nil {
			sub.PreviousID = &a.renewFrom.ID
		}
		if err := s.repo.Create(ctx, sub); err != nil {
			return err
		}
		if err := s.event(ctx, sub.ID, entity.SubEventCreated, map[string]any{
			"plan_id": sale.plan.ID, "plan_code": sale.plan.Code, "start_date": subFmt(sale.start), "end_date": subFmt(sale.end),
			"previous_id": sub.PreviousID, "chained": sale.chained,
		}); err != nil {
			return err
		}
		item := subMembershipItem(sale.plan, sale.start, sale.end)
		item.RefID = &sub.ID
		inv, err := s.invoices.CreateInvoice(ctx, InvoiceDraft{
			MemberID: &sale.member.ID, BranchID: sale.branchID, Source: a.source, Items: []InvoiceItemDraft{item},
			PromotionCode: a.promoCode, AppliesTo: entity.AppliesMembership, Note: a.note,
		})
		if err != nil {
			return err
		}
		sub.InvoiceID, sub.Price = &inv.ID, inv.Items[0].Amount
		if err := s.repo.UpdateFields(ctx, sub.ID, map[string]any{"invoice_id": inv.ID, "price": sub.Price}); err != nil {
			return err
		}
		action := AuditSell
		if a.renewFrom != nil {
			action = AuditRenew
			if err := s.event(ctx, a.renewFrom.ID, entity.SubEventRenewed, map[string]any{
				"new_subscription_id": sub.ID, "plan_id": sale.plan.ID, "start_date": subFmt(sale.start), "end_date": subFmt(sale.end),
			}); err != nil {
				return err
			}
		}
		if a.payment != nil {
			if _, err := s.invoices.AddPayment(ctx, inv.ID, *a.payment); err != nil {
				return err
			}
		}
		s.logAudit(ctx, action, auditEntitySubscription, sub.ID, nil, map[string]any{
			"member_id": sub.MemberID, "plan_id": sub.PlanID, "branch_id": sub.BranchID, "start_date": subFmt(sub.StartDate),
			"end_date": subFmt(sub.EndDate), "invoice_id": inv.ID, "total": inv.Total, "renew_from": a.renewFrom != nil,
		})
		subID = sub.ID
		return nil
	})
	return subID, err
}

func (s *subscriptionService) Sell(ctx context.Context, in v1.SellInput, pay *v1.PaymentInput) (*v1.SubscriptionDetail, error) {
	payment, err := subPayment(pay)
	if err != nil {
		return nil, err
	}
	sale, err := s.prepareSale(ctx, in.MemberID, in.PlanID, in.BranchID, in.StartDate, nil)
	if err != nil {
		return nil, err
	}
	id, err := s.sell(ctx, subSellArgs{sale: sale, promoCode: in.PromotionCode, note: in.Note, source: entity.InvoiceSourceCounter, payment: payment})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

func subCheckRenewable(old *entity.Subscription) error {
	switch old.Status {
	case entity.SubStatusActive, entity.SubStatusFrozen, entity.SubStatusExpired:
		return nil
	}
	return rescode.New(rescode.BadRequest, "Chỉ gia hạn được gói đang hoạt động, bảo lưu hoặc đã hết hạn")
}

func (s *subscriptionService) prepareRenew(ctx context.Context, id uint, in v1.RenewInput) (*entity.Subscription, *subSale, error) {
	old, err := s.loadSub(ctx, id, false)
	if err != nil {
		return nil, nil, err
	}
	if err := subCheckRenewable(old); err != nil {
		return nil, nil, err
	}
	planID := in.PlanID
	if planID == 0 {
		planID = old.PlanID
	}
	var minStart *time.Time
	if next := subAddDays(old.EndDate, 1); !next.Before(s.today()) {
		minStart = &next
	}
	sale, err := s.prepareSale(ctx, old.MemberID, planID, old.BranchID, "", minStart)
	if err != nil {
		return nil, nil, err
	}
	if sale.previousID == nil {
		sale.previousID = &old.ID
	}
	return old, sale, nil
}

func (s *subscriptionService) RenewQuote(ctx context.Context, id uint, in v1.RenewInput) (*v1.QuoteRes, error) {
	_, sale, err := s.prepareRenew(ctx, id, in)
	if err != nil {
		return nil, err
	}
	return s.quote(ctx, sale, in.PromotionCode)
}

func (s *subscriptionService) Renew(ctx context.Context, id uint, in v1.RenewInput) (*v1.SubscriptionDetail, error) {
	payment, err := subPayment(in.Payment)
	if err != nil {
		return nil, err
	}
	old, sale, err := s.prepareRenew(ctx, id, in)
	if err != nil {
		return nil, err
	}
	newID, err := s.sell(ctx, subSellArgs{sale: sale, promoCode: in.PromotionCode, note: in.Note,
		source: entity.InvoiceSourceCounter, payment: payment, renewFrom: old})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, newID)
}

// ---- Nâng cấp -------------------------------------------------------------------

type subUpgradeCalc struct {
	old     *entity.Subscription
	oldPlan *entity.MembershipPlan
	newPlan *entity.MembershipPlan
	member  *entity.Member
	quote   v1.UpgradeQuoteRes
}

func (s *subscriptionService) prepareUpgrade(ctx context.Context, id, planID uint) (*subUpgradeCalc, error) {
	old, err := s.loadSub(ctx, id, false)
	if err != nil {
		return nil, err
	}
	if old.Status != entity.SubStatusActive {
		return nil, rescode.New(rescode.BadRequest, "Chỉ nâng cấp được gói đang hoạt động")
	}
	if old.PlanID == planID {
		return nil, rescode.New(rescode.BadRequest, "Gói mới phải khác gói hiện tại")
	}
	oldPlan, err := s.repo.FindPlan(ctx, old.PlanID)
	if err != nil {
		return nil, err
	}
	newPlan, err := s.sellablePlan(ctx, planID)
	if err != nil {
		return nil, err
	}
	member, err := s.activeMember(ctx, old.MemberID, "hội viên")
	if err != nil {
		return nil, err
	}
	today := s.today()
	value, totalDays, remDays := subRemaining(old, oldPlan.Type, today)
	credit := math.Min(value, newPlan.Price)
	start := today
	end := subEndDate(start, newPlan.DurationDays)
	return &subUpgradeCalc{old: old, oldPlan: oldPlan, newPlan: newPlan, member: member, quote: v1.UpgradeQuoteRes{
		SubscriptionID: old.ID, CurrentPlan: subPlanRef(oldPlan), NewPlan: subPlanRef(newPlan), CurrentPrice: old.Price,
		TotalDays: totalDays, RemainingDays: remDays, TotalSessions: old.TotalSessions, RemainingSessions: old.RemainingSessions,
		RemainingValue: value, NewPrice: newPlan.Price, AmountDue: roundMoneyVND(newPlan.Price - credit),
		StartDate: subFmt(start), EndDate: subFmt(end),
	}}, nil
}

func (s *subscriptionService) UpgradeQuote(ctx context.Context, id uint, planID uint) (*v1.UpgradeQuoteRes, error) {
	c, err := s.prepareUpgrade(ctx, id, planID)
	if err != nil {
		return nil, err
	}
	return &c.quote, nil
}

func (s *subscriptionService) Upgrade(ctx context.Context, id uint, in v1.UpgradeInput) (*v1.SubscriptionDetail, error) {
	payment, err := subPayment(in.Payment)
	if err != nil {
		return nil, err
	}
	c, err := s.prepareUpgrade(ctx, id, in.PlanID)
	if err != nil {
		return nil, err
	}
	start, _ := time.Parse(consts.DateLayout, c.quote.StartDate)
	end, _ := time.Parse(consts.DateLayout, c.quote.EndDate)
	credit := roundMoneyVND(c.newPlan.Price - c.quote.AmountDue)
	var newID uint
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		old, err := s.repo.FindByID(ctx, c.old.ID, true)
		if err != nil {
			return err
		}
		if old.Status != entity.SubStatusActive {
			return rescode.New(rescode.Conflict, "Gói tập vừa thay đổi trạng thái, vui lòng tải lại")
		}
		sub := &entity.Subscription{
			MemberID: old.MemberID, PlanID: c.newPlan.ID, BranchID: old.BranchID, StartDate: start, EndDate: end,
			Price: c.newPlan.Price, Status: entity.SubStatusPending, PreviousID: &old.ID, SoldBy: reqctx.UserID(ctx),
			Note: invoiceTruncate(strings.TrimSpace(in.Note), 500),
		}
		if c.newPlan.Type == entity.PlanTypeSession && c.newPlan.SessionCount != nil {
			n := *c.newPlan.SessionCount
			sub.TotalSessions, sub.RemainingSessions = &n, &n
		}
		if err := s.repo.Create(ctx, sub); err != nil {
			return err
		}
		if err := s.event(ctx, sub.ID, entity.SubEventCreated, map[string]any{
			"plan_id": c.newPlan.ID, "plan_code": c.newPlan.Code, "start_date": subFmt(start), "end_date": subFmt(end),
			"upgrade_from": old.ID, "credit": credit,
		}); err != nil {
			return err
		}
		if err := s.repo.UpdateFields(ctx, old.ID, map[string]any{"status": entity.SubStatusCancelled}); err != nil {
			return err
		}
		if err := s.repo.CancelOpenFreezes(ctx, old.ID, entity.FreezeStatusCancelled); err != nil {
			return err
		}
		if err := s.event(ctx, old.ID, entity.SubEventUpgraded, map[string]any{
			"to_subscription_id": sub.ID, "to_plan_id": c.newPlan.ID, "remaining_days": c.quote.RemainingDays,
			"remaining_sessions": old.RemainingSessions, "remaining_value": c.quote.RemainingValue, "credit": credit,
		}); err != nil {
			return err
		}
		item := subMembershipItem(c.newPlan, start, end)
		item.RefID = &sub.ID
		item.Discount = credit
		item.Description = invoiceTruncate(fmt.Sprintf("Nâng cấp %s → %s (%s - %s), khấu trừ giá trị còn lại gói cũ %s",
			c.oldPlan.Code, c.newPlan.Code, invoiceDescDate(start), invoiceDescDate(end), formatMoneyVND(credit)), 255)
		inv, err := s.invoices.CreateInvoice(ctx, InvoiceDraft{
			MemberID: &old.MemberID, BranchID: old.BranchID, Items: []InvoiceItemDraft{item}, Note: in.Note,
		})
		if err != nil {
			return err
		}
		if err := s.repo.UpdateFields(ctx, sub.ID, map[string]any{"invoice_id": inv.ID}); err != nil {
			return err
		}
		if payment != nil {
			if _, err := s.invoices.AddPayment(ctx, inv.ID, *payment); err != nil {
				return err
			}
		}
		s.logAudit(ctx, AuditUpgrade, auditEntitySubscription, old.ID,
			map[string]any{"subscription_id": old.ID, "plan_id": old.PlanID, "status": old.Status},
			map[string]any{"new_subscription_id": sub.ID, "plan_id": c.newPlan.ID, "credit": credit, "invoice_id": inv.ID, "total": inv.Total})
		newID = sub.ID
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, newID)
}

// ---- Chuyển nhượng --------------------------------------------------------------

func (s *subscriptionService) Transfer(ctx context.Context, id uint, in v1.TransferInput) (*v1.SubscriptionDetail, error) {
	payment, err := subPayment(in.Payment)
	if err != nil {
		return nil, err
	}
	old, err := s.loadSub(ctx, id, false)
	if err != nil {
		return nil, err
	}
	if in.ToMemberID == old.MemberID {
		return nil, rescode.New(rescode.BadRequest, "Không thể chuyển nhượng cho chính hội viên đang sở hữu gói")
	}
	to, err := s.activeMember(ctx, in.ToMemberID, "hội viên nhận chuyển nhượng")
	if err != nil {
		return nil, err
	}
	plan, err := s.repo.FindPlan(ctx, old.PlanID)
	if err != nil {
		return nil, err
	}
	today := s.today()
	var newID uint
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		old, err := s.repo.FindByID(ctx, old.ID, true)
		if err != nil {
			return err
		}
		newSub, err := subBuildTransfer(old, plan, to.ID, today)
		if err != nil {
			return err
		}
		newSub.SoldBy = reqctx.UserID(ctx)
		newSub.Note = invoiceTruncate(strings.TrimSpace(in.Note), 500)
		fee := roundMoneyVND(plan.TransferFee)
		if fee <= 0 {
			newSub.Status = entity.SubStatusActive
		}
		if err := s.repo.Create(ctx, newSub); err != nil {
			return err
		}
		if err := s.event(ctx, newSub.ID, entity.SubEventCreated, map[string]any{
			"transfer_from": old.ID, "from_member_id": old.MemberID, "start_date": subFmt(newSub.StartDate), "end_date": subFmt(newSub.EndDate),
			"remaining_sessions": newSub.RemainingSessions, "transfer_fee": fee,
		}); err != nil {
			return err
		}
		if err := s.repo.UpdateFields(ctx, old.ID, map[string]any{"status": entity.SubStatusTransferred}); err != nil {
			return err
		}
		if err := s.repo.CancelOpenFreezes(ctx, old.ID, entity.FreezeStatusCancelled); err != nil {
			return err
		}
		if err := s.event(ctx, old.ID, entity.SubEventTransferred, map[string]any{
			"to_subscription_id": newSub.ID, "to_member_id": to.ID, "transfer_fee": fee,
		}); err != nil {
			return err
		}
		var invID uint
		if fee > 0 {
			inv, err := s.invoices.CreateInvoice(ctx, InvoiceDraft{
				MemberID: &to.ID, BranchID: old.BranchID, Note: in.Note,
				Items: []InvoiceItemDraft{{
					ItemType: entity.InvoiceItemFee, RefID: &newSub.ID, Quantity: 1, UnitPrice: fee,
					Description: invoiceTruncate(fmt.Sprintf("Phí chuyển nhượng gói %s – %s (từ gói #%d)", plan.Code, plan.Name, old.ID), 255),
				}},
			})
			if err != nil {
				return err
			}
			invID = inv.ID
			if err := s.repo.UpdateFields(ctx, newSub.ID, map[string]any{"invoice_id": inv.ID}); err != nil {
				return err
			}
			if payment != nil {
				if _, err := s.invoices.AddPayment(ctx, inv.ID, *payment); err != nil {
					return err
				}
			}
		} else if err := s.event(ctx, newSub.ID, entity.SubEventActivated, map[string]any{"transfer_from": old.ID}); err != nil {
			return err
		}
		s.logAudit(ctx, AuditTransfer, auditEntitySubscription, old.ID,
			map[string]any{"subscription_id": old.ID, "member_id": old.MemberID, "status": old.Status},
			map[string]any{"new_subscription_id": newSub.ID, "to_member_id": to.ID, "fee": fee, "invoice_id": invID})
		newID = newSub.ID
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, newID)
}

// subBuildTransfer dựng gói mới cho người nhận từ phần còn lại của gói old (hàm thuần):
// từ max(hôm nay, start cũ) đến end_date cũ, số buổi còn lại, giá trị = giá trị còn lại; PENDING đến khi thu đủ phí.
func subBuildTransfer(old *entity.Subscription, plan *entity.MembershipPlan, toMemberID uint, today time.Time) (*entity.Subscription, error) {
	if old.Status != entity.SubStatusActive {
		return nil, rescode.New(rescode.BadRequest, "Chỉ chuyển nhượng được gói đang hoạt động")
	}
	if subNorm(old.EndDate).Before(subNorm(today)) {
		return nil, rescode.New(rescode.BadRequest, "Gói tập đã hết hạn, không thể chuyển nhượng")
	}
	if old.RemainingSessions != nil && *old.RemainingSessions <= 0 {
		return nil, rescode.New(rescode.BadRequest, "Gói tập đã hết buổi, không thể chuyển nhượng")
	}
	start := subNorm(today)
	if subNorm(old.StartDate).After(start) {
		start = subNorm(old.StartDate)
	}
	value, _, _ := subRemaining(old, plan.Type, today)
	n := &entity.Subscription{
		MemberID: toMemberID, PlanID: old.PlanID, BranchID: old.BranchID, StartDate: start, EndDate: subNorm(old.EndDate),
		Price: value, Status: entity.SubStatusPending, PreviousID: &old.ID,
		FreezeDaysUsed: old.FreezeDaysUsed, FreezeTimesUsed: old.FreezeTimesUsed,
	}
	if old.RemainingSessions != nil {
		r := *old.RemainingSessions
		t := r
		n.TotalSessions, n.RemainingSessions = &t, &r
	}
	return n, nil
}

// ---- Hủy ------------------------------------------------------------------------

func (s *subscriptionService) Cancel(ctx context.Context, id uint, in v1.CancelInput) (*v1.SubscriptionDetail, error) {
	reason := strings.TrimSpace(in.Reason)
	if reason == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập lý do hủy")
	}
	var refund *RefundInput
	if in.Refund != nil && (in.Refund.Amount != 0 || in.Refund.Method != "") {
		refund = &RefundInput{Amount: in.Refund.Amount, Method: in.Refund.Method, Reason: "Hủy gói tập: " + reason}
	}
	if _, err := s.loadSub(ctx, id, false); err != nil {
		return nil, err
	}
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		sub, err := s.loadSub(ctx, id, true)
		if err != nil {
			return err
		}
		switch sub.Status {
		case entity.SubStatusPending, entity.SubStatusActive, entity.SubStatusFrozen:
		default:
			return rescode.New(rescode.BadRequest, "Gói tập đã kết thúc, không thể hủy")
		}
		var inv *entity.Invoice
		if sub.InvoiceID != nil {
			if inv, err = s.invoices.FindInvoice(ctx, *sub.InvoiceID); err != nil {
				return err
			}
		}
		if refund != nil && inv == nil {
			return rescode.New(rescode.BadRequest, "Gói tập không có hóa đơn để hoàn tiền")
		}
		before := map[string]any{"status": sub.Status}
		if sub.Status == entity.SubStatusPending && inv != nil && inv.Status == entity.InvoiceStatusUnpaid && inv.PaidAmount == 0 && refund == nil {
			// Chưa thu đồng nào → hủy luôn hóa đơn; VoidHandler sẽ chuyển gói sang CANCELLED.
			if err := s.invoices.Void(ctx, inv.ID, "Hủy gói tập: "+reason); err != nil {
				return err
			}
		} else {
			if refund != nil {
				if _, err := s.invoices.Refund(ctx, inv.ID, *refund); err != nil {
					return err
				}
			}
			if err := s.repo.UpdateFields(ctx, sub.ID, map[string]any{"status": entity.SubStatusCancelled}); err != nil {
				return err
			}
			if err := s.repo.CancelOpenFreezes(ctx, sub.ID, entity.FreezeStatusCancelled); err != nil {
				return err
			}
			data := map[string]any{"reason": reason}
			if refund != nil {
				data["refund_amount"], data["refund_method"] = roundMoneyVND(refund.Amount), refund.Method
			}
			if err := s.event(ctx, sub.ID, entity.SubEventCancelled, data); err != nil {
				return err
			}
		}
		after := map[string]any{"status": entity.SubStatusCancelled, "reason": reason}
		if refund != nil {
			after["refund_amount"] = roundMoneyVND(refund.Amount)
		}
		s.logAudit(ctx, AuditCancel, auditEntitySubscription, sub.ID, before, after)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

// ---- Handler hóa đơn ------------------------------------------------------------

// subForInvoice đọc (khóa) gói được dòng hóa đơn trỏ tới; nil nếu dòng không thuộc gói tập của hóa đơn này.
func (s *subscriptionService) subForInvoice(ctx context.Context, inv *entity.Invoice, item *entity.InvoiceItem) (*entity.Subscription, error) {
	if item.RefID == nil {
		return nil, nil
	}
	sub, err := s.repo.FindByID(ctx, *item.RefID, true)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if sub.InvoiceID != nil && *sub.InvoiceID != inv.ID {
		return nil, nil
	}
	return sub, nil
}

// onInvoicePaid: hóa đơn thanh toán đủ → gói PENDING chuyển ACTIVE.
func (s *subscriptionService) onInvoicePaid(ctx context.Context, inv *entity.Invoice, item *entity.InvoiceItem) error {
	sub, err := s.subForInvoice(ctx, inv, item)
	if err != nil || sub == nil || sub.Status != entity.SubStatusPending {
		return err
	}
	if err := s.repo.UpdateFields(ctx, sub.ID, map[string]any{"status": entity.SubStatusActive}); err != nil {
		return err
	}
	return s.event(ctx, sub.ID, entity.SubEventActivated, map[string]any{"invoice_id": inv.ID, "invoice_no": inv.InvoiceNo})
}

// onInvoiceVoid: hóa đơn bị hủy → gói PENDING chuyển CANCELLED; nếu gói là kết quả nâng cấp / chuyển nhượng
// thì khôi phục gói trước đó.
func (s *subscriptionService) onInvoiceVoid(ctx context.Context, inv *entity.Invoice, item *entity.InvoiceItem) error {
	sub, err := s.subForInvoice(ctx, inv, item)
	if err != nil || sub == nil || sub.Status != entity.SubStatusPending {
		return err
	}
	if err := s.repo.UpdateFields(ctx, sub.ID, map[string]any{"status": entity.SubStatusCancelled}); err != nil {
		return err
	}
	if err := s.event(ctx, sub.ID, entity.SubEventCancelled, map[string]any{"reason": inv.VoidReason, "invoice_id": inv.ID}); err != nil {
		return err
	}
	return s.restorePrevious(ctx, sub)
}

func (s *subscriptionService) restorePrevious(ctx context.Context, sub *entity.Subscription) error {
	if sub.PreviousID == nil {
		return nil
	}
	prev, err := s.repo.FindByID(ctx, *sub.PreviousID, true)
	if err != nil {
		return err
	}
	if prev.Status != entity.SubStatusCancelled && prev.Status != entity.SubStatusTransferred {
		return nil
	}
	ev, err := s.repo.LastEvent(ctx, prev.ID, entity.SubEventUpgraded, entity.SubEventTransferred)
	if err != nil || ev == nil {
		return err
	}
	var d struct {
		To uint `json:"to_subscription_id"`
	}
	if json.Unmarshal(ev.Data, &d) != nil || d.To != sub.ID {
		return nil
	}
	status, evType := entity.SubStatusActive, entity.SubEventActivated
	if subNorm(prev.EndDate).Before(s.today()) {
		status, evType = entity.SubStatusExpired, entity.SubEventExpired
	}
	if err := s.repo.UpdateFields(ctx, prev.ID, map[string]any{"status": status}); err != nil {
		return err
	}
	return s.event(ctx, prev.ID, evType, map[string]any{"restored": true, "cancelled_subscription_id": sub.ID})
}

// ---- Truy vấn -------------------------------------------------------------------

var subOrderable = []string{"id", "start_date", "end_date", "created_at", "price", "status"}

func (s *subscriptionService) List(ctx context.Context, req *v1.ListReq) (*v1.ListRes, error) {
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, req.BranchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền xem dữ liệu chi nhánh này")
	}
	today := s.today()
	f := repository.SubscriptionListFilter{BranchIDs: branchIDs, MemberID: req.MemberID, PlanID: req.PlanID, Status: req.Status}
	if req.ExpiringInDays > 0 {
		to := subAddDays(today, req.ExpiringInDays)
		f.ExpiringFrom, f.ExpiringTo = &today, &to
	}
	if req.From != "" {
		d, err := subParseDate(req.From, "Từ ngày")
		if err != nil {
			return nil, err
		}
		f.EndFrom = &d
	}
	if req.To != "" {
		d, err := subParseDate(req.To, "Đến ngày")
		if err != nil {
			return nil, err
		}
		f.EndTo = &d
	}
	q := req.PageReq.ToQuery(subOrderable, "id")
	q.OrderBy = "s." + q.OrderBy
	rows, total, err := s.repo.List(ctx, q, f)
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(rows, func(r repository.SubscriptionRow) *v1.Subscription { return toSubscriptionRes(r, today) }), total, q), nil
}

func (s *subscriptionService) Get(ctx context.Context, id uint) (*v1.SubscriptionDetail, error) {
	row, err := s.repo.FindRow(ctx, id)
	if err != nil {
		return nil, subMapNotFound(err, errSubNotFound)
	}
	if !reqctx.CanAccessBranch(ctx, row.BranchID) {
		return nil, errSubOutOfScope
	}
	return s.detail(ctx, row)
}

func (s *subscriptionService) detail(ctx context.Context, row *repository.SubscriptionRow) (*v1.SubscriptionDetail, error) {
	out := &v1.SubscriptionDetail{
		Subscription: *toSubscriptionRes(*row, s.today()),
		Member:       v1.MemberRef{ID: row.MemberID, MemberCode: row.MemberCode, FullName: row.MemberName, Phone: row.MemberPhone},
	}
	if plan, err := s.repo.FindPlan(ctx, row.PlanID); err == nil {
		out.Plan = subPlanRef(plan)
	} else {
		return nil, err
	}
	if m, err := s.repo.FindMember(ctx, row.MemberID); err == nil {
		out.Member.Email, out.Member.Status = m.Email, m.Status
	}
	if row.InvoiceID != nil {
		inv, err := s.invoices.Detail(ctx, *row.InvoiceID)
		if err != nil {
			return nil, err
		}
		out.Invoice = inv
	}
	freezes, err := s.repo.FreezesBySubscription(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	out.Freezes = dto.Map(freezes, func(f repository.FreezeRow) *v1.Freeze { return toFreezeRes(f) })
	events, err := s.repo.ListEvents(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	out.Events = dto.Map(events, func(e repository.SubscriptionEventRow) *v1.Event { return toEventRes(e) })
	return out, nil
}

func (s *subscriptionService) Events(ctx context.Context, id uint) ([]*v1.Event, error) {
	if _, err := s.loadSub(ctx, id, false); err != nil {
		return nil, err
	}
	events, err := s.repo.ListEvents(ctx, id)
	if err != nil {
		return nil, err
	}
	return dto.Map(events, func(e repository.SubscriptionEventRow) *v1.Event { return toEventRes(e) }), nil
}

// ListByMember: lịch sử gói của hội viên. Nhân viên thấy toàn bộ nếu hội viên thuộc chi nhánh của mình,
// ngược lại chỉ thấy các gói bán tại chi nhánh trong phạm vi.
func (s *subscriptionService) ListByMember(ctx context.Context, memberID uint) ([]*v1.Subscription, error) {
	m, err := s.repo.FindMember(ctx, memberID)
	if err != nil {
		return nil, subMapNotFound(err, rescode.New(rescode.NotFound, "Không tìm thấy hội viên"))
	}
	var branchIDs []uint
	if !reqctx.CanAccessBranch(ctx, m.HomeBranchID) {
		branchIDs = reqctx.AllowedBranches(ctx)
	}
	return s.listByMember(ctx, memberID, branchIDs)
}

func (s *subscriptionService) listByMember(ctx context.Context, memberID uint, branchIDs []uint) ([]*v1.Subscription, error) {
	rows, err := s.repo.ListByMember(ctx, memberID, branchIDs)
	if err != nil {
		return nil, err
	}
	today := s.today()
	return dto.Map(rows, func(r repository.SubscriptionRow) *v1.Subscription { return toSubscriptionRes(r, today) }), nil
}

func (s *subscriptionService) ListMemberSubscriptions(ctx context.Context, memberID uint) ([]*v1.Subscription, error) {
	return s.listByMember(ctx, memberID, nil)
}

func (s *subscriptionService) ListMemberInvoices(ctx context.Context, memberID uint, page dto.PageReq) (*invoicev1.ListRes, error) {
	return s.invoices.ListByMember(ctx, memberID, page)
}

// ---- Mapping --------------------------------------------------------------------

func toSubscriptionRes(r repository.SubscriptionRow, today time.Time) *v1.Subscription {
	out := &v1.Subscription{
		ID: r.ID, MemberID: r.MemberID, MemberCode: r.MemberCode, MemberName: r.MemberName, MemberPhone: r.MemberPhone,
		PlanID: r.PlanID, PlanCode: r.PlanCode, PlanName: r.PlanName, PlanType: r.PlanType, AccessScope: r.AccessScope,
		BranchID: r.BranchID, BranchName: r.BranchName, InvoiceID: r.InvoiceID, InvoiceNo: r.InvoiceNo, InvoiceStatus: r.InvoiceStatus,
		StartDate: subFmt(r.StartDate), EndDate: subFmt(r.EndDate), TotalSessions: r.TotalSessions, RemainingSessions: r.RemainingSessions,
		Price: r.Price, Status: r.Status, FreezeDaysUsed: r.FreezeDaysUsed, FreezeTimesUsed: r.FreezeTimesUsed, PreviousID: r.PreviousID,
		SoldBy: r.SoldBy, SoldByName: r.SoldByName, Note: r.Note, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
	switch r.Status {
	case entity.SubStatusPending, entity.SubStatusActive, entity.SubStatusFrozen:
		d := subDaysBetween(today, r.EndDate)
		out.DaysLeft = &d
	}
	return out
}

func toFreezeRes(f repository.FreezeRow) *v1.Freeze {
	return &v1.Freeze{
		ID: f.ID, SubscriptionID: f.SubscriptionID, MemberID: f.MemberID, MemberCode: f.MemberCode, MemberName: f.MemberName,
		PlanName: f.PlanName, BranchID: f.BranchID, BranchName: f.BranchName, StartDate: subFmt(f.StartDate), EndDate: subFmt(f.EndDate),
		Days: subFreezeDays(&f.SubscriptionFreeze), Reason: f.Reason, Status: f.Status, RequestedBy: f.RequestedBy, RequestedByName: f.RequestedByName,
		ReviewedBy: f.ReviewedBy, ReviewedByName: f.ReviewedByName, ReviewedAt: f.ReviewedAt, ReviewNote: f.ReviewNote, CreatedAt: f.CreatedAt,
	}
}

func toEventRes(e repository.SubscriptionEventRow) *v1.Event {
	data := json.RawMessage(e.Data)
	if len(data) == 0 {
		data = json.RawMessage("{}")
	}
	return &v1.Event{ID: e.ID, EventType: e.EventType, Data: data, ActorID: e.ActorID, ActorName: e.ActorName, CreatedAt: e.CreatedAt}
}

// subWarn ghi log cảnh báo cho tác vụ phụ (thông báo) – không làm hỏng nghiệp vụ chính.
func subWarn(ctx context.Context, format string, args ...any) {
	g.Log().Warningf(ctx, format, args...)
}
