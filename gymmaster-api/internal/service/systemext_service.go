package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/gogf/gf/v2/errors/gerror"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	v1 "gymmaster-api/api/systemext/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/storage/postgres"
)

type SystemExtService struct {
	repo     *repository.SystemExtRepository
	notifier Notifier
	audit    AuditService
	// invoices: dịch vụ hóa đơn DÙNG CHUNG (đã đăng ký Paid/Void/Refund handler của gói tập, PT, POS…).
	invoices InvoiceService
}

// SetInvoiceService gắn dịch vụ hóa đơn dùng chung để hoàn VNPay chạy đủ hook nghiệp vụ
// (vd. hoàn hết HĐ → hủy gói PENDING). Chưa gắn (unit test) → tự tạo bản riêng không có hook.
func (s *SystemExtService) SetInvoiceService(inv InvoiceService) { s.invoices = inv }

const (
	auditBranchTransfer   = "BRANCH_TRANSFER"
	auditBulkMemberPrefix = "BULK_"
)

// NewSystemExtService: audit tùy chọn (nil → không ghi nhật ký).
func NewSystemExtService(repo *repository.SystemExtRepository, notifier Notifier, audit ...AuditService) *SystemExtService {
	return &SystemExtService{repo: repo, notifier: notifier, audit: optionalAudit(audit)}
}

// Nhãn tiếng Việt cho trạng thái hiển thị trong kết quả tìm kiếm nhanh.
var (
	searchMemberStatusLabel = map[string]string{
		entity.MemberStatusActive:   "Đang hoạt động",
		entity.MemberStatusInactive: "Ngừng hoạt động",
		entity.MemberStatusBanned:   "Bị cấm",
	}
	searchInvoiceStatusLabel = map[string]string{
		entity.InvoiceStatusUnpaid:   "Chưa thanh toán",
		entity.InvoiceStatusPartial:  "Thanh toán một phần",
		entity.InvoiceStatusPaid:     "Đã thanh toán",
		entity.InvoiceStatusVoid:     "Đã hủy",
		entity.InvoiceStatusRefunded: "Đã hoàn tiền",
	}
)

// bulkMemberActionLabel: mô tả tiếng Việt của thao tác hàng loạt cho thông báo kết quả (REG-A9a).
var bulkMemberActionLabel = map[string]string{
	"NOTIFY":          "gửi thông báo",
	"TRANSFER_BRANCH": "chuyển chi nhánh",
	"ACTIVATE":        "kích hoạt lại",
	"DEACTIVATE":      "ngừng hoạt động",
}

// newVNPayRefundRef sinh mã tham chiếu hoàn VNPay duy nhất: thêm hậu tố ngẫu nhiên để hai lần hoàn
// trong cùng một giây không trùng mã (FIN-VNP-19).
func newVNPayRefundRef(txnID uint, now time.Time) string {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("VNP-REF-%d-%s-%s", txnID, now.Format("060102150405"), strings.ToUpper(hex.EncodeToString(b[:])))
}

func labelOr(m map[string]string, code string) string {
	if v, ok := m[code]; ok {
		return v
	}
	return code
}

// isStaffCaller: người dùng có ít nhất một vai trò khác MEMBER (tài khoản hội viên không dùng được API quản trị).
func isStaffCaller(ctx context.Context) bool {
	p := reqctx.Auth(ctx)
	if p == nil {
		return false
	}
	for _, r := range p.Roles {
		if r != consts.RoleMember {
			return true
		}
	}
	return false
}

// GlobalSearch (Ctrl+K): chỉ nhân viên; mỗi nhóm kết quả chỉ trả khi người dùng có quyền xem tương ứng,
// hội viên/hóa đơn được lọc theo chi nhánh người dùng được phép truy cập.
func (s *SystemExtService) GlobalSearch(ctx context.Context, q string) (*v1.GlobalSearchRes, error) {
	if !isStaffCaller(ctx) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền thực hiện thao tác này")
	}
	q = strings.TrimSpace(q)
	if q == "" {
		return &v1.GlobalSearchRes{Items: []*v1.SearchHit{}}, nil
	}
	if len([]rune(q)) > 100 {
		q = string([]rune(q)[:100])
	}
	like := "%" + escapeLike(q) + "%"
	hits := make([]*v1.SearchHit, 0, 16)
	branches := reqctx.AllowedBranches(ctx) // nil = toàn hệ thống
	db := s.repo.DB().WithContext(ctx)

	if reqctx.HasPermission(ctx, consts.PermMemberView) {
		var members []entity.Member
		// Tên so khớp không dấu (unaccent: "nguyen van" tìm được "Nguyễn Văn"), thêm email và SĐT đã chuẩn hóa
		// ("0912 345 678", "+84912345678" đều tìm được 0912345678).
		cond := "unaccent(full_name) ILIKE unaccent(?) OR member_code ILIKE ? OR email ILIKE ? OR phone ILIKE ?"
		args := []any{like, like, like, like}
		if digits := searchPhoneDigits(q); digits != "" {
			cond += " OR phone LIKE ?"
			args = append(args, "%"+digits+"%")
		}
		mq := db.Where("("+cond+")", args...)
		if branches != nil {
			mq = mq.Where("home_branch_id IN ?", branches)
		}
		if err := mq.Order("id DESC").Limit(6).Find(&members).Error; err != nil {
			return nil, err
		}
		for _, m := range members {
			hits = append(hits, &v1.SearchHit{
				ID:       m.ID,
				Category: "MEMBER",
				Title:    fmt.Sprintf("%s (%s)", m.FullName, m.MemberCode),
				Subtitle: fmt.Sprintf("SĐT: %s · Trạng thái: %s", m.Phone, labelOr(searchMemberStatusLabel, m.Status)),
				Link:     fmt.Sprintf("/app/members/%d", m.ID),
			})
		}
	}

	if reqctx.HasPermission(ctx, consts.PermInvoiceView) {
		var invoices []entity.Invoice
		iq := db.Where("invoice_no ILIKE ? OR note ILIKE ?", like, like)
		if branches != nil {
			iq = iq.Where("branch_id IN ?", branches)
		}
		if err := iq.Order("id DESC").Limit(5).Find(&invoices).Error; err != nil {
			return nil, err
		}
		for _, inv := range invoices {
			hits = append(hits, &v1.SearchHit{
				ID:       inv.ID,
				Category: "INVOICE",
				Title:    fmt.Sprintf("Hóa đơn %s", inv.InvoiceNo),
				Subtitle: fmt.Sprintf("Tổng: %s · Trạng thái: %s", formatMoneyVND(inv.Total), labelOr(searchInvoiceStatusLabel, inv.Status)),
				Link:     fmt.Sprintf("/app/invoices/%d", inv.ID),
			})
		}
	}

	if reqctx.HasPermission(ctx, consts.PermProductView) {
		var products []entity.Product
		if err := db.Where("unaccent(name) ILIKE unaccent(?) OR sku ILIKE ?", like, like).Limit(5).Find(&products).Error; err != nil {
			return nil, err
		}
		for _, p := range products {
			hits = append(hits, &v1.SearchHit{
				ID:       p.ID,
				Category: "PRODUCT",
				Title:    fmt.Sprintf("%s (%s)", p.Name, p.SKU),
				Subtitle: fmt.Sprintf("Giá bán: %s", formatMoneyVND(p.SellPrice)),
				Link:     "/app/products",
			})
		}
	}

	if reqctx.HasPermission(ctx, consts.PermPlanView) {
		var plans []entity.MembershipPlan
		if err := db.Where("unaccent(name) ILIKE unaccent(?) OR code ILIKE ?", like, like).Limit(4).Find(&plans).Error; err != nil {
			return nil, err
		}
		for _, pl := range plans {
			hits = append(hits, &v1.SearchHit{
				ID:       pl.ID,
				Category: "PLAN",
				Title:    fmt.Sprintf("%s (%s)", pl.Name, pl.Code),
				Subtitle: fmt.Sprintf("Giá gói: %.0fđ", pl.Price),
				Link:     "/app/plans",
			})
		}
	}

	return &v1.GlobalSearchRes{Items: hits}, nil
}

