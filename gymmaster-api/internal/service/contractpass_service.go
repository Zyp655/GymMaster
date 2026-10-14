package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"gorm.io/gorm"

	v1 "gymmaster-api/api/contractpass/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
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
		repo  repository.ContractPassRepository
		tx    postgres.TxManager
		audit AuditService
		now   func() time.Time
	}
)

func NewContractPassService(repo repository.ContractPassRepository, tx postgres.TxManager, audit AuditService) ContractPassService {
	return &contractPassService{repo: repo, tx: tx, audit: audit, now: time.Now}
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

func renderContractHTML(bodyHTML string, m *entity.Member, planName string) string {
	if strings.TrimSpace(planName) == "" {
		planName = "Gói tập tiêu chuẩn GymMaster"
	}
	replacer := strings.NewReplacer(
		"{{member_name}}", m.FullName,
		"{{member_code}}", m.MemberCode,
		"{{member_phone}}", m.Phone,
		"{{plan_name}}", planName,
	)
	return replacer.Replace(bodyHTML)
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
	q := req.PageReq.ToQuery(dayPassOrderable, "id")
	q.OrderBy = "dp." + q.OrderBy
	rows, total, err := s.repo.ListDayPasses(ctx, q, req.BranchID, strings.TrimSpace(req.Type))
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(rows, toDayPassItem), total, q), nil
}

func (s *contractPassService) CreateDayPass(ctx context.Context, req *v1.CreateDayPassReq) (*v1.DayPassItem, error) {
	buyer := strings.TrimSpace(req.BuyerName)
	phone := strings.TrimSpace(req.Phone)
	if req.BranchID == 0 || buyer == "" || phone == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập đầy đủ chi nhánh, họ tên và số điện thoại")
	}
	validDate := s.now().In(memberVNLoc)
	if strings.TrimSpace(req.ValidDate) != "" {
		parsed, err := time.ParseInLocation(consts.DateLayout, strings.TrimSpace(req.ValidDate), memberVNLoc)
		if err != nil {
			return nil, rescode.New(rescode.BadRequest, "Ngày hiệu lực phải có định dạng YYYY-MM-DD")
		}
		validDate = parsed
	}
	dp := &entity.DayPass{
		BranchID:  req.BranchID,
		BuyerName: buyer,
		Phone:     phone,
		Type:      entity.DayPassTypeDay,
		ValidDate: validDate,
		QRToken:   generateDayPassToken(),
		Price:     req.Price,
		CreatedBy: reqctx.UserID(ctx),
	}
	if err := s.repo.CreateDayPass(ctx, dp); err != nil {
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
	now := s.now()
	mc := row.MemberContract
	mc.Status = entity.ContractStatusSigned
	mc.SignedAt = &now
	mc.SignatureURL = sig
	if r := g.RequestFromCtx(ctx); r != nil {
		mc.IP = r.GetClientIp()
		mc.UserAgent = r.Header.Get("User-Agent")
	}
	if err := s.repo.UpdateMemberContract(ctx, &mc); err != nil {
		return nil, err
	}
	row.MemberContract = mc
	return toMemberContractItem(*row), nil
}

func (s *contractPassService) memberGuestQuota(ctx context.Context, memberID uint) int {
	quota := 2
	if plan, _, err := s.repo.FindActiveMemberPlan(ctx, memberID); err == nil && plan != nil {
		if plan.GuestPassesPerMonth > quota {
			quota = plan.GuestPassesPerMonth
		}
	}
	return quota
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
	quota := s.memberGuestQuota(ctx, m.ID)
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
	phone := strings.TrimSpace(req.Phone)
	if buyer == "" || phone == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập họ tên và số điện thoại khách mời")
	}

	now := s.now().In(memberVNLoc)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, memberVNLoc)
	monthEnd := monthStart.AddDate(0, 1, 0)

	used, err := s.repo.CountSponsorGuestPassesInMonth(ctx, m.ID, monthStart, monthEnd)
	if err != nil {
		return nil, err
	}
	quota := s.memberGuestQuota(ctx, m.ID)
	if used >= quota {
		return nil, rescode.New(rescode.Conflict, "Bạn đã sử dụng hết hạn mức vé mời bạn bè trong tháng này")
	}

	validDate := now
	if strings.TrimSpace(req.ValidDate) != "" {
		parsed, err := time.ParseInLocation(consts.DateLayout, strings.TrimSpace(req.ValidDate), memberVNLoc)
		if err != nil {
			return nil, rescode.New(rescode.BadRequest, "Ngày hiệu lực phải có định dạng YYYY-MM-DD")
		}
		validDate = parsed
	}
	branchID := req.BranchID
	if branchID == 0 {
		branchID = m.HomeBranchID
	}
	if branchID == 0 {
		branchID = 1
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
	if err := s.repo.CreateDayPass(ctx, dp); err != nil {
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
		BodyHTML:  t.BodyHTML,
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
		RenderedHTML:    r.RenderedHTML,
		Status:          r.Status,
		SignedAt:        r.SignedAt,
		SignatureURL:    r.SignatureURL,
		PDFURL:          r.PDFURL,
		IP:              r.IP,
		UserAgent:       r.UserAgent,
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
	}
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
