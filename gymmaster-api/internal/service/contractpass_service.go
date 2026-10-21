package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"html"
	"net/url"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"gorm.io/gorm"

	v1 "gymmaster-api/api/contractpass/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/clientip"
	"gymmaster-api/internal/pkg/htmlsafe"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/storage/postgres"
)

const (
	auditEntityContractTemplate = "contract_template"
	auditEntityMemberContract   = "member_contract"
	auditEntityDayPass          = "day_pass"
)

var (
	contractTemplateOrderable = []string{"id", "code", "name", "version", "created_at"}
	memberContractOrderable   = []string{"id", "status", "signed_at", "created_at"}
	dayPassOrderable          = []string{"id", "buyer_name", "valid_date", "price", "created_at"}
)

type (
	ContractPassService interface {
		ListTemplates(ctx context.Context, req *v1.ListContractTemplatesReq) (*v1.ListContractTemplatesRes, error)
		CreateTemplate(ctx context.Context, in v1.ContractTemplateInput) (*v1.ContractTemplateItem, error)
		UpdateTemplate(ctx context.Context, id uint, in v1.ContractTemplateInput) (*v1.ContractTemplateItem, error)
		DeleteTemplate(ctx context.Context, id uint) error

		ListMemberContracts(ctx context.Context, req *v1.ListMemberContractsReq) (*v1.ListMemberContractsRes, error)
		CreateMemberContract(ctx context.Context, req *v1.CreateMemberContractReq) (*v1.MemberContractItem, error)

		ListDayPasses(ctx context.Context, req *v1.ListDayPassesReq) (*v1.ListDayPassesRes, error)
		CreateDayPass(ctx context.Context, req *v1.CreateDayPassReq) (*v1.DayPassItem, error)

		MemberContracts(ctx context.Context) (v1.MemberContractsRes, error)
		SignMemberContract(ctx context.Context, req *v1.SignMemberContractReq) (*v1.MemberContractItem, error)
		MemberGuestPasses(ctx context.Context) (*v1.MemberGuestPassesRes, error)
		CreateGuestPass(ctx context.Context, req *v1.CreateGuestPassReq) (*v1.DayPassItem, error)
	}

	contractPassService struct {
		repo     repository.ContractPassRepository
		tx       postgres.TxManager
		audit    AuditService
		invoices InvoiceService
		now      func() time.Time
	}
)

// NewContractPassService: invoices (tùy chọn) là InvoiceService dùng chung – vé ngày có giá được lập hóa đơn
// + ghi nhận thu tiền để vào doanh thu.
func NewContractPassService(repo repository.ContractPassRepository, tx postgres.TxManager, audit AuditService, invoices ...InvoiceService) ContractPassService {
	s := &contractPassService{repo: repo, tx: tx, audit: audit, now: time.Now}
	if len(invoices) > 0 {
		s.invoices = invoices[0]
	}
	return s
}

func (s *contractPassService) withinTx(ctx context.Context, fn func(context.Context) error) error {
	if s.tx == nil {
		return fn(ctx)
	}
	return s.tx.WithTx(ctx, fn)
}

// passToday: hôm nay (giờ VN, 00:00).
func (s *contractPassService) passToday() time.Time {
	t := s.now().In(memberVNLoc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, memberVNLoc)
}

// passValidDate đọc ngày hiệu lực (rỗng = hôm nay), không cho phép ngày quá khứ.
func (s *contractPassService) passValidDate(raw string) (time.Time, error) {
	today := s.passToday()
	if strings.TrimSpace(raw) == "" {
		return today, nil
	}
	parsed, err := time.ParseInLocation(consts.DateLayout, strings.TrimSpace(raw), memberVNLoc)
	if err != nil {
		return time.Time{}, rescode.New(rescode.BadRequest, "Ngày hiệu lực phải có định dạng YYYY-MM-DD")
	}
	if parsed.Before(today) {
		return time.Time{}, rescode.New(rescode.BadRequest, "Ngày hiệu lực không được ở quá khứ")
	}
	return parsed, nil
}

// activeBranch kiểm tra chi nhánh tồn tại và đang hoạt động.
func (s *contractPassService) activeBranch(ctx context.Context, id uint) (*entity.Branch, error) {
	b, err := s.repo.FindBranch(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.BadRequest, "Chi nhánh không tồn tại")
		}
		return nil, err
	}
	if b.Status != consts.StatusActive {
		return nil, rescode.New(rescode.BadRequest, "Chi nhánh đang ngừng hoạt động")
	}
	return b, nil
}