// searchPhoneDigits: nếu từ khóa trông như SĐT (chỉ gồm số, khoảng trắng, dấu chấm/gạch, "+"), trả dãy số đã
// chuẩn hóa đầu 0 (84xxx/+84xxx → 0xxx) để so khớp cột phone; ngược lại trả "".
func searchPhoneDigits(q string) string {
	var b strings.Builder
	for _, r := range q {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '.' || r == '-' || r == '+' || r == '(' || r == ')':
		default:
			return ""
		}
	}
	d := b.String()
	if len(d) < 3 {
		return ""
	}
	if strings.HasPrefix(d, "84") && len(d) >= 11 {
		d = "0" + d[2:]
	}
	return d
}

// escapeLike thoát ký tự đặc biệt của LIKE để từ khóa được so khớp nguyên văn.
func escapeLike(q string) string {
	return strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(q)
}

// scopedMembers tải hội viên theo danh sách id, chỉ trong phạm vi chi nhánh người dùng được phép.
// Báo lỗi nếu có id không tồn tại hoặc ngoài phạm vi (không thao tác một phần).
func (s *SystemExtService) scopedMembers(ctx context.Context, ids []uint) ([]entity.Member, error) {
	uniq := make([]uint, 0, len(ids))
	seen := map[uint]bool{}
	for _, id := range ids {
		if id > 0 && !seen[id] {
			seen[id] = true
			uniq = append(uniq, id)
		}
	}
	if len(uniq) == 0 {
		return nil, rescode.New(rescode.BadRequest, "Chưa chọn danh sách hội viên")
	}
	q := s.repo.DB().WithContext(ctx).Where("id IN ?", uniq)
	if branches := reqctx.AllowedBranches(ctx); branches != nil {
		q = q.Where("home_branch_id IN ?", branches)
	}
	var members []entity.Member
	if err := q.Find(&members).Error; err != nil {
		return nil, err
	}
	if len(members) != len(uniq) {
		return nil, rescode.New(rescode.NotFound, "Có hội viên không tồn tại hoặc không thuộc chi nhánh bạn quản lý")
	}
	return members, nil
}

// ensureBranchActive kiểm tra chi nhánh đích tồn tại và đang hoạt động.
func (s *SystemExtService) ensureBranchActive(ctx context.Context, branchID uint) (*entity.Branch, error) {
	var b entity.Branch
	if err := s.repo.DB().WithContext(ctx).First(&b, branchID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.BadRequest, "Chi nhánh chuyển đến không tồn tại")
		}
		return nil, err
	}
	if b.Status != consts.StatusActive {
		return nil, rescode.New(rescode.BadRequest, "Chi nhánh chuyển đến đang ngừng hoạt động")
	}
	return &b, nil
}

func (s *SystemExtService) BulkMemberAction(ctx context.Context, req *v1.BulkMemberActionReq) (*v1.BulkMemberActionRes, error) {
	if len(req.MemberIDs) == 0 {
		return nil, rescode.New(rescode.BadRequest, "Chưa chọn danh sách hội viên")
	}
	// Kiểm tra tham số trước khi tải dữ liệu.
	switch req.Action {
	case "NOTIFY":
		if strings.TrimSpace(req.Message) == "" {
			return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập nội dung thông báo")
		}
	case "TRANSFER_BRANCH":
		if req.ToBranchID == 0 {
			return nil, rescode.New(rescode.BadRequest, "Vui lòng chọn chi nhánh chuyển đến")
		}
		if _, err := s.ensureBranchActive(ctx, req.ToBranchID); err != nil {
			return nil, err
		}
	}
	members, err := s.scopedMembers(ctx, req.MemberIDs)
	if err != nil {
		return nil, err
	}
	memberIDs := make([]uint, 0, len(members))
	for _, m := range members {
		memberIDs = append(memberIDs, m.ID)
	}

	affected, skipped := 0, 0
	var failed []string
	switch req.Action {
	case "NOTIFY":
		title := strings.TrimSpace(req.Title)
		if title == "" {
			title = "Thông báo từ trung tâm GymMaster"
		}
		msg := strings.TrimSpace(req.Message)
		var userIDs []uint
		for _, m := range members {
			if m.UserID != nil {
				userIDs = append(userIDs, *m.UserID)
			}
			affected++
		}
		if len(userIDs) > 0 && s.notifier != nil {
			_ = s.notifier.Notify(ctx, userIDs, NotifySystem, title, msg, "/me/notifications")
		}
	case "TRANSFER_BRANCH":
		for _, m := range members {
			if _, err := s.CreateBranchTransfer(ctx, &v1.CreateBranchTransferReq{
				MemberID:   m.ID,
				ToBranchID: req.ToBranchID,
				Reason:     req.Message,
			}); err != nil {
				failed = append(failed, m.MemberCode)
				continue
			}
			affected++
		}
	case "ACTIVATE", "DEACTIVATE":
		// ACTIVATE chỉ mở lại hội viên đang INACTIVE và KHÔNG phải hồ sơ đã ẩn danh (yêu cầu xóa dữ liệu đã duyệt);
		// DEACTIVATE chỉ áp cho hội viên đang ACTIVE (không biến BANNED thành INACTIVE). Hội viên bị bỏ qua được báo lại.
		from, to := entity.MemberStatusInactive, entity.MemberStatusActive
		if req.Action == "DEACTIVATE" {
			from, to = entity.MemberStatusActive, entity.MemberStatusInactive
		}
		q := s.repo.DB().WithContext(ctx).Model(&entity.Member{}).Where("id IN ? AND status = ?", memberIDs, from)
		if req.Action == "ACTIVATE" {
			q = q.Where("phone NOT LIKE 'ANON-%'").
				Where("NOT EXISTS (SELECT 1 FROM data_requests dr WHERE dr.member_id = members.id AND dr.type = ? AND dr.status = ?)",
					entity.DataRequestDelete, entity.DataRequestDone)
		}
		res := q.Updates(map[string]any{"status": to, "updated_by": reqctx.UserID(ctx), "updated_at": time.Now()})
		if res.Error != nil {
			return nil, res.Error
		}
		affected = int(res.RowsAffected)
		skipped = len(memberIDs) - affected
	}

	auditLogSafe(s.audit, ctx, auditBulkMemberPrefix+req.Action, auditEntityMember, 0,
		map[string]any{"member_ids": memberIDs},
		map[string]any{"action": req.Action, "affected": affected, "skipped": skipped, "to_branch_id": req.ToBranchID, "title": req.Title, "failed": failed})

	message := fmt.Sprintf("Đã %s cho %d hội viên", labelOr(bulkMemberActionLabel, req.Action), affected)
	if len(failed) > 0 {
		message += fmt.Sprintf("; %d hội viên không chuyển được (%s)", len(failed), strings.Join(failed, ", "))
	}
	if skipped > 0 {
		message += fmt.Sprintf("; bỏ qua %d hội viên không đủ điều kiện (đã ở trạng thái đích, bị cấm hoặc đã ẩn danh)", skipped)
	}
	return &v1.BulkMemberActionRes{Affected: affected, Message: message}, nil
}

