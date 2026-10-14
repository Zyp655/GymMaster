package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"gorm.io/gorm"

	v1 "gymmaster-api/api/pt/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/storage/postgres"
)

// Audit của module PT (entity: pt_contract | pt_session | trainer_commission).
const (
	ptAuditContract   = "pt_contract"
	ptAuditSession    = "pt_session"
	ptAuditCommission = "trainer_commission"

	ptActSell          = "SELL"
	ptActActivate      = "ACTIVATE"
	ptActChangeTrainer = "CHANGE_TRAINER"
	ptActCancel        = "CANCEL"
	ptActExpire        = "EXPIRE"
	ptActSchedule      = "SCHEDULE"
	ptActReschedule    = "RESCHEDULE"
	ptActComplete      = "COMPLETE"
	ptActNoShow        = "NO_SHOW"
	ptActAutoNoShow    = "AUTO_NO_SHOW"
	ptActPay           = "PAY"
)

// Loại thông báo riêng của PT (ngoài PT_SCHEDULED / PT_CANCELLED dùng chung).
const (
	NotifyPTNoShow         = "PT_NO_SHOW"
	NotifyCommissionPaid   = "COMMISSION_PAID"
	ptDefaultCancelMinutes = 720
	// ptNoShowGrace: buổi SCHEDULED quá giờ kết thúc hơn khoảng này mà chưa xác nhận → cron đánh dấu NO_SHOW (không trừ buổi).
	ptNoShowGrace = 2 * time.Hour
	// ptMaxRangeDays: khoảng ngày tối đa khi xem lịch buổi PT.
	ptMaxRangeDays = 62
)

type (
	// PTService: bán gói PT (hợp đồng), đặt/đổi/hủy/hoàn thành buổi PT, hoa hồng HLV,
	// cùng các hàm dùng chung cho cổng hội viên và cron.
	PTService interface {
		// ---- Hợp đồng (admin) ----
		Quote(ctx context.Context, in v1.SellInput) (*v1.QuoteRes, error)
		Sell(ctx context.Context, in v1.SellInput, pay *v1.PaymentInput) (*v1.ContractDetail, error)
		ListContracts(ctx context.Context, req *v1.ContractListReq) (*v1.ContractListRes, error)
		GetContract(ctx context.Context, id uint) (*v1.ContractDetail, error)
		ListByMember(ctx context.Context, memberID uint) ([]*v1.Contract, error)
		CheckChangeTrainer(ctx context.Context, id uint, in v1.ChangeTrainerInput) (*v1.ChangeTrainerCheckRes, error)
		ChangeTrainer(ctx context.Context, id uint, in v1.ChangeTrainerInput) (*v1.ContractDetail, error)
		CancelContract(ctx context.Context, id uint, in v1.CancelContractInput) (*v1.ContractDetail, error)

		// ---- Buổi PT (admin) ----
		ListSessions(ctx context.Context, req *v1.SessionListReq) ([]*v1.Session, error)
		GetSession(ctx context.Context, id uint) (*v1.Session, error)
		CreateSession(ctx context.Context, contractID uint, startAt string) (*v1.Session, error)
		RescheduleSession(ctx context.Context, id uint, startAt string) (*v1.Session, error)
		CancelSession(ctx context.Context, id uint, reason string, countAsUsed bool) (*v1.Session, error)
		CompleteSession(ctx context.Context, id uint, workoutNote, trainerNote string) (*v1.Session, error)
		MarkNoShow(ctx context.Context, id uint, note string) (*v1.Session, error)
		// TrainerScheduleAdmin: lịch dạy (PT + lớp) cho màn hình admin – có phạm vi chi nhánh, HLV chỉ xem của mình.
		TrainerScheduleAdmin(ctx context.Context, req *v1.TrainerScheduleReq) ([]*v1.ScheduleItem, error)

		// ---- Hoa hồng ----
		ListCommissions(ctx context.Context, req *v1.CommissionListReq) (*v1.CommissionListRes, error)
		CommissionSummary(ctx context.Context, req *v1.CommissionSummaryReq) (*v1.CommissionSummaryRes, error)
		PayCommissions(ctx context.Context, req *v1.CommissionPayReq) (*v1.CommissionPayRes, error)

		// ---- Dùng chung: cổng hội viên (memberID lấy từ tài khoản đăng nhập, không kiểm tra phạm vi chi nhánh) ----
		ListMemberContracts(ctx context.Context, memberID uint) ([]*v1.Contract, error)
		// ListMemberSessions: buổi PT của hội viên có start_at trong [from, to) (tối đa 62 ngày).
		ListMemberSessions(ctx context.Context, memberID uint, from, to time.Time) ([]*v1.Session, error)
		// BookSessionByMember: cùng quy tắc như đặt ở quầy; hợp đồng phải thuộc hội viên. source = PORTAL.
		BookSessionByMember(ctx context.Context, memberID, contractID uint, startAt time.Time) (*v1.Session, error)
		// CancelSessionByMember: chỉ được hủy trước giờ tập ít nhất pt.cancel_before_minutes (không tính buổi).
		CancelSessionByMember(ctx context.Context, memberID, sessionID uint, reason string) (*v1.Session, error)

		// ---- Dùng chung: lịch dạy của HLV ----
		// TrainerSchedule gộp buổi PT (trừ buổi đã hủy) và buổi lớp nhóm (chưa hủy) có start_at trong [from, to), sắp theo giờ.
		TrainerSchedule(ctx context.Context, trainerID uint, from, to time.Time) ([]*v1.ScheduleItem, error)

		// ---- Dùng chung: cron ----
		// ExpireContracts: hợp đồng ACTIVE có end_date < today (ngày VN) → EXPIRED; hủy buổi SCHEDULED sau hạn.
		ExpireContracts(ctx context.Context, today time.Time) (int, error)
		// MarkPTNoShows: buổi SCHEDULED có end_at < now − 2 giờ → NO_SHOW "chưa tính buổi" (counted=false,
		// KHÔNG trừ buổi, KHÔNG hoa hồng) + thông báo HLV và quản lý chi nhánh để xử lý. Trả số buổi đã đánh dấu.
		MarkPTNoShows(ctx context.Context, now time.Time) (int, error)
	}

	ptService struct {
		repo     repository.PTRepository
		invoices InvoiceService
		tx       postgres.TxManager
		settings SettingService
		audit    AuditService
		notifier Notifier
		now      func() time.Time
	}
)