func (s *contractPassService) currentMember(ctx context.Context) (*entity.Member, error) {
	uid := reqctx.UserID(ctx)
	if uid == nil {
		return nil, rescode.New(rescode.Unauthorized, "Bạn chưa đăng nhập")
	}
	m, err := s.repo.FindMemberByUserID(ctx, *uid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Tài khoản chưa được liên kết với hội viên")
		}
		return nil, err
	}
	return m, nil
}

// renderContractHTML: lọc mẫu theo allowlist rồi thay biến – mọi giá trị thay vào đều được HTML-escape
// (họ tên/SĐT do hội viên tự nhập → chống stored XSS khi nhân viên xem hợp đồng).
func renderContractHTML(bodyHTML string, m *entity.Member, planName string) string {
	if strings.TrimSpace(planName) == "" {
		planName = "Gói tập tiêu chuẩn GymMaster"
	}
	replacer := strings.NewReplacer(
		"{{member_name}}", html.EscapeString(m.FullName),
		"{{member_code}}", html.EscapeString(m.MemberCode),
		"{{member_phone}}", html.EscapeString(m.Phone),
		"{{plan_name}}", html.EscapeString(planName),
	)
	return replacer.Replace(htmlsafe.Sanitize(bodyHTML))
}

// signatureMaxBytes: dung lượng tối đa của ảnh chữ ký PNG (sau khi giải mã base64).
const signatureMaxBytes = 300 * 1024

var pngMagic = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}

// validateSignature chỉ nhận ảnh PNG dạng data URL (data:image/png;base64,..., ≤ 300KB, đúng chữ ký PNG)
// hoặc đường dẫn ảnh đã tải lên của hệ thống (/uploads/...). Mọi giá trị khác (javascript:, URL ngoài,
// chuỗi có dấu nháy...) bị từ chối để không thể chèn vào thuộc tính src khi hiển thị.
func validateSignature(sig string) error {
	const prefix = "data:image/png;base64,"
	if strings.HasPrefix(sig, prefix) {
		raw := sig[len(prefix):]
		if base64.StdEncoding.DecodedLen(len(raw)) > signatureMaxBytes+3 {
			return rescode.New(rescode.BadRequest, "Ảnh chữ ký quá lớn (tối đa 300KB)")
		}
		data, err := base64.StdEncoding.DecodeString(raw)
		if err != nil || len(data) < len(pngMagic) || string(data[:len(pngMagic)]) != string(pngMagic) {
			return rescode.New(rescode.BadRequest, "Ảnh chữ ký không hợp lệ")
		}
		if len(data) > signatureMaxBytes {
			return rescode.New(rescode.BadRequest, "Ảnh chữ ký quá lớn (tối đa 300KB)")
		}
		return nil
	}
	if strings.HasPrefix(sig, "/uploads/") && len(sig) <= 500 {
		u, err := url.Parse(sig)
		decoded, decErr := url.PathUnescape(sig)
		if err == nil && decErr == nil && u.Scheme == "" && u.Host == "" && u.RawQuery == "" && u.Fragment == "" &&
			strings.HasPrefix(decoded, "/uploads/") &&
			!strings.Contains(sig, "..") && !strings.Contains(decoded, "..") &&
			!strings.ContainsAny(sig, "\"'<> \t\r\n\\") && !strings.ContainsAny(decoded, "\"'<> \t\r\n\\") {
			return nil
		}
	}
	return rescode.New(rescode.BadRequest, "Chữ ký phải là ảnh PNG (data:image/png;base64) hoặc ảnh đã tải lên")
}

// sanitizeTemplateBody lọc nội dung mẫu hợp đồng theo allowlist trước khi lưu.
func sanitizeTemplateBody(raw string) (string, error) {
	body := strings.TrimSpace(htmlsafe.Sanitize(strings.TrimSpace(raw)))
	if body == "" {
		return "", rescode.New(rescode.BadRequest, "Vui lòng nhập đầy đủ mã, tên và nội dung mẫu hợp đồng")
	}
	return body, nil
}

func generateDayPassToken() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return "DP-" + strings.ToUpper(hex.EncodeToString(b[:]))
}

// ---- Admin: Contract Templates --------------------------------------------------