func (s *SystemExtService) ListBranchTransfers(ctx context.Context) (v1.ListBranchTransfersRes, error) {
	list, err := s.repo.ListBranchTransfers(ctx)
	if err != nil {
		return nil, err
	}
	res := make(v1.ListBranchTransfersRes, 0, len(list))
	for _, tr := range list {
		// Chỉ hiện lượt chuyển liên quan tới chi nhánh người dùng được phép truy cập.
		if !reqctx.CanAccessBranch(ctx, tr.FromBranchID) && !reqctx.CanAccessBranch(ctx, tr.ToBranchID) {
			continue
		}
		res = append(res, s.enrichTransfer(ctx, tr))
	}
	return res, nil
}

func (s *SystemExtService) CreateBranchTransfer(ctx context.Context, req *v1.CreateBranchTransferReq) (*v1.CreateBranchTransferRes, error) {
	var member entity.Member
	if err := s.repo.DB().WithContext(ctx).First(&member, req.MemberID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy hội viên")
		}
		return nil, err
	}
	// Chỉ chuyển được hội viên thuộc chi nhánh mình quản lý (OWNER: toàn hệ thống).
	if !reqctx.CanAccessBranch(ctx, member.HomeBranchID) {
		return nil, rescode.New(rescode.NotFound, "Không tìm thấy hội viên")
	}
	if member.HomeBranchID == req.ToBranchID {
		return nil, rescode.New(rescode.BadRequest, "Hội viên đã thuộc chi nhánh này")
	}
	if _, err := s.ensureBranchActive(ctx, req.ToBranchID); err != nil {
		return nil, err
	}

	now := time.Now()
	var approvedBy *uint
	if auth := reqctx.Auth(ctx); auth != nil {
		approvedBy = &auth.UserID
	}

	var tr entity.MemberBranchTransfer
	err := s.repo.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var activeSub entity.Subscription
		var subID *uint
		if err := tx.Where("member_id = ? AND status = 'ACTIVE'", member.ID).Order("id DESC").First(&activeSub).Error; err == nil {
			subID = &activeSub.ID
			_ = tx.Model(&entity.Subscription{}).Where("id = ?", activeSub.ID).Update("branch_id", req.ToBranchID).Error
		}
		tr = entity.MemberBranchTransfer{
			MemberID:       member.ID,
			FromBranchID:   member.HomeBranchID,
			ToBranchID:     req.ToBranchID,
			SubscriptionID: subID,
			TransferFee:    req.TransferFee,
			Reason:         strings.TrimSpace(req.Reason),
			Status:         "APPROVED",
			ApprovedBy:     approvedBy,
			ApprovedAt:     &now,
			CreatedAt:      now,
		}
		if err := tx.Create(&tr).Error; err != nil {
			return err
		}
		return tx.Model(&entity.Member{}).Where("id = ?", member.ID).Update("home_branch_id", req.ToBranchID).Error
	})
	if err != nil {
		return nil, err
	}
	item := s.enrichTransfer(ctx, tr)
	auditLogSafe(s.audit, ctx, auditBranchTransfer, auditEntityMember, member.ID,
		map[string]any{"home_branch_id": member.HomeBranchID},
		map[string]any{"home_branch_id": req.ToBranchID, "transfer_id": tr.ID, "subscription_id": tr.SubscriptionID,
			"transfer_fee": tr.TransferFee, "reason": tr.Reason})
	return &v1.CreateBranchTransferRes{BranchTransferItem: item}, nil
}

func (s *SystemExtService) enrichTransfer(ctx context.Context, tr entity.MemberBranchTransfer) *v1.BranchTransferItem {
	var m entity.Member
	var fb, tb entity.Branch
	_ = s.repo.DB().WithContext(ctx).First(&m, tr.MemberID).Error
	_ = s.repo.DB().WithContext(ctx).First(&fb, tr.FromBranchID).Error
	_ = s.repo.DB().WithContext(ctx).First(&tb, tr.ToBranchID).Error
	return &v1.BranchTransferItem{
		ID:             tr.ID,
		MemberID:       tr.MemberID,
		MemberName:     m.FullName,
		MemberCode:     m.MemberCode,
		FromBranchID:   tr.FromBranchID,
		FromBranchName: fb.Name,
		ToBranchID:     tr.ToBranchID,
		ToBranchName:   tb.Name,
		SubscriptionID: tr.SubscriptionID,
		TransferFee:    tr.TransferFee,
		Reason:         tr.Reason,
		Status:         tr.Status,
		ApprovedAt:     tr.ApprovedAt,
		CreatedAt:      tr.CreatedAt,
	}
}