// NewPTService tạo service PT và đăng ký PaidHandler/VoidHandler cho dòng hóa đơn "PT" vào InvoiceService dùng chung.
// settings/audit/notifier có thể nil (test). Handler idempotent nên tạo service nhiều lần cũng không sai dữ liệu.
func NewPTService(repo repository.PTRepository, invoices InvoiceService, tx postgres.TxManager,
	settings SettingService, audit AuditService, notifier Notifier) PTService {
	s := &ptService{repo: repo, invoices: invoices, tx: tx, settings: settings, audit: audit, notifier: notifier, now: time.Now}
	invoices.RegisterPaidHandler(entity.InvoiceItemPT, s.onInvoicePaid)
	invoices.RegisterVoidHandler(entity.InvoiceItemPT, s.onInvoiceVoid)
	return s
}

// ---- Tiện ích -------------------------------------------------------------------

var (
	errPTContractNotFound = rescode.New(rescode.NotFound, "Không tìm thấy hợp đồng PT")
	errPTSessionNotFound  = rescode.New(rescode.NotFound, "Không tìm thấy buổi PT")
	errPTOutOfScope       = rescode.New(rescode.PermissionDenied, "Hợp đồng PT thuộc chi nhánh ngoài phạm vi của bạn")
	errPTSessionScope     = rescode.New(rescode.PermissionDenied, "Buổi PT thuộc chi nhánh ngoài phạm vi của bạn")
	errPTNotOwnContract   = rescode.New(rescode.PermissionDenied, "Bạn chỉ được xem/thao tác hợp đồng PT của chính mình")
	errPTNotOwnSession    = rescode.New(rescode.PermissionDenied, "Bạn chỉ được xem/thao tác buổi PT của chính mình")
	errPTOnlyOwnData      = rescode.New(rescode.PermissionDenied, "Bạn chỉ được xem dữ liệu của chính mình")
)

func ptNotFound(err error, nf error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nf
	}
	return err
}

// ptTrainerOnly: người gọi chỉ có vai trò HLV (không OWNER/MANAGER/RECEPTIONIST…) → mặc định chỉ thấy dữ liệu của mình.
func ptTrainerOnly(ctx context.Context) bool {
	p := reqctx.Auth(ctx)
	if p == nil || len(p.Roles) == 0 {
		return false
	}
	for _, r := range p.Roles {
		if r != consts.RoleTrainer {
			return false
		}
	}
	return true
}

func ptSelf(ctx context.Context) uint {
	if p := reqctx.Auth(ctx); p != nil {
		return p.UserID
	}
	return 0
}

// ptPerSession: giá mỗi buổi = round(price / total_sessions).
func ptPerSession(price float64, total int) float64 {
	if total <= 0 {
		return 0
	}
	return roundMoneyVND(price / float64(total))
}

// ptCommission tính hoa hồng một buổi: base = giá/buổi, amount = round(base × rate / 100).
func ptCommission(price float64, total int, rate float64) (base, amount float64) {
	base = ptPerSession(price, total)
	return base, roundMoneyVND(base * rate / 100)
}

// ptPeriod: kỳ hoa hồng YYYY-MM theo giờ Việt Nam.
func ptPeriod(t time.Time) string { return t.In(subVNLoc).Format("2006-01") }