func (s *contractPassService) ListTemplates(ctx context.Context, req *v1.ListContractTemplatesReq) (*v1.ListContractTemplatesRes, error) {
	q := req.PageReq.ToQuery(contractTemplateOrderable, "id")
	items, total, err := s.repo.ListTemplates(ctx, q, strings.TrimSpace(req.Status))
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(items, toContractTemplateItem), total, q), nil
}

func (s *contractPassService) CreateTemplate(ctx context.Context, in v1.ContractTemplateInput) (*v1.ContractTemplateItem, error) {
	code := strings.ToUpper(strings.TrimSpace(in.Code))
	name := strings.TrimSpace(in.Name)
	body := strings.TrimSpace(in.BodyHTML)
	if code == "" || name == "" || body == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập đầy đủ mã, tên và nội dung mẫu hợp đồng")
	}
	body, err := sanitizeTemplateBody(body)
	if err != nil {
		return nil, err
	}
	dup, err := s.repo.ExistsTemplateCode(ctx, code, 0)
	if err != nil {
		return nil, err
	}
	if dup {
		return nil, rescode.New(rescode.Conflict, "Mã mẫu hợp đồng đã tồn tại")
	}
	ver := in.Version
	if ver <= 0 {
		ver = 1
	}
	status := strings.ToUpper(strings.TrimSpace(in.Status))
	if status == "" {
		status = consts.StatusActive
	}
	tpl := &entity.ContractTemplate{
		Code:     code,
		Name:     name,
		BodyHTML: body,
		Version:  ver,
		Status:   status,
	}
	if err := s.repo.CreateTemplate(ctx, tpl); err != nil {
		return nil, mapDuplicate(err, "Mã mẫu hợp đồng đã tồn tại")
	}
	res := toContractTemplateItem(*tpl)
	if s.audit != nil {
		s.audit.Log(ctx, AuditCreate, auditEntityContractTemplate, tpl.ID, nil, res)
	}
	return res, nil
}

func (s *contractPassService) UpdateTemplate(ctx context.Context, id uint, in v1.ContractTemplateInput) (*v1.ContractTemplateItem, error) {
	tpl, err := s.repo.FindTemplateByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy mẫu hợp đồng")
		}
		return nil, err
	}
	code := strings.ToUpper(strings.TrimSpace(in.Code))
	name := strings.TrimSpace(in.Name)
	body := strings.TrimSpace(in.BodyHTML)
	if code == "" || name == "" || body == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập đầy đủ mã, tên và nội dung mẫu hợp đồng")
	}
	body, err = sanitizeTemplateBody(body)
	if err != nil {
		return nil, err
	}
	dup, err := s.repo.ExistsTemplateCode(ctx, code, id)
	if err != nil {
		return nil, err
	}
	if dup {
		return nil, rescode.New(rescode.Conflict, "Mã mẫu hợp đồng đã tồn tại")
	}
	before := toContractTemplateItem(*tpl)
	tpl.Code = code
	tpl.Name = name
	tpl.BodyHTML = body
	if in.Version > 0 {
		tpl.Version = in.Version
	} else {
		tpl.Version++
	}
	if strings.TrimSpace(in.Status) != "" {
		tpl.Status = strings.ToUpper(strings.TrimSpace(in.Status))
	}
	if err := s.repo.UpdateTemplate(ctx, tpl); err != nil {
		return nil, mapDuplicate(err, "Mã mẫu hợp đồng đã tồn tại")
	}
	res := toContractTemplateItem(*tpl)
	if s.audit != nil {
		s.audit.Log(ctx, AuditUpdate, auditEntityContractTemplate, tpl.ID, before, res)
	}
	return res, nil
}

func (s *contractPassService) DeleteTemplate(ctx context.Context, id uint) error {
	tpl, err := s.repo.FindTemplateByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return rescode.New(rescode.NotFound, "Không tìm thấy mẫu hợp đồng")
		}
		return err
	}
	if err := s.repo.DeleteTemplate(ctx, id); err != nil {
		// Mẫu đã được dùng cho hợp đồng hội viên (FK ON DELETE RESTRICT) → 409 thay vì 500.
		if errors.Is(err, gorm.ErrForeignKeyViolated) {
			return rescode.New(rescode.Conflict, "Mẫu hợp đồng đã được dùng cho hợp đồng hội viên – hãy chuyển sang Ngừng hoạt động thay vì xóa")
		}
		return err
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditDelete, auditEntityContractTemplate, id, toContractTemplateItem(*tpl), nil)
	}
	return nil
}