// RefundOnlineTxn hoàn tiền một giao dịch VNPay. Tổng đã hoàn của giao dịch không được vượt số tiền giao dịch
// (vượt → 400, không tự cắt). Phần tiền CHƯA ghi vào hóa đơn (NEEDS_REFUND: thu vượt/hóa đơn đã thu đủ/đã hủy) được
// hoàn trước và không đụng tới hóa đơn/doanh thu; phần đã ghi nhận đi qua logic hoàn tiền chuẩn của hóa đơn
// (hoàn một phần vẫn giữ PAID, hoàn hết → REFUNDED, có phiếu hoàn gắn online_txn_id, audit, đồng bộ trả góp).
// amount = 0 → hoàn toàn bộ số còn có thể hoàn.
func (s *SystemExtService) RefundOnlineTxn(ctx context.Context, req *v1.RefundOnlineTxnReq) (*v1.RefundOnlineTxnRes, error) {
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập lý do hoàn tiền")
	}
	amount := roundMoneyVND(req.Amount)
	if amount < 0 {
		return nil, rescode.New(rescode.BadRequest, "Số tiền hoàn phải lớn hơn 0")
	}
	db := s.repo.DB()
	txm := postgres.NewTxManager(db)
	invSvc, _ := s.invoices.(*invoiceService)
	if invSvc == nil {
		invSvc = NewInvoiceService(repository.NewInvoiceRepository(db), nil, txm, s.audit).(*invoiceService)
	}

	var res *v1.RefundOnlineTxnRes
	err := txm.WithTx(ctx, func(ctx context.Context) error {
		conn := postgres.Conn(ctx, db)
		// Thứ tự khóa giống IPN: giao dịch → hóa đơn.
		var txn entity.OnlinePaymentTxn
		if err := conn.Clauses(clause.Locking{Strength: "UPDATE"}).First(&txn, req.ID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return rescode.New(rescode.NotFound, "Không tìm thấy giao dịch VNPay")
			}
			return err
		}
		var inv entity.Invoice
		if err := conn.First(&inv, txn.InvoiceID).Error; err != nil {
			return err
		}
		if !reqctx.CanAccessBranch(ctx, inv.BranchID) {
			return rescode.New(rescode.PermissionDenied, "Giao dịch thuộc chi nhánh ngoài phạm vi của bạn")
		}
		switch txn.Status {
		case entity.OnlineTxnSuccess, entity.OnlineTxnNeedsRefund:
		case entity.OnlineTxnRefunded:
			return rescode.New(rescode.Conflict, "Giao dịch VNPay đã được hoàn toàn bộ")
		default:
			return rescode.New(rescode.BadRequest, "Chỉ giao dịch VNPay đã nhận tiền (thành công / cần hoàn) mới có thể hoàn tiền")
		}
		refundable := roundMoneyVND(txn.Amount - txn.RefundedAmount)
		if refundable <= 0 {
			return rescode.New(rescode.Conflict, "Giao dịch VNPay đã được hoàn toàn bộ")
		}
		if amount == 0 {
			amount = refundable
		}
		if amount > refundable {
			return rescode.New(rescode.BadRequest, "Số tiền hoàn vượt quá số còn có thể hoàn của giao dịch ("+formatMoneyVND(refundable)+")")
		}
		recorded := txn.Amount
		if txn.RecordedAmount != nil {
			recorded = roundMoneyVND(*txn.RecordedAmount)
		}
		unrecorded := max(0, roundMoneyVND(txn.Amount-recorded))
		unrecordedRefunded := min(txn.RefundedAmount, unrecorded)
		fromUnrecorded := min(amount, roundMoneyVND(unrecorded-unrecordedRefunded))
		fromInvoice := roundMoneyVND(amount - fromUnrecorded)

		now := time.Now()
		gatewayRef := newVNPayRefundRef(txn.ID, now)
		out := &v1.RefundOnlineTxnRes{InvoiceID: txn.InvoiceID, Amount: amount, Status: "SUCCESS", GatewayRef: gatewayRef, CreatedAt: now}
		if fromInvoice > 0 {
			_, rf, err := invSvc.refund(ctx, txn.InvoiceID, RefundInput{Amount: fromInvoice, Method: entity.PaymentVNPay, Reason: reason}, &txn.ID, gatewayRef)
			if err != nil {
				return err
			}
			out.RefundID, out.CreatedAt = rf.ID, rf.CreatedAt
		}
		newRefunded := roundMoneyVND(txn.RefundedAmount + amount)
		status := txn.Status
		switch {
		case newRefunded >= txn.Amount:
			status = entity.OnlineTxnRefunded
		case txn.Status == entity.OnlineTxnNeedsRefund && unrecordedRefunded+fromUnrecorded >= unrecorded:
			status = entity.OnlineTxnSuccess // đã hoàn hết phần chưa ghi nhận
		}
		note := invoiceTruncate(strings.TrimSpace(fmt.Sprintf("%s | Hoàn %s (%s) lúc %s: %s", txn.Note, formatMoneyVND(amount), gatewayRef,
			now.In(subVNLoc).Format("02/01/2006 15:04"), reason)), 255)
		if err := conn.Model(&entity.OnlinePaymentTxn{}).Where("id = ?", txn.ID).Updates(map[string]any{
			"refunded_amount": newRefunded, "status": status, "note": strings.TrimPrefix(note, "| "),
		}).Error; err != nil {
			return err
		}
		auditLogSafe(s.audit, ctx, AuditRefund, "online_payment_txn", txn.ID,
			map[string]any{"status": txn.Status, "refunded_amount": txn.RefundedAmount},
			map[string]any{"status": status, "refunded_amount": newRefunded, "amount": amount, "from_invoice": fromInvoice,
				"from_unrecorded": fromUnrecorded, "refund_id": out.RefundID, "gateway_ref": gatewayRef, "reason": reason})
		res = out
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

// ==================== LEGACY DATA IMPORT (DRY-RUN & COMMIT) ====================

const (
	auditEntityLegacyImport = "legacy_import"
	auditLegacyImport       = "IMPORT"
	legacyImportNote        = "IMPORT"
	legacyImportDebtNote    = "Công nợ chuyển giao từ hệ thống cũ (IMPORT)"

	legacyImportStatusCompleted = "COMPLETED"
	legacyImportStatusFailed    = "FAILED"
)

// ValidateLegacyImportRows kiểm tra dữ liệu từng dòng (không cần DB): họ tên, SĐT Việt Nam hợp lệ (so khớp
// theo SĐT đã chuẩn hóa nên "0912 345 678" và "+84912345678" là trùng), ngày gói tập, công nợ.
func ValidateLegacyImportRows(rows []v1.LegacyImportRow) (int, []v1.LegacyImportErrorItem) {
	seenPhones := make(map[string]int)
	var errs []v1.LegacyImportErrorItem
	okCount := 0

	for idx, r := range rows {
		rowNum := idx + 1
		rawPhone := strings.TrimSpace(r.Phone)
		name := strings.TrimSpace(r.FullName)
		if name == "" {
			errs = append(errs, v1.LegacyImportErrorItem{Row: rowNum, Phone: rawPhone, Message: "Thiếu họ tên hội viên"})
			continue
		}
		phone, ok := NormalizeVNPhone(rawPhone)
		if !ok {
			errs = append(errs, v1.LegacyImportErrorItem{Row: rowNum, Phone: rawPhone, Message: "Số điện thoại không hợp lệ (SĐT Việt Nam 10 số)"})
			continue
		}
		if prevRow, dup := seenPhones[phone]; dup {
			errs = append(errs, v1.LegacyImportErrorItem{Row: rowNum, Phone: rawPhone, Message: fmt.Sprintf("Trùng số điện thoại với dòng %d trong file", prevRow)})
			continue
		}
		seenPhones[phone] = rowNum

		if (r.SubStartDate == "") != (r.SubEndDate == "") {
			errs = append(errs, v1.LegacyImportErrorItem{Row: rowNum, Phone: rawPhone, Message: "Cần nhập đủ ngày bắt đầu và ngày kết thúc gói tập"})
			continue
		}
		if r.SubStartDate != "" && r.SubEndDate != "" {
			st, err1 := time.Parse(consts.DateLayout, r.SubStartDate)
			et, err2 := time.Parse(consts.DateLayout, r.SubEndDate)
			if err1 != nil || err2 != nil || et.Before(st) {
				errs = append(errs, v1.LegacyImportErrorItem{Row: rowNum, Phone: rawPhone, Message: "Ngày bắt đầu/kết thúc gói tập không hợp lệ"})
				continue
			}
		}
		// Có mã gói tập nhưng thiếu ngày → trước đây bị bỏ qua im lặng (A5-01).
		if strings.TrimSpace(r.PlanCode) != "" && r.SubStartDate == "" {
			errs = append(errs, v1.LegacyImportErrorItem{Row: rowNum, Phone: rawPhone, Message: "Có mã gói tập nhưng thiếu ngày bắt đầu/kết thúc gói tập"})
			continue
		}
		if msg := validateLegacyPTFields(r); msg != "" {
			errs = append(errs, v1.LegacyImportErrorItem{Row: rowNum, Phone: rawPhone, Message: msg})
			continue
		}
		if r.RemainingSessions != nil && *r.RemainingSessions < 0 {
			errs = append(errs, v1.LegacyImportErrorItem{Row: rowNum, Phone: rawPhone, Message: "Số buổi còn lại không được âm"})
			continue
		}
		if r.DebtAmount < 0 {
			errs = append(errs, v1.LegacyImportErrorItem{Row: rowNum, Phone: rawPhone, Message: "Công nợ không được âm"})
			continue
		}
		okCount++
	}
	return okCount, errs
}

// validateLegacyPTFields: hợp đồng PT chuyển sang cần đủ mã gói PT, số buổi còn lại (> 0) và email HLV.
func validateLegacyPTFields(r v1.LegacyImportRow) string {
	code := strings.TrimSpace(r.PTPackageCode)
	if code == "" && r.PTRemainingSessions == nil {
		return ""
	}
	if code == "" || r.PTRemainingSessions == nil {
		return "Cần nhập đủ mã gói PT và số buổi PT còn lại"
	}
	if *r.PTRemainingSessions <= 0 {
		return "Số buổi PT còn lại phải lớn hơn 0"
	}
	if strings.TrimSpace(r.PTTrainerEmail) == "" {
		return "Thiếu email HLV phụ trách (pt_trainer_email) cho hợp đồng PT"
	}
	return ""
}

func (s *SystemExtService) ListLegacyImports(ctx context.Context) (v1.ListLegacyImportJobsRes, error) {
	list, err := s.repo.ListLegacyImports(ctx)
	if err != nil {
		return nil, err
	}
	res := make(v1.ListLegacyImportJobsRes, 0, len(list))
	for _, j := range list {
		if !reqctx.CanAccessBranch(ctx, j.BranchID) {
			continue
		}
		res = append(res, s.toImportJobItem(ctx, j))
	}
	return res, nil
}

// legacyRowPlan là kết quả đối chiếu một dòng hợp lệ với DB: hội viên sẵn có (cùng chi nhánh) hoặc tạo mới,
// gói tập dùng để ghi gói đang tập.
type legacyRowPlan struct {
	row      int
	in       v1.LegacyImportRow
	phone    string
	existing *entity.Member
	plan     *entity.MembershipPlan
	ptPkg    *entity.PTPackage
	trainer  uint
}

// planLegacyRows đối chiếu các dòng (đã qua ValidateLegacyImportRows) với DB:
//   - SĐT đã thuộc hội viên ở chi nhánh khác → lỗi (không gắn dữ liệu chéo chi nhánh);
//   - SĐT trùng hội viên cùng chi nhánh → gắn gói tập/công nợ vào hội viên đó;
//   - plan_code không tồn tại/ngừng bán → lỗi; không có plan_code → gói đang bán đầu tiên.
func (s *SystemExtService) planLegacyRows(ctx context.Context, db *gorm.DB, rows []v1.LegacyImportRow, branchID uint) ([]legacyRowPlan, []v1.LegacyImportErrorItem, error) {
	var defaultPlan *entity.MembershipPlan
	plansByCode := map[string]*entity.MembershipPlan{}
	out := make([]legacyRowPlan, 0, len(rows))
	var errs []v1.LegacyImportErrorItem

	for idx, r := range rows {
		rowNum := idx + 1
		phone, _ := NormalizeVNPhone(r.Phone)
		p := legacyRowPlan{row: rowNum, in: r, phone: phone}
		fail := func(msg string) {
			errs = append(errs, v1.LegacyImportErrorItem{Row: rowNum, Phone: strings.TrimSpace(r.Phone), Message: msg})
		}

		var found []entity.Member
		if err := db.WithContext(ctx).Where("phone = ?", phone).Limit(1).Find(&found).Error; err != nil {
			return nil, nil, err
		}
		if len(found) > 0 {
			if found[0].HomeBranchID != branchID {
				fail("Số điện thoại đã thuộc hội viên ở chi nhánh khác, không thể gắn dữ liệu chéo chi nhánh")
				continue
			}
			p.existing = &found[0]
		}

		if r.SubStartDate != "" {
			code := strings.ToUpper(strings.TrimSpace(r.PlanCode))
			if code == "" {
				if defaultPlan == nil {
					var plans []entity.MembershipPlan
					if err := db.WithContext(ctx).Where("status = ?", consts.StatusActive).Order("id ASC").Limit(1).Find(&plans).Error; err != nil {
						return nil, nil, err
					}
					if len(plans) == 0 {
						fail("Chưa có gói tập đang bán để ghi gói tập của hội viên")
						continue
					}
					defaultPlan = &plans[0]
				}
				p.plan = defaultPlan
			} else {
				pl, ok := plansByCode[code]
				if !ok {
					var plans []entity.MembershipPlan
					if err := db.WithContext(ctx).Where("UPPER(code) = ?", code).Limit(1).Find(&plans).Error; err != nil {
						return nil, nil, err
					}
					if len(plans) > 0 {
						pl = &plans[0]
					}
					plansByCode[code] = pl
				}
				if pl == nil {
					fail(fmt.Sprintf("Không tìm thấy gói tập có mã %s", code))
					continue
				}
				p.plan = pl
			}
		}

		if code := strings.ToUpper(strings.TrimSpace(r.PTPackageCode)); code != "" {
			var pkgs []entity.PTPackage
			if err := db.WithContext(ctx).Where("UPPER(code) = ? AND deleted_at IS NULL", code).Limit(1).Find(&pkgs).Error; err != nil {
				return nil, nil, err
			}
			if len(pkgs) == 0 {
				fail(fmt.Sprintf("Không tìm thấy gói PT có mã %s", code))
				continue
			}
			p.ptPkg = &pkgs[0]
			var trainerIDs []uint
			if err := db.WithContext(ctx).Table("users u").Select("u.id").
				Where("LOWER(u.email) = ? AND u.deleted_at IS NULL AND u.status = ?", strings.ToLower(strings.TrimSpace(r.PTTrainerEmail)), consts.StatusActive).
				Where(`(EXISTS (SELECT 1 FROM staff_profiles sp WHERE sp.user_id = u.id AND sp.position = 'TRAINER')
					OR EXISTS (SELECT 1 FROM user_roles ur JOIN roles ro ON ro.id = ur.role_id WHERE ur.user_id = u.id AND ro.code = 'TRAINER'))`).
				Where(`EXISTS (SELECT 1 FROM user_roles ur JOIN roles ro ON ro.id = ur.role_id
					WHERE ur.user_id = u.id AND ro.code <> 'MEMBER' AND (ur.branch_id = ? OR ur.branch_id IS NULL))`, branchID).
				Limit(1).Pluck("u.id", &trainerIDs).Error; err != nil {
				return nil, nil, err
			}
			if len(trainerIDs) == 0 {
				fail(fmt.Sprintf("Không tìm thấy HLV đang hoạt động thuộc chi nhánh có email %s", strings.TrimSpace(r.PTTrainerEmail)))
				continue
			}
			p.trainer = trainerIDs[0]
		}
		out = append(out, p)
	}
	return out, errs, nil
}

// importLegacyRow ghi một dòng (hội viên + gói tập + công nợ) trong transaction tx.
func importLegacyRow(tx *gorm.DB, p legacyRowPlan, branchID uint, actor *uint, now time.Time) error {
	r := p.in
	m := p.existing
	if m == nil {
		gender := strings.ToUpper(strings.TrimSpace(r.Gender))
		if gender != entity.MemberGenderMale && gender != entity.MemberGenderFemale {
			gender = entity.MemberGenderOther
		}
		m = &entity.Member{
			FullName:     strings.TrimSpace(r.FullName),
			Phone:        p.phone,
			Email:        strings.ToLower(strings.TrimSpace(r.Email)),
			Gender:       gender,
			HomeBranchID: branchID,
			Source:       entity.MemberSourceOther,
			Status:       entity.MemberStatusActive,
			Note:         "Chuyển đổi từ hệ thống cũ (IMPORT)",
		}
		m.CreatedBy, m.UpdatedBy = actor, actor
		if err := tx.Create(m).Error; err != nil {
			return mapDuplicate(err, "Số điện thoại đã được dùng cho hội viên khác (có thể đã bị xóa)")
		}
	}

	if p.plan != nil {
		st, _ := time.Parse(consts.DateLayout, r.SubStartDate)
		et, _ := time.Parse(consts.DateLayout, r.SubEndDate)
		// Chạy lại cùng file không nhân đôi gói tập đã nhập.
		var dup int64
		if err := tx.Model(&entity.Subscription{}).
			Where("member_id = ? AND plan_id = ? AND start_date = ? AND end_date = ? AND note = ?", m.ID, p.plan.ID, st, et, legacyImportNote).
			Count(&dup).Error; err != nil {
			return err
		}
		if dup == 0 {
			status := entity.SubStatusActive
			if et.Before(now.Truncate(24 * time.Hour)) {
				status = entity.SubStatusExpired
			}
			sub := entity.Subscription{
				MemberID:          m.ID,
				PlanID:            p.plan.ID,
				BranchID:          branchID,
				StartDate:         st,
				EndDate:           et,
				TotalSessions:     r.RemainingSessions,
				RemainingSessions: r.RemainingSessions,
				Price:             p.plan.Price,
				Status:            status,
				SoldBy:            actor,
				Note:              legacyImportNote,
			}
			if err := tx.Create(&sub).Error; err != nil {
				return err
			}
			// Đánh dấu nguồn để báo cáo phân biệt gói nhập từ hệ thống cũ với gói bán tại quầy (mặc định ADMIN).
			if err := tx.Model(&entity.Subscription{}).Where("id = ?", sub.ID).Update("source", legacyImportNote).Error; err != nil {
				return err
			}
		}
	}

	if p.ptPkg != nil && r.PTRemainingSessions != nil {
		if err := importLegacyPTContract(tx, p, m.ID, branchID, actor, now); err != nil {
			return err
		}
	}

	if r.DebtAmount > 0 {
		var dup int64
		if err := tx.Model(&entity.Invoice{}).
			Where("member_id = ? AND branch_id = ? AND total = ? AND note = ? AND status = ?", m.ID, branchID, r.DebtAmount, legacyImportDebtNote, entity.InvoiceStatusUnpaid).
			Count(&dup).Error; err != nil {
			return err
		}
		if dup == 0 {
			inv := entity.Invoice{
				MemberID: &m.ID,
				BranchID: branchID,
				Subtotal: r.DebtAmount,
				Total:    r.DebtAmount,
				Status:   entity.InvoiceStatusUnpaid,
				Source:   entity.InvoiceSourceCounter,
				Note:     legacyImportDebtNote,
				IssuedBy: actor,
				IssuedAt: now,
			}
			if err := tx.Create(&inv).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// importLegacyPTContract tạo hợp đồng PT ACTIVE với số buổi còn lại (giá 0: đã thu ở hệ thống cũ,
// không tính lại doanh thu), hiệu lực từ hôm nay theo thời hạn gói PT; chạy lại cùng file không nhân đôi.
func importLegacyPTContract(tx *gorm.DB, p legacyRowPlan, memberID, branchID uint, actor *uint, now time.Time) error {
	var dup int64
	if err := tx.Model(&entity.PTContract{}).
		Where("member_id = ? AND pt_package_id = ? AND trainer_id = ? AND note = ? AND status = ?",
			memberID, p.ptPkg.ID, p.trainer, legacyImportNote, entity.PTContractActive).
		Count(&dup).Error; err != nil {
		return err
	}
	if dup > 0 {
		return nil
	}
	start := subDateOf(now)
	c := entity.PTContract{
		MemberID:       memberID,
		TrainerID:      p.trainer,
		PTPackageID:    p.ptPkg.ID,
		BranchID:       branchID,
		TotalSessions:  *p.in.PTRemainingSessions,
		SessionMinutes: ptSessionMinutes(p.ptPkg),
		StartDate:      start,
		EndDate:        subEndDate(start, max(1, p.ptPkg.ValidityDays)),
		Price:          0,
		Status:         entity.PTContractActive,
		SoldBy:         actor,
		Note:           legacyImportNote,
	}
	if err := tx.Create(&c).Error; err != nil {
		return err
	}
	return tx.Model(&entity.PTContract{}).Where("id = ?", c.ID).Update("source", legacyImportNote).Error
}

// errLegacyRowsFailed: có dòng ghi lỗi → hủy toàn bộ transaction (không nhập dở dang; sửa file rồi chạy lại).
var errLegacyRowsFailed = errors.New("legacy import: có dòng lỗi")

// RunLegacyImport: DRY_RUN chỉ kiểm tra (kể cả đối chiếu DB); COMMIT ghi TẤT CẢ hoặc KHÔNG GÌ:
// có lỗi kiểm tra/ghi ở bất kỳ dòng nào → không ghi dòng nào, phiên FAILED kèm lỗi từng dòng.
func (s *SystemExtService) RunLegacyImport(ctx context.Context, req *v1.RunLegacyImportReq) (*v1.RunLegacyImportRes, error) {
	mode := req.Mode
	if mode == "" {
		mode = entity.ImportModeDryRun
	}
	if mode != entity.ImportModeDryRun && mode != entity.ImportModeCommit {
		return nil, rescode.New(rescode.BadRequest, "Chế độ nhập phải là DRY_RUN hoặc COMMIT")
	}
	if req.BranchID == 0 || !reqctx.CanAccessBranch(ctx, req.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền nhập dữ liệu cho chi nhánh này")
	}
	if len(req.Rows) == 0 {
		return nil, rescode.New(rescode.BadRequest, "File không có dòng dữ liệu nào")
	}
	db := s.repo.DB()
	var branch entity.Branch
	if err := db.WithContext(ctx).First(&branch, req.BranchID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.BadRequest, "Chi nhánh không tồn tại")
		}
		return nil, err
	}
	if branch.Status != consts.StatusActive {
		return nil, rescode.New(rescode.BadRequest, "Chi nhánh đang ngừng hoạt động")
	}

	_, rowErrs := ValidateLegacyImportRows(req.Rows)
	invalid := map[int]bool{}
	for _, e := range rowErrs {
		invalid[e.Row] = true
	}
	plans, planErrs, err := s.planLegacyRows(ctx, db, req.Rows, req.BranchID)
	if err != nil {
		return nil, err
	}
	for _, e := range planErrs {
		if !invalid[e.Row] {
			invalid[e.Row] = true
			rowErrs = append(rowErrs, e)
		}
	}
	actor := reqctx.UserID(ctx)
	now := time.Now()
	status := legacyImportStatusCompleted
	written := 0

	if mode == entity.ImportModeCommit {
		if len(rowErrs) > 0 {
			status = legacyImportStatusFailed
		} else {
			txErr := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
				for _, p := range plans {
					// Mỗi dòng một SAVEPOINT: lỗi một dòng không làm hỏng transaction, nhờ đó gom đủ lỗi mọi dòng.
					if err := tx.Transaction(func(rtx *gorm.DB) error {
						return importLegacyRow(rtx, p, req.BranchID, actor, now)
					}); err != nil {
						rowErrs = append(rowErrs, v1.LegacyImportErrorItem{Row: p.row, Phone: strings.TrimSpace(p.in.Phone), Message: legacyRowErrMessage(err)})
						continue
					}
					written++
				}
				if len(rowErrs) > 0 {
					return errLegacyRowsFailed
				}
				return nil
			})
			if txErr != nil {
				status, written = legacyImportStatusFailed, 0
				if !errors.Is(txErr, errLegacyRowsFailed) && len(rowErrs) == 0 {
					return nil, txErr
				}
			}
		}
	}
	sortLegacyErrors(rowErrs)

	okRows := len(req.Rows) - len(rowErrs)
	if mode == entity.ImportModeCommit {
		okRows = written
	}
	if rowErrs == nil {
		rowErrs = []v1.LegacyImportErrorItem{}
	}
	errBytes, _ := json.Marshal(rowErrs)
	summary, _ := json.Marshal(map[string]any{"mode": mode, "ok": okRows, "err": len(rowErrs), "written": written, "status": status})
	fileName := strings.TrimSpace(req.FileName)
	if fileName == "" {
		fileName = "legacy-import.xlsx"
	}
	job := &entity.LegacyImportJob{
		BranchID:    req.BranchID,
		FileName:    fileName,
		Mode:        mode,
		Status:      status,
		TotalRows:   len(req.Rows),
		OkRows:      okRows,
		ErrorRows:   len(rowErrs),
		SummaryJSON: datatypes.JSON(summary),
		ErrorsJSON:  datatypes.JSON(errBytes),
		CreatedBy:   actor,
		CreatedAt:   now,
	}
	if err := s.repo.CreateLegacyImportJob(ctx, job); err != nil {
		return nil, err
	}
	if mode == entity.ImportModeCommit {
		auditLogSafe(s.audit, ctx, auditLegacyImport, auditEntityLegacyImport, job.ID, nil, map[string]any{
			"branch_id": req.BranchID, "file_name": fileName, "status": status,
			"total_rows": job.TotalRows, "written_rows": written, "error_rows": job.ErrorRows,
		})
	}
	return &v1.RunLegacyImportRes{LegacyImportJobItem: s.toImportJobItem(ctx, *job)}, nil
}

