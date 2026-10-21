package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/repository"
)

// ---- Fake dùng chung cho test hóa đơn & gói tập (tiền tố invoice/sub) --------------

type subNoTx struct{}

func (subNoTx) WithTx(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

func subOwnerCtx() context.Context {
	return reqctx.WithAuth(context.Background(), &token.Payload{UserID: 1, Roles: []string{consts.RoleOwner}})
}

// subStaffCtx: lễ tân chỉ thuộc chi nhánh 2.
func subStaffCtx() context.Context {
	return reqctx.WithAuth(context.Background(), &token.Payload{UserID: 2, Roles: []string{consts.RoleReceptionist}, BranchIDs: []uint{2}})
}

// ---- Promotion repo ---------------------------------------------------------------

type invoiceFakePromoRepo struct {
	items map[uint]*entity.Promotion
}

func (f *invoiceFakePromoRepo) List(context.Context, dto.Query, time.Time) ([]entity.Promotion, int64, error) {
	return nil, 0, nil
}
func (f *invoiceFakePromoRepo) FindByID(_ context.Context, id uint) (*entity.Promotion, error) {
	if p, ok := f.items[id]; ok {
		cp := *p
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *invoiceFakePromoRepo) FindByCode(_ context.Context, code string) (*entity.Promotion, error) {
	for _, p := range f.items {
		if strings.EqualFold(p.Code, code) {
			cp := *p
			return &cp, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *invoiceFakePromoRepo) ExistsCode(context.Context, string, uint) (bool, error) {
	return false, nil
}
func (f *invoiceFakePromoRepo) Create(context.Context, *entity.Promotion) error { return nil }
func (f *invoiceFakePromoRepo) Update(context.Context, *entity.Promotion) error { return nil }
func (f *invoiceFakePromoRepo) Delete(context.Context, uint, *uint) error       { return nil }
func (f *invoiceFakePromoRepo) IncrementUsage(_ context.Context, id uint) (int64, error) {
	p, ok := f.items[id]
	if !ok || (p.UsageLimit != nil && p.UsedCount >= *p.UsageLimit) {
		return 0, nil
	}
	p.UsedCount++
	return 1, nil
}

// ---- Invoice repo -----------------------------------------------------------------

type invoiceFakeRepo struct {
	next     uint
	invoices map[uint]*entity.Invoice
	items    map[uint][]entity.InvoiceItem
	payments map[uint][]entity.Payment
	refunds  map[uint][]entity.Refund
	promos   *invoiceFakePromoRepo
	// schedules: kỳ trả góp theo hóa đơn; pendingTxns: giao dịch VNPay PENDING theo hóa đơn.
	schedules   map[uint][]entity.InstallmentSchedule
	pendingTxns map[uint]*entity.OnlinePaymentTxn
	// pendingLookup (tùy chọn) thay pendingTxns – nối với fake repo giao dịch VNPay trong test thanh toán trực tuyến.
	pendingLookup func(ctx context.Context, invoiceID uint, since time.Time) (*entity.OnlinePaymentTxn, error)
}

func newInvoiceFakeRepo(promos *invoiceFakePromoRepo) *invoiceFakeRepo {
	return &invoiceFakeRepo{invoices: map[uint]*entity.Invoice{}, items: map[uint][]entity.InvoiceItem{},
		payments: map[uint][]entity.Payment{}, refunds: map[uint][]entity.Refund{}, promos: promos,
		schedules: map[uint][]entity.InstallmentSchedule{}, pendingTxns: map[uint]*entity.OnlinePaymentTxn{}}
}

func (f *invoiceFakeRepo) id() uint { f.next++; return f.next }

func (f *invoiceFakeRepo) Create(_ context.Context, inv *entity.Invoice) error {
	inv.ID = f.id()
	inv.InvoiceNo = fmt.Sprintf("HD%05d", inv.ID)
	cp := *inv
	cp.Items = nil
	f.invoices[inv.ID] = &cp
	for i := range inv.Items {
		inv.Items[i].ID = f.id()
		inv.Items[i].InvoiceID = inv.ID
	}
	f.items[inv.ID] = append([]entity.InvoiceItem(nil), inv.Items...)
	return nil
}
func (f *invoiceFakeRepo) FindByID(_ context.Context, id uint, _ bool) (*entity.Invoice, error) {
	if inv, ok := f.invoices[id]; ok {
		cp := *inv
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *invoiceFakeRepo) FindRow(ctx context.Context, id uint) (*repository.InvoiceRow, error) {
	inv, err := f.FindByID(ctx, id, false)
	if err != nil {
		return nil, err
	}
	return &repository.InvoiceRow{Invoice: *inv, BranchName: "CN"}, nil
}
func (f *invoiceFakeRepo) Items(_ context.Context, id uint) ([]entity.InvoiceItem, error) {
	return append([]entity.InvoiceItem(nil), f.items[id]...), nil
}
func (f *invoiceFakeRepo) UpdateFields(_ context.Context, id uint, m map[string]any) error {
	inv := f.invoices[id]
	for k, v := range m {
		switch k {
		case "paid_amount":
			inv.PaidAmount = v.(float64)
		case "refunded_amount":
			inv.RefundedAmount = v.(float64)
		case "total":
			inv.Total = v.(float64)
		case "discount":
			inv.Discount = v.(float64)
		case "status":
			inv.Status = v.(string)
		case "voided_at":
			t := v.(time.Time)
			inv.VoidedAt = &t
		case "voided_by":
			inv.VoidedBy = v.(*uint)
		case "void_reason":
			inv.VoidReason = v.(string)
		case "write_off_amount":
			inv.WriteOffAmount = v.(float64)
		case "write_off_reason":
			inv.WriteOffReason = v.(string)
		default:
			panic("invoiceFakeRepo.UpdateFields: " + k)
		}
	}
	return nil
}
func (f *invoiceFakeRepo) CreatePayment(_ context.Context, p *entity.Payment) error {
	p.ID = f.id()
	f.payments[p.InvoiceID] = append(f.payments[p.InvoiceID], *p)
	return nil
}
func (f *invoiceFakeRepo) CreateRefund(_ context.Context, r *entity.Refund) error {
	r.ID = f.id()
	f.refunds[r.InvoiceID] = append(f.refunds[r.InvoiceID], *r)
	return nil
}
func (f *invoiceFakeRepo) Payments(_ context.Context, id uint) ([]repository.PaymentRow, error) {
	var out []repository.PaymentRow
	for _, p := range f.payments[id] {
		out = append(out, repository.PaymentRow{Payment: p})
	}
	return out, nil
}
func (f *invoiceFakeRepo) Refunds(_ context.Context, id uint) ([]repository.RefundRow, error) {
	var out []repository.RefundRow
	for _, r := range f.refunds[id] {
		out = append(out, repository.RefundRow{Refund: r})
	}
	return out, nil
}
func (f *invoiceFakeRepo) List(_ context.Context, _ dto.Query, flt repository.InvoiceListFilter) ([]repository.InvoiceRow, int64, error) {
	var out []repository.InvoiceRow
	for _, inv := range f.invoices {
		if flt.MemberID > 0 && (inv.MemberID == nil || *inv.MemberID != flt.MemberID) {
			continue
		}
		out = append(out, repository.InvoiceRow{Invoice: *inv})
	}
	return out, int64(len(out)), nil
}
func (f *invoiceFakeRepo) ListPayments(context.Context, dto.Query, repository.PaymentLedgerFilter) ([]repository.PaymentRow, int64, error) {
	return nil, 0, nil
}
func (f *invoiceFakeRepo) SumPayments(context.Context, repository.PaymentLedgerFilter) ([]repository.PaymentMethodTotal, error) {
	return nil, nil
}
func (f *invoiceFakeRepo) ReleasePromotion(_ context.Context, id uint) error {
	if p, ok := f.promos.items[id]; ok && p.UsedCount > 0 {
		p.UsedCount--
	}
	return nil
}
func (f *invoiceFakeRepo) FindBranch(_ context.Context, id uint) (*entity.Branch, error) {
	return &entity.Branch{BaseEntity: entity.BaseEntity{ID: id}, Name: "CN"}, nil
}
func (f *invoiceFakeRepo) FindMember(context.Context, uint) (*entity.Member, error) {
	return nil, gorm.ErrRecordNotFound
}
func (f *invoiceFakeRepo) UserName(context.Context, uint) (string, error) { return "Admin", nil }
func (f *invoiceFakeRepo) InstallmentSchedules(_ context.Context, invoiceID uint, _ bool) ([]entity.InstallmentSchedule, error) {
	return append([]entity.InstallmentSchedule(nil), f.schedules[invoiceID]...), nil
}
func (f *invoiceFakeRepo) SaveInstallmentSchedule(_ context.Context, sc *entity.InstallmentSchedule) error {
	for invID, list := range f.schedules {
		for i := range list {
			if list[i].ID == sc.ID {
				f.schedules[invID][i] = *sc
				return nil
			}
		}
	}
	return gorm.ErrRecordNotFound
}
func (f *invoiceFakeRepo) PendingOnlineTxn(ctx context.Context, invoiceID uint, since time.Time) (*entity.OnlinePaymentTxn, error) {
	if f.pendingLookup != nil {
		return f.pendingLookup(ctx, invoiceID, since)
	}
	if t, ok := f.pendingTxns[invoiceID]; ok && t.Status == entity.OnlineTxnPending && !t.CreatedAt.Before(since) {
		cp := *t
		return &cp, nil
	}
	return nil, nil
}

// ---- Subscription repo ------------------------------------------------------------

type subFakeRepo struct {
	next     uint
	subs     map[uint]*entity.Subscription
	events   []entity.SubscriptionEvent
	freezes  map[uint]*entity.SubscriptionFreeze
	plans    map[uint]*entity.MembershipPlan
	members  map[uint]*entity.Member
	branches map[uint]*entity.Branch
	inv      *invoiceFakeRepo
	now      func() time.Time
}

func (f *subFakeRepo) id() uint { f.next++; return f.next }

func (f *subFakeRepo) Create(_ context.Context, s *entity.Subscription) error {
	s.ID = f.id()
	if s.CreatedAt.IsZero() {
		s.CreatedAt = f.now()
	}
	cp := *s
	f.subs[s.ID] = &cp
	return nil
}
func (f *subFakeRepo) FindByID(_ context.Context, id uint, _ bool) (*entity.Subscription, error) {
	if s, ok := f.subs[id]; ok {
		cp := *s
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *subFakeRepo) UpdateFields(_ context.Context, id uint, m map[string]any) error {
	s := f.subs[id]
	for k, v := range m {
		switch k {
		case "status":
			s.Status = v.(string)
		case "end_date":
			s.EndDate = v.(time.Time)
		case "freeze_days_used":
			s.FreezeDaysUsed = v.(int)
		case "freeze_times_used":
			s.FreezeTimesUsed = v.(int)
		case "invoice_id":
			iv := v.(uint)
			s.InvoiceID = &iv
		case "price":
			s.Price = v.(float64)
		default:
			panic("subFakeRepo.UpdateFields: " + k)
		}
	}
	return nil
}
func (f *subFakeRepo) row(s *entity.Subscription) repository.SubscriptionRow {
	p, m := f.plans[s.PlanID], f.members[s.MemberID]
	return repository.SubscriptionRow{Subscription: *s, MemberCode: fmt.Sprintf("GM%06d", m.ID), MemberName: m.FullName,
		PlanCode: p.Code, PlanName: p.Name, PlanType: p.Type, AccessScope: p.AccessScope}
}
func (f *subFakeRepo) FindRow(_ context.Context, id uint) (*repository.SubscriptionRow, error) {
	s, ok := f.subs[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	r := f.row(s)
	return &r, nil
}
func (f *subFakeRepo) sorted() []*entity.Subscription {
	out := make([]*entity.Subscription, 0, len(f.subs))
	for _, s := range f.subs {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out
}
func (f *subFakeRepo) List(_ context.Context, _ dto.Query, flt repository.SubscriptionListFilter) ([]repository.SubscriptionRow, int64, error) {
	var out []repository.SubscriptionRow
	for _, s := range f.sorted() {
		if flt.MemberID > 0 && s.MemberID != flt.MemberID || flt.Status != "" && s.Status != flt.Status {
			continue
		}
		out = append(out, f.row(s))
	}
	return out, int64(len(out)), nil
}
func (f *subFakeRepo) ListByMember(_ context.Context, memberID uint, _ []uint) ([]repository.SubscriptionRow, error) {
	var out []repository.SubscriptionRow
	for _, s := range f.sorted() {
		if s.MemberID == memberID {
			out = append(out, f.row(s))
		}
	}
	return out, nil
}
func (f *subFakeRepo) LatestOpen(_ context.Context, memberID uint, scope string, branchID uint, today time.Time) (*entity.Subscription, error) {
	var best *entity.Subscription
	for _, s := range f.subs {
		p := f.plans[s.PlanID]
		if s.MemberID != memberID || (s.Status != entity.SubStatusActive && s.Status != entity.SubStatusFrozen) ||
			s.EndDate.Before(today) || p.AccessScope != scope || (scope == entity.AccessSingleBranch && s.BranchID != branchID) {
			continue
		}
		if best == nil || s.EndDate.After(best.EndDate) {
			best = s
		}
	}
	if best == nil {
		return nil, nil
	}
	cp := *best
	return &cp, nil
}
func (f *subFakeRepo) CreateEvent(_ context.Context, e *entity.SubscriptionEvent) error {
	e.ID = f.id()
	f.events = append(f.events, *e)
	return nil
}
func (f *subFakeRepo) ListEvents(_ context.Context, subID uint) ([]repository.SubscriptionEventRow, error) {
	var out []repository.SubscriptionEventRow
	for _, e := range f.events {
		if e.SubscriptionID == subID {
			out = append(out, repository.SubscriptionEventRow{SubscriptionEvent: e})
		}
	}
	return out, nil
}
func (f *subFakeRepo) eventTypes(subID uint) []string {
	var out []string
	for _, e := range f.events {
		if e.SubscriptionID == subID {
			out = append(out, e.EventType)
		}
	}
	return out
}
func (f *subFakeRepo) LastEvent(_ context.Context, subID uint, types ...string) (*entity.SubscriptionEvent, error) {
	for i := len(f.events) - 1; i >= 0; i-- {
		e := f.events[i]
		if e.SubscriptionID == subID {
			for _, t := range types {
				if e.EventType == t {
					return &e, nil
				}
			}
		}
	}
	return nil, nil
}
func (f *subFakeRepo) CreateFreeze(_ context.Context, fr *entity.SubscriptionFreeze) error {
	fr.ID = f.id()
	fr.Days = subFreezeDays(fr)
	cp := *fr
	f.freezes[fr.ID] = &cp
	return nil
}
func (f *subFakeRepo) FindFreeze(_ context.Context, id uint, _ bool) (*entity.SubscriptionFreeze, error) {
	if fr, ok := f.freezes[id]; ok {
		cp := *fr
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *subFakeRepo) UpdateFreezeFields(_ context.Context, id uint, m map[string]any) error {
	fr := f.freezes[id]
	for k, v := range m {
		switch k {
		case "status":
			fr.Status = v.(string)
		case "end_date":
			fr.EndDate = v.(time.Time)
			fr.Days = subFreezeDays(fr)
		case "reviewed_by":
			fr.ReviewedBy = v.(*uint)
		case "reviewed_at":
			t := v.(time.Time)
			fr.ReviewedAt = &t
		case "review_note":
			fr.ReviewNote = v.(string)
		default:
			panic("subFakeRepo.UpdateFreezeFields: " + k)
		}
	}
	return nil
}
func (f *subFakeRepo) freezeRow(fr *entity.SubscriptionFreeze) repository.FreezeRow {
	s := f.subs[fr.SubscriptionID]
	return repository.FreezeRow{SubscriptionFreeze: *fr, MemberID: s.MemberID, BranchID: s.BranchID, PlanName: f.plans[s.PlanID].Name}
}
func (f *subFakeRepo) FreezesBySubscription(_ context.Context, subID uint) ([]repository.FreezeRow, error) {
	var out []repository.FreezeRow
	for _, fr := range f.freezes {
		if fr.SubscriptionID == subID {
			out = append(out, f.freezeRow(fr))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}
func (f *subFakeRepo) FindFreezeRow(_ context.Context, id uint) (*repository.FreezeRow, error) {
	fr, ok := f.freezes[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	r := f.freezeRow(fr)
	return &r, nil
}
func (f *subFakeRepo) ListFreezes(context.Context, dto.Query, repository.FreezeListFilter) ([]repository.FreezeRow, int64, error) {
	return nil, 0, nil
}
func (f *subFakeRepo) CancelOpenFreezes(_ context.Context, subID uint, status string) error {
	for _, fr := range f.freezes {
		if fr.SubscriptionID == subID && (fr.Status == entity.FreezeStatusPending || fr.Status == entity.FreezeStatusApproved) {
			fr.Status = status
		}
	}
	return nil
}
func (f *subFakeRepo) IDsToExpire(_ context.Context, today time.Time) ([]uint, error) {
	var ids []uint
	for _, s := range f.subs {
		if subShouldExpire(s, today) {
			ids = append(ids, s.ID)
		}
	}
	return ids, nil
}
func (f *subFakeRepo) StalePending(_ context.Context, before time.Time) ([]entity.Subscription, error) {
	var out []entity.Subscription
	for _, s := range f.sorted() {
		if s.Status != entity.SubStatusPending || !s.CreatedAt.Before(before) {
			continue
		}
		if s.InvoiceID != nil {
			inv := f.inv.invoices[*s.InvoiceID]
			if inv.PaidAmount > 0 || inv.Status != entity.InvoiceStatusUnpaid {
				continue
			}
		}
		out = append(out, *s)
	}
	return out, nil
}
func (f *subFakeRepo) FreezeIDsToStart(_ context.Context, today time.Time) ([]uint, error) {
	var ids []uint
	for _, fr := range f.freezes {
		if fr.Status == entity.FreezeStatusApproved && !fr.StartDate.After(today) && !fr.EndDate.Before(today) &&
			f.subs[fr.SubscriptionID].Status == entity.SubStatusActive {
			ids = append(ids, fr.ID)
		}
	}
	return ids, nil
}
func (f *subFakeRepo) FreezeIDsToEnd(_ context.Context, today time.Time) ([]uint, error) {
	var ids []uint
	for _, fr := range f.freezes {
		if fr.Status == entity.FreezeStatusApproved && fr.EndDate.Before(today) && f.subs[fr.SubscriptionID].Status == entity.SubStatusFrozen {
			ids = append(ids, fr.ID)
		}
	}
	return ids, nil
}
func (f *subFakeRepo) ListExpiring(_ context.Context, dates []time.Time) ([]repository.ExpiringSubscription, error) {
	var out []repository.ExpiringSubscription
	for _, s := range f.subs {
		for _, d := range dates {
			if s.Status == entity.SubStatusActive && s.EndDate.Equal(d) {
				out = append(out, repository.ExpiringSubscription{SubscriptionID: s.ID, MemberID: s.MemberID, EndDate: s.EndDate})
			}
		}
	}
	return out, nil
}
func (f *subFakeRepo) FindPlan(_ context.Context, id uint) (*entity.MembershipPlan, error) {
	if p, ok := f.plans[id]; ok {
		cp := *p
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *subFakeRepo) FindMember(_ context.Context, id uint) (*entity.Member, error) {
	if m, ok := f.members[id]; ok {
		cp := *m
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *subFakeRepo) FindBranch(_ context.Context, id uint) (*entity.Branch, error) {
	if b, ok := f.branches[id]; ok {
		cp := *b
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}

// ---- Fixture ----------------------------------------------------------------------

type subTestFixture struct {
	svc    *subscriptionService
	inv    *invoiceService
	repo   *subFakeRepo
	invRep *invoiceFakeRepo
	promos *invoiceFakePromoRepo
	clock  time.Time
}

// subTestToday: hôm nay trong test là 14/10/2026 (giờ VN).
var subTestToday = time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)

func subD(m time.Month, d int) time.Time { return time.Date(2026, m, d, 0, 0, 0, 0, time.UTC) }

func subIntPtr(v int) *int { return &v }

func newSubFixture() *subTestFixture {
	fx := &subTestFixture{clock: time.Date(2026, 10, 14, 9, 0, 0, 0, subVNLoc)}
	limit, maxDisc := 100, 500000.0
	fx.promos = &invoiceFakePromoRepo{items: map[uint]*entity.Promotion{
		1: {BaseEntity: entity.BaseEntity{ID: 1}, Code: "CHAODONG", Name: "Chào đông", Type: entity.PromotionPercent, Value: 10,
			MaxDiscount: &maxDisc, MinAmount: 500000, AppliesTo: entity.AppliesMembership, Status: consts.StatusActive,
			StartAt: time.Date(2026, 10, 1, 0, 0, 0, 0, subVNLoc), EndAt: time.Date(2026, 12, 31, 23, 59, 59, 0, subVNLoc), UsageLimit: &limit},
	}}
	fx.invRep = newInvoiceFakeRepo(fx.promos)
	plan := func(id uint, code, typ string, days int, sessions *int, price float64, scope string, fd, ft int, fee float64) *entity.MembershipPlan {
		return &entity.MembershipPlan{BaseEntity: entity.BaseEntity{ID: id}, Code: code, Name: "Gói " + code, Type: typ, DurationDays: days,
			SessionCount: sessions, Price: price, AccessScope: scope, MaxFreezeDays: fd, MaxFreezeTimes: ft, TransferFee: fee, Status: consts.StatusActive}
	}
	uid := uint(50)
	fx.repo = &subFakeRepo{
		next: 1000, subs: map[uint]*entity.Subscription{}, freezes: map[uint]*entity.SubscriptionFreeze{},
		plans: map[uint]*entity.MembershipPlan{
			1: plan(1, "M1", entity.PlanTypeTime, 30, nil, 500000, entity.AccessSingleBranch, 7, 1, 100000),
			2: plan(2, "M3", entity.PlanTypeTime, 90, nil, 1350000, entity.AccessAllBranches, 15, 1, 100000),
			3: plan(3, "M12", entity.PlanTypeTime, 365, nil, 4800000, entity.AccessAllBranches, 60, 3, 200000),
			4: plan(4, "S12", entity.PlanTypeSession, 60, subIntPtr(12), 600000, entity.AccessSingleBranch, 0, 0, 0),
		},
		members: map[uint]*entity.Member{
			1: {BaseEntity: entity.BaseEntity{ID: 1}, FullName: "An", Status: entity.MemberStatusActive, HomeBranchID: 1, UserID: &uid},
			2: {BaseEntity: entity.BaseEntity{ID: 2}, FullName: "Bình", Status: entity.MemberStatusActive, HomeBranchID: 1},
			3: {BaseEntity: entity.BaseEntity{ID: 3}, FullName: "Cường", Status: entity.MemberStatusInactive, HomeBranchID: 1},
		},
		branches: map[uint]*entity.Branch{
			1: {BaseEntity: entity.BaseEntity{ID: 1}, Name: "CN1", Status: consts.StatusActive},
			2: {BaseEntity: entity.BaseEntity{ID: 2}, Name: "CN2", Status: consts.StatusActive},
		},
		inv: fx.invRep,
	}
	fx.repo.now = func() time.Time { return fx.clock }
	fx.inv = NewInvoiceService(fx.invRep, NewPromotionService(fx.promos, nil), subNoTx{}, nil).(*invoiceService)
	fx.inv.now = func() time.Time { return fx.clock }
	fx.svc = NewSubscriptionService(fx.repo, fx.inv, subNoTx{}, nil, nil).(*subscriptionService)
	fx.svc.now = func() time.Time { return fx.clock }
	return fx
}

// setDay đặt "hôm nay" của fixture (09:00 giờ VN).
func (fx *subTestFixture) setDay(d time.Time) {
	fx.clock = time.Date(d.Year(), d.Month(), d.Day(), 9, 0, 0, 0, subVNLoc)
}

// addSub tạo sẵn một gói trong repo.
func (fx *subTestFixture) addSub(s entity.Subscription) *entity.Subscription {
	_ = fx.repo.Create(context.Background(), &s)
	return fx.repo.subs[s.ID]
}