// ---- Admin: Member Contracts ----------------------------------------------------

func (s *contractPassService) ListMemberContracts(ctx context.Context, req *v1.ListMemberContractsReq) (*v1.ListMemberContractsRes, error) {
	q := req.PageReq.ToQuery(memberContractOrderable, "id")
	q.OrderBy = "mc." + q.OrderBy
	rows, total, err := s.repo.ListMemberContracts(ctx, q, req.MemberID, strings.TrimSpace(req.Status))
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(rows, toMemberContractItem), total, q), nil
}

func (s *contractPassService) CreateMemberContract(ctx context.Context, req *v1.CreateMemberContractReq) (*v1.MemberContractItem, error) {
	m, err := s.repo.FindMemberByID(ctx, req.MemberID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy hội viên")
		}
		return nil, err
	}
	var tpl *entity.ContractTemplate
	if req.TemplateID > 0 {
		tpl, err = s.repo.FindTemplateByID(ctx, req.TemplateID)
	} else {
		tpl, err = s.repo.FindFirstActiveTemplate(ctx)
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy mẫu hợp đồng điện tử")
		}
		return nil, err
	}
	planName := ""
	subID := req.SubscriptionID
	if plan, activeSubID, err := s.repo.FindActiveMemberPlan(ctx, m.ID); err == nil && plan != nil {
		planName = plan.Name
		if subID == nil {
			subID = activeSubID
		}
	}
	rendered := renderContractHTML(tpl.BodyHTML, m, planName)
	mc := &entity.MemberContract{
		MemberID:        m.ID,
		TemplateID:      tpl.ID,
		TemplateVersion: tpl.Version,
		SubscriptionID:  subID,
		RenderedHTML:    rendered,
		Status:          entity.ContractStatusPending,
	}
	if err := s.repo.CreateMemberContract(ctx, mc); err != nil {
		return nil, err
	}
	row, err := s.repo.FindMemberContractByID(ctx, mc.ID)
	if err != nil {
		row = &repository.MemberContractRow{
			MemberContract: *mc,
			MemberCode:     m.MemberCode,
			MemberName:     m.FullName,
			MemberPhone:    m.Phone,
			TemplateCode:   tpl.Code,
			TemplateName:   tpl.Name,
		}
	}
	res := toMemberContractItem(*row)
	if s.audit != nil {
		s.audit.Log(ctx, AuditCreate, auditEntityMemberContract, mc.ID, nil, res)
	}
	return res, nil
}

// ---- Admin: Day Passes ----------------------------------------------------------

func (s *contractPassService) ListDayPasses(ctx context.Context, req *v1.ListDayPassesReq) (*v1.ListDayPassesRes, error) {
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, req.BranchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được xem chi nhánh này")
	}
	q := req.PageReq.ToQuery(dayPassOrderable, "id")
	q.OrderBy = "dp." + q.OrderBy
	rows, total, err := s.repo.ListDayPasses(ctx, q, branchIDs, strings.TrimSpace(req.Type))
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(rows, toDayPassItem), total, q), nil
}