// legacyRowErrMessage: lỗi nghiệp vụ (rescode) hiển thị nguyên văn; lỗi DB khác chỉ báo chung (không lộ SQL).
func legacyRowErrMessage(err error) string {
	if c := gerror.Code(err).Code(); c >= rescode.BadRequest.Code() && c < rescode.Internal.Code() {
		return err.Error()
	}
	return "Không ghi được dòng này vào cơ sở dữ liệu, vui lòng kiểm tra lại dữ liệu"
}

func sortLegacyErrors(errs []v1.LegacyImportErrorItem) {
	slices.SortStableFunc(errs, func(a, b v1.LegacyImportErrorItem) int { return a.Row - b.Row })
}

func (s *SystemExtService) toImportJobItem(ctx context.Context, j entity.LegacyImportJob) *v1.LegacyImportJobItem {
	var errs []v1.LegacyImportErrorItem
	_ = json.Unmarshal(j.ErrorsJSON, &errs)
	if errs == nil {
		errs = []v1.LegacyImportErrorItem{}
	}
	branchName := ""
	if s.repo != nil && s.repo.DB() != nil {
		var b entity.Branch
		if err := s.repo.DB().WithContext(ctx).First(&b, j.BranchID).Error; err == nil {
			branchName = b.Name
		}
	}
	return &v1.LegacyImportJobItem{
		ID:         j.ID,
		BranchID:   j.BranchID,
		BranchName: branchName,
		FileName:   j.FileName,
		Mode:       j.Mode,
		Status:     j.Status,
		TotalRows:  j.TotalRows,
		OkRows:     j.OkRows,
		ErrorRows:  j.ErrorRows,
		Errors:     errs,
		CreatedAt:  j.CreatedAt,
	}
}