func ptValidPeriod(p string) error {
	t, err := time.Parse("2006-01", strings.TrimSpace(p))
	if err != nil || t.Format("2006-01") != strings.TrimSpace(p) {
		return rescode.New(rescode.BadRequest, "Kỳ phải có dạng YYYY-MM")
	}
	return nil
}

func ptFmtTime(t time.Time) string { return t.In(subVNLoc).Format("15:04 02/01/2006") }

func ptFmtMinutes(m int) string {
	if m%60 == 0 {
		return fmt.Sprintf("%d giờ", m/60)
	}
	return fmt.Sprintf("%d phút", m)
}

// ptRange đổi khoảng ngày from/to (YYYY-MM-DD, tính cả hai đầu, giờ VN) thành [from 00:00, to+1 00:00), tối đa 62 ngày.
func ptRange(from, to string) (time.Time, time.Time, error) {
	f, err := subParseDate(from, "Từ ngày")
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	t, err := subParseDate(to, "Đến ngày")
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	start := time.Date(f.Year(), f.Month(), f.Day(), 0, 0, 0, 0, subVNLoc)
	end := time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, subVNLoc)
	return start, end, ptCheckRange(start, end)
}

func ptCheckRange(from, to time.Time) error {
	if !from.Before(to) {
		return rescode.New(rescode.BadRequest, "Từ ngày phải trước hoặc bằng đến ngày")
	}
	if to.Sub(from) > ptMaxRangeDays*24*time.Hour+time.Hour { // +1h dư cho chuyển giờ (VN không có DST)
		return rescode.New(rescode.BadRequest, fmt.Sprintf("Khoảng thời gian xem lịch tối đa %d ngày", ptMaxRangeDays))
	}
	return nil
}

func (s *ptService) today() time.Time { return subDateOf(s.now()) }

func (s *ptService) cancelBeforeMinutes(ctx context.Context) int {
	if s.settings == nil {
		return ptDefaultCancelMinutes
	}
	return s.settings.Int(ctx, SettingPTCancelBeforeMinutes, ptDefaultCancelMinutes)
}

func (s *ptService) logAudit(ctx context.Context, action, entityName string, id uint, before, after any) {
	auditCatalog(ctx, s.audit, action, entityName, id, before, after)
}