func (s *contractPassService) CreateDayPass(ctx context.Context, req *v1.CreateDayPassReq) (*v1.DayPassItem, error) {
	buyer := strings.TrimSpace(req.BuyerName)
	if req.BranchID == 0 || buyer == "" || strings.TrimSpace(req.Phone) == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập đầy đủ chi nhánh, họ tên và số điện thoại")
	}
	if !reqctx.CanAccessBranch(ctx, req.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền bán vé ngày tại chi nhánh này")
	}
	if _, err := s.activeBranch(ctx, req.BranchID); err != nil {
		return nil, err
	}
	phone, ok := NormalizeVNPhone(req.Phone)
	if !ok {
		return nil, rescode.New(rescode.BadRequest, "Số điện thoại không hợp lệ")
	}
	price := roundMoneyVND(req.Price)
	if price < 0 {
		return nil, rescode.New(rescode.BadRequest, "Giá vé không được âm")
	}
	method := strings.ToUpper(strings.TrimSpace(req.PaymentMethod))
	if method == "" {
		method = entity.PaymentCash
	}
	if price > 0 {
		if err := invoiceValidMethod(method); err != nil {
			return nil, err
		}
		if method == entity.PaymentVNPay {
			return nil, rescode.New(rescode.BadRequest, "Phương thức thanh toán không hợp lệ")
		}
	}
	validDate, err := s.passValidDate(req.ValidDate)
	if err != nil {
		return nil, err
	}
	if err := checkPassDayOpen(ctx, req.BranchID, validDate, "bán vé ngày"); err != nil {
		return nil, err
	}
	dp := &entity.DayPass{
		BranchID:  req.BranchID,
		BuyerName: buyer,
		Phone:     phone,
		Type:      entity.DayPassTypeDay,
		ValidDate: validDate,
		QRToken:   generateDayPassToken(),
		Price:     price,
		CreatedBy: invoiceActorID(ctx),
	}
	err = s.withinTx(ctx, func(ctx context.Context) error {
		// Vé có giá → lập hóa đơn bán lẻ (POS) + ghi nhận thu tiền để vào doanh thu.
		if price > 0 && s.invoices != nil {
			inv, err := s.invoices.CreateInvoice(ctx, InvoiceDraft{
				BranchID: req.BranchID, Source: entity.InvoiceSourcePOS,
				Note: "Bán vé tập theo ngày cho " + buyer + " (" + phone + ")",
				Items: []InvoiceItemDraft{{
					ItemType: entity.InvoiceItemOther, Quantity: 1, UnitPrice: price,
					Description: invoiceTruncate("Vé tập theo ngày (Day Pass) – "+buyer+" – "+invoiceDescDate(validDate), 255),
				}},
			})
			if err != nil {
				return err
			}
			if _, err := s.invoices.AddPayment(ctx, inv.ID, PaymentInput{Amount: price, Method: method, Note: "Thu tiền vé ngày"}); err != nil {
				return err
			}
			dp.InvoiceID = &inv.ID
		}
		return s.repo.CreateDayPass(ctx, dp)
	})
	if err != nil {
		return nil, err
	}
	row, err := s.repo.FindDayPassByID(ctx, dp.ID)
	if err != nil {
		row = &repository.DayPassRow{DayPass: *dp}
	}
	res := toDayPassItem(*row)
	if s.audit != nil {
		s.audit.Log(ctx, AuditCreate, auditEntityDayPass, dp.ID, nil, res)
	}
	return res, nil
}

// ---- Member: Contracts & Guest Passes -------------------------------------------

func (s *contractPassService) MemberContracts(ctx context.Context) (v1.MemberContractsRes, error) {
	m, err := s.currentMember(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.ListContractsByMember(ctx, m.ID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		if tpl, err := s.repo.FindFirstActiveTemplate(ctx); err == nil && tpl != nil {
			planName := ""
			var subID *uint
			if plan, activeSubID, err := s.repo.FindActiveMemberPlan(ctx, m.ID); err == nil && plan != nil {
				planName = plan.Name
				subID = activeSubID
			}
			mc := &entity.MemberContract{
				MemberID:        m.ID,
				TemplateID:      tpl.ID,
				TemplateVersion: tpl.Version,
				SubscriptionID:  subID,
				RenderedHTML:    renderContractHTML(tpl.BodyHTML, m, planName),
				Status:          entity.ContractStatusPending,
			}
			if err := s.repo.CreateMemberContract(ctx, mc); err == nil {
				rows = []repository.MemberContractRow{
					{
						MemberContract: *mc,
						MemberCode:     m.MemberCode,
						MemberName:     m.FullName,
						MemberPhone:    m.Phone,
						TemplateCode:   tpl.Code,
						TemplateName:   tpl.Name,
					},
				}
			}
		}
	}
	return dto.Map(rows, toMemberContractItem), nil
}

func (s *contractPassService) SignMemberContract(ctx context.Context, req *v1.SignMemberContractReq) (*v1.MemberContractItem, error) {
	m, err := s.currentMember(ctx)
	if err != nil {
		return nil, err
	}
	sig := strings.TrimSpace(req.SignatureData)
	if sig == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng ký tên xác nhận")
	}
	row, err := s.repo.FindMemberContractByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy hợp đồng điện tử")
		}
		return nil, err
	}
	if row.MemberID != m.ID {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền ký hợp đồng này")
	}
	if row.Status == entity.ContractStatusSigned || row.SignedAt != nil {
		return nil, rescode.New(rescode.Conflict, "Hợp đồng đã được ký trước đó, không thể ký lại")
	}
	if err := validateSignature(sig); err != nil {
		return nil, err
	}
	now := s.now()
	mc := row.MemberContract
	mc.Status = entity.ContractStatusSigned
	mc.SignedAt = &now
	mc.SignatureURL = sig
	if r := g.RequestFromCtx(ctx); r != nil {
		mc.IP = clientip.Real(r)
		mc.UserAgent = r.Header.Get("User-Agent")
	}
	if err := s.repo.UpdateMemberContract(ctx, &mc); err != nil {
		return nil, err
	}
	row.MemberContract = mc
	return toMemberContractItem(*row), nil
}