// ==================== SETUP WIZARD (SPRINT 30 / MỤC 15.2) ====================

func (s *SystemExtService) GetSetupWizardStatus(ctx context.Context) (*v1.GetSetupWizardStatusRes, error) {
	db := s.repo.DB().WithContext(ctx)
	var branchCount, roomCount, planCount, staffCount, settingCount int64
	_ = db.Model(&entity.Branch{}).Count(&branchCount).Error
	_ = db.Model(&entity.Room{}).Count(&roomCount).Error
	_ = db.Model(&entity.MembershipPlan{}).Count(&planCount).Error
	_ = db.Model(&entity.User{}).Count(&staffCount).Error
	_ = db.Model(&entity.Setting{}).Count(&settingCount).Error

	var wizardSkipped entity.Setting
	skipped := db.Where("key = ?", "setup_wizard_skipped").First(&wizardSkipped).Error == nil

	steps := []*v1.SetupWizardStepStatus{
		{Key: "CENTER_INFO", Title: "1. Thông tin trung tâm Fitness", Completed: settingCount > 0, Count: settingCount, Route: "/app/settings"},
		{Key: "BRANCHES", Title: "2. Khởi tạo chi nhánh hoạt động", Completed: branchCount > 0, Count: branchCount, Route: "/app/branches"},
		{Key: "ROOMS", Title: "3. Thiết lập phòng tập & khu vực", Completed: roomCount > 0, Count: roomCount, Route: "/app/rooms"},
		{Key: "PLANS", Title: "4. Cấu hình gói tập & gói PT mẫu", Completed: planCount > 0, Count: planCount, Route: "/app/plans"},
		{Key: "STAFF", Title: "5. Tài khoản nhân sự & Huấn luyện viên", Completed: staffCount > 1, Count: staffCount, Route: "/app/staff"},
		{Key: "RULES", Title: "6. Quy định vận hành (Giờ mở cửa & Hủy lớp)", Completed: settingCount > 0, Count: settingCount, Route: "/app/settings"},
	}
	allDone := true
	for _, st := range steps {
		if !st.Completed {
			allDone = false
			break
		}
	}
	return &v1.GetSetupWizardStatusRes{
		Completed: allDone,
		Skipped:   skipped,
		Steps:     steps,
	}, nil
}