// notify gửi thông báo in-app (sau khi giao dịch đã commit); lỗi chỉ ghi log.
func (s *ptService) notify(ctx context.Context, userIDs []uint, typ, title, body, link string) {
	if s.notifier == nil {
		return
	}
	ids := make([]uint, 0, len(userIDs))
	seen := map[uint]bool{}
	for _, id := range userIDs {
		if id > 0 && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	if err := s.notifier.Notify(ctx, ids, typ, title, body, link); err != nil {
		g.Log().Warningf(ctx, "pt notify: %v", err)
	}
}

func ptPayment(p *v1.PaymentInput) (*PaymentInput, error) {
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

// ---- Truy cập hợp đồng ----------------------------------------------------------

// checkContractAccess: phạm vi chi nhánh + HLV chỉ thao tác hợp đồng của mình.
func (s *ptService) checkContractAccess(ctx context.Context, c *entity.PTContract) error {
	if !reqctx.CanAccessBranch(ctx, c.BranchID) {
		return errPTOutOfScope
	}
	if ptTrainerOnly(ctx) && c.TrainerID != ptSelf(ctx) {
		return errPTNotOwnContract
	}
	return nil
}

func (s *ptService) loadContract(ctx context.Context, id uint, lock bool) (*entity.PTContract, error) {
	c, err := s.repo.FindContract(ctx, id, lock)
	if err != nil {
		return nil, ptNotFound(err, errPTContractNotFound)
	}
	if err := s.checkContractAccess(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

// ---- Bán gói PT -----------------------------------------------------------------

type ptSale struct {
	member      *entity.Member
	pkg         *entity.PTPackage
	branchID    uint
	trainerID   uint
	trainerName string
	start, end  time.Time
}

func (s *ptService) prepareSale(ctx context.Context, in v1.SellInput) (*ptSale, error) {
	if !reqctx.CanAccessBranch(ctx, in.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền bán gói PT tại chi nhánh này")
	}
	branch, err := s.repo.FindBranch(ctx, in.BranchID)
	if err != nil {
		return nil, ptNotFound(err, rescode.New(rescode.BadRequest, "Chi nhánh không tồn tại"))
	}
	if branch.Status != consts.StatusActive {
		return nil, rescode.New(rescode.BadRequest, "Chi nhánh đang ngừng hoạt động")
	}
	member, err := s.repo.FindMember(ctx, in.MemberID)
	if err != nil {
		return nil, ptNotFound(err, rescode.New(rescode.NotFound, "Không tìm thấy hội viên"))
	}
	if member.Status != entity.MemberStatusActive {
		return nil, rescode.New(rescode.BadRequest, "Hội viên đang không hoạt động")
	}
	pkg, err := s.repo.FindPackage(ctx, in.PTPackageID)
	if err != nil || pkg.DeletedAt.Valid {
		if err == nil {
			err = gorm.ErrRecordNotFound
		}
		return nil, ptNotFound(err, rescode.New(rescode.NotFound, "Không tìm thấy gói PT"))
	}
	if pkg.Status != consts.StatusActive {
		return nil, rescode.New(rescode.BadRequest, "Gói PT "+pkg.Name+" đã ngừng bán")
	}
	if pkg.SessionCount <= 0 {
		return nil, rescode.New(rescode.BadRequest, "Gói PT không có số buổi hợp lệ")
	}
	ok, err := s.repo.TrainerInBranch(ctx, in.TrainerID, in.BranchID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, rescode.New(rescode.BadRequest, "HLV không thuộc chi nhánh, không phải HLV hoặc đã bị khóa")
	}
	trainerName, err := s.repo.UserName(ctx, in.TrainerID)
	if err != nil {
		return nil, err
	}
	today := s.today()
	start := today
	if strings.TrimSpace(in.StartDate) != "" {
		d, err := subParseDate(in.StartDate, "Ngày bắt đầu")
		if err != nil {
			return nil, err
		}
		if d.Before(today) {
			return nil, rescode.New(rescode.BadRequest, "Ngày bắt đầu không được ở quá khứ")
		}
		start = d
	}
	return &ptSale{
		member: member, pkg: pkg, branchID: in.BranchID, trainerID: in.TrainerID, trainerName: trainerName,
		start: start, end: subEndDate(start, max(1, pkg.ValidityDays)),
	}, nil
}

func ptItem(sale *ptSale) InvoiceItemDraft {
	desc := fmt.Sprintf("Gói PT %s – %s, HLV %s (%s - %s)", sale.pkg.Code, sale.pkg.Name, sale.trainerName,
		invoiceDescDate(sale.start), invoiceDescDate(sale.end))
	return InvoiceItemDraft{ItemType: entity.InvoiceItemPT, Description: invoiceTruncate(desc, 255), Quantity: 1, UnitPrice: sale.pkg.Price}
}

func ptPackageRef(p *entity.PTPackage) v1.PackageRef {
	return v1.PackageRef{ID: p.ID, Code: p.Code, Name: p.Name, SessionCount: p.SessionCount, SessionMinutes: p.SessionMinutes,
		ValidityDays: p.ValidityDays, Price: p.Price, Status: p.Status}
}

func ptSessionMinutes(p *entity.PTPackage) int {
	if p.SessionMinutes <= 0 {
		return 60
	}
	return p.SessionMinutes
}

func (s *ptService) Quote(ctx context.Context, in v1.SellInput) (*v1.QuoteRes, error) {
	sale, err := s.prepareSale(ctx, in)
	if err != nil {
		return nil, err
	}
	pv, err := s.invoices.Preview(ctx, InvoiceDraft{
		MemberID: &sale.member.ID, BranchID: sale.branchID, Items: []InvoiceItemDraft{ptItem(sale)},
		PromotionCode: in.PromotionCode, AppliesTo: entity.AppliesPT,
	})
	if err != nil {
		return nil, err
	}
	res := &v1.QuoteRes{
		MemberID: sale.member.ID, BranchID: sale.branchID, TrainerID: sale.trainerID, TrainerName: sale.trainerName,
		Package: ptPackageRef(sale.pkg), StartDate: subFmt(sale.start), EndDate: subFmt(sale.end),
		TotalSessions: sale.pkg.SessionCount, SessionMinutes: ptSessionMinutes(sale.pkg),
		Subtotal: pv.Subtotal, Discount: pv.Discount, Total: pv.Total, PricePerSession: ptPerSession(pv.Total, sale.pkg.SessionCount),
	}
	if pv.Promotion != nil {
		res.Promotion = &v1.PromotionApplied{ID: pv.Promotion.ID, Code: pv.Promotion.Code, Name: pv.Promotion.Name, Discount: pv.PromotionDiscount}
	}
	return res, nil
}

func (s *ptService) Sell(ctx context.Context, in v1.SellInput, pay *v1.PaymentInput) (*v1.ContractDetail, error) {
	payment, err := ptPayment(pay)
	if err != nil {
		return nil, err
	}
	sale, err := s.prepareSale(ctx, in)
	if err != nil {
		return nil, err
	}
	var id uint
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		c := &entity.PTContract{
			MemberID: sale.member.ID, TrainerID: sale.trainerID, PTPackageID: sale.pkg.ID, BranchID: sale.branchID,
			TotalSessions: sale.pkg.SessionCount, SessionMinutes: ptSessionMinutes(sale.pkg),
			StartDate: sale.start, EndDate: sale.end, Price: sale.pkg.Price, Status: entity.PTContractPending,
			SoldBy: reqctx.UserID(ctx), Note: invoiceTruncate(strings.TrimSpace(in.Note), 500),
		}
		if err := s.repo.CreateContract(ctx, c); err != nil {
			return err
		}
		item := ptItem(sale)
		item.RefID = &c.ID
		inv, err := s.invoices.CreateInvoice(ctx, InvoiceDraft{
			MemberID: &sale.member.ID, BranchID: sale.branchID, Items: []InvoiceItemDraft{item},
			PromotionCode: in.PromotionCode, AppliesTo: entity.AppliesPT, Note: in.Note,
		})
		if err != nil {
			return err
		}
		c.InvoiceID, c.Price = &inv.ID, inv.Items[0].Amount
		if err := s.repo.UpdateContractFields(ctx, c.ID, map[string]any{"invoice_id": inv.ID, "price": c.Price}); err != nil {
			return err
		}
		if payment != nil {
			if _, err := s.invoices.AddPayment(ctx, inv.ID, *payment); err != nil {
				return err
			}
		}
		s.logAudit(ctx, ptActSell, ptAuditContract, c.ID, nil, map[string]any{
			"member_id": c.MemberID, "trainer_id": c.TrainerID, "pt_package_id": c.PTPackageID, "branch_id": c.BranchID,
			"total_sessions": c.TotalSessions, "start_date": subFmt(c.StartDate), "end_date": subFmt(c.EndDate),
			"price": c.Price, "invoice_id": inv.ID, "total": inv.Total,
		})
		id = c.ID
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.detail(ctx, id)
}

// onInvoicePaid: hóa đơn PT thanh toán đủ → hợp đồng PENDING chuyển ACTIVE (idempotent).
func (s *ptService) onInvoicePaid(ctx context.Context, inv *entity.Invoice, item *entity.InvoiceItem) error {
	if item.RefID == nil {
		return nil
	}
	c, err := s.repo.FindContract(ctx, *item.RefID, true)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if c.Status != entity.PTContractPending {
		return nil
	}
	if err := s.repo.UpdateContractFields(ctx, c.ID, map[string]any{"status": entity.PTContractActive}); err != nil {
		return err
	}
	s.logAudit(ctx, ptActActivate, ptAuditContract, c.ID, map[string]any{"status": c.Status},
		map[string]any{"status": entity.PTContractActive, "invoice_id": inv.ID, "invoice_no": inv.InvoiceNo})
	return nil
}

// onInvoiceVoid: hóa đơn PT bị hủy → hợp đồng PENDING chuyển CANCELLED (idempotent).
func (s *ptService) onInvoiceVoid(ctx context.Context, inv *entity.Invoice, item *entity.InvoiceItem) error {
	if item.RefID == nil {
		return nil
	}
	c, err := s.repo.FindContract(ctx, *item.RefID, true)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if c.Status != entity.PTContractPending {
		return nil
	}
	if err := s.repo.UpdateContractFields(ctx, c.ID, map[string]any{"status": entity.PTContractCancelled}); err != nil {
		return err
	}
	if _, err := s.cancelScheduled(ctx, c.ID, nil, "Hủy hóa đơn: "+inv.VoidReason); err != nil {
		return err
	}
	s.logAudit(ctx, ptActCancel, ptAuditContract, c.ID, map[string]any{"status": c.Status},
		map[string]any{"status": entity.PTContractCancelled, "invoice_id": inv.ID, "reason": inv.VoidReason})
	return nil
}

// cancelScheduled hủy các buổi SCHEDULED của hợp đồng có start_at > after (nil = mọi buổi). Trả các buổi đã hủy.
func (s *ptService) cancelScheduled(ctx context.Context, contractID uint, after *time.Time, reason string) ([]entity.PTSession, error) {
	sessions, err := s.repo.ScheduledSessions(ctx, contractID, after)
	if err != nil {
		return nil, err
	}
	now := s.now()
	for _, ss := range sessions {
		if err := s.repo.UpdateSessionFields(ctx, ss.ID, map[string]any{
			"status": entity.PTSessionCancelled, "cancelled_at": now, "cancel_reason": invoiceTruncate(reason, 255),
		}); err != nil {
			return nil, err
		}
	}
	return sessions, nil
}

// ---- Truy vấn hợp đồng ----------------------------------------------------------

var ptContractOrderable = []string{"id", "start_date", "end_date", "created_at", "price", "status", "used_sessions"}

func (s *ptService) ListContracts(ctx context.Context, req *v1.ContractListReq) (*v1.ContractListRes, error) {
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, req.BranchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền xem dữ liệu chi nhánh này")
	}
	trainerID := req.TrainerID
	if ptTrainerOnly(ctx) {
		if trainerID != 0 && trainerID != ptSelf(ctx) {
			return nil, errPTOnlyOwnData
		}
		trainerID = ptSelf(ctx)
	}
	q := req.PageReq.ToQuery(ptContractOrderable, "id")
	q.OrderBy = "c." + q.OrderBy
	rows, total, err := s.repo.ListContracts(ctx, q, repository.PTContractFilter{
		BranchIDs: branchIDs, TrainerID: trainerID, MemberID: req.MemberID, PackageID: req.PTPackageID, Status: req.Status,
	})
	if err != nil {
		return nil, err
	}
	today := s.today()
	return dto.NewPageRes(dto.Map(rows, func(r repository.PTContractRow) *v1.Contract { return ptContractRes(r, today) }), total, q), nil
}

func ptContractRes(r repository.PTContractRow, today time.Time) *v1.Contract {
	c := &v1.Contract{
		ID: r.ID, MemberID: r.MemberID, MemberCode: r.MemberCode, MemberName: r.MemberName, MemberPhone: r.MemberPhone,
		TrainerID: r.TrainerID, TrainerName: r.TrainerName, PTPackageID: r.PTPackageID, PackageCode: r.PackageCode, PackageName: r.PackageName,
		BranchID: r.BranchID, BranchName: r.BranchName, InvoiceID: r.InvoiceID, InvoiceNo: r.InvoiceNo, InvoiceStatus: r.InvoiceStatus,
		TotalSessions: r.TotalSessions, UsedSessions: r.UsedSessions, ScheduledSessions: r.ScheduledSessions,
		RemainingSessions: max(0, r.TotalSessions-r.UsedSessions),
		BookableSessions:  max(0, r.TotalSessions-r.UsedSessions-r.ScheduledSessions),
		SessionMinutes:    r.SessionMinutes, StartDate: subFmt(r.StartDate), EndDate: subFmt(r.EndDate),
		Price: r.Price, PricePerSession: ptPerSession(r.Price, r.TotalSessions), Status: r.Status,
		SoldBy: r.SoldBy, SoldByName: r.SoldByName, Note: r.Note, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
	if r.Status == entity.PTContractPending || r.Status == entity.PTContractActive {
		d := subDaysBetween(today, r.EndDate)
		c.DaysLeft = &d
	}
	return c
}

func ptSessionRes(r repository.PTSessionRow) *v1.Session {
	return &v1.Session{
		ID: r.ID, ContractID: r.ContractID, TrainerID: r.TrainerID, TrainerName: r.TrainerName,
		MemberID: r.MemberID, MemberCode: r.MemberCode, MemberName: r.MemberName, MemberPhone: r.MemberPhone,
		BranchID: r.BranchID, BranchName: r.BranchName, PackageName: r.PackageName,
		StartAt: r.StartAt, EndAt: r.EndAt, Status: r.Status, Counted: r.PTSession.Counted(),
		WorkoutNote: r.WorkoutNote, TrainerNote: r.TrainerNote, CompletedAt: r.CompletedAt, CancelledAt: r.CancelledAt,
		CancelReason: r.CancelReason, Source: r.Source, CreatedBy: r.CreatedBy, CreatedByName: r.CreatedByName,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

// detail dựng chi tiết hợp đồng (không kiểm tra quyền).
func (s *ptService) detail(ctx context.Context, id uint) (*v1.ContractDetail, error) {
	row, err := s.repo.FindContractRow(ctx, id)
	if err != nil {
		return nil, ptNotFound(err, errPTContractNotFound)
	}
	out := &v1.ContractDetail{Contract: *ptContractRes(*row, s.today()), Sessions: []*v1.Session{}}
	if pkg, err := s.repo.FindPackage(ctx, row.PTPackageID); err == nil {
		out.Package = ptPackageRef(pkg)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	m, err := s.repo.FindMember(ctx, row.MemberID)
	if err == nil {
		out.Member = v1.MemberRef{ID: m.ID, MemberCode: m.MemberCode, FullName: m.FullName, Phone: m.Phone, Email: m.Email, Status: m.Status}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if row.InvoiceID != nil {
		inv, err := s.invoices.Detail(ctx, *row.InvoiceID)
		if err != nil {
			return nil, err
		}
		out.Invoice = inv
	}
	sessions, err := s.repo.SessionsByContract(ctx, id)
	if err != nil {
		return nil, err
	}
	out.Sessions = dto.Map(sessions, ptSessionRes)
	return out, nil
}

func (s *ptService) GetContract(ctx context.Context, id uint) (*v1.ContractDetail, error) {
	if _, err := s.loadContract(ctx, id, false); err != nil {
		return nil, err
	}
	return s.detail(ctx, id)
}

func (s *ptService) ListByMember(ctx context.Context, memberID uint) ([]*v1.Contract, error) {
	rows, err := s.repo.ListContractsByMember(ctx, memberID, reqctx.AllowedBranches(ctx))
	if err != nil {
		return nil, err
	}
	today, self, own := s.today(), ptSelf(ctx), ptTrainerOnly(ctx)
	out := make([]*v1.Contract, 0, len(rows))
	for _, r := range rows {
		if own && r.TrainerID != self {
			continue
		}
		out = append(out, ptContractRes(r, today))
	}
	return out, nil
}

func (s *ptService) ListMemberContracts(ctx context.Context, memberID uint) ([]*v1.Contract, error) {
	rows, err := s.repo.ListContractsByMember(ctx, memberID, nil)
	if err != nil {
		return nil, err
	}
	today := s.today()
	return dto.Map(rows, func(r repository.PTContractRow) *v1.Contract { return ptContractRes(r, today) }), nil
}

// ---- Đổi HLV --------------------------------------------------------------------

type ptTrainerChange struct {
	contract    *entity.PTContract
	trainerName string
	sessions    []entity.PTSession
	conflicts   []*v1.SessionConflict
}

func (s *ptService) prepareChangeTrainer(ctx context.Context, id uint, in v1.ChangeTrainerInput, lock bool) (*ptTrainerChange, error) {
	c, err := s.loadContract(ctx, id, lock)
	if err != nil {
		return nil, err
	}
	if c.Status != entity.PTContractPending && c.Status != entity.PTContractActive {
		return nil, rescode.New(rescode.BadRequest, "Chỉ đổi HLV cho hợp đồng chờ thanh toán hoặc đang hoạt động")
	}
	if in.TrainerID == c.TrainerID {
		return nil, rescode.New(rescode.BadRequest, "HLV mới phải khác HLV hiện tại")
	}
	ok, err := s.repo.TrainerInBranch(ctx, in.TrainerID, c.BranchID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, rescode.New(rescode.BadRequest, "HLV không thuộc chi nhánh, không phải HLV hoặc đã bị khóa")
	}
	name, err := s.repo.UserName(ctx, in.TrainerID)
	if err != nil {
		return nil, err
	}
	now := s.now()
	sessions, err := s.repo.ScheduledSessions(ctx, c.ID, &now)
	if err != nil {
		return nil, err
	}
	ch := &ptTrainerChange{contract: c, trainerName: name, sessions: sessions, conflicts: []*v1.SessionConflict{}}
	for _, ss := range sessions {
		reason, err := s.trainerConflict(ctx, in.TrainerID, ss.StartAt, ss.EndAt, ss.ID)
		if err != nil {
			return nil, err
		}
		if reason != "" {
			ch.conflicts = append(ch.conflicts, &v1.SessionConflict{SessionID: ss.ID, StartAt: ss.StartAt, EndAt: ss.EndAt, Reason: reason})
		}
	}
	return ch, nil
}

func (s *ptService) CheckChangeTrainer(ctx context.Context, id uint, in v1.ChangeTrainerInput) (*v1.ChangeTrainerCheckRes, error) {
	ch, err := s.prepareChangeTrainer(ctx, id, in, false)
	if err != nil {
		return nil, err
	}
	return &v1.ChangeTrainerCheckRes{OK: len(ch.conflicts) == 0, TrainerID: in.TrainerID, TrainerName: ch.trainerName,
		MovedSessions: len(ch.sessions), Conflicts: ch.conflicts}, nil
}

func ptConflictMessage(conflicts []*v1.SessionConflict) string {
	parts := make([]string, 0, len(conflicts))
	for i, c := range conflicts {
		if i == 5 {
			parts = append(parts, fmt.Sprintf("… và %d buổi khác", len(conflicts)-5))
			break
		}
		parts = append(parts, fmt.Sprintf("buổi #%d lúc %s (%s)", c.SessionID, ptFmtTime(c.StartAt), c.Reason))
	}
	return fmt.Sprintf("Không thể đổi HLV: %d buổi đã đặt bị trùng lịch của HLV mới: %s. Hãy đổi giờ hoặc hủy các buổi này trước",
		len(conflicts), strings.Join(parts, "; "))
}

func (s *ptService) ChangeTrainer(ctx context.Context, id uint, in v1.ChangeTrainerInput) (*v1.ContractDetail, error) {
	var ch *ptTrainerChange
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if ch, err = s.prepareChangeTrainer(ctx, id, in, true); err != nil {
			return err
		}
		if len(ch.conflicts) > 0 {
			return rescode.New(rescode.Conflict, ptConflictMessage(ch.conflicts))
		}
		for _, ss := range ch.sessions {
			if err := s.repo.UpdateSessionFields(ctx, ss.ID, map[string]any{"trainer_id": in.TrainerID}); err != nil {
				if isExclusionViolation(err) {
					return rescode.New(rescode.Conflict, "HLV mới vừa có buổi PT khác trùng giờ, vui lòng thử lại")
				}
				return err
			}
		}
		if err := s.repo.UpdateContractFields(ctx, id, map[string]any{"trainer_id": in.TrainerID}); err != nil {
			return err
		}
		s.logAudit(ctx, ptActChangeTrainer, ptAuditContract, id, map[string]any{"trainer_id": ch.contract.TrainerID},
			map[string]any{"trainer_id": in.TrainerID, "moved_sessions": len(ch.sessions), "note": strings.TrimSpace(in.Note)})
		return nil
	})
	if err != nil {
		return nil, err
	}
	res, err := s.detail(ctx, id)
	if err != nil {
		return nil, err
	}
	var memberUser *uint
	if m, err := s.repo.FindMember(ctx, ch.contract.MemberID); err == nil {
		memberUser = m.UserID
	}
	body := fmt.Sprintf("Hợp đồng PT #%d (%s) chuyển sang HLV %s; %d buổi đã đặt được chuyển theo.", id, res.MemberName, ch.trainerName, len(ch.sessions))
	s.notifyParticipants(ctx, []uint{in.TrainerID, ch.contract.TrainerID}, memberUser, NotifyPTScheduled, "Đổi huấn luyện viên PT", body,
		fmt.Sprintf("/admin/pt-contracts/%d", id))
	return res, nil
}

// ---- Hủy hợp đồng ---------------------------------------------------------------

func (s *ptService) CancelContract(ctx context.Context, id uint, in v1.CancelContractInput) (*v1.ContractDetail, error) {
	reason := strings.TrimSpace(in.Reason)
	if reason == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập lý do hủy")
	}
	var refund *RefundInput
	if in.Refund != nil && (in.Refund.Amount != 0 || in.Refund.Method != "") {
		refund = &RefundInput{Amount: in.Refund.Amount, Method: in.Refund.Method, Reason: "Hủy hợp đồng PT: " + reason}
	}
	if _, err := s.loadContract(ctx, id, false); err != nil {
		return nil, err
	}
	var (
		cancelled []entity.PTSession
		contract  *entity.PTContract
	)
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		c, err := s.loadContract(ctx, id, true)
		if err != nil {
			return err
		}
		contract = c
		if c.Status != entity.PTContractPending && c.Status != entity.PTContractActive {
			return rescode.New(rescode.BadRequest, "Hợp đồng PT đã kết thúc, không thể hủy")
		}
		var inv *entity.Invoice
		if c.InvoiceID != nil {
			if inv, err = s.invoices.FindInvoice(ctx, *c.InvoiceID); err != nil {
				return err
			}
		}
		if refund != nil && inv == nil {
			return rescode.New(rescode.BadRequest, "Hợp đồng PT không có hóa đơn để hoàn tiền")
		}
		if c.Status == entity.PTContractPending && inv != nil && inv.Status == entity.InvoiceStatusUnpaid && inv.PaidAmount == 0 && refund == nil {
			// Chưa thu đồng nào → hủy luôn hóa đơn (trả lượt khuyến mãi); VoidHandler chuyển hợp đồng sang CANCELLED.
			if err := s.invoices.Void(ctx, inv.ID, "Hủy hợp đồng PT: "+reason); err != nil {
				return err
			}
		} else {
			if refund != nil {
				if _, err := s.invoices.Refund(ctx, inv.ID, *refund); err != nil {
					return err
				}
			}
			if err := s.repo.UpdateContractFields(ctx, c.ID, map[string]any{"status": entity.PTContractCancelled}); err != nil {
				return err
			}
		}
		if cancelled, err = s.cancelScheduled(ctx, c.ID, nil, "Hủy hợp đồng PT: "+reason); err != nil {
			return err
		}
		after := map[string]any{"status": entity.PTContractCancelled, "reason": reason, "cancelled_sessions": len(cancelled)}
		if refund != nil {
			after["refund_amount"] = roundMoneyVND(refund.Amount)
			after["refund_method"] = refund.Method
		}
		s.logAudit(ctx, ptActCancel, ptAuditContract, c.ID, map[string]any{"status": c.Status, "used_sessions": c.UsedSessions}, after)
		return nil
	})
	if err != nil {
		return nil, err
	}
	res, err := s.detail(ctx, id)
	if err != nil {
		return nil, err
	}
	if len(cancelled) > 0 {
		var memberUser *uint
		if m, err := s.repo.FindMember(ctx, contract.MemberID); err == nil {
			memberUser = m.UserID
		}
		s.notifyParticipants(ctx, []uint{contract.TrainerID}, memberUser, NotifyPTCancelled, "Hủy hợp đồng PT",
			fmt.Sprintf("Hợp đồng PT #%d đã bị hủy, %d buổi đã đặt bị hủy theo. Lý do: %s", id, len(cancelled), reason),
			fmt.Sprintf("/admin/pt-contracts/%d", id))
	}
	return res, nil
}