// memberGuestQuota: hạn mức vé mời/tháng = guest_passes_per_month của gói tập đang hiệu lực
// (nhiều gói → lấy mức cao nhất); không có gói hiệu lực → 0.
func (s *contractPassService) memberGuestQuota(ctx context.Context, memberID uint) (int, []repository.ActiveSubPlan, error) {
	subs, err := s.repo.ListActiveSubPlans(ctx, memberID, s.passToday())
	if err != nil {
		return 0, nil, err
	}
	quota := 0
	for _, sp := range subs {
		quota = max(quota, sp.Plan.GuestPassesPerMonth)
	}
	return quota, subs, nil
}

// guestBranchAllowed: chi nhánh nằm trong phạm vi của một gói hiệu lực có vé mời
// (ALL_BRANCHES = mọi chi nhánh; SINGLE_BRANCH = chi nhánh của gói).
func guestBranchAllowed(subs []repository.ActiveSubPlan, branchID uint) bool {
	for _, sp := range subs {
		if sp.Plan.GuestPassesPerMonth <= 0 {
			continue
		}
		if sp.Plan.AccessScope == entity.AccessAllBranches || sp.Sub.BranchID == branchID {
			return true
		}
	}
	return false
}

func (s *contractPassService) MemberGuestPasses(ctx context.Context) (*v1.MemberGuestPassesRes, error) {
	m, err := s.currentMember(ctx)
	if err != nil {
		return nil, err
	}
	now := s.now().In(memberVNLoc)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, memberVNLoc)
	monthEnd := monthStart.AddDate(0, 1, 0)

	used, err := s.repo.CountSponsorGuestPassesInMonth(ctx, m.ID, monthStart, monthEnd)
	if err != nil {
		return nil, err
	}
	quota, _, err := s.memberGuestQuota(ctx, m.ID)
	if err != nil {
		return nil, err
	}
	remaining := quota - used
	if remaining < 0 {
		remaining = 0
	}
	rows, err := s.repo.ListSponsorGuestPasses(ctx, m.ID)
	if err != nil {
		return nil, err
	}
	items := dto.Map(rows, toDayPassItem)
	return &v1.MemberGuestPassesRes{
		Quota:         quota,
		UsedThisMonth: used,
		Remaining:     remaining,
		Passes:        items,
		Items:         items,
	}, nil
}