// wizardCancelBeforeMinutes đổi "hủy lớp trước N giờ" của Wizard sang phút cho booking.cancel_before_minutes
// (giới hạn như trang Cấu hình: tối đa 10080 phút = 168 giờ). 0 = không đổi cấu hình.
func wizardCancelBeforeMinutes(hours int) (int, error) {
	if hours < 0 || hours*60 > settingRules[SettingBookingCancelBeforeMinutes].max {
		return 0, rescode.New(rescode.BadRequest, "Thời hạn hủy lớp phải từ 1 đến 168 giờ")
	}
	return hours * 60, nil
}

func (s *SystemExtService) CompleteSetupWizard(ctx context.Context, req *v1.CompleteSetupWizardReq) (*v1.CompleteSetupWizardRes, error) {
	cancelMinutes, err := wizardCancelBeforeMinutes(req.CancelClassBefore)
	if err != nil {
		return nil, err
	}
	db := s.repo.DB().WithContext(ctx)
	saveRaw := func(k string, raw []byte) {
		st := entity.Setting{Key: k, Value: datatypes.JSON(raw), UpdatedAt: time.Now()}
		_ = db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
		}).Create(&st).Error
	}
	saveKV := func(k, v string) {
		if strings.TrimSpace(v) == "" {
			return
		}
		raw, _ := json.Marshal(v)
		saveRaw(k, raw)
	}
	saveKV("center_name", req.CenterName)
	saveKV("center_hotline", req.Hotline)
	saveKV("opening_hours", req.OpeningHours)
	if cancelMinutes > 0 {
		// Lưu đúng khóa mà đặt lớp đọc (số phút, kiểu số JSON) – SW-02.
		saveRaw(SettingBookingCancelBeforeMinutes, []byte(fmt.Sprintf("%d", cancelMinutes)))
	}
	if req.Skip {
		saveKV("setup_wizard_skipped", "true")
	}
	status, err := s.GetSetupWizardStatus(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.CompleteSetupWizardRes{GetSetupWizardStatusRes: status}, nil
}