func (s *contractPassService) CreateGuestPass(ctx context.Context, req *v1.CreateGuestPassReq) (*v1.DayPassItem, error) {
	m, err := s.currentMember(ctx)
	if err != nil {
		return nil, err
	}
	buyer := strings.TrimSpace(req.BuyerName)
	if buyer == "" || strings.TrimSpace(req.Phone) == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập họ tên và số điện thoại khách mời")
	}
	phone, ok := NormalizeVNPhone(req.Phone)
	if !ok {
		return nil, rescode.New(rescode.BadRequest, "Số điện thoại khách mời không hợp lệ")
	}

	now := s.now().In(memberVNLoc)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, memberVNLoc)
	monthEnd := monthStart.AddDate(0, 1, 0)

	quota, subs, err := s.memberGuestQuota(ctx, m.ID)
	if err != nil {
		return nil, err
	}
	if quota <= 0 {
		return nil, rescode.New(rescode.BadRequest, "Gói tập hiện tại của bạn không bao gồm vé mời bạn bè")
	}

	validDate, err := s.passValidDate(req.ValidDate)
	if err != nil {
		return nil, err
	}
	branchID := req.BranchID
	if branchID == 0 {
		// Mặc định: chi nhánh của gói có vé mời (ưu tiên chi nhánh nhà nếu nằm trong phạm vi).
		if guestBranchAllowed(subs, m.HomeBranchID) {
			branchID = m.HomeBranchID
		} else {
			for _, sp := range subs {
				if sp.Plan.GuestPassesPerMonth > 0 {
					branchID = sp.Sub.BranchID
					break
				}
			}
		}
	}
	if _, err := s.activeBranch(ctx, branchID); err != nil {
		return nil, err
	}
	if !guestBranchAllowed(subs, branchID) {
		return nil, rescode.New(rescode.BadRequest, "Chi nhánh không thuộc phạm vi gói tập của bạn")
	}
	if err := checkPassDayOpen(ctx, branchID, validDate, "tạo vé mời"); err != nil {
		return nil, err
	}
	sponsorID := m.ID
	dp := &entity.DayPass{
		BranchID:        branchID,
		BuyerName:       buyer,
		Phone:           phone,
		SponsorMemberID: &sponsorID,
		Type:            entity.DayPassTypeGuest,
		ValidDate:       validDate,
		QRToken:         generateDayPassToken(),
		Price:           0,
		CreatedBy:       reqctx.UserID(ctx),
	}
	// Đếm + tạo trong một transaction có khóa dòng hội viên: hai yêu cầu đồng thời không thể cùng lọt qua
	// bước kiểm tra hạn mức (race tạo vượt số vé mời/tháng).
	err = s.withinTx(ctx, func(ctx context.Context) error {
		if err := s.repo.LockMember(ctx, m.ID); err != nil {
			return err
		}
		used, err := s.repo.CountSponsorGuestPassesInMonth(ctx, m.ID, monthStart, monthEnd)
		if err != nil {
			return err
		}
		if used >= quota {
			return rescode.New(rescode.Conflict, "Bạn đã sử dụng hết hạn mức vé mời bạn bè trong tháng này")
		}
		return s.repo.CreateDayPass(ctx, dp)
	})
	if err != nil {
		return nil, err
	}
	row, err := s.repo.FindDayPassByID(ctx, dp.ID)
	if err != nil {
		row = &repository.DayPassRow{DayPass: *dp, SponsorName: m.FullName}
	}
	return toDayPassItem(*row), nil
}

// ---- Mappers --------------------------------------------------------------------

func toContractTemplateItem(t entity.ContractTemplate) *v1.ContractTemplateItem {
	return &v1.ContractTemplateItem{
		ID:        t.ID,
		Code:      t.Code,
		Name:      t.Name,
		BodyHTML:  htmlsafe.Sanitize(t.BodyHTML), // lọc cả dữ liệu cũ đã lưu trước khi có bộ lọc
		Version:   t.Version,
		Status:    t.Status,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

func toMemberContractItem(r repository.MemberContractRow) *v1.MemberContractItem {
	return &v1.MemberContractItem{
		ID:              r.ID,
		MemberID:        r.MemberID,
		MemberCode:      r.MemberCode,
		MemberName:      r.MemberName,
		MemberPhone:     r.MemberPhone,
		TemplateID:      r.TemplateID,
		TemplateCode:    r.TemplateCode,
		TemplateName:    r.TemplateName,
		TemplateVersion: r.TemplateVersion,
		SubscriptionID:  r.SubscriptionID,
		RenderedHTML:    htmlsafe.Sanitize(r.RenderedHTML), // lọc cả bản đã kết xuất trước khi có escape
		Status:          r.Status,
		SignedAt:        r.SignedAt,
		SignatureURL:    safeSignatureURL(r.SignatureURL),
		PDFURL:          r.PDFURL,
		IP:              r.IP,
		UserAgent:       r.UserAgent,
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
	}
}

// safeSignatureURL: dữ liệu chữ ký cũ không hợp lệ (lưu trước khi có kiểm tra) không được trả về client.
func safeSignatureURL(v string) string {
	if v == "" || validateSignature(v) != nil {
		return ""
	}
	return v
}

func toDayPassItem(r repository.DayPassRow) *v1.DayPassItem {
	return &v1.DayPassItem{
		ID:              r.ID,
		BranchID:        r.BranchID,
		BranchName:      r.BranchName,
		BuyerName:       r.BuyerName,
		Phone:           r.Phone,
		MemberID:        r.MemberID,
		SponsorMemberID: r.SponsorMemberID,
		SponsorName:     r.SponsorName,
		Type:            r.Type,
		ValidDate:       r.ValidDate.Format(consts.DateLayout),
		QRToken:         r.QRToken,
		Price:           r.Price,
		InvoiceID:       r.InvoiceID,
		UsedAt:          r.UsedAt,
		CreatedBy:       r.CreatedBy,
		CreatedAt:       r.CreatedAt,
	}
}
